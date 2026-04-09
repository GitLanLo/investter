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
}

func NewServices(
	assetRepo repository.AssetRepository,
	marketDataRepo repository.MarketDataRepository,
	watchlistRepo repository.WatchlistRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
) Services {
	return Services{
		Assets:     NewAssetService(assetRepo),
		MarketData: NewMarketDataService(assetRepo, marketDataRepo),
		Watchlist:  NewWatchlistService(watchlistRepo),
		Models:     NewModelRegistryService(modelRepo),
		Analysis:   NewAnalysisService(assetRepo, modelRepo, signalRepo),
	}
}
