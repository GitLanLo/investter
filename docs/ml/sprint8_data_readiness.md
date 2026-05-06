# Sprint 8 Data Readiness Report

## Overview
This document evaluates the readiness of the dataset for training the first neural sequence model in Sprint 8. The selected dataset is based on the Sprint 7 1h/h24 configuration using the expanded 25-instrument TQBR universe.

## Data Coverage and Quality
A raw data coverage analysis was performed using the `invest-ml raw-coverage-report` command on the historical backfill.

### Findings:
1. **Instrument Coverage:** Most TQBR shares exhibit a nominal coverage percentage of around ~65-77% for the 5m timeframe. This reflects the standard market trading hours (approximately 14 hours per day) measured against a 24/7 theoretical grid. These exchange non-trading periods are not critical gaps and are fully acceptable for sequence modeling.
2. **Duplicate Timestamps:** The report confirms zero duplicate timestamps across all datasets, ensuring clean monotonic time series.
3. **YNDX Excluded:** As noted in the `sprint7_universe_v1.json` configuration, `YNDX` remains explicitly disabled (`ml_enabled: false`) due to API resolution issues during backfill. It is correctly excluded from the training universe.
4. **USDRUB Factor Sparsity:** The `usdrub` factor (USD000UTSTOM) shows extremely low coverage (7.26%) and a high number of gaps (154 gaps). This sparsity is significantly higher than other factors (like `brent` and `rtsi`, which maintain >50% coverage).

## Decisions
*   **USDRUB Factor Ablation:** Due to the unacceptably low coverage of the `usdrub` factor, it will be removed from the default mandatory factor set. It can be moved behind an ablation flag if needed in the future, but for the Sprint 8 baseline sequence model, it will be omitted to prevent the model from learning noise or suffering from excessive forward-filled stale data.
*   **Trading Gaps:** The detected gap counts on TQBR instruments accurately reflect overnight and weekend market closures. These will not be treated as readiness blockers. The sequence builder will need to process these gaps correctly (e.g., using `purge_gap_bars` or ignoring overnight jumps depending on the sequence construction logic).

## Conclusion
The raw data is sufficiently clean, monotonic, and volumous within the available Tinkoff backfill limits. The 25 selected TQBR instruments are approved for sequence window construction.