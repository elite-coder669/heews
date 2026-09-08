"""Solar geometry helpers.

Computes solar zenith and azimuth from (lat, lon, datetime, timezone).
Used by WBGT (direct/diffuse separation) and UTCI (mean radiant temp).
"""
from __future__ import annotations

import math
from datetime import datetime, timezone


def solar_declination(d: datetime) -> float:
    N = d.timetuple().tm_yday
    return 23.45 * math.sin(math.radians(360.0 * (284 + N) / 365.0))


def equation_of_time_minutes(d: datetime) -> float:
    N = d.timetuple().tm_yday
    B = math.radians(360.0 * (N - 81) / 365.0)
    return 9.87 * math.sin(2 * B) - 7.53 * math.cos(B) - 1.5 * math.sin(B)


def solar_zenith(lat: float, lon: float, ts: datetime) -> float:
    if ts.tzinfo is None:
        ts = ts.replace(tzinfo=timezone.utc)
    delta = math.radians(solar_declination(ts))
    phi = math.radians(lat)
    # Local solar time approximation
    eot = equation_of_time_minutes(ts)
    lstm = 15.0 * lon / 15.0
    tc = 4.0 * (lon - lstm) + eot
    local_hours = ts.hour + ts.minute / 60.0 + ts.second / 3600.0 + tc / 60.0
    H = math.radians((local_hours - 12.0) * 15.0)
    cos_z = math.sin(phi) * math.sin(delta) + math.cos(phi) * math.cos(delta) * math.cos(H)
    cos_z = max(-1.0, min(1.0, cos_z))
    return math.degrees(math.acos(cos_z))


def mean_radiant_temp(shortwave_rad_wm2: float) -> float:
    """Tmrt approximation from shortwave radiation only."""
    sigma = 5.670374419e-8
    rad = max(shortwave_rad_wm2, 0.0)
    return ((rad / (5.4 * sigma)) ** 0.25) - 273.15