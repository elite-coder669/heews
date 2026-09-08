export type Band = 'LOW' | 'MODERATE' | 'HIGH' | 'VERY_HIGH' | 'EXTREME';
export type Priority = 'LOW' | 'MEDIUM' | 'HIGH';
export type Recheck = 15 | 30 | 60 | 120 | 360;

export interface Centroid { lat: number; lon: number; }

export interface Demographics {
  elderly_ratio: number;
  outdoor_worker_share: number;
  informal_housing_share: number;
  population: number;
  source_year: number;
}

export interface Ward {
  ward_id: string;
  ward_number: number;
  ward_name: string;
  zone: string;
  centroid: Centroid;
  demographics?: Demographics;
}

export interface WeatherSnapshot {
  temperature_c: number;
  humidity_pct: number;
  wind_ms: number;
  shortwave_rad_wm2: number;
  pressure_pa?: number;
  cloud_cover_pct?: number;
}

export interface WardWeather {
  ward_id: string;
  source: string;
  fetched_at: string;
  current: WeatherSnapshot;
}

export interface ThermalSnapshot {
  wbgt_c: number;
  utci_c: number;
  wbgt_quality: 'ok' | 'degraded';
}

export interface WardThermal {
  ward_id: string;
  physics_version: string;
  current: ThermalSnapshot;
}

export interface WardRisk {
  ward_id: string;
  computed_at: string;
  risk: {
    score: number;
    band: Band;
    top_factors: string[];
  };
  thermal?: ThermalSnapshot;
  weather?: WeatherSnapshot;
  model_version: string;
}

export interface ForecastCell {
  day: string;
  wbgt_c: number;
  utci_c: number;
  mri: number;
  band: Band;
}

export interface ForecastSeries {
  ward_id: string;
  series: ForecastCell[];
}

export interface CityForecast {
  city: string;
  horizon_hours: number;
  by_ward: ForecastSeries[];
}

export interface Alert {
  id: string;
  ward_id: string;
  created_at: string;
  severity: Band;
  headline: string;
  body: string;
  recommended_actions: Array<{ action: string; priority: Priority; target: string }>;
  status: 'ACTIVE' | 'ACK' | 'EXPIRED';
}

export interface ActionPlan {
  severity: Band;
  priority_wards: string[];
  reasoning_summary: string;
  key_factors: string[];
  recommended_actions: Array<{ action: string; priority: Priority; target: string }>;
  public_advisory: string;
  recheck_interval_minutes: Recheck;
  agent_run_id: string;
  fallback_used: boolean;
}

export interface VersionInfo {
  service: string;
  physics_version: string;
  risk_model_version: string;
  agent_version: string;
}

export type ApiMode = 'LIVE' | 'MOCK' | 'DEGRADED';

export interface HealthInfo {
  status: string;
  mode: ApiMode;
}