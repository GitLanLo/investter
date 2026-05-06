from __future__ import annotations

from pathlib import Path

from ml_core.pipelines.research_grid import ResearchGridConfig, run_research_grid


def test_run_research_grid_writes_completed_matrix(tmp_path: Path, monkeypatch) -> None:
    def fake_run_research_pipeline(provider, *, config):
        return {
            "dataset_version": config.dataset_version,
            "dataset_manifest": {
                "dataset_version": config.dataset_version,
                "tickers": ["SBER", "GAZP"],
                "train_range": {"rows": 10},
                "val_range": {"rows": 5},
                "test_range": {"rows": 4},
            },
            "research_summary": {
                "production_candidate": {
                    "model_name": "logreg_multiclass",
                    "selection_mode": "production_candidate",
                    "selected_threshold": 0.65,
                    "validation": {
                        "actionable_f1": 0.42,
                        "precision_actionable_signal": 0.51,
                        "signal_coverage": 0.22,
                        "actionable_expected_calibration_error": 0.12,
                    },
                    "test": {
                        "actionable_f1": 0.39,
                        "precision_actionable_signal": 0.49,
                        "signal_coverage": 0.2,
                        "actionable_expected_calibration_error": 0.14,
                    },
                },
                "models": {
                    "logreg_multiclass": {
                        "validation": {
                            "per_ticker": {
                                "SBER": {"actionable_f1": 0.4},
                                "GAZP": {"actionable_f1": 0.44},
                            }
                        }
                    }
                },
            },
        }

    monkeypatch.setattr("ml_core.pipelines.research_grid.run_research_pipeline", fake_run_research_pipeline)

    report = run_research_grid(
        object(),
        config=ResearchGridConfig(
            data_root=tmp_path / "data",
            dataset_output_root=tmp_path / "datasets",
            research_output_root=tmp_path / "research",
            grid_name="timeframe_horizon_matrix",
            tickers=["SBER", "GAZP"],
            timeframes=["5m", "15m"],
            horizons=[6],
        ),
    )

    matrix_path = tmp_path / "research" / "timeframe_horizon_matrix.json"
    assert matrix_path.exists()
    assert report["status"] == "completed"
    assert report["expected_result_count"] == 2
    assert report["completed_result_count"] == 2
    assert len(report["results"]) == 2
    assert report["results"][0]["rows"] == 19
    assert report["selected_configuration"]["timeframe"] == "5m"
