from __future__ import annotations

import pandas as pd

from ml_core.datasets.splits import SplitConfig, build_time_splits


def test_dataset_splits_respect_purge_gap_and_manifest() -> None:
    df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=30, freq="5min"),
            "ticker": ["SBER"] * 30,
            "feature_schema_version": ["feature_v1"] * 30,
            "ret_1": list(range(30)),
            "label_class": ["up_signal"] * 30,
            "label_up": [1] * 30,
            "label_down": [0] * 30,
            "label_no_trade": [0] * 30,
        }
    )

    train, val, test, manifest = build_time_splits(
        df,
        config=SplitConfig(dataset_version="v1", train_ratio=0.5, val_ratio=0.25, purge_gap_bars=2),
    )

    assert not train.empty
    assert not val.empty
    assert not test.empty
    assert train["timestamp"].max() < val["timestamp"].min()
    assert val["timestamp"].max() < test["timestamp"].min()
    assert manifest["dataset_version"] == "v1"
    assert "ret_1" in manifest["feature_columns"]

