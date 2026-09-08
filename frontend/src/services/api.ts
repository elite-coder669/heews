import type {
  Ward, WardRisk, WardThermal, WardWeather, CityForecast,
  Alert, ActionPlan, VersionInfo, HealthInfo,
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

export const api = {
  health: () => jget<HealthInfo>('/health'),
  version: () => jget<VersionInfo>('/version'),
  wards: () => jget<{ wards: Ward[] }>('/wards').then((d) => d.wards),
  ward: (id: string) => jget<Ward>(`/wards/${id}`),
  weather: (id: string) => jget<WardWeather>(`/wards/${id}/weather`),
  thermal: (id: string) => jget<WardThermal>(`/wards/${id}/thermal`),
  risk: (id: string) => jget<WardRisk>(`/wards/${id}/risk`),
  forecast: () => jget<CityForecast>('/forecast'),
  alerts: () => jget<{ alerts: Alert[] }>('/alerts').then((d) => d.alerts),
  actionPlan: (ward_ids: string[]) =>
    jpost<ActionPlan>('/agent/action-plan', { ward_ids, context: 'municipal planning' }),
};