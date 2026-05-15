package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

func newTestWatchlistRefreshService(
	watchlistRepo repository.WatchlistRepository,
	assetRepo repository.AssetRepository,
	marketData repository.MarketDataRepository,
	jobRepo repository.JobRunRepository,
	signalRepo repository.SignalRunRepository,
	analysis *AnalysisService,
	instruments InstrumentService,
	tinkoffCredentials *TinkoffCredentialService,
) *WatchlistRefreshService {
	return NewWatchlistRefreshService(
		watchlistRepo,
		assetRepo,
		marketData,
		jobRepo,
		signalRepo,
		analysis,
		instruments,
		tinkoffCredentials,
		"test-token",
		log.Default(),
	)
}

func TestGetFreshness_Empty(t *testing.T) {
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: nil},
		&MockAssetRepo{Assets: map[string]domain.Asset{}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{}},
		&MockJobRunRepo{},
		&MockSignalRepo{Signals: map[string][]domain.SignalRun{}},
		nil, nil, nil,
	)

	summary, err := svc.GetFreshness(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalItems != 0 {
		t.Errorf("expected 0 total items, got %d", summary.TotalItems)
	}
}

func TestGetFreshness_WithItems(t *testing.T) {
	now := time.Now().UTC()
	freshCandle := now.Add(-30 * time.Minute)
	staleCandle := now.Add(-3 * time.Hour)

	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{
			{AssetID: "A1"},
			{AssetID: "A2"},
			{AssetID: "A3"},
		}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"A1": {ID: "A1", Ticker: "T1", Name: "Asset1", ModelSupported: true},
			"A2": {ID: "A2", Ticker: "T2", Name: "Asset2", ModelSupported: false},
			"A3": {ID: "A3", Ticker: "T3", Name: "Asset3", ModelSupported: true},
		}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{
			"T1": {{Timestamp: freshCandle}},
			"T3": {{Timestamp: staleCandle}},
		}},
		&MockJobRunRepo{},
		&MockSignalRepo{Signals: map[string][]domain.SignalRun{
			"A1": {{CreatedAt: now.Add(-1 * time.Hour)}},
		}},
		nil, nil, nil,
	)

	summary, err := svc.GetFreshnessForUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalItems != 3 {
		t.Errorf("expected 3 total items, got %d", summary.TotalItems)
	}
	items := make(map[string]FreshnessItem, len(summary.Items))
	for _, item := range summary.Items {
		items[item.AssetID] = item
	}
	// A1: fresh data + fresh signal
	if !items["A1"].DataFresh {
		t.Error("A1 should have fresh data")
	}
	if !items["A1"].SignalFresh {
		t.Error("A1 should have fresh signal")
	}
	// A2: no candle data + watchlist-only
	if items["A2"].DataFresh {
		t.Error("A2 should have stale data")
	}
	if items["A2"].SignalFresh {
		t.Error("A2 should not have signal (watchlist-only)")
	}
	// A3: stale candle + no signal
	if items["A3"].DataFresh {
		t.Error("A3 should have stale data")
	}
}

func TestGetFreshnessForUserOnlyIncludesWatchlistAssets(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{{AssetID: "A1"}}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"A1": {ID: "A1", Ticker: "T1", Name: "Asset1", ModelSupported: true},
			"A2": {ID: "A2", Ticker: "T2", Name: "Asset2", ModelSupported: true},
		}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{
			"T1": {{Timestamp: now.Add(-30 * time.Minute), Close: 100}},
			"T2": {{Timestamp: now.Add(-30 * time.Minute), Close: 200}},
		}},
		&MockJobRunRepo{},
		&MockSignalRepo{},
		nil, nil, nil,
	)

	summary, err := svc.GetFreshnessForUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalItems != 1 || len(summary.Items) != 1 {
		t.Fatalf("expected only one watchlist asset, got total=%d len=%d", summary.TotalItems, len(summary.Items))
	}
	if summary.Items[0].AssetID != "A1" {
		t.Fatalf("unexpected asset in freshness: %s", summary.Items[0].AssetID)
	}
}

func TestRunWatchlistRefresh_CreatesJob(t *testing.T) {
	jobRepo := &MockJobRunRepo{}
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{
			{AssetID: "SBER"},
		}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Name: "Sberbank"},
		}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{}},
		jobRepo,
		&MockSignalRepo{Signals: map[string][]domain.SignalRun{}},
		nil, nil, nil,
	)

	run, err := svc.RunWatchlistRefresh(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.JobType != JobTypeWatchlistRefresh {
		t.Errorf("expected job type %s, got %s", JobTypeWatchlistRefresh, run.JobType)
	}
	if run.Status != domain.JobStatusSucceeded {
		t.Errorf("expected status %s, got %s", domain.JobStatusSucceeded, run.Status)
	}
}

func TestRunWatchlistSignalRefresh_CreatesJob(t *testing.T) {
	jobRepo := &MockJobRunRepo{}
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{
			{AssetID: "SBER"},
		}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Name: "Sberbank", ModelSupported: false},
		}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{}},
		jobRepo,
		&MockSignalRepo{Signals: map[string][]domain.SignalRun{}},
		nil, nil, nil,
	)

	run, err := svc.RunWatchlistSignalRefresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.JobType != JobTypeWatchlistSignalRefresh {
		t.Errorf("expected job type %s, got %s", JobTypeWatchlistSignalRefresh, run.JobType)
	}
	if run.Status != domain.JobStatusSucceeded {
		t.Errorf("expected status succeeded, got %s", run.Status)
	}
}

func TestRunWatchlistRefresh_NoJobRepo(t *testing.T) {
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: nil},
		&MockAssetRepo{Assets: map[string]domain.Asset{}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{}},
		nil, // no job repo
		&MockSignalRepo{Signals: map[string][]domain.SignalRun{}},
		nil, nil, nil,
	)

	_, err := svc.RunWatchlistRefresh(context.Background(), 50)
	if err != ErrJobsUnavailable {
		t.Errorf("expected ErrJobsUnavailable, got %v", err)
	}
}

type mockInstrumentService struct {
	open        bool
	instruments []domain.TinkoffInstrument
	lastToken   string
}

func (m *mockInstrumentService) FindInstrument(ctx context.Context, token string, query string) ([]domain.TinkoffInstrument, error) {
	m.lastToken = token
	return m.instruments, nil
}
func (m *mockInstrumentService) GetInstrumentByUID(ctx context.Context, token string, uid string) (domain.TinkoffInstrument, error) {
	m.lastToken = token
	if uid == "FAIL" {
		return domain.TinkoffInstrument{}, fmt.Errorf("api error")
	}
	return domain.TinkoffInstrument{UID: uid, Ticker: "T1", Name: "Asset1"}, nil
}
func (m *mockInstrumentService) GetCandles(ctx context.Context, token string, uid string, timeframe string, from time.Time, to time.Time) ([]domain.Candle, error) {
	m.lastToken = token
	if uid == "FAIL" {
		return nil, fmt.Errorf("api error")
	}
	return []domain.Candle{{Timestamp: time.Now().UTC()}}, nil
}
func (m *mockInstrumentService) IsMarketOpen(ctx context.Context, token string, exchange string) (bool, error) {
	m.lastToken = token
	return m.open, nil
}

type mockTinkoffCredentialRepo struct {
	byUserID     map[int64]domain.UserTinkoffCredential
	connections  map[int64]domain.BrokerConnection
	selections   map[int64]domain.BrokerAccountSelection
	nextBrokerID int64
}

func (m *mockTinkoffCredentialRepo) Upsert(ctx context.Context, cred domain.UserTinkoffCredential) error {
	if m.byUserID == nil {
		m.byUserID = make(map[int64]domain.UserTinkoffCredential)
	}
	m.byUserID[cred.UserID] = cred
	return nil
}

func (m *mockTinkoffCredentialRepo) GetByUserID(ctx context.Context, userID int64) (domain.UserTinkoffCredential, error) {
	cred, ok := m.byUserID[userID]
	if !ok {
		return domain.UserTinkoffCredential{}, fmt.Errorf("not found")
	}
	return cred, nil
}

func (m *mockTinkoffCredentialRepo) Delete(ctx context.Context, userID int64) error {
	delete(m.byUserID, userID)
	return nil
}

func (m *mockTinkoffCredentialRepo) CreateBrokerConnection(ctx context.Context, connection domain.BrokerConnection) (domain.BrokerConnection, error) {
	if m.connections == nil {
		m.connections = make(map[int64]domain.BrokerConnection)
	}
	m.nextBrokerID++
	connection.ID = m.nextBrokerID
	now := time.Now().UTC()
	connection.CreatedAt = now
	connection.UpdatedAt = now
	if connection.IsActive {
		for id, existing := range m.connections {
			if existing.UserID == connection.UserID {
				existing.IsActive = false
				m.connections[id] = existing
			}
		}
	}
	m.connections[connection.ID] = connection
	return connection, nil
}

func (m *mockTinkoffCredentialRepo) UpdateBrokerConnection(ctx context.Context, connection domain.BrokerConnection) (domain.BrokerConnection, error) {
	if m.connections == nil {
		return domain.BrokerConnection{}, sql.ErrNoRows
	}
	if _, ok := m.connections[connection.ID]; !ok {
		return domain.BrokerConnection{}, sql.ErrNoRows
	}
	connection.UpdatedAt = time.Now().UTC()
	if connection.IsActive {
		for id, existing := range m.connections {
			if existing.UserID == connection.UserID && id != connection.ID {
				existing.IsActive = false
				m.connections[id] = existing
			}
		}
	}
	m.connections[connection.ID] = connection
	return connection, nil
}

func (m *mockTinkoffCredentialRepo) ListBrokerConnections(ctx context.Context, userID int64) ([]domain.BrokerConnection, error) {
	items := make([]domain.BrokerConnection, 0)
	for _, connection := range m.connections {
		if connection.UserID == userID {
			items = append(items, connection)
		}
	}
	return items, nil
}

func (m *mockTinkoffCredentialRepo) GetBrokerConnection(ctx context.Context, userID int64, connectionID int64) (domain.BrokerConnection, error) {
	connection, ok := m.connections[connectionID]
	if !ok || connection.UserID != userID {
		return domain.BrokerConnection{}, sql.ErrNoRows
	}
	return connection, nil
}

func (m *mockTinkoffCredentialRepo) GetActiveBrokerConnection(ctx context.Context, userID int64) (domain.BrokerConnection, error) {
	for _, connection := range m.connections {
		if connection.UserID == userID && connection.IsActive {
			return connection, nil
		}
	}
	return domain.BrokerConnection{}, sql.ErrNoRows
}

func (m *mockTinkoffCredentialRepo) SetActiveBrokerConnection(ctx context.Context, userID int64, connectionID int64) error {
	if _, err := m.GetBrokerConnection(ctx, userID, connectionID); err != nil {
		return err
	}
	for id, connection := range m.connections {
		if connection.UserID == userID {
			connection.IsActive = id == connectionID
			m.connections[id] = connection
		}
	}
	return nil
}

func (m *mockTinkoffCredentialRepo) DeleteBrokerConnection(ctx context.Context, userID int64, connectionID int64) error {
	connection, ok := m.connections[connectionID]
	if ok && connection.UserID == userID {
		delete(m.connections, connectionID)
		if connection.IsActive {
			for id, existing := range m.connections {
				if existing.UserID == userID {
					existing.IsActive = true
					m.connections[id] = existing
					break
				}
			}
		}
	}
	return nil
}

func (m *mockTinkoffCredentialRepo) SaveBrokerAccountSelection(ctx context.Context, selection domain.BrokerAccountSelection) error {
	if m.selections == nil {
		m.selections = make(map[int64]domain.BrokerAccountSelection)
	}
	selection.UpdatedAt = time.Now().UTC()
	m.selections[selection.UserID] = selection
	return nil
}

func (m *mockTinkoffCredentialRepo) GetBrokerAccountSelection(ctx context.Context, userID int64) (domain.BrokerAccountSelection, error) {
	selection, ok := m.selections[userID]
	if !ok {
		return domain.BrokerAccountSelection{}, sql.ErrNoRows
	}
	return selection, nil
}

func TestGetFreshness_MarketClosed(t *testing.T) {
	now := time.Now().UTC()
	staleCandle := now.Add(-3 * time.Hour)

	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{{AssetID: "A1"}}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"A1": {ID: "A1", Ticker: "T1", Exchange: "MOEX", ModelSupported: true},
		}},
		&MockMarketDataRepo{Candles: map[string][]domain.Candle{
			"T1": {{Timestamp: staleCandle}},
		}},
		&MockJobRunRepo{},
		&MockSignalRepo{},
		nil,
		&mockInstrumentService{open: false}, // Market closed
		nil,
	)

	summary, err := svc.GetFreshness(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.StaleData != 0 {
		t.Errorf("expected 0 stale items when market is closed, got %d", summary.StaleData)
	}
	if !strings.HasPrefix(summary.Items[0].StaleReason, "market closed") {
		t.Errorf("expected reason to start with 'market closed', got %s", summary.Items[0].StaleReason)
	}
}

func TestRunWatchlistRefresh_PartialFailure(t *testing.T) {
	jobRepo := &MockJobRunRepo{}
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{
			{AssetID: "OK"},
			{AssetID: "FAIL"},
		}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"OK":   {ID: "OK", Ticker: "OK", InstrumentUID: "OK", IsActive: true},
			"FAIL": {ID: "FAIL", Ticker: "FAIL", InstrumentUID: "FAIL", IsActive: true},
		}},
		&MockMarketDataRepo{},
		jobRepo,
		&MockSignalRepo{},
		nil,
		&mockInstrumentService{open: true},
		nil,
	)

	run, _ := svc.RunWatchlistRefresh(context.Background(), 50)
	if run.Status != domain.JobStatusFailed {
		t.Errorf("expected job status failed due to partial failure, got %s", run.Status)
	}
	if !strings.Contains(run.ErrorMessage, "items failed") {
		t.Errorf("unexpected error message: %s", run.ErrorMessage)
	}
}

func TestRunWatchlistRefresh_ChecksFactorFreshness(t *testing.T) {
	now := time.Now().UTC()
	jobRepo := &MockJobRunRepo{}
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{{AssetID: "SBER"}}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		}},
		&MockMarketDataRepo{
			Factors: []domain.FactorBar{
				{Factor: "usdrub", Timeframe: "5m", Timestamp: now.Add(-30 * time.Minute)},
				{Factor: "brent", Timeframe: "5m", Timestamp: now.Add(-3 * time.Hour)},
			},
		},
		jobRepo,
		&MockSignalRepo{},
		nil,
		nil,
		nil,
	)

	run, err := svc.RunWatchlistRefresh(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := run.Payload["factors_checked"]; got != 2 {
		t.Errorf("expected 2 factors checked, got %v", got)
	}
	if got := run.Payload["fresh_factors"]; got != 1 {
		t.Errorf("expected 1 fresh factor, got %v", got)
	}
	if got := run.Payload["stale_factors"]; got != 1 {
		t.Errorf("expected 1 stale factor, got %v", got)
	}
}

func TestRunWatchlistRefresh_RefreshesConfiguredFactors(t *testing.T) {
	now := time.Now().UTC()
	jobRepo := &MockJobRunRepo{}
	marketRepo := &MockMarketDataRepo{}
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{{AssetID: "SBER"}}},
		&MockAssetRepo{Assets: map[string]domain.Asset{
			"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", IsActive: true},
		}},
		marketRepo,
		jobRepo,
		&MockSignalRepo{},
		nil,
		&mockInstrumentService{
			open: true,
			instruments: []domain.TinkoffInstrument{
				{UID: "usd-uid", Ticker: "USD000UTSTOM", ClassCode: "CETS"},
			},
		},
		nil,
	).WithFactorSpecs([]FactorRefreshSpec{
		{Alias: "usdrub", Ticker: "USD000UTSTOM", Timeframe: "5m", ClassCode: "CETS"},
	})

	run, err := svc.RunWatchlistRefresh(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := run.Payload["factors_refreshed"]; got != 1 {
		t.Errorf("expected 1 refreshed factor, got %v", got)
	}
	if len(marketRepo.Factors) != 1 {
		t.Fatalf("expected one appended factor bar, got %d", len(marketRepo.Factors))
	}
	if marketRepo.Factors[0].Factor != "usdrub" {
		t.Fatalf("unexpected factor alias: %s", marketRepo.Factors[0].Factor)
	}
	if marketRepo.Factors[0].Timestamp.Before(now.Add(-time.Minute)) {
		t.Fatalf("unexpected factor timestamp: %s", marketRepo.Factors[0].Timestamp)
	}
}

func TestRunWatchlistRefresh_ResolvesMissingInstrumentUID(t *testing.T) {
	jobRepo := &MockJobRunRepo{}
	marketRepo := &MockMarketDataRepo{}
	assetRepo := &MockAssetRepo{Assets: map[string]domain.Asset{
		"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", ModelSupported: true, IsActive: true},
	}}
	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{{AssetID: "SBER"}}},
		assetRepo,
		marketRepo,
		jobRepo,
		&MockSignalRepo{},
		nil,
		&mockInstrumentService{
			open: true,
			instruments: []domain.TinkoffInstrument{
				{UID: "sber-uid", Ticker: "SBER", ClassCode: "TQBR", Name: "Sberbank", Exchange: "MOEX"},
			},
		},
		nil,
	)

	run, err := svc.RunWatchlistRefresh(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != domain.JobStatusSucceeded {
		t.Fatalf("expected succeeded refresh, got %s: %s", run.Status, run.ErrorMessage)
	}
	if assetRepo.Assets["SBER"].InstrumentUID != "sber-uid" {
		t.Fatalf("expected resolved instrument uid, got %q", assetRepo.Assets["SBER"].InstrumentUID)
	}
	if len(marketRepo.Candles["SBER"]) == 0 {
		t.Fatal("expected candles appended for resolved ticker")
	}
}

func TestRunWatchlistRefreshForUserUsesStoredTinkoffToken(t *testing.T) {
	jobRepo := &MockJobRunRepo{}
	marketRepo := &MockMarketDataRepo{}
	assetRepo := &MockAssetRepo{Assets: map[string]domain.Asset{
		"SBER": {ID: "SBER", Ticker: "SBER", Timeframe: "5m", InstrumentUID: "sber-uid", IsActive: true},
	}}
	instruments := &mockInstrumentService{open: true}
	credRepo := &mockTinkoffCredentialRepo{}
	creds := NewTinkoffCredentialService(credRepo, "0123456789abcdef0123456789abcdef", instruments)
	if err := creds.SaveToken(context.Background(), 7, "user-token", false); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	svc := newTestWatchlistRefreshService(
		&MockWatchlistRepo{Items: []domain.WatchlistItem{{AssetID: "SBER"}}},
		assetRepo,
		marketRepo,
		jobRepo,
		&MockSignalRepo{},
		nil,
		instruments,
		creds,
	)

	run, err := svc.RunWatchlistRefreshForUser(context.Background(), 7, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != domain.JobStatusSucceeded {
		t.Fatalf("expected succeeded refresh, got %s: %s", run.Status, run.ErrorMessage)
	}
	if instruments.lastToken != "user-token" {
		t.Fatalf("expected stored user token, got %q", instruments.lastToken)
	}
}

func TestWatchlistRefreshScheduler_NilSafe(t *testing.T) {
	// Should not panic
	var scheduler *WatchlistRefreshScheduler
	scheduler.Start(context.Background())

	scheduler = NewWatchlistRefreshScheduler(nil, 0, 0, nil)
	scheduler.Start(context.Background())
}

func TestWatchlistSignalScheduler_NilSafe(t *testing.T) {
	var scheduler *WatchlistSignalScheduler
	scheduler.Start(context.Background())

	scheduler = NewWatchlistSignalScheduler(nil, 0, nil)
	scheduler.Start(context.Background())
}
