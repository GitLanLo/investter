import json
from pathlib import Path

import numpy as np
import pandas as pd
import pytest
import torch

from ml_core.datasets.sequences import SequenceDatasetConfig, build_sequence_dataset
from ml_core.training.neural import NeuralTrainingConfig, train_neural_sequence

def test_neural_training_smoke(tmp_path: Path):
    data_root = tmp_path / "data"
    output_root = tmp_path / "sequences"
    model_root = tmp_path / "models"

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

    # Write train and val splits
    for split in ["train", "val"]:
        split_dir = source_dir / f"split={split}"
        split_dir.mkdir()

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
        df.to_parquet(split_dir / "part-000.parquet", index=False)

    # Build sequence dataset
    seq_config = SequenceDatasetConfig(
        data_root=data_root,
        output_root=output_root,
        source_dataset_version=source_ds_version,
        sequence_dataset_version="seq_v1",
        timeframe="1h",
        horizon_bars=24,
        window_bars=10,
        stride=1,
        min_coverage_ratio=0.9,
        tickers=["SBER"],
        feature_cols=["f1", "f2"]
    )
    build_sequence_dataset(seq_config)

    # Train neural model
    train_config = NeuralTrainingConfig(
        sequence_root=output_root,
        dataset_version="seq_v1",
        window_bars=10,
        output_root=model_root / "sprint8_gru_smoke",
        model_family="gru",
        epochs=2,
        batch_size=4,
        hidden_size=8,
        num_layers=1,
        device="cpu"
    )

    manifest = train_neural_sequence(train_config)

    assert manifest.model_family == "gru"
    assert (model_root / "sprint8_gru_smoke" / "model.pth").exists()
    assert (model_root / "sprint8_gru_smoke" / "scaler.joblib").exists()
    assert (model_root / "sprint8_gru_smoke" / "model_manifest.json").exists()
