"""Risk model: deterministic MRI in [0, 100] + band + top factors.

Pinned by risk_model_version. No stochastic inference.
"""
from __future__ import annotations

RISK_MODEL_VERSION = "risk-v1.0.0"


def _norm(x: float, lo: float, hi: float) -> float:
    if hi == lo:
        return 0.0
    return max(0.0, min(1.0, (x - lo) / (hi - lo)))


def vulnerability_index(demo: dict) -> float:
    return (
        0.40 * float(demo.get("elderly_ratio", 0.0))
        + 0.30 * float(demo.get("outdoor_worker_share", 0.0))
        + 0.20 * float(demo.get("informal_housing_share", 0.0))
        + 0.10 * min(float(demo.get("population", 0)) / 30000.0, 1.0)
    )


def compute_mri(features: dict) -> dict:
    peak_wbgt = float(features.get("peak_wbgt_next_72h", 0.0))
    utci_now = float(features.get("utci_now", 0.0))
    persistence = float(features.get("persistence_days_high", 0.0))
    vuln = float(features.get("vulnerability_index", 0.0))
    overnight_gap = float(features.get("overnight_recovery_gap_c", 0.0))

    mri = 100.0 * (
        0.35 * _norm(peak_wbgt, 25.0, 35.0)
        + 0.20 * _norm(utci_now, 26.0, 46.0)
        + 0.15 * _norm(persistence, 0.0, 5.0)
        + 0.20 * _norm(vuln, 0.0, 1.0)
        + 0.10 * _norm(-overnight_gap, -10.0, 0.0)
    )
    mri = round(mri, 1)
    return {
        "mri": mri,
        "band": _band(mri),
        "top_factors": [
            f"Peak WBGT next 72h = {peak_wbgt:.1f} °C",
            f"UTCI now = {utci_now:.1f} °C",
            f"Vulnerability index = {vuln:.2f}",
            f"Persistence = {persistence:.1f} days",
            f"Overnight recovery gap = {overnight_gap:.1f} °C",
        ],
        "risk_model_version": RISK_MODEL_VERSION,
    }


def _band(score: float) -> str:
    if score < 20:
        return "LOW"
    if score < 40:
        return "MODERATE"
    if score < 60:
        return "HIGH"
    if score < 80:
        return "VERY_HIGH"
    return "EXTREME"