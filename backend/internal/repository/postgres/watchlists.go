package postgres

import (
	"context"
	"database/sql"
	"errors"

	"invest/backend/internal/domain"
)

type WatchlistRepository struct {
	db *sql.DB
}

func NewWatchlistRepository(db *sql.DB) *WatchlistRepository {
	return &WatchlistRepository{db: db}
}

func (r *WatchlistRepository) GetOrCreateByName(ctx context.Context, name string) (domain.Watchlist, error) {
	var watchlist domain.Watchlist
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, created_at, updated_at
		FROM watchlists
		WHERE name = $1
	`, name).Scan(&watchlist.ID, &watchlist.Name, &watchlist.CreatedAt, &watchlist.UpdatedAt)
	if err == nil {
		return watchlist, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Watchlist{}, err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO watchlists (name)
		VALUES ($1)
		RETURNING id, name, created_at, updated_at
	`, name).Scan(&watchlist.ID, &watchlist.Name, &watchlist.CreatedAt, &watchlist.UpdatedAt)
	if err != nil {
		return domain.Watchlist{}, err
	}

	return watchlist, nil
}

func (r *WatchlistRepository) ListItems(ctx context.Context, watchlistID int64) ([]domain.WatchlistItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, watchlist_id, asset_id, position, created_at
		FROM watchlist_items
		WHERE watchlist_id = $1
		ORDER BY position, id
	`, watchlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.WatchlistItem
	for rows.Next() {
		var item domain.WatchlistItem
		if err := rows.Scan(
			&item.ID,
			&item.WatchlistID,
			&item.AssetID,
			&item.Position,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *WatchlistRepository) AddItem(ctx context.Context, watchlistID int64, assetID string, position int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO watchlist_items (watchlist_id, asset_id, position)
		VALUES ($1, $2, $3)
		ON CONFLICT (watchlist_id, asset_id) DO UPDATE
		SET position = EXCLUDED.position
	`, watchlistID, assetID, position)

	return err
}
