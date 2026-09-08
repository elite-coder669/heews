# HEEWS — Hyderabad Heat-Health Early Warning System

Municipal heat-risk dashboard: backend (Go) fetches weather, a Python pipe computes WBGT/UTCI risk, and a React frontend maps ward-level heat alerts + action plans.

## Stack
- Backend: Go (net/http, own mux), cmd/server
- Pipeline: Python `ml/run_pipeline.py` — Open-Meteo fetch → WBGT (Liljegren) / UTCI / MRI risk scoring → PipelineDoc JSON
- Frontend: React + Vite (frontend/), API mode types LIVE | MOCK | DEGRADED
- No DB; all state in-memory per process

## Architecture
- `internal/orchestration` — Orchestrator holds wards/risks/thermal/weather/forecast/alerts + state; `RunOnce` mocks live (python3 exec) vs synthetic. repoRoot auto-found (env HEEWS_ROOT or walk-up to `ml/run_pipeline.py`).
- `internal/api` — REST: /health, /version, /wards, /wards/{id}/risk|thermal|weather, /forecast, /alerts, /agent/action-plan, /pipeline/run. Errors `{ok:false,error:{code,message}}`; per-endpoint 404 WARD_NOT_FOUND.
- `internal/config` — env-driven; LIVE mode execs pipeline with Hyderabad lat/lon 17.3850/78.4867 + OPEN_METEO_BASE. Agent: AGENT_LLM_PROVIDER=mock|openrouter, OPENROUTER_API_KEY, LLM_MODEL.
- `internal/agent` — isolated OpenRouter chat-completions client (base https://openrouter.ai/api/v1) + select-and-rank prompt/narrative types; strict-JSON parse with DisallowUnknownFields. Orchestration owns gating + validation.
- Frontend pages/App.tsx — horizon slider (0-5d) drives map bands from /api/forecast cells; 30s health re-sync; 60s alert re-sync; detail drawer per ward; DecisionPanel from agent plan.
- Fixtures: `data/fixtures/wards/hyderabad_wards.geojson` (real polygons → centroids via bbox centre), `data/fixtures/demographics/hyderabad_demo.json`.

## Conventions
- JSON API wrapped `{ok:true,data}` / `{ok:false,error}`; all structs tagged snake_case.
- Bands EXTREME/VERY_HIGH/HIGH/MODERATE/LOW; risk scores 0-100 (MRI).
- Wards keyed `ward_###`; ward_number 0-padded 3 digits.
- Frontend types mirror Go structs exactly (frontend/src/types/index.ts).

## Key Decisions
| Date | Decision | Why |
|------|----------|-----|
| 2026-09-08 | LIVE mode execs python pipeline (90s ctx); MOCK mode synthetic in-process | Single pipeline source of truth; MOCK for dev without API |
| 2026-09-08 | Risk struct keeps nested `risk` payload, adds risk_model_version | Matches Python risk envelope; contract stable |
| 2026-09-08 | Map/counters driven by forecast cell at selected horizon, live risk as fallback | One source of truth for the time slider |
| 2026-09-08 | Degraded mode keeps last good doc, flags fallback_used + DEGRADED health | Stale-data surfacing over blank screens |
| 2026-09-08 | APP_MODE defaults to LIVE; MOCK is explicit opt-in only | Real data is the product; mock hides regressions |
| 2026-09-08 | Live ingestion uses certifi/`/etc/ssl/cert.pem` SSL context | python.org macOS builds bundle no CA store → all fetches failed |
| 2026-09-08 | UTCI Tmrt = T_air + rad delta (cap +15 °C @ 900 W/m²) | Old SW/(5.4σ)^¼ returned ≈196 K (−78 °C) under overcast, poisoning UTCI |
| 2026-09-08 | Decision agent: rule-based first, optional LLM narrative on top, provenance on every field | Plan carries evidence flags, confidence, limitations, and per-action reason+evidence so every claim is inspectable |
| 2026-09-08 | Historical memory: live-run local journal (`data/historical/alert_history.json`) + demo fixture; damped similarity match → `matching_events`, `historical_context.type` local\|demo\|none | Precedent provenance over invented "case studies"; LIVE persists, MOCK/degraded never persist |
| 2026-09-08 | LLM implementation isolated in `backend/internal/agent` (OpenRouter client + select-and-rank prompt/narrative types); orchestration owns gating + validation | OrdfsDeterministic plan is the source of truth |
| 2026-09-08 | LLM may only select+rank from deterministic candidate pool; response treated as proposal, validated by `applyLLMProposal` (ids ∈ pool, priority ∈ LOW/MEDIUM/HIGH, precedent_ids must match known events, strict-JSON unknown field = reject); any failure → fallback_used | LLM cannot invent actions/numbers/evidence; rejection keeps deterministic plan intact |
| 2026-09-08 | Action-plan is horizon-scoped: single ward + `horizon_hours>0` builds the whole plan from that ward's forecast cell (`scope`/`scope_ward_id`/`scope_horizon_hours`/`scope_label`); `priority_wards` always current-risk ranked | Current metrics never mix with future reasoning; slider drives map AND selected-ward reasoning consistently (spec m0618) |
| 2026-09-08 | Municipal alerts live in a separate `o.manualAlerts` slice, prepended in ListAlerts, capped ~100, never overwritten by pipeline runs; `POST /api/alerts` (source=municipal, agent_run_id/agent_version/fallback_used) + `GET /api/agent/priority` endpoints added | Human-approved alerts outlive live-run pipeline overwrites; single deterministic ranking endpoint |
| 2026-09-08 | Console agent is PROACTIVE (no "Ask Decision Support" button); selection+horizon auto-fetch the plan; Review Actions → human approval → Issue Alert reuses validated plan; Public Advisory downstream of approved alert | Agent embedded in municipal workflow (Map → risk → why → history → review → alert → advisory), never autonomous dispatch |

## Current State
Server boots instantly with a synthetic baseline and runs the pipeline in a background goroutine (LIVE execs python async; MOCK is in-process); no render is blocked on ML inference. Default mode is LIVE and verified end-to-end against real Open-Meteo: SSL CA store fix, sane UTCI (T_air + rad delta), live weather/risk/forecast per ward. Frontend boot is 4 parallel calls (health/version/wards/forecast) with zero per-ward fan-out; map/counters drive off the forecast cells at the selected horizon; a 15s poll swaps in new data when `last_run` changes and self-recovers the boot if the backend comes up late. Decision agent (agent-v2.0.0) ships rule-based plans with provenance (evidence/confidence/limitations, per-action reason+evidence) plus historical-memory matching (learned live journal, demo fixture in MOCK/HEAT_DEMO_MEMORY). The LLM layer (`nvidia/nemotron-3.5-lightning:free` via OpenRouter) selects+ranks from the deterministic candidate pool and is verified live: `/api/agent/action-plan` returns fallback_used=false, LLM prose that only references pipeline numbers, and a validated reordered action subset.

Municipal Heat Intelligence Console upgrade (spec m0618) shipped. RiskMap = ward MRI choropleth with centroid CircleMarkers + lightweight metric tooltips (Ward#/MRI/WBGT/UTCI/band, never opens the agent panel); click opens the Ward Intelligence Panel. Forecast time slider (NOW→5D) drives the map AND the selected ward's agent reasoning consistently via the horizon-scoped plan (`scope:forecast` at `horizon_hours=day*24`); current vs future metrics never mix. Agent is PROACTIVE — selection+horizon auto-fetch the plan (no Ask Decision Support button). Review Actions → human approval → Issue Alert reuses the validated plan (never regenerates); Public Advisory is downstream of the approved alert, citizen-friendly (no MRI terms, never invents emergency info). Top Priority Wards panel is fed directly by the single deterministic `GET /api/agent/priority` ranking (click→focus+zoom). Alert Center rows click→focus+open drawer. Degraded/LLM-fail shows the deterministic plan + "AI narrative unavailable — deterministic decision plan shown" badge (fallback_used). Honest provenance: ward boundaries are synthetic demo fixtures (not real GHMC), surfaced in the DetailDrawer footnote + console footer. No per-ward fan-out (bulk /api/wards + /api/forecast).

Verified LIVE end-to-end via curl: horizon plan (ward_009 @48h → MODERATE "Thu 10", current priority stays ward_009 48.6 HIGH), priority ranking top5, POST /api/alerts (mun. first in GET, 404 on unknown ward), fallback failure test (bogus key → fallback_used=true, deterministic actions intact). Frontend `npm run build` passes (tsc+vite). Interactive point-and-click walkthrough still needs the Browser MCP extension connected (was not connected during testing). NEXT: connect browser to drive the 30-step click-through, if desired.