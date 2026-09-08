import type { Band } from '../types';
import { BandChip } from './BandChip';
import { RISK_COLORS } from '../styles/tokens';

interface DayRow {
  label: string;
  wbgt: number;
  utci: number;
  mri: number;
  band: Band;
}

interface Props {
  rows: DayRow[];
}

export default function ForecastStrip({ rows }: Props) {
  if (!rows.length) return null;
  return (
    <div className="forecast-strip">
      <h3>5-day risk forecast · selected ward</h3>
      <div className="forecast-grid">
        <div className="head">Day</div>
        {rows.map((r) => <div key={r.label} className="head">{r.label}</div>)}

        <div className="head">WBGT °C</div>
        {rows.map((r) => <div key={r.label}>{r.wbgt.toFixed(1)}</div>)}

        <div className="head">UTCI °C</div>
        {rows.map((r) => <div key={r.label}>{r.utci.toFixed(1)}</div>)}

        <div className="head">MRI</div>
        {rows.map((r) => (
          <div key={r.label} className="mri-bar">
            <div style={{ width: `${r.mri}%`, background: RISK_COLORS[r.band] }} />
          </div>
        ))}

        <div className="head">Band</div>
        {rows.map((r) => (
          <div key={r.label} className="band-cell">
            <BandChip band={r.band} />
          </div>
        ))}
      </div>
    </div>
  );
}