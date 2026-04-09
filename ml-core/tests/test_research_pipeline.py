from __future__ import annotations

from pathlib import Path

import pandas as pd

from ml_core.ingest.providers import LocalParquetProvider
from ml_core.pipelines.research import ResearchPipelineConfig, run_research_pipeline
from ml_core.storage.layouts import ingest_asset_frame, ingest_factor_frame


def test_run_research_pipeline_materializes_dataset_and_report(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    dataset_output_root = tmp_path / "datasets"
    research_output_root = tmp_path / "research"
    closes = [100 + ((i % 12) - 6) * 0.4 + (i // 12) * 0.1 for i in range(120)]
    factor_closes = [90 + ((i % 10) - 5) * 0.2 + (i // 10) * 0.05 for i in range(120)]

    asset_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=120, freq="5min"),
            "open": closes,
            "high": [value + 0.3 for value in closes],
            "low": [value - 0.3 for value in closes],
            "close": closes,
            "volume": [1000 + i for i in range(120)],
        }
    )
    factor_df = pd.DataFrame(
        {
            "timestamp": pd.date_range("2026-01-01T10:00:00Z", periods=120, freq="5min"),
            "open": factor_closes,
            "high": [value + 0.2 for value in factor_closes],
            "low": [value - 0.2 for value in factor_closes],
            "close": factor_closes,
            "volume": [500 + i for i in range(120)],
        }
    )

    ingest_asset_frame(asset_df, data_root=data_root, ticker="SBER", timeframe="5m", source="unit_test")
    ingest_factor_frame(factor_df, data_root=data_root, alias="usdrub", timeframe="5m", source="unit_test")

    summary = run_research_pipeline(
        LocalParquetProvider(root=data_root),
        config=ResearchPipelineConfig(
            data_root=data_root,
            dataset_output_root=dataset_output_root,
            research_output_root=research_output_root,
            dataset_version="pipeline_v1",
            tickers=["SBER"],
            timeframe="5m",
            factor_aliases=["usdrub"],
            run_ablation=True,
        ),
    )

    assert summary["dataset_version"] == "pipeline_v1"
    assert summary["research_summary"]["best_model"] in {"logreg_multiclass", "rf_multiclass"}
    assert summary["research_summary"]["production_candidate"]["model_name"] == summary["research_summary"]["best_model"]
    assert summary["walk_forward_summary"] is not None
    assert summary["walk_forward_summary"]["production_candidate"]["model_name"] == summary["walk_forward_summary"]["best_model"]
    assert (dataset_output_root / "dataset_version=pipeline_v1" / "manifest.json").exists()
    assert (research_output_root / "summary.json").exists()
    assert (research_output_root / "report.md").exists()
    assert (research_output_root / "pipeline_summary.json").exists()
    assert (research_output_root / "walk_forward" / "summary.json").exists()
    assert summary["ablation_summary"] is not None
    assert summary["ablation_summary"]["production_candidate"]["scenario_name"] == summary["ablation_summary"]["best_scenario"]
    assert (research_output_root / "ablation" / "summary.json").exists()
