package app

import (
	"database/sql"

	"invest/backend/internal/repository/filesystem"
	"invest/backend/internal/repository/postgres"
	"invest/backend/internal/service"
)

type Container struct {
	Services service.Services
}

func NewContainer(db *sql.DB, mlDataRoot string) Container {
	assetRepo := postgres.NewAssetRepository(db)
	marketDataRepo := filesystem.NewMarketDataRepository(mlDataRoot)
	watchlistRepo := postgres.NewWatchlistRepository(db)
	modelRepo := postgres.NewModelRegistryRepository(db)
	signalRepo := postgres.NewSignalRunRepository(db)

	return Container{
		Services: service.NewServices(assetRepo, marketDataRepo, watchlistRepo, modelRepo, signalRepo),
	}
}
