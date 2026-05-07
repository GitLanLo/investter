package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"invest/backend/internal/domain"
)

func TestAnalysisService_Run_RuntimeBlocking(t *testing.T) {
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

	assetRepo := &MockAssetRepo{
		Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "1h"},
		},
	}
	modelRepo := &MockModelRepo{
		Items: map[string]domain.ModelRegistryEntry{
			"available_v1": {ModelVersion: "available_v1", ManifestPath: pathAvailable, Timeframe: "1h", HorizonBars: 24},
			"metadata_v1":  {ModelVersion: "metadata_v1", ManifestPath: pathMetadata, Timeframe: "1h", HorizonBars: 24},
		},
	}
	signalRepo := &MockSignalRepo{}
	eventRepo := &MockSignalEventRepo{}
	svc := NewAnalysisService(assetRepo, modelRepo, signalRepo, eventRepo, nil)

	t.Run("specific metadata_only model returns blocked", func(t *testing.T) {
		_, err := svc.Run(context.Background(), RunAnalysisInput{
			AssetID:      "SBER",
			ModelVersion: "metadata_v1",
		})
		if err != ErrModelRuntimeBlocked {
			t.Fatalf("expected ErrModelRuntimeBlocked, got %v", err)
		}
	})

	t.Run("active available model succeeds", func(t *testing.T) {
		modelRepo.Active = modelRepo.Items["available_v1"]
		_, err := svc.Run(context.Background(), RunAnalysisInput{
			AssetID:      "SBER",
			ModelVersion: "active",
		})
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})

	t.Run("active metadata_only falls back to available with same config", func(t *testing.T) {
		modelRepo.Active = modelRepo.Items["metadata_v1"]

		run, err := svc.Run(context.Background(), RunAnalysisInput{
			AssetID:      "SBER",
			ModelVersion: "active",
		})
		if err != nil {
			t.Fatalf("expected fallback success, got %v", err)
		}
		if run.ModelVersion != "available_v1" {
			t.Fatalf("expected fallback to available_v1, got %s", run.ModelVersion)
		}
	})

	t.Run("active metadata_only without available fallback returns blocked", func(t *testing.T) {
		delete(modelRepo.Items, "available_v1")
		modelRepo.Active = modelRepo.Items["metadata_v1"]

		_, err := svc.Run(context.Background(), RunAnalysisInput{
			AssetID:      "SBER",
			ModelVersion: "active",
		})
		if err != ErrModelRuntimeBlocked {
			t.Fatalf("expected ErrModelRuntimeBlocked, got %v", err)
		}
	})
}

func TestAnalysisService_Run_UpsertsSignalEventsByAssetTimeModel(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "model_manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{
		"model_version": "available_v1",
		"model_type": "hgb_multiclass",
		"export_format": "joblib",
		"timeframe": "1h",
		"horizon_bars": 24,
		"feature_schema_version": "v1",
		"feature_columns": ["f1"],
		"decision_threshold": 0.5,
		"created_at": "2026-05-07T01:00:00Z"
	}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	assetRepo := &MockAssetRepo{
		Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "1h"},
		},
	}
	modelEntry := domain.ModelRegistryEntry{
		ModelVersion: "available_v1",
		ManifestPath: manifestPath,
		Timeframe:    "1h",
		HorizonBars:  24,
	}
	modelRepo := &MockModelRepo{
		Active: modelEntry,
		Items:  map[string]domain.ModelRegistryEntry{"available_v1": modelEntry},
	}
	signalRepo := &MockSignalRepo{}
	eventRepo := &MockSignalEventRepo{}
	svc := NewAnalysisService(assetRepo, modelRepo, signalRepo, eventRepo, nil)

	asOf := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if _, err := svc.Run(context.Background(), RunAnalysisInput{
			AssetID:   "SBER",
			AsOfTime:  asOf,
			Timeframe: "1h",
		}); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	if len(eventRepo.Events) != 1 {
		t.Fatalf("expected one idempotent signal event, got %d", len(eventRepo.Events))
	}
	event := eventRepo.Events[0]
	expectedSuffix := "|SBER|2026-05-07T10:00:00Z|available_v1"
	if !strings.HasSuffix(event.IdempotencyKey, expectedSuffix) {
		t.Fatalf("unexpected idempotency key %q", event.IdempotencyKey)
	}
}
