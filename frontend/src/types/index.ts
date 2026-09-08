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
  risk_model_version: string;
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

export interface AlertAction {
  action: string;
  priority: Priority;
  target: string;
  reason?: string;
  evidence?: string[];
}

export interface Alert {
  id: string;
  ward_id: string;
  created_at: string;
  severity: Band;
  headline: string;
  body: string;
  recommended_actions: AlertAction[];
  status: 'ACTIVE' | 'ACK' | 'EXPIRED';
  source?: 'pipeline' | 'municipal';
  agent_run_id?: string;
  agent_version?: string;
  fallback_used?: boolean;
}

export interface PriorityWard {
  ward_id: string;
  ward_name: string;
  score: number;
  band: Band;
}

export interface HistoricalContext {
  found: boolean;
  type: 'local' | 'demo' | 'none';
  event_id?: string;
  similarity_reason?: string;
  source?: string;
  synthetic: boolean;
}

export interface MatchingEvent {
  event_id: string;
  date: string;
  ward_name?: string;
  score: number;
  band: Band;
  similarity: number;
  dimensions?: Record<string, number>;
  actions_taken?: string[];
  outcome?: string;
}

export interface PlanEvidence {
  current_risk: boolean;
  vulnerability: boolean;
  forecast: boolean;
  historical_precedent: boolean;
  external_reference: boolean;
  general_guidance: boolean;
}

export interface ActionPlan {
  scope?: 'current' | 'forecast';
  scope_ward_id?: string;
  scope_horizon_hours?: number;
  scope_label?: string;
  severity: Band;
  priority_wards: PriorityWard[];
  reasoning_summary: string;
  key_factors: string[];
  recommended_actions: AlertAction[];
  public_advisory: string;
  recheck_interval_minutes: Recheck;
  agent_run_id: string;
  fallback_used: boolean;
  agent_version: string;
  why_this_matters: string;
  historical_context: HistoricalContext;
  matching_events?: MatchingEvent[];
  evidence: PlanEvidence;
  confidence: 'HIGH' | 'MEDIUM' | 'LOW';
  limitations: string[];
}

export interface PriorityResponse {
  priority_wards: PriorityWard[];
}

export interface ManualAlertInput {
  ward_id: string;
  severity: Band;
  headline: string;
  body: string;
  recommended_actions: AlertAction[];
  agent_run_id?: string;
  agent_version?: string;
  fallback_used?: boolean;
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
  last_run?: string;
  pipeline_run_id?: string;
}