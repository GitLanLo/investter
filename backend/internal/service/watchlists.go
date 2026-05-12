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

func (s *WatchlistService) GetUserWatchlist(ctx context.Context, userID int64) (domain.Watchlist, error) {
	return s.repo.GetOrCreateByName(ctx, userID, "default")
}

func (s *WatchlistService) ListItems(ctx context.Context, userID int64) ([]domain.WatchlistItem, error) {
	wl, err := s.GetUserWatchlist(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListItems(ctx, wl.ID)
}

func (s *WatchlistService) AddItem(ctx context.Context, userID int64, assetID string, position int) error {
	wl, err := s.GetUserWatchlist(ctx, userID)
	if err != nil {
		return err
	}

	if assetID == "" {
		return errors.New("assetID must not be empty")
	}
	if position < 0 {
		position = 0
	}

	return s.repo.AddItem(ctx, wl.ID, assetID, position)
}

func (s *WatchlistService) RemoveItem(ctx context.Context, userID int64, assetID string) error {
	wl, err := s.GetUserWatchlist(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.RemoveItem(ctx, wl.ID, assetID)
}
