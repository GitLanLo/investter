package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"invest/backend/internal/domain"
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

func TestLoadManifest_RuntimeStatusMapping(t *testing.T) {
	tests := []struct {
		name          string
		manifestJSON  string
		wantRuntime   string
		wantModelType string
	}{
		{
			name:          "torchscript neural is metadata only",
			manifestJSON:  testNeuralManifestJSON(),
			wantRuntime:   RuntimeStatusMetadataOnly,
			wantModelType: "gru",
		},
		{
			name: "hgb joblib is available",
			manifestJSON: `{
  "model_version": "hgb_v1",
  "model_type": "hgb_multiclass",
  "task_type": "multiclass",
  "classes": ["down_signal", "no_trade", "up_signal"],
  "timeframe": "1h",
  "horizon_bars": 24,
  "feature_schema_version": "feature_v1",
  "feature_columns": ["f1", "f2"],
  "normalization_artifact_path": "normalizer.joblib",
  "export_format": "joblib",
  "model_artifact_path": "model.joblib",
  "metrics": {"f1": 0.5},
  "decision_threshold": 0.55,
  "calibration": "identity",
  "created_at": "2026-05-07T01:00:00Z",
  "source_dataset_version": "sprint7_1h_h24_20260506"
}`,
			wantRuntime:   RuntimeStatusAvailable,
			wantModelType: "hgb_multiclass",
		},
		{
			name: "unknown export is unavailable",
			manifestJSON: `{
  "model_version": "custom_v1",
  "model_type": "custom",
  "task_type": "multiclass",
  "classes": ["down_signal", "no_trade", "up_signal"],
  "timeframe": "1h",
  "horizon_bars": 24,
  "feature_schema_version": "feature_v1",
  "feature_columns": ["f1", "f2"],
  "normalization_artifact_path": "normalizer.custom",
  "export_format": "custom_runtime",
  "model_artifact_path": "model.custom",
  "metrics": {"f1": 0.5},
  "decision_threshold": 0.55,
  "calibration": "identity",
  "created_at": "2026-05-07T01:00:00Z",
  "source_dataset_version": "sprint7_1h_h24_20260506"
}`,
			wantRuntime:   RuntimeStatusUnavailable,
			wantModelType: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			manifestPath := filepath.Join(tmpDir, "model_manifest.json")
			if err := os.WriteFile(manifestPath, []byte(tt.manifestJSON), 0644); err != nil {
				t.Fatal(err)
			}

			manifest, err := LoadManifest(manifestPath)
			if err != nil {
				t.Fatalf("failed to load manifest: %v", err)
			}

			if manifest.RuntimeStatus != tt.wantRuntime {
				t.Fatalf("expected runtime status %s, got %s", tt.wantRuntime, manifest.RuntimeStatus)
			}
			if manifest.ModelType != tt.wantModelType {
				t.Fatalf("expected model type %s, got %s", tt.wantModelType, manifest.ModelType)
			}
		})
	}
}

func TestActivateModel(t *testing.T) {
	manifestAvailable := `{
  "model_version": "available_v1",
  "model_type": "hgb_multiclass",
  "export_format": "joblib",
  "timeframe": "1h",
  "horizon_bars": 24,
  "feature_schema_version": "v1",
  "feature_columns": ["f1"],
  "decision_threshold": 0.5,
  "created_at": "2026-05-07T01:00:00Z"
}`
	manifestMetadataOnly := `{
  "model_version": "metadata_v1",
  "model_type": "gru",
  "export_format": "torchscript",
  "timeframe": "1h",
  "horizon_bars": 24,
  "feature_schema_version": "v1",
  "feature_columns": ["f1"],
  "decision_threshold": 0.5,
  "created_at": "2026-05-07T01:00:00Z"
}`

	tmpDir := t.TempDir()
	pathAvailable := filepath.Join(tmpDir, "available", "model_manifest.json")
	os.MkdirAll(filepath.Dir(pathAvailable), 0755)
	os.WriteFile(pathAvailable, []byte(manifestAvailable), 0644)

	pathMetadata := filepath.Join(tmpDir, "metadata", "model_manifest.json")
	os.MkdirAll(filepath.Dir(pathMetadata), 0755)
	os.WriteFile(pathMetadata, []byte(manifestMetadataOnly), 0644)

	repo := &MockModelRepo{
		Items: map[string]domain.ModelRegistryEntry{
			"available_v1": {ModelVersion: "available_v1", ManifestPath: pathAvailable},
			"metadata_v1":  {ModelVersion: "metadata_v1", ManifestPath: pathMetadata},
		},
	}
	svc := NewModelRegistryService(repo)

	t.Run("blocks metadata_only", func(t *testing.T) {
		err := svc.Activate(context.Background(), "metadata_v1")
		if !errors.Is(err, ErrModelRuntimeBlocked) {
			t.Fatalf("expected ErrModelRuntimeBlocked, got %v", err)
		}
		if repo.Active.ModelVersion != "" {
			t.Fatalf("expected no model to be activated")
		}
	})

	t.Run("allows available", func(t *testing.T) {
		err := svc.Activate(context.Background(), "available_v1")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if repo.Active.ModelVersion != "available_v1" {
			t.Fatalf("expected available_v1 to be active")
		}
	})

	t.Run("demotes previous", func(t *testing.T) {
		// Mock active
		repo.Active = domain.ModelRegistryEntry{ModelVersion: "previous", Status: domain.ModelStatusActive}
		repo.Items["next"] = domain.ModelRegistryEntry{ModelVersion: "next", ManifestPath: pathAvailable}

		err := svc.Activate(context.Background(), "next")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if repo.Active.ModelVersion != "next" {
			t.Fatalf("expected next to be active")
		}
	})
}
