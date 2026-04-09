from __future__ import annotations

from pathlib import Path

import pandas as pd

from ml_core.storage.layouts import write_dataset_split_frame
from ml_core.training.research import (
    AblationResearchConfig,
    BaselineResearchConfig,
    WalkForwardConfig,
    _select_production_model_candidate,
    run_ablation_research,
    run_baseline_research,
    run_walk_forward_research,
)

MODEL_NAMES = {
    "logreg_multiclass",
    "rf_multiclass",
    "extra_trees_multiclass",
    "hgb_multiclass",
}


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

    assert summary["best_model"] in MODEL_NAMES
    assert summary["research_candidate"]["model_name"] == summary["best_model"]
    assert "selection_mode" in summary["production_candidate"]
    assert "validation_gate" in summary["production_candidate"]
    assert (
        summary["production_candidate"]["validation"]["actionable_expected_calibration_error"] is None
        or summary["production_candidate"]["validation"]["actionable_expected_calibration_error"] >= 0.0
    )
    assert (output_root / "summary.json").exists()
    assert (output_root / "report.md").exists()
    assert (output_root / "logreg_multiclass" / "val_predictions.parquet").exists()
    assert (output_root / "logreg_multiclass" / "val_reliability.json").exists()
    assert (output_root / "rf_multiclass" / "feature_importance.csv").exists()
    assert (output_root / "extra_trees_multiclass" / "metrics.json").exists()
    assert (output_root / "hgb_multiclass" / "test_reliability.json").exists()


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

    assert summary["best_model"] in MODEL_NAMES
    assert summary["research_candidate"]["model_name"] == summary["best_model"]
    assert "validation_gate" in summary["production_candidate"]
    assert "actionable_expected_calibration_error" in summary["models"][summary["best_model"]]["walk_forward"]["probability_metrics_mean"]
    assert summary["fold_count"] >= 1
    assert (output_root / "summary.json").exists()
    assert (output_root / "report.md").exists()
    assert (output_root / "logreg_multiclass" / "walk_forward_predictions.parquet").exists()
    assert (output_root / "rf_multiclass" / "walk_forward_metrics.json").exists()
    assert (output_root / "extra_trees_multiclass" / "walk_forward_metrics.json").exists()
    assert (output_root / "hgb_multiclass" / "walk_forward_predictions.parquet").exists()


def test_run_ablation_research_writes_scenario_reports(tmp_path: Path) -> None:
    dataset_root = tmp_path / "dataset_version=v1"
    output_root = tmp_path / "ablation"

    train_df = _sample_split_df(rows=180, start="2026-01-01T10:00:00Z")
    val_df = _sample_split_df(rows=90, start="2026-01-02T10:00:00Z")
    test_df = _sample_split_df(rows=90, start="2026-01-03T10:00:00Z")

    write_dataset_split_frame(train_df, dataset_root=dataset_root, split_name="train")
    write_dataset_split_frame(val_df, dataset_root=dataset_root, split_name="val")
    write_dataset_split_frame(test_df, dataset_root=dataset_root, split_name="test")

    summary = run_ablation_research(
        dataset_root,
        config=AblationResearchConfig(output_root=output_root),
    )

    assert summary["best_scenario"] in {"full", "no_cross_asset", "no_regime", "core_price_volume_only"}
    assert summary["research_candidate"]["scenario_name"] == summary["best_scenario"]
    assert "validation_gate" in summary["production_candidate"]
    assert summary["production_candidate"]["model_name"] == summary["scenarios"][summary["production_candidate"]["scenario_name"]]["best_model"]
    assert (output_root / "summary.json").exists()
    assert (output_root / "report.md").exists()
    assert (output_root / "full" / "summary.json").exists()
    assert (output_root / "no_cross_asset" / "rf_multiclass" / "metrics.json").exists()
    assert (output_root / "no_cross_asset" / "rf_multiclass" / "val_reliability.json").exists()


def test_select_production_model_candidate_prefers_gated_model() -> None:
    results = {
        "research_best": {
            "validation": {
                "metrics": {"macro_f1": 0.35, "balanced_accuracy": 0.36},
                "signal_metrics": {
                    "actionable_f1": 0.20,
                    "precision_actionable_signal": 0.22,
                    "signal_coverage": 0.32,
                },
                "probability_metrics": {"actionable_expected_calibration_error": 0.41},
            }
        },
        "deployable": {
            "validation": {
                "metrics": {"macro_f1": 0.33, "balanced_accuracy": 0.34},
                "signal_metrics": {
                    "actionable_f1": 0.18,
                    "precision_actionable_signal": 0.37,
                    "signal_coverage": 0.11,
                },
                "probability_metrics": {"actionable_expected_calibration_error": 0.19},
            }
        },
    }

    candidate_name, gate = _select_production_model_candidate(results, fallback_name="research_best")

    assert candidate_name == "deployable"
    assert gate["passed"] is True


def test_select_production_model_candidate_falls_back_to_research_best() -> None:
    results = {
        "research_best": {
            "validation": {
                "metrics": {"macro_f1": 0.35, "balanced_accuracy": 0.36},
                "signal_metrics": {
                    "actionable_f1": 0.20,
                    "precision_actionable_signal": 0.22,
                    "signal_coverage": 0.04,
                },
                "probability_metrics": {"actionable_expected_calibration_error": 0.41},
            }
        },
        "also_blocked": {
            "validation": {
                "metrics": {"macro_f1": 0.33, "balanced_accuracy": 0.34},
                "signal_metrics": {
                    "actionable_f1": 0.18,
                    "precision_actionable_signal": 0.28,
                    "signal_coverage": 0.11,
                },
                "probability_metrics": {"actionable_expected_calibration_error": 0.31},
            }
        },
    }

    candidate_name, gate = _select_production_model_candidate(results, fallback_name="research_best")

    assert candidate_name == "research_best"
    assert gate["passed"] is False


def _sample_split_df(*, rows: int, start: str) -> pd.DataFrame:
    base = pd.date_range(start, periods=rows, freq="5min")
    labels = (["up_signal"] * (rows // 3)) + (["down_signal"] * (rows // 3)) + (["no_trade"] * (rows - 2 * (rows // 3)))
    return pd.DataFrame(
        {
            "timestamp": base,
            "ticker": ["SBER"] * rows,
            "ret_1": [0.01 * ((i % 5) - 2) for i in range(rows)],
            "ret_3": [0.02 * ((i % 3) - 1) for i in range(rows)],
            "usdrub_close": [92.0 + (i % 8) * 0.05 for i in range(rows)],
            "usdrub_ret_1": [0.001 * ((i % 4) - 2) for i in range(rows)],
            "asset_vs_brent_rel_strength_12": [0.002 * ((i % 6) - 3) for i in range(rows)],
            "vol_regime_flag": [i % 2 for i in range(rows)],
            "market_stress_proxy": [0.01 + (i % 5) * 0.002 for i in range(rows)],
            "atr_14_pct": [0.01 + (i % 4) * 0.001 for i in range(rows)],
            "volume_rel_12": [0.1 * ((i % 4) - 1) for i in range(rows)],
            "feature_schema_version": ["feature_v1"] * rows,
            "label_class": labels,
            "label_up": [1 if label == "up_signal" else 0 for label in labels],
            "label_down": [1 if label == "down_signal" else 0 for label in labels],
            "label_no_trade": [1 if label == "no_trade" else 0 for label in labels],
        }
    )
