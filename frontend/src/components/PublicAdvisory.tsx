import type { Band } from '../types';
import { BandChip } from './BandChip';

interface DayRow { label: string; band: Band; }
export default function PublicAdvisory({ areaName, rows, advisoryText }: {
  areaName: string;
  rows: DayRow[];
  advisoryText: string;
}) {
  const top = rows[0];
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
        <ul style={{ paddingLeft: 18, margin: 0 }}>
          <li>Avoid outdoor activity 12–4 PM</li>
          <li>Drink water regularly</li>
          <li>Check on elderly people</li>
          <li>Use designated cooling centres</li>
        </ul>
      </div>

      <div className="public-advisory">{advisoryText}</div>
    </div>
  );
}