from __future__ import annotations

from pathlib import Path
from typing import Iterable

import pandas as pd

from ml_core.contracts import FEATURE_SCHEMA_VERSION


RAW_REQUIRED_COLUMNS = {"timestamp", "open", "high", "low", "close", "volume"}


def ingest_asset_frame(
    df: pd.DataFrame,
    *,
    data_root: Path,
    ticker: str,
    timeframe: str,
    source: str,
    ingested_at: pd.Timestamp | None = None,
) -> list[Path]:
    prepared = _prepare_raw_frame(
        df,
        id_column="ticker",
        id_value=ticker,
        timeframe=timeframe,
        source=source,
        ingested_at=ingested_at,
    )
    base = data_root / "raw" / "candles" / f"ticker={ticker}" / f"timeframe={timeframe}"
    return write_partitioned_frame(prepared, base_path=base, time_column="timestamp")


def ingest_factor_frame(
    df: pd.DataFrame,
    *,
    data_root: Path,
    alias: str,
    timeframe: str,
    source: str,
    ingested_at: pd.Timestamp | None = None,
) -> list[Path]:
    prepared = _prepare_raw_frame(
        df,
        id_column="factor_alias",
        id_value=alias,
        timeframe=timeframe,
        source=source,
        ingested_at=ingested_at,
    )
    base = data_root / "raw" / "factors" / f"factor={alias}" / f"timeframe={timeframe}"
    return write_partitioned_frame(prepared, base_path=base, time_column="timestamp")


def write_feature_store(
    df: pd.DataFrame,
    *,
    data_root: Path,
    ticker: str,
    timeframe: str,
    horizon_bars: int,
    schema_version: str = FEATURE_SCHEMA_VERSION,
) -> list[Path]:
    prepared = df.copy()
    time_column = "asof_time" if "asof_time" in prepared.columns else "timestamp"
    prepared[time_column] = pd.to_datetime(prepared[time_column], utc=True)
    base = data_root / "features" / f"schema={schema_version}" / f"ticker={ticker}" / f"timeframe={timeframe}" / f"horizon={horizon_bars}"
    return write_partitioned_frame(prepared, base_path=base, time_column=time_column)


def load_feature_store(
    data_root: Path,
    *,
    ticker: str,
    timeframe: str,
    horizon_bars: int,
    schema_version: str = FEATURE_SCHEMA_VERSION,
    start: pd.Timestamp | None = None,
    end: pd.Timestamp | None = None,
) -> pd.DataFrame:
    base = data_root / "features" / f"schema={schema_version}" / f"ticker={ticker}" / f"timeframe={timeframe}" / f"horizon={horizon_bars}"
    df = _load_parquet_files(base.rglob("*.parquet"))
    if df.empty:
        return df

    time_column = "asof_time" if "asof_time" in df.columns else "timestamp"
    df[time_column] = pd.to_datetime(df[time_column], utc=True)
    if start is not None:
        df = df[df[time_column] >= _to_utc_timestamp(start)]
    if end is not None:
        df = df[df[time_column] <= _to_utc_timestamp(end)]
    return df.sort_values(time_column).reset_index(drop=True)


def write_dataset_split_frame(
    df: pd.DataFrame,
    *,
    dataset_root: Path,
    split_name: str,
) -> list[Path]:
    base = dataset_root / f"split={split_name}"
    if df.empty:
        base.mkdir(parents=True, exist_ok=True)
        out_path = base / "part-empty.parquet"
        df.to_parquet(out_path, index=False)
        return [out_path]
    return write_partitioned_frame(df, base_path=base, time_column="timestamp")


def write_partitioned_frame(
    df: pd.DataFrame,
    *,
    base_path: Path,
    time_column: str,
) -> list[Path]:
    if time_column not in df.columns:
        raise ValueError(f"df must contain {time_column} column")

    prepared = df.copy()
    prepared[time_column] = pd.to_datetime(prepared[time_column], utc=True)
    prepared = prepared.sort_values(time_column).reset_index(drop=True)
    prepared["_partition_date"] = prepared[time_column].dt.strftime("%Y-%m-%d")

    written: list[Path] = []
    for partition_date, chunk in prepared.groupby("_partition_date", sort=True):
        out_dir = base_path / f"date={partition_date}"
        out_dir.mkdir(parents=True, exist_ok=True)
        partition_df = chunk.drop(columns="_partition_date")
        existing_files = sorted(out_dir.glob("*.parquet"))
        if existing_files:
            existing = _load_parquet_files(existing_files)
            if not existing.empty:
                existing[time_column] = pd.to_datetime(existing[time_column], utc=True)
                partition_df = pd.concat([existing, partition_df], ignore_index=True)
                partition_df = (
                    partition_df.sort_values(time_column)
                    .drop_duplicates(subset=_dedupe_subset(partition_df, time_column), keep="last")
                    .reset_index(drop=True)
                )
            for existing_file in existing_files:
                existing_file.unlink()

        file_name = _partition_file_name(partition_df[time_column])
        out_path = out_dir / file_name
        partition_df.to_parquet(out_path, index=False)
        written.append(out_path)

    return written


def _prepare_raw_frame(
    df: pd.DataFrame,
    *,
    id_column: str,
    id_value: str,
    timeframe: str,
    source: str,
    ingested_at: pd.Timestamp | None = None,
) -> pd.DataFrame:
    missing = RAW_REQUIRED_COLUMNS - set(df.columns)
    if missing:
        raise ValueError(f"df is missing columns: {sorted(missing)}")

    prepared = df.copy()
    prepared["timestamp"] = pd.to_datetime(prepared["timestamp"], utc=True)
    prepared = prepared.sort_values("timestamp").drop_duplicates(subset=["timestamp"]).reset_index(drop=True)
    prepared[id_column] = id_value
    prepared["timeframe"] = timeframe
    prepared["source"] = source
    prepared["ingested_at"] = ingested_at or pd.Timestamp.now(tz="UTC")
    return prepared


def _partition_file_name(series: pd.Series) -> str:
    start = series.min().strftime("%Y%m%dT%H%M%SZ")
    end = series.max().strftime("%Y%m%dT%H%M%SZ")
    return f"part-{start}-{end}.parquet"


def _load_parquet_files(files: Iterable[Path]) -> pd.DataFrame:
    paths = sorted(Path(path) for path in files)
    if not paths:
        return pd.DataFrame()
    frames = [pd.read_parquet(path) for path in paths]
    out = pd.concat(frames, ignore_index=True)
    for candidate in ("asof_time", "timestamp"):
        if candidate in out.columns:
            out[candidate] = pd.to_datetime(out[candidate], utc=True)
            out = (
                out.sort_values(candidate)
                .drop_duplicates(subset=_dedupe_subset(out, candidate), keep="last")
                .reset_index(drop=True)
            )
            break
    return out


def _dedupe_subset(df: pd.DataFrame, time_column: str) -> list[str]:
    dimensions = [
        column
        for column in ("ticker", "factor_alias", "factor", "timeframe")
        if column in df.columns
    ]
    return [time_column, *dimensions]


def _to_utc_timestamp(value: pd.Timestamp) -> pd.Timestamp:
    ts = pd.Timestamp(value)
    if ts.tzinfo is None:
        return ts.tz_localize("UTC")
    return ts.tz_convert("UTC")
