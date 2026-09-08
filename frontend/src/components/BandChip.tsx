import type { Band } from '../types';
import { RISK_COLORS, RISK_LABEL } from '../styles/tokens';

export function BandChip({ band }: { band: Band | string }) {
  const cls = `chip chip-${band}`;
  return <span className={cls}>{RISK_LABEL[band as Band] ?? band}</span>;
}

export function FactorBar({ name, value, max = 100 }: { name: string; value: number; max?: number }) {
  const pct = Math.min(100, Math.max(0, (value / max) * 100));
  const high = pct >= 70;
  const color = high ? RISK_COLORS.VERY_HIGH : pct >= 40 ? RISK_COLORS.HIGH : RISK_COLORS.LOW;
  return (
    <div className="factor">
      <span className="name">{name}</span>
      <div className="bar"><div style={{ width: `${pct}%`, background: color }} /></div>
      <span className="delta">{value.toFixed(0)}</span>
    </div>
  );
}

export function PriorityChip({ priority }: { priority: 'LOW' | 'MEDIUM' | 'HIGH' | string }) {
  return <span className={`pri-${priority}`} style={{
    fontSize: 10, fontWeight: 700, letterSpacing: '0.06em', padding: '2px 8px', borderRadius: 999,
    background: priority === 'HIGH' ? '#DC2626' : priority === 'MEDIUM' ? '#F59E0B' : '#CBD5E1',
    color: priority === 'LOW' ? '#0F172A' : 'white',
  }}>{priority}</span>;
}