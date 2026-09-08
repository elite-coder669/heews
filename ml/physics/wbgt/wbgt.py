"""WBGT (Wet Bulb Globe Temperature).

Liljegren-style approximation with simplified fallback.
This is an implementation assumption; reference Liljegren 2008.
"""
from __future__ import annotations

from dataclasses import dataclass


@dataclass
class WBGTResult:
    wbgt_c: float
    quality: str  # "ok" | "degraded"


def simplified_wbgt(t_db_c: float, rh_pct: float, shortwave_wm2: float) -> WBGTResult:
    """Approximation when direct/diffuse radiation not available."""
    # Natural wet-bulb (Stull approximation, valid 5..99% RH, -20..50°C)
    Tw = (
        t_db_c * math_atan(0.151977 * (rh_pct + 8.313659) ** 0.5)
        + math_atan(t_db_c + rh_pct)
        - math_atan(rh_pct - 1.6763)
        + 0.00391838 * rh_pct ** 1.5 * math_atan(0.023101 * rh_pct)
        - 4.686035
    )
    Tg_simple = t_db_c + (shortwave_wm2 / 200.0)
    wbgt = 0.567 * Tw + 0.393 * Tg_simple + 0.1 * t_db_c
    return WBGTResult(wbgt_c=round(wbgt, 2), quality="degraded")


def liljegren_wbgt(
    t_db_c: float,
    rh_pct: float,
    wind_ms: float,
    pressure_pa: float,
    direct_wm2: float,
    diffuse_wm2: float,
    zenith_deg: float,
) -> WBGTResult:
    """Liljegren 2008 — operational approximation.

    For MVP we use the simplified approximation + explicit quality flag.
    """
    shortwave = direct_wm2 + diffuse_wm2
    if shortwave <= 0 or wind_ms is None or pressure_pa is None:
        return simplified_wbgt(t_db_c, rh_pct, max(shortwave, 0.0))

    Tw = (
        t_db_c * math_atan(0.151977 * (rh_pct + 8.313659) ** 0.5)
        + math_atan(t_db_c + rh_pct)
        - math_atan(rh_pct - 1.6763)
        + 0.00391838 * rh_pct ** 1.5 * math_atan(0.023101 * rh_pct)
        - 4.686035
    )
    # Globe temperature approximation
    Tg = t_db_c + (direct_wm2 / max(wind_ms, 0.5) / 60.0) + (diffuse_wm2 / 250.0)
    wbgt = 0.567 * Tw + 0.393 * Tg + 0.1 * t_db_c
    return WBGTResult(wbgt_c=round(wbgt, 2), quality="ok")


def math_atan(x: float) -> float:
    import math
    return math.atan(x)