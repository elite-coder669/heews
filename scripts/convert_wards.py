#!/usr/bin/env python3
"""
Normalizes raw ward GeoJSON files (from the DataMeet Municipal_Spatial_Data
repo, which has a different property schema per city) into the schema HEEWS
expects:

  { "type": "Feature",
    "properties": {"ward_number": <int>, "ward_name": <str>, "zone": <str>},
    "geometry": {...Polygon or MultiPolygon...} }

Usage: python3 convert_wards.py
Reads from ./municipal_spatial_data/Municipal_Spatial_Data-master/<City>/...
Writes to ./converted_wards/<city_slug>_wards.geojson
Also writes ./converted_wards/_manifest.json describing city -> {label, state, ward_count, source}
"""
import json
import math
import os
import re
import unicodedata

SRC_ROOT = "/home/claude/work/municipal_spatial_data/Municipal_Spatial_Data-master"
OUT_DIR = "/home/claude/work/converted_wards"
os.makedirs(OUT_DIR, exist_ok=True)

# city folder -> (source file, approximate state, display label)
# Only cities with a genuine ward-level polygon layer are included.
# (Mira-Bhayandar's file is a single city-boundary polygon, not wards, so it's excluded.)
CITIES = {
    "Ahmedabad":      ("Wards.geojson",              "Gujarat"),
    "Bangalore":      ("BBMP.geojson",                "Karnataka"),
    "Bhopal":         ("Bhopal_wards.geojson",        "Madhya Pradesh"),
    "Bhubaneswar":    ("Wards.GeoJSON",                "Odisha"),
    "Bodh_Gaya":      ("Bodh_Gaya_Wards.geojson",     "Bihar"),
    "Chandigarh":     ("Chandigarh_Wards.geojson",    "Chandigarh (UT)"),
    "Chennai":        ("Wards.geojson",                "Tamil Nadu"),
    "Coimbatore":     ("Cbe2011Wards.geojson",        "Tamil Nadu"),
    "Delhi":          ("Delhi_Wards.geojson",         "Delhi (NCT)"),
    "Faridabad":      ("Faridabad_Wards.geojson",     "Haryana"),
    "Hyderabad":      ("ghmc-wards.geojson",          "Telangana"),
    "Jaipur":         ("Jaipur_Wards.geojson",        "Rajasthan"),
    "Kanpur":         ("Kanpur_wards.geojson",        "Uttar Pradesh"),
    "Katihar":        ("Katihar_Wards.geojson",       "Bihar"),
    "Kishangarh":     ("Kishangarh_Wards.geojson",    "Rajasthan"),
    "Kochi":          ("KCH_wards.geojson",           "Kerala"),
    "Kolkata":        ("kolkata.geojson",              "West Bengal"),
    "Lucknow":        ("Lucknow_ward_boundary.geojson","Uttar Pradesh"),
    "Mumbai":         ("BMC_Wards.geojson",           "Maharashtra"),
    "NMMC":           ("NMC_ElectoralWards.geojson",  "Maharashtra"),
    "PCMC":           ("pcmc-electoral-wards.geojson","Maharashtra"),
    "Pune":           ("pune-electoral-wards_2022.geojson", "Maharashtra"),
    "Purnia":         ("Purnia_Wards.geojson",        "Bihar"),
    "Vadodara":       ("vardodara_wards.geojson",     "Gujarat"),
    "Vijayawada":     ("Vijayawada_Wards.geojson",    "Andhra Pradesh"),
}

NUM_RE = re.compile(r"(\d+)")

# Cities whose source file uses projected (non-lat/lon) coordinates, e.g.
# Web Mercator (EPSG:3857) meters instead of WGS84 degrees. Detected by
# coordinate magnitude (>180/>90) and reprojected before use, otherwise
# the wards would render in the wrong place (or off the map entirely).
def webmercator_to_lonlat(x, y):
    lon = x / 20037508.34 * 180.0
    lat_rad = y / 20037508.34 * 180.0
    lat = 180.0 / math.pi * (2 * math.atan(math.exp(lat_rad * math.pi / 180.0)) - math.pi / 2)
    return [lon, lat]


def reproject_if_needed(geom):
    def flat_sample(g):
        if g["type"] == "Polygon":
            return g["coordinates"][0][0]
        if g["type"] == "MultiPolygon":
            return g["coordinates"][0][0][0]
        return [0, 0]

    pt = flat_sample(geom)
    x, y = pt[0], pt[1]
    if abs(x) <= 180 and abs(y) <= 90:
        return geom  # already plain lat/lon

    def conv_ring(ring):
        return [webmercator_to_lonlat(c[0], c[1]) for c in ring]

    if geom["type"] == "Polygon":
        geom["coordinates"] = [conv_ring(r) for r in geom["coordinates"]]
    elif geom["type"] == "MultiPolygon":
        geom["coordinates"] = [[conv_ring(r) for r in poly] for poly in geom["coordinates"]]
    return geom


def slugify(name):
    n = unicodedata.normalize("NFKD", name).encode("ascii", "ignore").decode()
    n = re.sub(r"[^a-zA-Z0-9]+", "_", n).strip("_").lower()
    return n


def load_features(path):
    """Handles both standard FeatureCollection files and NDJSON-of-Features files (Kochi)."""
    with open(path, "r", encoding="utf-8-sig") as f:
        content = f.read()
    try:
        d = json.loads(content)
        if isinstance(d, dict) and d.get("type") == "FeatureCollection":
            return d["features"]
        if isinstance(d, dict) and d.get("type") == "Feature":
            return [d]
    except json.JSONDecodeError:
        pass
    # NDJSON fallback: concatenated JSON objects
    feats = []
    dec = json.JSONDecoder()
    idx, n = 0, len(content)
    while idx < n:
        while idx < n and content[idx] in " \n\t\r":
            idx += 1
        if idx >= n:
            break
        obj, end = dec.raw_decode(content, idx)
        if obj.get("type") == "Feature":
            feats.append(obj)
        idx = end
    return feats


def first_present(props, keys):
    for k in keys:
        if k in props and props[k] not in (None, ""):
            return props[k]
    return None


NAME_KEYS = ["Ward_Name", "ward_name", "WARD_NAME", "Ward Name", "Name", "name",
             "KGISWardName", "Name1", "Name2"]
NUM_KEYS = ["Ward_No", "ward_no", "WARD_NO", "Ward No", "Ward_Number", "wardno",
            "wardnum", "Ward Num", "KGISWardNo", "2011WardNumbers", "WARD"]
ZONE_KEYS = ["Zone_Name", "zone", "Zone", "ZONE_NAME", "Zone No", "Zone_No",
             "municipalzone", "Zone_No "]


def extract(props, idx_fallback, city_label):
    name = first_present(props, NAME_KEYS)
    num = first_present(props, NUM_KEYS)
    zone = first_present(props, ZONE_KEYS)

    # Hyderabad-style: "Ward 91 Khairatabad" all packed into `name`
    if name and num is None:
        m = re.match(r"\s*Ward\s+(\d+)\s+(.*)", str(name), re.IGNORECASE)
        if m:
            num = m.group(1)
            name = m.group(2)

    raw_num = num
    if num is not None and re.fullmatch(r"\s*\d+\s*", str(num)):
        # purely numeric field (e.g. "42") -> trust it directly
        num = int(str(num).strip())
    else:
        # mixed/prefixed codes (e.g. "CANT_1", "NDMC_3" in Delhi) aren't a
        # reliable unique ward number on their own -> use sequential index
        # instead, and keep the original code visible in the name.
        num = idx_fallback

    if not name:
        name = f"{city_label} Ward {num}"
    if raw_num is not None and not re.fullmatch(r"\s*\d+\s*", str(raw_num)):
        name = f"{name} ({raw_num})" if str(raw_num) not in str(name) else name
    name = str(name).strip()

    zone = str(zone).strip() if zone else ""

    return num, name, zone


def convert_city(city, fname, state):
    path = os.path.join(SRC_ROOT, city, fname)
    if not os.path.exists(path):
        print(f"  SKIP {city}: source file not found at {path}")
        return None
    feats = load_features(path)
    if len(feats) <= 1:
        print(f"  SKIP {city}: only {len(feats)} feature(s), not ward-level data")
        return None

    out_feats = []
    seen_numbers = set()
    for i, f in enumerate(feats, start=1):
        geom = f.get("geometry")
        if not geom or geom.get("type") not in ("Polygon", "MultiPolygon"):
            continue
        geom = reproject_if_needed(geom)
        props = f.get("properties", {}) or {}
        num, name, zone = extract(props, i, city)
        # de-duplicate ward numbers (some datasets repeat/parse ambiguously).
        # Probe upward by 1 so IDs stay small (backend zero-pads to 3 digits,
        # i.e. must stay under 1000) instead of jumping by large offsets.
        while num in seen_numbers:
            num += 1
        seen_numbers.add(num)
        out_feats.append({
            "type": "Feature",
            "properties": {"ward_number": num, "ward_name": name, "zone": zone},
            "geometry": geom,
        })

    if not out_feats:
        print(f"  SKIP {city}: no usable polygon features")
        return None

    fc = {"type": "FeatureCollection", "features": out_feats}
    slug = slugify(city)
    out_path = os.path.join(OUT_DIR, f"{slug}_wards.geojson")
    with open(out_path, "w") as f:
        json.dump(fc, f)
    print(f"  OK {city:15s} -> {slug}_wards.geojson  ({len(out_feats)} wards, state={state})")
    return {
        "slug": slug,
        "label": city.replace("_", " "),
        "state": state,
        "ward_count": len(out_feats),
        "source_file": fname,
    }


def main():
    manifest = {}
    for city, (fname, state) in CITIES.items():
        info = convert_city(city, fname, state)
        if info:
            manifest[info["slug"]] = info
    with open(os.path.join(OUT_DIR, "_manifest.json"), "w") as f:
        json.dump(manifest, f, indent=2)
    print(f"\nDone. {len(manifest)} cities converted. Manifest at {OUT_DIR}/_manifest.json")


if __name__ == "__main__":
    main()
