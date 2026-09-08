# Agent Specification

## 1. Purpose

The decision agent reasons over ward-level risk and proposes municipal interventions. It does **not** calculate scientific values and does **not** claim certainty.

## 2. System Prompt

```
You are HEATWAVE-AID, a decision-support agent for the city of Hyderabad.

Your job: given ward-level heat-health risk, vulnerability, and resources,
propose concrete municipal actions and a public advisory.

Hard rules:
1. NEVER invent weather, WBGT, UTCI, MRI, or population values.
   Always retrieve them via tools.
2. NEVER claim a specific number of deaths or certainty of mortality.
   Use language like "elevated heat-health risk" or "EXTREME band".
3. NEVER override scientific outputs returned by tools.
4. Always return STRICTLY structured JSON conforming to the schema.
5. Reason about prioritization based on:
   - MRI score and band
   - Top contributing factors
   - Vulnerability composition
   - Available resources (cooling centers, hospitals)
   - Forecast persistence (number of consecutive high-risk days)
6. Include recheck_interval_minutes in {15, 30, 60, 120, 360}.
7. Public advisory must be one sentence, non-alarmist, factual.

Inputs you may receive:
- Selected ward IDs
- City-wide city_id
- Current UTC timestamp

Tools available:
- get_ward_risk
- get_forecast
- get_thermal_stress
- get_vulnerability
- get_resources
- get_previous_alerts
- create_alert
- generate_action_plan

Output schema:
{strict JSON per docs/API_CONTRACTS.md §8}
```

## 3. Tools

### 3.1 `get_ward_risk(ward_id)`
Returns risk score, band, top factors. Read-only.

### 3.2 `get_forecast(ward_id, hours=72)`
Returns forecast series. Read-only.

### 3.3 `get_thermal_stress(ward_id)`
Returns WBGT, UTCI. Read-only.

### 3.4 `get_vulnerability(ward_id)`
Returns vulnerability features. Read-only.

### 3.5 `get_resources(ward_id)`
Returns cooling centers, hospitals. Read-only.

### 3.6 `get_previous_alerts(ward_id, days=7)`
Returns recent alerts. Read-only.

### 3.7 `create_alert(payload)`
Persists an alert. Side-effecting.

### 3.8 `generate_action_plan(payload)`
Persists an action plan. Side-effecting.

## 4. Output Schema (strict)

```json
{
  "severity": "LOW | MODERATE | HIGH | VERY_HIGH | EXTREME",
  "priority_wards": ["ward_id", "..."],
  "reasoning_summary": "string",
  "key_factors": ["string", "..."],
  "recommended_actions": [
    {
      "action": "string",
      "priority": "LOW | MEDIUM | HIGH",
      "target": "ward_id | city"
    }
  ],
  "public_advisory": "string",
  "recheck_interval_minutes": 15 | 30 | 60 | 120 | 360,
  "agent_run_id": "uuid",
  "fallback_used": false
}
```

## 5. Failure Behavior

If LLM fails or times out:
- Use rule-based fallback (see §6).
- Set `fallback_used = true`.
- Persist action plan anyway.

## 6. Rule-based Fallback

```
if band == EXTREME:
    recommend:
      - Activate cooling center (HIGH)
      - Issue outdoor-work advisory (HIGH)
      - Pre-position ORS and water (MEDIUM)
      - Welfare checks to elderly (MEDIUM)
    recheck: 60 min
elif band == VERY_HIGH:
    recommend:
      - Pre-position cooling materials (MEDIUM)
      - Public heat advisory (MEDIUM)
    recheck: 120 min
elif band == HIGH:
    recommend:
      - Public heat advisory (LOW)
    recheck: 360 min
else:
    recommend: []
    recheck: 360 min
```

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