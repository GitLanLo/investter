package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"invest/backend/internal/app"
	"invest/backend/internal/config"
	"invest/backend/internal/domain"
	"invest/backend/internal/service"
)

func TestPolicyPromotionEndpoints(t *testing.T) {
	manifestPath := writeTestManifest(t)

	assetRepo := &testAssetRepo{
		assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		},
	}
	modelRepo := &testModelRepo{
		active: domain.ModelRegistryEntry{ModelVersion: "v0", ManifestPath: manifestPath},
		items: map[string]domain.ModelRegistryEntry{
			"v0": {ModelVersion: "v0", ManifestPath: manifestPath},
			"v1": {ModelVersion: "v1", ManifestPath: manifestPath},
		},
	}
	policyRepo := &testPolicyValidationRepo{
		items: []domain.PolicyValidationRun{
			{
				ID:            1,
				ModelName:     "hgb_multiclass",
				ModelVersion:  "v1",
				DecisionState: domain.PolicyDecisionShadowLive,
				Validation:    domain.PolicyValidationMetrics{ActionableF1: 0.6},
				Test:          domain.PolicyValidationMetrics{ActionableF1: 0.5},
			},
		},
	}
	signalRepo := &testSignalRepo{}
	outcomeRepo := &testSignalOutcomeRepo{}
	eventRepo := &testSignalEventRepo{}

	// Pre-populate enough outcomes to satisfy promotion rules (min 20 signals)
	for i := 0; i < 25; i++ {
		outcomeRepo.Upsert(context.Background(), domain.SignalOutcome{
			SignalRunID: int64(i + 1),
			IsHit:       true,
		})
		signalRepo.Create(context.Background(), domain.SignalRun{
			ID:          int64(i + 1),
			SignalState: domain.SignalStateActionable,
			Policy: &domain.SignalPolicySnapshot{
				ModelName:      "hgb_multiclass",
				DatasetVersion: "",
			},
		})
	}

	policyService := service.NewPolicyValidationService(
		policyRepo,
		service.NewResearchArtifactsService(t.TempDir(), t.TempDir()),
		signalRepo,
	).WithOutcomeData(assetRepo, &testMarketDataRepo{}, outcomeRepo)

	models := service.NewModelRegistryService(modelRepo)
	promotion := service.NewPolicyPromotionService(policyRepo, models, policyService, eventRepo, nil)

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Policy:    policyService,
				Promotion: promotion,
				Models:    models,
			},
		},
	})

	t.Run("promote success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ml/policy/validation-runs/1/promote", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected 204 NoContent, got %d body=%s", resp.Code, resp.Body.String())
		}

		if modelRepo.active.ModelVersion != "v1" {
			t.Fatalf("expected model v1 to be active in registry")
		}

		run, _ := policyRepo.ListLatest(context.Background(), 1)
		if run[0].DecisionState != domain.PolicyDecisionActive {
			t.Fatalf("expected policy run to be active, got %s", run[0].DecisionState)
		}
	})

	t.Run("rollback success", func(t *testing.T) {
		// Mock a previous run to rollback to
		policyRepo.items = append(policyRepo.items, domain.PolicyValidationRun{
			ID:            2,
			ModelVersion:  "v0",
			DecisionState: domain.PolicyDecisionArchived,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ml/policy/validation-runs/2/rollback", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected 204 NoContent, got %d body=%s", resp.Code, resp.Body.String())
		}

		if modelRepo.active.ModelVersion != "v0" {
			t.Fatalf("expected model v0 to be reactivated in registry")
		}
	})
}
