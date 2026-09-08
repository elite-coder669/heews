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
    "agent_version": "agent-v1.0.0"
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
```json
{
  "ok": true,
  "data": {
    "alerts": [
      {
        "id": "alert_...",
        "ward_id": "ward_001",
        "created_at": "2026-09-07T12:05:00+05:30",
        "severity": "EXTREME",
        "headline": "Heat-health risk EXTREME for Somajiguda",
        "body": "...",
        "recommended_actions": [
          { "action": "Activate cooling center", "priority": "HIGH" }
        ],
        "status": "ACTIVE"
      }
    ]
  }
}
```

## 8. Decision Agent

### `POST /api/agent/action-plan`
Request:
```json
{
  "ward_ids": ["ward_001", "ward_017"],
  "context": "city-wide planning"
}
```
Response:
```json
{
  "ok": true,
  "data": {
    "severity": "EXTREME",
    "priority_wards": ["ward_001", "ward_017"],
    "reasoning_summary": "...",
    "key_factors": ["..."],
    "recommended_actions": [
      { "action": "Activate cooling center", "priority": "HIGH", "target": "ward_001" }
    ],
    "public_advisory": "...",
    "recheck_interval_minutes": 60,
    "agent_run_id": "..."
  }
}
```

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