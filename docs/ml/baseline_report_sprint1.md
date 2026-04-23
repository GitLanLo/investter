# Sprint 1 ML Baseline Report

Обновлено: 2026-04-21.

## Dataset

- Source: Tinkoff Invest API через `configs/mvp_universe_v1.json`.
- Dataset version: `mvp_v1_20260409`.
- Timeframe: `5m`.
- Horizon: `12` bars.
- Purge gap: `12` bars.
- Assets: `GAZP`, `LKOH`, `MOEX`, `NVTK`, `SBER`.
- Factors: `usdrub`, `brent`, `rtsi`.
- Feature count: `57`.

| Split | Rows | Range UTC |
| --- | ---: | --- |
| train | 7247 | 2026-03-30 07:00 - 2026-04-05 20:15 |
| validation | 1535 | 2026-04-06 04:25 - 2026-04-07 13:00 |
| test | 1540 | 2026-04-07 14:05 - 2026-04-09 05:55 |

## Model Suite

Selection policy uses validation only. Test split is audit only.

| Model | Selected threshold | Production gate | Val macro F1 | Val actionable F1 | Val precision | Val coverage | Val ECE | Test macro F1 | Test actionable F1 | Test precision | Test coverage | Test ECE |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `logreg_multiclass` | 0.55 | fail | 0.3225 | 0.1901 | 0.2349 | 0.3661 | 0.4582 | 0.4575 | 0.2199 | 0.3478 | 0.1942 | 0.3015 |
| `rf_multiclass` | 0.55 | pass | 0.4165 | 0.1686 | 0.4244 | 0.1336 | 0.1801 | 0.4818 | 0.1255 | 0.6429 | 0.0455 | 0.0653 |
| `extra_trees_multiclass` | 0.55 | pass | 0.4086 | 0.1117 | 0.5000 | 0.0678 | 0.1655 | 0.5193 | 0.0851 | 0.5172 | 0.0377 | 0.0655 |
| `hgb_multiclass` | 0.55 | fail | 0.4564 | 0.2906 | 0.4106 | 0.2951 | 0.3284 | 0.4305 | 0.2462 | 0.3178 | 0.2656 | 0.4217 |

## Decisions

- Research candidate: `hgb_multiclass`, because it has the strongest validation actionable F1 and coverage.
- Production candidate: `rf_multiclass`, because it passes validation gates for precision, coverage and actionable ECE.
- Runtime recommendation: use `rf_multiclass` only for shadow/live monitoring. Test coverage is low (`0.0455`), so the policy is not ready for automated trading decisions.

## Follow-Up

- Add calibration research for HGB because it has better signal coverage but fails ECE gate.
- Add per-asset/per-regime thresholds; SBER has materially lower actionable coverage than the rest of the MVP universe.
- Extend history beyond the current MVP window before treating test metrics as stable.
- Formalize futures rollover policy for `brent` and `rtsi` aliases.
