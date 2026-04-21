package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"invest/backend/internal/domain"
)

type SignalRunRepository struct {
	db *sql.DB
}

func NewSignalRunRepository(db *sql.DB) *SignalRunRepository {
	return &SignalRunRepository{db: db}
}

func (r *SignalRunRepository) Create(ctx context.Context, run domain.SignalRun) (domain.SignalRun, error) {
	rawProbabilities, err := json.Marshal(run.ClassProbabilities)
	if err != nil {
		return domain.SignalRun{}, err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO signal_runs (
			asset_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
		RETURNING id, created_at
	`,
		run.AssetID,
		run.ModelVersion,
		run.AsOfTime,
		run.SignalState,
		run.SignalDirection,
		run.SignalProbability,
		string(rawProbabilities),
		run.Threshold,
		run.Timeframe,
		run.HorizonBars,
	).Scan(&run.ID, &run.CreatedAt)
	if err != nil {
		return domain.SignalRun{}, err
	}

	return run, nil
}

func (r *SignalRunRepository) ListLatest(ctx context.Context, limit int) ([]domain.SignalRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			asset_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars,
			created_at
		FROM signal_runs
		ORDER BY as_of_time DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalRun
	for rows.Next() {
		var (
			item             domain.SignalRun
			rawProbabilities []byte
		)
		if err := rows.Scan(
			&item.ID,
			&item.AssetID,
			&item.ModelVersion,
			&item.AsOfTime,
			&item.SignalState,
			&item.SignalDirection,
			&item.SignalProbability,
			&rawProbabilities,
			&item.Threshold,
			&item.Timeframe,
			&item.HorizonBars,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawProbabilities, &item.ClassProbabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SignalRunRepository) ListByAsset(ctx context.Context, assetID string, limit int) ([]domain.SignalRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			asset_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars,
			created_at
		FROM signal_runs
		WHERE asset_id = $1
		ORDER BY as_of_time DESC, id DESC
		LIMIT $2
	`, assetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalRun
	for rows.Next() {
		var (
			item             domain.SignalRun
			rawProbabilities []byte
		)
		if err := rows.Scan(
			&item.ID,
			&item.AssetID,
			&item.ModelVersion,
			&item.AsOfTime,
			&item.SignalState,
			&item.SignalDirection,
			&item.SignalProbability,
			&rawProbabilities,
			&item.Threshold,
			&item.Timeframe,
			&item.HorizonBars,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawProbabilities, &item.ClassProbabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
