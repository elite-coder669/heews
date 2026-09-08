# Software Requirements Specification
## Extreme Heatwave Early Warning & Human Thermal Stress Index

**Document ID:** HEEWS-SRS-001
**Version:** 1.0
**Status:** Approved for Hackathon MVP
**Classification:** Decision-Support Prototype (NOT a clinically validated mortality prediction system)
**Last Updated:** 2026-09-07

---

## 0. Reading Guide

This document is the authoritative engineering specification for a 24-hour hackathon prototype that transforms weather forecasts into municipal-level heat-health action recommendations for Hyderabad.

**How to read this SRS:**

- Sections 1–3 explain the system in plain language for new engineers.
- Sections 4–7 list functional and non-functional requirements.
- Sections 8–17 define the architecture, contracts, and science.
- Sections 18–27 give team- and agent-level instructions.
- Sections 28–36 cover testing, failure handling, demo flow, and the Definition of Done.

If you are joining as an ML, Backend, Frontend, or Agent engineer, jump to **Section 26** for your copy-pasteable agent prompt.

---

## 1. Executive Summary

### 1.1 What this system does

The **Heatwave Early Warning & Human Thermal Stress Index (HEEWS)** system is a decision-support prototype for the city of Hyderabad. It pulls a 3–5 day weather forecast, converts it into per-ward human thermal-stress metrics (WBGT and UTCI), combines those metrics with ward-level vulnerability data, and produces a population-level heat health risk index (MRI) per ward. A reasoning agent then recommends concrete municipal interventions.

### 1.2 What this system is NOT

- It is **not** a clinical or epidemiological mortality predictor.
- It does **not** forecast weather; Open-Meteo does.
- It does **not** replace physicians, public health authorities, or municipal command structures.
- It is **not** an individual-level medical diagnostic system.

### 1.3 The core transformation

```
WEATHER  →  PHYSIOLOGICAL HEAT STRESS  →  POPULATION RISK  →  DECISION  →  ACTION
```

Every component in the architecture exists to implement exactly one arrow in this chain.

---

## 2. Problem Understanding

### 2.1 The problem

Cities like Hyderabad experience extreme heatwaves that cause preventable mortality and morbidity, especially among outdoor workers, the elderly, and residents of informal housing. Most existing systems either:

1. Provide raw temperature warnings (which underestimate danger on humid, calm, sunny days), or
2. Provide generic citywide advisories that do not target the most vulnerable wards.

### 2.2 Why raw temperature is insufficient

Air temperature (°C) tells only part of the story. Human heat stress is governed by:

- **Air temperature** (T)
- **Humidity** (RH) — affects evaporative cooling
- **Wind** (v) — affects convective cooling
- **Solar / thermal radiation** (Tmrt) — adds direct heat load
- **Pressure** — affects evaporation and respiration

A 41 °C dry, breezy day is physiologically far less dangerous than a 41 °C humid, still, intensely sunny day. The system must therefore compute integrated thermal-stress indices, not just T.

### 2.3 Why ward-level resolution matters

Citywide averages mask sharp intra-urban risk gradients caused by:

- Urban Heat Island (UHI) intensity differences across wards.
- Local vegetation / built-up area ratios.
- Vulnerability composition (elderly ratio, outdoor-worker density, housing quality).
- Proximity to cooling centers, hospitals, water points.

Heat-health decisions must therefore be made at **ward resolution**.

### 2.4 Why a reasoning agent is useful — and what it must not do

A reasoning agent can:

- Compare multiple wards simultaneously.
- Combine severity, vulnerability, and available resources.
- Generate plain-language explanations for decision-makers.
- Prioritize interventions when not all can be funded immediately.

But the agent **must not**:

- Invent weather values.
- Calculate WBGT or UTCI itself.
- Override scientific outputs.
- Claim mortality certainty.

The agent is a **reasoning layer over deterministic science**, never a substitute for it.

---

## 3. End-to-End System Explanation (Walkthrough)

A worked example with illustrative values, clearly labeled as illustrative:

```
Ward A (illustrative)
  Temperature:       41.2 °C
  Relative humidity: 58 %
  Wind speed:        2.1 m/s
  Shortwave rad.:    710 W/m²
  Pressure:          1004 hPa

  ↓
WBGT (Liljegren)  =  33.8 °C        (extreme on ISO 7243 scale)
UTCI              =  44.6 °C        (very strong heat stress)

  ↓
Vulnerability:
  elderly ratio         = 0.11
  outdoor worker share  = 0.18
  informal housing      = 0.22

  ↓
MRI = 82 / 100  →  Band = EXTREME

  ↓
Decision Agent reasoning:
  "EXTREME WBGT persists for 3 forecast days, with peak afternoon UTCI ≥ 44 °C,
   high outdoor-worker share (18 %), and elevated elderly ratio.
   Persistent multi-day heat reduces nighttime recovery."

  ↓
Recommended actions:
  - Activate ward-level cooling center (HIGH)
  - Issue afternoon outdoor-work advisory (HIGH)
  - Pre-position ORS and water points (MEDIUM)
  - Schedule welfare-check visits to elderly (MEDIUM)
  - Re-check risk in 60 minutes (forecast updating)
```

This walkthrough is the conceptual spine of the entire system.

---

## 4. Goals and Non-Goals

### 4.1 Goals

| # | Goal | Priority |
|---|------|----------|
| G1 | Fetch Open-Meteo forecast for Hyderabad | MUST |
| G2 | Downscale to ward-level weather | MUST |
| G3 | Compute WBGT and UTCI per ward per hour | MUST |
| G4 | Generate 3–5 day forecast heat-health risk per ward | MUST |
| G5 | Produce an explainable vulnerability-aware MRI | MUST |
| G6 | Provide a reasoning agent that proposes municipal actions | MUST |
| G7 | Render a GIS dashboard of Hyderabad ward risk | MUST |
| G8 | Provide a public-friendly summary view | SHOULD |
| G9 | Run end-to-end on synthetic data when Open-Meteo is offline | MUST |
| G10 | Demonstrate end-to-end demo in <5 minutes | MUST |

### 4.2 Non-Goals (Out of Scope for MVP)

- Clinically validated mortality prediction.
- Real-time individual-level health diagnosis.
- Hospital bed-level resource tracking.
- Cross-city federation.
- SMS / push delivery channels (interface only, no transport integration).
- Authenticated municipal workflows (read-only dashboard for MVP).
- Telemetry from weather stations.
- ML model training on labeled mortality ground truth (not available).

---

## 5. Functional Requirements

Each requirement has a unique ID and a MUST / SHOULD / COULD / CUT badge.

### 5.1 Weather Ingestion

| ID | Requirement | Priority |
|---|---|---|
| FR-W-001 | Fetch current weather for Hyderabad bounding box | MUST |
| FR-W-002 | Fetch 3–5 day hourly forecast | MUST |
| FR-W-003 | Normalize units (km/h → m/s for wind, hPa → Pa) | MUST |
| FR-W-004 | Validate presence of required fields | MUST |
| FR-W-005 | Store raw_weather_staging rows with source timestamp | MUST |
| FR-W-006 | Cache last-good response per pipeline_run_id | MUST |
| FR-W-007 | Support deterministic mock fixtures | MUST |
| FR-W-008 | Surface fetch errors with explicit codes | MUST |

### 5.2 Spatial Processing

| ID | Requirement | Priority |
|---|---|---|
| FR-S-001 | Load ward polygon GeoJSON | MUST |
| FR-S-002 | Validate polygon geometry (non-empty, valid CRS) | MUST |
| FR-S-003 | Assign stable ward_id (UUID v5 over canonical name + ward number) | MUST |
| FR-S-004 | Compute centroid latitude/longitude per ward | MUST |
| FR-S-005 | Run IDW interpolation from weather grid → centroid | MUST |
| FR-S-006 | Handle missing weather points gracefully | MUST |
| FR-S-007 | Reject duplicate ward_number without unique stable_id | MUST |
| FR-S-008 | Cache spatial features for offline fallback | SHOULD |

### 5.3 Physics Engine (WBGT + UTCI)

| ID | Requirement | Priority |
|---|---|---|
| FR-P-001 | Compute solar geometry (zenith, azimuth) for any lat/lon/datetime | MUST |
| FR-P-002 | Compute WBGT using Liljegren formulation (when radiation available) | MUST |
| FR-P-003 | Compute simplified WBGT fallback when radiation missing | MUST |
| FR-P-004 | Compute UTCI from T, RH, wind, Tmrt | MUST |
| FR-P-005 | Validate all physics inputs and reject impossible values | MUST |
| FR-P-006 | Emit thermal_stress JSON with explicit units | MUST |
| FR-P-007 | Pin physics_version string per run | MUST |
| FR-P-008 | Provide reference test cases for known WBGT/UTCI values | MUST |

> **Note:** Liljegren WBGT requires direct + diffuse radiation. If Open-Meteo does not return them, the system MUST degrade gracefully (use simplified WBGT and flag the run as `wbgt_quality=degraded`).

### 5.4 Risk Model (MRI)

| ID | Requirement | Priority |
|---|---|---|
| FR-R-001 | Construct thermal features (WBGT, UTCI, T, RH, v, rad) | MUST |
| FR-R-002 | Construct temporal features (current, peak, multi-day persistence) | MUST |
| FR-R-003 | Construct vulnerability features from demographics | MUST |
| FR-R-004 | Compute MRI in [0, 100] range | MUST |
| FR-R-005 | Map MRI → risk_band ∈ {LOW, MODERATE, HIGH, VERY_HIGH, EXTREME} | MUST |
| FR-R-006 | Return top contributing factors (feature importances or simple deltas) | MUST |
| FR-R-007 | Pin risk_model_version string per run | MUST |
| FR-R-008 | Be deterministic (same inputs → same outputs) | MUST |

### 5.5 Agent (Decision Layer)

| ID | Requirement | Priority |
|---|---|---|
| FR-A-001 | Agent retrieves risk per ward via tool | MUST |
| FR-A-002 | Agent retrieves forecast per ward via tool | MUST |
| FR-A-003 | Agent retrieves vulnerability per ward via tool | MUST |
| FR-A-004 | Agent retrieves available resources (cooling centers, hospitals) | MUST |
| FR-A-005 | Agent reasons over ranked wards and proposes actions | MUST |
| FR-A-006 | Agent returns strictly structured JSON | MUST |
| FR-A-007 | Agent never invents weather or thermal values | MUST |
| FR-A-008 | Agent never claims mortality certainty | MUST |
| FR-A-009 | Agent suggests re-check time based on forecast volatility | MUST |
| FR-A-010 | Agent can fall back to rule-based action if LLM unavailable | MUST |

### 5.6 Backend

| ID | Requirement | Priority |
|---|---|---|
| FR-B-001 | Expose REST API for wards / weather / thermal / risk / alerts | MUST |
| FR-B-002 | Run ingestion → physics → risk → agent on a schedule | MUST |
| FR-B-003 | Persist all intermediate outputs to database | MUST |
| FR-B-004 | Serve dashboard frontend | MUST |
| FR-B-005 | Provide health and version endpoints | MUST |
| FR-B-006 | Log pipeline_run_id and physics_version per response | MUST |
| FR-B-007 | Support MOCK and LIVE modes | MUST |

### 5.7 Frontend

| ID | Requirement | Priority |
|---|---|---|
| FR-F-001 | Render Hyderabad ward choropleth map | MUST |
| FR-F-002 | Color wards by risk_band | MUST |
| FR-F-003 | Show ward detail panel (T, RH, v, rad, WBGT, UTCI, MRI, band) | MUST |
| FR-F-004 | Show 3–5 day forecast timeline | MUST |
| FR-F-005 | Show agent recommendation panel | MUST |
| FR-F-006 | Show alert center | MUST |
| FR-F-007 | Display explanatory reasons for the risk | MUST |
| FR-F-008 | Never compute WBGT/UTCI on the client | MUST |

---

## 6. Non-Functional Requirements

For an MVP, MVP-bar must hold; production-bar is documented for future evolution.

| Concern | MVP Bar | Production Bar |
|---|---|---|
| Performance | p95 ward risk endpoint < 500 ms (cached) | < 100 ms with horizontal scaling |
| Reliability | Ingestion retries 3× with backoff | Multi-region, dead-letter queue |
| Observability | JSON logs with pipeline_run_id | Distributed tracing, metrics |
| Explainability | Top-3 contributing factors per ward | Full SHAP / counterfactuals |
| Reproducibility | Pin physics_version and risk_model_version | Immutable model artifacts |
| Maintainability | Single-command local run | CI/CD, lint, type-check |
| API consistency | Stable JSON contracts | OpenAPI 3.1 spec published |
| Fault tolerance | MOCK mode fallback | Circuit breakers |
| Security | No PII; risk-only outputs | RBAC, audit log |
| Privacy | Aggregate ward-level only | No PHI ever stored |
| Scalability | One city | Multi-city federation |
| Testability | Per-domain unit + integration tests | 90 %+ coverage |

---

## 7. Architecture

### 7.1 Domain separation

The system is split into **six orthogonal domains**. Each owns a single responsibility and exposes a stable contract.

```
DOMAIN 1 — WEATHER + SPATIAL   (data acquisition, geo)
DOMAIN 2 — PHYSICS             (deterministic thermal stress)
DOMAIN 3 — ML / RISK           (population-level risk scoring)
DOMAIN 4 — AGENT               (decision reasoning)
DOMAIN 5 — BACKEND             (orchestration, persistence, API)
DOMAIN 6 — FRONTEND            (visualization)
```

### 7.2 High-level data flow

```
                 ┌──────────────────┐
                 │   Open-Meteo     │
                 └────────┬─────────┘
                          ↓
                 ┌──────────────────┐
                 │ Weather Ingestion│
                 └────────┬─────────┘
                          ↓
                 ┌──────────────────┐
                 │ Spatial Downscale│   (IDW, ward centroids)
                 └────────┬─────────┘
                          ↓
                 ┌──────────────────┐
                 │ Ward Weather     │
                 └────────┬─────────┘
                          ↓
              ┌───────────┴───────────┐
              ↓                       ↓
       ┌──────────────┐        ┌──────────────┐
       │ WBGT / UTCI  │        │ Demographics │
       │ (Liljegren)  │        │  Vulnerability     │
       └──────┬───────┘        └──────┬───────┘
              └───────────┬───────────┘
                          ↓
                 ┌──────────────────┐
                 │ Risk / MRI Model │
                 └────────┬─────────┘
                          ↓
                 ┌──────────────────┐
                 │ Decision Agent   │
                 └────────┬─────────┘
                          ↓
                 ┌──────────────────┐
                 │ Backend / Alerts │
                 └────────┬─────────┘
                          ↓
                 ┌──────────────────┐
                 │ GIS Dashboard    │
                 └──────────────────┘
```

### 7.3 Component contract summary

| Edge | Producer | Consumer | Interface |
|---|---|---|---|
| weather → thermal | Weather Ingestion | Physics | `ward_weather` table / JSON |
| thermal → risk | Physics | Risk Model | `ward_thermal_stress` table / JSON |
| risk → agent | Risk Model | Agent | `ward_mortality_risk` table / JSON |
| agent → backend | Agent | Backend | Structured `decision_plan` JSON |
| backend → frontend | Backend | Frontend | REST `/api/*` |

---

## 8. Domain Ownership Matrix

| Domain | Owner Team | Inputs | Outputs | Tech Stack | Forbidden |
|---|---|---|---|---|---|
| Weather + Spatial | ML/DS | Open-Meteo API, ward GeoJSON | Normalized `ward_weather` | Python | Modifying thermal formulas |
| Physics | ML/DS | `ward_weather` | `ward_thermal_stress` (WBGT, UTCI) | Python | Modifying backend contracts |
| Risk | ML/DS | Thermal + vulnerability | `ward_mortality_risk` (MRI, band) | Python / ONNX | Producing narrative text |
| Agent | Agent engineer | Risk + context + tools | Structured `decision_plan` | Python + LLM | Calculating WBGT/UTCI |
| Backend | Backend | Domain outputs | REST API, alerts | Go | Re-implementing physics |
| Frontend | Frontend | Backend API | Dashboard | React | Calculating scientific values |

**Rule:** A team may consume another team's outputs but must not modify them. To change a contract, both teams must agree and the change must be recorded.

---

## 9. Data Flow (Step-by-Step)

1. Backend scheduler triggers `pipeline_run_id = uuid4()`.
2. Weather domain fetches Open-Meteo (current + forecast).
3. Spatial domain computes ward centroids, runs IDW.
4. Physics domain computes WBGT (Liljegren) and UTCI.
5. Risk domain computes MRI and band.
6. Agent domain receives top-K wards and proposes actions.
7. Backend persists all stages with the run ID.
8. Frontend polls backend and renders map.

---

## 10. Weather System

### 10.1 Source

Open-Meteo `https://api.open-meteo.com/v1/forecast` (free, no API key).

### 10.2 Required variables

```
temperature_2m           °C
relative_humidity_2m     %
wind_speed_10m           km/h   → convert to m/s
shortwave_radiation      W/m²
direct_radiation         W/m²
diffuse_radiation        W/m²
surface_pressure         hPa
cloud_cover              %
```

### 10.3 Unit normalization rules

- Wind: multiply km/h by `1000/3600` = `0.277777...` to get m/s.
- Pressure: multiply hPa by `100` to get Pa.
- Radiation: leave as W/m².
- Timezone: `Asia/Kolkata`.

### 10.4 Operational input layers

| Layer | Use | Retention |
|---|---|---|
| Current weather | Latest refresh of ward_weather | 24 h |
| Forecast | 3–5 day forecast, hourly | Next 5 days |
| Last 10 days archive | Persistence features | 10 d rolling |
| Historical | Optional calibration only | N/A |

### 10.5 Failure handling

If Open-Meteo returns HTTP error or times out, the system uses the **last good response** (max age 6 h) and emits `weather_source = cached`.

---

## 11. Spatial System

### 11.1 Ward data

- Source: Hyderabad ward GeoJSON (GHMC / open data portals).
- Each polygon includes `ward_number`, `ward_name`, `zone`.

### 11.2 Stable identifier

```
ward_id = uuid5(NAMESPACE_URL, f"{canonical_name}|{ward_number}")
```

Ward numbers alone are not unique across zones.

### 11.3 Centroid

```
centroid = polygon.representative_point()
lat = centroid.y
lon = centroid.x
```

### 11.4 Spatial interpolation (IDW)

```
IDW(centroid) = Σ w_i * v_i   /   Σ w_i
where w_i = 1 / d_i^p,  d_i = haversine(centroid, grid_point_i), p = 2
```

Nearby weather points dominate; distant points fade with inverse squared distance. For the MVP, the system uses Open-Meteo's gridded forecast directly sampled at centroid lat/lon, treating the API as a virtual grid; IDW is the documented fallback if raw grid data becomes available.

### 11.5 Edge cases

- Empty polygon → log + exclude from run.
- Duplicate ward_number → resolve by `ward_id` (UUID).
- Missing weather → flag ward as `data_degraded`.

---

## 12. Physics System

### 12.1 Why thermal stress ≠ air temperature

A 41 °C afternoon is not 41 °C physiologically. The body gains heat from the sun (radiation), loses heat via sweat evaporation (humidity- and wind-dependent), and via convection (wind). Two days with identical air temperature can produce wildly different physiological outcomes.

### 12.2 WBGT (Wet Bulb Globe Temperature)

WBGT blends:

- Natural wet-bulb temperature (humidity-driven evaporative cooling).
- Globe temperature (radiation + convection).
- Shade dry-bulb.

**Liljegren method** uses direct + diffuse shortwave radiation, solar zenith angle, wind speed, humidity, and pressure. If any of those are unavailable, the system MUST fall back to a simplified empirical WBGT and emit `wbgt_quality = degraded`.

### 12.3 UTCI (Universal Thermal Climate Index)

UTCI is the equivalent air temperature of a reference environment that produces the same physiological response as the actual environment. Inputs:

- T (air temperature, °C)
- Tmrt (mean radiant temperature, °C)
- v (wind speed at 10 m, m/s)
- RH (relative humidity, %)

UTCI is computationally more expensive than WBGT but captures radiation more accurately.

### 12.4 Solar geometry

For any (lat, lon, datetime, timezone), compute:

- Solar declination δ
- Equation of time EoT
- Hour angle H
- Zenith angle Z

These feed both WBGT (direct/diffuse separation) and UTCI (Tmrt).

### 12.5 Physics API

```
POST /physics/wbgt         {T, RH, v, p, direct_rad, diffuse_rad, lat, lon, ts} → {wbgt_c, quality}
POST /physics/utci         {T, RH, v, tmrt}                                    → {utci_c}
POST /physics/thermal-stress {T, RH, v, p, direct_rad, diffuse_rad, lat, lon, ts} → {wbgt_c, utci_c, quality}
```

---

## 13. ML / Risk System

### 13.1 Honest framing

No validated mortality ground truth is available to the team. The MRI is therefore a **physiologically grounded, vulnerability-aware heat-health risk indicator**, not a clinical mortality prediction. The system states this explicitly in the UI and PDFs.

### 13.2 Feature groups

| Group | Features |
|---|---|
| Thermal | WBGT, UTCI, T, RH, v, rad |
| Temporal | current stress, rolling 24 h mean, peak next 72 h, multi-day persistence, overnight recovery |
| Vulnerability | elderly ratio, outdoor-worker share, informal housing share, population density |
| Contextual (optional) | distance to nearest cooling center, hospital capacity index |

### 13.3 MRI formulation (MVP)

A transparent, deterministic weighted score in [0, 100]:

```
MRI = 100 * (
    0.35 * norm(peak_wbgt_next_72h) +
    0.20 * norm(utci_now) +
    0.15 * norm(persistence_days) +
    0.20 * norm(vulnerability_index) +
    0.10 * norm(overnight_recovery_gap)
)
```

Each `norm(x)` maps to [0, 1] using a documented piecewise-linear function. This is **calibrated against WBGT/UTCI thresholds**, not against observed mortality.

### 13.4 Risk bands

| Band | MRI range | Color (hex) |
|---|---|---|
| LOW | 0–19 | `#16A34A` |
| MODERATE | 20–39 | `#F59E0B` |
| HIGH | 40–59 | `#EA580C` |
| VERY_HIGH | 60–79 | `#DC2626` |
| EXTREME | 80–100 | `#7C1D6F` |

---

## 14. Agent System

### 14.1 Role

The agent is a **decision-support and orchestration layer**, never a replacement for science.

### 14.2 What the agent may do

- Retrieve risk and forecast per ward.
- Retrieve vulnerability.
- Retrieve resources (cooling centers, hospitals).
- Compare wards.
- Prioritize interventions.
- Generate explanations.
- Generate action plans.
- Decide re-check cadence.

### 14.3 What the agent must NOT do

- Invent weather or thermal values.
- Calculate WBGT or UTCI itself.
- Fabricate medical facts or mortality statistics.
- Override validated physics outputs.
- Claim mortality certainty.
- Modify raw data.

### 14.4 Agent tools

| Tool | Purpose |
|---|---|
| `get_ward_risk(ward_id)` | Retrieve MRI + band + top factors |
| `get_forecast(ward_id, hours)` | Retrieve forecast |
| `get_thermal_stress(ward_id)` | Retrieve WBGT, UTCI |
| `get_vulnerability(ward_id)` | Retrieve vulnerability features |
| `get_resources(ward_id)` | Retrieve cooling centers, hospitals |
| `get_previous_alerts(ward_id)` | Retrieve alert history |
| `create_alert(payload)` | Persist alert |
| `generate_action_plan(payload)` | Produce structured plan |

---

## 15. Backend System

### 15.1 Responsibilities

- HTTP API.
- Pipeline orchestration.
- Database persistence.
- Alert evaluation.
- ML/agent integration.
- Frontend serving.

### 15.2 API surface (high level)

| Method | Route | Purpose |
|---|---|---|
| GET | `/api/health` | Liveness |
| GET | `/api/version` | Build / physics / risk versions |
| GET | `/api/wards` | List wards |
| GET | `/api/wards/{id}` | Ward detail |
| GET | `/api/wards/{id}/weather` | Weather |
| GET | `/api/wards/{id}/thermal` | Thermal stress |
| GET | `/api/wards/{id}/risk` | MRI + band |
| GET | `/api/forecast` | Citywide forecast |
| GET | `/api/alerts` | Active alerts |
| POST | `/api/agent/action-plan` | Generate action plan |
| POST | `/api/pipeline/run` | Trigger full run |

Full schemas in `docs/API_CONTRACTS.md`.

---

## 16. Frontend System

### 16.1 Screens

1. **City Overview** — choropleth map of wards.
2. **Ward Detail** — full ward panel.
4. **Forecast Timeline** — 3–5 day risk timeline.
4. **Alert Center** — alerts with status.
5. **Decision Panel** — agent explanation + actions.

### 16.2 Frontend constraints

- Never compute WBGT/UTCI.
- Never claim mortality.
- Show band + score, never single-point death predictions.
- Display `physics_version` and `risk_model_version`.

---

## 17. Database Schema

> **Implementation assumption and is not specified in the source document.**

```
ward_boundaries            ward_id (PK), ward_number, ward_name, zone, geometry (GeoJSON)
ward_demographics          ward_id (PK, FK), elderly_ratio, outdoor_worker_share,
                           informal_housing_share, population, source_year
raw_weather_staging        id (PK), fetched_at, source, payload (JSONB), valid_until
ward_weather               ward_id, forecast_ts (PK), temperature_c, rh, wind_ms,
                           shortwave_rad_wm2, pressure_pa, cloud_cover
ward_thermal_stress        ward_id, forecast_ts (PK), wbgt_c, utci_c, wbgt_quality,
                           physics_version
ward_mortality_risk        ward_id, computed_at (PK), mri, band, top_factors (JSONB),
                           risk_model_version
cooling_centers            id (PK), ward_id, name, capacity, hours
hospital_capacity          id (PK), ward_id, facility_name, beds_available, updated_at
alerts                     id (PK), created_at, ward_id, severity, headline, body,
                           recommended_actions (JSONB), agent_run_id, status
pipeline_runs              id (PK), started_at, finished_at, status, stages (JSONB)
agent_decisions            id (PK), run_id, ward_id, plan (JSONB), reasoning (text)
```

`ward_id` is the stable UUID. Foreign keys reference it, not `ward_number`.

---

## 18. API Contracts (Summary)

See `docs/API_CONTRACTS.md` for full schemas.

### 18.1 Example: `GET /api/wards/{id}/risk`

```json
{
  "ward_id": "ward_001",
  "ward_name": "Somajiguda",
  "computed_at": "2026-09-07T12:00:00+05:30",
  "risk": {
    "score": 82,
    "band": "EXTREME",
    "top_factors": [
      "Peak WBGT next 72h = 34.7 °C",
      "UTCI now = 45.1 °C",
      "Outdoor worker share = 18%",
      "Persistence ≥ 3 days"
    ]
  },
  "thermal": {
    "wbgt_c": 33.8,
    "utci_c": 44.6,
    "wbgt_quality": "ok",
    "physics_version": "wbgt-liljegren-1.0"
  },
  "weather": {
    "temperature_c": 41.2,
    "humidity_pct": 58,
    "wind_ms": 2.1,
    "shortwave_rad_wm2": 710
  },
  "model_version": "risk-v1.0.0"
}
```

---

## 19. Agent Tool Contracts

See `docs/AGENT_SPEC.md`.

### 19.1 Output contract (strict)

```json
{
  "severity": "EXTREME",
  "priority_wards": ["ward_001", "ward_017"],
  "reasoning_summary": "...",
  "key_factors": ["..."],
  "recommended_actions": [
    {"action": "Activate cooling center", "priority": "HIGH", "target": "ward_001"}
  ],
  "public_advisory": "...",
  "recheck_interval_minutes": 60
}
```

---

## 20. Monorepo File Structure

```
heatwave-early-warning/
├── README.md
├── docker-compose.yml
├── .env.example
├── docs/
│   ├── SRS.md
│   ├── ARCHITECTURE.md
│   ├── API_CONTRACTS.md
│   ├── DATA_DICTIONARY.md
│   ├── AGENT_SPEC.md
│   ├── PHYSICS_SPEC.md
│   ├── ML_SPEC.md
│   └── DEMO_FLOW.md
├── backend/
├── ml/
├── agent/
├── frontend/
├── data/
└── scripts/
```

---

## 21. Scaffold Routes

See `docs/API_CONTRACTS.md` and `backend/internal/api/`. Key routes:

| Method | Route | Owner |
|---|---|---|
| GET | `/api/health` | Backend |
| GET | `/api/version` | Backend |
| GET | `/api/wards` | Backend |
| GET | `/api/wards/{id}` | Backend |
| GET | `/api/wards/{id}/weather` | Backend |
| GET | `/api/wards/{id}/thermal` | Backend |
| GET | `/api/wards/{id}/risk` | Backend |
| GET | `/api/forecast` | Backend |
| GET | `/api/alerts` | Backend |
| POST | `/api/agent/action-plan` | Backend + Agent |
| POST | `/api/pipeline/run` | Backend |

---

## 22. Environment Configuration

`.env.example`:

```
APP_MODE=LIVE                 # LIVE | MOCK
OPEN_METEO_BASE=https://api.open-meteo.com/v1/forecast
HYDERABAD_LAT=17.3850
HYDERABAD_LON=78.4867
DB_URL=postgresql://user:pass@localhost:5432/heatwave
AGENT_LLM_PROVIDER=openai     # openai | mock
AGENT_API_KEY=
LOG_LEVEL=info
```

`MOCK` mode swaps all external dependencies for deterministic fixtures under `data/fixtures/`.

---

## 23. Testing Strategy

| Layer | Tests |
|---|---|
| Weather | API parse, unit conversion, missing-field handling |
| Spatial | Geometry validation, centroid, IDW correctness |
| Physics | Reference WBGT/UTCI cases, edge inputs |
| Risk | Determinism, band boundaries |
| Agent | Tool selection, no hallucination, schema validity |
| Backend | Route tests, integration tests, error codes |
| Frontend | Map rendering, API failure handling, ward selection |

---

## 24. Failure Modes

| Failure | Impact | Detection | Fallback |
|---|---|---|---|
| Open-Meteo unavailable | No new forecast | HTTP timeout | Last good response |
| Missing radiation | WBGT degraded | Validation | Simplified WBGT |
| Invalid geometry | Spatial failure | Geometry check | Exclude + log |
| ML failure | No MRI | Inference error | Thermal-only status |
| Agent failure | No recommendation | LLM timeout | Rule-based action plan |
| Backend failure | Dashboard unavailable | Health check | Retry / cached page |

---

## 25. Observability

Every log line includes:

```
pipeline_run_id
weather_fetch_timestamp
ward_id
forecast_timestamp
physics_version
risk_model_version
agent_run_id
alert_id
```

Reproducibility matters because alerts must be auditable.

---

## 26. Security & Safety

- No clinical diagnosis.
- No individual-level prediction.
- No fabricated mortality.
- All numeric outputs include provenance (`physics_version`, `risk_model_version`).
- Agent never overwrites scientific outputs.
- All communications label "decision-support prototype, not medical advice."

---

## 27. 24-Hour Execution Plan

| Hour | Stream A (ML/Physics) | Stream B (Backend) | Stream C (Frontend) | Stream D (Agent) |
|---|---|---|---|---|
| 0–2 | Contracts + fixtures | Repo scaffold + DB | Repo scaffold + UI shell | Repo scaffold + tools |
| 2–5 | Weather ingest + spatial | Routes + DB schema | Map base layer | Tool schemas |
| 5–8 | WBGT + UTCI | Pipeline orchestration | API client + state | Tool implementations |
| 8–11 | Risk model | Persist + serve | Ward detail | Prompt + LLM hook |
| 11–14 | Vulnerability features | Alert persistence | Forecast timeline | Structured output |
| 14–17 | Integration | End-to-end API | Dashboard wiring | Action plan |
| 17–20 | Mock fixtures | Health checks | Polish | Rule-based fallback |
| 20–22 | Tests | Tests | Tests | Tests |
| 22–24 | Demo polish | Demo polish | Demo polish | Demo polish |

Streams parallelize. Integration begins at hour 14.

---

## 28. Domain-Specific Agent Prompts (Copy-Paste)

### Prompt A — ML + Physics Agent

```
You are the ML + Physics engineer for the Heatwave Early Warning hackathon.
You own: weather ingestion, spatial downscaling, WBGT, UTCI, risk model.

Stack: Python 3.11, NumPy, pandas, geopandas, shapely, scikit-learn, optionally PyTorch.

Files you OWN and may modify:
  ml/ingestion/**
  ml/spatial/**
  ml/physics/**
  ml/risk/**
  ml/features/**
  ml/inference/**
  ml/tests/**
  data/fixtures/weather/**
  data/fixtures/wards/**
  data/fixtures/thermal/**

Files you may READ but NOT modify:
  backend/internal/api/**
  agent/tools/**
  docs/API_CONTRACTS.md
  docs/PHYSICS_SPEC.md

Contracts you must obey:
  - ward_weather schema in docs/API_CONTRACTS.md
  - ward_thermal_stress schema in docs/API_CONTRACTS.md
  - ward_mortality_risk schema in docs/API_CONTRACTS.md
  - physics_version and risk_model_version strings

Tasks:
  1. Fetch Open-Meteo forecast + current weather.
  2. Normalize units (km/h → m/s, hPa → Pa).
  3. Load ward GeoJSON; compute stable ward_id UUID v5.
  4. Compute centroids; sample Open-Meteo grid at centroid.
  5. Implement Liljegren WBGT with simplified fallback.
  6. Implement UTCI (use a reference implementation).
  7. Build vulnerability features from ward_demographics.
  8. Build risk model producing MRI ∈ [0, 100] + band.
  9. Emit top-3 contributing factors.
  10. Provide deterministic fixtures under data/fixtures.

Acceptance:
  - pytest passes
  - JSON outputs validate against API_CONTRACTS.md
  - Mock mode works without network
  - physics_version and risk_model_version emitted

Forbidden:
  - Calculating decisions / action plans
  - Calling LLM APIs
  - Modifying backend contracts without coordination
```

### Prompt B — Backend Agent

```
You are the Backend engineer for the Heatwave Early Warning hackathon.
You own: HTTP API, database, pipeline orchestration, alert persistence.

Stack: Go 1.22+, Chi or Echo, pgx, sqlx.

Files you OWN:
  backend/**

Files you may READ:
  ml/**
  agent/**
  docs/API_CONTRACTS.md

Tasks:
  1. Scaffold Go module, DB schema, migrations.
  2. Implement routes per API_CONTRACTS.md.
  3. Implement pipeline scheduler (cron or in-process ticker).
  4. Persist raw_weather_staging, ward_weather, ward_thermal_stress,
     ward_mortality_risk, alerts, pipeline_runs.
  5. Implement MOCK mode flag.
  6. Provide /api/health and /api/version.
  7. Serve frontend build artifacts.
  8. Forward agent requests to the agent service.

Acceptance:
  - go test ./... passes
  - All routes return JSON matching API_CONTRACTS.md
  - MOCK mode works without Open-Meteo
  - pipeline_run_id stamped on every response

Forbidden:
  - Implementing WBGT/UTCI
  - Implementing MRI
  - Modifying agent prompts
```

### Prompt C — Frontend Agent

```
You are the Frontend engineer for the Heatwave Early Warning hackathon.
You own: dashboard, GIS map, charts, ward detail, alert center.

Stack: React 18, Vite, TypeScript, react-leaflet or maplibre-gl, recharts.

Files you OWN:
  frontend/**

Tasks:
  1. Build map of Hyderabad wards with choropleth risk coloring.
  2. Build ward detail panel (T, RH, v, rad, WBGT, UTCI, MRI, band).
  3. Build 3–5 day forecast timeline chart.
  4. Build alert center.
  5. Build decision-support panel.
  6. Consume API per docs/API_CONTRACTS.md.
  7. Display physics_version and risk_model_version.
  8. Add disclaimer banner: "Decision-support prototype. Not medical advice."

Acceptance:
  - npm run build succeeds
  - Map renders all wards
  - Click ward → detail panel populates
  - Disclaimer banner visible
  - No scientific computation in browser

Forbidden:
  - Computing WBGT/UTCI on browser
  - Caching raw weather responses longer than 10 minutes
```

### Prompt D — Decision Agent

```
You are the Decision Agent engineer for the Heatwave Early Warning hackathon.
You own: agent prompts, tools, schemas, workflows, action planning.

Stack: Python 3.11, LangGraph or custom tool loop, OpenAI or compatible LLM.

Files you OWN:
  agent/**

Contracts:
  - Output strictly follows agent output schema in docs/AGENT_SPEC.md
  - Tools per docs/AGENT_SPEC.md (do not invent new tools)
  - Never modify scientific outputs

Tasks:
  1. Implement tools: get_ward_risk, get_forecast, get_thermal_stress,
     get_vulnerability, get_resources, get_previous_alerts, create_alert,
     generate_action_plan.
  2. Implement system prompt enforcing factual grounding + structured output.
  3. Implement retry / fallback rule-based plan if LLM fails.
  4. Validate every output against schema before returning.
  5. Emit agent_run_id on every decision.

Acceptance:
  - Tool tests pass
  - Output validates against schema
  - Rule-based fallback passes golden test cases
  - No scientific value invention in unit tests

Forbidden:
  - Computing WBGT/UTCI
  - Inventing weather values
  - Overriding risk model
  - Claiming mortality certainty
```

---

## 29. Demo Story (3–5 Minutes)

1. Open Hyderabad choropleth. Show ward risk colors.
2. Click a high-risk ward → show WBGT, UTCI, MRI, band, top factors.
3. Switch to forecast timeline → show 3–5 day heat persistence.
4. Open decision panel → ask: "What should the municipality do?"
5. Agent retrieves context, generates structured action plan.
6. Show alert center → new alert appears.
7. Switch to MOCK mode → demo offline determinism.

---

## 30. Competitive Differentiation

The system combines six layers of intelligence:

1. Weather (Open-Meteo)
2. Spatial (ward-level downscaling)
3. Physiological (WBGT, UTCI)
4. Vulnerability (demographics)
5. Temporal (persistence, recovery)
6. Decision (agent over deterministic outputs)

Plus an agentic loop: **Sense → Predict → Reason → Act → Re-evaluate**.

Most existing systems stop at #1 or #2. We go to #6 with explicit, grounded reasoning.

---

## 31. Limitations

- Hackathon prototype. Not clinically validated.
- No ground-truth mortality dataset.
- Open-Meteo resolution (~9 km) may under-resolve UHI.
- No real-time weather station telemetry.
- Agent decisions are advisory, not authoritative.
- MVP is read-only; no municipal workflow automation.

---

## 32. Traceability Matrix

| Requirement | Component | Implementation | Contract | Demo Evidence |
|---|---|---|---|---|
| Fetch weather | ml/ingestion | open_meteo.py | raw_weather_staging | Live fetch in dashboard |
| Downscale to ward | ml/spatial | idw.py | ward_weather | Map colors per ward |
| Compute WBGT | ml/physics | wbgt_liljegren.py | ward_thermal_stress | Ward detail panel |
| Compute UTCI | ml/physics | utci.py | ward_thermal_stress | Ward detail panel |
| Vulnerability features | ml/risk | features.py | ward_demographics | Top factors |
| MRI | ml/risk | risk_model.py | ward_mortality_risk | Map + score |
| Decision | agent | agent.py | decision_plan JSON | Decision panel |
| Alert | backend | alerts.go | alerts table | Alert center |
| Map | frontend | Map.tsx | /api/wards | Live demo |
| Disclaimer | frontend | Disclaimer.tsx | static | Visible banner |

---

## 33. Definition of Done

The MVP is complete only when:

- [ ] Hyderabad ward polygons load
- [ ] Stable ward IDs exist
- [ ] Open-Meteo data is fetched
- [ ] Weather units are normalized
- [ ] Ward-level weather is generated
- [ ] WBGT calculated (Liljegren with fallback)
- [ ] UTCI calculated
- [ ] 3–5 day forecast exists
- [ ] Vulnerability features exist
- [ ] MRI generated
- [ ] Risk bands exist
- [ ] Agent retrieves risk
- [ ] Agent retrieves forecast
- [ ] Agent explains risk
- [ ] Agent generates structured actions
- [ ] Backend exposes APIs
- [ ] Frontend renders map
- [ ] Ward details work
- [ ] Alerts demonstrated
- [ ] MOCK mode works
- [ ] LIVE mode works
- [ ] End-to-end demo runs < 5 minutes

---

## 34. Closing Note

The system is a **decision-support prototype**. It is scientifically grounded, transparent, and reproducible. It does not claim to predict mortality. It exists to give a municipal command center a defensible, ward-by-ward, evidence-based set of options when a heatwave is approaching.

If a new team member cannot understand the architecture, their responsibility, the interfaces, and the implementation plan from this document alone, this document is incomplete.