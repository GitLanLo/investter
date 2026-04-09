from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import json
import pickle

import numpy as np
import pandas as pd
from pandas.api.types import is_bool_dtype, is_numeric_dtype
from sklearn.compose import ColumnTransformer
from sklearn.ensemble import RandomForestClassifier
from sklearn.impute import SimpleImputer
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import balanced_accuracy_score, f1_score, precision_score, recall_score
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler


TARGET_CLASS_ORDER = ["down_signal", "no_trade", "up_signal"]


@dataclass(slots=True)
class BaselineTrainingConfig:
    output_root: Path
    model_group: str = "baseline_pack_v1"
    random_state: int = 42


def train_baseline_pack(
    train_df: pd.DataFrame,
    val_df: pd.DataFrame,
    *,
    config: BaselineTrainingConfig,
) -> dict:
    feature_columns = _feature_columns(train_df)
    if not feature_columns:
        raise ValueError("no feature columns found for baseline training")

    x_train = train_df[feature_columns]
    y_train = train_df["label_class"].astype(str)
    x_val = val_df[feature_columns]
    y_val = val_df["label_class"].astype(str)

    config.output_root.mkdir(parents=True, exist_ok=True)

    results: dict[str, dict] = {}
    for model_name, pipeline in _model_pack(config.random_state).items():
        pipeline.fit(x_train, y_train)
        pred = pipeline.predict(x_val)
        prob = pipeline.predict_proba(x_val)
        metrics = _classification_metrics(y_val, pred)
        classes = list(pipeline.named_steps["model"].classes_)

        model_dir = config.output_root / model_name
        model_dir.mkdir(parents=True, exist_ok=True)
        with (model_dir / "model.pkl").open("wb") as fh:
            pickle.dump(pipeline, fh)

        predictions = pd.DataFrame({"label_class": y_val.reset_index(drop=True), "pred_class": pred})
        for idx, cls in enumerate(classes):
            predictions[f"prob_{cls}"] = prob[:, idx]
        predictions.to_parquet(model_dir / "predictions.parquet", index=False)

        report = {
            "model_name": model_name,
            "feature_columns": feature_columns,
            "classes": classes,
            "metrics": metrics,
        }
        (model_dir / "metrics.json").write_text(
            json.dumps(report, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        results[model_name] = report

    summary = {
        "model_group": config.model_group,
        "feature_columns": feature_columns,
        "models": results,
    }
    (config.output_root / "summary.json").write_text(
        json.dumps(summary, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    return summary


def _feature_columns(df: pd.DataFrame) -> list[str]:
    reserved = {
        "timestamp",
        "asof_time",
        "ticker",
        "timeframe",
        "source",
        "ingested_at",
        "factor_alias",
        "feature_schema_version",
        "target_schema_version",
        "label_class",
        "label_up",
        "label_down",
        "label_no_trade",
        "label_asof_time",
        "label_horizon_end_time",
        "label_is_complete",
        "horizon_bars",
        "move_pct",
        "barrier_up_price",
        "barrier_down_price",
    }
    return [
        col
        for col in df.columns
        if col not in reserved
        and (is_numeric_dtype(df[col]) or is_bool_dtype(df[col]))
        and df[col].notna().any()
    ]


def _model_pack(random_state: int) -> dict[str, Pipeline]:
    numeric_preprocessor = Pipeline(
        steps=[
            ("imputer", SimpleImputer(strategy="median")),
            ("scaler", StandardScaler()),
        ]
    )
    passthrough_preprocessor = Pipeline(
        steps=[
            ("imputer", SimpleImputer(strategy="median")),
        ]
    )

    return {
        "logreg_multiclass": Pipeline(
            steps=[
                ("preprocessor", ColumnTransformer([("num", numeric_preprocessor, slice(0, None))])),
                (
                    "model",
                    LogisticRegression(
                        max_iter=500,
                        class_weight="balanced",
                        random_state=random_state,
                    ),
                ),
            ]
        ),
        "rf_multiclass": Pipeline(
            steps=[
                ("preprocessor", ColumnTransformer([("num", passthrough_preprocessor, slice(0, None))])),
                (
                    "model",
                    RandomForestClassifier(
                        n_estimators=200,
                        max_depth=8,
                        min_samples_leaf=4,
                        n_jobs=1,
                        class_weight="balanced_subsample",
                        random_state=random_state,
                    ),
                ),
            ]
        ),
    }


def _classification_metrics(y_true: pd.Series, y_pred: np.ndarray) -> dict[str, float]:
    return {
        "macro_f1": float(f1_score(y_true, y_pred, average="macro", zero_division=0)),
        "precision_macro": float(precision_score(y_true, y_pred, average="macro", zero_division=0)),
        "recall_macro": float(recall_score(y_true, y_pred, average="macro", zero_division=0)),
        "balanced_accuracy": float(balanced_accuracy_score(y_true, y_pred)),
    }
