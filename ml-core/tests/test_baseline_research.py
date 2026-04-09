from __future__ import annotations

from pathlib import Path

import pandas as pd

from ml_core.storage.layouts import write_dataset_split_frame
from ml_core.training.research import (
    BaselineResearchConfig,
    WalkForwardConfig,
    run_baseline_research,
    run_walk_forward_research,
)


def test_run_baseline_research_writes_summary_and_report(tmp_path: Path) -> None:
    dataset_root = tmp_path / "dataset_version=v1"
    output_root = tmp_path / "research"

    train_df = _sample_split_df(rows=45, start="2026-01-01T10:00:00Z")
    val_df = _sample_split_df(rows=15, start="2026-01-02T10:00:00Z")
    test_df = _sample_split_df(rows=15, start="2026-01-03T10:00:00Z")

    write_dataset_split_frame(train_df, dataset_root=dataset_root, split_name="train")
    write_dataset_split_frame(val_df, dataset_root=dataset_root, split_name="val")
    write_dataset_split_frame(test_df, dataset_root=dataset_root, split_name="test")

    summary = run_baseline_research(
        dataset_root,
        config=BaselineResearchConfig(output_root=output_root),
    )

    assert summary["best_model"] in {"logreg_multiclass", "rf_multiclass"}
    assert (output_root / "summary.json").exists()
    assert (output_root / "report.md").exists()
    assert (output_root / "logreg_multiclass" / "val_predictions.parquet").exists()
    assert (output_root / "rf_multiclass" / "feature_importance.csv").exists()


def test_run_walk_forward_research_writes_fold_reports(tmp_path: Path) -> None:
    dataset_root = tmp_path / "dataset_version=v1"
    output_root = tmp_path / "walk_forward"

    train_df = _sample_split_df(rows=600, start="2026-01-01T10:00:00Z")
    val_df = _sample_split_df(rows=240, start="2026-01-04T12:00:00Z")
    test_df = _sample_split_df(rows=120, start="2026-01-06T12:00:00Z")

    write_dataset_split_frame(train_df, dataset_root=dataset_root, split_name="train")
    write_dataset_split_frame(val_df, dataset_root=dataset_root, split_name="val")
    write_dataset_split_frame(test_df, dataset_root=dataset_root, split_name="test")

    summary = run_walk_forward_research(
        dataset_root,
        config=WalkForwardConfig(
            output_root=output_root,
            min_train_timestamps=180,
            min_validation_timestamps=60,
            max_folds=3,
            step_ratio=0.1,
        ),
    )

    assert summary["best_model"] in {"logreg_multiclass", "rf_multiclass"}
    assert summary["fold_count"] >= 1
    assert (output_root / "summary.json").exists()
    assert (output_root / "report.md").exists()
    assert (output_root / "logreg_multiclass" / "walk_forward_predictions.parquet").exists()
    assert (output_root / "rf_multiclass" / "walk_forward_metrics.json").exists()


def _sample_split_df(*, rows: int, start: str) -> pd.DataFrame:
    base = pd.date_range(start, periods=rows, freq="5min")
    labels = (["up_signal"] * (rows // 3)) + (["down_signal"] * (rows // 3)) + (["no_trade"] * (rows - 2 * (rows // 3)))
    return pd.DataFrame(
        {
            "timestamp": base,
            "ticker": ["SBER"] * rows,
            "ret_1": [0.01 * ((i % 5) - 2) for i in range(rows)],
            "ret_3": [0.02 * ((i % 3) - 1) for i in range(rows)],
            "atr_14_pct": [0.01 + (i % 4) * 0.001 for i in range(rows)],
            "volume_rel_12": [0.1 * ((i % 4) - 1) for i in range(rows)],
            "feature_schema_version": ["feature_v1"] * rows,
            "label_class": labels,
            "label_up": [1 if label == "up_signal" else 0 for label in labels],
            "label_down": [1 if label == "down_signal" else 0 for label in labels],
            "label_no_trade": [1 if label == "no_trade" else 0 for label in labels],
        }
    )
