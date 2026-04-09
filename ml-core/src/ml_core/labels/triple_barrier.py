from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import pandas as pd

from ml_core.contracts import TARGET_SCHEMA_VERSION, DEFAULT_HORIZON_BARS


@dataclass(slots=True)
class TripleBarrierConfig:
    horizon_bars: int = DEFAULT_HORIZON_BARS
    min_move_pct: float = 0.003
    atr_mult: float = 1.0
    atr_column: str = "atr_14"
    price_column: str = "close"
    high_column: str = "high"
    low_column: str = "low"
    target_schema_version: str = TARGET_SCHEMA_VERSION


def apply_triple_barrier_labels(
    df: pd.DataFrame,
    *,
    config: TripleBarrierConfig | None = None,
) -> pd.DataFrame:
    cfg = config or TripleBarrierConfig()
    required = {"timestamp", cfg.price_column, cfg.high_column, cfg.low_column, cfg.atr_column}
    missing = required - set(df.columns)
    if missing:
        raise ValueError(f"df is missing columns: {sorted(missing)}")

    out = df.copy()
    out["timestamp"] = pd.to_datetime(out["timestamp"], utc=True)
    out = out.sort_values("timestamp").reset_index(drop=True)

    price = out[cfg.price_column].astype("float64").to_numpy()
    high = out[cfg.high_column].astype("float64").to_numpy()
    low = out[cfg.low_column].astype("float64").to_numpy()
    atr = out[cfg.atr_column].astype("float64").to_numpy()

    move_pct = np.maximum(cfg.min_move_pct, cfg.atr_mult * atr / np.where(price == 0, np.nan, price))
    label_class: list[str | None] = []
    label_up: list[int | None] = []
    label_down: list[int | None] = []
    label_no_trade: list[int | None] = []
    barrier_up_price: list[float | None] = []
    barrier_down_price: list[float | None] = []
    label_horizon_end_time: list[pd.Timestamp | None] = []
    label_is_complete: list[bool] = []

    for idx in range(len(out)):
        start = idx + 1
        end = idx + 1 + cfg.horizon_bars
        if end > len(out):
            label_class.append(None)
            label_up.append(None)
            label_down.append(None)
            label_no_trade.append(None)
            barrier_up_price.append(None)
            barrier_down_price.append(None)
            label_horizon_end_time.append(pd.NaT)
            label_is_complete.append(False)
            continue

        entry = price[idx]
        current_move_pct = move_pct[idx]
        upper = entry * (1.0 + current_move_pct)
        lower = entry * (1.0 - current_move_pct)

        cls = "no_trade"
        for future_idx in range(start, end):
            hit_up = high[future_idx] >= upper
            hit_down = low[future_idx] <= lower

            if hit_up and hit_down:
                cls = "no_trade"
                break
            if hit_up:
                cls = "up_signal"
                break
            if hit_down:
                cls = "down_signal"
                break

        label_class.append(cls)
        label_up.append(1 if cls == "up_signal" else 0)
        label_down.append(1 if cls == "down_signal" else 0)
        label_no_trade.append(1 if cls == "no_trade" else 0)
        barrier_up_price.append(float(upper))
        barrier_down_price.append(float(lower))
        label_horizon_end_time.append(out.loc[end - 1, "timestamp"])
        label_is_complete.append(True)

    out["move_pct"] = move_pct
    out["horizon_bars"] = cfg.horizon_bars
    out["barrier_up_price"] = barrier_up_price
    out["barrier_down_price"] = barrier_down_price
    out["label_asof_time"] = out["timestamp"]
    out["label_horizon_end_time"] = label_horizon_end_time
    out["label_is_complete"] = label_is_complete
    out["label_class"] = pd.Series(label_class, dtype="string")
    out["label_up"] = pd.Series(label_up, dtype="Int8")
    out["label_down"] = pd.Series(label_down, dtype="Int8")
    out["label_no_trade"] = pd.Series(label_no_trade, dtype="Int8")
    out["target_schema_version"] = cfg.target_schema_version
    return out

