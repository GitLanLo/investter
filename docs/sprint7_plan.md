# Sprint 7: Dataset Expansion & Research Grid

## Objective
Expand the trading universe to 20-50 liquid instruments and execute a comprehensive research grid across multiple timeframes (5m, 15m, 1h) and horizons (6, 12, 24). Analyze the results to determine the optimal default timeframe and horizon for Sprint 8 production deployment.

## Scope

### 1. Universe Expansion (20-50 Instruments)
- Create `configs/sprint7_universe_v1.json` based on the MVP universe, expanding it to include 20-50 liquid TQBR shares.
- Enhance `ml-core/src/ml_core/ingest/universe.py` to support new specification fields: `name`, `sector`, `liquidity_tier`, `ml_enabled`, `training_exclusion_reason`, and `notes`.

### 2. Tinkoff Ingest Improvements
- Support `TINKOFF_CA_CERT_FILE` in `ml-core/src/ml_core/ingest/tinkoff.py` (matching backend behavior).
- Implement robust retry/backoff mechanisms and record partial failures gracefully in the universe summary.
- Update `tinkoff-sync-universe` to handle multiple timeframes from the configuration or accept an explicit `--timeframe` override.

### 3. Storage Layout for Multi-Timeframe/Horizon
- Resolve collisions in the feature store layout. Currently, it saves to `features/schema=<schema>/ticker=<ticker>`, which mixes different timeframes and horizons.
- Update `ml-core/src/ml_core/storage/layouts.py` to include `timeframe` and `horizon_bars` dimensions in the path, or clearly separate pure features from labeled features.
- Update `ml-core/src/ml_core/pipelines/materialize.py` to pass these dimensions during read/write operations.
- Fix `run-research-pipeline` in `ml-core/src/ml_core/cli.py` to correctly initialize `FeatureBuildConfig(timeframe=args.timeframe)`.
- Update API contracts in `docs/contracts/data_model_contracts_v1.md`.

### 4. Data Quality & Coverage Reporting
- Create `ml-core/src/ml_core/qa/coverage.py`.
- Expand data QA checks in `ml-core/src/ml_core/qa/raw.py` to calculate: coverage percentage, duplicate count, latest candle timestamp, largest gap, and overall status per ticker/timeframe.
- Add a new CLI command: `raw-coverage-report`.
- Generate artifact: `artifacts/research/sprint7/data_coverage.json`.
- Document findings: `docs/ml/sprint7_data_coverage.md`.

### 5. Research Grid Execution
- Implement `ml-core/src/ml_core/pipelines/research_grid.py`.
- Add CLI command: `run-research-grid`.
- Execute combinations: `timeframes=[5m, 15m, 1h]` x `horizons=[6, 12, 24]`.
- For each run, generate a dataset version formatted as `sprint7_<tf>_h<horizon>_<date>`.
- Utilize existing research and training pipelines.

### 6. Research Decision & Reporting
- Generate artifact: `artifacts/research/sprint7/timeframe_horizon_matrix.json`.
- Create decision document: `docs/ml/sprint7_timeframe_horizon_decision.md`.
- Compare configurations based on: actionable F1, precision, signal coverage, calibration ECE, per-instrument stability, and dataset size.
- Decision criteria: Rank configurations using validation metrics; reserve test metrics strictly for auditing (consistent with `ml-core/src/ml_core/training/research.py`).

### 7. UI & Backend Integration
- Update `backend/internal/service/research_artifacts.go` to locate and parse Sprint 7 matrix and decision artifacts.
- Extend backend DTOs if necessary.
- Add a "Sprint 7 research decision" summary block to the frontend (`frontend/src/App.tsx`).
- Update frontend data models (`types.ts`, `api.ts`, `mock.ts`).

### 8. Automation & Checks
- Update `Makefile` with a new target: `sprint7-check` (runs `ml-test`, `backend-test`, `frontend-build`, and `sprint7-smoke`).
- Add `sprint7-smoke` to verify the presence and structure of `/ml/research/overview`, `/ml/research/documents`, and decision artifacts.
- Introduce a separate command for the heavy grid execution: `sprint7-research-run`.

## Non-Goals
- Neural network implementation is deferred.
- Futures will not be used as training assets in this sprint (requires rollover policy implementation).

## Exit Criteria
- [ ] `configs/sprint7_universe_v1.json` contains 20+ ML-enabled instruments.
- [ ] Coverage reports exist for every ticker and timeframe.
- [ ] 9 comparable dataset/research runs are completed.
- [ ] `docs/ml/sprint7_timeframe_horizon_decision.md` justifies the chosen default.
- [ ] `make sprint7-check` passes successfully.
- [ ] Raw data and large artifacts remain excluded from source control.