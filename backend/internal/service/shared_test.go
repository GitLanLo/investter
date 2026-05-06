package service

import (
	"context"
	"database/sql"
	"time"

	"invest/backend/internal/domain"
)

// --- Shared Test Doubles ---

type MockAssetRepo struct {
	Assets map[string]domain.Asset
}

func (m *MockAssetRepo) List(ctx context.Context) ([]domain.Asset, error) {
	out := make([]domain.Asset, 0, len(m.Assets))
	for _, a := range m.Assets {
		out = append(out, a)
	}
	return out, nil
}

func (m *MockAssetRepo) GetByID(ctx context.Context, id string) (domain.Asset, error) {
	a, ok := m.Assets[id]
	if !ok {
		return domain.Asset{}, sql.ErrNoRows
	}
	return a, nil
}

func (m *MockAssetRepo) Upsert(ctx context.Context, asset domain.Asset) error {
	if m.Assets == nil {
		m.Assets = make(map[string]domain.Asset)
	}
	m.Assets[asset.ID] = asset
	return nil
}

type MockMarketDataRepo struct {
	Candles map[string][]domain.Candle
	Factors []domain.FactorBar
}

func (m *MockMarketDataRepo) ListCandles(ctx context.Context, ticker string, timeframe string, from time.Time, to time.Time, limit int) ([]domain.Candle, error) {
	cands := m.Candles[ticker]
	filtered := make([]domain.Candle, 0)
	for _, c := range cands {
		if !from.IsZero() && c.Timestamp.Before(from) {
			continue
		}
		if !to.IsZero() && c.Timestamp.After(to) {
			continue
		}
		filtered = append(filtered, c)
	}
	// Sort by time (newest first as expected by service)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (m *MockMarketDataRepo) ListFactors(ctx context.Context, timeframe string, from time.Time, to time.Time) ([]domain.FactorBar, error) {
	filtered := make([]domain.FactorBar, 0)
	for _, factor := range m.Factors {
		if factor.Timeframe != "" && factor.Timeframe != timeframe {
			continue
		}
		if !from.IsZero() && factor.Timestamp.Before(from) {
			continue
		}
		if !to.IsZero() && factor.Timestamp.After(to) {
			continue
		}
		filtered = append(filtered, factor)
	}
	return filtered, nil
}

func (m *MockMarketDataRepo) AppendCandles(ctx context.Context, ticker string, timeframe string, candles []domain.Candle) error {
	if m.Candles == nil {
		m.Candles = make(map[string][]domain.Candle)
	}
	m.Candles[ticker] = append(m.Candles[ticker], candles...)
	return nil
}

func (m *MockMarketDataRepo) AppendFactors(ctx context.Context, alias string, timeframe string, factors []domain.FactorBar) error {
	m.Factors = append(m.Factors, factors...)
	return nil
}

type MockWatchlistRepo struct {
	Items []domain.WatchlistItem
}

func (m *MockWatchlistRepo) GetOrCreateByName(ctx context.Context, name string) (domain.Watchlist, error) {
	return domain.Watchlist{ID: 1, Name: name}, nil
}

func (m *MockWatchlistRepo) ListItems(ctx context.Context, watchlistID int64) ([]domain.WatchlistItem, error) {
	return m.Items, nil
}

func (m *MockWatchlistRepo) AddItem(ctx context.Context, watchlistID int64, assetID string, position int) error {
	return nil
}

type MockJobRunRepo struct {
	Runs []domain.JobRun
	Seq  int64
}

func (m *MockJobRunRepo) Create(ctx context.Context, run domain.JobRun) (domain.JobRun, error) {
	m.Seq++
	run.ID = m.Seq
	run.StartedAt = time.Now().UTC()
	m.Runs = append(m.Runs, run)
	return run, nil
}

func (m *MockJobRunRepo) Finish(ctx context.Context, id int64, status string, payload map[string]any, errorMessage string) (domain.JobRun, error) {
	for i, r := range m.Runs {
		if r.ID == id {
			m.Runs[i].Status = status
			m.Runs[i].Payload = payload
			m.Runs[i].ErrorMessage = errorMessage
			m.Runs[i].FinishedAt = time.Now().UTC()
			return m.Runs[i], nil
		}
	}
	return domain.JobRun{}, ErrJobsUnavailable
}

func (m *MockJobRunRepo) ListLatest(ctx context.Context, limit int) ([]domain.JobRun, error) {
	if limit > len(m.Runs) {
		limit = len(m.Runs)
	}
	return m.Runs[:limit], nil
}

type MockSignalRepo struct {
	Signals map[string][]domain.SignalRun
}

func (m *MockSignalRepo) Create(ctx context.Context, run domain.SignalRun) (domain.SignalRun, error) {
	return run, nil
}

func (m *MockSignalRepo) ListLatest(ctx context.Context, limit int) ([]domain.SignalRun, error) {
	return nil, nil
}

func (m *MockSignalRepo) ListByAsset(ctx context.Context, assetID string, limit int) ([]domain.SignalRun, error) {
	return m.Signals[assetID], nil
}

func (m *MockSignalRepo) ListByPolicySnapshot(ctx context.Context, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalRun, error) {
	return nil, nil
}
