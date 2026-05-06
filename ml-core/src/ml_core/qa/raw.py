from __future__ import annotations

from pathlib import Path
import json

import pandas as pd


TIMEFRAME_TO_DELTA = {
    "1m": pd.Timedelta(minutes=1),
    "2m": pd.Timedelta(minutes=2),
    "3m": pd.Timedelta(minutes=3),
    "5m": pd.Timedelta(minutes=5),
    "10m": pd.Timedelta(minutes=10),
    "15m": pd.Timedelta(minutes=15),
    "30m": pd.Timedelta(minutes=30),
    "1h": pd.Timedelta(hours=1),
    "2h": pd.Timedelta(hours=2),
    "4h": pd.Timedelta(hours=4),
    "1d": pd.Timedelta(days=1),
    "1w": pd.Timedelta(days=7),
    "1mo": pd.Timedelta(days=30),
}


def build_raw_qa_report(
    df: pd.DataFrame,
    *,
    timeframe: str,
    dataset_name: str,
    source: str,
) -> dict:
    report = {
        "dataset_name": dataset_name,
        "source": source,
        "timeframe": timeframe,
        "rows": int(len(df)),
        "duplicate_timestamps": 0,
        "is_monotonic": True,
        "gap_count": 0,
        "largest_gap_minutes": None,
        "null_counts": {},
        "range": {"min": None, "max": None},
        "status": "ok",
    }
    if df.empty:
        report["status"] = "empty"
        return report

    prepared = df.copy()
    prepared["timestamp"] = pd.to_datetime(prepared["timestamp"], utc=True)
    prepared = prepared.sort_values("timestamp").reset_index(drop=True)

    report["duplicate_timestamps"] = int(prepared["timestamp"].duplicated().sum())
    report["is_monotonic"] = bool(prepared["timestamp"].is_monotonic_increasing)
    report["latest_candle_timestamp"] = prepared["timestamp"].max().isoformat()
    report["range"] = {
        "min": prepared["timestamp"].min().isoformat(),
        "max": prepared["timestamp"].max().isoformat(),
    }
    report["null_counts"] = {
        column: int(prepared[column].isna().sum())
        for column in prepared.columns
        if prepared[column].isna().any()
    }

    expected_delta = TIMEFRAME_TO_DELTA.get(timeframe)
    if expected_delta is not None and len(prepared) > 1:
        diffs = prepared["timestamp"].diff().dropna()
        gaps = diffs[diffs > expected_delta]
        report["gap_count"] = int(len(gaps))
        if not gaps.empty:
            report["largest_gap_minutes"] = float(gaps.max() / pd.Timedelta(minutes=1))

        total_time = prepared["timestamp"].max() - prepared["timestamp"].min()
        expected_raw_bars = int(total_time / expected_delta) + 1
        if expected_raw_bars > 0:
            report["coverage_percentage"] = round((len(prepared) / expected_raw_bars) * 100.0, 2)
        else:
            report["coverage_percentage"] = 0.0

    if report["duplicate_timestamps"] > 0 or not report["is_monotonic"]:
        report["status"] = "error"
    elif report["gap_count"] > 0 or report["null_counts"]:
        report["status"] = "warning"

    return report


def write_raw_qa_report(
    report: dict,
    *,
    data_root: Path,
    dataset_kind: str,
    dataset_name: str,
    timeframe: str,
    source: str,
) -> Path:
    out_dir = (
        data_root
        / "_meta"
        / "qa"
        / "raw"
        / f"source={source}"
        / f"kind={dataset_kind}"
        / f"name={dataset_name}"
        / f"timeframe={timeframe}"
    )
    out_dir.mkdir(parents=True, exist_ok=True)
    out_path = out_dir / "report.json"
    out_path.write_text(json.dumps(report, indent=2, ensure_ascii=False, default=str), encoding="utf-8")
    return out_path
