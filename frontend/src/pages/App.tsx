import { useEffect, useState, useCallback } from 'react';
import { api, ApiError } from '../services/api';
import type {
  Ward, WardRisk, WardThermal, WardWeather, Alert, ActionPlan,
  VersionInfo, HealthInfo, ApiMode, Band,
} from '../types';
import { StatusHeader, Disclaimer, StateBanner } from '../components/StatusBar';
import RiskMap from '../components/RiskMap';
import DetailDrawer from '../components/DetailDrawer';
import DecisionPanel from '../components/DecisionPanel';
import ForecastStrip from '../components/ForecastStrip';
import { AlertTicker, AlertCenter } from '../components/AlertViews';
import PublicAdvisory from '../components/PublicAdvisory';
import ErrorBoundary from '../components/ErrorBoundary';
import { fetchWardPolygons, fallbackBoxes } from '../services/polygons';

const BAND_ORDER = ['EXTREME', 'VERY_HIGH', 'HIGH', 'MODERATE', 'LOW'] as const;

function lastUpdatedString(): string {
  const d = new Date();
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function band(b: string): Band {
  return (BAND_ORDER as readonly string[]).includes(b) ? (b as Band) : 'HIGH';
}

function buildSyntheticForecast(current?: WardRisk | null) {
  const base = current?.risk.score ?? 50;
  return [
    { label: 'Today',    wbgt: 32 + base*0.02, utci: 42 + base*0.04, mri: Math.min(95, base),         band: band(current?.risk.band ?? 'HIGH') },
    { label: 'Tomorrow', wbgt: 34 + base*0.02, utci: 44 + base*0.04, mri: Math.min(98, base + 6),     band: band('VERY_HIGH') },
    { label: '+2d',      wbgt: 35 + base*0.02, utci: 46 + base*0.04, mri: Math.min(99, base + 12),    band: band('EXTREME') },
    { label: '+3d',      wbgt: 34 + base*0.02, utci: 45 + base*0.04, mri: Math.min(97, base + 8),     band: band('EXTREME') },
    { label: '+4d',      wbgt: 33 + base*0.02, utci: 42 + base*0.04, mri: Math.max(20, base - 8),     band: band('HIGH') },
  ];
}

export default function App() {
  const [view, setView] = useState<'console' | 'alerts' | 'public'>('console');
  const [mode, setMode] = useState<ApiMode>('LIVE');
  const [health, setHealth] = useState<HealthInfo | null>(null);
  const [version, setVersion] = useState<VersionInfo | null>(null);
  const [wards, setWards] = useState<Ward[]>([]);
  const [risksByWard, setRisksByWard] = useState<Record<string, WardRisk>>({});
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [risk, setRisk] = useState<WardRisk | null>(null);
  const [thermal, setThermal] = useState<WardThermal | null>(null);
  const [weather, setWeather] = useState<WardWeather | null>(null);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [plan, setPlan] = useState<ActionPlan | null>(null);
  const [planLoading, setPlanLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<string>(lastUpdatedString());
  const [polygons, setPolygons] = useState<Record<string, [number, number][]>>({});
  const [bandFilter, setBandFilter] = useState<string | null>(null);
  const [initialLoaded, setInitialLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [h, v, ws] = await Promise.all([
          api.health().catch(() => ({ status: 'unknown', mode: 'DEGRADED' as ApiMode })),
          api.version().catch(() => null),
          api.wards().catch(() => [] as Ward[]),
        ]);
        if (cancelled) return;
        setHealth(h as HealthInfo);
        setVersion(v as VersionInfo | null);
        setMode((h as HealthInfo).mode);
        setWards(ws);
        const polys = await fetchWardPolygons().catch(() => ({} as Record<string, [number, number][]>));
        if (cancelled) return;
        setPolygons({ ...fallbackBoxes(ws), ...polys });

        const riskEntries = await Promise.all(
          ws.slice(0, 50).map((w) => api.risk(w.ward_id).catch(() => null)),
        );
        if (cancelled) return;
        const map: Record<string, WardRisk> = {};
        ws.slice(0, 50).forEach((w, i) => {
          const r = riskEntries[i] as WardRisk | null;
          if (r) map[w.ward_id] = r;
        });
        setRisksByWard(map);
        setUpdatedAt(lastUpdatedString());
      } catch (e: any) {
        if (cancelled) return;
        setError(e instanceof ApiError ? `${e.code}: ${e.message}` : String(e));
        setMode('DEGRADED');
      } finally {
        if (!cancelled) setInitialLoaded(true);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    const tick = async () => {
      try {
        const a = await api.alerts().catch(() => [] as Alert[]);
        setAlerts(a);
        setUpdatedAt(lastUpdatedString());
      } catch { /* keep stale */ }
    };
    tick();
    const id = setInterval(tick, 60_000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    if (!selectedId) { setRisk(null); setThermal(null); setWeather(null); setPlan(null); return; }
    let cancelled = false;
    (async () => {
      setDetailLoading(true);
      try {
        const [r, t, w] = await Promise.all([
          api.risk(selectedId),
          api.thermal(selectedId).catch(() => null),
          api.weather(selectedId).catch(() => null),
        ]);
        if (cancelled) return;
        setRisk(r);
        setThermal(t);
        setWeather(w);
      } catch (e: any) {
        if (!cancelled) setError(String(e));
      } finally {
        if (!cancelled) setDetailLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [selectedId]);

  const onAskAgent = useCallback(async () => {
    if (!selectedId) return;
    setPlanLoading(true);
    try {
      const p = await api.actionPlan([selectedId]);
      setPlan(p);
    } catch (e: any) {
      setError(String(e));
    } finally {
      setPlanLoading(false);
    }
  }, [selectedId]);

  const counts = BAND_ORDER.reduce((acc, b) => {
    acc[b] = Object.values(risksByWard).filter((r) => r.risk.band === b).length;
    return acc;
  }, {} as Record<string, number>);

  const filteredWards = bandFilter
    ? wards.filter((w) => risksByWard[w.ward_id]?.risk.band === bandFilter)
    : wards;

  const forecastRows = buildSyntheticForecast(risk);

  const topNav = (
    <nav style={{ padding: '8px 24px', borderBottom: '1px solid var(--border)', display: 'flex', gap: 12, alignItems: 'center' }}>
      <button onClick={() => setView('alerts')} className={view === 'alerts' ? 'primary' : ''}>
        Alert Center
      </button>
      <button onClick={() => setView('public')} className={view === 'public' ? 'primary' : ''}>
        Public Advisory
      </button>
      {view !== 'console' && (
        <button onClick={() => setView('console')} className="primary">
          ← Console
        </button>
      )}
      <span style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--muted)' }}>
        {initialLoaded ? `${wards.length} wards · ${Object.keys(risksByWard).length} risked` : 'Loading wards…'}
      </span>
    </nav>
  );

  const header = (
    <>
      <StatusHeader version={version ?? undefined} mode={mode} lastUpdate={updatedAt} />
      <StateBanner mode={mode} lastUpdate={updatedAt} health={health ?? undefined} />
      <Disclaimer health={health ?? undefined} />
      {error && <div className="state-banner failure">Backend error: {error}</div>}
      {topNav}
    </>
  );

  if (view === 'alerts') {
    return (
      <div className="app-shell">
        {header}
        <ErrorBoundary>
          <AlertCenter alerts={alerts} />
        </ErrorBoundary>
      </div>
    );
  }

  if (view === 'public') {
    return (
      <div className="app-shell">
        {header}
        <ErrorBoundary>
          <PublicAdvisory
            areaName={wards.find((w) => w.ward_id === selectedId)?.ward_name ?? 'Hyderabad'}
            rows={forecastRows.map((r) => ({ label: r.label, band: r.band }))}
            advisoryText={plan?.public_advisory ?? 'Stay hydrated and avoid midday sun.'}
          />
        </ErrorBoundary>
      </div>
    );
  }

  return (
    <div className="app-shell">
      {header}

      <section className="situation">
        <h2>City situation · next 72 hours</h2>
        <div className="counters">
          {(['EXTREME', 'VERY_HIGH', 'HIGH', 'MODERATE'] as const).map((b) => (
            <button
              key={b}
              className={`counter ${bandFilter === b ? 'active' : ''}`}
              onClick={() => setBandFilter(bandFilter === b ? null : b)}
              aria-label={`Filter to ${b}`}
            >
              <span className="count">{counts[b] ?? 0}</span>
              <span className="name">{b.replace('_', ' ')}</span>
            </button>
          ))}
        </div>
      </section>

      <main className="main-grid">
        <div className="map-pane">
          <ErrorBoundary>
            {wards.length > 0 ? (
              <RiskMap
                wards={filteredWards}
                risks={Object.fromEntries(
                  Object.entries(risksByWard).map(([k, v]) => [k, { band: v.risk.band, score: v.risk.score }]),
                )}
                selectedId={selectedId}
                onSelect={setSelectedId}
                polygons={polygons}
              />
            ) : (
              <div className="drawer-empty" style={{ height: '100%' }}>
                <h3>Loading Hyderabad wards…</h3>
                <p>Backend is reachable. Awaiting ward list.</p>
              </div>
            )}
          </ErrorBoundary>
          <div className="time-slider">
            <span className="now-label">NOW</span>
            <input type="range" min={0} max={120} defaultValue={0} aria-label="Forecast horizon" />
            <span className="now-label muted">+5d</span>
          </div>
        </div>

        <aside className="detail-pane">
          <ErrorBoundary>
            <DetailDrawer
              ward={wards.find((w) => w.ward_id === selectedId)}
              risk={risk}
              thermal={thermal}
              weather={weather}
              loading={detailLoading}
            />
          </ErrorBoundary>
          {selectedId && (
            <button
              onClick={onAskAgent}
              className="primary"
              style={{ marginTop: 12, width: '100%' }}
              disabled={planLoading}
            >
              {planLoading ? 'Reasoning…' : 'Ask Decision Support'}
            </button>
          )}
          <ErrorBoundary>
            <DecisionPanel plan={plan} loading={planLoading} />
          </ErrorBoundary>
        </aside>
      </main>

      <ErrorBoundary>
        <ForecastStrip rows={forecastRows} />
      </ErrorBoundary>
      <ErrorBoundary>
        <AlertTicker alerts={alerts} />
      </ErrorBoundary>
    </div>
  );
}