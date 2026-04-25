package service

import (
	"context"

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
		{ID: "SBER", Ticker: "SBER", Name: "Sberbank", Exchange: "MOEX", Timeframe: "5m", IsActive: true},
		{ID: "GAZP", Ticker: "GAZP", Name: "Gazprom", Exchange: "MOEX", Timeframe: "5m", IsActive: true},
		{ID: "LKOH", Ticker: "LKOH", Name: "Lukoil", Exchange: "MOEX", Timeframe: "5m", IsActive: true},
		{ID: "MOEX", Ticker: "MOEX", Name: "Moscow Exchange", Exchange: "MOEX", Timeframe: "5m", IsActive: true},
		{ID: "NVTK", Ticker: "NVTK", Name: "Novatek", Exchange: "MOEX", Timeframe: "5m", IsActive: true},
	}

	for _, asset := range defaults {
		if err := s.repo.Upsert(ctx, asset); err != nil {
			return err
		}
	}

	return nil
}
