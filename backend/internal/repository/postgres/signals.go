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
			user_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars,
			policy_status,
			policy_model_name,
			policy_scenario_name,
			policy_calibration_method,
			policy_threshold,
			policy_dataset_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at
	`,
		run.AssetID,
		sql.NullInt64{Int64: run.UserID, Valid: run.UserID > 0},
		run.ModelVersion,
		run.AsOfTime,
		run.SignalState,
		run.SignalDirection,
		run.SignalProbability,
		string(rawProbabilities),
		run.Threshold,
		run.Timeframe,
		run.HorizonBars,
		nullablePolicyString(run.Policy, func(policy domain.SignalPolicySnapshot) string { return policy.PolicyStatus }),
		nullablePolicyString(run.Policy, func(policy domain.SignalPolicySnapshot) string { return policy.ModelName }),
		nullablePolicyString(run.Policy, func(policy domain.SignalPolicySnapshot) string { return policy.ScenarioName }),
		nullablePolicyString(run.Policy, func(policy domain.SignalPolicySnapshot) string { return policy.CalibrationMethod }),
		nullablePolicyThreshold(run.Policy),
		nullablePolicyString(run.Policy, func(policy domain.SignalPolicySnapshot) string { return policy.DatasetVersion }),
	).Scan(&run.ID, &run.CreatedAt)
	if err != nil {
		return domain.SignalRun{}, err
	}

	return run, nil
}

func (r *SignalRunRepository) ListLatest(ctx context.Context, userID int64, limit int) ([]domain.SignalRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			asset_id,
			user_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars,
			policy_status,
			policy_model_name,
			policy_scenario_name,
			policy_calibration_method,
			policy_threshold,
			policy_dataset_version,
			created_at
		FROM signal_runs
		WHERE user_id = $1
		ORDER BY as_of_time DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalRun
	for rows.Next() {
		var (
			item             domain.SignalRun
			rawProbabilities []byte
			userIDValue      sql.NullInt64
			policyStatus     sql.NullString
			policyModelName  sql.NullString
			policyScenario   sql.NullString
			policyMethod     sql.NullString
			policyThreshold  sql.NullFloat64
			policyDataset    sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.AssetID,
			&userIDValue,
			&item.ModelVersion,
			&item.AsOfTime,
			&item.SignalState,
			&item.SignalDirection,
			&item.SignalProbability,
			&rawProbabilities,
			&item.Threshold,
			&item.Timeframe,
			&item.HorizonBars,
			&policyStatus,
			&policyModelName,
			&policyScenario,
			&policyMethod,
			&policyThreshold,
			&policyDataset,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.UserID = userIDValue.Int64
		if err := json.Unmarshal(rawProbabilities, &item.ClassProbabilities); err != nil {
			return nil, err
		}
		item.Policy = signalPolicyFromNullableFields(
			policyStatus,
			policyModelName,
			policyScenario,
			policyMethod,
			policyThreshold,
			policyDataset,
		)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SignalRunRepository) ListByAsset(ctx context.Context, userID int64, assetID string, limit int) ([]domain.SignalRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			asset_id,
			user_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars,
			policy_status,
			policy_model_name,
			policy_scenario_name,
			policy_calibration_method,
			policy_threshold,
			policy_dataset_version,
			created_at
		FROM signal_runs
		WHERE user_id = $1 AND asset_id = $2
		ORDER BY as_of_time DESC, id DESC
		LIMIT $3
	`, userID, assetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalRun
	for rows.Next() {
		var (
			item             domain.SignalRun
			rawProbabilities []byte
			userIDValue      sql.NullInt64
			policyStatus     sql.NullString
			policyModelName  sql.NullString
			policyScenario   sql.NullString
			policyMethod     sql.NullString
			policyThreshold  sql.NullFloat64
			policyDataset    sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.AssetID,
			&userIDValue,
			&item.ModelVersion,
			&item.AsOfTime,
			&item.SignalState,
			&item.SignalDirection,
			&item.SignalProbability,
			&rawProbabilities,
			&item.Threshold,
			&item.Timeframe,
			&item.HorizonBars,
			&policyStatus,
			&policyModelName,
			&policyScenario,
			&policyMethod,
			&policyThreshold,
			&policyDataset,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.UserID = userIDValue.Int64
		if err := json.Unmarshal(rawProbabilities, &item.ClassProbabilities); err != nil {
			return nil, err
		}
		item.Policy = signalPolicyFromNullableFields(
			policyStatus,
			policyModelName,
			policyScenario,
			policyMethod,
			policyThreshold,
			policyDataset,
		)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SignalRunRepository) ListByPolicySnapshot(
	ctx context.Context,
	userID int64,
	modelName string,
	calibrationMethod string,
	datasetVersion string,
	limit int,
) ([]domain.SignalRun, error) {
	query := `
		SELECT
			id,
			asset_id,
			user_id,
			model_version,
			as_of_time,
			signal_state,
			signal_direction,
			signal_probability,
			class_probabilities,
			threshold,
			timeframe,
			horizon_bars,
			policy_status,
			policy_model_name,
			policy_scenario_name,
			policy_calibration_method,
			policy_threshold,
			policy_dataset_version,
			created_at
		FROM signal_runs
		WHERE policy_model_name = $1
		  AND policy_calibration_method = $2
		  AND policy_dataset_version = $3
	`
	var rows *sql.Rows
	var err error

	if userID == 0 {
		query += `
			ORDER BY as_of_time DESC, id DESC
			LIMIT $4
		`
		rows, err = r.db.QueryContext(ctx, query, modelName, calibrationMethod, datasetVersion, limit)
	} else {
		query += `
			AND user_id = $4
			ORDER BY as_of_time DESC, id DESC
			LIMIT $5
		`
		rows, err = r.db.QueryContext(ctx, query, modelName, calibrationMethod, datasetVersion, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalRun
	for rows.Next() {
		var (
			item             domain.SignalRun
			rawProbabilities []byte
			userIDValue      sql.NullInt64
			policyStatus     sql.NullString
			policyModelName  sql.NullString
			policyScenario   sql.NullString
			policyMethod     sql.NullString
			policyThreshold  sql.NullFloat64
			policyDataset    sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.AssetID,
			&userIDValue,
			&item.ModelVersion,
			&item.AsOfTime,
			&item.SignalState,
			&item.SignalDirection,
			&item.SignalProbability,
			&rawProbabilities,
			&item.Threshold,
			&item.Timeframe,
			&item.HorizonBars,
			&policyStatus,
			&policyModelName,
			&policyScenario,
			&policyMethod,
			&policyThreshold,
			&policyDataset,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.UserID = userIDValue.Int64
		if err := json.Unmarshal(rawProbabilities, &item.ClassProbabilities); err != nil {
			return nil, err
		}
		item.Policy = signalPolicyFromNullableFields(
			policyStatus,
			policyModelName,
			policyScenario,
			policyMethod,
			policyThreshold,
			policyDataset,
		)
		items = append(items, item)
	}

	return items, rows.Err()
}

func nullablePolicyString(policy *domain.SignalPolicySnapshot, value func(domain.SignalPolicySnapshot) string) sql.NullString {
	if policy == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value(*policy), Valid: true}
}

func nullablePolicyThreshold(policy *domain.SignalPolicySnapshot) sql.NullFloat64 {
	if policy == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: policy.Threshold, Valid: true}
}

func signalPolicyFromNullableFields(
	status sql.NullString,
	modelName sql.NullString,
	scenarioName sql.NullString,
	calibrationMethod sql.NullString,
	threshold sql.NullFloat64,
	datasetVersion sql.NullString,
) *domain.SignalPolicySnapshot {
	if !status.Valid && !modelName.Valid && !threshold.Valid && !datasetVersion.Valid {
		return nil
	}
	return &domain.SignalPolicySnapshot{
		PolicyStatus:      status.String,
		ModelName:         modelName.String,
		ScenarioName:      scenarioName.String,
		CalibrationMethod: calibrationMethod.String,
		Threshold:         threshold.Float64,
		DatasetVersion:    datasetVersion.String,
	}
}

type SignalEventRepository struct {
	db *sql.DB
}

func NewSignalEventRepository(db *sql.DB) *SignalEventRepository {
	return &SignalEventRepository{db: db}
}

func (r *SignalEventRepository) Create(ctx context.Context, event domain.SignalEvent) (domain.SignalEvent, error) {
	rawPayload, err := json.Marshal(event.Payload)
	if err != nil {
		return domain.SignalEvent{}, err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO signal_events (user_id, signal_run_id, event_type, model_version, ticker, idempotency_key, payload)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
		RETURNING id, created_at
	`,
		nullInt(event.UserID),
		event.SignalRunID,
		event.EventType,
		event.ModelVersion,
		event.Ticker,
		event.IdempotencyKey,
		string(rawPayload),
	).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return domain.SignalEvent{}, err
	}

	return event, nil
}

func (r *SignalEventRepository) Upsert(ctx context.Context, event domain.SignalEvent) (domain.SignalEvent, error) {
	if event.IdempotencyKey == "" {
		return r.Create(ctx, event)
	}

	rawPayload, err := json.Marshal(event.Payload)
	if err != nil {
		return domain.SignalEvent{}, err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO signal_events (user_id, signal_run_id, event_type, model_version, ticker, idempotency_key, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO UPDATE
		SET user_id = COALESCE(signal_events.user_id, EXCLUDED.user_id),
		    signal_run_id = EXCLUDED.signal_run_id,
		    event_type = EXCLUDED.event_type,
		    model_version = EXCLUDED.model_version,
		    ticker = EXCLUDED.ticker,
		    payload = EXCLUDED.payload
		RETURNING id, created_at
	`,
		nullInt(event.UserID),
		event.SignalRunID,
		event.EventType,
		event.ModelVersion,
		event.Ticker,
		event.IdempotencyKey,
		string(rawPayload),
	).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return domain.SignalEvent{}, err
	}

	return event, nil
}

func (r *SignalEventRepository) ListLatest(ctx context.Context, userID int64, limit int) ([]domain.SignalEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, signal_run_id, event_type, model_version, ticker, idempotency_key, payload, created_at
		FROM signal_events
		WHERE user_id = $1 OR user_id IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalEvent
	for rows.Next() {
		var (
			item       domain.SignalEvent
			rawPayload []byte
			ik         sql.NullString
			uid        sql.NullInt64
		)
		if err := rows.Scan(
			&item.ID,
			&uid,
			&item.SignalRunID,
			&item.EventType,
			&item.ModelVersion,
			&item.Ticker,
			&ik,
			&rawPayload,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.UserID = uid.Int64
		item.IdempotencyKey = ik.String
		if len(rawPayload) > 0 {
			_ = json.Unmarshal(rawPayload, &item.Payload)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SignalEventRepository) ListByAsset(ctx context.Context, userID int64, assetID string, limit int) ([]domain.SignalEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, signal_run_id, event_type, model_version, ticker, idempotency_key, payload, created_at
		FROM signal_events
		WHERE (user_id = $1 OR user_id IS NULL) AND ticker = $2
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, userID, assetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SignalEvent
	for rows.Next() {
		var (
			item       domain.SignalEvent
			rawPayload []byte
			ik         sql.NullString
			uid        sql.NullInt64
		)
		if err := rows.Scan(
			&item.ID,
			&uid,
			&item.SignalRunID,
			&item.EventType,
			&item.ModelVersion,
			&item.Ticker,
			&ik,
			&rawPayload,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.UserID = uid.Int64
		item.IdempotencyKey = ik.String
		if len(rawPayload) > 0 {
			_ = json.Unmarshal(rawPayload, &item.Payload)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func nullInt(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}
