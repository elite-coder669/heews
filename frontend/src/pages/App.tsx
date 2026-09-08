import { useEffect, useState, useCallback, useRef } from 'react';
import { api } from '../services/api';
import type {
  Ward, WardRisk, WardThermal, WardWeather, Alert, ActionPlan,
  VersionInfo, HealthInfo, ApiMode, Band, CityForecast, PriorityWard,
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

function riskAtDay(
  forecast: CityForecast | null,
  wardId: string | undefined,
  day: number,
): { band: Band; score: number; wbgt?: number; utci?: number } | undefined {
  if (!forecast) return undefined;
  const series = forecast.by_ward.find((s) => s.ward_id === wardId);
  if (!series || series.series.length === 0) return undefined;
  const cell = series.series[Math.min(day, series.series.length - 1)];
  return { band: cell.band, score: cell.mri, wbgt: cell.wbgt_c, utci: cell.utci_c };
}

export default function App() {
  const [view, setView] = useState<'console' | 'alerts' | 'public'>('console');
  const [mode, setMode] = useState<ApiMode>('LIVE');
  const [health, setHealth] = useState<HealthInfo | null>(null);
  const [version, setVersion] = useState<VersionInfo | null>(null);
  const [wards, setWards] = useState<Ward[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [focusId, setFocusId] = useState<string | null>(null);
  const [risk, setRisk] = useState<WardRisk | null>(null);
  const [thermal, setThermal] = useState<WardThermal | null>(null);
  const [weather, setWeather] = useState<WardWeather | null>(null);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [priorityWards, setPriorityWards] = useState<PriorityWard[]>([]);
  const [forecast, setForecast] = useState<CityForecast | null>(null);
  const [horizonDay, setHorizonDay] = useState(0);
  const [plan, setPlan] = useState<ActionPlan | null>(null);
  const [planLoading, setPlanLoading] = useState(false);
  const [issuing, setIssuing] = useState(false);
  const [approvedAlert, setApprovedAlert] = useState<Alert | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<string>(lastUpdatedString());
  const [polygons, setPolygons] = useState<Record<string, [number, number][]>>({});
  const [bandFilter, setBandFilter] = useState<string | null>(null);
  const [initialLoaded, setInitialLoaded] = useState(false);
  const bootState = useRef({ running: false, loaded: false });
  const polledRun = useRef<string | null>(null);

  const loadCore = useCallback(async () => {
    if (bootState.current.running) return false;
    bootState.current.running = true;
    try {
      const [h, v, ws, fc] = await Promise.all([
        api.health().catch(() => ({ status: 'unknown', mode: 'DEGRADED' as ApiMode })),
        api.version().catch(() => null),
        api.wards().catch(() => [] as Ward[]),
        api.forecast().catch(() => null as CityForecast | null),
      ]);
      if (ws.length > 0) {
        const polys = await fetchWardPolygons().catch(() => ({} as Record<string, [number, number][]>));
        setHealth(h as HealthInfo);
        setVersion(v as VersionInfo | null);
        setMode((h as HealthInfo).mode);
        setWards(ws);
        setForecast(fc);
        setPolygons({ ...fallbackBoxes(ws), ...polys });
        setUpdatedAt(lastUpdatedString());
        bootState.current.loaded = true;
        return true;
      }
      return false;
    } catch (e: any) {
      setError(String(e));
      return false;
    } finally {
      bootState.current.running = false;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      for (let attempt = 1; attempt <= 3 && !cancelled; attempt++) {
        const ok = await loadCore();
        if (ok || cancelled) break;
        await new Promise((r) => setTimeout(r, 1000 * attempt));
      }
      if (!cancelled) setInitialLoaded(true);
    })();
    return () => { cancelled = true; };
  }, [loadCore]);

  // Background sync: health + forecast + alerts + priority
  useEffect(() => {
    let cancelled = false;
    const sync = async () => {
      try {
        const h = await api.health();
        if (cancelled) return;
        setHealth(h);
        setMode(h.mode);
        const fc = await api.forecast().catch(() => null as CityForecast | null);
        const a = await api.alerts().catch(() => [] as Alert[]);
        const p = await api.priority().catch(() => [] as PriorityWard[]);
        if (cancelled) return;
        if (fc) setForecast(fc);
        setAlerts(a);
        setPriorityWards(p);
        setUpdatedAt(lastUpdatedString());
      } catch { /* keep stale */ }
    };
    const tick = async () => {
      try {
        const h = await api.health();
        if (cancelled) return;
        setHealth(h);
        setMode(h.mode);
        if (!bootState.current.loaded && h.status === 'ok') {
          await loadCore();
        } else if (h.last_run && h.last_run !== polledRun.current) {
          polledRun.current = h.last_run;
          const [fc, a] = await Promise.all([
            api.forecast().catch(() => null as CityForecast | null),
            api.alerts().catch(() => [] as Alert[]),
          ]);
          if (!cancelled) {
            if (fc) setForecast(fc);
            setAlerts(a);
            setUpdatedAt(lastUpdatedString());
          }
        }
      } catch { /* keep stale */ }
    };
    sync();
    tick();
    const id = setInterval(tick, 15_000);
    return () => { cancelled = true; clearInterval(id); };
  }, [loadCore]);

  // Proactive detail + agent plan on selection & horizon change (cached while loading)
  useEffect(() => {
    if (!selectedId) {
      setRisk(null); setThermal(null); setWeather(null); setPlan(null);
      return;
    }
    let cancelled = false;
    (async () => {
      setDetailLoading(true);
      setPlanLoading(true);
      try {
        const [r, t, w, p] = await Promise.all([
          api.risk(selectedId),
          api.thermal(selectedId).catch(() => null),
          api.weather(selectedId).catch(() => null),
          api.actionPlan([selectedId], horizonDay * 24).catch(() => null),
        ]);
        if (cancelled) return;
        setRisk(r);
        setThermal(t);
        setWeather(w);
        if (p) setPlan(p);
      } catch (e: any) {
        if (!cancelled) setError(String(e));
      } finally {
        if (!cancelled) { setDetailLoading(false); setPlanLoading(false); }
      }
    })();
    return () => { cancelled = true; };
  }, [selectedId, horizonDay]);

  const handleSelectWard = useCallback((id: string) => {
    setSelectedId(id);
    setView('console');
  }, []);

  const handleFocusWard = useCallback((id: string) => {
    setFocusId(id);
    setSelectedId(id);
    setView('console');
  }, []);

  const handleIssueAlert = useCallback(async (p: ActionPlan) => {
    if (!p.scope_ward_id && !selectedId) return;
    const wardId = p.scope_ward_id ?? selectedId!;
    setIssuing(true);
    try {
      const alert = await api.issueAlert({
        ward_id: wardId,
        severity: p.severity,
        headline: `Heat alert for ${p.scope_label ?? 'current conditions'} — ${p.severity.replace('_', ' ')} risk`,
        body: p.why_this_matters || p.reasoning_summary,
        recommended_actions: p.recommended_actions,
        agent_run_id: p.agent_run_id,
        agent_version: p.agent_version,
        fallback_used: p.fallback_used,
      });
      setApprovedAlert(alert);
      const a = await api.alerts().catch(() => [] as Alert[]);
      setAlerts(a);
    } catch (e: any) {
      setError(String(e));
    } finally {
      setIssuing(false);
    }
  }, [selectedId]);

  const displayedRisks = Object.fromEntries(
    wards.map((w) => {
      const cell = riskAtDay(forecast, w.ward_id, horizonDay);
      return [w.ward_id, {
        band: cell?.band ?? 'MODERATE',
        score: cell?.score ?? 40,
        wbgt: cell?.wbgt,
        utci: cell?.utci,
      }];
    }),
  ) as Record<string, { band: Band; score: number; wbgt?: number; utci?: number }>;

  const counts = BAND_ORDER.reduce((acc, b) => {
    acc[b] = wards.filter((w) => displayedRisks[w.ward_id]?.band === b).length;
    return acc;
  }, {} as Record<string, number>);

  const filteredWards = bandFilter
    ? wards.filter((w) => displayedRisks[w.ward_id]?.band === bandFilter)
    : wards;

  const selectedSeries = forecast?.by_ward.find((s) => s.ward_id === selectedId);
  const forecastSeries = (selectedSeries ?? forecast?.by_ward[0])?.series ?? [];
  const forecastRows = forecastSeries.map((c) => ({
    label: c.day,
    wbgt: c.wbgt_c,
    utci: c.utci_c,
    mri: c.mri,
    band: c.band,
  }));
  const horizonLabel = forecastSeries[Math.min(horizonDay, forecastSeries.length - 1)]?.day ?? 'Now';
  const selectedWard = wards.find((w) => w.ward_id === selectedId);
  const approvedWard = approvedAlert ? wards.find((w) => w.ward_id === approvedAlert.ward_id) : undefined;

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
        {initialLoaded ? `${wards.length} wards · ${Object.keys(displayedRisks).length} risked` : 'Loading wards…'}
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
          <AlertCenter alerts={alerts} onSelectWard={handleFocusWard} />
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
            areaName={approvedWard?.ward_name ?? selectedWard?.ward_name ?? 'Hyderabad'}
            rows={forecastRows.map((r) => ({ label: r.label, band: r.band }))}
            advisory={approvedAlert}
            advisoryText={approvedAlert?.body ?? plan?.public_advisory ?? 'Stay hydrated and avoid midday sun.'}
          />
        </ErrorBoundary>
      </div>
    );
  }

  return (
    <div className="app-shell">
      {header}

      <section className="situation">
        <h2>City situation · selected horizon: {horizonLabel}</h2>
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
                risks={displayedRisks}
                selectedId={selectedId}
                onSelect={handleSelectWard}
                polygons={polygons}
                focusId={focusId}
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
            <input
              type="range"
              min={0}
              max={120}
              step={24}
              value={horizonDay * 24}
              onChange={(e) => setHorizonDay(Math.round(Number(e.target.value) / 24))}
              aria-label="Forecast horizon"
            />
            <span className="now-label muted">{horizonLabel}</span>
          </div>
        </div>

        <aside className="detail-pane">
          {priorityWards.length > 0 && (
            <div className="card" style={{ marginBottom: 12 }}>
              <div className="section-title">Top Priority Wards</div>
              <ol className="priority-list">
                {priorityWards.slice(0, 5).map((pw, i) => (
                  <li key={pw.ward_id}>
                    <button
                      className={`priority-row ${selectedId === pw.ward_id ? 'active' : ''}`}
                      onClick={() => handleFocusWard(pw.ward_id)}
                    >
                      <span className="rank">{i + 1}</span>
                      <span className="name">{pw.ward_name}</span>
                      <span className="score" style={{ color: pw.band === 'EXTREME' || pw.band === 'VERY_HIGH' || pw.band === 'HIGH' ? 'var(--danger)' : 'var(--text)' }}>
                        MRI {pw.score.toFixed(0)} · {pw.band.replace('_', ' ')}
                      </span>
                    </button>
                  </li>
                ))}
              </ol>
            </div>
          )}
          <ErrorBoundary>
            <DetailDrawer
              ward={selectedWard}
              risk={risk}
              thermal={thermal}
              weather={weather}
              forecastSeries={forecastSeries}
              horizonDay={horizonDay}
              loading={detailLoading}
            />
          </ErrorBoundary>
          <ErrorBoundary>
            <DecisionPanel
              plan={plan}
              loading={planLoading}
              wardName={selectedWard?.ward_name}
              onIssueAlert={handleIssueAlert}
              issuing={issuing}
            />
          </ErrorBoundary>
        </aside>
      </main>

      <ErrorBoundary>
        <ForecastStrip rows={forecastRows} />
      </ErrorBoundary>
      <ErrorBoundary>
        <AlertTicker alerts={alerts} />
      </ErrorBoundary>
      <footer className="provenance">
        Ward boundaries in this console are synthetic demo fixtures, not official GHMC boundaries.
        All risk/thermal values are pipeline outputs.
      </footer>
    </div>
  );
}
