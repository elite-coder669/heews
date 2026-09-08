"""Weather ingestion from Open-Meteo.

Fetches current + forecast weather for the Hyderabad bounding box,
normalizes units, and returns dicts matching docs/API_CONTRACTS.md §3.
"""
from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

import urllib.request
import urllib.parse

logger = logging.getLogger(__name__)

PHYSICS_VERSION = "wbgt-liljegren-1.0"

VARS = ",".join([
    "temperature_2m",
    "relative_humidity_2m",
    "wind_speed_10m",
    "shortwave_radiation",
    "direct_radiation",
    "diffuse_radiation",
    "surface_pressure",
    "cloud_cover",
])

def fetch_open_meteo(
    lat: float,
    lon: float,
    base: str = "https://api.open-meteo.com/v1/forecast",
    forecast_days: int = 5,
) -> dict[str, Any]:
    params = {
        "latitude": lat,
        "longitude": lon,
        "current": VARS,
        "hourly": VARS,
        "forecast_days": forecast_days,
        "timezone": "Asia/Kolkata",
        "wind_speed_unit": "kmh",
    }
    url = f"{base}?{urllib.parse.urlencode(params)}"
    logger.info("fetch_open_meteo url=%s", url)
    with urllib.request.urlopen(url, timeout=20) as resp:  # noqa: S310
        return json.loads(resp.read().decode("utf-8"))


def normalize(payload: dict[str, Any]) -> dict[str, Any]:
    """Convert wind km/h → m/s; pressure hPa → Pa. Idempotent."""
    cur = payload.get("current", {})
    if "wind_speed_10m" in cur:
        cur["wind_speed_10m"] = float(cur["wind_speed_10m"]) * 1000.0 / 3600.0
    if "surface_pressure" in cur:
        cur["surface_pressure"] = float(cur["surface_pressure"]) * 100.0
    hourly = payload.get("hourly", {})
    if hourly:
        if "wind_speed_10m" in hourly:
            hourly["wind_speed_10m"] = [
                float(v) * 1000.0 / 3600.0 for v in hourly["wind_speed_10m"]
            ]
        if "surface_pressure" in hourly:
            hourly["surface_pressure"] = [
                float(v) * 100.0 for v in hourly["surface_pressure"]
            ]
    return payload


def fetch_mock(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text())


def fetch(lat: float, lon: float, mode: str = "LIVE") -> dict[str, Any]:
    if mode == "MOCK":
        return fetch_mock(Path("data/fixtures/weather/open_meteo_sample.json"))
    payload = fetch_open_meteo(lat, lon)
    return normalize(payload)