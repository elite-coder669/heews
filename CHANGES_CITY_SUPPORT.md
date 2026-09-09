# Multi-city ward support — what changed

## The problem
The app only ever loaded one file: `data/fixtures/wards/hyderabad_wards.geojson`,
which held 10 hand-drawn square polygons, not real ward boundaries.

## What this repo (Municipal_Spatial_Data) actually contains
It does **not** have state-level GeoJSON. It has real ward-boundary polygons
for **25 individual Indian cities/municipalities**, each scraped from a
different source with a different property schema. There is no public
dataset that gives ward-level boundaries for every state in India — this is
the closest thing that exists, and it's city-level.

## What was done
1. `scripts/convert_wards.py` — normalizes all 25 cities' raw GeoJSON
   (different field names, one NDJSON-formatted file, one file in a
   projected coordinate system) into the schema the app expects:
   `{ward_number, ward_name, zone}` + Polygon/MultiPolygon geometry.
   Re-run it if you add a new city folder from the Municipal_Spatial_Data
   repo — see the `CITIES` dict at the top of the script.
2. `data/fixtures/wards/<city>_wards.geojson` — real ward polygons for all
   25 cities (Hyderabad's old 10 squares are replaced with the real 145
   GHMC wards).
3. `data/fixtures/demographics/<city>_demo.json` — synthetic per-ward
   demographics (elderly ratio, outdoor worker share, etc.) for every city.
   **These numbers are made up**, same as the original Hyderabad fixture —
   there's no real demographic dataset wired in. Swap in real census data
   here if you get access to it.
4. `data/fixtures/wards/cities_manifest.json` — city → state, ward count,
   source file, for reference.
5. Backend (Go): added a `CITY` env var (`backend/internal/config/cities.go`
   has the lookup table of all 25 cities' label/state/lat/lon). Every
   hardcoded "Hyderabad" string in the orchestration/agent code now reads
   from config instead. New `/api/config` endpoint exposes the active city.
6. Frontend (React): asks the backend which city it's running (`/api/config`)
   and fetches the matching `/data/<city>_wards.geojson` instead of a
   hardcoded path.

## Known limitation
Switching cities means restarting the backend with a different `CITY=`
value — there's no in-app dropdown to switch cities live without a restart.
Wiring that up would mean either running one backend process per city, or a
larger change to make the orchestrator's in-memory state keyed by city
instead of a single global city. Flagging this as a good next step, not
something silently missing.
