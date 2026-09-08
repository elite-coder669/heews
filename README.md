# HEEWS — Extreme Heatwave Early Warning & Human Thermal Stress Index

Decision-support prototype that turns weather into municipal heat-health action for Hyderabad.

```
WEATHER  →  PHYSIOLOGICAL HEAT STRESS  →  POPULATION RISK  →  DECISION  →  ACTION
```

## Quickstart (MOCK mode, no internet)

```bash
cp .env.example .env
docker compose up --build
# Frontend: http://localhost:5173
# Backend:  http://localhost:8080/api/health
```

## Quickstart (dev mode)

Backend:
```bash
cd backend
go mod tidy
go run ./cmd/server
```

Frontend:
```bash
cd frontend
npm install
npm run dev
```

ML / agent (Python):
```bash
cd ml && pip install -r ../requirements.txt
python -m pytest ml/tests
```

## Switching LIVE / MOCK

Edit `.env`:

```
APP_MODE=LIVE   # uses Open-Meteo + OpenStreetMap fixtures
APP_MODE=MOCK   # deterministic offline fixtures
```

## Layout

```
docs/        SRS, architecture, contracts, agent spec, physics spec, ML spec, demo
backend/     Go service: API, orchestration, alerts
ml/          Python: ingestion, spatial, physics (WBGT, UTCI), risk model
agent/       Python: tools, prompts, decision agent
frontend/    React + Leaflet + Recharts dashboard
data/        Ward GeoJSON, demographics, fixtures
scripts/     Dev and demo scripts
tools/       PDF generator
```

## Demo

See `docs/DEMO_FLOW.md`. ~3:30 walkthrough.

## Status

Hackathon prototype. Not clinically validated. Read `docs/SRS.md` §31 for limitations.
