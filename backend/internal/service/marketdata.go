package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type MarketDataService struct {
	assetRepo repository.AssetRepository
	repo      repository.MarketDataRepository
}

func NewMarketDataService(
	assetRepo repository.AssetRepository,
	repo repository.MarketDataRepository,
) *MarketDataService {
	return &MarketDataService{
		assetRepo: assetRepo,
		repo:      repo,
	}
}

func (s *MarketDataService) ListCandles(
	ctx context.Context,
	assetID string,
	from time.Time,
	to time.Time,
	limit int,
) (domain.Asset, []domain.Candle, error) {
	asset, err := s.loadAsset(ctx, assetID)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	if limit <= 0 {
		limit = 500
	}
	if limit > 20000 {
		limit = 20000
	}

	items, err := s.repo.ListCandles(ctx, asset.Ticker, asset.Timeframe, from, to, limit)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	return asset, items, nil
}

func (s *MarketDataService) ListFactors(
	ctx context.Context,
	assetID string,
	from time.Time,
	to time.Time,
) (domain.Asset, []domain.FactorBar, error) {
	asset, err := s.loadAsset(ctx, assetID)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	items, err := s.repo.ListFactors(ctx, asset.Timeframe, from, to)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	return asset, items, nil
}

func (s *MarketDataService) loadAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	if assetID == "" {
		return domain.Asset{}, errors.New("assetID must not be empty")
	}

	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Asset{}, ErrAssetNotFound
		}
		return domain.Asset{}, err
	}

	return asset, nil
}
