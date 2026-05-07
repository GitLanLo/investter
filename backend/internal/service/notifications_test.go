package service

import (
	"context"
	"testing"

	"invest/backend/internal/domain"
)

func TestNotificationService_Evaluate_ThresholdRequiresProbability(t *testing.T) {
	repo := &MockNotificationRepo{
		Rules: []domain.NotificationRule{
			{
				ID:              1,
				EventType:       domain.SignalEventDecisionThresholdTriggered,
				Severity:        domain.SeverityWarning,
				Threshold:       0.7,
				IsEnabled:       true,
				CooldownMinutes: 0,
			},
		},
	}
	svc := NewNotificationService(repo)

	err := svc.Evaluate(context.Background(), domain.SignalEvent{
		ID:           1,
		EventType:    domain.SignalEventDecisionThresholdTriggered,
		ModelVersion: "available_v1",
		Ticker:       "SBER",
		Payload:      map[string]any{"direction": domain.SignalDirectionUp},
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(repo.Events) != 0 {
		t.Fatalf("expected no notification without probability, got %d", len(repo.Events))
	}
}

func TestNotificationService_Evaluate_UsesSavedSignalEventIDForDedupe(t *testing.T) {
	repo := &MockNotificationRepo{
		Rules: []domain.NotificationRule{
			{
				ID:              1,
				EventType:       domain.SignalEventDecisionThresholdTriggered,
				Severity:        domain.SeverityWarning,
				Direction:       domain.SignalDirectionUp,
				Threshold:       0.6,
				IsEnabled:       true,
				CooldownMinutes: 0,
			},
		},
	}
	svc := NewNotificationService(repo)
	event := domain.SignalEvent{
		ID:           42,
		EventType:    domain.SignalEventDecisionThresholdTriggered,
		ModelVersion: "available_v1",
		Ticker:       "SBER",
		Payload: map[string]any{
			"direction":   domain.SignalDirectionUp,
			"state":       domain.SignalStateActionable,
			"probability": 0.75,
		},
	}

	for i := 0; i < 2; i++ {
		if err := svc.Evaluate(context.Background(), event); err != nil {
			t.Fatalf("evaluate %d: %v", i+1, err)
		}
	}
	if len(repo.Events) != 1 {
		t.Fatalf("expected one deduplicated notification event, got %d", len(repo.Events))
	}
	if repo.Events[0].SignalEventID == nil || *repo.Events[0].SignalEventID != 42 {
		t.Fatalf("expected saved signal_event_id 42, got %v", repo.Events[0].SignalEventID)
	}
}
