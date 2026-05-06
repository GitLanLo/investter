# Sprint 7 Data Coverage

## Artifact

Coverage artifact: `artifacts/research/sprint7/data_coverage.json`

Generated on 2026-05-06 after the Sprint 7 Tinkoff backfill.

## Summary

- Total QA reports: 28
- Asset reports: 25
- Factor reports: 3
- Backfill window: 2026-04-10 to 2026-04-22
- Duplicate timestamp errors: none
- Report status: all reports are `warning`

The warnings are caused by gap detection over calendar time. The current QA check counts exchange closures, weekends and different trading sessions as gaps. This is expected for the current raw QA implementation and should be replaced with trading-schedule-aware coverage in a later sprint.

## Universe

The Sprint 7 data-backed universe contains 25 ML-enabled TQBR shares:

`AFLT`, `ALRS`, `CBOM`, `CHMF`, `GAZP`, `GMKN`, `IRAO`, `LKOH`, `MGNT`, `MOEX`, `MTSS`, `NLMK`, `NVTK`, `PHOR`, `PIKK`, `PLZL`, `ROSN`, `SBER`, `SBERP`, `SNGS`, `SNGSP`, `T`, `TATN`, `TRNFP`, `VTBR`.

`YNDX` is disabled in `configs/sprint7_universe_v1.json` because Tinkoff `FindInstrument` did not resolve it on `TQBR` during the backfill.

## Coverage Range

- Minimum coverage percentage: 7.26
- Maximum coverage percentage: 77.10
- Lowest coverage factor: `usdrub`
- Typical equity coverage range: roughly 63-77 percent under the current calendar-time denominator

## Follow-Up

- Make raw coverage trading-calendar-aware so normal non-trading periods do not degrade the coverage score.
- Extend the backfill horizon beyond this short Sprint 7 window before any production promotion.
- Re-test `usdrub` ingest and consider a factor ablation if currency coverage remains sparse.
