#!/usr/bin/env python3
"""HEEWS live vertical-slice pipeline.

Open-Meteo weather -> ward spatialization -> solar geometry -> WBGT/UTCI ->
vulnerability-aware MRI -> 5-day forecast -> threshold alerts.

Emits ONE contract-compliant JSON document to stdout. Pure stdlib.

Run from the repository root:
    python3 ml/run_pipeline.py --mode LIVE
    python3 ml/run_pipeline.py --mode MOCK
"""

import argparse
import json
import math
import sys
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "ml"))

from ingestion.open_meteo import fetch, normalize, PHYSICS_VERSION                      # noqa: E402
from physics.solar.solar_geometry import solar_zenith, mean_radiant_temp               # noqa: E402
from physics.wbgt.wbgt import liljegren_wbgt                                            # noqa: E402
from physics.utci.utci import utci_polynomial                                          # noqa: E402
from risk.risk_model import compute_mri, vulnerability_index                            # noqa: E402
from spatial.wards import load_wards                                                   # noqa: E402

IST = timezone(timedelta(hours=5, minutes=30))
UTC = timezone.utc
MRIT_BAND = "HIGH"
WARN_WBGT = 28.0
GEOJSON = ROOT / "data/fixtures/wards/hyderabad_wards.geojson"
DEMO = ROOT / "data/fixtures/demographics/hyderabad_demo.json"


def ward_contract_id(num: int) -> str:
    return f"ward_{num:03d}"


def load_demographics() -> dict:
    if not DEMO.exists():
        return {}
    return {d["ward_id"]: d for d in json.loads(DEMO.read_text())}


def default_demographics() -> dict:
    return {
        "elderly_ratio": 0.10,
        "outdoor_worker_share": 0.15,
        "informal_housing_share": 0.20,
        "population": 30000,
        "source_year": 2021,
    }


def fnum(v):
    try:
        return float(v)
    except (TypeError, ValueError):
        return 0.0


def parse_ist(s: str) -> datetime:
    dt = datetime.fromisoformat(s)
    return dt.replace(tzinfo=IST)


def hour_arrays(payload: dict) -> dict:
    """Return per-hour dicts of lists from a (normalized or raw) payload."""
    hourly = payload.get("hourly", {})
    keys = [
        "time", "temperature_2m", "relative_humidity_2m", "wind_speed_10m",
        "shortwave_radiation", "direct_radiation", "diffuse_radiation",
        "surface_pressure", "cloud_cover",
    ]
    out = {}
    for k in keys:
        raw = hourly.get(k, [])
        if k == "time":
            out[k] = [parse_ist(x) for x in raw]
        else:
            out[k] = [fnum(x) for x in raw]
    n = min(len(out["time"]) for out_k in out.values() if isinstance(out_k, list))
    for k in keys:
        out[k] = out[k][:n]
    return out


def compute_hourly_thermal(hours: dict) -> tuple:
    """Return (wbgt[], utci[], quality) for the whole hourly horizon."""
    n = len(hours["time"])
    wbgts, utcis = [], []
    quality = "ok"
    for i in range(n):
        t = hours["temperature_2m"][i]
        rh = hours["relative_humidity_2m"][i]
        wind = hours["wind_speed_10m"][i]
        press = hours["surface_pressure"][i]
        direct = hours["direct_radiation"][i]
        diffuse = hours["diffuse_radiation"][i]
        sw = direct + diffuse
        dt_utc = hours["time"][i].astimezone(UTC)
        zen = solar_zenith(hours["lat"], hours["lon"], dt_utc)
        res = liljegren_wbgt(t, rh, wind, press, direct, diffuse, zen)
        if res.quality != "ok":
            quality = "degraded"
        wbgts.append(res.wbgt_c)
        tmrt = mean_radiant_temp(sw, t)
        utcis.append(utci_polynomial(t, tmrt, wind, rh))
    return wbgts, utcis, quality


def overnight_gap(wbgt_day: list) -> float:
    """Diurnal recovery gap: daily peak WBGT minus overnight-min WBGT."""
    if not wbgt_day:
        return 0.0
    day_idx = 0
    peak = max(wbgt_day)
    night_idx = [i for i in range(len(wbgt_day)) if 22 <= i % 24 or i % 24 <= 6]
    if night_idx:
        return peak - min(wbgt_day[i] for i in night_idx)
    return peak - min(wbgt_day)


def forecast_cells(wbgt_day_by_idx):
    """Per-day windows of wbgt with UTCI at the hour of peak WBGT."""
    pass


def build_forecast_series(days_windows: list, utci_by_hour: list,
                          wbgt_by_hour: list, vuln: float) -> list:
    series = []
    now = datetime.now(IST)
    for d, window in enumerate(days_windows):
        if not window:
            continue
        peak_wbgt = max(window)
        peak_i = window.index(peak_wbgt) + d * 24
        utci_peak = utci_by_hour[peak_i] if peak_i < len(utci_by_hour) else 0.0
        persistence = sum(
            1 for w in days_windows[d:] if w and max(w) >= WARN_WBGT
        )
        gap = overnight_gap(window)
        mri = compute_mri({
            "peak_wbgt_next_72h": peak_wbgt,
            "utci_now": utci_peak,
            "persistence_days_high": persistence,
            "vulnerability_index": vuln,
            "overnight_recovery_gap_c": gap,
        })
        if d == 0:
            label = "Today"
        elif d == 1:
            label = "Tomorrow"
        else:
            day_dt = now + timedelta(days=d)
            label = day_dt.strftime("%a %d")
        series.append({
            "day": label,
            "wbgt_c": round(peak_wbgt, 1),
            "utci_c": round(utci_peak, 1),
            "mri": mri["mri"],
            "band": mri["band"],
        })
    return series


ACTIONS_BY_BAND = {
    "EXTREME": [
        ("Activate cooling centre", "HIGH", "ward cooling centre / GHMC"),
        ("Issue outdoor-work advisory", "HIGH", "construction & delivery workers"),
        ("Pre-position ORS and water", "MEDIUM", "public distribution points"),
    ],
    "VERY_HIGH": [
        ("Pre-position cooling materials", "MEDIUM", "ward resource depot"),
    ],
    "HIGH": [
        ("Issue public heat advisory", "LOW", "residents, schools, clinics"),
    ],
}


def alert_for(risk: dict, ward: dict) -> dict | None:
    band = risk["risk"]["band"]
    if band not in ACTIONS_BY_BAND:
        return None
    now = datetime.now(IST).isoformat()
    wbgt = risk.get("wbgt_c", 0.0)
    utci = risk.get("utci_c", 0.0)
    actions = [
        {"action": a, "priority": p, "target": tg}
        for a, p, tg in ACTIONS_BY_BAND[band]
    ]
    return {
        "id": uuid.uuid4().hex[:12],
        "ward_id": ward["ward_id"],
        "created_at": now,
        "severity": band,
        "headline": f"{band} heat risk in {ward['ward_name']}, {ward['zone']} zone",
        "body": (
            f"MRI {risk['risk']['score']}/100 · WBGT {wbgt:.1f} °C · "
            f"UTCI {utci:.1f} °C. Above danger thresholds for vulnerable groups."
        ),
        "recommended_actions": actions,
        "status": "ACTIVE",
    }


def process_ward(ward: dict, demo: dict, mode: str, base: str,
                 fallback_lat: float, fallback_lon: float) -> dict:
    wid = ward_contract_id(ward["ward_number"])
    lat = ward["centroid"]["lat"] or fallback_lat
    lon = ward["centroid"]["lon"] or fallback_lon
    demo_row = demo.get(wid, default_demographics())
    vuln = vulnerability_index(demo_row)

    payload = fetch(lat, lon, mode=mode)
    if mode == "MOCK":
        payload = normalize(payload)
    hours = hour_arrays(payload)
    hours["lat"], hours["lon"] = lat, lon

    wbgts, utcis, quality = compute_hourly_thermal(hours)
    n = min(len(wbgts), 72)
    peak72 = max(wbgts[:n], default=0.0)

    days_windows = []
    for d in range(5):
        i0 = d * 24
        i1 = min((d + 1) * 24, len(wbgts))
        if i0 >= len(wbgts):
            days_windows.append([])
        else:
            days_windows.append(wbgts[i0:i1])

    persistence = sum(1 for w in days_windows if w and max(w) >= WARN_WBGT)
    gap = overnight_gap(days_windows[0]) if days_windows and days_windows[0] else 0.0

    now_utci = utcis[0] if utcis else 0.0
    mri = compute_mri({
        "peak_wbgt_next_72h": peak72,
        "utci_now": now_utci,
        "persistence_days_high": persistence,
        "vulnerability_index": vuln,
        "overnight_recovery_gap_c": gap,
    })

    now = datetime.now(IST).isoformat()
    cur = payload.get("current", {})
    wbgt_now = wbgts[0] if wbgts else 0.0
    utci_now = now_utci

    risk = {
        "ward_id": wid,
        "computed_at": now,
        "risk": {
            "score": mri["mri"],
            "band": mri["band"],
            "top_factors": mri["top_factors"],
        },
        "wbgt_c": round(wbgt_now, 2),
        "utci_c": round(utci_now, 2),
        "wbgt_quality": quality,
        "risk_model_version": mri["risk_model_version"],
    }

    thermal = {
        "ward_id": wid,
        "physics_version": PHYSICS_VERSION,
        "current": {"wbgt_c": round(wbgt_now, 2), "utci_c": round(utci_now, 2),
                    "wbgt_quality": quality},
    }

    weather = {
        "ward_id": wid,
        "source": "open-meteo" if mode == "LIVE" else "open-meteo-sample",
        "fetched_at": now,
        "current": {
            "temperature_c": round(fnum(cur.get("temperature_2m")), 2),
            "humidity_pct": round(fnum(cur.get("relative_humidity_2m")), 1),
            "wind_ms": round(fnum(cur.get("wind_speed_10m")), 2),
            "shortwave_rad_wm2": round(fnum(cur.get("shortwave_radiation")), 1),
            "pressure_pa": round(fnum(cur.get("surface_pressure")), 1),
            "cloud_cover_pct": round(fnum(cur.get("cloud_cover")), 1),
        },
    }

    forecast_series = build_forecast_series(days_windows, utcis, wbgts, vuln)

    return {
        "wards": {
            "ward_id": wid,
            "ward_number": ward["ward_number"],
            "ward_name": ward["ward_name"],
            "zone": ward["zone"],
            "centroid": {"lat": lat, "lon": lon},
            "demographics": demo_row,
        },
        "risk": risk,
        "thermal": thermal,
        "weather": weather,
        "forecast_series": forecast_series,
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="HEEWS live pipeline")
    ap.add_argument("--mode", choices=["LIVE", "MOCK"], default="LIVE")
    ap.add_argument("--lat", type=float, default=17.3850)
    ap.add_argument("--lon", type=float, default=78.4867)
    ap.add_argument("--base", default="https://api.open-meteo.com/v1/forecast")
    args = ap.parse_args()

    demo = load_demographics()
    geo_wards = load_wards(GEOJSON) if GEOJSON.exists() else []

    wards_out, risks, thermal, weather = {}, {}, {}, {}
    by_ward, alerts = [], []

    for w in geo_wards:
        try:
            res = process_ward(w, demo, args.mode, args.base, args.lat, args.lon)
        except Exception as exc:  # keep the slice alive on a bad ward
            sys.stderr.write(f"warning: ward {w.get('ward_number')} failed: {exc}\n")
            continue
        wid = res["wards"]["ward_id"]
        wards_out[wid] = res["wards"]
        risks[wid] = res["risk"]
        thermal[wid] = res["thermal"]
        weather[wid] = res["weather"]
        by_ward.append({"ward_id": wid, "series": res["forecast_series"]})
        alert = alert_for(res["risk"], res["wards"])
        if alert:
            alerts.append(alert)

    doc = {
        "generated_at": datetime.now(IST).isoformat(),
        "mode": args.mode,
        "wards": list(wards_out.values()),
        "risks": risks,
        "thermal": thermal,
        "weather": weather,
        "forecast": {
            "city": "Hyderabad",
            "horizon_hours": 120,
            "by_ward": by_ward,
        },
        "alerts": alerts,
    }
    json.dump(doc, sys.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())