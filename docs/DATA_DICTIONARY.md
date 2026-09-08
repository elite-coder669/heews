# Data Dictionary

## 1. Tables

### `ward_boundaries`
| Column | Type | Notes |
|---|---|---|
| `ward_id` | TEXT (PK) | UUID v5 |
| `ward_number` | INTEGER | may not be unique |
| `ward_name` | TEXT | canonical |
| `zone` | TEXT | GHMC zone |
| `geometry` | JSONB (GeoJSON) | EPSG:4326 |

### `ward_demographics`
| Column | Type | Notes |
|---|---|---|
| `ward_id` | TEXT (PK, FK) | UUID |
| `elderly_ratio` | NUMERIC | [0,1] |
| `outdoor_worker_share` | NUMERIC | [0,1] |
| `informal_housing_share` | NUMERIC | [0,1] |
| `population` | INTEGER | |
| `source_year` | INTEGER | census / survey year |

### `raw_weather_staging`
| Column | Type | Notes |
|---|---|---|
| `id` | UUID (PK) | |
| `fetched_at` | TIMESTAMPTZ | |
| `source` | TEXT | open-meteo / mock |
| `payload` | JSONB | raw Open-Meteo response |
| `valid_until` | TIMESTAMPTZ | cache window |

### `ward_weather`
| Column | Type | Notes |
|---|---|---|
| `ward_id` | TEXT (FK) | |
| `forecast_ts` | TIMESTAMPTZ | |
| `temperature_c` | NUMERIC | |
| `humidity_pct` | NUMERIC | |
| `wind_ms` | NUMERIC | |
| `shortwave_rad_wm2` | NUMERIC | |
| `pressure_pa` | NUMERIC | |
| `cloud_cover_pct` | NUMERIC | |
| PRIMARY KEY | `(ward_id, forecast_ts)` | |

### `ward_thermal_stress`
| Column | Type | Notes |
|---|---|---|
| `ward_id` | TEXT (FK) | |
| `forecast_ts` | TIMESTAMPTZ | |
| `wbgt_c` | NUMERIC | |
| `utci_c` | NUMERIC | |
| `wbgt_quality` | TEXT | ok / degraded |
| `physics_version` | TEXT | |
| PRIMARY KEY | `(ward_id, forecast_ts)` | |

### `ward_mortality_risk`
| Column | Type | Notes |
|---|---|---|
| `ward_id` | TEXT (FK) | |
| `computed_at` | TIMESTAMPTZ | |
| `mri` | NUMERIC | [0,100] |
| `band` | TEXT | |
| `top_factors` | JSONB | |
| `risk_model_version` | TEXT | |
| PRIMARY KEY | `(ward_id, computed_at)` | |

### `alerts`
| Column | Type | Notes |
|---|---|---|
| `id` | UUID (PK) | |
| `created_at` | TIMESTAMPTZ | |
| `ward_id` | TEXT | |
| `severity` | TEXT | |
| `headline` | TEXT | |
| `body` | TEXT | |
| `recommended_actions` | JSONB | |
| `agent_run_id` | UUID | |
| `status` | TEXT | ACTIVE / ACK / EXPIRED |

### `pipeline_runs`
| Column | Type | Notes |
|---|---|---|
| `id` | UUID (PK) | |
| `started_at` | TIMESTAMPTZ | |
| `finished_at` | TIMESTAMPTZ | |
| `status` | TEXT | |
| `stages` | JSONB | per-stage status |

### `agent_decisions`
| Column | Type | Notes |
|---|---|---|
| `id` | UUID (PK) | |
| `run_id` | UUID | |
| `ward_id` | TEXT | |
| `plan` | JSONB | strict schema |
| `reasoning` | TEXT | |
| `fallback_used` | BOOLEAN | |

## 2. Stable IDs

`ward_id` is the authoritative join key. It is computed as:

```
uuid5(NAMESPACE_URL, f"{canonical_ward_name}|{ward_number}|{city_id}")
```

## 3. Units Convention

- Temperature: °C
- Humidity: %
- Wind: m/s
- Pressure: Pa
- Radiation: W/m²
- Time: ISO-8601 with timezone (`Asia/Kolkata`)