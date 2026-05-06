from __future__ import annotations

from dataclasses import dataclass
from typing import Mapping

import numpy as np
import pandas as pd

from ml_core.contracts import FEATURE_SCHEMA_VERSION, DEFAULT_TIMEFRAME, DEFAULT_TIMEZONE


@dataclass(slots=True)
class FeatureBuildConfig:
    timeframe: str = DEFAULT_TIMEFRAME
    timezone: str = DEFAULT_TIMEZONE
    factor_ffill_limit: int = 12
    factor_staleness_minutes: int = 60
    feature_schema_version: str = FEATURE_SCHEMA_VERSION


def build_feature_frame(
    asset_df: pd.DataFrame,
    *,
    factor_frames: Mapping[str, pd.DataFrame] | None = None,
    ticker: str,
    config: FeatureBuildConfig | None = None,
) -> pd.DataFrame:
    cfg = config or FeatureBuildConfig()
    df = _prepare_asset_frame(asset_df, ticker=ticker, timeframe=cfg.timeframe)
    df = _add_core_price_features(df, timezone=cfg.timezone)
    df = _add_trend_features(df)
    df = _add_volatility_features(df)
    df = _add_volume_features(df)
    df = _add_calendar_features(df, timezone=cfg.timezone)

    factor_frames = factor_frames or {}
    if factor_frames:
        df = _merge_factor_frames(
            df,
            factor_frames=factor_frames,
            tolerance=pd.Timedelta(minutes=cfg.factor_staleness_minutes),
            ffill_limit=cfg.factor_ffill_limit,
        )
        df = _add_cross_asset_features(df, aliases=tuple(factor_frames.keys()))

    df = _add_regime_features(df, aliases=tuple(factor_frames.keys()))
    df["feature_schema_version"] = cfg.feature_schema_version
    return df


def _prepare_asset_frame(asset_df: pd.DataFrame, *, ticker: str, timeframe: str) -> pd.DataFrame:
    required = {"timestamp", "open", "high", "low", "close", "volume"}
    missing = required - set(asset_df.columns)
    if missing:
        raise ValueError(f"asset_df is missing columns: {sorted(missing)}")

    df = asset_df.copy()
    df["timestamp"] = _to_utc_ns(df["timestamp"])
    df = df.sort_values("timestamp").reset_index(drop=True)
    df["ticker"] = ticker
    df["timeframe"] = timeframe
    df["asof_time"] = df["timestamp"]
    return df


def _add_core_price_features(df: pd.DataFrame, *, timezone: str) -> pd.DataFrame:
    px = df["close"].replace(0, np.nan)
    prev_close = px.shift(1)

    for window in (1, 3, 6, 12, 24):
        df[f"ret_{window}"] = px / px.shift(window) - 1.0
    df["log_ret_1"] = np.log(px) - np.log(prev_close)
    df["close_to_prev_close"] = px / prev_close - 1.0
    df["hl_range_pct"] = (df["high"] - df["low"]) / px
    df["oc_range_pct"] = (df["close"] - df["open"]) / df["open"].replace(0, np.nan)

    local_day = df["timestamp"].dt.tz_convert(timezone).dt.date
    day_open = df.groupby(local_day, sort=False)["open"].transform("first")
    df["close_to_day_open_pct"] = px / day_open.replace(0, np.nan) - 1.0
    return df


def _add_trend_features(df: pd.DataFrame) -> pd.DataFrame:
    close = df["close"].replace(0, np.nan)

    for span in (12, 26, 60):
        ema = close.ewm(span=span, adjust=False).mean()
        df[f"ema_{span}_dist"] = close / ema - 1.0

    for window in (20, 60):
        sma = close.rolling(window, min_periods=window).mean()
        df[f"sma_{window}_dist"] = close / sma - 1.0

    delta = close.diff()
    gain = delta.clip(lower=0.0)
    loss = (-delta).clip(lower=0.0)
    for window in (7, 14):
        avg_gain = gain.ewm(alpha=1 / window, adjust=False).mean()
        avg_loss = loss.ewm(alpha=1 / window, adjust=False).mean()
        rs = avg_gain / (avg_loss + 1e-12)
        df[f"rsi_{window}"] = 100 - 100 / (1 + rs)

    df["momentum_12"] = close / close.shift(12) - 1.0
    df["momentum_24"] = close / close.shift(24) - 1.0
    return df


def _add_volatility_features(df: pd.DataFrame) -> pd.DataFrame:
    close = df["close"].replace(0, np.nan)
    prev_close = close.shift(1)

    tr = pd.concat(
        [
            df["high"] - df["low"],
            (df["high"] - prev_close).abs(),
            (df["low"] - prev_close).abs(),
        ],
        axis=1,
    ).max(axis=1)
    df["atr_14"] = tr.rolling(14, min_periods=14).mean()
    df["atr_14_pct"] = df["atr_14"] / close

    ret1 = (np.log(close) - np.log(prev_close)).replace([np.inf, -np.inf], np.nan)
    df["realized_vol_12"] = ret1.rolling(12, min_periods=12).std()
    df["realized_vol_24"] = ret1.rolling(24, min_periods=24).std()

    mid = close.rolling(20, min_periods=20).mean()
    std = close.rolling(20, min_periods=20).std()
    upper = mid + 2.0 * std
    lower = mid - 2.0 * std
    df["bb_width_20"] = (upper - lower) / mid.replace(0, np.nan)

    range_pct = (df["high"] - df["low"]) / close
    range_mean = range_pct.rolling(24, min_periods=24).mean()
    range_std = range_pct.rolling(24, min_periods=24).std()
    df["range_zscore_24"] = (range_pct - range_mean) / (range_std + 1e-12)
    return df


def _add_volume_features(df: pd.DataFrame) -> pd.DataFrame:
    volume = df["volume"].astype("float64")
    vol_mean_12 = volume.rolling(12, min_periods=12).mean()
    vol_mean_24 = volume.rolling(24, min_periods=24).mean()
    vol_std_24 = volume.rolling(24, min_periods=24).std()

    df["volume_rel_12"] = volume / vol_mean_12 - 1.0
    df["volume_rel_24"] = volume / vol_mean_24 - 1.0
    df["volume_zscore_24"] = (volume - vol_mean_24) / (vol_std_24 + 1e-12)
    df["turnover_proxy"] = df["close"] * volume
    df["volume_price_trend_component"] = df["log_ret_1"] * df["volume_rel_12"]
    return df


def _add_calendar_features(df: pd.DataFrame, *, timezone: str) -> pd.DataFrame:
    local = df["timestamp"].dt.tz_convert(timezone)
    minute_of_day = local.dt.hour * 60 + local.dt.minute
    day_of_week = local.dt.dayofweek

    session_open = 10 * 60
    session_close = 18 * 60 + 45
    evening_start = 19 * 60
    evening_end = 23 * 60 + 50

    minute_of_session = (minute_of_day - session_open).clip(lower=0)
    angle_session = 2.0 * np.pi * (minute_of_session / (14 * 60))
    angle_dow = 2.0 * np.pi * (day_of_week / 7.0)

    df["minute_of_session_sin"] = np.sin(angle_session)
    df["minute_of_session_cos"] = np.cos(angle_session)
    df["day_of_week_sin"] = np.sin(angle_dow)
    df["day_of_week_cos"] = np.cos(angle_dow)
    df["is_opening_window"] = (minute_of_day <= session_open + 30).astype("int8")
    df["is_closing_window"] = (minute_of_day >= session_close - 30).astype("int8")
    df["is_evening_session"] = (
        (minute_of_day >= evening_start) & (minute_of_day <= evening_end)
    ).astype("int8")
    return df


def _merge_factor_frames(
    df: pd.DataFrame,
    *,
    factor_frames: Mapping[str, pd.DataFrame],
    tolerance: pd.Timedelta,
    ffill_limit: int,
) -> pd.DataFrame:
    out = df.copy()
    out["timestamp"] = _to_utc_ns(out["timestamp"])
    for alias, frame in factor_frames.items():
        if frame.empty:
            continue
        factor = frame.copy()
        factor["timestamp"] = _to_utc_ns(factor["timestamp"])
        factor = factor.sort_values("timestamp").reset_index(drop=True)
        factor = factor[["timestamp", "close"]].rename(columns={"close": f"{alias}_close"})
        out = pd.merge_asof(out, factor, on="timestamp", direction="backward", tolerance=tolerance)
        out[f"{alias}_close"] = out[f"{alias}_close"].ffill(limit=ffill_limit)
    return out


def _to_utc_ns(values: pd.Series) -> pd.Series:
    return pd.to_datetime(values, utc=True).astype("datetime64[ns, UTC]")


def _add_cross_asset_features(df: pd.DataFrame, *, aliases: tuple[str, ...]) -> pd.DataFrame:
    for alias in aliases:
        close_col = f"{alias}_close"
        if close_col not in df.columns:
            continue
        factor_close = df[close_col].replace(0, np.nan)
        df[f"{alias}_ret_1"] = factor_close / factor_close.shift(1) - 1.0
        df[f"{alias}_ret_6"] = factor_close / factor_close.shift(6) - 1.0

    if "rtsi_close" in df.columns:
        df["asset_vs_rtsi_rel_strength_12"] = df["ret_12"] - (
            df["rtsi_close"] / df["rtsi_close"].shift(12) - 1.0
        )
    else:
        df["asset_vs_rtsi_rel_strength_12"] = np.nan

    if "brent_close" in df.columns:
        df["asset_vs_brent_rel_strength_12"] = df["ret_12"] - (
            df["brent_close"] / df["brent_close"].shift(12) - 1.0
        )
    else:
        df["asset_vs_brent_rel_strength_12"] = np.nan

    return df


def _add_regime_features(df: pd.DataFrame, *, aliases: tuple[str, ...]) -> pd.DataFrame:
    atr_median = df["atr_14_pct"].rolling(60, min_periods=20).median()
    df["vol_regime_flag"] = (df["atr_14_pct"] > atr_median).astype("int8")

    trend_strength = df["ema_60_dist"].abs()
    trend_median = trend_strength.rolling(60, min_periods=20).median()
    df["trend_regime_flag"] = (trend_strength > trend_median).astype("int8")

    factor_ret_cols = [f"{alias}_ret_1" for alias in aliases if f"{alias}_ret_1" in df.columns]
    if factor_ret_cols:
        factor_abs = df[factor_ret_cols].abs()
        df["market_stress_proxy"] = factor_abs.mean(axis=1)
        df["cross_asset_dispersion_proxy"] = df[factor_ret_cols].std(axis=1)
    else:
        df["market_stress_proxy"] = np.nan
        df["cross_asset_dispersion_proxy"] = np.nan

    return df
