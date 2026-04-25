package repository

import (
	"context"
	"time"

	"invest/backend/internal/domain"
)

type AssetRepository interface {
	List(ctx context.Context) ([]domain.Asset, error)
	GetByID(ctx context.Context, id string) (domain.Asset, error)
	Upsert(ctx context.Context, asset domain.Asset) error
}

type MarketDataRepository interface {
	ListCandles(ctx context.Context, ticker string, timeframe string, from time.Time, to time.Time, limit int) ([]domain.Candle, error)
	ListFactors(ctx context.Context, timeframe string, from time.Time, to time.Time) ([]domain.FactorBar, error)
}

type WatchlistRepository interface {
	GetOrCreateByName(ctx context.Context, name string) (domain.Watchlist, error)
	ListItems(ctx context.Context, watchlistID int64) ([]domain.WatchlistItem, error)
	AddItem(ctx context.Context, watchlistID int64, assetID string, position int) error
}

type ModelRegistryRepository interface {
	List(ctx context.Context) ([]domain.ModelRegistryEntry, error)
	GetActive(ctx context.Context) (domain.ModelRegistryEntry, error)
	GetByVersion(ctx context.Context, version string) (domain.ModelRegistryEntry, error)
	Register(ctx context.Context, entry domain.ModelRegistryEntry) error
}

type SignalRunRepository interface {
	Create(ctx context.Context, run domain.SignalRun) (domain.SignalRun, error)
	ListLatest(ctx context.Context, limit int) ([]domain.SignalRun, error)
	ListByAsset(ctx context.Context, assetID string, limit int) ([]domain.SignalRun, error)
	ListByPolicySnapshot(ctx context.Context, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalRun, error)
}

type SignalOutcomeRepository interface {
	Upsert(ctx context.Context, outcome domain.SignalOutcome) (domain.SignalOutcome, error)
	ListByPolicySnapshot(ctx context.Context, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalOutcome, error)
}

type JobRunRepository interface {
	Create(ctx context.Context, run domain.JobRun) (domain.JobRun, error)
	Finish(ctx context.Context, id int64, status string, payload map[string]any, errorMessage string) (domain.JobRun, error)
	ListLatest(ctx context.Context, limit int) ([]domain.JobRun, error)
}

type PolicyValidationRunRepository interface {
	Create(ctx context.Context, run domain.PolicyValidationRun) (domain.PolicyValidationRun, error)
	ListLatest(ctx context.Context, limit int) ([]domain.PolicyValidationRun, error)
	UpdateDecisionState(ctx context.Context, id int64, decisionState string, notes string) (domain.PolicyValidationRun, error)
}
