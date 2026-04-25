package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"invest/backend/internal/domain"
)

type JobRunRepository struct {
	db *sql.DB
}

func NewJobRunRepository(db *sql.DB) *JobRunRepository {
	return &JobRunRepository{db: db}
}

func (r *JobRunRepository) Create(ctx context.Context, run domain.JobRun) (domain.JobRun, error) {
	rawPayload, err := json.Marshal(normalizeJobPayload(run.Payload))
	if err != nil {
		return domain.JobRun{}, err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO job_runs (job_type, status, payload)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id, started_at
	`, run.JobType, run.Status, string(rawPayload)).Scan(&run.ID, &run.StartedAt)
	if err != nil {
		return domain.JobRun{}, err
	}
	run.Payload = normalizeJobPayload(run.Payload)
	return run, nil
}

func (r *JobRunRepository) Finish(
	ctx context.Context,
	id int64,
	status string,
	payload map[string]any,
	errorMessage string,
) (domain.JobRun, error) {
	rawPayload, err := json.Marshal(normalizeJobPayload(payload))
	if err != nil {
		return domain.JobRun{}, err
	}

	var item domain.JobRun
	var rawStoredPayload []byte
	err = r.db.QueryRowContext(ctx, `
		UPDATE job_runs
		SET status = $2,
		    finished_at = NOW(),
		    payload = $3::jsonb,
		    error_message = NULLIF($4, '')
		WHERE id = $1
		RETURNING id, job_type, status, started_at, finished_at, payload, COALESCE(error_message, '')
	`, id, status, string(rawPayload), errorMessage).Scan(
		&item.ID,
		&item.JobType,
		&item.Status,
		&item.StartedAt,
		&item.FinishedAt,
		&rawStoredPayload,
		&item.ErrorMessage,
	)
	if err != nil {
		return domain.JobRun{}, err
	}
	if err := json.Unmarshal(rawStoredPayload, &item.Payload); err != nil {
		return domain.JobRun{}, err
	}
	return item, nil
}

func (r *JobRunRepository) ListLatest(ctx context.Context, limit int) ([]domain.JobRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, job_type, status, started_at, finished_at, payload, COALESCE(error_message, '')
		FROM job_runs
		ORDER BY started_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.JobRun{}
	for rows.Next() {
		var item domain.JobRun
		var rawPayload []byte
		var finishedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.JobType,
			&item.Status,
			&item.StartedAt,
			&finishedAt,
			&rawPayload,
			&item.ErrorMessage,
		); err != nil {
			return nil, err
		}
		if finishedAt.Valid {
			item.FinishedAt = finishedAt.Time
		}
		if err := json.Unmarshal(rawPayload, &item.Payload); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func normalizeJobPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return payload
}
