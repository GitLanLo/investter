from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import json
import pickle

import numpy as np
import pandas as pd

from ml_core.training.baselines import _classification_metrics, _feature_columns, _model_pack


@dataclass(slots=True)
class BaselineResearchConfig:
    output_root: Path
    model_group: str = "baseline_research_v1"
    random_state: int = 42
    decision_threshold: float = 0.65


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
    results: dict[str, dict] = {}

    x_train = train_df[feature_columns]
    y_train = train_df["label_class"].astype(str)

    for model_name, pipeline in _model_pack(config.random_state).items():
        pipeline.fit(x_train, y_train)
        model_dir = config.output_root / model_name
        model_dir.mkdir(parents=True, exist_ok=True)

        with (model_dir / "model.pkl").open("wb") as fh:
            pickle.dump(pipeline, fh)

        val_result = evaluate_split(
            pipeline,
            split_df=val_df,
            feature_columns=feature_columns,
            split_name="val",
            decision_threshold=config.decision_threshold,
        )
        test_result = evaluate_split(
            pipeline,
            split_df=test_df,
            feature_columns=feature_columns,
            split_name="test",
            decision_threshold=config.decision_threshold,
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
            "decision_threshold": config.decision_threshold,
            "validation": val_result["summary"],
            "test": test_result["summary"],
            "top_feature_importance": importance.head(20).to_dict(orient="records"),
        }
        (model_dir / "metrics.json").write_text(
            json.dumps(report, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        results[model_name] = report

    best_model_name = max(
        results,
        key=lambda name: (
            results[name]["validation"]["metrics"]["macro_f1"],
            results[name]["validation"]["metrics"]["balanced_accuracy"],
        ),
    )
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
    if split_df.empty:
        return {
            "summary": {
                "split_name": split_name,
                "rows": 0,
                "metrics": {},
                "signal_metrics": {},
                "per_ticker": {},
            },
            "predictions": pd.DataFrame(),
        }

    x = split_df[feature_columns]
    y_true = split_df["label_class"].astype(str)
    pred = pipeline.predict(x)
    prob = pipeline.predict_proba(x)
    classes = list(pipeline.named_steps["model"].classes_)

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
            "rows": int(len(split_df)),
            "metrics": _classification_metrics(y_true, pred),
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

    return {
        "signal_coverage": float(coverage),
        "precision_actionable_signal": float(precision),
        "recall_actionable_signal": float(recall),
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
                f"- val macro_f1: `{val['metrics'].get('macro_f1', 0):.4f}`",
                f"- val balanced_accuracy: `{val['metrics'].get('balanced_accuracy', 0):.4f}`",
                f"- val signal_coverage: `{val['signal_metrics'].get('signal_coverage', 0):.4f}`",
                f"- test macro_f1: `{test['metrics'].get('macro_f1', 0):.4f}`",
                f"- test balanced_accuracy: `{test['metrics'].get('balanced_accuracy', 0):.4f}`",
                f"- test signal_coverage: `{test['signal_metrics'].get('signal_coverage', 0):.4f}`",
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
