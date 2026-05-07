package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"invest/backend/internal/domain"
)

func TestPolicyPromotionService_Promote(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "model_manifest.json")
	os.WriteFile(manifestPath, []byte(`{
		"model_version": "v1",
		"model_type": "hgb_multiclass",
		"export_format": "joblib",
		"timeframe": "1h",
		"horizon_bars": 24,
		"feature_schema_version": "v1",
		"feature_columns": ["f1"],
		"decision_threshold": 0.5,
		"created_at": "2026-05-07T00:00:00Z"
	}`), 0644)

	policyRepo := &MockPolicyRepo{
		items: map[int64]domain.PolicyValidationRun{
			1: {
				ID:            1,
				ModelVersion:  "v1",
				DecisionState: domain.PolicyDecisionShadowLive,
				ModelName:     "hgb_multiclass",
			},
		},
	}
	modelRepo := &MockModelRepo{
		Items: map[string]domain.ModelRegistryEntry{
			"v1": {ModelVersion: "v1", ManifestPath: manifestPath},
		},
	}
	eventRepo := &MockSignalEventRepo{}
	
	validation := &MockPolicyValidationProvider{
		summary: domain.PolicyOutcomeSummary{
			ValidationRunID: 1,
			CanPromote:      true,
		},
	}
	
	svc := NewPolicyPromotionService(policyRepo, NewModelRegistryService(modelRepo), validation, eventRepo, nil)

	t.Run("promotion success", func(t *testing.T) {
		item := policyRepo.items[1]
		item.DecisionState = domain.PolicyDecisionShadowLive
		policyRepo.items[1] = item

		err := svc.Promote(context.Background(), 1)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		
		if modelRepo.Active.ModelVersion != "v1" {
			t.Fatalf("expected model v1 to be active")
		}
		if policyRepo.items[1].DecisionState != domain.PolicyDecisionActive {
			t.Fatalf("expected policy 1 to be active, got %s", policyRepo.items[1].DecisionState)
		}
	})

	t.Run("blocks when blockers exist", func(t *testing.T) {
		item := policyRepo.items[1]
		item.DecisionState = domain.PolicyDecisionShadowLive
		policyRepo.items[1] = item

		validation.summary.CanPromote = false
		validation.summary.PromotionBlockers = []domain.PolicyPromotionBlocker{{Code: "test"}}
		
		err := svc.Promote(context.Background(), 1)
		if err == nil || !errors.Is(err, ErrPromotionBlocked) {
			t.Fatalf("expected ErrPromotionBlocked, got %v", err)
		}
	})
}

type MockPolicyValidationProvider struct {
	summary domain.PolicyOutcomeSummary
}

func (m *MockPolicyValidationProvider) LoadOutcomeSummary(context.Context, int) (domain.PolicyOutcomeSummary, error) {
	return m.summary, nil
}

func (m *MockPolicyValidationProvider) LoadOutcomeSummaryByID(ctx context.Context, runID int64, limit int) (domain.PolicyOutcomeSummary, error) {
	return m.summary, nil
}
