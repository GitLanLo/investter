package app

import (
	"database/sql"
	"log"

	"invest/backend/internal/config"
	"invest/backend/internal/repository/filesystem"
	"invest/backend/internal/repository/postgres"
	"invest/backend/internal/service"
)

type Container struct {
	Services service.Services
}

func NewContainer(db *sql.DB, cfg config.Config) Container {
	assetRepo := postgres.NewAssetRepository(db)
	marketDataRepo := filesystem.NewMarketDataRepository(cfg.MLDataRoot)
	watchlistRepo := postgres.NewWatchlistRepository(db)
	modelRepo := postgres.NewModelRegistryRepository(db)
	signalRepo := postgres.NewSignalRunRepository(db)
	eventRepo := postgres.NewSignalEventRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	outcomeRepo := postgres.NewSignalOutcomeRepository(db)
	jobRepo := postgres.NewJobRunRepository(db)
	policyRepo := postgres.NewPolicyValidationRunRepository(db)
	tinkoffAdapter := service.NewTinkoffAdapter(cfg.TinkoffInvestToken, cfg.TinkoffInvestTarget, cfg.TinkoffCACertFile)

	svcs := service.NewServices(assetRepo, marketDataRepo, watchlistRepo, modelRepo, signalRepo, eventRepo, notificationRepo, outcomeRepo, jobRepo, policyRepo, cfg.MLDataRoot, cfg.MLResearchRoot, tinkoffAdapter)

	watchlistRefresh := service.NewWatchlistRefreshService(
		watchlistRepo, assetRepo, marketDataRepo, jobRepo, signalRepo,
		svcs.Analysis, tinkoffAdapter, log.Default(),
	)
	if factorSpecs, err := service.LoadUniverseFactorSpecs(cfg.MLUniverseConfigPath); err == nil {
		watchlistRefresh = watchlistRefresh.WithFactorSpecs(factorSpecs)
	} else {
		log.Printf("watchlist refresh factor specs unavailable: %v", err)
	}
	svcs.WatchlistRefresh = watchlistRefresh

	return Container{Services: svcs}
}
