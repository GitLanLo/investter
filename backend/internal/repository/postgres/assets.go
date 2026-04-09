package postgres

import (
	"context"
	"database/sql"
	"errors"

	"invest/backend/internal/domain"
)

type AssetRepository struct {
	db *sql.DB
}

func NewAssetRepository(db *sql.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

func (r *AssetRepository) List(ctx context.Context) ([]domain.Asset, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ticker, name, exchange, timeframe, is_active, created_at, updated_at
		FROM assets
		ORDER BY ticker
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Asset
	for rows.Next() {
		var asset domain.Asset
		if err := rows.Scan(
			&asset.ID,
			&asset.Ticker,
			&asset.Name,
			&asset.Exchange,
			&asset.Timeframe,
			&asset.IsActive,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, asset)
	}

	return items, rows.Err()
}

func (r *AssetRepository) GetByID(ctx context.Context, id string) (domain.Asset, error) {
	var asset domain.Asset
	err := r.db.QueryRowContext(ctx, `
		SELECT id, ticker, name, exchange, timeframe, is_active, created_at, updated_at
		FROM assets
		WHERE id = $1
	`, id).Scan(
		&asset.ID,
		&asset.Ticker,
		&asset.Name,
		&asset.Exchange,
		&asset.Timeframe,
		&asset.IsActive,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Asset{}, err
		}
		return domain.Asset{}, err
	}

	return asset, nil
}

func (r *AssetRepository) Upsert(ctx context.Context, asset domain.Asset) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO assets (id, ticker, name, exchange, timeframe, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET ticker = EXCLUDED.ticker,
		    name = EXCLUDED.name,
		    exchange = EXCLUDED.exchange,
		    timeframe = EXCLUDED.timeframe,
		    is_active = EXCLUDED.is_active,
		    updated_at = NOW()
	`,
		asset.ID,
		asset.Ticker,
		asset.Name,
		asset.Exchange,
		asset.Timeframe,
		asset.IsActive,
	)

	return err
}
