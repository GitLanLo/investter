from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

import pandas as pd

RESAMPLE_BASE_TIMEFRAME = "5m"
RESAMPLE_RULES = {
    "15m": "15min",
    "1h": "1h",
}


def _filter_time_range(
    df: pd.DataFrame,
    *,
    start: pd.Timestamp | None = None,
    end: pd.Timestamp | None = None,
) -> pd.DataFrame:
    if df.empty:
        return df
    out = df.copy()
    out["timestamp"] = pd.to_datetime(out["timestamp"], utc=True)
    if start is not None:
        out = out[out["timestamp"] >= _to_utc_timestamp(start)]
    if end is not None:
        out = out[out["timestamp"] <= _to_utc_timestamp(end)]
    return out.sort_values("timestamp").drop_duplicates(subset=["timestamp"], keep="last").reset_index(drop=True)


class MarketDataProvider(ABC):
    @abstractmethod
    def fetch_asset_history(
        self,
        ticker: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        raise NotImplementedError

    @abstractmethod
    def fetch_factor_history(
        self,
        alias: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        raise NotImplementedError


@dataclass(slots=True)
class LocalParquetProvider(MarketDataProvider):
    root: Path

    def fetch_asset_history(
        self,
        ticker: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        root = _resolve_data_root(self.root)
        base = root / "raw" / "candles" / f"ticker={ticker}" / f"timeframe={timeframe}"
        frame = _load_parquet_files(base.rglob("*.parquet"), start=start, end=end)
        if not frame.empty or timeframe not in RESAMPLE_RULES:
            return frame

        base_5m = root / "raw" / "candles" / f"ticker={ticker}" / f"timeframe={RESAMPLE_BASE_TIMEFRAME}"
        return _resample_ohlcv(
            _load_parquet_files(base_5m.rglob("*.parquet"), start=start, end=end),
            timeframe=timeframe,
            id_column="ticker",
            id_value=ticker,
        )

    def fetch_factor_history(
        self,
        alias: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        root = _resolve_data_root(self.root)
        base = root / "raw" / "factors" / f"factor={alias}" / f"timeframe={timeframe}"
        frame = _load_parquet_files(base.rglob("*.parquet"), start=start, end=end)
        if not frame.empty or timeframe not in RESAMPLE_RULES:
            return frame

        base_5m = root / "raw" / "factors" / f"factor={alias}" / f"timeframe={RESAMPLE_BASE_TIMEFRAME}"
        return _resample_ohlcv(
            _load_parquet_files(base_5m.rglob("*.parquet"), start=start, end=end),
            timeframe=timeframe,
            id_column="factor_alias",
            id_value=alias,
        )


def _resample_ohlcv(
    df: pd.DataFrame,
    *,
    timeframe: str,
    id_column: str,
    id_value: str,
) -> pd.DataFrame:
    if df.empty:
        return df
    rule = RESAMPLE_RULES.get(timeframe)
    if rule is None:
        return df

    prepared = df.copy()
    prepared["timestamp"] = pd.to_datetime(prepared["timestamp"], utc=True)
    prepared = prepared.sort_values("timestamp").drop_duplicates(subset=["timestamp"], keep="last")
    prepared = prepared.set_index("timestamp")
    aggregations = {
        "open": "first",
        "high": "max",
        "low": "min",
        "close": "last",
        "volume": "sum",
    }
    out = prepared.resample(rule, label="left", closed="left").agg(aggregations).dropna(subset=["open", "close"])
    out = out.reset_index()
    out[id_column] = id_value
    out["timeframe"] = timeframe
    out["source"] = "local_parquet_resampled_5m"
    if "ingested_at" in prepared.columns:
        out["ingested_at"] = prepared["ingested_at"].max()
    return out.sort_values("timestamp").reset_index(drop=True)


def _resolve_data_root(root: Path) -> Path:
    root = Path(root)
    if (root / "raw").exists() or (root / "features").exists():
        return root
    if (root / "data" / "raw").exists() or (root / "data" / "features").exists():
        return root / "data"
    return root


def _to_utc_timestamp(value: pd.Timestamp) -> pd.Timestamp:
    ts = pd.Timestamp(value)
    if ts.tzinfo is None:
        return ts.tz_localize("UTC")
    return ts.tz_convert("UTC")


def _load_parquet_files(
    files: Iterable[Path],
    *,
    start: pd.Timestamp | None = None,
    end: pd.Timestamp | None = None,
) -> pd.DataFrame:
    paths = sorted(Path(p) for p in files)
    if not paths:
        return pd.DataFrame()

    frames = [pd.read_parquet(path) for path in paths]
    df = pd.concat(frames, ignore_index=True)
    return _filter_time_range(df, start=start, end=end)
