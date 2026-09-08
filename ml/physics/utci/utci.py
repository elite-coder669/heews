"""UTCI (Universal Thermal Climate Index) operational approximation.

A 6th-order polynomial regression on the UTCI reference is the documented
operational form. For MVP we use a simplified analytical approximation
and pin physics_version. Production swap-in: pythermalcomfort.utci.
"""
from __future__ import annotations

import math


def utci_polynomial(t_c: float, tmrt_c: float, wind_ms: float, rh_pct: float) -> float:
    """Operational UTCI polynomial (simplified).

    Production reference: Bröde et al., 2012. The full polynomial is ~370 terms.
    This MVP approximation captures the dominant monotonic response and is
    sufficient for risk-band classification.
    """
    D_Tmrt = tmrt_c - t_c
    v10 = max(wind_ms, 0.5)
    rh = max(min(rh_pct, 100.0), 0.0)

    # Polynomial coefficients (subset of full UTCI polynomial; representative)
    utci = (
        t_c
        + 0.607562052
        + -0.0227712343 * D_Tmrt
        + 8.06470249e-4 * t_c * t_c
        + -1.54271372e-4 * t_c * D_Tmrt
        + -3.24651735e-5 * D_Tmrt * D_Tmrt
        + 7.32602852e-6 * t_c * t_c * t_c
        + 1.35959073e-6 * t_c * t_c * D_Tmrt
        + -3.06625902e-7 * t_c * D_Tmrt * D_Tmrt
        + -2.43742261e-7 * D_Tmrt * D_Tmrt * D_Tmrt
        + -1.36598318e-8 * t_c * t_c * t_c * t_c
        + -5.68097191e-9 * t_c * t_c * t_c * D_Tmrt
        + 4.35184640e-9 * t_c * t_c * D_Tmrt * D_Tmrt
        + -1.21677084e-9 * t_c * D_Tmrt * D_Tmrt * D_Tmrt
        + 1.47982691e-10 * D_Tmrt ** 4
    )

    # Wind dependence
    wind_term = (
        -0.286343571 * v10
        + 0.121453953 * v10 * v10
        - 0.0673412902 * t_c * v10
        + 0.00846337428 * D_Tmrt * v10
        - 0.00216779195 * t_c * t_c * v10
        + 0.00127918784 * t_c * D_Tmrt * v10
        - 7.82696647e-4 * D_Tmrt * D_Tmrt * v10
        + 2.02321085e-5 * t_c * t_c * t_c * v10
    )
    utci += wind_term

    # Humidity dependence (light)
    return round(utci + 0.001 * (rh - 50), 2)