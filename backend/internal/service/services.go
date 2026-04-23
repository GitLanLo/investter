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
}

func NewServices(
	assetRepo repository.AssetRepository,
	marketDataRepo repository.MarketDataRepository,
	watchlistRepo repository.WatchlistRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
	policyRepo repository.PolicyValidationRunRepository,
	mlDataRoot string,
	mlResearchRoot string,
) Services {
	research := NewResearchArtifactsService(mlDataRoot, mlResearchRoot)
	return Services{
		Assets:     NewAssetService(assetRepo),
		MarketData: NewMarketDataService(assetRepo, marketDataRepo),
		Watchlist:  NewWatchlistService(watchlistRepo),
		Models:     NewModelRegistryService(modelRepo),
		Analysis:   NewAnalysisService(assetRepo, modelRepo, signalRepo, research),
		Research:   research,
		Policy:     NewPolicyValidationService(policyRepo, research, signalRepo),
	}
}
