package service

import (
	"context"
	"errors"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type WatchlistService struct {
	repo repository.WatchlistRepository
}

func NewWatchlistService(repo repository.WatchlistRepository) *WatchlistService {
	return &WatchlistService{repo: repo}
}

func (s *WatchlistService) EnsureDefault(ctx context.Context) (domain.Watchlist, error) {
	return s.repo.GetOrCreateByName(ctx, "default")
}

func (s *WatchlistService) ListItems(ctx context.Context, watchlistID int64) ([]domain.WatchlistItem, error) {
	return s.repo.ListItems(ctx, watchlistID)
}

func (s *WatchlistService) AddItem(ctx context.Context, watchlistID int64, assetID string, position int) error {
	if watchlistID <= 0 {
		return errors.New("watchlistID must be positive")
	}
	if assetID == "" {
		return errors.New("assetID must not be empty")
	}
	if position < 0 {
		return errors.New("position must be non-negative")
	}

	return s.repo.AddItem(ctx, watchlistID, assetID, position)
}
