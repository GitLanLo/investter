import json
from pathlib import Path

import numpy as np
import pandas as pd
import pytest

from ml_core.datasets.sequences import SequenceDatasetConfig, build_sequence_dataset

def test_build_sequence_dataset(tmp_path: Path):
    data_root = tmp_path / "data"
    output_root = tmp_path / "sequences"

    source_ds_version = "source_v1"
    source_dir = data_root / f"dataset_version={source_ds_version}"
    source_dir.mkdir(parents=True)

    # Write source manifest
    source_manifest = {
        "dataset_version": source_ds_version,
        "feature_schema_version": "v1",
        "target_schema_version": "v1",
        "timeframe": "1h",
        "horizon_bars": 24,
        "tickers": ["SBER"],
        "feature_columns": ["f1", "f2"]
    }
    (source_dir / "manifest.json").write_text(json.dumps(source_manifest))

    # Write train split
    train_dir = source_dir / "split=train"
    train_dir.mkdir()

    dates = pd.date_range("2026-01-01", periods=100, freq="1h")
    df = pd.DataFrame({
        "ticker": ["SBER"] * 100,
        "asof_time": dates,
        "f1": np.random.randn(100),
        "f2": np.random.randn(100),
        "label_class": ["up", "down", "no_trade", "up"] * 25,
        "label_up": [1.0] * 100,
        "label_down": [0.0] * 100,
        "label_no_trade": [0.0] * 100,
        "feature_schema_version": ["v1"] * 100
    })
    df.to_parquet(train_dir / "part-000.parquet", index=False)

    config = SequenceDatasetConfig(
        data_root=data_root,
        output_root=output_root,
        source_dataset_version=source_ds_version,
        sequence_dataset_version="seq_v1",
        timeframe="1h",
        horizon_bars=24,
        window_bars=48,
        stride=1,
        min_coverage_ratio=0.9,
        tickers=["SBER"],
        feature_cols=["f1", "f2"]
    )

    manifest = build_sequence_dataset(config)

    assert manifest.dataset_version == "seq_v1"
    assert manifest.window_bars == 48
    assert manifest.feature_order == ["f1", "f2"]
    assert manifest.input_shape == [48, 2]

    assert "train" in manifest.splits
    train_split = manifest.splits["train"]
    assert train_split.tickers == ["SBER"]
    # 100 rows, window=48, stride=1 -> 100 - 48 + 1 = 53 windows
    assert train_split.windows == 53

    # verify saved parquet file
    out_dir = output_root / "dataset_version=seq_v1" / "window=48"
    train_out_path = out_dir / "split=train" / "ticker=SBER.parquet"
    assert train_out_path.exists()

    seq_df = pd.read_parquet(train_out_path)
    assert len(seq_df) == 53
    assert "features" in seq_df.columns
    assert "label" in seq_df.columns

    # Check features shape
    first_features = seq_df.iloc[0]["features"]
    assert len(first_features) == 48 * 2  # flattened 48x2
