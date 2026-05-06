from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
import json
import statistics

import pandas as pd

from ml_core.ingest.providers import MarketDataProvider
from ml_core.labels.triple_barrier import TripleBarrierConfig
from ml_core.pipelines.research import ResearchPipelineConfig, run_research_pipeline
from ml_core.datasets.splits import SplitConfig
from ml_core.features.build import FeatureBuildConfig
from ml_core.training.research import BaselineResearchConfig

@dataclass(slots=True)
class ResearchGridConfig:
    data_root: Path
    dataset_output_root: Path
    research_output_root: Path
    grid_name: str
    tickers: list[str]
    factor_aliases: list[str] = field(default_factory=list)
    timeframes: list[str] = field(default_factory=lambda: ["5m", "15m", "1h"])
    horizons: list[int] = field(default_factory=lambda: [6, 12, 24])
    start: pd.Timestamp | None = None
    end: pd.Timestamp | None = None
    decision_threshold: float = 0.65

def run_research_grid(
    provider: MarketDataProvider,
    *,
    config: ResearchGridConfig,
) -> dict:
    if not config.tickers:
        raise ValueError("tickers must not be empty")

    results = []
    failures = []
    matrix_output_dir = config.research_output_root
    run_output_dir = matrix_output_dir / config.grid_name
    matrix_output_dir.mkdir(parents=True, exist_ok=True)
    run_output_dir.mkdir(parents=True, exist_ok=True)

    timestamp_str = pd.Timestamp.now(tz="UTC").strftime("%Y%m%d")

    for tf in config.timeframes:
        for horizon in config.horizons:
            dataset_version = f"sprint7_{tf}_h{horizon}_{timestamp_str}"
            run_output_root = run_output_dir / f"{tf}_h{horizon}"

            pipeline_config = ResearchPipelineConfig(
                data_root=config.data_root,
                dataset_output_root=config.dataset_output_root,
                research_output_root=run_output_root,
                dataset_version=dataset_version,
                tickers=config.tickers,
                timeframe=tf,
                factor_aliases=config.factor_aliases,
                start=config.start,
                end=config.end,
                feature_config=FeatureBuildConfig(timeframe=tf),
                label_config=TripleBarrierConfig(horizon_bars=horizon),
                split_config=SplitConfig(
                    dataset_version=dataset_version,
                    timeframe=tf,
                    horizon_bars=horizon,
                ),
                research_config=BaselineResearchConfig(
                    output_root=run_output_root,
                    decision_threshold=config.decision_threshold,
                ),
                run_walk_forward=False, # Save time in grid search
                run_ablation=False,
            )

            try:
                summary = run_research_pipeline(provider, config=pipeline_config)
                results.append(_grid_result_from_pipeline_summary(summary, timeframe=tf, horizon=horizon))
            except Exception as exc:
                failed = {
                    "status": "failed",
                    "timeframe": tf,
                    "horizon": horizon,
                    "dataset_version": dataset_version,
                    "error": str(exc),
                }
                results.append(failed)
                failures.append(failed)

    matrix_file = matrix_output_dir / "timeframe_horizon_matrix.json"

    final_report = {
        "status": "failed" if failures else "completed",
        "grid_name": config.grid_name,
        "timeframes": config.timeframes,
        "horizons": config.horizons,
        "expected_result_count": len(config.timeframes) * len(config.horizons),
        "completed_result_count": len(results) - len(failures),
        "failure_count": len(failures),
        "failures": failures,
        "selected_configuration": _select_configuration(results),
        "results": results,
    }

    matrix_file.write_text(json.dumps(final_report, indent=2, ensure_ascii=False, default=str), encoding="utf-8")
    if failures:
        raise RuntimeError(f"research grid failed for {len(failures)} combination(s); see {matrix_file}")

    return final_report


def _grid_result_from_pipeline_summary(summary: dict, *, timeframe: str, horizon: int) -> dict:
    manifest = summary["dataset_manifest"]
    research_summary = summary["research_summary"]
    prod_candidate = research_summary.get("production_candidate", {})
    val_metrics = prod_candidate.get("validation", {})
    test_metrics = prod_candidate.get("test", {})
    train_rows = _range_rows(manifest, "train_range")
    val_rows = _range_rows(manifest, "val_range")
    test_rows = _range_rows(manifest, "test_range")
    model_name = prod_candidate.get("model_name")
    per_ticker = _per_ticker_metrics(research_summary, model_name)

    return {
        "status": "completed",
        "timeframe": timeframe,
        "horizon": horizon,
        "dataset_version": manifest.get("dataset_version", summary.get("dataset_version", "")),
        "model_name": model_name,
        "selection_mode": prod_candidate.get("selection_mode"),
        "selected_threshold": prod_candidate.get("selected_threshold"),
        "rows": train_rows + val_rows + test_rows,
        "train_rows": train_rows,
        "val_rows": val_rows,
        "test_rows": test_rows,
        "tickers": manifest.get("tickers", []),
        "val_actionable_f1": val_metrics.get("actionable_f1"),
        "val_precision": val_metrics.get("precision_actionable_signal"),
        "val_coverage": val_metrics.get("signal_coverage"),
        "val_ece": val_metrics.get("actionable_expected_calibration_error"),
        "test_actionable_f1": test_metrics.get("actionable_f1"),
        "test_precision": test_metrics.get("precision_actionable_signal"),
        "test_coverage": test_metrics.get("signal_coverage"),
        "test_ece": test_metrics.get("actionable_expected_calibration_error"),
        **_per_ticker_stability(per_ticker),
    }


def _range_rows(manifest: dict, key: str) -> int:
    value = manifest.get(key, {})
    if isinstance(value, dict):
        return int(value.get("rows", 0) or 0)
    return 0


def _per_ticker_metrics(research_summary: dict, model_name: object) -> dict:
    if not isinstance(model_name, str):
        return {}
    model_report = research_summary.get("models", {}).get(model_name, {})
    validation = model_report.get("validation", {})
    per_ticker = validation.get("per_ticker", {})
    return per_ticker if isinstance(per_ticker, dict) else {}


def _per_ticker_stability(per_ticker: dict) -> dict:
    values = [
        float(metrics.get("actionable_f1"))
        for metrics in per_ticker.values()
        if isinstance(metrics, dict) and metrics.get("actionable_f1") is not None
    ]
    if not values:
        return {
            "val_per_ticker_actionable_f1_min": None,
            "val_per_ticker_actionable_f1_max": None,
            "val_per_ticker_actionable_f1_std": None,
        }
    return {
        "val_per_ticker_actionable_f1_min": min(values),
        "val_per_ticker_actionable_f1_max": max(values),
        "val_per_ticker_actionable_f1_std": statistics.pstdev(values) if len(values) > 1 else 0.0,
    }


def _select_configuration(results: list[dict]) -> dict | None:
    completed = [item for item in results if item.get("status") == "completed"]
    if not completed:
        return None

    return max(
        completed,
        key=lambda item: (
            _metric(item, "val_actionable_f1"),
            -_metric(item, "val_ece"),
            _metric(item, "val_precision"),
            _metric(item, "val_coverage"),
            -_metric(item, "val_per_ticker_actionable_f1_std"),
        ),
    )


def _metric(item: dict, key: str) -> float:
    value = item.get(key)
    if value is None:
        return 0.0
    return float(value)
