from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
import json

import pandas as pd

from ml_core.datasets.splits import SplitConfig
from ml_core.features.build import FeatureBuildConfig
from ml_core.ingest.providers import MarketDataProvider
from ml_core.labels.triple_barrier import TripleBarrierConfig
from ml_core.pipelines.materialize import (
    FeatureStoreMaterializationConfig,
    materialize_dataset,
    materialize_feature_store,
)
from ml_core.training.research import (
    BaselineResearchConfig,
    WalkForwardConfig,
    run_baseline_research,
    run_walk_forward_research,
)


@dataclass(slots=True)
class ResearchPipelineConfig:
    data_root: Path
    dataset_output_root: Path
    research_output_root: Path
    dataset_version: str
    tickers: list[str]
    timeframe: str = "5m"
    factor_aliases: list[str] = field(default_factory=list)
    start: pd.Timestamp | None = None
    end: pd.Timestamp | None = None
    feature_config: FeatureBuildConfig = field(default_factory=FeatureBuildConfig)
    label_config: TripleBarrierConfig = field(default_factory=TripleBarrierConfig)
    split_config: SplitConfig | None = None
    research_config: BaselineResearchConfig | None = None
    run_walk_forward: bool = True
    walk_forward_config: WalkForwardConfig | None = None


def run_research_pipeline(
    provider: MarketDataProvider,
    *,
    config: ResearchPipelineConfig,
) -> dict:
    if not config.tickers:
        raise ValueError("tickers must not be empty")

    materializations: list[dict] = []
    for ticker in config.tickers:
        materializations.append(
            materialize_feature_store(
                provider,
                config=FeatureStoreMaterializationConfig(
                    data_root=config.data_root,
                    ticker=ticker,
                    timeframe=config.timeframe,
                    factor_aliases=config.factor_aliases,
                    start=config.start,
                    end=config.end,
                    feature_config=config.feature_config,
                    label_config=config.label_config,
                ),
            )
        )

    split_config = config.split_config or SplitConfig(
        dataset_version=config.dataset_version,
        timeframe=config.timeframe,
        horizon_bars=config.label_config.horizon_bars,
        feature_schema_version=config.feature_config.feature_schema_version,
        target_schema_version=config.label_config.target_schema_version,
    )
    split_config.dataset_version = config.dataset_version

    dataset_manifest = materialize_dataset(
        data_root=config.data_root,
        output_root=config.dataset_output_root,
        dataset_version=config.dataset_version,
        tickers=config.tickers,
        split_config=split_config,
        schema_version=config.feature_config.feature_schema_version,
    )

    dataset_root = config.dataset_output_root / f"dataset_version={config.dataset_version}"
    research_config = config.research_config or BaselineResearchConfig(output_root=config.research_output_root)
    research_config.output_root.mkdir(parents=True, exist_ok=True)
    research_summary = run_baseline_research(dataset_root, config=research_config)
    walk_forward_summary = None
    if config.run_walk_forward:
        walk_forward_config = config.walk_forward_config or WalkForwardConfig(
            output_root=config.research_output_root / "walk_forward",
            decision_threshold=research_config.decision_threshold,
            random_state=research_config.random_state,
            purge_gap_bars=split_config.purge_gap_bars,
        )
        walk_forward_summary = run_walk_forward_research(dataset_root, config=walk_forward_config)

    pipeline_summary = {
        "dataset_version": config.dataset_version,
        "timeframe": config.timeframe,
        "tickers": config.tickers,
        "factor_aliases": config.factor_aliases,
        "dataset_root": str(dataset_root),
        "research_output_root": str(research_config.output_root),
        "feature_store_materializations": materializations,
        "dataset_manifest": dataset_manifest,
        "research_summary": research_summary,
        "walk_forward_summary": walk_forward_summary,
    }
    (research_config.output_root / "pipeline_summary.json").write_text(
        json.dumps(pipeline_summary, indent=2, ensure_ascii=False, default=str),
        encoding="utf-8",
    )
    return pipeline_summary
