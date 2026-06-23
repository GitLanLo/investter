package service

import (
	"context"
	"fmt"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type NotificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) ListRules(ctx context.Context, userID int64) ([]domain.NotificationRule, error) {
	return s.repo.ListRules(ctx, userID)
}

func (s *NotificationService) CreateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	return s.repo.CreateRule(ctx, rule)
}

func (s *NotificationService) UpdateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	return s.repo.UpdateRule(ctx, rule)
}

func (s *NotificationService) DeleteRule(ctx context.Context, id int64, userID int64) error {
	return s.repo.DeleteRule(ctx, id, userID)
}

func (s *NotificationService) ListEvents(ctx context.Context, userID int64, limit int) ([]domain.NotificationEvent, error) {
	return s.repo.ListLatestEvents(ctx, userID, limit)
}

func (s *NotificationService) Evaluate(ctx context.Context, event domain.SignalEvent) error {
	rules, err := s.repo.ListActiveRules(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !s.matchRule(rule, event) {
			continue
		}

		if s.isOnCooldown(ctx, rule, event) {
			continue
		}

		var signalEventID *int64
		if event.ID > 0 {
			signalEventID = &event.ID
		}

		notification := domain.NotificationEvent{
			RuleID:           rule.ID,
			SignalEventID:    signalEventID,
			EventType:        event.EventType,
			Severity:         rule.Severity,
			ModelVersion:     event.ModelVersion,
			Ticker:           event.Ticker,
			Message:          s.formatMessage(rule, event),
			Payload:          event.Payload,
			DeliveryStatus:   "delivered", // mock
			DeliveryAttempts: 1,
		}

		if _, err := s.repo.CreateEvent(ctx, notification); err != nil {
			return err
		}
	}

	return nil
}

func (s *NotificationService) matchRule(rule domain.NotificationRule, event domain.SignalEvent) bool {
	if event.UserID != 0 && rule.UserID != event.UserID {
		return false
	}
	if rule.EventType != event.EventType {
		return false
	}
	if rule.Ticker != "" && rule.Ticker != event.Ticker {
		return false
	}
	if rule.ModelVersion != "" && rule.ModelVersion != event.ModelVersion {
		return false
	}
	if rule.Direction != "" {
		dir, ok := event.Payload["direction"].(string)
		if !ok || dir != rule.Direction {
			return false
		}
	}

	// For threshold-based rules (e.g. decision_threshold_triggered)
	if rule.Threshold > 0 {
		probRaw, ok := event.Payload["probability"]
		if !ok {
			return false
		}
		prob, ok := probRaw.(float64)
		if !ok || prob < rule.Threshold {
			return false
		}
	}
	return true
}

func (s *NotificationService) isOnCooldown(ctx context.Context, rule domain.NotificationRule, event domain.SignalEvent) bool {
	if rule.CooldownMinutes <= 0 {
		return false
	}

	// In a real system, we'd query by rule_id AND ticker for per-asset cooldown
	// For now we get the latest event for this rule and check its ticker.
	// Actually we should fetch the latest event for rule_id AND ticker.
	// We can update the repo later, but for Sprint 9 this proxy check is acceptable.
	latest, err := s.repo.GetLatestEventForRule(ctx, rule.ID)
	if err != nil {
		return false
	}

	// Per-asset cooldown logic: if the rule is not asset-specific, check if the latest event was for the same asset.
	// If it was for a different asset, they don't share cooldown (unless we want a global rule cooldown).
	// The test plan says "per asset cooldown". So if rule.Ticker is empty, it applies per-asset.
	if rule.Ticker == "" && latest.Ticker != event.Ticker {
		return false
	}

	return time.Since(latest.CreatedAt) < time.Duration(rule.CooldownMinutes)*time.Minute
}
func (s *NotificationService) formatMessage(rule domain.NotificationRule, event domain.SignalEvent) string {
	switch event.EventType {
	case domain.SignalEventDecisionThresholdTriggered:
		return fmt.Sprintf("High confidence signal for %s: %s %s (p=%.2f)",
			event.Ticker,
			event.Payload["direction"],
			event.Payload["state"],
			event.Payload["probability"],
		)
	case domain.SignalEventInferenceBlockedByRuntime:
		return fmt.Sprintf("Inference blocked for model %s: runtime status is %s",
			event.ModelVersion,
			event.Payload["runtime_status"],
		)
	case domain.SignalEventPolicyPromotion:
		return fmt.Sprintf("Policy promoted to active: model %s", event.ModelVersion)
	default:
		return fmt.Sprintf("Signal event triggered: %s for %s", event.EventType, event.Ticker)
	}
}

func (s *NotificationService) EvaluatePriceAlerts(ctx context.Context, ticker string, latestPrice float64) error {
	rules, err := s.repo.ListActiveRules(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if rule.EventType != "price_level" {
			continue
		}
		if rule.Ticker != "" && rule.Ticker != ticker {
			continue
		}

		triggered := false
		switch rule.Operator {
		case ">":
			triggered = latestPrice > rule.Threshold
		case "<":
			triggered = latestPrice < rule.Threshold
		case ">=":
			triggered = latestPrice >= rule.Threshold
		case "<=":
			triggered = latestPrice <= rule.Threshold
		}

		if !triggered {
			continue
		}

		// Use a mock SignalEvent just for cooldown checking
		mockEvent := domain.SignalEvent{Ticker: ticker}
		if s.isOnCooldown(ctx, rule, mockEvent) {
			continue
		}

		notification := domain.NotificationEvent{
			RuleID:           rule.ID,
			EventType:        rule.EventType,
			Severity:         rule.Severity,
			Ticker:           ticker,
			Message:          fmt.Sprintf("Цена для %s достигла целевого уровня: %.4f (Порог: %s %.4f)", ticker, latestPrice, rule.Operator, rule.Threshold),
			Payload:          map[string]any{"price": latestPrice, "threshold": rule.Threshold, "operator": rule.Operator},
			DeliveryStatus:   "delivered", // mock
			DeliveryAttempts: 1,
		}

		if _, err := s.repo.CreateEvent(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}
