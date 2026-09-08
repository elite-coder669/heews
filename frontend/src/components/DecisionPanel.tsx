import type { ActionPlan } from '../types';
import { BandChip, PriorityChip } from './BandChip';

export default function DecisionPanel({ plan, loading }: { plan: ActionPlan | null; loading: boolean }) {
  if (loading) {
    return (
      <div className="decision-panel">
        <header><h2>Decision Support</h2></header>
        <p className="muted">Reasoning over risk + vulnerability + resources…</p>
      </div>
    );
  }
  if (!plan) {
    return (
      <div className="decision-panel">
        <header><h2>Decision Support</h2></header>
        <p className="muted">Select a ward and request an action plan to see reasoning and recommended actions.</p>
      </div>
    );
  }

  return (
    <div className="decision-panel">
      <header>
        <h2>Decision Support</h2>
        <BandChip band={plan.severity} />
      </header>

      <div className="reasoning">
        <strong>Why is this ward prioritized?</strong>
        <p style={{ margin: '6px 0 0' }}>{plan.reasoning_summary}</p>
        {plan.key_factors.length > 0 && (
          <ul style={{ margin: '8px 0 0', paddingLeft: 18 }}>
            {plan.key_factors.map((k, i) => <li key={i}>{k}</li>)}
          </ul>
        )}
      </div>

      <ul className="actions-list">
        {plan.recommended_actions.map((a, i) => (
          <li key={i}>
            <span className="idx">{String(i + 1).padStart(2, '0')}</span>
            <span>
              <strong>{a.action}</strong>
              <span className="target">{a.target}</span>
            </span>
            <PriorityChip priority={a.priority} />
          </li>
        ))}
      </ul>

      {plan.public_advisory && (
        <div className="public-advisory">
          <strong>Public advisory:</strong> {plan.public_advisory}
        </div>
      )}

      <div className="row" style={{ marginTop: 12, fontSize: 12 }}>
        <span className="muted">Re-check in {plan.recheck_interval_minutes} min</span>
        <span className="mono muted">{plan.agent_run_id.slice(0, 8)}</span>
        {plan.fallback_used && <span className="muted"> · rule-based fallback</span>}
      </div>
    </div>
  );
}