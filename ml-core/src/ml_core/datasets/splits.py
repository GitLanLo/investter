from __future__ import annotations

from dataclasses import asdict, dataclass
from pathlib import Path
import json

import pandas as pd
from pandas.api.types import is_bool_dtype, is_numeric_dtype

from ml_core.contracts import (
    DATASET_SCHEMA_VERSION,
    FEATURE_SCHEMA_VERSION,
    TARGET_SCHEMA_VERSION,
    DEFAULT_HORIZON_BARS,
    DEFAULT_TIMEFRAME,
)
from ml_core.storage.layouts import write_dataset_split_frame


@dataclass(slots=True)
class SplitConfig:
    dataset_version: str
    train_ratio: float = 0.7
    val_ratio: float = 0.15
    purge_gap_bars: int = DEFAULT_HORIZON_BARS
    feature_schema_version: str = FEATURE_SCHEMA_VERSION
    target_schema_version: str = TARGET_SCHEMA_VERSION
    dataset_schema_version: str = DATASET_SCHEMA_VERSION
    timeframe: str = DEFAULT_TIMEFRAME
    horizon_bars: int = DEFAULT_HORIZON_BARS


def build_time_splits(
    df: pd.DataFrame,
    *,
    config: SplitConfig,
) -> tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, dict]:
    if "timestamp" not in df.columns:
        raise ValueError("df must contain timestamp column")

    data = df.copy()
    data["timestamp"] = pd.to_datetime(data["timestamp"], utc=True)
    data = data.sort_values("timestamp").reset_index(drop=True)

    if "label_class" in data.columns:
        data = data[data["label_class"].notna()].reset_index(drop=True)

    unique_ts = pd.Index(sorted(data["timestamp"].unique()))
    if len(unique_ts) < 3:
        raise ValueError("not enough timestamps to create train/val/test splits")

    train_end_idx = max(0, int(len(unique_ts) * config.train_ratio) - 1)
    val_end_idx = max(train_end_idx + 1, int(len(unique_ts) * (config.train_ratio + config.val_ratio)) - 1)
    val_end_idx = min(val_end_idx, len(unique_ts) - 1)

    train_end_ts = unique_ts[train_end_idx]
    val_start_idx = min(train_end_idx + config.purge_gap_bars + 1, len(unique_ts) - 1)
    val_start_ts = unique_ts[val_start_idx]
    val_end_ts = unique_ts[val_end_idx]
    test_start_idx = min(val_end_idx + config.purge_gap_bars + 1, len(unique_ts))

    train = data[data["timestamp"] <= train_end_ts].copy()
    val = data[(data["timestamp"] >= val_start_ts) & (data["timestamp"] <= val_end_ts)].copy()
    if test_start_idx >= len(unique_ts):
        test = data.iloc[0:0].copy()
        test_start_ts = None
    else:
        test_start_ts = unique_ts[test_start_idx]
        test = data[data["timestamp"] >= test_start_ts].copy()

    manifest = {
        "dataset_version": config.dataset_version,
        "dataset_schema_version": config.dataset_schema_version,
        "feature_schema_version": config.feature_schema_version,
        "target_schema_version": config.target_schema_version,
        "timeframe": config.timeframe,
        "horizon_bars": config.horizon_bars,
        "purge_gap_bars": config.purge_gap_bars,
        "feature_columns": sorted(_feature_columns_from_frame(data)),
        "tickers": sorted(data["ticker"].dropna().unique().tolist()) if "ticker" in data.columns else [],
        "train_range": _range_info(train),
        "val_range": _range_info(val),
        "test_range": _range_info(test),
        "split_config": asdict(config),
    }

    return train, val, test, manifest


def write_dataset_splits(
    df: pd.DataFrame,
    *,
    output_root: Path,
    config: SplitConfig,
) -> dict:
    train, val, test, manifest = build_time_splits(df, config=config)
    dataset_root = output_root / f"dataset_version={config.dataset_version}"
    dataset_root.mkdir(parents=True, exist_ok=True)

    write_dataset_split_frame(train, dataset_root=dataset_root, split_name="train")
    write_dataset_split_frame(val, dataset_root=dataset_root, split_name="val")
    write_dataset_split_frame(test, dataset_root=dataset_root, split_name="test")
    (dataset_root / "manifest.json").write_text(
        json.dumps(manifest, indent=2, ensure_ascii=False, default=str),
        encoding="utf-8",
    )
    return manifest


def _range_info(df: pd.DataFrame) -> dict:
    if df.empty:
        return {"min": None, "max": None, "rows": 0}
    return {
        "min": df["timestamp"].min().isoformat(),
        "max": df["timestamp"].max().isoformat(),
        "rows": int(len(df)),
    }


def _is_feature_column(column: str) -> bool:
    reserved = {
        "timestamp",
        "asof_time",
        "ticker",
        "timeframe",
        "source",
        "ingested_at",
        "factor_alias",
        "feature_schema_version",
        "target_schema_version",
        "label_class",
        "label_up",
        "label_down",
        "label_no_trade",
        "label_asof_time",
        "label_horizon_end_time",
        "label_is_complete",
        "horizon_bars",
        "move_pct",
        "barrier_up_price",
        "barrier_down_price",
    }
    return column not in reserved


def _feature_columns_from_frame(df: pd.DataFrame) -> list[str]:
    return [
        column
        for column in df.columns
        if _is_feature_column(column)
        and (is_numeric_dtype(df[column]) or is_bool_dtype(df[column]))
        and df[column].notna().any()
    ]
