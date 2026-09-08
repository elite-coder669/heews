# Demo Flow (3–5 minutes)

## Setup
- App is in MOCK mode (deterministic) by default for reliable demo.
- City: Hyderabad. Wards: ~150 (use real or synthetic fixtures).

## Script

### 1. City Overview (30 s)
- Open `/` → choropleth map renders.
- Wards colored by current band: a clear gradient from greens to deep red.
- Banner: "Decision-support prototype. Not medical advice."

### 2. Pick a high-risk ward (30 s)
- Click on a red/purple ward → detail panel.
- Show:
    - Temperature 41.2 °C
    - Humidity 58 %
    - Wind 2.1 m/s
    - Radiation 710 W/m²
    - **WBGT 33.8 °C** (Liljegren)
    - **UTCI 44.6 °C** (Very Strong Heat Stress)
    - **MRI 82** → **EXTREME**
    - Top factors (3 bullets)

### 3. Forecast timeline (30 s)
- Switch to forecast tab.
- Show 3–5 day persistence of the EXTREME band.
- Highlight overnight recovery gap (small or absent).

### 4. Decision agent (60 s)
- Click "Ask Agent".
- Query: "What should the municipality do for the next 24 hours?"
- Agent retrieves risk, forecast, vulnerability, resources.
- Returns structured action plan:
    - "Activate ward cooling center — HIGH"
    - "Issue outdoor-work advisory 12:00–16:00 — HIGH"
    - "Pre-position ORS + water — MEDIUM"
    - "Welfare checks — MEDIUM"
- Public advisory shown in plain English.

### 5. Alert (20 s)
- Click "Generate Alert" → alert appears in Alert Center.
- Show provenance: `physics_version`, `risk_model_version`, `agent_run_id`.

### 6. MOCK mode proof (30 s)
- Open `.env.example` → switch to `MOCK=true`.
- Refresh dashboard → identical results with no internet.
- This proves reproducibility and auditability.

### 7. Closing (20 s)
- "We integrated six layers of intelligence:
  weather + spatial + physiological + vulnerability + temporal + decision."
- "The agent reasons; the physics decides; the system supports."

Total: ≈ 3:30.

## Failure Demo (optional)
- If time permits, demonstrate Open-Meteo timeout → cached fallback banner appears.