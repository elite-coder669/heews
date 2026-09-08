import type { Ward, WardRisk, WardThermal, WardWeather, ForecastCell } from '../types';
import { BandChip, PriorityChip } from './BandChip';

interface Props {
  ward?: Ward;
  risk?: WardRisk | null;
  thermal?: WardThermal | null;
  weather?: WardWeather | null;
  forecastSeries?: ForecastCell[];
  horizonDay: number;
  loading: boolean;
}

function forecastChip(cell: ForecastCell, idx: number, active: boolean) {
  return (
    <div key={idx} className="fc-chip" data-active={active}>
      <span className="day">{cell.day}</span>
      <BandChip band={cell.band} />
      <span className="mri">MRI {cell.mri.toFixed(0)}</span>
    </div>
  );
}

export default function DetailDrawer({
  ward, risk, thermal, weather, forecastSeries, horizonDay, loading,
}: Props) {
  if (loading) {
    return (
      <div className="drawer-empty">
        <h3>Loading ward intelligence…</h3>
        <p>Fetching risk, thermal stress, and forecast context.</p>
      </div>
    );
  }

  if (!ward || !risk) {
    return (
      <div className="drawer-empty">
        <h3>Select a ward</h3>
        <p>Click a ward on the map to open its intelligence panel: current risk, thermal stress, forecast trend, and why it matters.</p>
      </div>
    );
  }

  const band = risk.risk.band.toLowerCase().replace('_', ' ');
  const wbgt = thermal?.current?.wbgt_c;
  const utci = thermal?.current?.utci_c;
  const ahead = (forecastSeries ?? []).slice(0, 3);

  return (
    <div>
      <div className="ward-header">
        <h2>{ward.ward_name}</h2>
        <span className="zone">Ward #{ward.ward_number} · {ward.zone}</span>
      </div>

      {/* Metrics grid */}
      <div className="card" style={{ marginBottom: 12 }}>
        <div className="section-title">Ward Intelligence</div>
        <div className="metrics-grid">
          <div className="metric">
            <span className="k">MRI</span>
            <span className="v">{risk.risk.score.toFixed(0)}<small>/100</small></span>
          </div>
          <div className="metric">
            <span className="k">Band</span>
            <span className="v"><BandChip band={risk.risk.band} /></span>
          </div>
          <div className="metric">
            <span className="k">WBGT</span>
            <span className="v">{wbgt !== undefined ? `${wbgt.toFixed(1)} °C` : '—'}</span>
          </div>
          <div className="metric">
            <span className="k">UTCI</span>
            <span className="v">{utci !== undefined ? `${utci.toFixed(1)} °C` : '—'}</span>
          </div>
        </div>
        <div className="row"><span className="label">Data quality</span><span className="value">
          {thermal?.current?.wbgt_quality === 'degraded' ? 'WBGT degraded — radiation missing' : 'Good'}
        </span></div>
        <div className="row"><span className="label">Computed</span><span className="value mono">{risk.computed_at}</span></div>
      </div>

      {/* Forecast NOW→48H chips */}
      {ahead.length > 0 && (
        <div className="card" style={{ marginBottom: 12 }}>
          <div className="section-title">Forecast trend</div>
          <div className="fc-row">
            {ahead.map((c, i) => forecastChip(c, i, i === horizonDay))}
          </div>
        </div>
      )}

      {/* Why this risk — deterministic facts only */}
      <div className="card">
        <div className="section-title">Why is this ward {band}?</div>
        {risk.risk.top_factors.length > 0 ? (
          <ul className="why-factors">
            {risk.risk.top_factors.map((f, i) => <li key={i}>{f}</li>)}
          </ul>
        ) : (
          <p className="muted">No risk factors available.</p>
        )}
        <p style={{ marginTop: 12, fontSize: 12, color: 'var(--muted)' }}>
          Factors are deterministic pipeline outputs. Risk model version{' '}
          <span className="mono">{risk.risk_model_version}</span>. Ward boundaries in this
          demo are synthetic fixtures, not official GHMC boundaries.
        </p>
      </div>

      {/* Vulnerability */}
      {ward.demographics && (
        <div className="card" style={{ marginTop: 12 }}>
          <div className="section-title">Vulnerability composition</div>
          <div className="row"><span className="label">Elderly ratio</span><span className="value">{(ward.demographics.elderly_ratio * 100).toFixed(1)} %</span></div>
          <div className="row"><span className="label">Outdoor workers</span><span className="value">{(ward.demographics.outdoor_worker_share * 100).toFixed(1)} %</span></div>
          <div className="row"><span className="label">Informal housing</span><span className="value">{(ward.demographics.informal_housing_share * 100).toFixed(1)} %</span></div>
          <div className="row"><span className="label">Population</span><span className="value">{ward.demographics.population.toLocaleString()}</span></div>
        </div>
      )}
    </div>
  );
}
