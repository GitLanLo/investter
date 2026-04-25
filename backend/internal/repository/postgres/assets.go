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
		SELECT id, ticker, name, exchange, timeframe, is_active, figi, instrument_uid, class_code, instrument_type, lot, currency, api_trade_available, first_1min_candle_date, first_1day_candle_date, model_supported, created_at, updated_at
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
			&asset.Figi,
			&asset.InstrumentUID,
			&asset.ClassCode,
			&asset.InstrumentType,
			&asset.Lot,
			&asset.Currency,
			&asset.APITradeAvailable,
			&asset.First1MinCandleDate,
			&asset.First1DayCandleDate,
			&asset.ModelSupported,
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
		SELECT id, ticker, name, exchange, timeframe, is_active, figi, instrument_uid, class_code, instrument_type, lot, currency, api_trade_available, first_1min_candle_date, first_1day_candle_date, model_supported, created_at, updated_at
		FROM assets
		WHERE id = $1
	`, id).Scan(
		&asset.ID,
		&asset.Ticker,
		&asset.Name,
		&asset.Exchange,
		&asset.Timeframe,
		&asset.IsActive,
		&asset.Figi,
		&asset.InstrumentUID,
		&asset.ClassCode,
		&asset.InstrumentType,
		&asset.Lot,
		&asset.Currency,
		&asset.APITradeAvailable,
		&asset.First1MinCandleDate,
		&asset.First1DayCandleDate,
		&asset.ModelSupported,
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
		INSERT INTO assets (id, ticker, name, exchange, timeframe, is_active, figi, instrument_uid, class_code, instrument_type, lot, currency, api_trade_available, first_1min_candle_date, first_1day_candle_date, model_supported)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE
		SET ticker = EXCLUDED.ticker,
		    name = EXCLUDED.name,
		    exchange = EXCLUDED.exchange,
		    timeframe = EXCLUDED.timeframe,
		    is_active = EXCLUDED.is_active,
		    figi = EXCLUDED.figi,
		    instrument_uid = EXCLUDED.instrument_uid,
		    class_code = EXCLUDED.class_code,
		    instrument_type = EXCLUDED.instrument_type,
		    lot = EXCLUDED.lot,
		    currency = EXCLUDED.currency,
		    api_trade_available = EXCLUDED.api_trade_available,
		    first_1min_candle_date = EXCLUDED.first_1min_candle_date,
		    first_1day_candle_date = EXCLUDED.first_1day_candle_date,
		    model_supported = EXCLUDED.model_supported,
		    updated_at = NOW()
	`,
		asset.ID,
		asset.Ticker,
		asset.Name,
		asset.Exchange,
		asset.Timeframe,
		asset.IsActive,
		asset.Figi,
		asset.InstrumentUID,
		asset.ClassCode,
		asset.InstrumentType,
		asset.Lot,
		asset.Currency,
		asset.APITradeAvailable,
		asset.First1MinCandleDate,
		asset.First1DayCandleDate,
		asset.ModelSupported,
	)

	return err
}
