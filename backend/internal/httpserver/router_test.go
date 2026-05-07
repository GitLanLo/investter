package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"

	"invest/backend/internal/app"
	"invest/backend/internal/config"
	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
	"invest/backend/internal/repository/filesystem"
	"invest/backend/internal/service"
)

func TestRootIndex(t *testing.T) {
	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB:        &sql.DB{},
		Container: app.Container{},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /: %d body=%s", resp.Code, resp.Body.String())
	}

	var body apiIndexResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode root index: %v", err)
	}
	if body.Service != "invest-backend" {
		t.Fatalf("unexpected service: %s", body.Service)
	}
	if body.Links["health"] != "/health" {
		t.Fatalf("expected health link, got %+v", body.Links)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	missingResp := httptest.NewRecorder()
	router.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("unexpected status for unknown path: %d", missingResp.Code)
	}
}

func TestAnalysisEndpoints(t *testing.T) {
	manifestPath := writeTestManifest(t)

	assetRepo := &testAssetRepo{
		assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		},
	}
	modelRepo := &testModelRepo{
		active: domain.ModelRegistryEntry{
			ModelVersion:         "baseline_stub_v1",
			ModelType:            "stub_deterministic_v1",
			Status:               domain.ModelStatusActive,
			Timeframe:            "5m",
			HorizonBars:          12,
			FeatureSchemaVersion: "feature_v1",
			ManifestPath:         manifestPath,
		},
	}
	signalRepo := &testSignalRepo{}
	eventRepo := &testSignalEventRepo{}

	router := NewRouter(config.Config{
		AppEnv:                   "test",
		OutcomeSchedulerInterval: 15 * time.Minute,
		OutcomeSchedulerLimit:    1000,
	}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Assets:    service.NewAssetService(assetRepo),
				Watchlist: service.NewWatchlistService(&testWatchlistRepo{}),
				Models:    service.NewModelRegistryService(modelRepo),
				Analysis:  service.NewAnalysisService(assetRepo, modelRepo, signalRepo, eventRepo, nil),
			},
		},
	})

	runReq := httptest.NewRequest(http.MethodPost, "/analysis/run", strings.NewReader(`{
		"asset_id": "SBER",
		"as_of_time": "2026-04-09T09:55:00Z",
		"model_version": "active",
		"timeframe": "5m"
	}`))
	runReq.Header.Set("Content-Type", "application/json")
	runResp := httptest.NewRecorder()
	router.ServeHTTP(runResp, runReq)

	if runResp.Code != http.StatusCreated {
		t.Fatalf("unexpected status for POST /analysis/run: %d body=%s", runResp.Code, runResp.Body.String())
	}

	var created signalDTO
	if err := json.Unmarshal(runResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created signal: %v", err)
	}
	if created.AssetID != "SBER" {
		t.Fatalf("unexpected asset_id: %s", created.AssetID)
	}
	if created.ModelVersion != "baseline_stub_v1" {
		t.Fatalf("unexpected model_version: %s", created.ModelVersion)
	}
	if created.Timeframe != "5m" {
		t.Fatalf("unexpected timeframe: %s", created.Timeframe)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/signals/latest?limit=10", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /signals/latest: %d body=%s", listResp.Code, listResp.Body.String())
	}

	var listed signalsResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode signals list: %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("expected 1 signal item, got %d", len(listed.Items))
	}
	if listed.Items[0].AssetID != "SBER" {
		t.Fatalf("unexpected listed asset_id: %s", listed.Items[0].AssetID)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/assets/SBER/signals?limit=10", nil)
	historyResp := httptest.NewRecorder()
	router.ServeHTTP(historyResp, historyReq)

	if historyResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /assets/SBER/signals: %d body=%s", historyResp.Code, historyResp.Body.String())
	}

	var history signalsResponse
	if err := json.Unmarshal(historyResp.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode signal history: %v", err)
	}
	if len(history.Items) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history.Items))
	}
	if history.Items[0].AssetID != "SBER" {
		t.Fatalf("unexpected history asset_id: %s", history.Items[0].AssetID)
	}
}

func TestModelManifestEndpointsExposeSprint9RuntimeState(t *testing.T) {
	manifestPath := writeTestNeuralManifest(t)
	modelRepo := &testModelRepo{
		active: domain.ModelRegistryEntry{
			ModelVersion:         "sprint8_gru_1h_h24_w96",
			ModelType:            "gru",
			Status:               domain.ModelStatusActive,
			Timeframe:            "1h",
			HorizonBars:          24,
			FeatureSchemaVersion: "sprint8",
			ManifestPath:         manifestPath,
		},
	}

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Models: service.NewModelRegistryService(modelRepo),
			},
		},
	})

	for _, path := range []string{"/ml/models/active", "/ml/models/sprint8_gru_1h_h24_w96"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("unexpected status for GET %s: %d body=%s", path, resp.Code, resp.Body.String())
		}

		var body mlModelManifestDTO
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode model manifest response: %v", err)
		}
		if body.ModelVersion != "sprint8_gru_1h_h24_w96" {
			t.Fatalf("unexpected model version: %+v", body)
		}
		if body.ModelFamily != "gru" || body.ModelType != "gru" {
			t.Fatalf("expected GRU model aliases, got %+v", body)
		}
		if body.InputWindowBars != 96 || len(body.InputTensorShape) != 2 || body.InputTensorShape[0] != 96 {
			t.Fatalf("unexpected neural input shape: %+v", body)
		}
		if body.RuntimeStatus != service.RuntimeStatusMetadataOnly {
			t.Fatalf("expected metadata_only runtime status, got %+v", body)
		}
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/ml/models/missing", nil)
	missingResp := httptest.NewRecorder()
	router.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing model, got %d body=%s", missingResp.Code, missingResp.Body.String())
	}
}

func TestModelActivation(t *testing.T) {
	manifestPath := writeTestManifest(t)
	neuralPath := writeTestNeuralManifest(t)

	modelRepo := &testModelRepo{
		active: domain.ModelRegistryEntry{ModelVersion: "baseline", ManifestPath: manifestPath},
		items: map[string]domain.ModelRegistryEntry{
			"neural": {ModelVersion: "neural", ManifestPath: neuralPath},
			"next":   {ModelVersion: "next", ManifestPath: manifestPath},
		},
	}
	eventRepo := &testSignalEventRepo{}

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Models:    service.NewModelRegistryService(modelRepo),
				Promotion: service.NewPolicyPromotionService(nil, service.NewModelRegistryService(modelRepo), nil, eventRepo, nil),
			},
		},
	})

	t.Run("blocks metadata_only activation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/ml/models/neural/activate", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d body=%s", resp.Code, resp.Body.String())
		}
		if modelRepo.active.ModelVersion != "baseline" {
			t.Fatalf("expected baseline to remain active")
		}
	})

	t.Run("allows available model activation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/ml/models/next/activate", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d body=%s", resp.Code, resp.Body.String())
		}
		if modelRepo.active.ModelVersion != "next" {
			t.Fatalf("expected next to be active")
		}
	})
}

func TestMarketDataEndpoints(t *testing.T) {
	dataRoot := t.TempDir()
	writeTestCandleParquet(t, filepath.Join(
		dataRoot,
		"raw",
		"candles",
		"ticker=SBER",
		"timeframe=5m",
		"date=2026-04-09",
		"part-20260409T095500Z-20260409T100500Z.parquet",
	), []testCandleRow{
		{Timestamp: mustTime(t, "2026-04-09T09:55:00Z"), Open: 301.1, High: 301.7, Low: 300.9, Close: 301.5, Volume: 123456, Ticker: "SBER", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
		{Timestamp: mustTime(t, "2026-04-09T10:00:00Z"), Open: 301.5, High: 301.9, Low: 301.2, Close: 301.8, Volume: 120000, Ticker: "SBER", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
		{Timestamp: mustTime(t, "2026-04-09T10:05:00Z"), Open: 301.8, High: 302.0, Low: 301.4, Close: 301.6, Volume: 118000, Ticker: "SBER", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
	})
	writeTestFactorParquet(t, filepath.Join(
		dataRoot,
		"raw",
		"factors",
		"factor=usdrub",
		"timeframe=5m",
		"date=2026-04-09",
		"part-20260409T095500Z-20260409T100000Z.parquet",
	), []testFactorRow{
		{Timestamp: mustTime(t, "2026-04-09T09:55:00Z"), Open: 92.1, High: 92.2, Low: 92.0, Close: 92.15, Volume: 10, Factor: "usdrub", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
		{Timestamp: mustTime(t, "2026-04-09T10:00:00Z"), Open: 92.15, High: 92.25, Low: 92.1, Close: 92.2, Volume: 12, Factor: "usdrub", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
	})
	writeTestFactorParquet(t, filepath.Join(
		dataRoot,
		"raw",
		"factors",
		"factor=brent",
		"timeframe=5m",
		"date=2026-04-09",
		"part-20260409T095500Z-20260409T100000Z.parquet",
	), []testFactorRow{
		{Timestamp: mustTime(t, "2026-04-09T09:55:00Z"), Open: 81.3, High: 81.5, Low: 81.2, Close: 81.42, Volume: 8, Factor: "brent", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
		{Timestamp: mustTime(t, "2026-04-09T10:00:00Z"), Open: 81.42, High: 81.6, Low: 81.3, Close: 81.55, Volume: 9, Factor: "brent", Timeframe: "5m", Source: "test", IngestedAt: mustTime(t, "2026-04-09T10:06:00Z")},
	})

	assetRepo := &testAssetRepo{
		assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		},
	}

	router := NewRouter(config.Config{
		AppEnv:                   "test",
		OutcomeSchedulerInterval: 15 * time.Minute,
		OutcomeSchedulerLimit:    1000,
	}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Assets:     service.NewAssetService(assetRepo),
				MarketData: service.NewMarketDataService(assetRepo, filesystem.NewMarketDataRepository(dataRoot)),
			},
		},
	})

	candlesReq := httptest.NewRequest(
		http.MethodGet,
		"/assets/SBER/candles?from=2026-04-09T09:55:00Z&to=2026-04-09T10:05:00Z&limit=2",
		nil,
	)
	candlesResp := httptest.NewRecorder()
	router.ServeHTTP(candlesResp, candlesReq)

	if candlesResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /assets/{id}/candles: %d body=%s", candlesResp.Code, candlesResp.Body.String())
	}

	var candles candlesResponse
	if err := json.Unmarshal(candlesResp.Body.Bytes(), &candles); err != nil {
		t.Fatalf("decode candles response: %v", err)
	}
	if candles.AssetID != "SBER" {
		t.Fatalf("unexpected asset_id: %s", candles.AssetID)
	}
	if len(candles.Items) != 2 {
		t.Fatalf("expected 2 candles after limit, got %d", len(candles.Items))
	}
	if candles.Items[0].Timestamp != "2026-04-09T10:00:00Z" || candles.Items[1].Timestamp != "2026-04-09T10:05:00Z" {
		t.Fatalf("unexpected candle timestamps: %+v", candles.Items)
	}

	factorsReq := httptest.NewRequest(
		http.MethodGet,
		"/assets/SBER/factors?from=2026-04-09T09:55:00Z&to=2026-04-09T10:00:00Z",
		nil,
	)
	factorsResp := httptest.NewRecorder()
	router.ServeHTTP(factorsResp, factorsReq)

	if factorsResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /assets/{id}/factors: %d body=%s", factorsResp.Code, factorsResp.Body.String())
	}

	var factors factorsResponse
	if err := json.Unmarshal(factorsResp.Body.Bytes(), &factors); err != nil {
		t.Fatalf("decode factors response: %v", err)
	}
	if len(factors.Items) != 4 {
		t.Fatalf("expected 4 factor items, got %d", len(factors.Items))
	}
	if factors.Items[0].Factor != "brent" || factors.Items[1].Factor != "usdrub" {
		t.Fatalf("unexpected factor ordering: %+v", factors.Items)
	}
}

func TestResearchOverviewEndpoint(t *testing.T) {
	dataRoot := t.TempDir()
	researchRoot := t.TempDir()

	writeJSONFile(t, filepath.Join(
		dataRoot,
		"datasets",
		"dataset_version=test_live",
		"manifest.json",
	), map[string]any{
		"dataset_version":        "test_live",
		"feature_schema_version": "feature_v1",
		"timeframe":              "5m",
		"horizon_bars":           12,
		"tickers":                []string{"SBER", "GAZP"},
		"train_range":            map[string]any{"rows": 100},
		"val_range":              map[string]any{"rows": 20},
		"test_range":             map[string]any{"rows": 20},
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"ablation_run",
		"summary.json",
	), map[string]any{
		"production_candidate": map[string]any{
			"scenario_name": "core_price_volume_only",
			"model_name":    "hgb_multiclass",
		},
		"research_candidate": map[string]any{
			"scenario_name": "no_regime",
			"model_name":    "hgb_multiclass",
		},
		"scenarios": map[string]any{
			"core_price_volume_only": map[string]any{},
		},
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"calibration_run",
		"summary.json",
	), map[string]any{
		"model_dir": "artifacts/research/test_live/core_price_volume_only/hgb_multiclass",
		"production_candidate": map[string]any{
			"method": "platt",
		},
		"research_candidate": map[string]any{
			"method": "identity",
		},
		"methods": map[string]any{
			"platt": map[string]any{"available": true},
		},
	})

	router := NewRouter(config.Config{
		AppEnv:                   "test",
		OutcomeSchedulerInterval: 15 * time.Minute,
		OutcomeSchedulerLimit:    1000,
	}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Research: service.NewResearchArtifactsService(dataRoot, researchRoot),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ml/research/overview", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/research/overview: %d body=%s", resp.Code, resp.Body.String())
	}

	var payload mlResearchOverviewResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode research overview: %v", err)
	}
	if payload.DatasetManifest["dataset_version"] != "test_live" {
		t.Fatalf("unexpected dataset version: %+v", payload.DatasetManifest)
	}
	if payload.ResearchSummary["production_candidate"].(map[string]any)["scenario_name"] != "core_price_volume_only" {
		t.Fatalf("unexpected research summary: %+v", payload.ResearchSummary)
	}
	if payload.CalibrationSummary["production_candidate"].(map[string]any)["method"] != "platt" {
		t.Fatalf("unexpected calibration summary: %+v", payload.CalibrationSummary)
	}
}

func TestResearchDocumentsEndpoint(t *testing.T) {
	dataRoot := t.TempDir()
	researchRoot := t.TempDir()

	writeJSONFile(t, filepath.Join(
		dataRoot,
		"datasets",
		"dataset_version=test_live",
		"manifest.json",
	), map[string]any{
		"dataset_version":        "test_live",
		"feature_schema_version": "feature_v1",
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"ablation_run",
		"summary.json",
	), map[string]any{
		"production_candidate": map[string]any{"scenario_name": "core_price_volume_only"},
		"research_candidate":   map[string]any{"scenario_name": "no_regime"},
		"scenarios":            map[string]any{"core_price_volume_only": map[string]any{}},
	})
	writeTextFile(t, filepath.Join(researchRoot, "ablation_run", "report.md"), "# Research report\n\nBest scenario.")
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"calibration_run",
		"summary.json",
	), map[string]any{
		"model_dir":            "artifacts/research/test_live/core_price_volume_only/hgb_multiclass",
		"production_candidate": map[string]any{"method": "platt"},
		"research_candidate":   map[string]any{"method": "identity"},
		"methods":              map[string]any{"platt": map[string]any{"available": true}},
	})
	writeTextFile(t, filepath.Join(researchRoot, "calibration_run", "report.md"), "# Calibration report\n\nPlatt wins.")

	router := NewRouter(config.Config{
		AppEnv:                   "test",
		OutcomeSchedulerInterval: 15 * time.Minute,
		OutcomeSchedulerLimit:    1000,
	}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Research: service.NewResearchArtifactsService(dataRoot, researchRoot),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ml/research/documents", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/research/documents: %d body=%s", resp.Code, resp.Body.String())
	}

	var payload mlResearchDocumentsResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode research documents: %v", err)
	}
	if len(payload.Items) != 5 {
		t.Fatalf("unexpected documents count: %d", len(payload.Items))
	}
	if payload.Items[0].Key != "dataset_manifest" {
		t.Fatalf("unexpected first document: %+v", payload.Items[0])
	}
	if payload.Items[3].Key != "research_report" || payload.Items[3].ContentType != "markdown" {
		t.Fatalf("unexpected research report document: %+v", payload.Items[3])
	}
	if !strings.Contains(payload.Items[3].Content, "Best scenario") {
		t.Fatalf("unexpected research report content: %q", payload.Items[3].Content)
	}
}

func TestProductionPolicyEndpoint(t *testing.T) {
	dataRoot := t.TempDir()
	researchRoot := t.TempDir()

	writeJSONFile(t, filepath.Join(
		dataRoot,
		"datasets",
		"dataset_version=test_live",
		"manifest.json",
	), map[string]any{
		"dataset_version":        "test_live",
		"feature_schema_version": "feature_v1",
		"timeframe":              "5m",
		"horizon_bars":           12,
		"train_range":            map[string]any{"rows": 100},
		"val_range":              map[string]any{"rows": 20},
		"test_range":             map[string]any{"rows": 30},
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"ablation_run",
		"summary.json",
	), map[string]any{
		"production_candidate": map[string]any{
			"scenario_name":      "core_price_volume_only",
			"model_name":         "hgb_multiclass",
			"selected_threshold": 0.55,
			"validation_gate":    map[string]any{"passed": true},
		},
		"research_candidate": map[string]any{"scenario_name": "no_regime"},
		"scenarios":          map[string]any{"core_price_volume_only": map[string]any{}},
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"calibration_run",
		"summary.json",
	), map[string]any{
		"model_dir": "artifacts/research/test_live/core_price_volume_only/hgb_multiclass",
		"production_candidate": map[string]any{
			"method":             "platt",
			"selected_threshold": 0.30,
			"validation_gate":    map[string]any{"passed": true},
			"validation": map[string]any{
				"actionable_f1":                         0.36,
				"precision_actionable_signal":           0.37,
				"signal_coverage":                       0.54,
				"actionable_expected_calibration_error": 0.01,
			},
			"test": map[string]any{
				"actionable_f1":                         0.27,
				"precision_actionable_signal":           0.25,
				"signal_coverage":                       0.48,
				"actionable_expected_calibration_error": 0.12,
			},
		},
		"research_candidate": map[string]any{"method": "identity"},
		"methods":            map[string]any{"platt": map[string]any{"available": true}},
	})

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Research: service.NewResearchArtifactsService(dataRoot, researchRoot),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ml/policy/production", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/policy/production: %d body=%s", resp.Code, resp.Body.String())
	}

	var payload mlProductionPolicyResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode production policy: %v", err)
	}
	if payload.PolicyStatus != "production_candidate" {
		t.Fatalf("unexpected policy status: %+v", payload)
	}
	if payload.ModelName != "hgb_multiclass" || payload.CalibrationMethod != "platt" {
		t.Fatalf("unexpected production policy identity: %+v", payload)
	}
	if payload.Threshold != 0.30 || payload.Validation.ActionableECE != 0.01 {
		t.Fatalf("unexpected production policy metrics: %+v", payload)
	}
	if payload.DatasetVersion != "test_live" || payload.TestRows != 30 {
		t.Fatalf("unexpected dataset metadata: %+v", payload)
	}
}

func TestInstrumentCatalogEndpoints(t *testing.T) {
	first1Min := mustTime(t, "2024-01-10T07:00:00Z")
	first1Day := mustTime(t, "2020-01-10T07:00:00Z")
	assetRepo := &testAssetRepo{assets: map[string]domain.Asset{}}
	watchlistRepo := &testWatchlistRepo{}
	instruments := &testInstrumentService{
		items: map[string]domain.TinkoffInstrument{
			"uid-sber": {
				UID:                 "uid-sber",
				Figi:                "figi-sber",
				Ticker:              "SBER",
				ClassCode:           "TQBR",
				Isin:                "RU0009029540",
				Lot:                 10,
				Currency:            "rub",
				Name:                "Sberbank",
				Exchange:            "MOEX",
				InstrumentType:      "share",
				APITradeAvailable:   true,
				First1MinCandleDate: &first1Min,
				First1DayCandleDate: &first1Day,
			},
		},
	}

	router := NewRouter(config.Config{
		AppEnv:                   "test",
		OutcomeSchedulerInterval: 15 * time.Minute,
		OutcomeSchedulerLimit:    1000,
	}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Assets:      service.NewAssetService(assetRepo),
				Watchlist:   service.NewWatchlistService(watchlistRepo),
				Instruments: instruments,
			},
		},
	})

	searchReq := httptest.NewRequest(http.MethodGet, "/instruments/search?query=sbe", nil)
	searchResp := httptest.NewRecorder()
	router.ServeHTTP(searchResp, searchReq)

	if searchResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /instruments/search: %d body=%s", searchResp.Code, searchResp.Body.String())
	}

	var searchResult instrumentsSearchResponse
	if err := json.Unmarshal(searchResp.Body.Bytes(), &searchResult); err != nil {
		t.Fatalf("decode instruments search: %v", err)
	}
	if len(searchResult.Items) != 1 || searchResult.Items[0].UID != "uid-sber" {
		t.Fatalf("unexpected search result: %+v", searchResult)
	}

	detailsReq := httptest.NewRequest(http.MethodGet, "/instruments/uid-sber", nil)
	detailsResp := httptest.NewRecorder()
	router.ServeHTTP(detailsResp, detailsReq)

	if detailsResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /instruments/{uid}: %d body=%s", detailsResp.Code, detailsResp.Body.String())
	}

	var instrument instrumentDTO
	if err := json.Unmarshal(detailsResp.Body.Bytes(), &instrument); err != nil {
		t.Fatalf("decode instrument details: %v", err)
	}
	if instrument.Ticker != "SBER" || instrument.ClassCode != "TQBR" {
		t.Fatalf("unexpected instrument payload: %+v", instrument)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/watchlist", strings.NewReader(`{"instrument_uid":"uid-sber","position":2}`))
	addReq.Header.Set("Content-Type", "application/json")
	addResp := httptest.NewRecorder()
	router.ServeHTTP(addResp, addReq)

	if addResp.Code != http.StatusCreated {
		t.Fatalf("unexpected status for POST /watchlist: %d body=%s", addResp.Code, addResp.Body.String())
	}

	var created watchlistResponse
	if err := json.Unmarshal(addResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode watchlist create response: %v", err)
	}
	if len(created.Items) != 1 || created.Items[0].AssetID != "uid-sber" {
		t.Fatalf("unexpected watchlist payload: %+v", created)
	}
	if created.Items[0].Asset == nil || created.Items[0].Asset.Ticker != "SBER" || !created.Items[0].Asset.APITradeAvailable {
		t.Fatalf("expected enriched asset metadata in watchlist item, got %+v", created.Items[0])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/watchlist", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /watchlist: %d body=%s", listResp.Code, listResp.Body.String())
	}

	var listed watchlistResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode watchlist response: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Asset == nil || listed.Items[0].Asset.InstrumentUID != "uid-sber" {
		t.Fatalf("unexpected enriched watchlist response: %+v", listed)
	}
}

func TestPolicyValidationRunsEndpoint(t *testing.T) {
	restoreNow := service.SetPolicyValidationNowForTest(mustTime(t, "2026-04-24T10:30:00Z"))
	defer restoreNow()

	dataRoot := t.TempDir()
	researchRoot := t.TempDir()

	writeJSONFile(t, filepath.Join(
		dataRoot,
		"datasets",
		"dataset_version=test_live",
		"manifest.json",
	), map[string]any{
		"dataset_version":        "test_live",
		"feature_schema_version": "feature_v1",
		"timeframe":              "5m",
		"horizon_bars":           12,
		"train_range":            map[string]any{"rows": 100},
		"val_range":              map[string]any{"rows": 20},
		"test_range":             map[string]any{"rows": 30},
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"ablation_run",
		"summary.json",
	), map[string]any{
		"production_candidate": map[string]any{
			"scenario_name":      "core_price_volume_only",
			"model_name":         "rf_multiclass",
			"selected_threshold": 0.55,
			"validation_gate":    map[string]any{"passed": true},
		},
		"research_candidate": map[string]any{"scenario_name": "no_regime"},
		"scenarios":          map[string]any{"core_price_volume_only": map[string]any{}},
	})
	writeJSONFile(t, filepath.Join(
		researchRoot,
		"calibration_run",
		"summary.json",
	), map[string]any{
		"model_dir": "artifacts/research/test_live/core_price_volume_only/rf_multiclass",
		"production_candidate": map[string]any{
			"method":             "platt",
			"selected_threshold": 0.30,
			"validation_gate":    map[string]any{"passed": true},
			"validation": map[string]any{
				"actionable_f1":                         0.36,
				"precision_actionable_signal":           0.37,
				"signal_coverage":                       0.54,
				"actionable_expected_calibration_error": 0.01,
			},
			"test": map[string]any{
				"actionable_f1":                         0.27,
				"precision_actionable_signal":           0.25,
				"signal_coverage":                       0.48,
				"actionable_expected_calibration_error": 0.12,
			},
		},
		"research_candidate": map[string]any{"method": "identity"},
		"methods":            map[string]any{"platt": map[string]any{"available": true}},
	})

	policyRepo := &testPolicyValidationRepo{}
	signalRepo := &testSignalRepo{}
	outcomeRepo := &testSignalOutcomeRepo{}
	eventRepo := &testSignalEventRepo{}
	jobRepo := &testJobRunRepo{}
	modelRepo := &testModelRepo{}
	assetRepo := &testAssetRepo{
		assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		},
	}
	marketRepo := &testMarketDataRepo{
		candles: map[string][]domain.Candle{
			"SBER": {
				{Timestamp: mustTime(t, "2026-04-22T10:00:00Z"), Close: 100, Ticker: "SBER", Timeframe: "5m"},
				{Timestamp: mustTime(t, "2026-04-22T10:05:00Z"), Close: 101, Ticker: "SBER", Timeframe: "5m"},
				{Timestamp: mustTime(t, "2026-04-22T10:10:00Z"), Close: 102, Ticker: "SBER", Timeframe: "5m"},
				{Timestamp: mustTime(t, "2026-04-22T10:15:00Z"), Close: 103, Ticker: "SBER", Timeframe: "5m"},
			},
		},
	}
	policyService := service.NewPolicyValidationService(
		policyRepo,
		service.NewResearchArtifactsService(dataRoot, researchRoot),
		signalRepo,
	).WithOutcomeData(assetRepo, marketRepo, outcomeRepo)
	models := service.NewModelRegistryService(modelRepo)
	router := NewRouter(config.Config{
		AppEnv:                   "test",
		OutcomeSchedulerInterval: 15 * time.Minute,
		OutcomeSchedulerLimit:    1000,
	}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Policy:    policyService,
				Promotion: service.NewPolicyPromotionService(policyRepo, models, policyService, eventRepo, nil),
				Jobs:      service.NewJobService(jobRepo, policyService),
			},
		},
	})

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/ml/policy/validation-runs",
		strings.NewReader(`{"notes":"shadow candidate"}`),
	)
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected status for POST /ml/policy/validation-runs: %d body=%s", createResp.Code, createResp.Body.String())
	}

	var created mlPolicyValidationRunDTO
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created policy validation run: %v", err)
	}
	if created.ModelName != "rf_multiclass" || created.DecisionState != domain.PolicyDecisionCandidate {
		t.Fatalf("unexpected created validation run: %+v", created)
	}
	if created.Notes != "shadow candidate" || created.Validation.ActionableECE != 0.01 {
		t.Fatalf("unexpected validation run details: %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/ml/policy/validation-runs?limit=5", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/policy/validation-runs: %d body=%s", listResp.Code, listResp.Body.String())
	}

	var listed mlPolicyValidationRunsResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode policy validation runs: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != created.ID {
		t.Fatalf("unexpected validation run list: %+v", listed)
	}

	updateReq := httptest.NewRequest(
		http.MethodPatch,
		"/ml/policy/validation-runs/1",
		strings.NewReader(`{"decision_state":"shadow_live","notes":"approved for shadow live"}`),
	)
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)

	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for PATCH /ml/policy/validation-runs/1: %d body=%s", updateResp.Code, updateResp.Body.String())
	}

	var updated mlPolicyValidationRunDTO
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated policy validation run: %v", err)
	}
	if updated.DecisionState != domain.PolicyDecisionShadowLive || updated.Notes != "approved for shadow live" {
		t.Fatalf("unexpected updated validation run: %+v", updated)
	}

	_, _ = signalRepo.Create(context.Background(), domain.SignalRun{
		AssetID:           "SBER",
		ModelVersion:      "baseline_stub_v1",
		AsOfTime:          mustTime(t, "2026-04-22T10:00:00Z"),
		SignalState:       domain.SignalStateActionable,
		SignalDirection:   domain.SignalDirectionUp,
		SignalProbability: 0.71,
		ClassProbabilities: domain.SignalClassProbabilities{
			Up:      0.71,
			Down:    0.12,
			NoTrade: 0.17,
		},
		Threshold:   0.65,
		Timeframe:   "5m",
		HorizonBars: 2,
		Policy: &domain.SignalPolicySnapshot{
			PolicyStatus:      "production_candidate",
			ModelName:         "rf_multiclass",
			ScenarioName:      "core_price_volume_only",
			CalibrationMethod: "platt",
			Threshold:         0.30,
			DatasetVersion:    "test_live",
		},
	})
	_, _ = signalRepo.Create(context.Background(), domain.SignalRun{
		AssetID:           "SBER",
		ModelVersion:      "baseline_stub_v1",
		AsOfTime:          mustTime(t, "2026-04-22T10:10:00Z"),
		SignalState:       domain.SignalStateActionable,
		SignalDirection:   domain.SignalDirectionDown,
		SignalProbability: 0.68,
		ClassProbabilities: domain.SignalClassProbabilities{
			Up:      0.14,
			Down:    0.68,
			NoTrade: 0.18,
		},
		Threshold:   0.65,
		Timeframe:   "5m",
		HorizonBars: 2,
		Policy: &domain.SignalPolicySnapshot{
			PolicyStatus:      "production_candidate",
			ModelName:         "rf_multiclass",
			ScenarioName:      "core_price_volume_only",
			CalibrationMethod: "platt",
			Threshold:         0.30,
			DatasetVersion:    "test_live",
		},
	})

	shadowReq := httptest.NewRequest(http.MethodGet, "/ml/policy/shadow-summary", nil)
	shadowResp := httptest.NewRecorder()
	router.ServeHTTP(shadowResp, shadowReq)

	if shadowResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/policy/shadow-summary: %d body=%s", shadowResp.Code, shadowResp.Body.String())
	}

	var shadow mlPolicyShadowSummaryDTO
	if err := json.Unmarshal(shadowResp.Body.Bytes(), &shadow); err != nil {
		t.Fatalf("decode policy shadow summary: %v", err)
	}
	if shadow.ValidationRunID != 1 || shadow.SignalsTotal != 2 || shadow.ActionableSignals != 2 {
		t.Fatalf("unexpected shadow summary: %+v", shadow)
	}

	outcomeMaterializeReq := httptest.NewRequest(http.MethodPost, "/ml/policy/outcomes?limit=10", nil)
	outcomeMaterializeResp := httptest.NewRecorder()
	router.ServeHTTP(outcomeMaterializeResp, outcomeMaterializeReq)

	if outcomeMaterializeResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for POST /ml/policy/outcomes: %d body=%s", outcomeMaterializeResp.Code, outcomeMaterializeResp.Body.String())
	}

	if len(outcomeRepo.items) != 1 {
		t.Fatalf("expected 1 persisted outcome, got %d", len(outcomeRepo.items))
	}

	outcomeReq := httptest.NewRequest(http.MethodGet, "/ml/policy/outcomes", nil)
	outcomeResp := httptest.NewRecorder()
	router.ServeHTTP(outcomeResp, outcomeReq)

	if outcomeResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/policy/outcomes: %d body=%s", outcomeResp.Code, outcomeResp.Body.String())
	}

	var outcomes mlPolicyOutcomeSummaryDTO
	if err := json.Unmarshal(outcomeResp.Body.Bytes(), &outcomes); err != nil {
		t.Fatalf("decode policy outcomes: %v", err)
	}
	if outcomes.ValidationRunID != 1 || outcomes.MaturedSignals != 1 || outcomes.PendingSignals != 1 {
		t.Fatalf("unexpected outcome maturity summary: %+v", outcomes)
	}
	if outcomes.OverduePendingSignals != 1 {
		t.Fatalf("expected one overdue pending outcome, got %+v", outcomes)
	}
	if outcomes.HitSignals != 1 || outcomes.MissSignals != 0 || outcomes.RealizedPrecision != 1 {
		t.Fatalf("unexpected realized precision summary: %+v", outcomes)
	}
	if outcomes.CanPromote {
		t.Fatalf("expected promote to stay blocked, got %+v", outcomes)
	}
	if len(outcomes.PromotionBlockers) == 0 {
		t.Fatalf("expected promotion blockers, got %+v", outcomes)
	}
	if outcomes.LastSignalAt != "2026-04-22T10:10:00Z" {
		t.Fatalf("unexpected last signal timestamp: %+v", outcomes)
	}
	if outcomes.AverageActionReturnPct <= 0 {
		t.Fatalf("expected positive action return, got %+v", outcomes)
	}

	outcomeHistoryReq := httptest.NewRequest(http.MethodGet, "/ml/policy/outcomes/history?limit=10", nil)
	outcomeHistoryResp := httptest.NewRecorder()
	router.ServeHTTP(outcomeHistoryResp, outcomeHistoryReq)

	if outcomeHistoryResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /ml/policy/outcomes/history: %d body=%s", outcomeHistoryResp.Code, outcomeHistoryResp.Body.String())
	}

	var history mlPolicyOutcomeHistoryResponse
	if err := json.Unmarshal(outcomeHistoryResp.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode policy outcome history: %v", err)
	}
	if len(history.Items) != 1 {
		t.Fatalf("expected 1 outcome history item, got %+v", history)
	}
	if history.Items[0].SignalDirection != domain.SignalDirectionUp || history.Items[0].AssetID != "SBER" {
		t.Fatalf("unexpected outcome history item: %+v", history.Items[0])
	}

	jobReq := httptest.NewRequest(http.MethodPost, "/jobs/outcomes/materialize?limit=10", nil)
	jobResp := httptest.NewRecorder()
	router.ServeHTTP(jobResp, jobReq)

	if jobResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for POST /jobs/outcomes/materialize: %d body=%s", jobResp.Code, jobResp.Body.String())
	}

	var job jobRunDTO
	if err := json.Unmarshal(jobResp.Body.Bytes(), &job); err != nil {
		t.Fatalf("decode job run response: %v", err)
	}
	if job.JobType != service.JobTypeOutcomesMaterialize || job.Status != domain.JobStatusSucceeded {
		t.Fatalf("unexpected job run payload: %+v", job)
	}

	jobsListReq := httptest.NewRequest(http.MethodGet, "/jobs/runs?limit=5", nil)
	jobsListResp := httptest.NewRecorder()
	router.ServeHTTP(jobsListResp, jobsListReq)

	if jobsListResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /jobs/runs: %d body=%s", jobsListResp.Code, jobsListResp.Body.String())
	}

	var jobs jobRunsResponse
	if err := json.Unmarshal(jobsListResp.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode jobs list: %v", err)
	}
	if len(jobs.Items) == 0 || jobs.Items[0].JobType != service.JobTypeOutcomesMaterialize {
		t.Fatalf("unexpected jobs list payload: %+v", jobs)
	}

	schedulerReq := httptest.NewRequest(http.MethodGet, "/jobs/scheduler", nil)
	schedulerResp := httptest.NewRecorder()
	router.ServeHTTP(schedulerResp, schedulerReq)

	if schedulerResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for GET /jobs/scheduler: %d body=%s", schedulerResp.Code, schedulerResp.Body.String())
	}

	var scheduler jobSchedulerStatusDTO
	if err := json.Unmarshal(schedulerResp.Body.Bytes(), &scheduler); err != nil {
		t.Fatalf("decode scheduler status: %v", err)
	}
	if scheduler.Enabled || scheduler.Interval != "15m0s" || scheduler.Limit != 1000 || scheduler.RunOnStart {
		t.Fatalf("unexpected scheduler payload: %+v", scheduler)
	}
}

func TestSprint6Endpoints(t *testing.T) {
	assetRepo := &testAssetRepo{
		assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		},
	}
	watchlistRepo := &testWatchlistRepo{
		items: []domain.WatchlistItem{{AssetID: "SBER"}},
	}
	jobRepo := &testJobRunRepo{}
	instruments := &testInstrumentService{
		items: map[string]domain.TinkoffInstrument{
			"uid-sber": {UID: "uid-sber", Ticker: "SBER", Name: "Sberbank"},
		},
	}

	refreshSvc := service.NewWatchlistRefreshService(
		watchlistRepo, assetRepo, &testMarketDataRepo{}, jobRepo, &testSignalRepo{},
		nil, instruments, nil,
	)

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				WatchlistRefresh: refreshSvc,
			},
		},
	})

	// Test aliases
	endpoints := []string{"/jobs/data-refresh", "/jobs/signals/run"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodPost, ep, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("unexpected status for %s: %d body=%s", ep, resp.Code, resp.Body.String())
		}
	}
}

func TestCORSPreflight(t *testing.T) {
	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB:        &sql.DB{},
		Container: app.Container{},
	})

	req := httptest.NewRequest(http.MethodOptions, "/analysis/run", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("unexpected status for CORS preflight: %d", resp.Code)
	}
	if resp.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("unexpected allow-origin header: %q", resp.Header().Get("Access-Control-Allow-Origin"))
	}
	if resp.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected allow-methods header")
	}
}

type testAssetRepo struct {
	assets map[string]domain.Asset
}

func (r *testAssetRepo) List(context.Context) ([]domain.Asset, error) {
	items := make([]domain.Asset, 0, len(r.assets))
	for _, asset := range r.assets {
		items = append(items, asset)
	}
	return items, nil
}

func (r *testAssetRepo) GetByID(_ context.Context, id string) (domain.Asset, error) {
	asset, ok := r.assets[id]
	if !ok {
		return domain.Asset{}, sql.ErrNoRows
	}
	return asset, nil
}

func (r *testAssetRepo) Upsert(_ context.Context, asset domain.Asset) error {
	if r.assets == nil {
		r.assets = map[string]domain.Asset{}
	}
	r.assets[asset.ID] = asset
	return nil
}

type testWatchlistRepo struct {
	items []domain.WatchlistItem
}

func (r *testWatchlistRepo) GetOrCreateByName(context.Context, string) (domain.Watchlist, error) {
	return domain.Watchlist{ID: 1, Name: "default"}, nil
}

func (r *testWatchlistRepo) ListItems(_ context.Context, _ int64) ([]domain.WatchlistItem, error) {
	out := make([]domain.WatchlistItem, len(r.items))
	copy(out, r.items)
	return out, nil
}

func (r *testWatchlistRepo) AddItem(_ context.Context, watchlistID int64, assetID string, position int) error {
	for index := range r.items {
		if r.items[index].AssetID == assetID {
			r.items[index].Position = position
			return nil
		}
	}
	r.items = append(r.items, domain.WatchlistItem{
		ID:          int64(len(r.items) + 1),
		WatchlistID: watchlistID,
		AssetID:     assetID,
		Position:    position,
		CreatedAt:   time.Now().UTC(),
	})
	return nil
}

type testInstrumentService struct {
	items map[string]domain.TinkoffInstrument
}

func (s *testInstrumentService) FindInstrument(_ context.Context, query string) ([]domain.TinkoffInstrument, error) {
	out := make([]domain.TinkoffInstrument, 0, len(s.items))
	for _, item := range s.items {
		if strings.Contains(strings.ToLower(item.Ticker), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(item.Name), strings.ToLower(query)) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *testInstrumentService) GetInstrumentByUID(_ context.Context, uid string) (domain.TinkoffInstrument, error) {
	item, ok := s.items[uid]
	if !ok {
		return domain.TinkoffInstrument{}, sql.ErrNoRows
	}
	return item, nil
}

func (s *testInstrumentService) GetCandles(ctx context.Context, uid string, timeframe string, from time.Time, to time.Time) ([]domain.Candle, error) {
	return nil, nil
}

func (s *testInstrumentService) IsMarketOpen(ctx context.Context, exchange string) (bool, error) {
	return true, nil
}

type testModelRepo struct {
	active domain.ModelRegistryEntry
	items  map[string]domain.ModelRegistryEntry
}

func (r *testModelRepo) List(context.Context) ([]domain.ModelRegistryEntry, error) {
	items := make([]domain.ModelRegistryEntry, 0, 1+len(r.items))
	if r.active.ModelVersion != "" {
		items = append(items, r.active)
	}
	for _, item := range r.items {
		items = append(items, item)
	}
	return items, nil
}

func (r *testModelRepo) GetActive(context.Context) (domain.ModelRegistryEntry, error) {
	if r.active.ModelVersion == "" {
		return domain.ModelRegistryEntry{}, sql.ErrNoRows
	}
	return r.active, nil
}

func (r *testModelRepo) GetByVersion(_ context.Context, version string) (domain.ModelRegistryEntry, error) {
	if version == r.active.ModelVersion {
		return r.active, nil
	}
	if item, ok := r.items[version]; ok {
		return item, nil
	}
	return domain.ModelRegistryEntry{}, sql.ErrNoRows
}

func (r *testModelRepo) Register(context.Context, domain.ModelRegistryEntry) error {
	return nil
}

func (r *testModelRepo) Activate(_ context.Context, version string) error {
	if version == r.active.ModelVersion {
		return nil
	}
	if item, ok := r.items[version]; ok {
		r.active = item
		r.active.Status = domain.ModelStatusActive
		return nil
	}
	return sql.ErrNoRows
}

type testSignalRepo struct {
	items []domain.SignalRun
}

type testCandleRow struct {
	Timestamp  time.Time `parquet:"timestamp"`
	Open       float64   `parquet:"open"`
	High       float64   `parquet:"high"`
	Low        float64   `parquet:"low"`
	Close      float64   `parquet:"close"`
	Volume     int64     `parquet:"volume"`
	Ticker     string    `parquet:"ticker"`
	Timeframe  string    `parquet:"timeframe"`
	Source     string    `parquet:"source"`
	IngestedAt time.Time `parquet:"ingested_at"`
}

type testFactorRow struct {
	Timestamp  time.Time `parquet:"timestamp"`
	Open       float64   `parquet:"open"`
	High       float64   `parquet:"high"`
	Low        float64   `parquet:"low"`
	Close      float64   `parquet:"close"`
	Volume     int64     `parquet:"volume"`
	Factor     string    `parquet:"factor_alias"`
	Timeframe  string    `parquet:"timeframe"`
	Source     string    `parquet:"source"`
	IngestedAt time.Time `parquet:"ingested_at"`
}

func (r *testSignalRepo) Create(_ context.Context, run domain.SignalRun) (domain.SignalRun, error) {
	run.ID = int64(len(r.items) + 1)
	run.CreatedAt = time.Now().UTC()
	r.items = append(r.items, run)
	return run, nil
}

func (r *testSignalRepo) ListLatest(_ context.Context, limit int) ([]domain.SignalRun, error) {
	if limit > len(r.items) {
		limit = len(r.items)
	}
	out := make([]domain.SignalRun, 0, limit)
	for i := len(r.items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.items[i])
	}
	return out, nil
}

func (r *testSignalRepo) ListByAsset(_ context.Context, assetID string, limit int) ([]domain.SignalRun, error) {
	filtered := make([]domain.SignalRun, 0, len(r.items))
	for i := len(r.items) - 1; i >= 0; i-- {
		if r.items[i].AssetID == assetID {
			filtered = append(filtered, r.items[i])
		}
		if len(filtered) == limit {
			break
		}
	}
	return filtered, nil
}

func (r *testSignalRepo) ListByPolicySnapshot(
	_ context.Context,
	modelName string,
	calibrationMethod string,
	datasetVersion string,
	limit int,
) ([]domain.SignalRun, error) {
	filtered := make([]domain.SignalRun, 0, len(r.items))
	for i := len(r.items) - 1; i >= 0; i-- {
		policy := r.items[i].Policy
		if policy != nil &&
			policy.ModelName == modelName &&
			policy.CalibrationMethod == calibrationMethod &&
			policy.DatasetVersion == datasetVersion {
			filtered = append(filtered, r.items[i])
		}
		if len(filtered) == limit {
			break
		}
	}
	return filtered, nil
}

type testPolicyValidationRepo struct {
	items []domain.PolicyValidationRun
}

type testMarketDataRepo struct {
	candles map[string][]domain.Candle
	factors []domain.FactorBar
}

type testSignalOutcomeRepo struct {
	items map[int64]domain.SignalOutcome
}

type testJobRunRepo struct {
	items []domain.JobRun
}

func (r *testMarketDataRepo) ListCandles(
	_ context.Context,
	ticker string,
	_ string,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.Candle, error) {
	items := make([]domain.Candle, 0, len(r.candles[ticker]))
	for _, candle := range r.candles[ticker] {
		if candle.Timestamp.Before(from) || candle.Timestamp.After(to) {
			continue
		}
		items = append(items, candle)
		if limit > 0 && len(items) == limit {
			break
		}
	}
	return items, nil
}

func (r *testMarketDataRepo) ListFactors(context.Context, string, time.Time, time.Time) ([]domain.FactorBar, error) {
	return r.factors, nil
}

func (r *testMarketDataRepo) AppendCandles(ctx context.Context, ticker string, timeframe string, candles []domain.Candle) error {
	if r.candles == nil {
		r.candles = make(map[string][]domain.Candle)
	}
	r.candles[ticker] = append(r.candles[ticker], candles...)
	return nil
}

func (r *testMarketDataRepo) AppendFactors(ctx context.Context, alias string, timeframe string, factors []domain.FactorBar) error {
	r.factors = append(r.factors, factors...)
	return nil
}

func (r *testSignalOutcomeRepo) Upsert(_ context.Context, outcome domain.SignalOutcome) (domain.SignalOutcome, error) {
	if r.items == nil {
		r.items = map[int64]domain.SignalOutcome{}
	}
	if existing, ok := r.items[outcome.SignalRunID]; ok {
		outcome.ID = existing.ID
		outcome.CreatedAt = existing.CreatedAt
		outcome.UpdatedAt = time.Now().UTC()
		r.items[outcome.SignalRunID] = outcome
		return outcome, nil
	}
	outcome.ID = int64(len(r.items) + 1)
	outcome.CreatedAt = time.Now().UTC()
	outcome.UpdatedAt = outcome.CreatedAt
	r.items[outcome.SignalRunID] = outcome
	return outcome, nil
}

func (r *testSignalOutcomeRepo) ListByPolicySnapshot(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	limit int,
) ([]domain.SignalOutcome, error) {
	items := make([]domain.SignalOutcome, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *testJobRunRepo) Create(_ context.Context, run domain.JobRun) (domain.JobRun, error) {
	run.ID = int64(len(r.items) + 1)
	run.StartedAt = time.Now().UTC()
	if run.Payload == nil {
		run.Payload = map[string]any{}
	}
	r.items = append(r.items, run)
	return run, nil
}

func (r *testJobRunRepo) Finish(_ context.Context, id int64, status string, payload map[string]any, errorMessage string) (domain.JobRun, error) {
	for index := range r.items {
		if r.items[index].ID == id {
			r.items[index].Status = status
			r.items[index].Payload = payload
			r.items[index].ErrorMessage = errorMessage
			r.items[index].FinishedAt = time.Now().UTC()
			return r.items[index], nil
		}
	}
	return domain.JobRun{}, sql.ErrNoRows
}

func (r *testJobRunRepo) ListLatest(_ context.Context, limit int) ([]domain.JobRun, error) {
	if limit > len(r.items) {
		limit = len(r.items)
	}
	out := make([]domain.JobRun, 0, limit)
	for i := len(r.items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.items[i])
	}
	return out, nil
}

func (r *testPolicyValidationRepo) Create(_ context.Context, run domain.PolicyValidationRun) (domain.PolicyValidationRun, error) {
	run.ID = int64(len(r.items) + 1)
	run.CreatedAt = time.Now().UTC()
	r.items = append(r.items, run)
	return run, nil
}

func (r *testPolicyValidationRepo) ListLatest(_ context.Context, limit int) ([]domain.PolicyValidationRun, error) {
	if limit > len(r.items) {
		limit = len(r.items)
	}
	out := make([]domain.PolicyValidationRun, 0, limit)
	for i := len(r.items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.items[i])
	}
	return out, nil
}

func (r *testPolicyValidationRepo) UpdateDecisionState(
	_ context.Context,
	id int64,
	decisionState string,
	notes string,
) (domain.PolicyValidationRun, error) {
	for index := range r.items {
		if r.items[index].ID == id {
			r.items[index].DecisionState = decisionState
			r.items[index].Notes = notes
			return r.items[index], nil
		}
	}
	return domain.PolicyValidationRun{}, sql.ErrNoRows
}

func (r *testPolicyValidationRepo) GetByID(ctx context.Context, id int64) (domain.PolicyValidationRun, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.PolicyValidationRun{}, sql.ErrNoRows
}

func (r *testPolicyValidationRepo) DemoteCurrentAndPromote(ctx context.Context, targetRunID int64, demoteReason string, promotionLog domain.PolicyPromotionLog) error {
	for i, item := range r.items {
		if item.DecisionState == domain.PolicyDecisionActive || item.DecisionState == domain.PolicyDecisionPromoted {
			item.DecisionState = domain.PolicyDecisionArchived
			r.items[i] = item
		}
	}
	for i, item := range r.items {
		if item.ID == targetRunID {
			item.DecisionState = domain.PolicyDecisionActive
			r.items[i] = item
		}
	}
	return nil
}

func (r *testPolicyValidationRepo) LogPromotion(ctx context.Context, log domain.PolicyPromotionLog) error {
	return nil
}

var _ repository.AssetRepository = (*testAssetRepo)(nil)
var _ repository.WatchlistRepository = (*testWatchlistRepo)(nil)
var _ repository.ModelRegistryRepository = (*testModelRepo)(nil)
var _ repository.SignalRunRepository = (*testSignalRepo)(nil)
var _ repository.SignalOutcomeRepository = (*testSignalOutcomeRepo)(nil)
var _ repository.JobRunRepository = (*testJobRunRepo)(nil)
var _ repository.PolicyValidationRunRepository = (*testPolicyValidationRepo)(nil)
var _ repository.MarketDataRepository = (*testMarketDataRepo)(nil)

func writeTestManifest(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "model_manifest.json")
	payload := `{
		"model_version": "baseline_stub_v1",
		"model_type": "stub_deterministic_v1",
		"task_type": "multiclass_signal_classification",
		"classes": ["up_signal", "down_signal", "no_trade"],
		"timeframe": "5m",
		"horizon_bars": 12,
		"feature_schema_version": "feature_v1",
		"feature_columns": ["ret_1", "ret_3", "close_to_prev_close", "ema_12_dist", "atr_14_pct"],
		"normalization_artifact_path": "",
		"export_format": "stub",
		"model_artifact_path": "model.stub",
		"metrics": {"macro_f1": 0.0},
		"decision_threshold": 0.65,
		"created_at": "2026-04-09T00:00:00Z"
	}`

	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func writeTestNeuralManifest(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "model_manifest.json")
	payload := `{
		"model_version": "sprint8_gru_1h_h24_w96",
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
		"metrics": {"f1_macro": 0.36},
		"threshold": 0.33,
		"calibration": "none",
		"created_at": "2026-05-07T01:31:13Z",
		"source_dataset_version": "sprint8_1h_h24_20260506",
		"input_window_bars": 96,
		"input_tensor_shape": [96, 48],
		"normalization": "standard"
	}`

	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write neural manifest: %v", err)
	}
	return path
}

func writeTestCandleParquet(t *testing.T, path string, rows []testCandleRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir candle parquet dir: %v", err)
	}
	if err := parquet.WriteFile(path, rows); err != nil {
		t.Fatalf("write candle parquet: %v", err)
	}
}

func writeTestFactorParquet(t *testing.T, path string, rows []testFactorRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir factor parquet dir: %v", err)
	}
	if err := parquet.WriteFile(path, rows); err != nil {
		t.Fatalf("write factor parquet: %v", err)
	}
}

func writeJSONFile(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir json dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json payload: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write json file: %v", err)
	}
}

func writeTextFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir text dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write text file: %v", err)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return ts
}

type testSignalEventRepo struct {
	items []domain.SignalEvent
}

func (r *testSignalEventRepo) Create(ctx context.Context, e domain.SignalEvent) (domain.SignalEvent, error) {
	r.items = append(r.items, e)
	return e, nil
}

func (r *testSignalEventRepo) Upsert(ctx context.Context, e domain.SignalEvent) (domain.SignalEvent, error) {
	r.items = append(r.items, e)
	return e, nil
}

func (r *testSignalEventRepo) ListLatest(ctx context.Context, limit int) ([]domain.SignalEvent, error) {
	return r.items, nil
}

func (r *testSignalEventRepo) ListByAsset(ctx context.Context, assetID string, limit int) ([]domain.SignalEvent, error) {
	return r.items, nil
}

type testNotificationRepo struct {
	rules []domain.NotificationRule
	events []domain.NotificationEvent
}

func (r *testNotificationRepo) ListRules(ctx context.Context) ([]domain.NotificationRule, error) {
	return r.rules, nil
}
func (r *testNotificationRepo) ListActiveRules(ctx context.Context) ([]domain.NotificationRule, error) {
	return r.rules, nil
}
func (r *testNotificationRepo) GetRuleByID(ctx context.Context, id int64) (domain.NotificationRule, error) {
	return domain.NotificationRule{}, nil
}
func (r *testNotificationRepo) CreateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	rule.ID = int64(len(r.rules) + 1)
	r.rules = append(r.rules, rule)
	return rule, nil
}
func (r *testNotificationRepo) UpdateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	return rule, nil
}
func (r *testNotificationRepo) DeleteRule(ctx context.Context, id int64) error {
	return nil
}
func (r *testNotificationRepo) CreateEvent(ctx context.Context, event domain.NotificationEvent) (domain.NotificationEvent, error) {
	event.ID = int64(len(r.events) + 1)
	r.events = append(r.events, event)
	return event, nil
}
func (r *testNotificationRepo) ListLatestEvents(ctx context.Context, limit int) ([]domain.NotificationEvent, error) {
	return r.events, nil
}
func (r *testNotificationRepo) GetLatestEventForRule(ctx context.Context, ruleID int64) (domain.NotificationEvent, error) {
	return domain.NotificationEvent{}, nil
}
