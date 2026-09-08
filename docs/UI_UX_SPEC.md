# UI / UX Specification — Municipal Heat Intelligence Console

**Document ID:** HEEWS-UI-001
**Version:** 1.0
**Status:** Approved for Hackathon MVP
**Audience:** Frontend engineer, judges, mentors
**Last Updated:** 2026-09-07

This document is the authoritative UX specification for the HEEWS dashboard. It is binding for the frontend team.

---

## 1. UX Goal

When a municipal official opens HEEWS, they must immediately understand:

> **Where is heat dangerous right now, where will it become dangerous, and what should I do?**

Not "Here are 15 weather charts." The dashboard is a **decision-support console**, not a weather viewer.

The user narrative we are building toward:

```
See  →  Understand  →  Predict  →  Explain  →  Act
```

---

## 2. User Personas

| Persona | Goal | Primary Use |
|---|---|---|
| Municipal Administrator | Allocate resources city-wide | Map + Action panel |
| Emergency / Health Official | Triage wards, justify interventions | Detail panel + Reasoning |
| Field Operator | Confirm on-ground readiness | Alerts + Public advisory |
| Public User | Personal safety | Public advisory only |
| System Administrator | Pipeline health | Status bar |

---

## 3. User Journeys

1. **Daily check** — Open dashboard → see city summary → scan map → close.
2. **Triage** — Click Extreme → filter map → click ward → read reasoning → forward action plan.
3. **Forecast review** — Move time slider from NOW → +72h → identify which wards escalate.
4. **Alert response** — Open Alert Center → read recommended action → acknowledge / dispatch.
5. **Public safety** — Switch to Public Advisory → share simplified message.

---

## 4. Information Architecture

```
HEEWS
├── City Console (default)
│   ├── Header
│   │   ├── Logo + Title
│   │   ├── LIVE/MOCK indicator
│   │   ├── Last-update timestamp
│   │   └── Pipeline status
│   ├── Situation Summary
│   │   └── 4 risk-band counters (clickable filters)
│   ├── Main Layout (60/40 split)
│   │   ├── GIS Map (left, 60%)
│   │   │   ├── Choropleth layer
│   │   │   ├── Time slider (NOW … +5d)
│   │   │   ├── Risk legend
│   │   │   └── Map controls
│   │   └── Detail Drawer (right, 40%)
│   │       ├── Selected ward header
│   │       ├── Risk card
│   │       ├── Thermal card (WBGT, UTCI)
│   │       ├── Weather card
│   │       ├── Why? card (explanations)
│   │       └── Vulnerability card
│   ├── Forecast Strip (5-day table + sparkline)
│   ├── Decision Support Panel (agent plan)
│   └── Alert Ticker (latest 3 alerts)
├── Alert Center (full page)
├── Public Advisory (read-only simplified view)
└── About / Disclaimer (modal)
```

---

## 5. Dashboard Layout (Single-Screen Mental Model)

The whole story on one screen:

```
┌──────────────────────────────────────────────────────────┐
│ HEEWS  ●LIVE  Updated 16:20  Forecast → Sep 12          │ Header
├──────────────────────────────────────────────────────────┤
│  EXTREME RISK · HYDERABAD · NEXT 72 H                   │
│  [ 7 Extreme ] [ 18 High ] [ 42 Moderate ] [ 83 Low ]   │ Summary
├────────────────────────────┬─────────────────────────────┤
│                            │  WARD 42                    │
│        HYDERABAD MAP       │  EXTREME  ·  MRI 82         │
│                            │  ─────────                  │
│     🟥🟥🟧🟨🟩             │  WBGT  34.7 °C              │
│     🟥🟥🟧🟩🟩             │  UTCI  45.1 °C              │
│                            │  ─────────                  │
│   NOW ── 12h ── 24h ── 72h │  WHY?                       │
│                            │  ↑ WBGT  ↑ UTCI            │
│                            │  ↑ Persistence              │
│                            │  ↑ Vulnerability            │
├────────────────────────────┴─────────────────────────────┤
│  5-DAY FORECAST: bars + table                            │
├──────────────────────────────────────────────────────────┤
│  DECISION SUPPORT · 3 actions for Ward 42               │
│  [01 Activate cooling centre] [02 Restrict outdoor work] │
│  [03 Increase monitoring]                                │
└──────────────────────────────────────────────────────────┘
```

---

## 6. GIS Map Specification

The map is the hero. It occupies 55–65% of the main dashboard.

### 6.1 Behavior

- Polygons per ward.
- Click → select ward, populate detail drawer.
- Hover → tooltip with `ward_name` + band + score.
- Color by `risk_band` using the contract palette.
- Time slider drives the layer: NOW / +12h / +24h / +48h / +72h / +5d.
- Filter buttons at top of map: All / Extreme / High+ / Moderate+ / Low.

### 6.2 Color Palette (matches backend contract)

| Band | Hex |
|---|---|
| LOW | `#16A34A` |
| MODERATE | `#FACC15` |
| HIGH | `#F97316` |
| VERY HIGH | `#DC2626` |
| EXTREME | `#7C1D6F` |

Red/orange/yellow are reserved for **risk semantics only**. Never for buttons, icons, or accents.

### 6.3 Map Controls

- Zoom +/-
- Locate Hyderabad button
- Risk legend (bottom-left)
- Forecast time slider (bottom)

### 6.4 Library

React + `react-leaflet` + OpenStreetMap tiles (no API key required).

---

## 7. Ward Detail Drawer Specification

Right-side drawer. Never navigate away from the map.

### 7.1 Levels of Disclosure (progressive)

**Level 1 — Risk:**
```
WARD 42          Somajiguda
━━━━━━━━━━━━━━━━━━━━━━━━━━━
EXTREME
MRI  82 / 100
Data quality: GOOD
```

**Level 2 — Thermal:**
```
PHYSIOLOGICAL STRESS
WBGT  34.7 °C     (Very strong heat stress)
UTCI  45.1 °C     (Extreme heat stress)
```

**Level 3 — Weather:**
```
ENVIRONMENT
Temperature   41.2 °C
Humidity       58 %
Wind          2.1 m/s
Radiation    710 W/m²
```

**Level 4 — Provenance:**
```
physics   wbgt-liljegren-1.0
risk      risk-v1.0.0
computed  2026-09-07 16:20 IST
```

### 7.2 Why? Card

```
WHY IS THIS WARD EXTREME?

↑ WBGT             ██████████  very strong heat stress
↑ UTCI             █████████   extreme perceived thermal load
↑ Persistence      ████████    heat remains elevated for 72h
↑ Vulnerability    ███████     high outdoor-worker share

[ Expand narrative ]
> High thermal stress is expected to persist for the next 72 hours,
> while this ward has elevated outdoor-worker exposure (~18%).
```

---

## 8. Forecast Visualization

Two parts in one strip:

### 8.1 Sparkline

```
Today    +1      +2      +3      +4
 ●━━━━━━●━━━━━━●━━━━━━●
 68      79      88      84      71
```

### 8.2 Table

| Day | WBGT °C | UTCI °C | MRI | Band |
|---|---|---|---|---|
| Today | 32.4 | 41.8 | 68 | HIGH |
| Tomorrow | 34.1 | 44.3 | 79 | VERY_HIGH |
| +2d | 35.2 | 46.0 | 88 | EXTREME |
| +3d | 34.8 | 45.2 | 84 | EXTREME |
| +4d | 32.7 | 42.1 | 71 | HIGH |

---

## 9. Risk Visualization

Consistent across:

- Map fills
- Counter chips
- Band badges in detail
- Alert entries
- Forecast cells

Never re-color risk by context.

---

## 10. Decision Support Panel

Looks like an **operations console**, not a chatbot.

```
┌─────────────────────────────────────────────┐
│ MUNICIPAL DECISION SUPPORT       Ward 42    │
│                                             │
│ Why is Ward 42 prioritized?                 │
│                                             │
│  EXTREME band with sustained thermal        │
│  stress combined with elevated outdoor-     │
│  worker exposure over the next 72 hours.    │
│                                             │
│  PRIORITY ACTIONS                           │
│                                             │
│  01   Activate cooling centre               │
│       PRIORITY  HIGH                       │
│                                             │
│  02   Restrict outdoor work 12:00–16:00      │
│       PRIORITY  HIGH                       │
│                                             │
│  03   Increase monitoring cadence           │
│       PRIORITY  MEDIUM                     │
│                                             │
│  PUBLIC ADVISORY                            │
│  Stay hydrated, avoid midday sun, check on  │
│  elderly neighbors.                          │
│                                             │
│  Re-check in 60 min · agent-run-7c1d…       │
└─────────────────────────────────────────────┘
```

Three pillars visible: **Reasoning → Actions → Advisory**.

---

## 11. Alert Center (separate page)

```
ALERT CENTER                          [ All ] [ Active ] [ ACK ]

🔴 EXTREME — Ward 42 · MRI 88
   Expected peak: Tomorrow 14:00
   Action: Activate cooling centre
   [View ward]   [Acknowledge]

🟠 HIGH — Ward 17 · MRI 74
   Expected peak: Sep 9
   Action: Outdoor work advisory
   [View ward]   [Acknowledge]
```

Sorted by severity, then by computed_at.

---

## 12. Public Advisory (simplified view)

Plain English. No ML jargon.

EXTREME HEAT EXPECTED
Today:        Very High
Tomorrow:     Extreme
Day after:    Extreme

Protect yourself
- Avoid outdoor activity 12–4 PM
- Drink water regularly
- Check on elderly people
- Use cooling centres

Your area
Ward 42 — EXTREME
```

---

## 13. Design System

### 13.1 Color Tokens

```css
--bg:        #F8FAFC
--surface:   #FFFFFF
--text:      #0F172A
--muted:     #64748B
--primary:   #0F172A
--blue:      #2563EB

--risk-low:       #16A34A
--risk-moderate:  #FACC15
--risk-high:      #F97316
--risk-very-high: #DC2626
--risk-extreme:   #7C1D6F

--warning:   #F59E0B
--danger:    #DC2626
```

### 13.2 Typography

- Sans-serif only (Inter / Helvetica Neue).
- Body: 14 px / 1.45.
- Display: 28–34 px (cover-style numerics for MRI).
- Mono: 13 px (timestamps, IDs).

### 13.3 Spacing

4 / 8 / 12 / 16 / 24 / 32 / 48 px grid. No arbitrary values.

### 13.4 Radius

6 px on cards, 4 px on chips, 999 px on status pills.

---

## 14. Component Library

| Component | Purpose |
|---|---|
| `<StatusHeader />` | LIVE indicator, last update, forecast window |
| `<SituationSummary />` | 4 risk-band counters |
| `<RiskMap />` | Leaflet choropleth + time slider |
| `<DetailDrawer />` | Ward detail panel (Levels 1–4) |
| `<WhyCard />` | Explainability bars |
| `<ForecastStrip />` | 5-day table + sparkline |
| `<DecisionPanel />` | Reasoning + actions + advisory |
| `<AlertTicker />` | Latest 3 alerts inline |
| `<AlertCenter />` | Full alert list page |
| `<PublicAdvisory />` | Plain-English view |
| `<Disclaimer />` | Always-visible banner |
| `<StateBanner />` | LIVE / STALE / DEGRADED / MOCK |

---

## 15. API → UI Mapping

| Backend endpoint | UI consumer |
|---|---|
| `GET /api/version` | `<StatusHeader />` |
| `GET /api/wards` | `<RiskMap />`, drawer headers |
| `GET /api/wards/{id}/weather` | drawer Level 3 |
| `GET /api/wards/{id}/thermal` | drawer Level 2 |
| `GET /api/wards/{id}/risk` | drawer Level 1 + counters + alerts |
| `GET /api/forecast` | `<ForecastStrip />`, time-slider series |
| `GET /api/alerts` | `<AlertTicker />`, `<AlertCenter />` |
| `POST /api/agent/action-plan` | `<DecisionPanel />` |

Frontend must NOT compute WBGT, UTCI, or MRI.

---

## 16. Loading States

| State | UI |
|---|---|
| Loading | `Loading ward risk…` inline spinner |
| Live | `● LIVE · Updated 2 min ago` |
| Stale (>30 min) | `⚠ DATA MAY BE STALE · Last update 47 min ago` |
| Missing radiation | `WBGT degraded — radiation missing` (qualifier on the WBGT row) |
| Pipeline failure | `Risk unavailable. Showing last validated result.` |
| Mock mode | `DEMO MODE · Synthetic data` (banner, not red) |

---

## 17. Error States

- Network failure → toast + retry button; keep last successful render.
- 404 ward → toast; clear drawer.
- 500 backend → full-page banner; backend health link.

## 18. Empty States

- No wards → empty map with "Loading Hyderabad wards…".
- No alerts → "No active alerts. Monitor continues."
- No forecast → "Forecast unavailable — current risk only."

---

## 19. Accessibility (WCAG 2.2 AA targets)

- Color is never the only signal. Every band chip has a text label.
- All interactive elements keyboard-reachable.
- Map has a textual fallback: a sortable table of all wards.
- ARIA labels on map polygons.
- Contrast ≥ 4.5:1 for body text.
- Focus rings visible.

---

## 20. Responsive Behavior

| Breakpoint | Layout |
|---|---|
| ≥ 1280 px | 60/40 split, full drawer |
| 768–1279 px | Stacked: map → drawer → forecast → decision |
| < 768 px | Mobile list view: Risk → Location → Reason → Action (no map) |

---

## 21. Frontend Performance

- Initial paint < 2 s on M2 + local backend.
- Map polygons memoized by ward_id.
- Forecast strip paginates to top 5 days.
- No client-side scientific computation.
- Long lists (>50 wards) virtualized.

## 22. Frontend Security

- Read-only; no auth in MVP.
- No PII rendered.
- Banner: "Decision-support prototype. Not medical advice." visible on every page.
- `physics_version` and `risk_model_version` always rendered.

---

## 23. Demo Flow

1. Open `/` → choropleth colors obvious.
2. Click 7-Extreme counter → map filters to extreme wards.
3. Click Ward 42 → drawer populates with progressive disclosure.
4. Move time slider → +48h → map shifts colors.
5. Click Ask Decision Support → agent plan renders.
6. Open Alert Center → alert for Ward 42 appears.

---

## 24. Frontend Definition of Done

- [ ] Header shows LIVE indicator + timestamp
- [ ] 4 risk-band counters, clickable filters
- [ ] Choropleth map renders all wards
- [ ] Click ward → drawer populates
- [ ] Drawer shows Levels 1–4
- [ ] Why? card renders top factors
- [ ] Time slider re-colors map
- [ ] Forecast strip renders 5-day table + sparkline
- [ ] Decision panel renders agent plan
- [ ] Alert ticker shows latest 3
- [ ] Alert Center page works
- [ ] Public Advisory page works
- [ ] Disclaimer banner always visible
- [ ] StateBanner covers LIVE / STALE / DEGRADED / MOCK
- [ ] Mobile list view works
- [ ] No WBGT/UTCI computed in browser
- [ ] npm run build succeeds
- [ ] npm run lint passes

---

## 25. Closing Principle

> The frontend does not own scientific computation.
> It consumes ward_risk, ward_thermal, ward_forecast, and agent_plan —
> then visualizes the story: See → Understand → Predict → Explain → Act.

If the frontend engineer starts writing `calculateWBGT(...)`, **stop them**.