from __future__ import annotations

from pathlib import Path

import pandas as pd

from ml_core.ingest.providers import LocalParquetProvider
from ml_core.storage.layouts import ingest_asset_frame, ingest_factor_frame, write_dataset_split_frame


def test_ingest_writes_contract_aligned_raw_layout_and_provider_reads_it(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    asset_df = pd.DataFrame(
        {
            "timestamp": [
                "2026-01-01T10:00:00Z",
                "2026-01-01T10:05:00Z",
                "2026-01-02T10:00:00Z",
            ],
            "open": [100.0, 101.0, 102.0],
            "high": [100.5, 101.5, 102.5],
            "low": [99.5, 100.5, 101.5],
            "close": [100.2, 101.2, 102.2],
            "volume": [1000, 1010, 1020],
        }
    )
    factor_df = pd.DataFrame(
        {
            "timestamp": [
                "2026-01-01T10:00:00Z",
                "2026-01-01T10:05:00Z",
                "2026-01-02T10:00:00Z",
            ],
            "open": [90.0, 90.1, 90.2],
            "high": [90.3, 90.4, 90.5],
            "low": [89.7, 89.8, 89.9],
            "close": [90.1, 90.2, 90.3],
            "volume": [500, 510, 520],
        }
    )

    asset_paths = ingest_asset_frame(
        asset_df,
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        source="unit_test",
    )
    factor_paths = ingest_factor_frame(
        factor_df,
        data_root=data_root,
        alias="usdrub",
        timeframe="5m",
        source="unit_test",
    )

    assert any("raw/candles/ticker=SBER/timeframe=5m/date=2026-01-01" in str(path) for path in asset_paths)
    assert any("raw/factors/factor=usdrub/timeframe=5m/date=2026-01-02" in str(path) for path in factor_paths)

    provider = LocalParquetProvider(root=data_root)
    loaded_asset = provider.fetch_asset_history("SBER", "5m")
    loaded_factor = provider.fetch_factor_history("usdrub", "5m")

    assert len(loaded_asset) == 3
    assert len(loaded_factor) == 3
    assert {"ticker", "timeframe", "source", "ingested_at"}.issubset(loaded_asset.columns)
    assert {"factor_alias", "timeframe", "source", "ingested_at"}.issubset(loaded_factor.columns)


def test_partition_writer_rewrites_existing_partition_without_duplicates(tmp_path: Path) -> None:
    data_root = tmp_path / "data"

    first = pd.DataFrame(
        {
            "timestamp": ["2026-01-01T10:00:00Z", "2026-01-01T10:05:00Z"],
            "open": [100.0, 101.0],
            "high": [100.5, 101.5],
            "low": [99.5, 100.5],
            "close": [100.2, 101.2],
            "volume": [1000, 1010],
        }
    )
    second = pd.DataFrame(
        {
            "timestamp": ["2026-01-01T10:05:00Z", "2026-01-01T10:10:00Z"],
            "open": [101.0, 102.0],
            "high": [101.5, 102.5],
            "low": [100.5, 101.5],
            "close": [101.2, 102.2],
            "volume": [1010, 1020],
        }
    )

    ingest_asset_frame(first, data_root=data_root, ticker="SBER", timeframe="5m", source="unit_test")
    ingest_asset_frame(second, data_root=data_root, ticker="SBER", timeframe="5m", source="unit_test")

    provider = LocalParquetProvider(root=data_root)
    loaded_asset = provider.fetch_asset_history("SBER", "5m")
    assert len(loaded_asset) == 3
    assert loaded_asset["timestamp"].is_unique


def test_local_provider_resamples_5m_history_when_higher_timeframe_is_missing(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    asset_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=6, freq="5min"),
            "open": [100.0, 101.0, 102.0, 103.0, 104.0, 105.0],
            "high": [101.0, 102.0, 103.0, 104.0, 105.0, 106.0],
            "low": [99.0, 100.0, 101.0, 102.0, 103.0, 104.0],
            "close": [100.5, 101.5, 102.5, 103.5, 104.5, 105.5],
            "volume": [10, 20, 30, 40, 50, 60],
        }
    )

    ingest_asset_frame(asset_df, data_root=data_root, ticker="SBER", timeframe="5m", source="unit_test")

    provider = LocalParquetProvider(root=data_root)
    loaded_asset = provider.fetch_asset_history("SBER", "15m")

    assert len(loaded_asset) == 2
    assert loaded_asset["timeframe"].eq("15m").all()
    assert loaded_asset.iloc[0]["open"] == 100.0
    assert loaded_asset.iloc[0]["high"] == 103.0
    assert loaded_asset.iloc[0]["low"] == 99.0
    assert loaded_asset.iloc[0]["close"] == 102.5
    assert loaded_asset.iloc[0]["volume"] == 60


def test_dataset_partition_writer_preserves_same_timestamp_for_different_tickers(tmp_path: Path) -> None:
    dataset_root = tmp_path / "dataset"
    first = pd.DataFrame(
        {
            "timestamp": ["2026-01-01T10:00:00Z", "2026-01-01T10:00:00Z"],
            "ticker": ["SBER", "GAZP"],
            "timeframe": ["5m", "5m"],
            "close": [100.0, 200.0],
        }
    )
    second = pd.DataFrame(
        {
            "timestamp": ["2026-01-01T10:00:00Z", "2026-01-01T10:05:00Z"],
            "ticker": ["SBER", "SBER"],
            "timeframe": ["5m", "5m"],
            "close": [101.0, 102.0],
        }
    )

    write_dataset_split_frame(first, dataset_root=dataset_root, split_name="train")
    write_dataset_split_frame(second, dataset_root=dataset_root, split_name="train")

    loaded = pd.concat(
        [pd.read_parquet(path) for path in sorted((dataset_root / "split=train").rglob("*.parquet"))],
        ignore_index=True,
    )
    assert len(loaded) == 3
    assert set(loaded["ticker"]) == {"SBER", "GAZP"}
    sber_close = loaded.loc[
        (loaded["ticker"] == "SBER") & (loaded["timestamp"] == pd.Timestamp("2026-01-01T10:00:00Z")),
        "close",
    ].item()
    assert sber_close == 101.0
