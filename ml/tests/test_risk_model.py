"""Risk model tests."""
from ml.risk.risk_model import compute_mri, vulnerability_index


def test_mri_extreme():
    r = compute_mri({
        "peak_wbgt_next_72h": 35.0,
        "utci_now": 46.0,
        "persistence_days_high": 5.0,
        "vulnerability_index": 1.0,
        "overnight_recovery_gap_c": -10.0,
    })
    assert r["mri"] >= 80
    assert r["band"] == "EXTREME"


def test_mri_low():
    r = compute_mri({
        "peak_wbgt_next_72h": 20.0,
        "utci_now": 20.0,
        "persistence_days_high": 0.0,
        "vulnerability_index": 0.0,
        "overnight_recovery_gap_c": 5.0,
    })
    assert r["mri"] < 20
    assert r["band"] == "LOW"


def test_vulnerability_index_bounds():
    v = vulnerability_index({"elderly_ratio": 1, "outdoor_worker_share": 1, "informal_housing_share": 1, "population": 100000})
    assert 0.0 <= v <= 1.0