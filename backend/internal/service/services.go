package service

import (
	"context"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type InstrumentService interface {
	FindInstrument(ctx context.Context, query string) ([]domain.TinkoffInstrument, error)
	GetInstrumentByUID(ctx context.Context, uid string) (domain.TinkoffInstrument, error)
}

type Services struct {
	Assets      *AssetService
	MarketData  *MarketDataService
	Watchlist   *WatchlistService
	Models      *ModelRegistryService
	Analysis    *AnalysisService
	Research    *ResearchArtifactsService
	Policy      *PolicyValidationService
	Jobs        *JobService
	Instruments InstrumentService
}

func NewServices(
	assetRepo repository.AssetRepository,
	marketDataRepo repository.MarketDataRepository,
	watchlistRepo repository.WatchlistRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
	outcomeRepo repository.SignalOutcomeRepository,
	jobRepo repository.JobRunRepository,
	policyRepo repository.PolicyValidationRunRepository,
	mlDataRoot string,
	mlResearchRoot string,
	instruments InstrumentService,
) Services {
	research := NewResearchArtifactsService(mlDataRoot, mlResearchRoot)
	policy := NewPolicyValidationService(policyRepo, research, signalRepo)
	if assetRepo != nil && marketDataRepo != nil {
		policy = policy.WithOutcomeData(assetRepo, marketDataRepo, outcomeRepo)
	}
	return Services{
		Assets:      NewAssetService(assetRepo),
		MarketData:  NewMarketDataService(assetRepo, marketDataRepo),
		Watchlist:   NewWatchlistService(watchlistRepo),
		Models:      NewModelRegistryService(modelRepo),
		Analysis:    NewAnalysisService(assetRepo, modelRepo, signalRepo, research),
		Research:    research,
		Policy:      policy,
		Jobs:        NewJobService(jobRepo, policy),
		Instruments: instruments,
	}
}
