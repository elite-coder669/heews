import type {
  Ward, WardRisk, WardThermal, WardWeather, CityForecast,
  Alert, ActionPlan, VersionInfo, HealthInfo, PriorityResponse, ManualAlertInput,
} from '../types';

const BASE = (import.meta as any).env?.VITE_API_BASE ?? '/api';

export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

async function jget<T>(path: string): Promise<T> {
  const r = await fetch(`${BASE}${path}`);
  if (!r.ok) throw new ApiError(r.status, 'HTTP_' + r.status, r.statusText);
  const body = await r.json();
  if (!body.ok) throw new ApiError(r.status, body.error?.code || 'API', body.error?.message || 'failed');
  return body.data as T;
}

async function jpost<T>(path: string, payload: unknown): Promise<T> {
  const r = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!r.ok) throw new ApiError(r.status, 'HTTP_' + r.status, r.statusText);
  const body = await r.json();
  if (!body.ok) throw new ApiError(r.status, body.error?.code || 'API', body.error?.message || 'failed');
  return body.data as T;
}

export interface AppConfig {
  city: string;       // slug, e.g. "hyderabad" — matches /data/<city>_wards.geojson
  city_label: string;  // e.g. "Hyderabad"
  city_state: string;  // e.g. "Telangana"
  city_lat: number;
  city_lon: number;
}

export const api = {
  health: () => jget<HealthInfo>('/health'),
  version: () => jget<VersionInfo>('/version'),
  config: () => jget<AppConfig>('/config'),
  wards: () => jget<{ wards: Ward[] }>('/wards').then((d) => d.wards),
  ward: (id: string) => jget<Ward>(`/wards/${id}`),
  weather: (id: string) => jget<WardWeather>(`/wards/${id}/weather`),
  thermal: (id: string) => jget<WardThermal>(`/wards/${id}/thermal`),
  risk: (id: string) => jget<WardRisk>(`/wards/${id}/risk`),
  forecast: () => jget<CityForecast>('/forecast'),
  alerts: () => jget<{ alerts: Alert[] }>('/alerts').then((d) => d.alerts),
  actionPlan: (ward_ids: string[], horizonHours: number = 0) =>
    jpost<ActionPlan>('/agent/action-plan', {
      ward_ids,
      context: 'municipal planning',
      horizon_hours: horizonHours,
    }),
  priority: () => jget<PriorityResponse>('/agent/priority').then((d) => d.priority_wards),
  issueAlert: (input: ManualAlertInput) => jpost<Alert>('/alerts', input),
};