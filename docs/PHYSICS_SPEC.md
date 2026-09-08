# Physics Specification

## 1. Scope

The physics domain computes WBGT and UTCI per ward per timestamp. It is fully deterministic given inputs.

## 2. Inputs

| Variable | Unit | Source |
|---|---|---|
| `temperature_2m` | °C | Open-Meteo |
| `relative_humidity_2m` | % | Open-Meteo |
| `wind_speed_10m` | m/s | Open-Meteo (converted) |
| `shortwave_radiation` | W/m² | Open-Meteo |
| `direct_radiation` | W/m² | Open-Meteo (optional) |
| `diffuse_radiation` | W/m² | Open-Meteo (optional) |
| `surface_pressure` | Pa | Open-Meteo (converted) |
| `latitude` | ° | ward centroid |
| `longitude` | ° | ward centroid |
| `timestamp` | ISO-8601 with tz | per forecast row |

## 3. WBGT (Liljegren Method)

The Liljegren formulation models heat exchange at a globe thermometer:

```
WBGT ≈ 0.567 * Tnwb + 0.393 * Tg + 0.1 * Tdb_shaded
```

Where:
- `Tnwb` = natural wet-bulb temperature (humidity + radiation effects)
- `Tg` = globe temperature (radiation + convection)
- `Tdb_shaded` = dry-bulb temperature in shade

To compute these, the engine:

1. Computes solar zenith and azimuth from (lat, lon, timestamp).
2. Splits shortwave radiation into direct and diffuse components.
3. Models globe temperature using iterative heat balance.
4. Models natural wet-bulb using psychrometric relations.
5. Composes final WBGT.

> This is an **implementation assumption and is not specified in the source document** in algorithmic detail; we adopt Liljegren because it is the documented reference for modern WBGT computation.

## 4. Simplified WBGT Fallback

If direct or diffuse radiation is missing:

```
WBGT_simplified = 0.567 * Tw + 0.393 * Tg_simple + 0.1 * Tdb
Tg_simple = Tdb + (shortwave_rad / 200)
```

The output includes `wbgt_quality = "degraded"`.

## 5. UTCI

The UTCI is the equivalent air temperature at reference conditions that produces the same physiological strain as the actual environment.

Reference operational approximation:

```
UTCI = T + offset(T, Tmrt-T, v, RH)
```

Where `offset` is a polynomial in (T, Tmrt-T, v, RH) calibrated on the UTCI regression (6th-order). For the MVP, an open-source reference implementation (pythermalcomfort or utci-fortran) is acceptable.

Inputs:
- T (°C)
- Tmrt (°C, computed from radiation + ambient)
- v (m/s)
- RH (%)

## 6. Mean Radiant Temperature (Tmrt)

Estimated from shortwave radiation:

```
Tmrt ≈ ((shortwave_rad / (5.4 * 10^-8 * σ)))^0.25 - 273.15  (rough)
```

More rigorous formulation uses angle factors, which the MVP may simplify.

## 7. Solar Geometry

Given (lat, lon, ts):

1. Day-of-year N.
2. Solar declination δ = 23.45 * sin(360*(284+N)/365).
3. Equation of time.
4. Local solar time.
5. Hour angle H.
6. Zenith Z = acos(sin(lat)·sin(δ) + cos(lat)·cos(δ)·cos(H)).

## 8. Internal API

```
POST /physics/wbgt
  req: { T, RH, v, p, direct_rad, diffuse_rad, lat, lon, ts }
  res: { wbgt_c, quality }

POST /physics/utci
  req: { T, RH, v, tmrt }
  res: { utci_c }

POST /physics/thermal-stress
  req: { T, RH, v, p, direct_rad, diffuse_rad, lat, lon, ts }
  res: { wbgt_c, utci_c, wbgt_quality, physics_version }
```

`physics_version` is stamped on every output.

## 9. Validation

- T in [-10, 55] °C, else reject.
- RH in [0, 100] %, else reject.
- v ≥ 0, else reject.
- Pressure in [80 000, 110 000] Pa, else reject.
- Solar zenith clamp to [0, 180].

## 10. Reference Test Cases

| Case | Inputs | Expected | Notes |
|---|---|---|---|
| Cool dry | T=20, RH=40, v=2, rad=200, p=101325 | WBGT ≈ 18–19 | sanity |
| Hot humid still | T=40, RH=70, v=0.5, rad=300, p=100500 | WBGT ≥ 32 | extreme |
| Hot dry breezy | T=40, RH=20, v=5, rad=300, p=100500 | WBGT ≤ 30 | less extreme |