package service

import (
	"context"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type InstrumentService interface {
	FindInstrument(ctx context.Context, token string, query string) ([]domain.TinkoffInstrument, error)
	GetInstrumentByUID(ctx context.Context, token string, uid string) (domain.TinkoffInstrument, error)
	GetCandles(ctx context.Context, token string, uid string, timeframe string, from time.Time, to time.Time) ([]domain.Candle, error)
	IsMarketOpen(ctx context.Context, token string, exchange string) (bool, error)
}

type BrokerClient interface {
	GetAccounts(ctx context.Context, token string) ([]domain.BrokerAccount, error)
	GetPortfolio(ctx context.Context, token string, accountID string) (domain.BrokerPortfolio, error)
	GetPositions(ctx context.Context, token string, accountID string) (domain.BrokerPositions, error)
	GetOperationsByCursor(ctx context.Context, token string, request domain.BrokerOperationsRequest) (domain.BrokerOperationsPage, error)
	GetOrders(ctx context.Context, token string, accountID string) ([]domain.BrokerOrder, error)
	PostOrder(ctx context.Context, token string, request domain.BrokerPlaceOrderRequest) (domain.BrokerOrder, error)
	CancelOrder(ctx context.Context, token string, accountID string, orderID string) (domain.BrokerCancelOrderResult, error)
	GetOrderState(ctx context.Context, token string, accountID string, orderID string) (domain.BrokerOrder, error)
	OpenSandboxAccount(ctx context.Context, token string) (string, error)
	SandboxPayIn(ctx context.Context, token string, accountID string, amount domain.MoneyValue) (domain.BrokerSandboxPayInResult, error)
}

type SandboxAwareInstrumentService interface {
	WithSandboxTarget(isSandbox bool) InstrumentService
}

func InstrumentServiceForSandbox(instruments InstrumentService, isSandbox bool) InstrumentService {
	if instruments == nil {
		return nil
	}
	if targeted, ok := instruments.(SandboxAwareInstrumentService); ok {
		return targeted.WithSandboxTarget(isSandbox)
	}
	return instruments
}

type Services struct {
	Auth               *AuthService
	TinkoffCredentials *TinkoffCredentialService
	Assets             *AssetService
	MarketData         *MarketDataService
	Watchlist          *WatchlistService
	Models             *ModelRegistryService
	Analysis           *AnalysisService
	Research           *ResearchArtifactsService
	Policy             *PolicyValidationService
	Promotion          *PolicyPromotionService
	Notifications      *NotificationService
	Monitoring         *MonitoringService
	Jobs               *JobService
	WatchlistRefresh   *WatchlistRefreshService
	Instruments        InstrumentService
}

func NewServices(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	tinkoffCredRepo repository.TinkoffCredentialRepository,
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
	jwtSecret string,
	encryptionKey string,
	instruments InstrumentService,
) Services {
	research := NewResearchArtifactsService(mlDataRoot, mlResearchRoot)
	models := NewModelRegistryService(modelRepo)
	notifications := NewNotificationService(notificationRepo)
	policy := NewPolicyValidationService(policyRepo, research, signalRepo).
		WithOutcomeData(assetRepo, marketData, outcomeRepo)

	return Services{
		Auth:               NewAuthService(userRepo, authRepo, jwtSecret),
		TinkoffCredentials: NewTinkoffCredentialService(tinkoffCredRepo, encryptionKey, instruments),
		Assets:             NewAssetService(assetRepo),
		MarketData:         NewMarketDataService(assetRepo, marketData, signalRepo, instruments),
		Watchlist:          NewWatchlistService(watchlistRepo),
		Models:             models,
		Analysis:           NewAnalysisService(assetRepo, modelRepo, signalRepo, eventRepo, notifications, research),
		Research:           research,
		Policy:             policy,
		Promotion:          NewPolicyPromotionService(policyRepo, models, policy, eventRepo, notifications),
		Notifications:      notifications,
		Monitoring:         NewMonitoringService(assetRepo, modelRepo, signalRepo, policyRepo, notificationRepo, jobRepo, policy),
		Jobs:               NewJobService(jobRepo, policy),
		Instruments:        instruments,
	}
}
