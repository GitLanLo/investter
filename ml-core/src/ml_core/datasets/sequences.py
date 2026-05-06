from __future__ import annotations

from dataclasses import dataclass, field, asdict
from pathlib import Path
import json
import hashlib

import numpy as np
import pandas as pd

from ml_core.contracts import FEATURE_SCHEMA_VERSION

@dataclass(slots=True)
class SequenceDatasetConfig:
    data_root: Path
    output_root: Path
    source_dataset_version: str
    sequence_dataset_version: str
    timeframe: str
    horizon_bars: int
    window_bars: int
    stride: int = 1
    min_coverage_ratio: float = 0.9
    tickers: list[str] | None = None
    feature_cols: list[str] | None = None
    exclude_cols: list[str] | None = None

@dataclass(slots=True)
class SequenceSplitManifest:
    windows: int
    tickers: list[str]
    file_checksums: dict[str, str]

@dataclass(slots=True)
class SequenceManifest:
    dataset_version: str
    source_dataset_version: str
    timeframe: str
    horizon_bars: int
    window_bars: int
    stride: int
    feature_count: int
    feature_order: list[str]
    input_shape: list[int]
    class_mapping: dict[str, int]
    splits: dict[str, SequenceSplitManifest]

def _compute_sha256(path: Path) -> str:
    sha256_hash = hashlib.sha256()
    with open(path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    return sha256_hash.hexdigest()

def build_sequence_dataset(config: SequenceDatasetConfig) -> SequenceManifest:
    source_root = config.data_root / f"dataset_version={config.source_dataset_version}"
    if not source_root.exists():
        raise FileNotFoundError(f"Source dataset not found: {source_root}")

    out_dir = config.output_root / f"dataset_version={config.sequence_dataset_version}" / f"window={config.window_bars}"
    if out_dir.exists():
        import shutil
        shutil.rmtree(out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    source_manifest_path = source_root / "manifest.json"
    if not source_manifest_path.exists():
        raise FileNotFoundError(f"Source manifest not found: {source_manifest_path}")

    source_manifest = json.loads(source_manifest_path.read_text())

    # Feature inference
    first_split = source_root / "split=train"
    if not first_split.exists():
        raise ValueError("Train split not found in source dataset")
    sample_df = pd.read_parquet(next(first_split.rglob("*.parquet")))

    if config.feature_cols:
        feature_cols = config.feature_cols
    else:
        # Use features from source manifest if available, otherwise infer
        source_features = source_manifest.get("feature_columns", [])
        if source_features:
            feature_cols = source_features
        else:
            # Exclude non-features
            exclude = {
                "asof_time", "ticker", "label_class", "label_up", "label_down", "label_no_trade",
                "feature_schema_version", "source", "ingested_at", "timeframe", "timestamp",
                "label_asof_time", "label_horizon_end_time", "label_is_complete"
            }
            # Only include numeric types
            numeric_cols = sample_df.select_dtypes(include=[np.number]).columns
            feature_cols = sorted(col for col in numeric_cols if col not in exclude)

    if config.exclude_cols:
        feature_cols = [c for c in feature_cols if c not in config.exclude_cols]

    # Final validation: check if all feature_cols exist in sample_df
    missing = [c for c in feature_cols if c not in sample_df.columns]
    if missing:
        raise ValueError(f"Requested features missing from source dataset: {missing}")

    class_mapping = {"down_signal": 0, "no_trade": 1, "up_signal": 2}

    splits_meta = {}

    for split in ["train", "val", "test"]:
        split_dir = source_root / f"split={split}"
        if not split_dir.exists():
            continue

        out_split_dir = out_dir / f"split={split}"
        out_split_dir.mkdir(parents=True, exist_ok=True)

        split_tickers = set()
        split_windows = 0
        split_checksums = {}

        # Load all data for the split first to build windows across files
        all_split_data = []
        for p in split_dir.rglob("*.parquet"):
            df = pd.read_parquet(p)
            if df.empty:
                continue
            if config.tickers:
                df = df[df["ticker"].isin(config.tickers)]
                if df.empty:
                    continue
            all_split_data.append(df)

        if not all_split_data:
            continue

        full_df = pd.concat(all_split_data, ignore_index=True)
        full_df = full_df.sort_values(["ticker", "asof_time"])

        for ticker, t_df in full_df.groupby("ticker"):
            if len(t_df) < config.window_bars:
                continue

            t_df = t_df.reset_index(drop=True)

            # Windowing logic
            windows_features = []
            windows_labels = []
            windows_meta = []

            features_arr = t_df[feature_cols].to_numpy(dtype=np.float32)
            labels_arr = t_df["label_class"].map(class_mapping).to_numpy(dtype=np.int64)
            asof_arr = t_df["asof_time"].to_numpy()

            for i in range(0, len(t_df) - config.window_bars + 1, config.stride):
                f_win = features_arr[i : i + config.window_bars]

                # Missing check
                if np.isnan(f_win).mean() > (1.0 - config.min_coverage_ratio):
                    continue

                # Forward fill any isolated NaNs inside the window
                if np.isnan(f_win).any():
                    f_win = pd.DataFrame(f_win).ffill().bfill().to_numpy()

                # Target is the label of the LAST bar in the window
                target_idx = i + config.window_bars - 1
                target_label = labels_arr[target_idx]

                windows_features.append(f_win)
                windows_labels.append(target_label)
                windows_meta.append({
                    "ticker": ticker,
                    "asof_time": asof_arr[target_idx]
                })

            if not windows_features:
                continue

            # Save ticker windows
            out_path = out_split_dir / f"ticker={ticker}.parquet"

            res_df = pd.DataFrame(windows_meta)
            res_df["label"] = windows_labels
            res_df["features"] = [w.flatten() for w in windows_features]

            res_df.to_parquet(out_path, index=False)

            split_tickers.add(ticker)
            split_windows += len(windows_features)
            split_checksums[out_path.name] = _compute_sha256(out_path)

        if split_windows > 0:
            splits_meta[split] = SequenceSplitManifest(
                windows=split_windows,
                tickers=sorted(list(split_tickers)),
                file_checksums=split_checksums
            )

    manifest = SequenceManifest(
        dataset_version=config.sequence_dataset_version,
        source_dataset_version=config.source_dataset_version,
        timeframe=config.timeframe,
        horizon_bars=config.horizon_bars,
        window_bars=config.window_bars,
        stride=config.stride,
        feature_count=len(feature_cols),
        feature_order=feature_cols,
        input_shape=[config.window_bars, len(feature_cols)],
        class_mapping=class_mapping,
        splits=splits_meta,
    )

    manifest_path = out_dir / "manifest.json"
    manifest_path.write_text(json.dumps(asdict(manifest), indent=2, ensure_ascii=False, default=str), encoding="utf-8")

    return manifest
