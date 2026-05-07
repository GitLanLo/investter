package service

import (
	"context"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type InstrumentService interface {
	FindInstrument(ctx context.Context, query string) ([]domain.TinkoffInstrument, error)
	GetInstrumentByUID(ctx context.Context, uid string) (domain.TinkoffInstrument, error)
	GetCandles(ctx context.Context, uid string, timeframe string, from time.Time, to time.Time) ([]domain.Candle, error)
	IsMarketOpen(ctx context.Context, exchange string) (bool, error)
}

type Services struct {
	Assets           *AssetService
	MarketData       *MarketDataService
	Watchlist        *WatchlistService
	Models           *ModelRegistryService
	Analysis         *AnalysisService
	Research         *ResearchArtifactsService
	Policy           *PolicyValidationService
	Promotion        *PolicyPromotionService
	Notifications    *NotificationService
	Monitoring       *MonitoringService
	Jobs             *JobService
	WatchlistRefresh *WatchlistRefreshService
	Instruments      InstrumentService
}

func NewServices(
	assetRepo repository.AssetRepository,
	marketData repository.MarketDataRepository,
	watchlistRepo repository.WatchlistRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
	eventRepo repository.SignalEventRepository,
	notificationRepo repository.NotificationRepository,
	outcomeRepo repository.SignalOutcomeRepository,
	jobRepo repository.JobRunRepository,
	policyRepo repository.PolicyValidationRunRepository,
	mlDataRoot string,
	mlResearchRoot string,
	instruments InstrumentService,
) Services {
	research := NewResearchArtifactsService(mlDataRoot, mlResearchRoot)
	models := NewModelRegistryService(modelRepo)
	notifications := NewNotificationService(notificationRepo)
	policy := NewPolicyValidationService(policyRepo, research, signalRepo).
		WithOutcomeData(assetRepo, marketData, outcomeRepo)

	return Services{
		Assets:        NewAssetService(assetRepo),
		MarketData:    NewMarketDataService(assetRepo, marketData),
		Watchlist:     NewWatchlistService(watchlistRepo),
		Models:        models,
		Analysis:      NewAnalysisService(assetRepo, modelRepo, signalRepo, eventRepo, notifications, research),
		Research:      research,
		Policy:        policy,
		Promotion:     NewPolicyPromotionService(policyRepo, models, policy, eventRepo, notifications),
		Notifications: notifications,
		Monitoring:    NewMonitoringService(assetRepo, modelRepo, signalRepo, policyRepo, notificationRepo, jobRepo, policy),
		Jobs:          NewJobService(jobRepo, policy),
		Instruments:   instruments,
	}
}
