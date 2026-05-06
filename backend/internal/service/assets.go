package service

import (
	"context"
	"database/sql"
	"errors"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type AssetService struct {
	repo repository.AssetRepository
}

func NewAssetService(repo repository.AssetRepository) *AssetService {
	return &AssetService{repo: repo}
}

func (s *AssetService) List(ctx context.Context) ([]domain.Asset, error) {
	return s.repo.List(ctx)
}

func (s *AssetService) GetByID(ctx context.Context, id string) (domain.Asset, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AssetService) Upsert(ctx context.Context, asset domain.Asset) error {
	return s.repo.Upsert(ctx, asset)
}

func (s *AssetService) SeedDefaults(ctx context.Context) error {
	defaults := []domain.Asset{
		{ID: "SBER", Ticker: "SBER", Name: "Sberbank", Exchange: "MOEX", Timeframe: "5m", IsActive: true, ModelSupported: true},
		{ID: "GAZP", Ticker: "GAZP", Name: "Gazprom", Exchange: "MOEX", Timeframe: "5m", IsActive: true, ModelSupported: true},
		{ID: "LKOH", Ticker: "LKOH", Name: "Lukoil", Exchange: "MOEX", Timeframe: "5m", IsActive: true, ModelSupported: true},
		{ID: "MOEX", Ticker: "MOEX", Name: "Moscow Exchange", Exchange: "MOEX", Timeframe: "5m", IsActive: true, ModelSupported: true},
		{ID: "NVTK", Ticker: "NVTK", Name: "Novatek", Exchange: "MOEX", Timeframe: "5m", IsActive: true, ModelSupported: true},
	}

	for _, asset := range defaults {
		existing, err := s.repo.GetByID(ctx, asset.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			asset.Figi = existing.Figi
			asset.InstrumentUID = existing.InstrumentUID
			asset.ClassCode = existing.ClassCode
			asset.InstrumentType = existing.InstrumentType
			asset.Lot = existing.Lot
			asset.Currency = existing.Currency
			asset.APITradeAvailable = existing.APITradeAvailable
			asset.First1MinCandleDate = existing.First1MinCandleDate
			asset.First1DayCandleDate = existing.First1DayCandleDate
		}
		if err := s.repo.Upsert(ctx, asset); err != nil {
			return err
		}
	}

	return nil
}
