from __future__ import annotations

import argparse
from pathlib import Path
import json

import pandas as pd

from ml_core.datasets.splits import SplitConfig, write_dataset_splits
from ml_core.datasets.sequences import SequenceDatasetConfig, build_sequence_dataset
from ml_core.features.build import FeatureBuildConfig, build_feature_frame
from ml_core.ingest.providers import LocalParquetProvider
from ml_core.ingest.tinkoff import sync_tinkoff_asset_history, sync_tinkoff_factor_history, sync_tinkoff_universe
from ml_core.labels.triple_barrier import TripleBarrierConfig, apply_triple_barrier_labels
from ml_core.pipelines.materialize import (
    FeatureStoreMaterializationConfig,
    materialize_dataset,
    materialize_feature_store,
)
from ml_core.pipelines.research import ResearchPipelineConfig, run_research_pipeline
from ml_core.pipelines.research_grid import ResearchGridConfig, run_research_grid
from ml_core.qa.coverage import generate_coverage_report, write_coverage_report
from ml_core.storage.layouts import ingest_asset_frame, ingest_factor_frame
from ml_core.training.baselines import BaselineTrainingConfig, train_baseline_pack
from ml_core.training.calibration import (
    CalibrationAuditConfig,
    DEFAULT_CALIBRATION_METHODS,
    DEFAULT_CALIBRATION_THRESHOLD_GRID,
    run_saved_model_calibration_audit,
)
from ml_core.training.research import (
    AblationResearchConfig,
    BaselineResearchConfig,
    DEFAULT_THRESHOLD_GRID,
    WalkForwardConfig,
    run_ablation_research,
    run_baseline_research,
    run_walk_forward_research,
)
from ml_core.training.neural import NeuralTrainingConfig, train_neural_sequence


def main() -> None:
    parser = argparse.ArgumentParser(prog="invest-ml")
    subparsers = parser.add_subparsers(dest="command", required=True)

    build_features = subparsers.add_parser("build-features")
    build_features.add_argument("--asset-parquet", required=True)
    build_features.add_argument("--output", required=True)
    build_features.add_argument("--ticker", required=True)
    build_features.add_argument("--factor", action="append", default=[], help="alias=path/to/parquet")

    apply_labels = subparsers.add_parser("apply-labels")
    apply_labels.add_argument("--input-parquet", required=True)
    apply_labels.add_argument("--output", required=True)
    apply_labels.add_argument("--horizon-bars", type=int, default=12)
    apply_labels.add_argument("--min-move-pct", type=float, default=0.003)
    apply_labels.add_argument("--atr-mult", type=float, default=1.0)

    split_dataset = subparsers.add_parser("split-dataset")
    split_dataset.add_argument("--input-parquet", required=True)
    split_dataset.add_argument("--output-root", required=True)
    split_dataset.add_argument("--dataset-version", required=True)
    split_dataset.add_argument("--train-ratio", type=float, default=0.7)
    split_dataset.add_argument("--val-ratio", type=float, default=0.15)
    split_dataset.add_argument("--purge-gap-bars", type=int, default=12)

    train_baselines = subparsers.add_parser("train-baselines")
    train_baselines.add_argument("--train-parquet", required=True)
    train_baselines.add_argument("--val-parquet", required=True)
    train_baselines.add_argument("--output-root", required=True)

    run_research = subparsers.add_parser("run-baseline-research")
    run_research.add_argument("--dataset-root", required=True)
    run_research.add_argument("--output-root", required=True)
    run_research.add_argument("--decision-threshold", type=float, default=0.65)

    run_walk_forward = subparsers.add_parser("run-walk-forward-research")
    run_walk_forward.add_argument("--dataset-root", required=True)
    run_walk_forward.add_argument("--output-root", required=True)
    run_walk_forward.add_argument("--decision-threshold", type=float, default=0.65)
    run_walk_forward.add_argument("--initial-train-ratio", type=float, default=0.6)
    run_walk_forward.add_argument("--validation-ratio", type=float, default=0.1)
    run_walk_forward.add_argument("--step-ratio", type=float, default=0.05)
    run_walk_forward.add_argument("--purge-gap-bars", type=int, default=12)
    run_walk_forward.add_argument("--min-train-timestamps", type=int, default=60)
    run_walk_forward.add_argument("--min-validation-timestamps", type=int, default=24)
    run_walk_forward.add_argument("--max-folds", type=int, default=6)

    run_ablation = subparsers.add_parser("run-ablation-research")
    run_ablation.add_argument("--dataset-root", required=True)
    run_ablation.add_argument("--output-root", required=True)
    run_ablation.add_argument("--decision-threshold", type=float, default=0.65)
    run_ablation.add_argument("--threshold", action="append", type=float, default=[])

    run_calibration_audit = subparsers.add_parser("run-calibration-audit")
    run_calibration_audit.add_argument("--model-dir", required=True)
    run_calibration_audit.add_argument("--output-root", required=True)
    run_calibration_audit.add_argument("--decision-threshold", type=float)
    run_calibration_audit.add_argument("--threshold", action="append", type=float, default=[])
    run_calibration_audit.add_argument("--method", action="append", default=[])
    run_calibration_audit.add_argument("--calibration-bins", type=int, default=8)
    run_calibration_audit.add_argument("--min-fit-rows", type=int, default=20)

    run_research_pipeline_cmd = subparsers.add_parser("run-research-pipeline")
    run_research_pipeline_cmd.add_argument("--data-root", required=True)
    run_research_pipeline_cmd.add_argument("--dataset-output-root", required=True)
    run_research_pipeline_cmd.add_argument("--research-output-root", required=True)
    run_research_pipeline_cmd.add_argument("--dataset-version", required=True)
    run_research_pipeline_cmd.add_argument("--ticker", action="append", required=True)
    run_research_pipeline_cmd.add_argument("--timeframe", default="5m")
    run_research_pipeline_cmd.add_argument("--factor", action="append", default=[])
    run_research_pipeline_cmd.add_argument("--start")
    run_research_pipeline_cmd.add_argument("--end")
    run_research_pipeline_cmd.add_argument("--train-ratio", type=float, default=0.7)
    run_research_pipeline_cmd.add_argument("--val-ratio", type=float, default=0.15)
    run_research_pipeline_cmd.add_argument("--purge-gap-bars", type=int, default=12)
    run_research_pipeline_cmd.add_argument("--horizon-bars", type=int, default=12)
    run_research_pipeline_cmd.add_argument("--min-move-pct", type=float, default=0.003)
    run_research_pipeline_cmd.add_argument("--atr-mult", type=float, default=1.0)
    run_research_pipeline_cmd.add_argument("--decision-threshold", type=float, default=0.65)
    run_research_pipeline_cmd.add_argument("--no-walk-forward", action="store_true")
    run_research_pipeline_cmd.add_argument("--wf-initial-train-ratio", type=float, default=0.6)
    run_research_pipeline_cmd.add_argument("--wf-validation-ratio", type=float, default=0.1)
    run_research_pipeline_cmd.add_argument("--wf-step-ratio", type=float, default=0.05)
    run_research_pipeline_cmd.add_argument("--wf-max-folds", type=int, default=6)
    run_research_pipeline_cmd.add_argument("--run-ablation", action="store_true")
    run_research_pipeline_cmd.add_argument("--ablation-threshold", action="append", type=float, default=[])

    run_research_grid_cmd = subparsers.add_parser("run-research-grid")
    run_research_grid_cmd.add_argument("--data-root", required=True)
    run_research_grid_cmd.add_argument("--dataset-output-root", required=True)
    run_research_grid_cmd.add_argument("--research-output-root", required=True)
    run_research_grid_cmd.add_argument("--grid-name", required=True)
    run_research_grid_cmd.add_argument("--ticker", action="append", required=True)
    run_research_grid_cmd.add_argument("--factor", action="append", default=[])
    run_research_grid_cmd.add_argument("--timeframe", action="append", default=[])
    run_research_grid_cmd.add_argument("--horizon", action="append", type=int, default=[])
    run_research_grid_cmd.add_argument("--start")
    run_research_grid_cmd.add_argument("--end")
    run_research_grid_cmd.add_argument("--decision-threshold", type=float, default=0.65)

    ingest_asset = subparsers.add_parser("ingest-asset-parquet")
    ingest_asset.add_argument("--input-parquet", required=True)
    ingest_asset.add_argument("--data-root", required=True)
    ingest_asset.add_argument("--ticker", required=True)
    ingest_asset.add_argument("--timeframe", default="5m")
    ingest_asset.add_argument("--source", default="local_parquet")

    ingest_factor = subparsers.add_parser("ingest-factor-parquet")
    ingest_factor.add_argument("--input-parquet", required=True)
    ingest_factor.add_argument("--data-root", required=True)
    ingest_factor.add_argument("--alias", required=True)
    ingest_factor.add_argument("--timeframe", default="5m")
    ingest_factor.add_argument("--source", default="local_parquet")

    materialize_features = subparsers.add_parser("materialize-feature-store")
    materialize_features.add_argument("--data-root", required=True)
    materialize_features.add_argument("--ticker", required=True)
    materialize_features.add_argument("--timeframe", default="5m")
    materialize_features.add_argument("--factor", action="append", default=[])
    materialize_features.add_argument("--start")
    materialize_features.add_argument("--end")
    materialize_features.add_argument("--horizon-bars", type=int, default=12)
    materialize_features.add_argument("--min-move-pct", type=float, default=0.003)
    materialize_features.add_argument("--atr-mult", type=float, default=1.0)

    materialize_dataset_cmd = subparsers.add_parser("materialize-dataset")
    materialize_dataset_cmd.add_argument("--data-root", required=True)
    materialize_dataset_cmd.add_argument("--output-root", required=True)
    materialize_dataset_cmd.add_argument("--dataset-version", required=True)
    materialize_dataset_cmd.add_argument("--ticker", action="append", required=True)
    materialize_dataset_cmd.add_argument("--timeframe", default="5m")
    materialize_dataset_cmd.add_argument("--horizon-bars", type=int, default=12)
    materialize_dataset_cmd.add_argument("--train-ratio", type=float, default=0.7)
    materialize_dataset_cmd.add_argument("--val-ratio", type=float, default=0.15)
    materialize_dataset_cmd.add_argument("--purge-gap-bars", type=int, default=12)

    build_sequence = subparsers.add_parser("build-sequence-dataset")
    build_sequence.add_argument("--data-root", required=True)
    build_sequence.add_argument("--output-root", required=True)
    build_sequence.add_argument("--source-dataset-version", required=True)
    build_sequence.add_argument("--sequence-dataset-version", required=True)
    build_sequence.add_argument("--timeframe", required=True)
    build_sequence.add_argument("--horizon-bars", type=int, required=True)
    build_sequence.add_argument("--window-bars", type=int, required=True)
    build_sequence.add_argument("--stride", type=int, default=1)
    build_sequence.add_argument("--min-coverage-ratio", type=float, default=0.9)
    build_sequence.add_argument("--ticker", action="append", default=[])
    build_sequence.add_argument("--feature-col", action="append", default=[])
    build_sequence.add_argument("--exclude-feature", action="append", default=[])

    train_neural = subparsers.add_parser("train-neural-sequence")
    train_neural.add_argument("--sequence-root", required=True)
    train_neural.add_argument("--dataset-version", required=True)
    train_neural.add_argument("--window", type=int, required=True)
    train_neural.add_argument("--model-family", default="gru")
    train_neural.add_argument("--output-root", required=True)
    train_neural.add_argument("--epochs", type=int, default=50)
    train_neural.add_argument("--batch-size", type=int, default=64)
    train_neural.add_argument("--lr", type=float, default=1e-3)
    train_neural.add_argument("--hidden-size", type=int, default=64)
    train_neural.add_argument("--num-layers", type=int, default=2)
    train_neural.add_argument("--dropout", type=float, default=0.2)
    train_neural.add_argument("--patience", type=int, default=5)
    train_neural.add_argument("--device", default="cpu")

    tinkoff_asset = subparsers.add_parser("tinkoff-sync-asset")
    tinkoff_asset.add_argument("--data-root", required=True)
    tinkoff_asset.add_argument("--ticker", required=True)
    tinkoff_asset.add_argument("--from", dest="from_time", required=True)
    tinkoff_asset.add_argument("--to", dest="to_time", required=True)
    tinkoff_asset.add_argument("--timeframe", default="5m")
    tinkoff_asset.add_argument("--instrument-kind", default="share")
    tinkoff_asset.add_argument("--class-code")

    tinkoff_factor = subparsers.add_parser("tinkoff-sync-factor")
    tinkoff_factor.add_argument("--data-root", required=True)
    tinkoff_factor.add_argument("--alias", required=True)
    tinkoff_factor.add_argument("--ticker", required=True)
    tinkoff_factor.add_argument("--from", dest="from_time", required=True)
    tinkoff_factor.add_argument("--to", dest="to_time", required=True)
    tinkoff_factor.add_argument("--timeframe", default="5m")
    tinkoff_factor.add_argument("--instrument-kind", required=True)
    tinkoff_factor.add_argument("--class-code")
    tinkoff_factor.add_argument("--no-incremental", action="store_true")
    tinkoff_factor.add_argument("--overlap-bars", type=int, default=3)

    tinkoff_asset.add_argument("--no-incremental", action="store_true")
    tinkoff_asset.add_argument("--overlap-bars", type=int, default=3)

    tinkoff_universe = subparsers.add_parser("tinkoff-sync-universe")
    tinkoff_universe.add_argument("--config", required=True)
    tinkoff_universe.add_argument("--data-root", required=True)
    tinkoff_universe.add_argument("--from", dest="from_time", required=True)
    tinkoff_universe.add_argument("--to", dest="to_time", required=True)
    tinkoff_universe.add_argument("--no-incremental", action="store_true")
    tinkoff_universe.add_argument("--overlap-bars", type=int, default=3)
    tinkoff_universe.add_argument("--timeframe", help="override timeframe for all assets/factors")

    raw_coverage = subparsers.add_parser("raw-coverage-report")
    raw_coverage.add_argument("--data-root", required=True)
    raw_coverage.add_argument("--output", required=True)

    args = parser.parse_args()

    if args.command == "build-features":
        asset_df = pd.read_parquet(args.asset_parquet)
        factor_frames: dict[str, pd.DataFrame] = {}
        for item in args.factor:
            alias, path = item.split("=", 1)
            factor_frames[alias] = pd.read_parquet(path)

        out = build_feature_frame(
            asset_df,
            factor_frames=factor_frames,
            ticker=args.ticker,
            config=FeatureBuildConfig(),
        )
        Path(args.output).parent.mkdir(parents=True, exist_ok=True)
        out.to_parquet(args.output, index=False)
        return

    if args.command == "apply-labels":
        df = pd.read_parquet(args.input_parquet)
        out = apply_triple_barrier_labels(
            df,
            config=TripleBarrierConfig(
                horizon_bars=args.horizon_bars,
                min_move_pct=args.min_move_pct,
                atr_mult=args.atr_mult,
            ),
        )
        Path(args.output).parent.mkdir(parents=True, exist_ok=True)
        out.to_parquet(args.output, index=False)
        return

    if args.command == "split-dataset":
        df = pd.read_parquet(args.input_parquet)
        write_dataset_splits(
            df,
            output_root=Path(args.output_root),
            config=SplitConfig(
                dataset_version=args.dataset_version,
                train_ratio=args.train_ratio,
                val_ratio=args.val_ratio,
                purge_gap_bars=args.purge_gap_bars,
            ),
        )
        return

    if args.command == "train-baselines":
        train_df = pd.read_parquet(args.train_parquet)
        val_df = pd.read_parquet(args.val_parquet)
        train_baseline_pack(
            train_df,
            val_df,
            config=BaselineTrainingConfig(output_root=Path(args.output_root)),
        )
        return

    if args.command == "run-baseline-research":
        summary = run_baseline_research(
            Path(args.dataset_root),
            config=BaselineResearchConfig(
                output_root=Path(args.output_root),
                decision_threshold=args.decision_threshold,
            ),
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "run-walk-forward-research":
        summary = run_walk_forward_research(
            Path(args.dataset_root),
            config=WalkForwardConfig(
                output_root=Path(args.output_root),
                decision_threshold=args.decision_threshold,
                initial_train_ratio=args.initial_train_ratio,
                validation_ratio=args.validation_ratio,
                step_ratio=args.step_ratio,
                purge_gap_bars=args.purge_gap_bars,
                min_train_timestamps=args.min_train_timestamps,
                min_validation_timestamps=args.min_validation_timestamps,
                max_folds=args.max_folds,
            ),
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "run-ablation-research":
        threshold_grid = tuple(args.threshold) if args.threshold else DEFAULT_THRESHOLD_GRID
        summary = run_ablation_research(
            Path(args.dataset_root),
            config=AblationResearchConfig(
                output_root=Path(args.output_root),
                decision_threshold=args.decision_threshold,
                threshold_grid=threshold_grid,
            ),
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "run-calibration-audit":
        threshold_grid = tuple(args.threshold) if args.threshold else DEFAULT_CALIBRATION_THRESHOLD_GRID
        methods = tuple(args.method) if args.method else DEFAULT_CALIBRATION_METHODS
        summary = run_saved_model_calibration_audit(
            Path(args.model_dir),
            config=CalibrationAuditConfig(
                output_root=Path(args.output_root),
                decision_threshold=args.decision_threshold,
                threshold_grid=threshold_grid,
                calibration_bins=args.calibration_bins,
                methods=methods,
                min_fit_rows=args.min_fit_rows,
            ),
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "run-research-pipeline":
        provider = LocalParquetProvider(root=Path(args.data_root))
        ablation_thresholds = tuple(args.ablation_threshold) if args.ablation_threshold else DEFAULT_THRESHOLD_GRID
        summary = run_research_pipeline(
            provider,
            config=ResearchPipelineConfig(
                data_root=Path(args.data_root),
                dataset_output_root=Path(args.dataset_output_root),
                research_output_root=Path(args.research_output_root),
                dataset_version=args.dataset_version,
                tickers=args.ticker,
                timeframe=args.timeframe,
                factor_aliases=args.factor,
                start=pd.Timestamp(args.start) if args.start else None,
                end=pd.Timestamp(args.end) if args.end else None,
                label_config=TripleBarrierConfig(
                    horizon_bars=args.horizon_bars,
                    min_move_pct=args.min_move_pct,
                    atr_mult=args.atr_mult,
                ),
                split_config=SplitConfig(
                    dataset_version=args.dataset_version,
                    train_ratio=args.train_ratio,
                    val_ratio=args.val_ratio,
                    purge_gap_bars=args.purge_gap_bars,
                    timeframe=args.timeframe,
                    horizon_bars=args.horizon_bars,
                ),
                research_config=BaselineResearchConfig(
                    output_root=Path(args.research_output_root),
                    decision_threshold=args.decision_threshold,
                ),
                run_walk_forward=not args.no_walk_forward,
                walk_forward_config=WalkForwardConfig(
                    output_root=Path(args.research_output_root) / "walk_forward",
                    decision_threshold=args.decision_threshold,
                    initial_train_ratio=args.wf_initial_train_ratio,
                    validation_ratio=args.wf_validation_ratio,
                    step_ratio=args.wf_step_ratio,
                    purge_gap_bars=args.purge_gap_bars,
                    max_folds=args.wf_max_folds,
                ),
                run_ablation=args.run_ablation,
                ablation_config=AblationResearchConfig(
                    output_root=Path(args.research_output_root) / "ablation",
                    decision_threshold=args.decision_threshold,
                    threshold_grid=ablation_thresholds,
                ),
            ),
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "run-research-grid":
        provider = LocalParquetProvider(root=Path(args.data_root))
        timeframes = list(dict.fromkeys(args.timeframe or ["5m", "15m", "1h"]))
        horizons = list(dict.fromkeys(args.horizon or [6, 12, 24]))

        summary = run_research_grid(
            provider,
            config=ResearchGridConfig(
                data_root=Path(args.data_root),
                dataset_output_root=Path(args.dataset_output_root),
                research_output_root=Path(args.research_output_root),
                grid_name=args.grid_name,
                tickers=args.ticker,
                factor_aliases=args.factor,
                timeframes=timeframes,
                horizons=horizons,
                start=pd.Timestamp(args.start) if args.start else None,
                end=pd.Timestamp(args.end) if args.end else None,
                decision_threshold=args.decision_threshold,
            )
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "ingest-asset-parquet":
        df = pd.read_parquet(args.input_parquet)
        ingest_asset_frame(
            df,
            data_root=Path(args.data_root),
            ticker=args.ticker,
            timeframe=args.timeframe,
            source=args.source,
        )
        return

    if args.command == "ingest-factor-parquet":
        df = pd.read_parquet(args.input_parquet)
        ingest_factor_frame(
            df,
            data_root=Path(args.data_root),
            alias=args.alias,
            timeframe=args.timeframe,
            source=args.source,
        )
        return

    if args.command == "materialize-feature-store":
        provider = LocalParquetProvider(root=Path(args.data_root))
        materialize_feature_store(
            provider,
            config=FeatureStoreMaterializationConfig(
                data_root=Path(args.data_root),
                ticker=args.ticker,
                timeframe=args.timeframe,
                factor_aliases=args.factor,
                start=_parse_optional_timestamp(args.start),
                end=_parse_optional_timestamp(args.end),
                feature_config=FeatureBuildConfig(timeframe=args.timeframe),
                label_config=TripleBarrierConfig(
                    horizon_bars=args.horizon_bars,
                    min_move_pct=args.min_move_pct,
                    atr_mult=args.atr_mult,
                ),
            ),
        )
        return

    if args.command == "materialize-dataset":
        materialize_dataset(
            data_root=Path(args.data_root),
            output_root=Path(args.output_root),
            dataset_version=args.dataset_version,
            tickers=args.ticker,
            timeframe=args.timeframe,
            horizon_bars=args.horizon_bars,
            split_config=SplitConfig(
                dataset_version=args.dataset_version,
                train_ratio=args.train_ratio,
                val_ratio=args.val_ratio,
                purge_gap_bars=args.purge_gap_bars,
                timeframe=args.timeframe,
                horizon_bars=args.horizon_bars,
            ),
        )
        return

    if args.command == "build-sequence-dataset":
        config = SequenceDatasetConfig(
            data_root=Path(args.data_root),
            output_root=Path(args.output_root),
            source_dataset_version=args.source_dataset_version,
            sequence_dataset_version=args.sequence_dataset_version,
            timeframe=args.timeframe,
            horizon_bars=args.horizon_bars,
            window_bars=args.window_bars,
            stride=args.stride,
            min_coverage_ratio=args.min_coverage_ratio,
            tickers=args.ticker if args.ticker else None,
            feature_cols=args.feature_col if args.feature_col else None,
            exclude_cols=args.exclude_feature if args.exclude_feature else None,
        )
        manifest = build_sequence_dataset(config)
        import dataclasses
        print(json.dumps(dataclasses.asdict(manifest), indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "train-neural-sequence":
        config = NeuralTrainingConfig(
            sequence_root=Path(args.sequence_root),
            dataset_version=args.dataset_version,
            window_bars=args.window,
            output_root=Path(args.output_root),
            model_family=args.model_family,
            epochs=args.epochs,
            batch_size=args.batch_size,
            learning_rate=args.lr,
            hidden_size=args.hidden_size,
            num_layers=args.num_layers,
            dropout=args.dropout,
            early_stopping_patience=args.patience,
            device=args.device,
        )
        manifest = train_neural_sequence(config)
        print(manifest.to_json())
        return

    if args.command == "tinkoff-sync-asset":
        summary = sync_tinkoff_asset_history(
            data_root=Path(args.data_root),
            ticker=args.ticker,
            timeframe=args.timeframe,
            start=_parse_required_timestamp(args.from_time),
            end=_parse_required_timestamp(args.to_time),
            instrument_kind=args.instrument_kind,
            class_code=args.class_code,
            incremental=not args.no_incremental,
            overlap_bars=args.overlap_bars,
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "tinkoff-sync-factor":
        summary = sync_tinkoff_factor_history(
            data_root=Path(args.data_root),
            alias=args.alias,
            ticker=args.ticker,
            timeframe=args.timeframe,
            start=_parse_required_timestamp(args.from_time),
            end=_parse_required_timestamp(args.to_time),
            instrument_kind=args.instrument_kind,
            class_code=args.class_code,
            incremental=not args.no_incremental,
            overlap_bars=args.overlap_bars,
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "tinkoff-sync-universe":
        summary = sync_tinkoff_universe(
            config_path=Path(args.config),
            data_root=Path(args.data_root),
            start=_parse_required_timestamp(args.from_time),
            end=_parse_required_timestamp(args.to_time),
            incremental=not args.no_incremental,
            overlap_bars=args.overlap_bars,
            timeframe_override=args.timeframe,
        )
        print(json.dumps(summary, indent=2, ensure_ascii=False, default=str))
        return

    if args.command == "raw-coverage-report":
        report = generate_coverage_report(Path(args.data_root))
        write_coverage_report(report, Path(args.output))
        print(json.dumps(report, indent=2, ensure_ascii=False, default=str))
        return

def _parse_optional_timestamp(value: str | None) -> pd.Timestamp | None:
    if not value:
        return None
    ts = pd.Timestamp(value)
    if ts.tzinfo is None:
        return ts.tz_localize("UTC")
    return ts.tz_convert("UTC")


def _parse_required_timestamp(value: str) -> pd.Timestamp:
    ts = _parse_optional_timestamp(value)
    if ts is None:
        raise ValueError("timestamp value is required")
    return ts


if __name__ == "__main__":
    main()
