import type { Band, Alert } from '../types';
import { BandChip } from './BandChip';

interface DayRow { label: string; band: Band; }
export default function PublicAdvisory({ areaName, rows, advisory, advisoryText }: {
  areaName: string;
  rows: DayRow[];
  advisory?: Alert | null;
  advisoryText: string;
}) {
  const top = rows[0];
  const actions = advisory?.recommended_actions ?? [];
  return (
    <div className="public-shell">
      <h1 className={`severity-${top?.band.toLowerCase()}`}>
        {top?.band === 'EXTREME' ? 'EXTREME HEAT' :
         top?.band === 'VERY_HIGH' ? 'VERY HIGH HEAT' :
         top?.band === 'HIGH' ? 'HIGH HEAT' :
         top?.band === 'MODERATE' ? 'MODERATE HEAT' : 'HEAT ADVISORY'} EXPECTED
      </h1>
      <h2>Your area: {areaName}</h2>

      <div className="card">
        <div className="section-title">Next days</div>
        {rows.map((r) => (
          <div key={r.label} className="public-day-row">
            <strong>{r.label}</strong>
            <BandChip band={r.band} />
          </div>
        ))}
      </div>

      <div className="card" style={{ marginTop: 16 }}>
        <div className="section-title">Protect yourself</div>
        {actions.length > 0 ? (
          <ul style={{ paddingLeft: 18, margin: 0 }}>
            {actions.map((a, i) => (
              <li key={i}>{a.action}</li>
            ))}
          </ul>
        ) : (
          <ul style={{ paddingLeft: 18, margin: 0 }}>
            <li>Avoid outdoor activity 12–4 PM</li>
            <li>Drink water regularly</li>
            <li>Check on elderly people</li>
          </ul>
        )}
      </div>

      <div className="public-advisory">{advisoryText}</div>

      {advisory && (
        <p className="muted" style={{ marginTop: 16, fontSize: 11 }}>
          Advisory issued {new Date(advisory.created_at).toLocaleString()} for {advisory.ward_id} after
          municipal review of the decision agent plan.
        </p>
      )}
    </div>
  );
}
