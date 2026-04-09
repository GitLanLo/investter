from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

import pandas as pd


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
        base = _resolve_data_root(self.root) / "raw" / "candles" / f"ticker={ticker}" / f"timeframe={timeframe}"
        return _load_parquet_files(base.rglob("*.parquet"), start=start, end=end)

    def fetch_factor_history(
        self,
        alias: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        base = _resolve_data_root(self.root) / "raw" / "factors" / f"factor={alias}" / f"timeframe={timeframe}"
        return _load_parquet_files(base.rglob("*.parquet"), start=start, end=end)


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
