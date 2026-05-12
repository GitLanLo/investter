package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"invest/backend/internal/domain"
	"github.com/lib/pq"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) ListRules(ctx context.Context, userID int64) ([]domain.NotificationRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, ticker, event_type, target_indicator, operator, threshold, secondary_threshold, 
		       severity, direction, model_version, is_enabled, cooldown_minutes, expires_at, trigger_mode, delivery_channels,
		       created_at, updated_at
		FROM notification_rules_v2
		WHERE user_id = $1
		ORDER BY id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.NotificationRule
	for rows.Next() {
		var item domain.NotificationRule
		var ticker, targetIndicator, operator, direction, modelVersion sql.NullString
		var threshold, secondaryThreshold sql.NullFloat64
		var expiresAt *time.Time
		var deliveryChannels pq.StringArray

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&ticker,
			&item.EventType,
			&targetIndicator,
			&operator,
			&threshold,
			&secondaryThreshold,
			&item.Severity,
			&direction,
			&modelVersion,
			&item.IsEnabled,
			&item.CooldownMinutes,
			&expiresAt,
			&item.TriggerMode,
			&deliveryChannels,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Ticker = ticker.String
		item.TargetIndicator = targetIndicator.String
		item.Operator = operator.String
		item.Direction = direction.String
		item.ModelVersion = modelVersion.String
		item.Threshold = threshold.Float64
		item.SecondaryThreshold = secondaryThreshold.Float64
		item.ExpiresAt = expiresAt
		item.DeliveryChannels = []string(deliveryChannels)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) ListActiveRules(ctx context.Context) ([]domain.NotificationRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, ticker, event_type, target_indicator, operator, threshold, secondary_threshold, 
		       severity, direction, model_version, is_enabled, cooldown_minutes, expires_at, trigger_mode, delivery_channels,
		       created_at, updated_at
		FROM notification_rules_v2
		WHERE is_enabled = TRUE AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.NotificationRule
	for rows.Next() {
		var item domain.NotificationRule
		var ticker, targetIndicator, operator, direction, modelVersion sql.NullString
		var threshold, secondaryThreshold sql.NullFloat64
		var expiresAt *time.Time
		var deliveryChannels pq.StringArray

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&ticker,
			&item.EventType,
			&targetIndicator,
			&operator,
			&threshold,
			&secondaryThreshold,
			&item.Severity,
			&direction,
			&modelVersion,
			&item.IsEnabled,
			&item.CooldownMinutes,
			&expiresAt,
			&item.TriggerMode,
			&deliveryChannels,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Ticker = ticker.String
		item.TargetIndicator = targetIndicator.String
		item.Operator = operator.String
		item.Direction = direction.String
		item.ModelVersion = modelVersion.String
		item.Threshold = threshold.Float64
		item.SecondaryThreshold = secondaryThreshold.Float64
		item.ExpiresAt = expiresAt
		item.DeliveryChannels = []string(deliveryChannels)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) GetRuleByID(ctx context.Context, id int64, userID int64) (domain.NotificationRule, error) {
	var item domain.NotificationRule
	var ticker, targetIndicator, operator, direction, modelVersion sql.NullString
	var threshold, secondaryThreshold sql.NullFloat64
	var expiresAt *time.Time
	var deliveryChannels pq.StringArray

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, ticker, event_type, target_indicator, operator, threshold, secondary_threshold, 
		       severity, direction, model_version, is_enabled, cooldown_minutes, expires_at, trigger_mode, delivery_channels,
		       created_at, updated_at
		FROM notification_rules_v2
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(
		&item.ID,
		&item.UserID,
		&ticker,
		&item.EventType,
		&targetIndicator,
		&operator,
		&threshold,
		&secondaryThreshold,
		&item.Severity,
		&direction,
		&modelVersion,
		&item.IsEnabled,
		&item.CooldownMinutes,
		&expiresAt,
		&item.TriggerMode,
		&deliveryChannels,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.NotificationRule{}, err
	}
	item.Ticker = ticker.String
	item.TargetIndicator = targetIndicator.String
	item.Operator = operator.String
	item.Direction = direction.String
	item.ModelVersion = modelVersion.String
	item.Threshold = threshold.Float64
	item.SecondaryThreshold = secondaryThreshold.Float64
	item.ExpiresAt = expiresAt
	item.DeliveryChannels = []string(deliveryChannels)
	return item, nil
}

func (r *NotificationRepository) CreateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	if rule.TriggerMode == "" {
		rule.TriggerMode = "once"
	}
	if len(rule.DeliveryChannels) == 0 {
		rule.DeliveryChannels = []string{"app"}
	}

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO notification_rules_v2 (
			user_id, ticker, event_type, target_indicator, operator, threshold, secondary_threshold, 
			severity, direction, model_version, is_enabled, cooldown_minutes, expires_at, trigger_mode, delivery_channels
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`,
		rule.UserID,
		nullString(rule.Ticker),
		rule.EventType,
		nullString(rule.TargetIndicator),
		nullString(rule.Operator),
		nullFloat(rule.Threshold),
		nullFloat(rule.SecondaryThreshold),
		rule.Severity,
		nullString(rule.Direction),
		nullString(rule.ModelVersion),
		rule.IsEnabled,
		rule.CooldownMinutes,
		rule.ExpiresAt,
		rule.TriggerMode,
		stringArray(rule.DeliveryChannels),
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return domain.NotificationRule{}, err
	}
	return rule, nil
}

func (r *NotificationRepository) UpdateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	if rule.TriggerMode == "" {
		rule.TriggerMode = "once"
	}
	if len(rule.DeliveryChannels) == 0 {
		rule.DeliveryChannels = []string{"app"}
	}

	err := r.db.QueryRowContext(ctx, `
		UPDATE notification_rules_v2
		SET ticker = $3,
		    event_type = $4,
		    target_indicator = $5,
		    operator = $6,
		    threshold = $7,
		    secondary_threshold = $8,
		    severity = $9,
		    direction = $10,
		    model_version = $11,
		    is_enabled = $12,
		    cooldown_minutes = $13,
		    expires_at = $14,
		    trigger_mode = $15,
		    delivery_channels = $16,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`,
		rule.ID,
		rule.UserID,
		nullString(rule.Ticker),
		rule.EventType,
		nullString(rule.TargetIndicator),
		nullString(rule.Operator),
		nullFloat(rule.Threshold),
		nullFloat(rule.SecondaryThreshold),
		rule.Severity,
		nullString(rule.Direction),
		nullString(rule.ModelVersion),
		rule.IsEnabled,
		rule.CooldownMinutes,
		rule.ExpiresAt,
		rule.TriggerMode,
		stringArray(rule.DeliveryChannels),
	).Scan(&rule.UpdatedAt)
	if err != nil {
		return domain.NotificationRule{}, err
	}
	return rule, nil
}

func (r *NotificationRepository) DeleteRule(ctx context.Context, id int64, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM notification_rules_v2 WHERE id = $1 AND user_id = $2`, id, userID)
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

func (r *NotificationRepository) ListLatestEvents(ctx context.Context, userID int64, limit int) ([]domain.NotificationEvent, error) {
	query := `
		SELECT e.id, e.rule_id, e.signal_event_id, e.event_type, e.severity, e.model_version, e.ticker, e.message, e.payload, e.delivery_status, e.delivery_attempts, e.created_at
		FROM notification_events e
	`
	var rows *sql.Rows
	var err error

	if userID == 0 {
		query += `
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT $1
		`
		rows, err = r.db.QueryContext(ctx, query, limit)
	} else {
		query += `
			JOIN notification_rules_v2 nr ON e.rule_id = nr.id
			WHERE nr.user_id = $1
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT $2
		`
		rows, err = r.db.QueryContext(ctx, query, userID, limit)
	}

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
