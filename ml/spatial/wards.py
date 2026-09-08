"""Spatial downscaling: ward centroids + IDW interpolation.

MVP uses Open-Meteo's gridded forecast sampled at ward centroid.
IDW is the documented fallback when raw grid data are available.
"""
from __future__ import annotations

import hashlib
import json
import logging
import math
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

WARD_ID_NAMESPACE = "heatwave.wards"


def stable_ward_id(ward_name: str, ward_number: int) -> str:
    raw = f"{ward_name.strip().lower()}|{ward_number}".encode()
    h = hashlib.sha1(raw).hexdigest()[:12]  # noqa: S324
    return f"ward_{h}"


def centroid_of_geometry(geom: dict[str, Any]) -> tuple[float, float]:
    """Return (lat, lon) for a Polygon GeoJSON geometry.

    MVP uses bounding-box center as a stand-in for representative_point.
    """
    coords = geom.get("coordinates", [])
    if geom.get("type") == "Polygon":
        ring = coords[0]
    elif geom.get("type") == "MultiPolygon":
        ring = coords[0][0]
    else:
        raise ValueError(f"Unsupported geometry type: {geom.get('type')}")
    lats = [c[1] for c in ring]
    lons = [c[0] for c in ring]
    return (sum(lats) / len(lats), sum(lons) / len(lons))


def load_wards(path: Path) -> list[dict[str, Any]]:
    fc = json.loads(path.read_text())
    wards: list[dict[str, Any]] = []
    for f in fc.get("features", []):
        props = f.get("properties", {})
        geom = f.get("geometry", {})
        lat, lon = centroid_of_geometry(geom)
        wards.append({
            "ward_id": stable_ward_id(props.get("ward_name", ""), int(props.get("ward_number", 0))),
            "ward_number": int(props.get("ward_number", 0)),
            "ward_name": props.get("ward_name", ""),
            "zone": props.get("zone", ""),
            "centroid": {"lat": lat, "lon": lon},
        })
    return wards


def idw(
    centroid_lat: float,
    centroid_lon: float,
    grid_points: list[tuple[float, float, float]],
    power: int = 2,
) -> float:
    """Inverse-distance-weighted interpolation.

    grid_points: [(lat, lon, value), ...]
    Returns interpolated value at centroid.
    """
    if not grid_points:
        raise ValueError("grid_points is empty")

    def hav(a_lat: float, a_lon: float, b_lat: float, b_lon: float) -> float:
        R = 6371.0
        dlat = math.radians(b_lat - a_lat)
        dlon = math.radians(b_lon - a_lon)
        a = math.sin(dlat / 2) ** 2 + math.cos(math.radians(a_lat)) * math.cos(math.radians(b_lat)) * math.sin(dlon / 2) ** 2
        return 2 * R * math.asin(math.sqrt(a))

    num = 0.0
    den = 0.0
    for lat, lon, v in grid_points:
        d = hav(centroid_lat, centroid_lon, lat, lon)
        if d < 1e-6:
            return v
        w = 1.0 / (d ** power)
        num += w * v
        den += w
    return num / den