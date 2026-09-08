# Agent Specification

## 1. Purpose

The decision agent reasons over ward-level risk and proposes municipal interventions. It does **not** calculate scientific values and does **not** claim certainty. Determinism is the source of truth: the rule-based layer builds the plan; an optional LLM may only **select and rank** from that deterministic candidate pool, never generate actions, numbers, evidence, or events.

## 2. System Prompt

The prompt sent to the LLM via OpenRouter (see `backend/internal/agent/openrouter.go`):

```
You are the Municipal Heat Decision Agent. Respond only with strict JSON.
Never invent numbers, events, or outcomes.
```

The full user prompt (see `backend/internal/agent/narrator.go` `buildPrompt`) carries:

- Severity band, ward name, MRI, WBGT, UTCI, confidence
- Whether vulnerability and precedent data were available
- The deterministic candidate pool, one line per action:
  `- <ID>: <Text> (current priority <P>, evidence: <e>)`
- Matching historical events (when precedent found)
- Forecast trajectory
- Optional municipal context

The instruction in the prompt: **select and rank a non-empty subset** of the candidates best-first, output only `{why, reasoning, actions:[{id,priority}], precedent_event_ids}`, where `precedent_event_ids` must be empty when no precedent was found. No other fields.

## 3. Orchestration Boundary

The LLM client is isolated in `backend/internal/agent` (OpenRouter transport + prompt/narrative types). `internal/orchestration` computes only pre-computed plan facts and passes them in. Reversely, the LLM response crosses the boundary back as a **proposal** and is validated before it can touch the plan (see §5). This separation keeps the transport testable in isolation (`httptest`, `agent/agent_test.go`) and the deterministic plan authoritative.

## 4. Output Schema (strict)

```json
{
  "severity": "LOW | MODERATE | HIGH | VERY_HIGH | EXTREME",
  "priority_wards": [
    { "ward_id": "ward_001", "ward_name": "string", "score": 51.2, "band": "EXTREME" }
  ],
  "reasoning_summary": "string",
  "key_factors": ["string", "..."],
  "recommended_actions": [
    {
      "action": "string",
      "priority": "LOW | MEDIUM | HIGH",
      "target": "ward_id | city",
      "reason": "string",
      "evidence": ["current_risk", "vulnerability", "forecast", "historical_precedent"]
    }
  ],
  "public_advisory": "string",
  "recheck_interval_minutes": 15 | 30 | 60 | 120 | 360,
  "agent_run_id": "uuid",
  "fallback_used": false,
  "agent_version": "agent-v2.0.0",
  "why_this_matters": "string",
  "historical_context": {
    "found": true,
    "type": "local | demo | none",
    "event_id": "string?",
    "similarity_reason": "string?",
    "source": "string?",
    "synthetic": false
  },
  "matching_events": [
    {
      "event_id": "string",
      "date": "string",
      "ward_name": "string?",
      "score": 46,
      "band": "HIGH",
      "similarity": 1.0,
      "dimensions": { "wbgt": 0.94 },
      "actions_taken": ["string"],
      "outcome": "string"
    }
  ],
  "evidence": {
    "current_risk": true,
    "vulnerability": true,
    "forecast": true,
    "historical_precedent": true,
    "external_reference": false,
    "general_guidance": false
  },
  "confidence": "HIGH | MEDIUM | LOW",
  "limitations": ["string", "..."]
}
```

Every `recommended_action` carries a `reason` and an `evidence` tag list, so the provenance of the recommendation is inspectable. `external_reference` is always `false` in this prototype — no cited external sources are ever claimed.

## 5. Failure Behavior

When `AGENT_LLM_PROVIDER=openrouter` and an API key and model are configured, the LLM is called (8s timeout) with the candidate pool. The response is treated as a **proposal**, not a verdict. Every failure mode below sets `fallback_used = true` and appends a limitation to `limitations`, and the deterministic plan is persisted anyway:

- **LLM call error / timeout** → limitation: "LLM narrative unavailable; deterministic reasoning used."
- **Validation gate rejection** → limitation: "LLM proposal rejected by validation (<reason>); deterministic reasoning used."

### 5.1 Validation gate (`applyLLMProposal`)

The proposal is accepted only if all of:

1. `why` and `reasoning` are non-empty prose.
2. `actions` is a non-empty subset of the candidate pool, with no duplicate IDs and no unknown IDs ("unrecognised action id").
3. Every selected `priority` is one of `LOW | MEDIUM | HIGH`.
4. `precedent_event_ids` is empty when no historical precedent was found; otherwise every ID must match a known matching event.

The strict JSON parser (`json.Decoder` + `DisallowUnknownFields`) independently rejects completions that smuggle extra invented fields — an unknown key anywhere in the response is a hard error.

On acceptance, the plan is mutated **only** to: reorder/subset `recommended_actions` per the model's ranking (overriding priorities), and replace `reasoning_summary` / `why_this_matters`. Action texts, targets, per-action reasons, and evidence tags stay deterministic. A rejection leaves the plan untouched (deterministic).

### 5.2 Forecast-scoped plans (horizon awareness)

`buildActionPlan(wardIDs, ctx, horizonHours)` runs the deterministic layer against either current or forecast metrics, controlled by a `planScope`:

- **Current scope** (`horizon_hours == 0` or `> 1` ward requested): the top ward's **current** risk drives severity, key_factors, evidence, memory matching, and the LLM narrative context — the pre-upgrade behavior.
- **Forecast scope** (exactly one ward + `horizon_hours > 0`): the ward's forecast cell at `idx = clamp(horizon_hours/24, 0, len-1)` supplies the scoped metric (MRI/WBGT/UTCI/band). `key_factors` become e.g. `MRI at Thu 10 = 39.4 (MODERATE)`, `WBGT at Thu 10 = 31.1 °C`, `UTCI at Thu 10 = 23.0 °C`, plus trajectory. The `whyThisMatters` prose is forecast-anchored (`"<ward> is forecast to be at <band> heat risk at the <label> horizon …"`). The plan carries `scope`, `scope_ward_id`, `scope_horizon_hours`, `scope_label`.

Guarantees:
- Current metrics are **never** mixed into future reasoning, and vice-versa — the scope selects one metric source for the whole plan.
- `priority_wards` is **always** current-risk ranked (spec §19), never the forecast cell.
- The LLM `NarrativeInput` gets the scoped metric + `scope_label`, so the model only ever sees the same horizon the rest of the plan uses.
- All validation gates, fallback semantics, and memory matching remain identical regardless of scope.

## 6. Rule-based Reasoning

The deterministic layer always runs; the optional LLM selects and ranks from its output (never appends to it):

```
candidates = wards in scope, sorted by risk score desc
top = highest-risk ward; severity = top band
vuln = vulnerability index(top demographics)
forecast = forecast series(top ward)
trajectory = worsening | roughly stable | improving | unknown   # first series value vs max of the rest

evidence:
  current_risk        = score > 0
  vulnerability       = vuln > 0
  forecast            = forecast series present
  historical_precedent = matching memory event found
  external_reference  = false (always)
  general_guidance    = !historical_precedent

if band == EXTREME:
    recommend:
      - Activate cooling council centres and night shelters (HIGH)
      - Issue outdoor-work advisory, shift timings (HIGH)
      - Pre-position ORS, water, cooling materials (MEDIUM)
    recheck: 60 min
elif band == VERY_HIGH:
    recommend:
      - Pre-position ORS, water, cooling materials (MEDIUM)
    recheck: 60 min
elif band == HIGH:
    recommend:
      - Issue municipal heat-health advisory (LOW)
    recheck: 360 min
else:
    recommend:
      - City-wide awareness advisory (LOW)
    recheck: 360 min
if band in (EXTREME, VERY_HIGH):
    append Direct vulnerable-population outreach (MEDIUM), quoting vuln
if historical precedent found:
    append "Replicate outcome-backed precedent actions from memory" (MEDIUM)
```

The chosen LLM model is `nvidia/nemotron-3.5-lightning:free` (default `LLM_MODEL`); OpenRouter is the only provider.

Confidence: `HIGH` when best match similarity ≥ 0.65; else `MEDIUM` when both current_risk and vulnerability evidence hold; else `LOW`. `matching_events` (≤3) come from a damped similarity search over the historical memory (learned local journal + optional demo fixture): risk score, WBGT, UTCI, vulnerability profile, ward and season dimensions, renormalized over present dimensions. LIVE pipeline runs append to the local journal (`data/historical/alert_history.json`); mock/degraded runs never persist.

## 7. Test Scenarios

| Scenario | Expected Behavior |
|---|---|
| High-risk EXTREME band | Returns EXTREME severity, cooling-center action |
| Low-risk LOW band | Returns LOW severity, advisory only |
| Missing vulnerability | Falls back to thermal-only reasoning |
| Conflicting data | Defers to highest severity band, notes conflict |
| Missing data | Marks degraded quality, uses rule-based fallback |
| LLM timeout | Uses fallback, sets `fallback_used=true` |

## 8. Audit Trail

Every agent run persists:
- `agent_run_id` (UUID)
- Inputs (ward_ids, context)
- Tool calls (timestamps, results)
- Final plan
- `fallback_used` flag
- `physics_version`, `risk_model_version` retrieved