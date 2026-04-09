from __future__ import annotations

from pathlib import Path
import json

import pandas as pd

from ml_core.training.calibration import CalibrationAuditConfig, run_saved_model_calibration_audit


def test_run_saved_model_calibration_audit_writes_summary_and_selects_calibrated_method(tmp_path: Path) -> None:
    model_dir = tmp_path / "model"
    output_root = tmp_path / "calibration_audit"
    model_dir.mkdir(parents=True, exist_ok=True)

    val_predictions = _sample_predictions_frame(start="2026-01-01T10:00:00Z")
    test_predictions = _sample_predictions_frame(start="2026-01-02T10:00:00Z")
    val_predictions.to_parquet(model_dir / "val_predictions.parquet", index=False)
    test_predictions.to_parquet(model_dir / "test_predictions.parquet", index=False)
    (model_dir / "metrics.json").write_text(
        json.dumps(
            {
                "model_name": "rf_multiclass",
                "selected_threshold": 0.55,
                "feature_columns": ["ret_1", "ret_3"],
            },
            indent=2,
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )

    summary = run_saved_model_calibration_audit(
        model_dir,
        config=CalibrationAuditConfig(
            output_root=output_root,
            min_fit_rows=6,
        ),
    )

    assert summary["research_candidate"]["method"] in {"platt", "isotonic"}
    assert summary["production_candidate"]["method"] in {"platt", "isotonic"}
    assert summary["production_candidate"]["validation_gate"]["passed"] is True
    assert (output_root / "summary.json").exists()
    assert (output_root / "report.md").exists()
    assert (output_root / "identity" / "metrics.json").exists()
    assert (output_root / "platt" / "val_reliability.json").exists()
    assert (output_root / "isotonic" / "test_reliability.json").exists()


def _sample_predictions_frame(*, start: str) -> pd.DataFrame:
    directional_patterns = [
        (0.85, "up_signal", "up_signal"),
        (0.85, "up_signal", "down_signal"),
        (0.85, "down_signal", "down_signal"),
        (0.85, "down_signal", "up_signal"),
        (0.65, "up_signal", "up_signal"),
        (0.65, "up_signal", "down_signal"),
        (0.65, "down_signal", "down_signal"),
        (0.65, "down_signal", "up_signal"),
        (0.55, "up_signal", "up_signal"),
        (0.55, "up_signal", "down_signal"),
        (0.55, "down_signal", "down_signal"),
        (0.55, "down_signal", "up_signal"),
    ]
    rows: list[dict] = []
    ts = pd.date_range(start, periods=len(directional_patterns) + 4, freq="5min")

    for idx, (prob, predicted_direction, label_class) in enumerate(directional_patterns):
        residual = 1.0 - prob - 0.05
        if predicted_direction == "up_signal":
            prob_up = prob
            prob_down = 0.05
        else:
            prob_up = 0.05
            prob_down = prob
        rows.append(
            {
                "timestamp": ts[idx],
                "ticker": "SBER",
                "label_class": label_class,
                "pred_class": predicted_direction,
                "prob_down_signal": prob_down,
                "prob_no_trade": residual,
                "prob_up_signal": prob_up,
            }
        )

    for idx in range(4):
        rows.append(
            {
                "timestamp": ts[len(directional_patterns) + idx],
                "ticker": "SBER",
                "label_class": "no_trade",
                "pred_class": "no_trade",
                "prob_down_signal": 0.15,
                "prob_no_trade": 0.70,
                "prob_up_signal": 0.15,
            }
        )

    return pd.DataFrame(rows)
