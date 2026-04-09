from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import json
import pickle

import numpy as np
import pandas as pd

from ml_core.training.baselines import _classification_metrics, _feature_columns, _model_pack

DEFAULT_THRESHOLD_GRID = (0.55, 0.6, 0.65, 0.7, 0.75, 0.8)


@dataclass(slots=True)
class BaselineResearchConfig:
    output_root: Path
    model_group: str = "baseline_research_v1"
    random_state: int = 42
    decision_threshold: float = 0.65
    tune_decision_threshold: bool = True
    threshold_grid: tuple[float, ...] = DEFAULT_THRESHOLD_GRID


@dataclass(slots=True)
class AblationResearchConfig:
    output_root: Path
    model_group: str = "ablation_research_v1"
    random_state: int = 42
    decision_threshold: float = 0.65
    threshold_grid: tuple[float, ...] = DEFAULT_THRESHOLD_GRID


@dataclass(slots=True)
class WalkForwardConfig:
    output_root: Path
    model_group: str = "walk_forward_research_v1"
    random_state: int = 42
    decision_threshold: float = 0.65
    initial_train_ratio: float = 0.6
    validation_ratio: float = 0.1
    step_ratio: float = 0.05
    purge_gap_bars: int = 12
    min_train_timestamps: int = 60
    min_validation_timestamps: int = 24
    max_folds: int = 6


def run_baseline_research(
    dataset_root: Path,
    *,
    config: BaselineResearchConfig,
) -> dict:
    train_df = load_dataset_split(dataset_root, "train")
    val_df = load_dataset_split(dataset_root, "val")
    test_df = load_dataset_split(dataset_root, "test")

    if train_df.empty or val_df.empty:
        raise ValueError("train and val splits must be non-empty")

    feature_columns = _feature_columns(train_df)
    if not feature_columns:
        raise ValueError("no feature columns found for research run")

    config.output_root.mkdir(parents=True, exist_ok=True)
    results = _run_model_suite(
        train_df=train_df,
        val_df=val_df,
        test_df=test_df,
        feature_columns=feature_columns,
        output_root=config.output_root,
        random_state=config.random_state,
        decision_threshold=config.decision_threshold,
        tune_decision_threshold=config.tune_decision_threshold,
        threshold_grid=config.threshold_grid,
    )
    best_model_name = _best_model_name(results)
    summary = {
        "model_group": config.model_group,
        "feature_columns": feature_columns,
        "decision_threshold": config.decision_threshold,
        "dataset_root": str(dataset_root),
        "best_model": best_model_name,
        "models": results,
    }
    (config.output_root / "summary.json").write_text(
        json.dumps(summary, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    (config.output_root / "report.md").write_text(
        render_markdown_report(summary),
        encoding="utf-8",
    )
    return summary


def run_ablation_research(
    dataset_root: Path,
    *,
    config: AblationResearchConfig,
) -> dict:
    train_df = load_dataset_split(dataset_root, "train")
    val_df = load_dataset_split(dataset_root, "val")
    test_df = load_dataset_split(dataset_root, "test")

    if train_df.empty or val_df.empty:
        raise ValueError("train and val splits must be non-empty")

    full_feature_columns = _feature_columns(train_df)
    if not full_feature_columns:
        raise ValueError("no feature columns found for ablation run")

    scenarios = build_feature_ablation_scenarios(full_feature_columns)
    config.output_root.mkdir(parents=True, exist_ok=True)

    scenario_summaries: dict[str, dict] = {}
    for scenario_name, scenario_feature_columns in scenarios.items():
        scenario_output_root = config.output_root / scenario_name
        scenario_output_root.mkdir(parents=True, exist_ok=True)

        results = _run_model_suite(
            train_df=train_df,
            val_df=val_df,
            test_df=test_df,
            feature_columns=scenario_feature_columns,
            output_root=scenario_output_root,
            random_state=config.random_state,
            decision_threshold=config.decision_threshold,
            tune_decision_threshold=True,
            threshold_grid=config.threshold_grid,
        )
        best_model_name = _best_model_name(results)
        scenario_summary = {
            "scenario_name": scenario_name,
            "feature_columns": scenario_feature_columns,
            "feature_count": len(scenario_feature_columns),
            "best_model": best_model_name,
            "models": results,
        }
        (scenario_output_root / "summary.json").write_text(
            json.dumps(scenario_summary, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        scenario_summaries[scenario_name] = scenario_summary

    best_scenario_name = _best_scenario_name(scenario_summaries)
    summary = {
        "model_group": config.model_group,
        "dataset_root": str(dataset_root),
        "decision_threshold": config.decision_threshold,
        "scenario_count": len(scenario_summaries),
        "best_scenario": best_scenario_name,
        "scenarios": scenario_summaries,
    }
    (config.output_root / "summary.json").write_text(
        json.dumps(summary, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    (config.output_root / "report.md").write_text(
        render_ablation_markdown_report(summary),
        encoding="utf-8",
    )
    return summary


def _run_model_suite(
    *,
    train_df: pd.DataFrame,
    val_df: pd.DataFrame,
    test_df: pd.DataFrame,
    feature_columns: list[str],
    output_root: Path,
    random_state: int,
    decision_threshold: float,
    tune_decision_threshold: bool,
    threshold_grid: tuple[float, ...],
) -> dict[str, dict]:
    results: dict[str, dict] = {}
    x_train = train_df[feature_columns]
    y_train = train_df["label_class"].astype(str)

    for model_name, pipeline in _model_pack(random_state).items():
        pipeline.fit(x_train, y_train)
        model_dir = output_root / model_name
        model_dir.mkdir(parents=True, exist_ok=True)

        with (model_dir / "model.pkl").open("wb") as fh:
            pickle.dump(pipeline, fh)

        val_predictions, classes = build_prediction_frame(
            pipeline,
            split_df=val_df,
            feature_columns=feature_columns,
        )
        threshold_tuning = tune_threshold_from_validation(
            val_predictions,
            classes=classes,
            default_threshold=decision_threshold,
            threshold_grid=threshold_grid,
            enabled=tune_decision_threshold,
        )
        selected_threshold = float(threshold_tuning["selected_threshold"])

        val_result = summarize_prediction_frame(
            val_predictions,
            split_name="val",
            classes=classes,
            decision_threshold=selected_threshold,
        )
        test_predictions, _ = build_prediction_frame(
            pipeline,
            split_df=test_df,
            feature_columns=feature_columns,
        )
        test_result = summarize_prediction_frame(
            test_predictions,
            split_name="test",
            classes=classes,
            decision_threshold=selected_threshold,
        )

        val_result["predictions"].to_parquet(model_dir / "val_predictions.parquet", index=False)
        test_result["predictions"].to_parquet(model_dir / "test_predictions.parquet", index=False)

        importance = compute_feature_importance(
            pipeline.named_steps["model"],
            feature_columns=feature_columns,
        )
        importance.to_csv(model_dir / "feature_importance.csv", index=False)

        report = {
            "model_name": model_name,
            "feature_columns": feature_columns,
            "decision_threshold": decision_threshold,
            "selected_threshold": selected_threshold,
            "threshold_tuning": threshold_tuning,
            "validation": val_result["summary"],
            "test": test_result["summary"],
            "top_feature_importance": importance.head(20).to_dict(orient="records"),
        }
        (model_dir / "metrics.json").write_text(
            json.dumps(report, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        results[model_name] = report

    return results


def run_walk_forward_research(
    dataset_root: Path,
    *,
    config: WalkForwardConfig,
) -> dict:
    train_df = load_dataset_split(dataset_root, "train")
    val_df = load_dataset_split(dataset_root, "val")
    walk_forward_df = pd.concat([train_df, val_df], ignore_index=True)

    if walk_forward_df.empty:
        raise ValueError("train+val history must be non-empty for walk-forward research")

    feature_columns = _feature_columns(walk_forward_df)
    if not feature_columns:
        raise ValueError("no feature columns found for walk-forward run")

    folds = build_walk_forward_folds(walk_forward_df, config=config)
    config.output_root.mkdir(parents=True, exist_ok=True)

    results: dict[str, dict] = {}
    for model_name in _model_pack(config.random_state):
        fold_summaries: list[dict] = []
        prediction_frames: list[pd.DataFrame] = []
        importance_frames: list[pd.Series] = []

        for fold in folds:
            pipeline = _model_pack(config.random_state)[model_name]
            train_frame = fold["train"]
            validation_frame = fold["validation"]

            pipeline.fit(
                train_frame[feature_columns],
                train_frame["label_class"].astype(str),
            )
            validation_result = evaluate_split(
                pipeline,
                split_df=validation_frame,
                feature_columns=feature_columns,
                split_name=f"fold_{fold['fold_id']}",
                decision_threshold=config.decision_threshold,
            )

            fold_summary = {
                "fold_id": fold["fold_id"],
                "train_range": fold["train_range"],
                "validation_range": fold["validation_range"],
                "train_rows": fold["train_rows"],
                "validation_rows": fold["validation_rows"],
                **validation_result["summary"],
            }
            fold_summaries.append(fold_summary)

            predictions = validation_result["predictions"].copy()
            predictions["fold_id"] = fold["fold_id"]
            prediction_frames.append(predictions)

            importance = compute_feature_importance(
                pipeline.named_steps["model"],
                feature_columns=feature_columns,
            ).set_index("feature")["importance"]
            importance_frames.append(importance.rename(f"fold_{fold['fold_id']}"))

        model_dir = config.output_root / model_name
        model_dir.mkdir(parents=True, exist_ok=True)

        all_predictions = pd.concat(prediction_frames, ignore_index=True) if prediction_frames else pd.DataFrame()
        all_predictions.to_parquet(model_dir / "walk_forward_predictions.parquet", index=False)

        importance_frame = (
            pd.concat(importance_frames, axis=1).fillna(0.0).mean(axis=1).rename("importance").reset_index()
            if importance_frames
            else pd.DataFrame(columns=["feature", "importance"])
        )
        importance_frame.to_csv(model_dir / "feature_importance.csv", index=False)

        aggregate = aggregate_walk_forward_folds(fold_summaries)
        report = {
            "model_name": model_name,
            "feature_columns": feature_columns,
            "decision_threshold": config.decision_threshold,
            "walk_forward": aggregate,
            "folds": fold_summaries,
            "top_feature_importance": importance_frame.head(20).to_dict(orient="records"),
        }
        (model_dir / "walk_forward_metrics.json").write_text(
            json.dumps(report, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        results[model_name] = report

    best_model_name = max(
        results,
        key=lambda name: (
            results[name]["walk_forward"]["validation_metrics_mean"].get("macro_f1", 0.0),
            results[name]["walk_forward"]["validation_metrics_mean"].get("balanced_accuracy", 0.0),
        ),
    )
    summary = {
        "model_group": config.model_group,
        "dataset_root": str(dataset_root),
        "decision_threshold": config.decision_threshold,
        "best_model": best_model_name,
        "fold_count": len(folds),
        "fold_config": {
            "initial_train_ratio": config.initial_train_ratio,
            "validation_ratio": config.validation_ratio,
            "step_ratio": config.step_ratio,
            "purge_gap_bars": config.purge_gap_bars,
            "min_train_timestamps": config.min_train_timestamps,
            "min_validation_timestamps": config.min_validation_timestamps,
            "max_folds": config.max_folds,
        },
        "models": results,
    }
    (config.output_root / "summary.json").write_text(
        json.dumps(summary, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    (config.output_root / "report.md").write_text(
        render_walk_forward_markdown_report(summary),
        encoding="utf-8",
    )
    return summary


def load_dataset_split(dataset_root: Path, split_name: str) -> pd.DataFrame:
    split_root = dataset_root / f"split={split_name}"
    paths = sorted(split_root.rglob("*.parquet"))
    if not paths:
        return pd.DataFrame()
    frames = [pd.read_parquet(path) for path in paths]
    df = pd.concat(frames, ignore_index=True)
    if "timestamp" in df.columns:
        df["timestamp"] = pd.to_datetime(df["timestamp"], utc=True)
        df = df.sort_values("timestamp").reset_index(drop=True)
    return df


def build_walk_forward_folds(
    df: pd.DataFrame,
    *,
    config: WalkForwardConfig,
) -> list[dict]:
    data = df.copy()
    if "timestamp" not in data.columns:
        raise ValueError("df must contain timestamp column for walk-forward evaluation")

    data["timestamp"] = pd.to_datetime(data["timestamp"], utc=True)
    data = data.sort_values("timestamp").reset_index(drop=True)
    data = data[data["label_class"].notna()].reset_index(drop=True)

    unique_ts = pd.Index(sorted(data["timestamp"].unique()))
    if len(unique_ts) < 3:
        raise ValueError("not enough timestamps for walk-forward evaluation")

    initial_train_size = max(
        config.min_train_timestamps,
        int(len(unique_ts) * config.initial_train_ratio),
    )
    validation_size = max(
        config.min_validation_timestamps,
        int(len(unique_ts) * config.validation_ratio),
    )
    step_size = max(1, int(len(unique_ts) * config.step_ratio))

    max_validation_size = len(unique_ts) - config.purge_gap_bars - 2
    if max_validation_size < 1:
        raise ValueError("not enough timestamps to build walk-forward folds with current config")
    validation_size = min(validation_size, max_validation_size)

    max_train_end_idx = len(unique_ts) - config.purge_gap_bars - validation_size - 1
    initial_train_size = min(initial_train_size, max_train_end_idx + 1)
    if initial_train_size < 2 or initial_train_size - 1 > max_train_end_idx:
        raise ValueError("not enough timestamps to build walk-forward folds with current config")

    folds: list[dict] = []
    train_end_idx = initial_train_size - 1
    fold_id = 1
    while train_end_idx <= max_train_end_idx and len(folds) < config.max_folds:
        validation_start_idx = train_end_idx + config.purge_gap_bars + 1
        validation_end_idx = validation_start_idx + validation_size - 1
        if validation_end_idx >= len(unique_ts):
            break

        train_end_ts = unique_ts[train_end_idx]
        validation_start_ts = unique_ts[validation_start_idx]
        validation_end_ts = unique_ts[validation_end_idx]

        train_frame = data[data["timestamp"] <= train_end_ts].copy()
        validation_frame = data[
            (data["timestamp"] >= validation_start_ts) & (data["timestamp"] <= validation_end_ts)
        ].copy()
        if train_frame.empty or validation_frame.empty:
            train_end_idx += step_size
            continue

        folds.append(
            {
                "fold_id": fold_id,
                "train": train_frame,
                "validation": validation_frame,
                "train_rows": int(len(train_frame)),
                "validation_rows": int(len(validation_frame)),
                "train_range": _timestamp_range(train_frame),
                "validation_range": _timestamp_range(validation_frame),
            }
        )
        fold_id += 1
        train_end_idx += step_size

    if not folds:
        raise ValueError("walk-forward evaluation produced no folds")
    return folds


def evaluate_split(
    pipeline,
    *,
    split_df: pd.DataFrame,
    feature_columns: list[str],
    split_name: str,
    decision_threshold: float,
) -> dict:
    predictions, classes = build_prediction_frame(
        pipeline,
        split_df=split_df,
        feature_columns=feature_columns,
    )
    return summarize_prediction_frame(
        predictions,
        split_name=split_name,
        classes=classes,
        decision_threshold=decision_threshold,
    )


def build_prediction_frame(
    pipeline,
    *,
    split_df: pd.DataFrame,
    feature_columns: list[str],
) -> tuple[pd.DataFrame, list[str]]:
    classes = list(pipeline.named_steps["model"].classes_)
    if split_df.empty:
        columns = ["timestamp", "ticker", "label_class", "pred_class", *[f"prob_{cls}" for cls in classes]]
        return pd.DataFrame(columns=columns), classes

    x = split_df[feature_columns]
    y_true = split_df["label_class"].astype(str)
    pred = pipeline.predict(x)
    prob = pipeline.predict_proba(x)

    predictions = pd.DataFrame(
        {
            "timestamp": split_df["timestamp"].reset_index(drop=True) if "timestamp" in split_df.columns else None,
            "ticker": split_df["ticker"].reset_index(drop=True) if "ticker" in split_df.columns else None,
            "label_class": y_true.reset_index(drop=True),
            "pred_class": pred,
        }
    )
    for idx, cls in enumerate(classes):
        predictions[f"prob_{cls}"] = prob[:, idx]
    return predictions, classes


def summarize_prediction_frame(
    predictions: pd.DataFrame,
    *,
    split_name: str,
    classes: list[str],
    decision_threshold: float,
) -> dict:
    if predictions.empty:
        return {
            "summary": {
                "split_name": split_name,
                "rows": 0,
                "metrics": {},
                "signal_metrics": {},
                "per_ticker": {},
            },
            "predictions": predictions,
        }

    signal_metrics = compute_signal_metrics(
        predictions,
        classes=classes,
        decision_threshold=decision_threshold,
    )
    per_ticker = compute_per_ticker_metrics(
        predictions,
        classes=classes,
        decision_threshold=decision_threshold,
    )

    return {
        "summary": {
            "split_name": split_name,
            "rows": int(len(predictions)),
            "metrics": _classification_metrics(
                predictions["label_class"].astype(str),
                predictions["pred_class"].astype(str).to_numpy(),
            ),
            "signal_metrics": signal_metrics,
            "per_ticker": per_ticker,
        },
        "predictions": predictions,
    }


def compute_signal_metrics(
    predictions: pd.DataFrame,
    *,
    classes: list[str],
    decision_threshold: float,
) -> dict[str, float]:
    actionable_mask, predicted_direction = actionable_signal_mask(
        predictions,
        classes=classes,
        decision_threshold=decision_threshold,
    )
    true_directional = predictions["label_class"].isin(["up_signal", "down_signal"])
    actionable_correct = actionable_mask & (predicted_direction == predictions["label_class"])

    actionable_count = int(actionable_mask.sum())
    true_directional_count = int(true_directional.sum())
    actionable_correct_count = int(actionable_correct.sum())
    coverage = actionable_count / len(predictions) if len(predictions) else 0.0
    precision = actionable_correct_count / actionable_count if actionable_count else 0.0
    recall = actionable_correct_count / true_directional_count if true_directional_count else 0.0
    actionable_f1 = (2.0 * precision * recall / (precision + recall)) if (precision + recall) else 0.0

    return {
        "signal_coverage": float(coverage),
        "precision_actionable_signal": float(precision),
        "recall_actionable_signal": float(recall),
        "actionable_f1": float(actionable_f1),
        "directional_hit_rate": float(precision),
        "actionable_count": actionable_count,
    }


def compute_per_ticker_metrics(
    predictions: pd.DataFrame,
    *,
    classes: list[str],
    decision_threshold: float,
) -> dict[str, dict]:
    if "ticker" not in predictions.columns or predictions["ticker"].isna().all():
        return {}

    result: dict[str, dict] = {}
    for ticker, frame in predictions.groupby("ticker"):
        result[str(ticker)] = compute_signal_metrics(
            frame.reset_index(drop=True),
            classes=classes,
            decision_threshold=decision_threshold,
        )
    return result


def tune_threshold_from_validation(
    predictions: pd.DataFrame,
    *,
    classes: list[str],
    default_threshold: float,
    threshold_grid: tuple[float, ...],
    enabled: bool,
) -> dict:
    if predictions.empty or not enabled:
        return {
            "enabled": enabled,
            "default_threshold": float(default_threshold),
            "selected_threshold": float(default_threshold),
            "candidates": [],
        }

    candidates = sorted({float(value) for value in threshold_grid if 0.0 < float(value) <= 1.0})
    if not candidates:
        candidates = [float(default_threshold)]

    candidate_reports: list[dict] = []
    best_threshold = float(default_threshold)
    best_rank: tuple[float, float, float, float, float] | None = None
    for threshold in candidates:
        signal_metrics = compute_signal_metrics(
            predictions,
            classes=classes,
            decision_threshold=float(threshold),
        )
        candidate_report = {
            "threshold": float(threshold),
            **signal_metrics,
        }
        candidate_reports.append(candidate_report)

        rank = (
            signal_metrics.get("actionable_f1", 0.0),
            signal_metrics.get("precision_actionable_signal", 0.0),
            signal_metrics.get("recall_actionable_signal", 0.0),
            signal_metrics.get("signal_coverage", 0.0),
            -abs(float(threshold) - float(default_threshold)),
        )
        if best_rank is None or rank > best_rank:
            best_rank = rank
            best_threshold = float(threshold)

    selected_metrics = next(
        (item for item in candidate_reports if item["threshold"] == best_threshold),
        {},
    )
    return {
        "enabled": True,
        "default_threshold": float(default_threshold),
        "selected_threshold": best_threshold,
        "selected_metrics": selected_metrics,
        "candidates": candidate_reports,
    }


def build_feature_ablation_scenarios(feature_columns: list[str]) -> dict[str, list[str]]:
    scenarios = {
        "full": list(feature_columns),
        "no_cross_asset": [col for col in feature_columns if not _is_cross_asset_feature(col)],
        "no_regime": [col for col in feature_columns if not _is_regime_feature(col)],
        "core_price_volume_only": [col for col in feature_columns if _is_core_price_volume_feature(col)],
    }
    return {name: columns for name, columns in scenarios.items() if columns}


def actionable_signal_mask(
    predictions: pd.DataFrame,
    *,
    classes: list[str],
    decision_threshold: float,
) -> tuple[pd.Series, pd.Series]:
    prob_by_class = {cls: predictions.get(f"prob_{cls}", pd.Series(np.zeros(len(predictions)))) for cls in classes}
    up = prob_by_class.get("up_signal", pd.Series(np.zeros(len(predictions))))
    down = prob_by_class.get("down_signal", pd.Series(np.zeros(len(predictions))))
    no_trade = prob_by_class.get("no_trade", pd.Series(np.zeros(len(predictions))))

    predicted_direction = np.where(up >= down, "up_signal", "down_signal")
    directional_prob = np.where(up >= down, up, down)
    winning_class = np.where(
        (up >= down) & (up >= no_trade),
        "up_signal",
        np.where((down > up) & (down >= no_trade), "down_signal", "no_trade"),
    )
    actionable = (winning_class != "no_trade") & (directional_prob >= decision_threshold)
    return pd.Series(actionable), pd.Series(predicted_direction)


def _is_cross_asset_feature(feature_name: str) -> bool:
    prefixes = ("usdrub_", "brent_", "rtsi_")
    return (
        feature_name.startswith(prefixes)
        or feature_name.startswith("asset_vs_")
        or feature_name in {"market_stress_proxy", "cross_asset_dispersion_proxy"}
    )


def _is_regime_feature(feature_name: str) -> bool:
    return feature_name in {
        "vol_regime_flag",
        "trend_regime_flag",
        "market_stress_proxy",
        "cross_asset_dispersion_proxy",
    }


def _is_core_price_volume_feature(feature_name: str) -> bool:
    price_prefixes = ("ret_", "close_to_", "hl_range_", "oc_range_")
    volume_prefixes = ("volume_rel_", "volume_zscore_")
    return (
        feature_name.startswith(price_prefixes)
        or feature_name in {"log_ret_1", "turnover_proxy", "volume_price_trend_component"}
        or feature_name.startswith(volume_prefixes)
    )


def _best_model_name(results: dict[str, dict]) -> str:
    return max(
        results,
        key=lambda name: (
            results[name]["validation"]["metrics"].get("macro_f1", 0.0),
            results[name]["validation"]["signal_metrics"].get("actionable_f1", 0.0),
            results[name]["validation"]["signal_metrics"].get("precision_actionable_signal", 0.0),
            results[name]["validation"]["metrics"].get("balanced_accuracy", 0.0),
        ),
    )


def _best_scenario_name(results: dict[str, dict]) -> str:
    return max(
        results,
        key=lambda name: (
            results[name]["models"][results[name]["best_model"]]["validation"]["signal_metrics"].get("actionable_f1", 0.0),
            results[name]["models"][results[name]["best_model"]]["validation"]["metrics"].get("macro_f1", 0.0),
            results[name]["models"][results[name]["best_model"]]["validation"]["signal_metrics"].get("precision_actionable_signal", 0.0),
        ),
    )


def aggregate_walk_forward_folds(folds: list[dict]) -> dict:
    return {
        "fold_count": len(folds),
        "rows_total": int(sum(fold.get("rows", fold.get("validation_rows", 0)) for fold in folds)),
        "validation_metrics_mean": _aggregate_numeric_dicts([fold.get("metrics", {}) for fold in folds], reducer="mean"),
        "validation_metrics_std": _aggregate_numeric_dicts([fold.get("metrics", {}) for fold in folds], reducer="std"),
        "signal_metrics_mean": _aggregate_numeric_dicts([fold.get("signal_metrics", {}) for fold in folds], reducer="mean"),
        "signal_metrics_std": _aggregate_numeric_dicts([fold.get("signal_metrics", {}) for fold in folds], reducer="std"),
        "per_ticker_mean": _aggregate_per_ticker_metrics(folds, reducer="mean"),
        "per_ticker_std": _aggregate_per_ticker_metrics(folds, reducer="std"),
    }


def _aggregate_per_ticker_metrics(folds: list[dict], *, reducer: str) -> dict[str, dict[str, float]]:
    by_ticker: dict[str, list[dict[str, float]]] = {}
    for fold in folds:
        for ticker, metrics in fold.get("per_ticker", {}).items():
            by_ticker.setdefault(str(ticker), []).append(metrics)

    return {
        ticker: _aggregate_numeric_dicts(items, reducer=reducer)
        for ticker, items in sorted(by_ticker.items())
    }


def _aggregate_numeric_dicts(items: list[dict], *, reducer: str) -> dict[str, float]:
    if not items:
        return {}

    keys = sorted({key for item in items for key, value in item.items() if isinstance(value, (int, float))})
    aggregated: dict[str, float] = {}
    for key in keys:
        values = np.asarray(
            [float(item[key]) for item in items if key in item and isinstance(item[key], (int, float))],
            dtype="float64",
        )
        if values.size == 0:
            continue
        if reducer == "std":
            aggregated[key] = float(np.nanstd(values))
        else:
            aggregated[key] = float(np.nanmean(values))
    return aggregated


def _timestamp_range(df: pd.DataFrame) -> dict[str, str | int | None]:
    if df.empty:
        return {"min": None, "max": None, "rows": 0}
    ts = pd.to_datetime(df["timestamp"], utc=True)
    return {
        "min": ts.min().isoformat(),
        "max": ts.max().isoformat(),
        "rows": int(len(df)),
    }


def compute_feature_importance(model, *, feature_columns: list[str]) -> pd.DataFrame:
    if hasattr(model, "feature_importances_"):
        importance = np.asarray(model.feature_importances_, dtype="float64")
    elif hasattr(model, "coef_"):
        coef = np.asarray(model.coef_, dtype="float64")
        importance = np.mean(np.abs(coef), axis=0)
    else:
        importance = np.zeros(len(feature_columns), dtype="float64")

    if importance.ndim > 1:
        importance = np.asarray(importance).reshape(-1)
    if len(importance) != len(feature_columns):
        aligned = np.zeros(len(feature_columns), dtype="float64")
        aligned[: min(len(aligned), len(importance))] = importance[: min(len(aligned), len(importance))]
        importance = aligned

    frame = pd.DataFrame({"feature": feature_columns, "importance": importance})
    return frame.sort_values("importance", ascending=False).reset_index(drop=True)


def render_markdown_report(summary: dict) -> str:
    lines = [
        "# Baseline Research Report",
        "",
        f"- model_group: `{summary['model_group']}`",
        f"- dataset_root: `{summary['dataset_root']}`",
        f"- decision_threshold: `{summary['decision_threshold']}`",
        f"- best_model: `{summary['best_model']}`",
        "",
        "## Models",
        "",
    ]

    for model_name, report in summary["models"].items():
        val = report["validation"]
        test = report["test"]
        lines.extend(
            [
                f"### {model_name}",
                "",
                f"- selected_threshold: `{report.get('selected_threshold', report['decision_threshold']):.2f}`",
                f"- val macro_f1: `{val['metrics'].get('macro_f1', 0):.4f}`",
                f"- val balanced_accuracy: `{val['metrics'].get('balanced_accuracy', 0):.4f}`",
                f"- val actionable_f1: `{val['signal_metrics'].get('actionable_f1', 0):.4f}`",
                f"- val signal_coverage: `{val['signal_metrics'].get('signal_coverage', 0):.4f}`",
                f"- test macro_f1: `{test['metrics'].get('macro_f1', 0):.4f}`",
                f"- test balanced_accuracy: `{test['metrics'].get('balanced_accuracy', 0):.4f}`",
                f"- test actionable_f1: `{test['signal_metrics'].get('actionable_f1', 0):.4f}`",
                f"- test signal_coverage: `{test['signal_metrics'].get('signal_coverage', 0):.4f}`",
                "",
            ]
        )

    return "\n".join(lines) + "\n"


def render_ablation_markdown_report(summary: dict) -> str:
    lines = [
        "# Ablation Research Report",
        "",
        f"- model_group: `{summary['model_group']}`",
        f"- dataset_root: `{summary['dataset_root']}`",
        f"- decision_threshold: `{summary['decision_threshold']}`",
        f"- best_scenario: `{summary['best_scenario']}`",
        "",
        "## Scenarios",
        "",
    ]

    for scenario_name, scenario in summary["scenarios"].items():
        best_model = scenario["best_model"]
        report = scenario["models"][best_model]
        val = report["validation"]
        lines.extend(
            [
                f"### {scenario_name}",
                "",
                f"- feature_count: `{scenario['feature_count']}`",
                f"- best_model: `{best_model}`",
                f"- selected_threshold: `{report.get('selected_threshold', report['decision_threshold']):.2f}`",
                f"- val macro_f1: `{val['metrics'].get('macro_f1', 0):.4f}`",
                f"- val actionable_f1: `{val['signal_metrics'].get('actionable_f1', 0):.4f}`",
                f"- val signal_coverage: `{val['signal_metrics'].get('signal_coverage', 0):.4f}`",
                "",
            ]
        )

    return "\n".join(lines) + "\n"


def render_walk_forward_markdown_report(summary: dict) -> str:
    lines = [
        "# Walk-Forward Research Report",
        "",
        f"- model_group: `{summary['model_group']}`",
        f"- dataset_root: `{summary['dataset_root']}`",
        f"- decision_threshold: `{summary['decision_threshold']}`",
        f"- best_model: `{summary['best_model']}`",
        f"- fold_count: `{summary['fold_count']}`",
        "",
        "## Models",
        "",
    ]

    for model_name, report in summary["models"].items():
        aggregate = report["walk_forward"]
        lines.extend(
            [
                f"### {model_name}",
                "",
                f"- mean macro_f1: `{aggregate['validation_metrics_mean'].get('macro_f1', 0):.4f}`",
                f"- mean balanced_accuracy: `{aggregate['validation_metrics_mean'].get('balanced_accuracy', 0):.4f}`",
                f"- mean signal_coverage: `{aggregate['signal_metrics_mean'].get('signal_coverage', 0):.4f}`",
                f"- std macro_f1: `{aggregate['validation_metrics_std'].get('macro_f1', 0):.4f}`",
                f"- std balanced_accuracy: `{aggregate['validation_metrics_std'].get('balanced_accuracy', 0):.4f}`",
                "",
            ]
        )

    return "\n".join(lines) + "\n"
