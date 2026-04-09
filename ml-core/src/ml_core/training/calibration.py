from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import json
import pickle

import numpy as np
import pandas as pd
from sklearn.isotonic import IsotonicRegression
from sklearn.linear_model import LogisticRegression


DEFAULT_CALIBRATION_METHODS = ("identity", "platt", "isotonic")
DEFAULT_CALIBRATION_THRESHOLD_GRID = (0.3, 0.35, 0.4, 0.45, 0.5, 0.55, 0.6)
CALIBRATION_RESEARCH_POLICY = {
    "selection_basis": "validation_fit_only",
    "ranking": [
        "validation.signal_metrics.actionable_f1",
        "validation.probability_metrics.actionable_expected_calibration_error_lower_is_better",
        "validation.signal_metrics.precision_actionable_signal",
        "validation.signal_metrics.signal_coverage",
    ],
    "fit_scope": "predicted directional rows on validation split",
    "test_usage": "audit_only",
}
CALIBRATION_PRODUCTION_POLICY = {
    "selection_basis": "validation_fit_only_gated",
    "hard_gates": {
        "validation.signal_metrics.precision_actionable_signal_min": 0.30,
        "validation.signal_metrics.signal_coverage_min": 0.05,
        "validation.probability_metrics.actionable_expected_calibration_error_max": 0.30,
    },
    "ranking": [
        "validation.signal_metrics.actionable_f1",
        "validation.signal_metrics.precision_actionable_signal",
        "validation.signal_metrics.signal_coverage",
        "validation.probability_metrics.actionable_expected_calibration_error_lower_is_better",
    ],
    "fallback": "research_candidate_if_none_pass",
    "test_usage": "audit_only",
}


@dataclass(slots=True)
class CalibrationAuditConfig:
    output_root: Path
    decision_threshold: float | None = None
    threshold_grid: tuple[float, ...] = DEFAULT_CALIBRATION_THRESHOLD_GRID
    calibration_bins: int = 8
    methods: tuple[str, ...] = DEFAULT_CALIBRATION_METHODS
    min_fit_rows: int = 20


def run_saved_model_calibration_audit(
    model_dir: Path,
    *,
    config: CalibrationAuditConfig,
) -> dict:
    config.output_root.mkdir(parents=True, exist_ok=True)

    model_report = json.loads((model_dir / "metrics.json").read_text(encoding="utf-8"))
    val_predictions = pd.read_parquet(model_dir / "val_predictions.parquet")
    test_predictions = pd.read_parquet(model_dir / "test_predictions.parquet")
    classes = infer_prediction_classes(val_predictions)
    base_threshold = float(
        config.decision_threshold
        if config.decision_threshold is not None
        else model_report.get("selected_threshold", model_report.get("decision_threshold", 0.55))
    )

    method_reports: dict[str, dict] = {}
    fit_frame = build_directional_calibration_frame(val_predictions, classes=classes)
    for method_name in _normalize_methods(config.methods):
        method_dir = config.output_root / method_name
        method_dir.mkdir(parents=True, exist_ok=True)

        fitted = fit_directional_calibrator(
            fit_frame,
            method=method_name,
            min_fit_rows=config.min_fit_rows,
        )
        if not fitted["available"]:
            report = {
                "method": method_name,
                "available": False,
                "reason": fitted["reason"],
                "fit_summary": fitted["fit_summary"],
            }
            (method_dir / "metrics.json").write_text(
                json.dumps(report, indent=2, ensure_ascii=False),
                encoding="utf-8",
            )
            method_reports[method_name] = report
            continue

        with (method_dir / "calibrator.pkl").open("wb") as fh:
            pickle.dump(fitted["calibrator"], fh)

        val_eval = evaluate_calibrated_predictions(
            val_predictions,
            classes=classes,
            fitted=fitted,
            split_name="val",
            default_threshold=base_threshold,
            threshold_grid=config.threshold_grid,
            calibration_bins=config.calibration_bins,
            tune_threshold=True,
        )
        test_eval = evaluate_calibrated_predictions(
            test_predictions,
            classes=classes,
            fitted=fitted,
            split_name="test",
            default_threshold=float(val_eval["selected_threshold"]),
            threshold_grid=(float(val_eval["selected_threshold"]),),
            calibration_bins=config.calibration_bins,
            tune_threshold=False,
        )

        (method_dir / "val_reliability.json").write_text(
            json.dumps(val_eval["reliability"], indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        (method_dir / "test_reliability.json").write_text(
            json.dumps(test_eval["reliability"], indent=2, ensure_ascii=False),
            encoding="utf-8",
        )

        report = {
            "method": method_name,
            "available": True,
            "calibrator_spec": fitted["spec"],
            "fit_summary": fitted["fit_summary"],
            "selected_threshold": float(val_eval["selected_threshold"]),
            "threshold_tuning": val_eval["threshold_tuning"],
            "validation": val_eval["summary"],
            "test": test_eval["summary"],
        }
        (method_dir / "metrics.json").write_text(
            json.dumps(report, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )
        method_reports[method_name] = report

    available_reports = {name: report for name, report in method_reports.items() if report.get("available")}
    if not available_reports:
        raise ValueError("no calibration methods available for audit")

    research_method_name = _best_calibration_method_name(available_reports)
    production_method_name, production_gate = _select_production_calibration_candidate(
        available_reports,
        fallback_name=research_method_name,
    )
    summary = {
        "model_dir": str(model_dir),
        "model_name": model_report.get("model_name"),
        "feature_columns": model_report.get("feature_columns", []),
        "base_selected_threshold": base_threshold,
        "research_candidate_policy": CALIBRATION_RESEARCH_POLICY,
        "research_candidate": _build_calibration_candidate_summary(
            research_method_name,
            available_reports[research_method_name],
            selection_mode="research_candidate",
        ),
        "production_candidate_policy": CALIBRATION_PRODUCTION_POLICY,
        "production_candidate": _build_calibration_candidate_summary(
            production_method_name,
            available_reports[production_method_name],
            selection_mode="production_candidate" if production_gate["passed"] else "fallback_research_candidate",
            validation_gate=production_gate,
        ),
        "methods": method_reports,
    }
    (config.output_root / "summary.json").write_text(
        json.dumps(summary, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    (config.output_root / "report.md").write_text(
        render_calibration_markdown_report(summary),
        encoding="utf-8",
    )
    return summary


def infer_prediction_classes(predictions: pd.DataFrame) -> list[str]:
    classes = sorted(
        column.removeprefix("prob_")
        for column in predictions.columns
        if column.startswith("prob_")
    )
    if not classes:
        raise ValueError("predictions must contain prob_* columns")
    return classes


def build_directional_calibration_frame(
    predictions: pd.DataFrame,
    *,
    classes: list[str],
) -> pd.DataFrame:
    up = predictions.get("prob_up_signal", pd.Series(np.zeros(len(predictions)), index=predictions.index)).astype("float64")
    down = predictions.get("prob_down_signal", pd.Series(np.zeros(len(predictions)), index=predictions.index)).astype("float64")
    no_trade = predictions.get("prob_no_trade", pd.Series(np.zeros(len(predictions)), index=predictions.index)).astype("float64")
    predicted_direction = np.where(up >= down, "up_signal", "down_signal")
    directional_prob = np.where(up >= down, up, down)
    winning_class = np.where(
        (up >= down) & (up >= no_trade),
        "up_signal",
        np.where((down > up) & (down >= no_trade), "down_signal", "no_trade"),
    )
    directional_prediction = winning_class != "no_trade"
    directional_success = (
        (pd.Series(predicted_direction, index=predictions.index).astype(str) == predictions["label_class"].astype(str))
        & predictions["label_class"].isin(["up_signal", "down_signal"])
    ).astype("int8")

    return pd.DataFrame(
        {
            "timestamp": predictions["timestamp"] if "timestamp" in predictions.columns else None,
            "ticker": predictions["ticker"] if "ticker" in predictions.columns else None,
            "label_class": predictions["label_class"].astype(str),
            "predicted_direction": predicted_direction,
            "winning_class": winning_class,
            "directional_prediction": directional_prediction.astype("bool"),
            "directional_prob": np.asarray(directional_prob, dtype="float64"),
            "directional_success": directional_success.astype("int8"),
            **{
                f"prob_{cls}": predictions[f"prob_{cls}"].astype("float64")
                for cls in classes
                if f"prob_{cls}" in predictions.columns
            },
        }
    )


def fit_directional_calibrator(
    fit_frame: pd.DataFrame,
    *,
    method: str,
    min_fit_rows: int,
) -> dict:
    directional_rows = fit_frame[fit_frame["directional_prediction"]].copy()
    fit_summary = {
        "fit_rows": int(len(directional_rows)),
        "positive_rate": float(directional_rows["directional_success"].mean()) if not directional_rows.empty else None,
    }
    if method == "identity":
        return {
            "available": True,
            "reason": None,
            "calibrator": {"method": "identity"},
            "spec": {"method": "identity"},
            "fit_summary": fit_summary,
        }

    if len(directional_rows) < min_fit_rows:
        return {
            "available": False,
            "reason": f"need at least {min_fit_rows} predicted directional rows",
            "calibrator": None,
            "spec": None,
            "fit_summary": fit_summary,
        }

    x = directional_rows["directional_prob"].to_numpy(dtype="float64")
    y = directional_rows["directional_success"].to_numpy(dtype="int8")
    if len(np.unique(y)) < 2:
        return {
            "available": False,
            "reason": "need both positive and negative directional outcomes",
            "calibrator": None,
            "spec": None,
            "fit_summary": fit_summary,
        }

    if method == "platt":
        model = LogisticRegression(max_iter=1000, random_state=42)
        model.fit(x.reshape(-1, 1), y)
        return {
            "available": True,
            "reason": None,
            "calibrator": {"method": "platt", "model": model},
            "spec": {
                "method": "platt",
                "coef": float(model.coef_[0][0]),
                "intercept": float(model.intercept_[0]),
            },
            "fit_summary": fit_summary,
        }

    if method == "isotonic":
        model = IsotonicRegression(out_of_bounds="clip", y_min=0.0, y_max=1.0)
        model.fit(x, y)
        return {
            "available": True,
            "reason": None,
            "calibrator": {"method": "isotonic", "model": model},
            "spec": {
                "method": "isotonic",
                "x_thresholds": [float(value) for value in model.X_thresholds_],
                "y_thresholds": [float(value) for value in model.y_thresholds_],
            },
            "fit_summary": fit_summary,
        }

    raise ValueError(f"unsupported calibration method: {method}")


def evaluate_calibrated_predictions(
    predictions: pd.DataFrame,
    *,
    classes: list[str],
    fitted: dict,
    split_name: str,
    default_threshold: float,
    threshold_grid: tuple[float, ...],
    calibration_bins: int,
    tune_threshold: bool,
) -> dict:
    frame = build_directional_calibration_frame(predictions, classes=classes)
    frame["calibrated_directional_prob"] = apply_directional_calibrator(
        frame["directional_prob"].to_numpy(dtype="float64"),
        fitted=fitted,
    )

    threshold_tuning = tune_threshold_from_calibration_frame(
        frame,
        default_threshold=default_threshold,
        threshold_grid=threshold_grid,
        enabled=tune_threshold,
    )
    selected_threshold = float(threshold_tuning["selected_threshold"])

    signal_metrics = compute_action_metrics_from_calibration_frame(
        frame,
        decision_threshold=selected_threshold,
    )
    probability_metrics, reliability = compute_calibrated_probability_metrics(
        frame,
        decision_threshold=selected_threshold,
        calibration_bins=calibration_bins,
    )

    return {
        "selected_threshold": selected_threshold,
        "threshold_tuning": threshold_tuning,
        "summary": {
            "split_name": split_name,
            "rows": int(len(frame)),
            "directional_rows": int(frame["directional_prediction"].sum()),
            "signal_metrics": signal_metrics,
            "probability_metrics": probability_metrics,
        },
        "reliability": reliability,
    }


def apply_directional_calibrator(probabilities: np.ndarray, *, fitted: dict) -> np.ndarray:
    method = fitted["calibrator"]["method"]
    if method == "identity":
        return np.clip(probabilities.astype("float64"), 0.0, 1.0)
    if method == "platt":
        model: LogisticRegression = fitted["calibrator"]["model"]
        return model.predict_proba(probabilities.reshape(-1, 1))[:, 1].astype("float64")
    if method == "isotonic":
        model: IsotonicRegression = fitted["calibrator"]["model"]
        return np.asarray(model.transform(probabilities), dtype="float64")
    raise ValueError(f"unsupported calibration method: {method}")


def tune_threshold_from_calibration_frame(
    frame: pd.DataFrame,
    *,
    default_threshold: float,
    threshold_grid: tuple[float, ...],
    enabled: bool,
) -> dict:
    if frame.empty or not enabled:
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
    best_rank: tuple[float, float, float, float] | None = None
    for threshold in candidates:
        signal_metrics = compute_action_metrics_from_calibration_frame(
            frame,
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
            signal_metrics.get("signal_coverage", 0.0),
            -abs(float(threshold) - float(default_threshold)),
        )
        if best_rank is None or rank > best_rank:
            best_rank = rank
            best_threshold = float(threshold)

    return {
        "enabled": True,
        "default_threshold": float(default_threshold),
        "selected_threshold": best_threshold,
        "selected_metrics": next(
            (item for item in candidate_reports if item["threshold"] == best_threshold),
            {},
        ),
        "candidates": candidate_reports,
    }


def compute_action_metrics_from_calibration_frame(
    frame: pd.DataFrame,
    *,
    decision_threshold: float,
) -> dict[str, float]:
    actionable_mask = frame["directional_prediction"] & (frame["calibrated_directional_prob"] >= float(decision_threshold))
    true_directional = frame["label_class"].isin(["up_signal", "down_signal"])
    actionable_correct = actionable_mask & (frame["directional_success"] == 1)

    actionable_count = int(actionable_mask.sum())
    true_directional_count = int(true_directional.sum())
    actionable_correct_count = int(actionable_correct.sum())
    coverage = actionable_count / len(frame) if len(frame) else 0.0
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


def compute_calibrated_probability_metrics(
    frame: pd.DataFrame,
    *,
    decision_threshold: float,
    calibration_bins: int,
) -> tuple[dict[str, float | int | None], dict]:
    directional_mask = frame["directional_prediction"]
    directional_probs = frame.loc[directional_mask, "calibrated_directional_prob"].to_numpy(dtype="float64")
    directional_targets = frame.loc[directional_mask, "directional_success"].to_numpy(dtype="float64")
    directional_count = int(directional_mask.sum())
    directional_brier = (
        float(np.mean((directional_probs - directional_targets) ** 2))
        if directional_count
        else None
    )
    directional_reliability = build_reliability_bins(
        directional_probs,
        directional_targets,
        lower_bound=0.0,
        calibration_bins=calibration_bins,
    )

    actionable_mask = directional_mask & (frame["calibrated_directional_prob"] >= float(decision_threshold))
    actionable_probs = frame.loc[actionable_mask, "calibrated_directional_prob"].to_numpy(dtype="float64")
    actionable_targets = frame.loc[actionable_mask, "directional_success"].to_numpy(dtype="float64")
    actionable_count = int(actionable_mask.sum())
    actionable_brier = (
        float(np.mean((actionable_probs - actionable_targets) ** 2))
        if actionable_count
        else None
    )
    actionable_reliability = build_reliability_bins(
        actionable_probs,
        actionable_targets,
        lower_bound=float(decision_threshold),
        calibration_bins=calibration_bins,
    )

    return (
        {
            "directional_brier_score": directional_brier,
            "directional_expected_calibration_error": directional_reliability["expected_calibration_error"],
            "directional_average_confidence": directional_reliability["average_confidence"],
            "directional_empirical_precision": directional_reliability["empirical_precision"],
            "directional_count": directional_count,
            "actionable_brier_score": actionable_brier,
            "actionable_expected_calibration_error": actionable_reliability["expected_calibration_error"],
            "actionable_average_confidence": actionable_reliability["average_confidence"],
            "actionable_empirical_precision": actionable_reliability["empirical_precision"],
            "actionable_count": actionable_count,
        },
        {
            "directional": directional_reliability,
            "actionable": actionable_reliability,
        },
    )


def build_reliability_bins(
    probabilities: np.ndarray,
    targets: np.ndarray,
    *,
    lower_bound: float,
    calibration_bins: int,
) -> dict:
    if probabilities.size == 0:
        return {
            "lower_bound": float(lower_bound),
            "bin_count": int(calibration_bins),
            "expected_calibration_error": None,
            "average_confidence": None,
            "empirical_precision": None,
            "bins": [],
        }

    edges = np.linspace(float(lower_bound), 1.0, max(2, calibration_bins) + 1)
    assignments = np.digitize(probabilities, edges[1:-1], right=False)
    bins: list[dict] = []
    ece = 0.0

    for idx in range(len(edges) - 1):
        mask = assignments == idx
        if not np.any(mask):
            continue

        bin_probs = probabilities[mask]
        bin_targets = targets[mask]
        confidence_mean = float(np.mean(bin_probs))
        empirical_precision = float(np.mean(bin_targets))
        gap = empirical_precision - confidence_mean
        count = int(mask.sum())
        bins.append(
            {
                "bin_index": idx,
                "range_min": float(edges[idx]),
                "range_max": float(edges[idx + 1]),
                "count": count,
                "confidence_mean": confidence_mean,
                "empirical_precision": empirical_precision,
                "gap": float(gap),
            }
        )
        ece += abs(gap) * (count / probabilities.size)

    return {
        "lower_bound": float(lower_bound),
        "bin_count": int(calibration_bins),
        "expected_calibration_error": float(ece),
        "average_confidence": float(np.mean(probabilities)),
        "empirical_precision": float(np.mean(targets)),
        "bins": bins,
    }


def _normalize_methods(methods: tuple[str, ...]) -> tuple[str, ...]:
    normalized = []
    for method in methods:
        candidate = str(method).strip().lower()
        if candidate and candidate not in normalized:
            normalized.append(candidate)
    return tuple(normalized or DEFAULT_CALIBRATION_METHODS)


def _best_calibration_method_name(results: dict[str, dict]) -> str:
    return max(
        results,
        key=lambda name: (
            results[name]["validation"]["signal_metrics"].get("actionable_f1", 0.0),
            -_none_safe(results[name]["validation"]["probability_metrics"].get("actionable_expected_calibration_error")),
            results[name]["validation"]["signal_metrics"].get("precision_actionable_signal", 0.0),
            results[name]["validation"]["signal_metrics"].get("signal_coverage", 0.0),
        ),
    )


def _select_production_calibration_candidate(
    results: dict[str, dict],
    *,
    fallback_name: str,
) -> tuple[str, dict]:
    passing = {name: report for name, report in results.items() if _calibration_production_gate(report)["passed"]}
    if passing:
        selected_name = max(passing, key=lambda name: _production_calibration_rank(results[name]))
        return selected_name, _calibration_production_gate(results[selected_name])
    return fallback_name, _calibration_production_gate(results[fallback_name])


def _production_calibration_rank(report: dict) -> tuple[float, float, float, float]:
    return (
        report["validation"]["signal_metrics"].get("actionable_f1", 0.0),
        report["validation"]["signal_metrics"].get("precision_actionable_signal", 0.0),
        report["validation"]["signal_metrics"].get("signal_coverage", 0.0),
        -_none_safe(report["validation"]["probability_metrics"].get("actionable_expected_calibration_error")),
    )


def _calibration_production_gate(report: dict) -> dict:
    precision_threshold = CALIBRATION_PRODUCTION_POLICY["hard_gates"][
        "validation.signal_metrics.precision_actionable_signal_min"
    ]
    coverage_threshold = CALIBRATION_PRODUCTION_POLICY["hard_gates"][
        "validation.signal_metrics.signal_coverage_min"
    ]
    ece_threshold = CALIBRATION_PRODUCTION_POLICY["hard_gates"][
        "validation.probability_metrics.actionable_expected_calibration_error_max"
    ]
    checks = {
        "precision_actionable_signal": {
            "actual": report["validation"]["signal_metrics"].get("precision_actionable_signal"),
            "required_min": precision_threshold,
            "passed": report["validation"]["signal_metrics"].get("precision_actionable_signal") is not None
            and float(report["validation"]["signal_metrics"].get("precision_actionable_signal")) >= float(precision_threshold),
        },
        "signal_coverage": {
            "actual": report["validation"]["signal_metrics"].get("signal_coverage"),
            "required_min": coverage_threshold,
            "passed": report["validation"]["signal_metrics"].get("signal_coverage") is not None
            and float(report["validation"]["signal_metrics"].get("signal_coverage")) >= float(coverage_threshold),
        },
        "actionable_expected_calibration_error": {
            "actual": report["validation"]["probability_metrics"].get("actionable_expected_calibration_error"),
            "required_max": ece_threshold,
            "passed": report["validation"]["probability_metrics"].get("actionable_expected_calibration_error") is not None
            and float(report["validation"]["probability_metrics"].get("actionable_expected_calibration_error")) <= float(ece_threshold),
        },
    }
    return {
        "passed": all(check["passed"] for check in checks.values()),
        "checks": checks,
    }


def _build_calibration_candidate_summary(
    method_name: str,
    report: dict,
    *,
    selection_mode: str,
    validation_gate: dict | None = None,
) -> dict:
    summary = {
        "method": method_name,
        "selection_mode": selection_mode,
        "selected_threshold": report.get("selected_threshold"),
        "fit_summary": report.get("fit_summary", {}),
        "validation": report["validation"],
        "test": report["test"],
    }
    if validation_gate is not None:
        summary["validation_gate"] = validation_gate
    return summary


def render_calibration_markdown_report(summary: dict) -> str:
    lines = [
        "# Calibration Audit Report",
        "",
        f"- model_dir: `{summary['model_dir']}`",
        f"- model_name: `{summary['model_name']}`",
        f"- base_selected_threshold: `{summary['base_selected_threshold']:.2f}`",
        f"- research_candidate: `{summary['research_candidate']['method']}`",
        f"- production_candidate: `{summary['production_candidate']['method']}`",
        f"- production_gate_passed: `{summary['production_candidate']['validation_gate']['passed']}`",
        "",
        "## Methods",
        "",
    ]

    for method_name, report in summary["methods"].items():
        lines.append(f"### {method_name}")
        lines.append("")
        if not report.get("available"):
            lines.append(f"- status: `unavailable`")
            lines.append(f"- reason: `{report['reason']}`")
            lines.append("")
            continue

        val = report["validation"]
        test = report["test"]
        gate = _calibration_production_gate(report)
        lines.extend(
            [
                f"- selected_threshold: `{report['selected_threshold']:.2f}`",
                f"- production_gate_passed: `{gate['passed']}`",
                f"- val actionable_f1: `{val['signal_metrics'].get('actionable_f1', 0):.4f}`",
                f"- val precision_actionable_signal: `{val['signal_metrics'].get('precision_actionable_signal', 0):.4f}`",
                f"- val signal_coverage: `{val['signal_metrics'].get('signal_coverage', 0):.4f}`",
                f"- val actionable_ece: `{_format_metric(val['probability_metrics'].get('actionable_expected_calibration_error'))}`",
                f"- test actionable_f1: `{test['signal_metrics'].get('actionable_f1', 0):.4f}`",
                f"- test precision_actionable_signal: `{test['signal_metrics'].get('precision_actionable_signal', 0):.4f}`",
                f"- test signal_coverage: `{test['signal_metrics'].get('signal_coverage', 0):.4f}`",
                f"- test actionable_ece: `{_format_metric(test['probability_metrics'].get('actionable_expected_calibration_error'))}`",
                "",
            ]
        )

    return "\n".join(lines) + "\n"


def _none_safe(value: float | None) -> float:
    return float(value) if value is not None else float("inf")


def _format_metric(value: float | None, *, precision: int = 4) -> str:
    if value is None:
        return "n/a"
    return f"{float(value):.{precision}f}"
