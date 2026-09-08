import type { HealthInfo, VersionInfo, ApiMode } from '../types';

export function StateBanner({ mode, lastUpdate, health }: {
  mode: ApiMode;
  lastUpdate?: string;
  health?: HealthInfo;
}) {
  if (mode === 'LIVE') {
    return (
      <div className="state-banner live">
        ● LIVE · data updated {lastUpdate ?? 'recently'} · pipeline healthy
      </div>
    );
  }
  if (mode === 'DEGRADED') {
    return (
      <div className="state-banner degraded">
        ⚠ DATA MAY BE STALE · last successful update: {lastUpdate ?? 'unknown'}
      </div>
    );
  }
  if (mode === 'MOCK') {
    return (
      <div className="state-banner mock">
        DEMO MODE · Synthetic data · set APP_MODE=LIVE for real forecast
      </div>
    );
  }
  return <div className="state-banner stale">Pipeline status unknown</div>;
}

export function Disclaimer({ health }: { health?: HealthInfo }) {
  return (
    <div className="disclaimer">
      <strong>Decision-support prototype.</strong> Not medical advice. ·
      {' '}<span className="mono">{health?.status ?? '—'}</span>
    </div>
  );
}

export function StatusHeader({ version, mode, lastUpdate }: {
  version?: VersionInfo;
  mode: ApiMode;
  lastUpdate?: string;
}) {
  return (
    <header className="status-header">
      <div className="logo">HEEWS · Municipal Heat Intelligence</div>
      <div className="meta">
        <span className={`mode-pill ${mode === 'LIVE' ? 'live' : mode === 'MOCK' ? 'mock' : 'degraded'}`}>
          {mode}
        </span>
        <span><strong>Updated</strong> {lastUpdate ?? '—'}</span>
        {version && <span><strong>Physics</strong> {version.physics_version}</span>}
        {version && <span><strong>Risk model</strong> {version.risk_model_version}</span>}
        <button
          className="refresh-btn"
          onClick={() => window.location.reload()}
          title="Refresh page to re-fetch data"
        >
          ↻
        </button>
      </div>
    </header>
  );
}