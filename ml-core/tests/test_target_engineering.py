from __future__ import annotations

import pandas as pd

from ml_core.labels.triple_barrier import TripleBarrierConfig, apply_triple_barrier_labels


def test_triple_barrier_assigns_up_signal_before_down_signal() -> None:
    df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=5, freq="5min"),
            "open": [100, 100, 100, 100, 100],
            "high": [100, 101.0, 100.2, 100.2, 100.2],
            "low": [100, 99.8, 99.8, 99.8, 99.8],
            "close": [100, 100.5, 100.2, 100.1, 100.0],
            "volume": [1, 1, 1, 1, 1],
            "atr_14": [0.1, 0.1, 0.1, 0.1, 0.1],
        }
    )

    out = apply_triple_barrier_labels(
        df,
        config=TripleBarrierConfig(horizon_bars=2, min_move_pct=0.005, atr_mult=0.0),
    )

    assert out.loc[0, "label_class"] == "up_signal"
    assert int(out.loc[0, "label_up"]) == 1
    assert int(out.loc[0, "label_down"]) == 0


def test_triple_barrier_marks_ambiguous_bar_as_no_trade() -> None:
    df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=4, freq="5min"),
            "open": [100, 100, 100, 100],
            "high": [100, 101.0, 100.1, 100.1],
            "low": [100, 99.0, 99.9, 99.9],
            "close": [100, 100.0, 100.0, 100.0],
            "volume": [1, 1, 1, 1],
            "atr_14": [0.1, 0.1, 0.1, 0.1],
        }
    )

    out = apply_triple_barrier_labels(
        df,
        config=TripleBarrierConfig(horizon_bars=2, min_move_pct=0.005, atr_mult=0.0),
    )

    assert out.loc[0, "label_class"] == "no_trade"
    assert int(out.loc[0, "label_no_trade"]) == 1


def test_triple_barrier_marks_incomplete_tail_as_missing() -> None:
    df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=3, freq="5min"),
            "open": [100, 100, 100],
            "high": [100, 100, 100],
            "low": [100, 100, 100],
            "close": [100, 100, 100],
            "volume": [1, 1, 1],
            "atr_14": [0.1, 0.1, 0.1],
        }
    )

    out = apply_triple_barrier_labels(
        df,
        config=TripleBarrierConfig(horizon_bars=2, min_move_pct=0.005, atr_mult=0.0),
    )

    assert pd.isna(out.loc[1, "label_class"])
    assert bool(out.loc[1, "label_is_complete"]) is False

