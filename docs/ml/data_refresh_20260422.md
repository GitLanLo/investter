# Data Refresh 2026-04-22

Обновлено: 2026-04-22.

## Source

- Provider: Tinkoff Invest API REST.
- Universe: `configs/mvp_universe_v1.json`.
- Sync mode: incremental with `12` overlap bars.
- Requested range: `2026-02-01T07:00:00Z` to `2026-04-22T15:00:00Z`.
- Latest persisted candle timestamp: `2026-04-22T07:35:00Z`.

## Ingest Summary

| Group | Rows | Items |
| --- | ---: | --- |
| assets | 13051 | `SBER`, `GAZP`, `LKOH`, `MOEX`, `NVTK` |
| factors | 3944 | `usdrub`, `brent`, `rtsi` |

QA reports are warning-level because session/weekend gaps are expected in 5-minute market data. Duplicate timestamps were `0` in the incremental run.

## Dataset

- Dataset version: `mvp_live_20260422`.
- Feature count: `57`.

| Split | Rows | Range UTC |
| --- | ---: | --- |
| train | 59145 | 2026-02-01 07:00 - 2026-03-29 04:50 |
| validation | 12771 | 2026-03-29 05:55 - 2026-04-10 10:40 |
| test | 12641 | 2026-04-10 11:45 - 2026-04-22 06:35 |

## Baseline Research

- Research artifacts: `artifacts/research/baseline_mvp_live_20260422`.
- Research candidate: `hgb_multiclass`, threshold `0.55`.
- Production-gated baseline: `logreg_multiclass`, threshold `0.55`.

| Candidate | Val actionable F1 | Val precision | Val coverage | Val ECE | Test actionable F1 | Test precision | Test coverage | Test ECE |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `hgb_multiclass` research | 0.2633 | 0.3587 | 0.2283 | 0.3045 | 0.2567 | 0.4002 | 0.1751 | 0.2812 |
| `logreg_multiclass` production-gated | 0.1435 | 0.4078 | 0.0841 | 0.2067 | 0.1073 | 0.3954 | 0.0582 | 0.2208 |

## Calibration Audit

- Calibration artifacts: `artifacts/research/calibration_audit_logreg_mvp_live_20260422`.
- Calibrated model: `logreg_multiclass`.
- Production calibration method: `identity`.
- Production threshold: `0.30`.
- Validation gate: passed.

| Split | Actionable F1 | Precision | Coverage | Actionable ECE |
| --- | ---: | ---: | ---: | ---: |
| validation | 0.3430 | 0.3082 | 0.4939 | 0.1647 |
| test | 0.3095 | 0.2885 | 0.4291 | 0.1763 |

## Decision

The refreshed `mvp_live_20260422` policy is now a Sprint 3 `production_candidate` after calibration audit. It is suitable for shadow/live validation, but not for automated trading execution.
