# Architecture

This document accompanies `SRS.md` Section 7. It provides deeper detail on each component, edge, and failure boundary.

## 1. Layered View

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND (React + Leaflet/MapLibre)                          │
│   - Map   - Ward detail   - Forecast   - Alerts   - Actions  │
└─────────────────────────────────────────────────────────────┘
                          │ HTTP / JSON
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ BACKEND (Go)                                                 │
│   /api/*  →  Services  →  Repository  →  PostgreSQL           │
│   Pipeline Orchestrator (cron + on-demand)                   │
└─────────────────────────────────────────────────────────────┘
           │                │                │
           ▼                ▼                ▼
   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
   │ ML SERVICE   │ │ PHYSICS      │ │ AGENT        │
   │ (Python)     │ │ (Python)     │ │ (Python+LLM) │
   │              │ │              │ │              │
   │ weather      │ │ WBGT         │ │ tools        │
   │ spatial      │ │ UTCI         │ │ reasoning    │
   │ features     │ │ solar geom   │ │ action plan  │
   │ risk model   │ │              │ │              │
   └──────────────┘ └──────────────┘ └──────────────┘
                          │
                          ▼
              ┌─────────────────────────┐
              │ Open-Meteo (LIVE)       │
              │ data/fixtures/* (MOCK)  │
              └─────────────────────────┘
```

## 2. Per-Component Detail

### 2.1 Weather Ingestion
- **Owns:** HTTP fetching, unit normalization, raw payload storage.
- **Reads:** Open-Meteo.
- **Writes:** `raw_weather_staging`, `ward_weather`.
- **External:** Open-Meteo (no API key required).

### 2.2 Spatial Downscaling
- **Owns:** Ward polygon loading, centroid, IDW.
- **Reads:** `ward_boundaries`, `raw_weather_staging`.
- **Writes:** `ward_weather`.

### 2.3 Physics Engine
- **Owns:** Liljegren WBGT, simplified WBGT fallback, UTCI, solar geometry.
- **Reads:** `ward_weather`.
- **Writes:** `ward_thermal_stress`.

### 2.4 Risk Model
- **Owns:** Feature engineering, MRI scoring, banding.
- **Reads:** `ward_thermal_stress`, `ward_demographics`.
- **Writes:** `ward_mortality_risk`.

### 2.5 Decision Agent
- **Owns:** Rule-based action planning, provenance (evidence + confidence + limitations), historical-memory matching, optional LLM narrative layer, action-plan API.
- **Reads:** `ward_risk`, `ward_forecast`, `ward_demographics`, historical memory (learned local journal `data/historical/alert_history.json` + optional demo fixture).
- **Writes:** `agent_decisions` (in-memory), `alerts`, local historical journal (LIVE pipeline runs only).

### 2.6 Backend
- **Owns:** HTTP API, persistence, orchestration, alerting.
- **Reads/Writes:** All tables; coordinates ML + agent.

### 2.7 Frontend
- **Owns:** Map, charts, panels.
- **Reads:** Backend API only.

## 3. Contract Boundaries

Contracts are JSON schemas declared in `API_CONTRACTS.md`. They are enforced at module boundaries; cross-domain validation tests run in CI.

## 4. Reliability Strategy

- Every external fetch has retry + cached fallback.
- Every long-running stage persists progress.
- Agent has a hard timeout and rule-based fallback.
- Frontend has offline banner when backend health check fails.

## 5. Observability

Every log line includes:
```
ts, level, service, pipeline_run_id, ward_id, stage, latency_ms, physics_version, risk_model_version
```

## 6. Why This Architecture

- **Deterministic by default** (science runs offline reproducible).
- **Agentic only at decision boundary** (LLM never touches data).
- **Parallelizable** (ML, Backend, Frontend, Agent work independently).
- **Demoable in MOCK mode** (no internet required).
- **Honest** (no mortality claims, no fabricated data).