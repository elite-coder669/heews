import type { Alert } from '../types';
import { riskColor } from '../styles/tokens';

export function AlertTicker({ alerts }: { alerts: Alert[] }) {
  if (!alerts.length) return null;
  return (
    <div className="alert-ticker">
      {alerts.slice(0, 3).map((a) => (
        <div key={a.id} className="alert-card" style={{ borderLeftColor: riskColor(a.severity) }}>
          <h4>{a.ward_id} · {a.severity.replace('_', ' ')}</h4>
          <div className="ts">{new Date(a.created_at).toLocaleString()}</div>
          <p style={{ margin: '6px 0 0', fontSize: 12 }}>{a.headline}</p>
        </div>
      ))}
    </div>
  );
}

export function AlertCenter({ alerts, onSelectWard }: { alerts: Alert[]; onSelectWard?: (wardId: string) => void }) {
  return (
    <div className="alert-center">
      <h1>Alert Center</h1>
      {alerts.length === 0 && <p className="muted">No active alerts. Monitor continues.</p>}
      {alerts.map((a) => (
        <button
          key={a.id}
          className="alert-row"
          style={{ borderLeftColor: riskColor(a.severity), width: '100%', textAlign: 'left', cursor: 'pointer' }}
          onClick={() => onSelectWard?.(a.ward_id)}
          title="Open in console map"
        >
          <div className="body">
            <h3>{a.ward_id} · {a.severity.replace('_', ' ')}</h3>
            <div className="meta">
              {new Date(a.created_at).toLocaleString()} · {a.status}
              {a.source && <span> · {a.source === 'municipal' ? 'municipal (approved)' : 'pipeline'}</span>}
            </div>
            <p style={{ marginTop: 6 }}>{a.headline}</p>
            {a.recommended_actions.length > 0 && (
              <ul style={{ marginTop: 6 }}>
                {a.recommended_actions.map((ra, i) => (
                  <li key={i}><span style={{ fontWeight: 600 }}>{ra.priority}:</span> {ra.action}</li>
                ))}
              </ul>
            )}
          </div>
        </button>
      ))}
    </div>
  );
}
