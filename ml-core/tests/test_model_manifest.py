import json
from pathlib import Path
from ml_core.modeling.manifest import ModelManifest

def test_model_manifest_serialization(tmp_path: Path):
    manifest = ModelManifest(
        model_version="v1",
        model_family="gru",
        task="multiclass",
        classes=["down", "no_trade", "up"],
        input_timeframe="1h",
        prediction_horizon_bars=24,
        feature_schema_version="v1",
        feature_order=["f1", "f2"],
        export_format="onnx",
        model_artifact_path="model.onnx",
        artifact_sha256="fake_hash",
        metrics={"f1": 0.5},
        threshold=0.33,
        calibration="temperature",
        created_at="2026-05-06T00:00:00Z",
        source_dataset_version="sprint8_data",
        input_window_bars=96,
        input_tensor_shape=[96, 2],
        normalization="standard"
    )

    file_path = tmp_path / "model_manifest.json"
    manifest.save(file_path)

    loaded = ModelManifest.load(file_path)
    assert loaded.model_family == "gru"
    assert loaded.input_window_bars == 96
    assert loaded.input_tensor_shape == [96, 2]
    assert loaded.feature_order == ["f1", "f2"]
