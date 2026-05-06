from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
import json

import pandas as pd

from ml_core.contracts import FEATURE_SCHEMA_VERSION
from ml_core.datasets.splits import SplitConfig, write_dataset_splits
from ml_core.features.build import FeatureBuildConfig, build_feature_frame
from ml_core.ingest.providers import LocalParquetProvider, MarketDataProvider
from ml_core.labels.triple_barrier import TripleBarrierConfig, apply_triple_barrier_labels
from ml_core.storage.layouts import load_feature_store, write_feature_store


@dataclass(slots=True)
class FeatureStoreMaterializationConfig:
    data_root: Path
    ticker: str
    timeframe: str
    factor_aliases: list[str] = field(default_factory=list)
    start: pd.Timestamp | None = None
    end: pd.Timestamp | None = None
    feature_config: FeatureBuildConfig = field(default_factory=FeatureBuildConfig)
    label_config: TripleBarrierConfig = field(default_factory=TripleBarrierConfig)


def materialize_feature_store(
    provider: MarketDataProvider,
    *,
    config: FeatureStoreMaterializationConfig,
) -> dict:
    asset_df = provider.fetch_asset_history(
        config.ticker,
        config.timeframe,
        start=config.start,
        end=config.end,
    )
    if asset_df.empty:
        raise ValueError(f"no raw asset history found for ticker={config.ticker}")

    factor_frames: dict[str, pd.DataFrame] = {}
    for alias in config.factor_aliases:
        factor_frame = provider.fetch_factor_history(
            alias,
            config.timeframe,
            start=config.start,
            end=config.end,
        )
        if not factor_frame.empty:
            factor_frames[alias] = factor_frame

    features = build_feature_frame(
        asset_df,
        factor_frames=factor_frames,
        ticker=config.ticker,
        config=config.feature_config,
    )
    labeled = apply_triple_barrier_labels(features, config=config.label_config)
    written = write_feature_store(
        labeled,
        data_root=config.data_root,
        ticker=config.ticker,
        timeframe=config.timeframe,
        horizon_bars=config.label_config.horizon_bars,
        schema_version=config.feature_config.feature_schema_version,
    )

    return {
        "ticker": config.ticker,
        "timeframe": config.timeframe,
        "rows": int(len(labeled)),
        "feature_schema_version": config.feature_config.feature_schema_version,
        "target_schema_version": config.label_config.target_schema_version,
        "factor_aliases": sorted(factor_frames.keys()),
        "written_files": [str(path) for path in written],
        "range": _range_info(labeled, "timestamp"),
    }


def materialize_dataset(
    *,
    data_root: Path,
    output_root: Path,
    dataset_version: str,
    tickers: list[str],
    timeframe: str,
    horizon_bars: int,
    split_config: SplitConfig | None = None,
    schema_version: str = FEATURE_SCHEMA_VERSION,
) -> dict:
    if not tickers:
        raise ValueError("tickers must not be empty")

    frames: list[pd.DataFrame] = []
    for ticker in tickers:
        frame = load_feature_store(
            data_root,
            ticker=ticker,
            timeframe=timeframe,
            horizon_bars=horizon_bars,
            schema_version=schema_version,
        )
        if frame.empty:
            raise ValueError(f"no feature store rows found for ticker={ticker}")
        frames.append(frame)

    dataset = pd.concat(frames, ignore_index=True)
    dataset["timestamp"] = pd.to_datetime(dataset["timestamp"], utc=True)
    dataset = dataset.sort_values("timestamp").reset_index(drop=True)

    config = split_config or SplitConfig(dataset_version=dataset_version)
    config.dataset_version = dataset_version
    manifest = write_dataset_splits(dataset, output_root=output_root, config=config)

    dataset_root = output_root / f"dataset_version={dataset_version}"
    (dataset_root / "materialization.json").write_text(
        json.dumps(
            {
                "tickers": tickers,
                "rows": int(len(dataset)),
                "schema_version": schema_version,
                "range": _range_info(dataset, "timestamp"),
            },
            indent=2,
            ensure_ascii=False,
            default=str,
        ),
        encoding="utf-8",
    )
    return manifest


def default_local_provider(data_root: Path) -> LocalParquetProvider:
    return LocalParquetProvider(root=data_root)


def _range_info(df: pd.DataFrame, time_column: str) -> dict:
    if df.empty:
        return {"min": None, "max": None, "rows": 0}
    return {
        "min": pd.to_datetime(df[time_column], utc=True).min().isoformat(),
        "max": pd.to_datetime(df[time_column], utc=True).max().isoformat(),
        "rows": int(len(df)),
    }
