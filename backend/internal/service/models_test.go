package service

import (
	"os"
	"path/filepath"
	"testing"
)

func testNeuralManifestJSON() string {
	return `{
  "model_version": "sprint8_gru_v1",
  "model_family": "gru",
  "task": "multiclass",
  "classes": ["down", "no_trade", "up"],
  "input_timeframe": "1h",
  "prediction_horizon_bars": 24,
  "feature_schema_version": "sprint8",
  "feature_order": ["f1", "f2"],
  "export_format": "torchscript",
  "model_artifact_path": "model.pt",
  "artifact_sha256": "fake",
  "metrics": {"f1": 0.5},
  "threshold": 0.33,
  "calibration": "none",
  "created_at": "2026-05-07T01:00:00Z",
  "source_dataset_version": "sprint8_data",
  "input_window_bars": 96,
  "input_tensor_shape": [96, 2],
  "normalization": "standard"
}`
}

func testNeuralManifestJSONWithNaiveCreatedAt() string {
	return `{
  "model_version": "sprint8_gru_v1",
  "model_family": "gru",
  "task": "multiclass",
  "classes": ["down", "no_trade", "up"],
  "input_timeframe": "1h",
  "prediction_horizon_bars": 24,
  "feature_schema_version": "sprint8",
  "feature_order": ["f1", "f2"],
  "export_format": "torchscript",
  "model_artifact_path": "model.pt",
  "artifact_sha256": "fake",
  "metrics": {"f1": 0.5},
  "threshold": 0.33,
  "calibration": "none",
  "created_at": "2026-05-07T01:00:00.123456",
  "source_dataset_version": "sprint8_data",
  "input_window_bars": 96,
  "input_tensor_shape": [96, 2],
  "normalization": "standard"
}`
}

func TestLoadManifest_Neural(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "model_manifest.json")
	if err := os.WriteFile(manifestPath, []byte(testNeuralManifestJSON()), 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if manifest.ModelVersion != "sprint8_gru_v1" {
		t.Errorf("expected model version sprint8_gru_v1, got %s", manifest.ModelVersion)
	}
	if manifest.ModelType != "gru" {
		t.Errorf("expected model type gru (aliased from model_family), got %s", manifest.ModelType)
	}
	if manifest.Timeframe != "1h" {
		t.Errorf("expected timeframe 1h (aliased from input_timeframe), got %s", manifest.Timeframe)
	}
	if manifest.RuntimeStatus != RuntimeStatusMetadataOnly {
		t.Errorf("expected runtime status metadata_only, got %s", manifest.RuntimeStatus)
	}
	if manifest.InputWindowBars != 96 {
		t.Errorf("expected input window bars 96, got %d", manifest.InputWindowBars)
	}
}

func TestLoadManifest_NaiveCreatedAt(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "model_manifest.json")
	if err := os.WriteFile(manifestPath, []byte(testNeuralManifestJSONWithNaiveCreatedAt()), 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if manifest.CreatedAt.IsZero() {
		t.Fatalf("expected created_at to be parsed")
	}
}
