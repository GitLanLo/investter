from __future__ import annotations

from pathlib import Path

import pandas as pd

from ml_core.contracts import FEATURE_SCHEMA_VERSION
from ml_core.ingest.providers import LocalParquetProvider
from ml_core.pipelines.materialize import FeatureStoreMaterializationConfig, materialize_dataset, materialize_feature_store
from ml_core.storage.layouts import ingest_asset_frame, ingest_factor_frame, load_feature_store


def test_materialize_feature_store_and_dataset_pipeline(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    dataset_root = tmp_path / "datasets"
    asset_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=90, freq="5min"),
            "open": [100 + i * 0.1 for i in range(90)],
            "high": [100.2 + i * 0.1 for i in range(90)],
            "low": [99.8 + i * 0.1 for i in range(90)],
            "close": [100 + i * 0.1 for i in range(90)],
            "volume": [1000 + i for i in range(90)],
        }
    )
    factor_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=90, freq="5min"),
            "open": [90 + i * 0.05 for i in range(90)],
            "high": [90.2 + i * 0.05 for i in range(90)],
            "low": [89.8 + i * 0.05 for i in range(90)],
            "close": [90 + i * 0.05 for i in range(90)],
            "volume": [500 + i for i in range(90)],
        }
    )

    ingest_asset_frame(
        asset_df,
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        source="unit_test",
    )
    for alias in ("usdrub", "brent", "rtsi"):
        ingest_factor_frame(
            factor_df,
            data_root=data_root,
            alias=alias,
            timeframe="5m",
            source="unit_test",
        )

    provider = LocalParquetProvider(root=data_root)
    result = materialize_feature_store(
        provider,
        config=FeatureStoreMaterializationConfig(
            data_root=data_root,
            ticker="SBER",
            timeframe="5m",
            factor_aliases=["usdrub", "brent", "rtsi"],
        ),
    )

    assert result["rows"] == 90
    assert result["feature_schema_version"] == FEATURE_SCHEMA_VERSION
    assert result["factor_aliases"] == ["brent", "rtsi", "usdrub"]

    feature_frame = load_feature_store(data_root, ticker="SBER", timeframe="5m", horizon_bars=12)
    assert not feature_frame.empty
    assert {"label_class", "label_up", "label_down", "label_no_trade"}.issubset(feature_frame.columns)
    assert feature_frame["feature_schema_version"].eq(FEATURE_SCHEMA_VERSION).all()

    manifest = materialize_dataset(
        data_root=data_root,
        output_root=dataset_root,
        dataset_version="v1",
        tickers=["SBER"],
        timeframe="5m",
        horizon_bars=12,
    )

    assert manifest["dataset_version"] == "v1"
    assert (dataset_root / "dataset_version=v1" / "manifest.json").exists()
    assert list((dataset_root / "dataset_version=v1" / "split=train").rglob("*.parquet"))
    assert list((dataset_root / "dataset_version=v1" / "split=val").rglob("*.parquet"))
    assert list((dataset_root / "dataset_version=v1" / "split=test").rglob("*.parquet"))
