# Sprint 8: Neural Sequence Model Report (Refined)

## Objective
The goal of Sprint 8 was to implement the first neural sequence model (GRU/CNN) for the selected timeframe (1h) and horizon (24h), and to establish the deployment contract between `ml-core` and `backend`.

## Refinements and Leakage Fix
- **Leakage Fix:** Refined `build-sequence-dataset` to strictly use `feature_columns` from the source dataset manifest. This ensures non-feature columns like `label_class` or improperly joined cross-asset features are not leaked into the sequence.
- **Fair Evaluation:** Expanded the backfill coverage to ensure the `test` split has enough data for windowed sequences (window size 48 and 96).
- **Data Coverage:**
    - Train split: 25 assets, ~5000 windows.
    - Val split: 25 assets, ~3000 windows.
    - Test split: 25 assets, ~800 windows (expanded from 0).

## Model Architecture
- **Primary Model:** 2-layer GRU with 64 hidden units.
- **Normalization:** Standard scaling per feature across the entire sequence, fitted on the `train` split only.
- **Export Format:** TorchScript (`model.pt`).

## Refined Training Results
Training was performed for 10 epochs with a batch size of 32.

| Window Size | Val Actionable F1 | Val Actionable Precision | Test Actionable F1 | Test Actionable Precision |
|-------------|-------------------|--------------------------|-------------------|---------------------------|
| 48          | 0.3442            | 0.2516                   | 0.3500            | 0.2509                    |
| 96          | **0.5039**        | **0.5182**               | 0.2694            | 0.2735                    |

### Comparison with Tabular Baseline
Sprint 7 Tabular Baseline (1h/h24):
- **Val Actionable F1:** 0.4938
- **Test Actionable F1:** 0.4586

**Conclusion:**
- The GRU model with **window=96** shows competitive performance on the **validation** set (0.5039), slightly outperforming the tabular baseline (0.4938).
- However, there is a significant **drop on the test set** (0.2694), indicating potential overfitting to the validation period or sensitivity to the specific market regime in the test window.
- The model with **window=48** shows consistent but lower performance (~0.35 F1).

## Deployment Readiness
- **Contract:** Updated Go backend to support `ModelManifest` with neural fields.
- **Visibility:** Added "Model Card" to the operator frontend to audit deployment metadata and runtime status.
- **Artifacts:**
    - `model.pt` (TorchScript)
    - `scaler.joblib` (StandardScaler)
    - `model_manifest.json`
    - `predictions_val.parquet` / `predictions_test.parquet`

## Registry Integration
The model `sprint8_gru_1h_h24_w96` is registered in the backend and its metadata is accessible via `/ml/models/{version}`. Runtime status is currently `metadata_only` as native PyTorch inference in Go is scheduled for Sprint 9.

## Blocker: Data Volume vs Complexity
While sequence models show promise, the 1h/h24 dataset remains relatively small for deep learning. The performance gap between Val and Test suggests that either more data (longer backfill) or more aggressive regularization/feature selection is required.
