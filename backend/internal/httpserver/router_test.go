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

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
		DB: &sql.DB{},
		Container: app.Container{
			Services: service.Services{
				Assets:    service.NewAssetService(assetRepo),
				Watchlist: service.NewWatchlistService(&testWatchlistRepo{}),
				Models:    service.NewModelRegistryService(modelRepo),
				Analysis:  service.NewAnalysisService(assetRepo, modelRepo, signalRepo),
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

	router := NewRouter(config.Config{AppEnv: "test"}, Dependencies{
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

func (r *testAssetRepo) Upsert(context.Context, domain.Asset) error {
	return nil
}

type testWatchlistRepo struct{}

func (r *testWatchlistRepo) GetOrCreateByName(context.Context, string) (domain.Watchlist, error) {
	return domain.Watchlist{ID: 1, Name: "default"}, nil
}

func (r *testWatchlistRepo) ListItems(context.Context, int64) ([]domain.WatchlistItem, error) {
	return nil, nil
}

func (r *testWatchlistRepo) AddItem(context.Context, int64, string, int) error {
	return nil
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

var _ repository.AssetRepository = (*testAssetRepo)(nil)
var _ repository.WatchlistRepository = (*testWatchlistRepo)(nil)
var _ repository.ModelRegistryRepository = (*testModelRepo)(nil)
var _ repository.SignalRunRepository = (*testSignalRepo)(nil)

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

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return ts
}
