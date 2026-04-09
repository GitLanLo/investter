package postgres

import (
	"context"
	"database/sql"
	"errors"

	"invest/backend/internal/domain"
)

type ModelRegistryRepository struct {
	db *sql.DB
}

func NewModelRegistryRepository(db *sql.DB) *ModelRegistryRepository {
	return &ModelRegistryRepository{db: db}
}

func (r *ModelRegistryRepository) List(ctx context.Context) ([]domain.ModelRegistryEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, model_version, model_type, status, timeframe, horizon_bars, feature_schema_version, manifest_path, created_at
		FROM model_registry
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.ModelRegistryEntry
	for rows.Next() {
		var item domain.ModelRegistryEntry
		if err := rows.Scan(
			&item.ID,
			&item.ModelVersion,
			&item.ModelType,
			&item.Status,
			&item.Timeframe,
			&item.HorizonBars,
			&item.FeatureSchemaVersion,
			&item.ManifestPath,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *ModelRegistryRepository) GetActive(ctx context.Context) (domain.ModelRegistryEntry, error) {
	var item domain.ModelRegistryEntry
	err := r.db.QueryRowContext(ctx, `
		SELECT id, model_version, model_type, status, timeframe, horizon_bars, feature_schema_version, manifest_path, created_at
		FROM model_registry
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, domain.ModelStatusActive).Scan(
		&item.ID,
		&item.ModelVersion,
		&item.ModelType,
		&item.Status,
		&item.Timeframe,
		&item.HorizonBars,
		&item.FeatureSchemaVersion,
		&item.ManifestPath,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ModelRegistryEntry{}, err
		}
		return domain.ModelRegistryEntry{}, err
	}
	return item, nil
}

func (r *ModelRegistryRepository) GetByVersion(ctx context.Context, version string) (domain.ModelRegistryEntry, error) {
	var item domain.ModelRegistryEntry
	err := r.db.QueryRowContext(ctx, `
		SELECT id, model_version, model_type, status, timeframe, horizon_bars, feature_schema_version, manifest_path, created_at
		FROM model_registry
		WHERE model_version = $1
	`, version).Scan(
		&item.ID,
		&item.ModelVersion,
		&item.ModelType,
		&item.Status,
		&item.Timeframe,
		&item.HorizonBars,
		&item.FeatureSchemaVersion,
		&item.ManifestPath,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ModelRegistryEntry{}, err
		}
		return domain.ModelRegistryEntry{}, err
	}
	return item, nil
}

func (r *ModelRegistryRepository) Register(ctx context.Context, entry domain.ModelRegistryEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO model_registry (model_version, model_type, status, timeframe, horizon_bars, feature_schema_version, manifest_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (model_version) DO UPDATE
		SET model_type = EXCLUDED.model_type,
		    status = EXCLUDED.status,
		    timeframe = EXCLUDED.timeframe,
		    horizon_bars = EXCLUDED.horizon_bars,
		    feature_schema_version = EXCLUDED.feature_schema_version,
		    manifest_path = EXCLUDED.manifest_path
	`,
		entry.ModelVersion,
		entry.ModelType,
		entry.Status,
		entry.Timeframe,
		entry.HorizonBars,
		entry.FeatureSchemaVersion,
		entry.ManifestPath,
	)

	return err
}
