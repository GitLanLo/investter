package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"invest/backend/internal/domain"
)

func TestMonitoringService_GetSummary(t *testing.T) {
	assetRepo := &MockAssetRepo{}
	modelRepo := &MockModelRepo{}
	signalRepo := &MockSignalRepo{}
	policyRepo := &MockPolicyRepo{}
	notificationRepo := &MockNotificationRepo{}
	jobRepo := &MockJobRunRepo{}

	svc := NewMonitoringService(assetRepo, modelRepo, signalRepo, policyRepo, notificationRepo, jobRepo, nil)

	t.Run("get summary basic", func(t *testing.T) {
		summary, err := svc.GetSummary(context.Background())
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if summary.GeneratedAt.IsZero() {
			t.Fatalf("expected generated_at to be set")
		}
		if summary.Status == "" {
			t.Fatalf("expected status to be set")
		}
		if summary.Items == nil {
			t.Fatalf("expected items array to be initialized")
		}
	})
}

func TestMonitoringService_GetSummary_ReportsFailedJobs(t *testing.T) {
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

	now := time.Now().UTC()
	assetRepo := &MockAssetRepo{
		Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", IsActive: true},
		},
	}
	modelRepo := &MockModelRepo{
		Active: domain.ModelRegistryEntry{ModelVersion: "available_v1", ManifestPath: manifestPath},
		Items:  map[string]domain.ModelRegistryEntry{"available_v1": {ModelVersion: "available_v1", ManifestPath: manifestPath}},
	}
	signalRepo := &MockSignalRepo{
		Latest: []domain.SignalRun{{AssetID: "SBER", AsOfTime: now}},
		Signals: map[string][]domain.SignalRun{
			"SBER": {{AssetID: "SBER", AsOfTime: now}},
		},
	}
	jobRepo := &MockJobRunRepo{
		Runs: []domain.JobRun{{JobType: "signals", Status: domain.JobStatusFailed, StartedAt: now}},
	}
	svc := NewMonitoringService(assetRepo, modelRepo, signalRepo, &MockPolicyRepo{}, &MockNotificationRepo{}, jobRepo, nil)

	summary, err := svc.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if summary.Status != "error" {
		t.Fatalf("expected error status, got %s", summary.Status)
	}
	if summary.Jobs.FailedLast24h != 1 {
		t.Fatalf("expected one failed job, got %d", summary.Jobs.FailedLast24h)
	}
	if summary.Severity.Critical == 0 {
		t.Fatalf("expected critical severity count")
	}
}
