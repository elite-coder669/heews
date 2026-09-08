import type { Ward, WardRisk, WardThermal, WardWeather } from '../types';
import { BandChip, FactorBar, PriorityChip } from './BandChip';
import { RISK_COLORS } from '../styles/tokens';

interface Props {
  ward?: Ward;
  risk?: WardRisk | null;
  thermal?: WardThermal | null;
  weather?: WardWeather | null;
  loading: boolean;
}

export default function DetailDrawer({ ward, risk, thermal, weather, loading }: Props) {
  if (loading) {
    return (
      <div className="drawer-empty">
        <h3>Loading ward risk…</h3>
        <p>Fetching forecast and computed risk index.</p>
      </div>
    );
  }

  if (!ward || !risk) {
    return (
      <div className="drawer-empty">
        <h3>Select a ward</h3>
        <p>Click any polygon on the map to inspect risk, thermal stress, weather, and explanations.</p>
      </div>
    );
  }

  return (
    <div>
      <div className="ward-header">
        <h2>{ward.ward_name}</h2>
        <span className="zone">Ward #{ward.ward_number} · {ward.zone}</span>
      </div>

      {/* Level 1 — Risk */}
      <div className="card" style={{ marginBottom: 12 }}>
        <div className="section-title">Risk</div>
        <div className="score-display">
          <span className="num">{risk.risk.score.toFixed(0)}</span>
          <span className="out">/ 100</span>
          <BandChip band={risk.risk.band} />
        </div>
        <div className="row"><span className="label">Data quality</span><span className="value">
          {thermal?.current.wbgt_quality === 'degraded' ? 'WBGT degraded — radiation missing' : 'Good'}
        </span></div>
        <div className="row"><span className="label">Computed</span><span className="value mono">{risk.computed_at}</span></div>
      </div>

      {/* Level 2 — Thermal */}
      {thermal && (
        <div className="card" style={{ marginBottom: 12 }}>
          <div className="section-title">Physiological Stress</div>
          <div className="row"><span className="label">WBGT</span><span className="value">{thermal.current.wbgt_c.toFixed(1)} °C</span></div>
          <div className="row"><span className="label">UTCI</span><span className="value">{thermal.current.utci_c.toFixed(1)} °C</span></div>
          <div className="row"><span className="label">WBGT quality</span><span className="value">{thermal.current.wbgt_quality}</span></div>
          <div className="row"><span className="label">Physics version</span><span className="value mono">{thermal.physics_version}</span></div>
        </div>
      )}

      {/* Level 3 — Weather */}
      {weather && (
        <div className="card" style={{ marginBottom: 12 }}>
          <div className="section-title">Environment</div>
          <div className="row"><span className="label">Temperature</span><span className="value">{weather.current.temperature_c.toFixed(1)} °C</span></div>
          <div className="row"><span className="label">Humidity</span><span className="value">{weather.current.humidity_pct.toFixed(0)} %</span></div>
          <div className="row"><span className="label">Wind</span><span className="value">{weather.current.wind_ms.toFixed(1)} m/s</span></div>
          <div className="row"><span className="label">Radiation</span><span className="value">{weather.current.shortwave_rad_wm2.toFixed(0)} W/m²</span></div>
          {weather.current.cloud_cover_pct !== undefined && (
            <div className="row"><span className="label">Cloud cover</span><span className="value">{weather.current.cloud_cover_pct.toFixed(0)} %</span></div>
          )}
        </div>
      )}

      {/* Level 4 — Why? */}
      <div className="card">
        <div className="section-title">Why is this ward {risk.risk.band.toLowerCase().replace('_', ' ')}?</div>
        <div className="factors">
          {risk.risk.top_factors.map((f, i) => {
            const pct = 90 - i * 12;
            return <FactorBar key={i} name={f.length > 28 ? f.slice(0, 26) + '…' : f} value={pct} />;
          })}
        </div>
        <p style={{ marginTop: 12, fontSize: 12, color: 'var(--muted)' }}>
          {risk.risk.top_factors[0]}. Combined with vulnerability composition, this drives the
          band classification. Risk model version <span className="mono">{risk.model_version}</span>.
        </p>
      </div>

      {/* Vulnerability */}
      {ward.demographics && (
        <div className="card" style={{ marginTop: 12 }}>
          <div className="section-title">Vulnerability</div>
          <div className="row"><span className="label">Elderly ratio</span><span className="value">{(ward.demographics.elderly_ratio * 100).toFixed(1)} %</span></div>
          <div className="row"><span className="label">Outdoor workers</span><span className="value">{(ward.demographics.outdoor_worker_share * 100).toFixed(1)} %</span></div>
          <div className="row"><span className="label">Informal housing</span><span className="value">{(ward.demographics.informal_housing_share * 100).toFixed(1)} %</span></div>
          <div className="row"><span className="label">Population</span><span className="value">{ward.demographics.population.toLocaleString()}</span></div>
        </div>
      )}
    </div>
  );
}