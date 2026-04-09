from __future__ import annotations

from pathlib import Path

import pandas as pd

from ml_core.training.baselines import BaselineTrainingConfig, train_baseline_pack


def test_train_baseline_pack_writes_summary_and_models(tmp_path: Path) -> None:
    rows = 60
    df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=rows, freq="5min"),
            "ticker": ["SBER"] * rows,
            "ret_1": [0.01 * ((i % 5) - 2) for i in range(rows)],
            "ret_3": [0.02 * ((i % 3) - 1) for i in range(rows)],
            "atr_14_pct": [0.01 + (i % 4) * 0.001 for i in range(rows)],
            "volume_rel_12": [0.1 * ((i % 4) - 1) for i in range(rows)],
            "label_class": (
                ["up_signal"] * 20 + ["down_signal"] * 20 + ["no_trade"] * 20
            ),
        }
    )

    train_df = pd.concat(
        [
            df.iloc[:15],
            df.iloc[20:35],
            df.iloc[40:55],
        ],
        ignore_index=True,
    )
    val_df = pd.concat(
        [
            df.iloc[15:20],
            df.iloc[35:40],
            df.iloc[55:60],
        ],
        ignore_index=True,
    )
    output_root = tmp_path / "baseline_pack"

    summary = train_baseline_pack(
        train_df,
        val_df,
        config=BaselineTrainingConfig(output_root=output_root),
    )

    assert "logreg_multiclass" in summary["models"]
    assert "rf_multiclass" in summary["models"]
    assert "extra_trees_multiclass" in summary["models"]
    assert "hgb_multiclass" in summary["models"]
    assert (output_root / "summary.json").exists()
    assert (output_root / "logreg_multiclass" / "model.pkl").exists()
    assert (output_root / "rf_multiclass" / "metrics.json").exists()
    assert (output_root / "extra_trees_multiclass" / "metrics.json").exists()
    assert (output_root / "hgb_multiclass" / "predictions.parquet").exists()
