package repository

import (
	"context"
	"time"

	"invest/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id int64) (domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	UpdateAccess(ctx context.Context, id int64, role string, permissions []string) (domain.User, error)
}

type AuthRepository interface {
	SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (userID int64, expiresAt time.Time, err error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteUserRefreshTokens(ctx context.Context, userID int64) error
}

type TinkoffCredentialRepository interface {
	Upsert(ctx context.Context, cred domain.UserTinkoffCredential) error
	GetByUserID(ctx context.Context, userID int64) (domain.UserTinkoffCredential, error)
	Delete(ctx context.Context, userID int64) error
	CreateBrokerConnection(ctx context.Context, connection domain.BrokerConnection) (domain.BrokerConnection, error)
	UpdateBrokerConnection(ctx context.Context, connection domain.BrokerConnection) (domain.BrokerConnection, error)
	ListBrokerConnections(ctx context.Context, userID int64) ([]domain.BrokerConnection, error)
	GetBrokerConnection(ctx context.Context, userID int64, connectionID int64) (domain.BrokerConnection, error)
	GetActiveBrokerConnection(ctx context.Context, userID int64) (domain.BrokerConnection, error)
	SetActiveBrokerConnection(ctx context.Context, userID int64, connectionID int64) error
	DeleteBrokerConnection(ctx context.Context, userID int64, connectionID int64) error
	SaveBrokerAccountSelection(ctx context.Context, selection domain.BrokerAccountSelection) error
	GetBrokerAccountSelection(ctx context.Context, userID int64) (domain.BrokerAccountSelection, error)
}

type AssetRepository interface {
	List(ctx context.Context) ([]domain.Asset, error)
	GetByID(ctx context.Context, id string) (domain.Asset, error)
	Upsert(ctx context.Context, asset domain.Asset) error
}

type MarketDataRepository interface {
	ListCandles(ctx context.Context, ticker string, timeframe string, from time.Time, to time.Time, limit int) ([]domain.Candle, error)
	ListFactors(ctx context.Context, timeframe string, from time.Time, to time.Time) ([]domain.FactorBar, error)
	AppendCandles(ctx context.Context, ticker string, timeframe string, candles []domain.Candle) error
	AppendFactors(ctx context.Context, alias string, timeframe string, factors []domain.FactorBar) error
}

type WatchlistRepository interface {
	GetOrCreateByName(ctx context.Context, userID int64, name string) (domain.Watchlist, error)
	ListItems(ctx context.Context, watchlistID int64) ([]domain.WatchlistItem, error)
	AddItem(ctx context.Context, watchlistID int64, assetID string, position int) error
	RemoveItem(ctx context.Context, watchlistID int64, assetID string) error
}

type ModelRegistryRepository interface {
	List(ctx context.Context) ([]domain.ModelRegistryEntry, error)
	GetActive(ctx context.Context) (domain.ModelRegistryEntry, error)
	GetByVersion(ctx context.Context, version string) (domain.ModelRegistryEntry, error)
	Register(ctx context.Context, entry domain.ModelRegistryEntry) error
	Activate(ctx context.Context, version string) error
}

type SignalRunRepository interface {
	Create(ctx context.Context, run domain.SignalRun) (domain.SignalRun, error)
	ListLatest(ctx context.Context, userID int64, limit int) ([]domain.SignalRun, error)
	ListByAsset(ctx context.Context, userID int64, assetID string, limit int) ([]domain.SignalRun, error)
	ListByPolicySnapshot(ctx context.Context, userID int64, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalRun, error)
}

type SignalEventRepository interface {
	Create(ctx context.Context, event domain.SignalEvent) (domain.SignalEvent, error)
	Upsert(ctx context.Context, event domain.SignalEvent) (domain.SignalEvent, error)
	ListLatest(ctx context.Context, userID int64, limit int) ([]domain.SignalEvent, error)
	ListByAsset(ctx context.Context, userID int64, assetID string, limit int) ([]domain.SignalEvent, error)
}

type SignalOutcomeRepository interface {
	Upsert(ctx context.Context, outcome domain.SignalOutcome) (domain.SignalOutcome, error)
	ListByPolicySnapshot(ctx context.Context, userID int64, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalOutcome, error)
}

type NotificationRepository interface {
	ListRules(ctx context.Context, userID int64) ([]domain.NotificationRule, error)
	ListActiveRules(ctx context.Context) ([]domain.NotificationRule, error)
	GetRuleByID(ctx context.Context, id int64, userID int64) (domain.NotificationRule, error)
	CreateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error)
	UpdateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error)
	DeleteRule(ctx context.Context, id int64, userID int64) error

	CreateEvent(ctx context.Context, event domain.NotificationEvent) (domain.NotificationEvent, error)
	ListLatestEvents(ctx context.Context, userID int64, limit int) ([]domain.NotificationEvent, error)
	GetLatestEventForRule(ctx context.Context, ruleID int64) (domain.NotificationEvent, error)
}

type JobRunRepository interface {
	Create(ctx context.Context, run domain.JobRun) (domain.JobRun, error)
	Finish(ctx context.Context, id int64, status string, payload map[string]any, errorMessage string) (domain.JobRun, error)
	ListLatest(ctx context.Context, limit int) ([]domain.JobRun, error)
}

type PolicyValidationRunRepository interface {
	Create(ctx context.Context, run domain.PolicyValidationRun) (domain.PolicyValidationRun, error)
	GetByID(ctx context.Context, id int64) (domain.PolicyValidationRun, error)
	ListLatest(ctx context.Context, limit int) ([]domain.PolicyValidationRun, error)
	UpdateDecisionState(ctx context.Context, id int64, decisionState string, notes string) (domain.PolicyValidationRun, error)
	DemoteCurrentAndPromote(ctx context.Context, targetRunID int64, demoteReason string, log domain.PolicyPromotionLog) error
	LogPromotion(ctx context.Context, log domain.PolicyPromotionLog) error
}
