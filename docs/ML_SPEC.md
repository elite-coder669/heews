# ML / Risk Specification

## 1. Honest Framing

The team does not have access to validated mortality ground truth for Hyderabad. The MRI is therefore a **physiologically grounded, vulnerability-aware heat-health risk indicator**, NOT a clinical mortality prediction.

We explicitly state this in:
- The UI disclaimer banner.
- API responses (`risk_model_version` always present).
- The PDF documentation.

## 2. Inputs

| Group | Features |
|---|---|
| Thermal (now) | wbgt_now, utci_now |
| Thermal (forecast) | wbgt_peak_next_72h, utci_peak_next_72h |
| Temporal | persistence_days_high, overnight_recovery_gap_c, mean_wbgt_next_72h |
| Vulnerability | elderly_ratio, outdoor_worker_share, informal_housing_share, population_density |
| Contextual (optional) | distance_to_nearest_cooling_center_km, hospital_beds_per_1000 |

## 3. MRI Formula (MVP, transparent)

```
norm(x, lo, hi) = clip((x - lo) / (hi - lo), 0, 1)

MRI = 100 * (
    0.35 * norm(peak_wbgt_next_72h, 25, 35) +
    0.20 * norm(utci_now, 26, 46) +
    0.15 * norm(persistence_days_high, 0, 5) +
    0.20 * norm(vulnerability_index, 0, 1) +
    0.10 * norm(-overnight_recovery_gap_c, -10, 0)
)

vulnerability_index = (
    0.40 * elderly_ratio +
    0.30 * outdoor_worker_share +
    0.20 * informal_housing_share +
    0.10 * min(population_density / 30000, 1)
)
```

The piecewise norms are documented and pinned per `risk_model_version`.

## 4. Risk Bands

| Band | Score |
|---|---|
| LOW | 0–19 |
| MODERATE | 20–39 |
| HIGH | 40–59 |
| VERY_HIGH | 60–79 |
| EXTREME | 80–100 |

## 5. Top Contributing Factors

Reported to the agent as 3–5 bullet strings:

```
"Peak WBGT next 72h = X.X °C"
"UTCI now = Y.Y °C"
"Outdoor worker share = Z%"
"Persistence = N days"
"Overnight recovery gap = X.X °C"
```

## 6. Determinism

Given identical inputs and `risk_model_version`, MRI must be identical. No stochastic inference in MVP.

## 7. Validation

- All scores clipped to [0, 100].
- Bands computed from final score.
- Top-factors sorted by absolute contribution magnitude.

## 8. Future Evolution

When labeled mortality data become available, swap the transparent weighted score for a calibrated supervised model. Until then, we publish the formula openly and document the assumption.