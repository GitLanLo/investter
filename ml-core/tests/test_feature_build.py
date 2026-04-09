from __future__ import annotations

import pandas as pd

from ml_core.features.build import FeatureBuildConfig, build_feature_frame


def test_feature_build_adds_core_and_cross_asset_columns() -> None:
    asset_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=80, freq="5min"),
            "open": [100 + i * 0.1 for i in range(80)],
            "high": [100.2 + i * 0.1 for i in range(80)],
            "low": [99.8 + i * 0.1 for i in range(80)],
            "close": [100 + i * 0.1 for i in range(80)],
            "volume": [1000 + i for i in range(80)],
        }
    )
    factor_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=80, freq="5min"),
            "close": [90 + i * 0.05 for i in range(80)],
        }
    )

    out = build_feature_frame(
        asset_df,
        factor_frames={"usdrub": factor_df, "rtsi": factor_df, "brent": factor_df},
        ticker="SBER",
        config=FeatureBuildConfig(),
    )

    expected_columns = {
        "ret_1",
        "log_ret_1",
        "close_to_prev_close",
        "ema_12_dist",
        "atr_14_pct",
        "volume_rel_12",
        "minute_of_session_sin",
        "usdrub_ret_1",
        "asset_vs_rtsi_rel_strength_12",
        "vol_regime_flag",
        "feature_schema_version",
    }
    assert expected_columns.issubset(set(out.columns))
    assert (out["ticker"] == "SBER").all()
