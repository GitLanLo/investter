package app

import (
	"database/sql"

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
	outcomeRepo := postgres.NewSignalOutcomeRepository(db)
	jobRepo := postgres.NewJobRunRepository(db)
	policyRepo := postgres.NewPolicyValidationRunRepository(db)
	tinkoffAdapter := service.NewTinkoffAdapter(cfg.TinkoffInvestToken, cfg.TinkoffInvestTarget)

	return Container{
		Services: service.NewServices(assetRepo, marketDataRepo, watchlistRepo, modelRepo, signalRepo, outcomeRepo, jobRepo, policyRepo, cfg.MLDataRoot, cfg.MLResearchRoot, tinkoffAdapter),
	}
}
