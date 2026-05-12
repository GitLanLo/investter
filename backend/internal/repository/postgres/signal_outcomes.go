package postgres

import (
	"context"
	"database/sql"

	"invest/backend/internal/domain"
)

type SignalOutcomeRepository struct {
	db *sql.DB
}

func NewSignalOutcomeRepository(db *sql.DB) *SignalOutcomeRepository {
	return &SignalOutcomeRepository{db: db}
}

func (r *SignalOutcomeRepository) Upsert(ctx context.Context, outcome domain.SignalOutcome) (domain.SignalOutcome, error) {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO signal_outcomes (
			signal_run_id,
			asset_id,
			matured_at,
			entry_price,
			exit_price,
			raw_return_pct,
			action_return_pct,
			is_hit
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (signal_run_id) DO UPDATE SET
			asset_id = EXCLUDED.asset_id,
			matured_at = EXCLUDED.matured_at,
			entry_price = EXCLUDED.entry_price,
			exit_price = EXCLUDED.exit_price,
			raw_return_pct = EXCLUDED.raw_return_pct,
			action_return_pct = EXCLUDED.action_return_pct,
			is_hit = EXCLUDED.is_hit,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`,
		outcome.SignalRunID,
		outcome.AssetID,
		outcome.MaturedAt,
		outcome.EntryPrice,
		outcome.ExitPrice,
		outcome.RawReturnPct,
		outcome.ActionReturnPct,
		outcome.IsHit,
	).Scan(&outcome.ID, &outcome.CreatedAt, &outcome.UpdatedAt)
	if err != nil {
		return domain.SignalOutcome{}, err
	}
	return outcome, nil
}

func (r *SignalOutcomeRepository) ListByPolicySnapshot(
	ctx context.Context,
	userID int64,
	modelName string,
	calibrationMethod string,
	datasetVersion string,
	limit int,
) ([]domain.SignalOutcome, error) {
	query := `
		SELECT
			o.id,
			o.signal_run_id,
			o.asset_id,
			o.matured_at,
			o.entry_price,
			o.exit_price,
			o.raw_return_pct,
			o.action_return_pct,
			o.is_hit,
			o.created_at,
			o.updated_at
		FROM signal_outcomes o
		INNER JOIN signal_runs s ON s.id = o.signal_run_id
		WHERE s.policy_model_name = $1
		  AND s.policy_calibration_method = $2
		  AND s.policy_dataset_version = $3
	`
	var rows *sql.Rows
	var err error

	if userID == 0 {
		query += `
			ORDER BY s.as_of_time DESC, o.id DESC
			LIMIT $4
		`
		rows, err = r.db.QueryContext(ctx, query, modelName, calibrationMethod, datasetVersion, limit)
	} else {
		query += `
			AND s.user_id = $4
			ORDER BY s.as_of_time DESC, o.id DESC
			LIMIT $5
		`
		rows, err = r.db.QueryContext(ctx, query, modelName, calibrationMethod, datasetVersion, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.SignalOutcome{}
	for rows.Next() {
		var item domain.SignalOutcome
		if err := rows.Scan(
			&item.ID,
			&item.SignalRunID,
			&item.AssetID,
			&item.MaturedAt,
			&item.EntryPrice,
			&item.ExitPrice,
			&item.RawReturnPct,
			&item.ActionReturnPct,
			&item.IsHit,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
