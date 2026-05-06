# Sprint 8: Neural Sequence Model And Deployable Contract
Status: closed.

## Objective
Turn the Sprint 7 timeframe/horizon decision into the first neural sequence model that can be evaluated against the tabular baseline and exported through an explicit deployable model contract.

Sprint 8 starts from the Sprint 7 selected default:

*   timeframe: 1h
*   prediction horizon: 24 bars
*   ML universe: 25 data-backed TQBR instruments
*   baseline comparator: hgb_multiclass from dataset version sprint7_1h_h24_20260506
*   selection artifact: artifacts/research/sprint7/timeframe_horizon_matrix.json

## Non-Goals
*   Do not promote the neural model to production automatically.
*   Do not add live trading or order execution.
*   Do not build a full retraining scheduler.
*   Do not block Sprint 8 completion on beating the baseline; a well-documented failure is acceptable.

## Workstream 1: Data Readiness
**Goal:** make the sequence dataset large and clean enough for a first neural run.

**Tasks:**
*   Extend the historical backfill window beyond the short Sprint 7 window where Tinkoff API limits permit.
*   Keep YNDX disabled until the correct listed instrument is confirmed.
*   Produce a Sprint 8 data coverage report for the selected universe.
*   Decide whether usdrub stays in the default factor set or moves behind an ablation flag due to low coverage.
*   Keep exchange non-trading periods from being treated as critical gaps in Sprint 8 readiness decisions.

**Files:**
*   `configs/sprint7_universe_v1.json`
*   `ml-core/src/ml_core/ingest/tinkoff.py`
*   `ml-core/src/ml_core/ingest/providers.py`
*   `ml-core/src/ml_core/qa/coverage.py`
*   `ml-core/src/ml_core/qa/raw.py`
*   `docs/ml/sprint8_data_readiness.md`

**Expected artifacts:**
*   `artifacts/research/sprint8/data_coverage.json`
*   `docs/ml/sprint8_data_readiness.md`

**Done when:**
*   The selected universe has a documented row/window count.
*   Critical duplicate timestamp errors are absent.
*   Any remaining coverage warning is explicitly classified as acceptable, fixed, or excluded.

## Workstream 2: Sequence Dataset Builder
**Goal:** create leakage-safe neural input windows from the Sprint 7 selected tabular feature dataset.

**Tasks:**
*   Add a sequence dataset builder for windows 48 and 96.
*   Preserve train/validation/test split boundaries before windowing.
*   Enforce stable feature ordering.
*   Store label metadata for multiclass down, no_trade, up.
*   Add configurable stride, minimum coverage per window, and ticker filtering.
*   Write a manifest with input shape, window length, feature count, class mapping, source dataset version, timeframe, horizon, split counts and checksum metadata.

**Files:**
*   `ml-core/src/ml_core/datasets/sequences.py`
*   `ml-core/src/ml_core/cli.py`
*   `ml-core/src/ml_core/storage/layouts.py`
*   `ml-core/tests/test_sequence_dataset.py`
*   `docs/contracts/data_model_contracts_v1.md`

**Suggested storage layout:**
```
data/sequences/
  dataset_version=sprint8_1h_h24_<DATE>/
    window=48/
      manifest.json
      split=train/part-000.parquet
      split=val/part-000.parquet
      split=test/part-000.parquet
    window=96/
      manifest.json
      split=train/part-000.parquet
      split=val/part-000.parquet
      split=test/part-000.parquet
```

**Done when:**
*   `invest-ml build-sequence-dataset` can build both window sizes.
*   Unit tests cover window shape, split isolation, feature ordering and manifest contents.
*   The manifest is readable without loading the full dataset.

## Workstream 3: Neural Training Baseline
**Goal:** train a small sequence model that is simple enough to debug and strong enough to compare fairly.

**Tasks:**
*   Add a compact neural model family:
    *   first choice: GRU or temporal CNN;
    *   fallback: small MLP over flattened window only if sequence dependencies block progress.
*   Train multiclass down/no_trade/up.
*   Add class weighting or focal loss for class imbalance.
*   Use deterministic seeds and early stopping on validation actionable F1.
*   Save validation/test predictions and per-class probability outputs.
*   Keep training configurable enough to run a tiny smoke test in CI and a full run locally.

**Files:**
*   `ml-core/src/ml_core/training/sequence_models.py`
*   `ml-core/src/ml_core/training/neural.py`
*   `ml-core/src/ml_core/training/neural_calibration.py`
*   `ml-core/src/ml_core/cli.py`
*   `ml-core/tests/test_neural_training_smoke.py`

**Expected CLI:**
```bash
invest-ml train-neural-sequence \
  --sequence-root data/sequences \
  --dataset-version sprint8_1h_h24_<DATE> \
  --window 96 \
  --model-family gru \
  --output-root artifacts/models/sprint8_gru_1h_h24_w96
```

**Done when:**
*   A tiny fixture training run completes in tests.
*   A full local run writes model artifacts, predictions and metrics.
*   Training failures produce a useful error instead of a partial green run.

## Workstream 4: Evaluation And Calibration
**Goal:** decide whether the neural model improves the Sprint 7 baseline or clearly fails with evidence.

**Tasks:**
*   Compare neural metrics against the Sprint 7 hgb_multiclass baseline.
*   Report validation and test actionable F1, precision, coverage, ECE and per-ticker stability.
*   Run calibration audit on validation and test predictions.
*   Choose an operating threshold using validation data only.
*   Document whether the neural model is a candidate, rejected, or needs more data.

**Files:**
*   `ml-core/src/ml_core/pipelines/research.py`
*   `ml-core/src/ml_core/training/metrics.py`
*   `ml-core/src/ml_core/training/neural_calibration.py`
*   `backend/internal/service/research_artifacts.go`
*   `frontend/src/App.tsx`
*   `docs/ml/sprint8_neural_report.md`

**Expected artifacts:**
*   `artifacts/research/sprint8/neural_vs_baseline.json`
*   `artifacts/research/sprint8/calibration_report.json`
*   `docs/ml/sprint8_neural_report.md`

**Done when:**
*   The report includes a side-by-side neural vs baseline table.
*   A failure is explicit if the neural model underperforms.
*   The selected threshold and calibration method are traceable to validation metrics.

## Workstream 5: Deployable Model Contract
**Goal:** make the neural artifact consumable by runtime services without guessing shapes, feature order or normalization.

**Tasks:**
*   Extend the model manifest contract with neural sequence fields.
*   Export the model to ONNX if feasible.
*   If ONNX is not feasible in Sprint 8, export TorchScript or a local framework artifact and mark the runtime as unavailable in backend metadata.
*   Add golden parity tests between the training runtime and exported artifact where export support exists.
*   Store normalizer and feature order next to the model.
*   Add artifact checksums.

**Files:**
*   `docs/contracts/data_model_contracts_v1.md`
*   `ml-core/src/ml_core/modeling/manifest.py`
*   `ml-core/src/ml_core/modeling/export.py`
*   `ml-core/tests/test_model_manifest.py`
*   `ml-core/tests/test_model_export_parity.py`
*   `backend/internal/service/models.go`

**Required manifest fields:**
*   model_family
*   task
*   classes
*   input_timeframe
*   prediction_horizon_bars
*   input_window_bars
*   input_tensor_shape
*   feature_order
*   normalization
*   threshold
*   calibration
*   export_format
*   artifact_sha256
*   source_dataset_version

**Done when:**
*   The model directory is self-describing.
*   Backend can read the manifest without importing ML training code.
*   A contract test fails if feature order, tensor shape or class mapping is missing.

## Workstream 6: Backend Integration
**Goal:** let the operator and API see the Sprint 8 neural artifact status without pretending that unsupported runtime inference works.

**Tasks:**
*   Add model registry loading for neural manifests.
*   Expose neural model metadata through existing policy/research APIs or a small model metadata endpoint.
*   Report runtime availability explicitly:
    *   `available` when backend can execute the artifact;
    *   `metadata_only` when the model can be inspected but not executed;
    *   `unavailable` when required runtime dependencies are missing.
*   Keep tabular baseline inference working as the fallback path.
*   Add tests for manifest loading and unsupported runtime reporting.

**Files:**
*   `backend/internal/service/models.go`
*   `backend/internal/service/analysis.go`
*   `backend/internal/service/research_artifacts.go`
*   `backend/internal/httpserver/router.go`
*   `backend/internal/service/*_test.go`

**Done when:**
*   Backend does not silently substitute the baseline for a requested neural model.
*   API output clearly states whether neural runtime inference is enabled.
*   Existing Sprint 6 and Sprint 7 checks still pass.

## Workstream 7: Frontend Visibility
**Goal:** show Sprint 8 neural evidence and runtime status in the operator GUI.

**Tasks:**
*   Add a neural model card to the research/artifact area.
*   Display selected timeframe, horizon, window size, class mapping, threshold and runtime status.
*   Show neural vs baseline metrics.
*   Show calibration status and key blockers.
*   Keep empty states explicit when artifacts are missing.

**Files:**
*   `frontend/src/App.tsx`
*   `frontend/src/lib/types.ts`
*   `frontend/src/lib/api.ts`
*   `frontend/src/lib/mock.ts`

**Done when:**
*   The UI can distinguish metadata_only from executable runtime status.
*   Missing Sprint 8 artifacts do not break the page.
*   `npm run build` passes.

## Workstream 8: Automation And Smoke Checks
**Goal:** make Sprint 8 repeatable without requiring a full expensive training run in every check.

**Tasks:**
*   Add Make targets:
    *   `sprint8-data-check`
    *   `sprint8-sequence-smoke`
    *   `sprint8-train-smoke`
    *   `sprint8-check`
*   Validate manifest fields with `jq` or a small Python validation command.
*   Keep full training as an explicit local target, not part of default smoke.
*   Add backend/frontend smoke coverage for neural artifact visibility.

**Files:**
*   `Makefile`
*   `ml-core/tests/test_sequence_dataset.py`
*   `ml-core/tests/test_neural_training_smoke.py`
*   `backend/internal/service/*_test.go`
*   `frontend/src/App.tsx`

**Expected checks:**
```bash
make ml-test
make backend-test
make frontend-build
make sprint8-check
```

**Done when:**
*   `make sprint8-check` fails if the sequence manifest or neural model manifest is missing required fields.
*   The smoke path verifies artifact readability and API/UI contract, not just HTTP 200.

## Implementation Order
1.  Update contracts and storage layout for sequence datasets and neural manifests.
2.  Build sequence dataset generation with tests.
3.  Run data readiness and build window=48 and window=96 datasets.
4.  Add the smallest neural training path and smoke tests.
5.  Run full local neural training for both window sizes if data volume is sufficient.
6.  Compare against the Sprint 7 tabular baseline and write the neural report.
7.  Export the model and manifest, then add parity/golden checks where feasible.
8.  Wire backend artifact metadata and runtime status.
9.  Add frontend visibility for neural results.
10. Close Sprint 8 with `make sprint8-check` and a completion note.

## Exit Criteria
*   [x] Sequence dataset manifests exist for selected Sprint 7 default 1h/h24 and at least one window size.
*   [x] At least one neural sequence model is trained and evaluated.
*   [x] Neural results are compared against the Sprint 7 hgb_multiclass baseline.
*   [x] The conclusion is documented as candidate, rejected, or blocked by data volume.
*   [x] A deployable model artifact directory and manifest exist.
*   [x] Backend can read and report the neural manifest and runtime status.
*   [x] Frontend shows Sprint 8 neural evidence without breaking when artifacts are absent.
*   [x] `make sprint8-check` passes.

## Closure

*   Report: `docs/ml/baseline_report_sprint8.md`.
*   Data readiness note: `docs/ml/sprint8_data_readiness.md`.
*   Selected artifact: `artifacts/models/sprint8_gru_1h_h24_w96/model_manifest.json`.
*   Runtime status: `metadata_only`.
*   Test conclusion: GRU window 96 is competitive on validation but drops on test, so it should not be promoted automatically.
*   Sprint 9 starts with explicit model activation guards, promotion/rollback, notifications, and monitoring.

## Main Risks
*   The Sprint 7 data window is too short for neural training; Sprint 8 should prioritize longer backfill before spending time on model complexity.
*   ONNX runtime integration may be heavier than the sprint budget; metadata-only backend support is acceptable if the limitation is explicit.
*   Neural calibration may be worse than the tabular baseline; this is a valid Sprint 8 outcome if backed by metrics.
*   Factor coverage may distort sequence learning; factor ablation should remain available.
