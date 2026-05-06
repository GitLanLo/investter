# Sprint 7: Timeframe And Horizon Decision

## Objective

Select the default timeframe and prediction horizon for Sprint 8 neural modeling using a comparable baseline grid, not intuition.

## Inputs

- Matrix artifact: `artifacts/research/sprint7/timeframe_horizon_matrix.json`
- Coverage artifact: `artifacts/research/sprint7/data_coverage.json`
- Universe config: `configs/sprint7_universe_v1.json`
- Run date: 2026-05-06
- Data window used for the Sprint 7 expansion backfill: 2026-04-10 to 2026-04-22
- Data-backed ML universe: 25 TQBR shares
- Factors: `usdrub`, `brent`, `rtsi`

`YNDX` is left in the config as disabled because Tinkoff `FindInstrument` did not resolve `YNDX` on `TQBR` during the Sprint 7 backfill. It is excluded from ML training until the correct listed ticker is confirmed.

## Method

The grid ran 9 comparable baseline research jobs:

- Timeframes: `5m`, `15m`, `1h`
- Horizons: `6`, `12`, `24` bars
- Selection basis: validation metrics only
- Test metrics: audit only
- Ranking order: validation actionable F1, lower validation ECE, validation precision, validation coverage, lower per-ticker F1 dispersion

## Matrix

| Timeframe | Horizon | Rows | Val F1 | Val precision | Val coverage | Val ECE | Test F1 | Test precision | Test coverage | Test ECE |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 5m | 6 | 137630 | 0.142 | 0.358 | 0.070 | 0.285 | 0.050 | 0.348 | 0.007 | 0.365 |
| 5m | 12 | 137480 | 0.231 | 0.451 | 0.156 | 0.218 | 0.072 | 0.252 | 0.031 | 0.413 |
| 5m | 24 | 137180 | 0.332 | 0.493 | 0.321 | 0.205 | 0.127 | 0.393 | 0.064 | 0.318 |
| 15m | 6 | 48175 | 0.250 | 0.435 | 0.206 | 0.234 | 0.094 | 0.395 | 0.034 | 0.254 |
| 15m | 12 | 48025 | 0.343 | 0.500 | 0.358 | 0.191 | 0.257 | 0.605 | 0.107 | 0.072 |
| 15m | 24 | 47725 | 0.428 | 0.516 | 0.588 | 0.206 | 0.399 | 0.637 | 0.256 | 0.085 |
| 1h | 6 | 12381 | 0.335 | 0.473 | 0.380 | 0.240 | 0.319 | 0.508 | 0.236 | 0.163 |
| 1h | 12 | 12227 | 0.457 | 0.494 | 0.718 | 0.252 | 0.383 | 0.559 | 0.381 | 0.154 |
| 1h | 24 | 11919 | 0.494 | 0.508 | 0.832 | 0.256 | 0.459 | 0.606 | 0.535 | 0.109 |

## Decision

Selected default: `1h`, horizon `24` bars.

Rationale:

- It has the highest validation actionable F1: `0.494`.
- It has the highest validation signal coverage: `0.832`.
- It keeps validation precision above `0.50`.
- Test audit remains directionally consistent: actionable F1 `0.459`, precision `0.606`, coverage `0.535`.
- Per-ticker validation F1 dispersion is acceptable for this grid: std `0.077`, min `0.358`, max `0.659`.

## Implication For Sprint 8

Sprint 8 neural sequence work should start with:

- timeframe: `1h`
- horizon bars: `24`
- model family: sequence model using the expanded 25-instrument Sprint 7 dataset
- baseline comparator: `hgb_multiclass` from `sprint7_1h_h24_20260506`

## Caveats

- The expansion backfill window is short: 2026-04-10 to 2026-04-22. The result is suitable for selecting the next research default, but longer history is still required before production promotion.
- All raw QA reports are `warning` because the current gap detector treats exchange non-trading periods as gaps. There are no duplicate timestamp errors in the coverage artifact.
- `usdrub` coverage is low relative to equity/futures factors; Sprint 8 should either improve the currency factor ingest path or test a no-currency ablation before final promotion.
