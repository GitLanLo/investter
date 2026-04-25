package service

import (
	"invest/backend/internal/repository"
)

type Services struct {
	Assets     *AssetService
	MarketData *MarketDataService
	Watchlist  *WatchlistService
	Models     *ModelRegistryService
	Analysis   *AnalysisService
	Research   *ResearchArtifactsService
	Policy     *PolicyValidationService
	Jobs       *JobService
	Instruments *TinkoffAdapter
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
	instruments *TinkoffAdapter,
) Services {
	research := NewResearchArtifactsService(mlDataRoot, mlResearchRoot)
	policy := NewPolicyValidationService(policyRepo, research, signalRepo)
	if assetRepo != nil && marketDataRepo != nil {
		policy = policy.WithOutcomeData(assetRepo, marketDataRepo, outcomeRepo)
	}
	return Services{
		Assets:     NewAssetService(assetRepo),
		MarketData: NewMarketDataService(assetRepo, marketDataRepo),
		Watchlist:  NewWatchlistService(watchlistRepo),
		Models:     NewModelRegistryService(modelRepo),
		Analysis:   NewAnalysisService(assetRepo, modelRepo, signalRepo, research),
		Research:   research,
		Policy:     policy,
		Jobs:       NewJobService(jobRepo, policy),
		Instruments: instruments,
	}
}
