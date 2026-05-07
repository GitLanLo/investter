package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"invest/backend/internal/domain"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) ListRules(ctx context.Context) ([]domain.NotificationRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ticker, event_type, severity, direction, model_version, threshold, is_enabled, cooldown_minutes, created_at, updated_at
		FROM notification_rules_v2
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.NotificationRule
	for rows.Next() {
		var item domain.NotificationRule
		var ticker sql.NullString
		var direction sql.NullString
		var modelVersion sql.NullString
		var threshold sql.NullFloat64
		if err := rows.Scan(
			&item.ID,
			&ticker,
			&item.EventType,
			&item.Severity,
			&direction,
			&modelVersion,
			&threshold,
			&item.IsEnabled,
			&item.CooldownMinutes,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Ticker = ticker.String
		item.Direction = direction.String
		item.ModelVersion = modelVersion.String
		item.Threshold = threshold.Float64
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) ListActiveRules(ctx context.Context) ([]domain.NotificationRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ticker, event_type, severity, direction, model_version, threshold, is_enabled, cooldown_minutes, created_at, updated_at
		FROM notification_rules_v2
		WHERE is_enabled = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.NotificationRule
	for rows.Next() {
		var item domain.NotificationRule
		var ticker sql.NullString
		var direction sql.NullString
		var modelVersion sql.NullString
		var threshold sql.NullFloat64
		if err := rows.Scan(
			&item.ID,
			&ticker,
			&item.EventType,
			&item.Severity,
			&direction,
			&modelVersion,
			&threshold,
			&item.IsEnabled,
			&item.CooldownMinutes,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Ticker = ticker.String
		item.Direction = direction.String
		item.ModelVersion = modelVersion.String
		item.Threshold = threshold.Float64
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) GetRuleByID(ctx context.Context, id int64) (domain.NotificationRule, error) {
	var item domain.NotificationRule
	var ticker sql.NullString
	var direction sql.NullString
	var modelVersion sql.NullString
	var threshold sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, ticker, event_type, severity, direction, model_version, threshold, is_enabled, cooldown_minutes, created_at, updated_at
		FROM notification_rules_v2
		WHERE id = $1
	`, id).Scan(
		&item.ID,
		&ticker,
		&item.EventType,
		&item.Severity,
		&direction,
		&modelVersion,
		&threshold,
		&item.IsEnabled,
		&item.CooldownMinutes,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.NotificationRule{}, err
	}
	item.Ticker = ticker.String
	item.Direction = direction.String
	item.ModelVersion = modelVersion.String
	item.Threshold = threshold.Float64
	return item, nil
}

func (r *NotificationRepository) CreateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO notification_rules_v2 (ticker, event_type, severity, direction, model_version, threshold, is_enabled, cooldown_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`,
		nullString(rule.Ticker),
		rule.EventType,
		rule.Severity,
		nullString(rule.Direction),
		nullString(rule.ModelVersion),
		nullFloat(rule.Threshold),
		rule.IsEnabled,
		rule.CooldownMinutes,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return domain.NotificationRule{}, err
	}
	return rule, nil
}

func (r *NotificationRepository) UpdateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	err := r.db.QueryRowContext(ctx, `
		UPDATE notification_rules_v2
		SET ticker = $2,
		    event_type = $3,
		    severity = $4,
		    direction = $5,
		    model_version = $6,
		    threshold = $7,
		    is_enabled = $8,
		    cooldown_minutes = $9,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`,
		rule.ID,
		nullString(rule.Ticker),
		rule.EventType,
		rule.Severity,
		nullString(rule.Direction),
		nullString(rule.ModelVersion),
		nullFloat(rule.Threshold),
		rule.IsEnabled,
		rule.CooldownMinutes,
	).Scan(&rule.UpdatedAt)
	if err != nil {
		return domain.NotificationRule{}, err
	}
	return rule, nil
}

func (r *NotificationRepository) DeleteRule(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM notification_rules_v2 WHERE id = $1`, id)
	return err
}

func (r *NotificationRepository) CreateEvent(ctx context.Context, event domain.NotificationEvent) (domain.NotificationEvent, error) {
	rawPayload, err := json.Marshal(event.Payload)
	if err != nil {
		return domain.NotificationEvent{}, err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO notification_events (rule_id, signal_event_id, event_type, severity, model_version, ticker, message, payload, delivery_status, delivery_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (rule_id, signal_event_id) WHERE signal_event_id IS NOT NULL DO NOTHING
		RETURNING id, created_at
	`,
		event.RuleID,
		event.SignalEventID,
		event.EventType,
		event.Severity,
		event.ModelVersion,
		nullString(event.Ticker),
		event.Message,
		string(rawPayload),
		event.DeliveryStatus,
		event.DeliveryAttempts,
	).Scan(&event.ID, &event.CreatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.NotificationEvent{}, err
	}

	return event, nil
}

func (r *NotificationRepository) ListLatestEvents(ctx context.Context, limit int) ([]domain.NotificationEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, rule_id, signal_event_id, event_type, severity, model_version, ticker, message, payload, delivery_status, delivery_attempts, created_at
		FROM notification_events
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.NotificationEvent
	for rows.Next() {
		var item domain.NotificationEvent
		var ticker sql.NullString
		var rawPayload []byte
		if err := rows.Scan(
			&item.ID,
			&item.RuleID,
			&item.SignalEventID,
			&item.EventType,
			&item.Severity,
			&item.ModelVersion,
			&ticker,
			&item.Message,
			&rawPayload,
			&item.DeliveryStatus,
			&item.DeliveryAttempts,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Ticker = ticker.String
		if len(rawPayload) > 0 {
			_ = json.Unmarshal(rawPayload, &item.Payload)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) GetLatestEventForRule(ctx context.Context, ruleID int64) (domain.NotificationEvent, error) {
	var item domain.NotificationEvent
	var ticker sql.NullString
	var rawPayload []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, rule_id, signal_event_id, event_type, severity, model_version, ticker, message, payload, delivery_status, delivery_attempts, created_at
		FROM notification_events
		WHERE rule_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, ruleID).Scan(
		&item.ID,
		&item.RuleID,
		&item.SignalEventID,
		&item.EventType,
		&item.Severity,
		&item.ModelVersion,
		&ticker,
		&item.Message,
		&rawPayload,
		&item.DeliveryStatus,
		&item.DeliveryAttempts,
		&item.CreatedAt,
	)
	if err != nil {
		return domain.NotificationEvent{}, err
	}
	item.Ticker = ticker.String
	if len(rawPayload) > 0 {
		_ = json.Unmarshal(rawPayload, &item.Payload)
	}
	return item, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullFloat(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}
