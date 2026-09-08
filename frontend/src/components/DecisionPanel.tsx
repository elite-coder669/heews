import { useState, useEffect } from 'react';
import type { ActionPlan } from '../types';
import { BandChip, PriorityChip } from './BandChip';

interface Props {
  plan: ActionPlan | null;
  loading: boolean;
  wardName?: string;
  onIssueAlert?: (plan: ActionPlan) => void;
  issuing?: boolean;
}

export default function DecisionPanel({ plan, loading, wardName, onIssueAlert, issuing }: Props) {
  const [reviewing, setReviewing] = useState(false);

  useEffect(() => { setReviewing(false); }, [plan]);

  if (loading) {
    return (
      <div className="decision-panel">
        <header><h2>Municipal Decision Agent</h2></header>
        <p className="muted">Reasoning over risk + vulnerability + precedent…</p>
      </div>
    );
  }
  if (!plan) {
    return (
      <div className="decision-panel">
        <header><h2>Municipal Decision Agent</h2></header>
        <p className="muted">Select a ward on the map. The agent builds a proactive decision plan from current risk, forecast trend, vulnerability, and historical precedent.</p>
      </div>
    );
  }

  const hc = plan.historical_context;
  const fallback = plan.fallback_used;

  return (
    <div className="decision-panel">
      <header>
        <h2>Municipal Decision Agent</h2>
        <BandChip band={plan.severity} />
      </header>

      {plan.scope_label && (
        <div className="scope-badge" data-scope={plan.scope}>
          Scope: {plan.scope_label}{plan.scope === 'forecast' ? ` · ${plan.scope_horizon_hours}h ahead` : ''}
        </div>
      )}
      {fallback && (
        <div className="fallback-badge">
          AI narrative unavailable — deterministic decision plan shown
        </div>
      )}

      <div className="reasoning">
        <strong>Why this ward is prioritized</strong>
        <p style={{ margin: '6px 0 0' }}>{plan.why_this_matters || plan.reasoning_summary}</p>
        {plan.key_factors.length > 0 && (
          <ul style={{ margin: '8px 0 0', paddingLeft: 18 }}>
            {plan.key_factors.map((k, i) => <li key={i}>{k}</li>)}
          </ul>
        )}
        {plan.confidence && (
          <p className="muted" style={{ margin: '8px 0 0' }}>
            Model confidence: {plan.confidence}
          </p>
        )}
      </div>

      {/* Historical context */}
      <div className="hist-section">
        <div className="section-title">Historical Context</div>
        {hc.found && hc.type !== 'none' ? (
          <>
            <p className="muted" style={{ margin: '0 0 6px' }}>
              <strong>{Math.round((plan.matching_events?.[0]?.similarity ?? 0) * 100)}% similar</strong> precedent · source{' '}
              <span className="mono">{hc.source}</span> · {hc.type === 'local' ? 'learned local journal' : hc.synthetic ? 'synthetic demo memory' : 'demo'}
            </p>
            {hc.similarity_reason && (
              <p style={{ margin: '0 0 6px', fontSize: 12 }}>Why similar: {hc.similarity_reason}</p>
            )}
            {plan.matching_events?.map((m) => (
              <div key={m.event_id} className="match-row">
                <span><strong>{m.ward_name ?? m.event_id.slice(0, 8)}</strong> · {m.date}</span>
                <span className="muted">MRI {m.score.toFixed(0)} · {m.band}</span>
                {m.actions_taken && m.actions_taken.length > 0 && (
                  <span className="actions">Took: {m.actions_taken.join('; ')}</span>
                )}
                {m.outcome && <span className="outcome">Outcome: {m.outcome}</span>}
              </div>
            ))}
          </>
        ) : (
          <p className="muted" style={{ margin: 0, fontSize: 12 }}>
            No sufficiently similar local historical event found for this ward and conditions.
          </p>
        )}
      </div>

      {/* Recommended actions */}
      <ul className="actions-list">
        {plan.recommended_actions.map((a, i) => (
          <li key={i}>
            <span className="idx">{String(i + 1).padStart(2, '0')}</span>
            <span>
              <strong>{a.action}</strong>
              <span className="target">{a.target}</span>
              <span style={{ display: 'block', fontSize: 12, color: 'var(--muted)' }}>
                {a.evidence && a.evidence.length > 0 ? `Based on: ${a.evidence.join(', ')}` : a.reason}
              </span>
            </span>
            <PriorityChip priority={a.priority} />
          </li>
        ))}
      </ul>

      {plan.limitations.length > 0 && (
        <div className="row" style={{ marginTop: 10, fontSize: 11 }}>
          {plan.limitations.map((l, i) => (
            <span key={i} className="muted" style={{ display: 'block' }}>· {l}</span>
          ))}
        </div>
      )}

      {plan.public_advisory && (
        <div className="public-advisory">
          <strong>Public advisory:</strong> {plan.public_advisory}
        </div>
      )}

      {/* Review Actions → Issue Alert */}
      {onIssueAlert && !reviewing && (
        <button
          className="primary"
          style={{ width: '100%', marginTop: 12 }}
          onClick={() => setReviewing(true)}
        >
          Review Actions
        </button>
      )}
      {reviewing && (
        <div className="review-box">
          <p className="muted" style={{ margin: '0 0 8px', fontSize: 12 }}>
            Review the recommended actions above. Issue Alert will publish a municipal alert for{' '}
            <strong>{wardName}</strong> reusing this validated plan as-is, and route a citizen-friendly
            advisory downstream. No action is dispatched autonomously.
          </p>
          <div className="review-actions">
            <button onClick={() => setReviewing(false)}>Cancel</button>
            <button
              className="primary"
              disabled={issuing}
              onClick={() => onIssueAlert?.(plan)}
            >
              {issuing ? 'Issuing…' : 'Issue Alert'}
            </button>
          </div>
        </div>
      )}

      <div className="row" style={{ marginTop: 12, fontSize: 12 }}>
        <span className="muted">Re-check in {plan.recheck_interval_minutes} min</span>
        <span className="mono muted">{plan.agent_run_id.slice(0, 8)}</span>
        <span className="mono muted">{plan.agent_version}</span>
        {fallback && <span className="muted"> · deterministic</span>}
      </div>
    </div>
  );
}
