# API Contracts

All endpoints return JSON. All responses include `pipeline_run_id`, `physics_version`, `risk_model_version` where applicable.

## Common Envelope

```json
{
  "ok": true,
  "data": { ... },
  "meta": {
    "pipeline_run_id": "...",
    "generated_at": "2026-09-07T12:00:00+05:30"
  }
}
```

Error envelope:

```json
{
  "ok": false,
  "error": {
    "code": "WARDS_NOT_FOUND",
    "message": "...",
    "remediation": "..."
  }
}
```

## 1. Health & Version

### `GET /api/health`
```json
{ "ok": true, "data": { "status": "ok", "mode": "LIVE" } }
```

### `GET /api/version`
```json
{
  "ok": true,
  "data": {
    "service": "0.1.0",
    "physics_version": "wbgt-liljegren-1.0",
    "risk_model_version": "risk-v1.0.0",
    "agent_version": "agent-v2.0.0"
  }
}
```

## 2. Wards

### `GET /api/wards`
```json
{
  "ok": true,
  "data": {
    "wards": [
      {
        "ward_id": "ward_001",
        "ward_number": 1,
        "ward_name": "Somajiguda",
        "zone": "Central",
        "centroid": { "lat": 17.43, "lon": 78.49 }
      }
    ]
  }
}
```

### `GET /api/wards/{id}`
```json
{
  "ok": true,
  "data": {
    "ward_id": "ward_001",
    "ward_name": "Somajiguda",
    "zone": "Central",
    "centroid": { "lat": 17.43, "lon": 78.49 },
    "demographics": {
      "elderly_ratio": 0.11,
      "outdoor_worker_share": 0.18,
      "informal_housing_share": 0.22,
      "population": 42000,
      "source_year": 2021
    }
  }
}
```

## 3. Weather

### `GET /api/wards/{id}/weather`
```json
{
  "ok": true,
  "data": {
    "ward_id": "ward_001",
    "source": "open-meteo",
    "fetched_at": "2026-09-07T12:00:00+05:30",
    "current": {
      "temperature_c": 41.2,
      "humidity_pct": 58,
      "wind_ms": 2.1,
      "shortwave_rad_wm2": 710,
      "pressure_pa": 100400,
      "cloud_cover_pct": 30
    },
    "forecast": [
      {
        "ts": "2026-09-07T13:00:00+05:30",
        "temperature_c": 41.7,
        "humidity_pct": 55,
        "wind_ms": 2.0,
        "shortwave_rad_wm2": 760
      }
    ]
  }
}
```

## 4. Thermal Stress

### `GET /api/wards/{id}/thermal`
```json
{
  "ok": true,
  "data": {
    "ward_id": "ward_001",
    "physics_version": "wbgt-liljegren-1.0",
    "current": {
      "wbgt_c": 33.8,
      "utci_c": 44.6,
      "wbgt_quality": "ok"
    },
    "forecast": [
      {
        "ts": "2026-09-07T13:00:00+05:30",
        "wbgt_c": 34.1,
        "utci_c": 45.0,
        "wbgt_quality": "ok"
      }
    ]
  }
}
```

## 5. Risk

### `GET /api/wards/{id}/risk`
```json
{
  "ok": true,
  "data": {
    "ward_id": "ward_001",
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
    "model_version": "risk-v1.0.0",
    "thermal": {
      "wbgt_c": 33.8,
      "utci_c": 44.6,
      "wbgt_quality": "ok"
    }
  }
}
```

Risk bands:

| Band | Score range |
|---|---|
| LOW | 0–19 |
| MODERATE | 20–39 |
| HIGH | 40–59 |
| VERY_HIGH | 60–79 |
| EXTREME | 80–100 |

## 6. Forecast

### `GET /api/forecast`
```json
{
  "ok": true,
  "data": {
    "city": "Hyderabad",
    "horizon_hours": 96,
    "by_ward": [
      {
        "ward_id": "ward_001",
        "series": [
          { "ts": "2026-09-07T13:00:00+05:30", "wbgt_c": 34.1, "mri": 82, "band": "EXTREME" }
        ]
      }
    ]
  }
}
```

## 7. Alerts

### `GET /api/alerts`
Returns pipeline alerts plus any municipal-approved (manual) alerts. Manual alerts are always listed **first**. Pipeline alerts carry no `source` field; manual alerts set `source`, `agent_run_id`, `agent_version`, `fallback_used`.
```json
{
  "ok": true,
  "data": {
    "alerts": [
      {
        "id": "86fd966aa856d37a430fa069d1631928",
        "ward_id": "ward_009",
        "created_at": "2026-09-08T18:11:33+05:30",
        "severity": "HIGH",
        "headline": "Heat alert for Thu 10 — HIGH risk",
        "body": "...",
        "recommended_actions": [
          { "action": "Issue municipal heat-health advisory", "priority": "LOW" }
        ],
        "status": "ACTIVE",
        "source": "municipal",
        "agent_run_id": "run-accept1",
        "agent_version": "agent-v2.0.0",
        "fallback_used": false
      },
      {
        "id": "ae3400f36a5c",
        "ward_id": "ward_001",
        "created_at": "2026-09-08T18:22:36+05:30",
        "severity": "HIGH",
        "headline": "...",
        "body": "...",
        "recommended_actions": [ { "action": "...", "priority": "HIGH" } ],
        "status": "ACTIVE"
      }
    ]
  }
}
```

### `POST /api/alerts`
Issues a municipal alert (human-approved). The alert must reference an existing ward; on an unknown ward it returns `404 WARD_NOT_FOUND`. After review, the alert reuses the validated agent plan as-is (never regenerated).
Request:
```json
{
  "ward_id": "ward_009",
  "severity": "HIGH",
  "headline": "Heat alert for Thu 10 — HIGH risk",
  "body": "LB Nagar is at high heat risk...",
  "recommended_actions": [ { "action": "Issue municipal heat-health advisory", "priority": "LOW", "target": "Hyderabad" } ],
  "agent_run_id": "run-accept1",
  "agent_version": "agent-v2.0.0",
  "fallback_used": false
}
```
Response `201`:
```json
{ "ok": true, "data": { "id": "86fd966aa856d37a430fa069d1631928", "ward_id": "ward_009", "severity": "HIGH", "status": "ACTIVE", "source": "municipal", ... } }
```
- `severity` must be one of `EXTREME`,`VERY_HIGH`,`HIGH`,`MODERATE`,`LOW`, or empty (defaulted). Missing `ward_id`/`headline` → `400 BAD_REQUEST`.
- Municipal alerts live in a separate list and are never overwritten by pipeline runs (which replace only the pipeline-alert slice). The combined list is capped at ~100 entries, newest first.

## 8. Decision Agent

### `POST /api/agent/action-plan`
Request:
```json
{
  "ward_ids": ["ward_009"],
  "context": "municipal planning",
  "horizon_hours": 48
}
```
- `ward_ids`: empty (or `{}`) targets all wards. A **single** ward + `horizon_hours > 0` produces a **forecast-scoped** plan where severity/MRI/WBGT/UTCI/key_factors/why/LLM-context are built from that ward's forecast cell at `idx = clamp(horizon_hours/24, 0, len-1)` — current metrics are never mixed into future reasoning.
- `horizon_hours`: 0 (or omitted) = the existing current-risk plan.
- `priority_wards` is **always** current-risk ranked (never the forecast cell).

Response:
```json
{
  "ok": true,
  "data": {
    "scope": "forecast",
    "scope_ward_id": "ward_009",
    "scope_horizon_hours": 48,
    "scope_label": "Forecast day: Thu 10",
    "severity": "MODERATE",
    "priority_wards": [
      { "ward_id": "ward_009", "ward_name": "...", "score": 48.6, "band": "HIGH" }
    ],
    "reasoning_summary": "...",
    "key_factors": ["..."],
    "recommended_actions": [
      {
        "action": "Activate cooling council centres and night shelters",
        "priority": "HIGH",
        "target": "ward_001",
        "reason": "Severity EXTREME; MRI 51.2...",
        "evidence": ["current_risk", "vulnerability", "forecast", "historical_precedent"]
      }
    ],
    "public_advisory": "...",
    "recheck_interval_minutes": 60,
    "agent_run_id": "...",
    "fallback_used": false,
    "agent_version": "agent-v2.0.0",
    "why_this_matters": "...",
    "historical_context": {
      "found": true,
      "type": "local",
      "event_id": "local-8771ba8d",
      "similarity_reason": "matching vulnerability profile, same ward",
      "source": "local-journal",
      "synthetic": false
    },
    "matching_events": [
      {
        "event_id": "local-8771ba8d",
        "date": "2026-09-08",
        "ward_name": "...",
        "score": 46,
        "band": "HIGH",
        "similarity": 1.0,
        "dimensions": { "wbgt": 0.94, "mri": 0.82 },
        "actions_taken": ["..."],
        "outcome": "..."
      }
    ],
    "evidence": {
      "current_risk": true,
      "vulnerability": true,
      "forecast": true,
      "historical_precedent": true,
      "external_reference": false,
      "general_guidance": false
    },
    "confidence": "HIGH",
    "limitations": ["..."]
  }
}
```
- `scope`: `current` | `forecast` (only when single-ward + horizon); `scope_ward_id` / `scope_horizon_hours` / `scope_label` describe the forecast cell the reasoning is anchored to. When `scope == "current"` these are omitted.
- `priority_wards` sorted by **current** risk score (not the forecast cell), `band` from the MRI band of each ward.
- `historical_context.type`: `local` (learnt journal) | `demo` (synthetic fixture) | `none`.
- `matching_events` (≤3) ranked by damped similarity to the top-risk ward; populated only when a match clears the threshold.
- `evidence`, `confidence` (HIGH/MEDIUM/LOW), and `limitations` make the provenance of each judgement explicit; `external_reference` is always false in this prototype.
- `fallback_used` is true when the LLM proposal is skipped, rejected by the validation gate, or the orchestrator is degraded; the deterministic reasoning then stands in place of an LLM. `reasoning_summary` and `why_this_matters` are LLM-written only when `fallback_used` is false — else they are the deterministic prose.

### `GET /api/agent/priority`
Returns the single deterministic current-risk ranking of all wards (the same ranking inside the action-plan's `priority_wards`). The frontend consumes this directly rather than re-ranking.
```json
{
  "ok": true,
  "data": {
    "priority_wards": [
      { "ward_id": "ward_009", "ward_name": "LB Nagar", "score": 48.6, "band": "HIGH" },
      { "ward_id": "ward_010", "ward_name": "Dilsukhnagar", "score": 47.1, "band": "HIGH" },
      { "ward_id": "ward_004", "ward_name": "Secunderabad", "score": 46.6, "band": "HIGH" }
    ]
  }
}
```

### `GET /api/agent/memory`
Returns the historical-memory status:
```json
{ "ok": true, "data": {
  "local_events": 10,
  "demo_events": 0,
  "enabled": false,
  "synthetic": false,
  "local_file": "/…/data/historical/alert_history.json",
  "demo_file": "/…/data/historical/demo_heat_events.json"
} }
```
- `enabled` = demo fixture loaded (`demo_events > 0`). Demo memory loads in MOCK mode or when `HEAT_DEMO_MEMORY=true`.
- LIVE pipeline runs append an event to the local journal when each run completes; degraded/mock runs never persist.

## 9. Pipeline

### `POST /api/pipeline/run`
Request:
```json
{ "trigger": "manual" }
```
Response:
```json
{
  "ok": true,
  "data": {
    "pipeline_run_id": "...",
    "stages": [
      { "stage": "weather", "status": "ok", "latency_ms": 320 },
      { "stage": "physics", "status": "ok", "latency_ms": 180 },
      { "stage": "risk", "status": "ok", "latency_ms": 60 },
      { "stage": "agent", "status": "ok", "latency_ms": 1450 }
    ]
  }
}
```

## 10. Validation Rules

- All timestamps are ISO-8601 with timezone.
- All temperature in °C, wind in m/s, pressure in Pa, radiation in W/m².
- All scores in [0, 100].
- All band values in {LOW, MODERATE, HIGH, VERY_HIGH, EXTREME}.
- All `ward_id` are UUIDs (text-encoded).