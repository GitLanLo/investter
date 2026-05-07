package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type PolicyValidationProvider interface {
	LoadOutcomeSummary(ctx context.Context, limit int) (domain.PolicyOutcomeSummary, error)
	LoadOutcomeSummaryByID(ctx context.Context, runID int64, limit int) (domain.PolicyOutcomeSummary, error)
}

type PolicyPromotionService struct {
	policyRepo repository.PolicyValidationRunRepository
	models     *ModelRegistryService
	validation PolicyValidationProvider
	eventRepo  repository.SignalEventRepository
	notifier   *NotificationService
}

func NewPolicyPromotionService(
	policyRepo repository.PolicyValidationRunRepository,
	models *ModelRegistryService,
	validation PolicyValidationProvider,
	eventRepo repository.SignalEventRepository,
	notifier *NotificationService,
) *PolicyPromotionService {
	return &PolicyPromotionService{
		policyRepo: policyRepo,
		models:     models,
		validation: validation,
		eventRepo:  eventRepo,
		notifier:   notifier,
	}
}

var ErrPromotionBlocked = errors.New("policy promotion blocked by rules")

func (s *PolicyPromotionService) Promote(ctx context.Context, runID int64) error {
	selected, err := s.policyRepo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPolicyValidationRunNotFound
		}
		return err
	}

	if selected.DecisionState != domain.PolicyDecisionShadowLive && selected.DecisionState != domain.PolicyDecisionApproved {
		return fmt.Errorf("policy in state %s cannot be promoted directly to active", selected.DecisionState)
	}

	// 1. Check blockers
	summary, err := s.validation.LoadOutcomeSummaryByID(ctx, runID, 1000)
	if err != nil {
		return err
	}
	if !summary.CanPromote {
		blockerCodes := make([]string, len(summary.PromotionBlockers))
		for i, b := range summary.PromotionBlockers {
			blockerCodes[i] = b.Code
		}
		_ = s.policyRepo.LogPromotion(ctx, domain.PolicyPromotionLog{
			PolicyValidationRunID: runID,
			Actor:                 "system",
			PreviousState:         selected.DecisionState,
			NextState:             selected.DecisionState,
			Blockers:              blockerCodes,
			Notes:                 "promotion blocked",
		})
		return fmt.Errorf("%w: %v", ErrPromotionBlocked, summary.PromotionBlockers)
	}

	// 2. Identify model version
	modelVersion := selected.ModelVersion
	if modelVersion == "" {
		return errors.New("model version missing from policy validation run")
	}

	// Save previous active model for compensation
	var prevActiveVersion string
	prevActive, err := s.models.repo.GetActive(ctx)
	if err == nil {
		prevActiveVersion = prevActive.ModelVersion
	}

	// 3. Activate in registry
	if err := s.models.Activate(ctx, modelVersion); err != nil {
		return fmt.Errorf("model activation failed: %w", err)
	}

	// 4. Update policy states and Log (Atomic)
	promoLog := domain.PolicyPromotionLog{
		PolicyValidationRunID: runID,
		Actor:                 "system",
		PreviousState:         selected.DecisionState,
		NextState:             domain.PolicyDecisionActive,
		Blockers:              nil,
		Notes:                 "promoted to active",
	}

	if err := s.policyRepo.DemoteCurrentAndPromote(ctx, runID, "demoted by promotion of run "+fmt.Sprint(runID), promoLog); err != nil {
		// Compensate: return previous model to active
		if prevActiveVersion != "" {
			_ = s.models.Activate(context.Background(), prevActiveVersion)
		}
		return err
	}

	if s.eventRepo != nil {
		event := domain.SignalEvent{
			EventType:      domain.SignalEventPolicyPromotion,
			ModelVersion:   modelVersion,
			IdempotencyKey: fmt.Sprintf("%s|%d", domain.SignalEventPolicyPromotion, runID),
			Payload:        map[string]any{"run_id": runID},
		}
		saved, _ := s.eventRepo.Upsert(ctx, event)
		if s.notifier != nil {
			_ = s.notifier.Evaluate(ctx, saved)
		}
	}

	return nil
}

func (s *PolicyPromotionService) Rollback(ctx context.Context, runID int64) error {
	selected, err := s.policyRepo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPolicyValidationRunNotFound
		}
		return err
	}
	
	if selected.ModelVersion == "" {
		return errors.New("rollback target has no model version")
	}
	
	var prevActiveVersion string
	prevActive, err := s.models.repo.GetActive(ctx)
	if err == nil {
		prevActiveVersion = prevActive.ModelVersion
	}

	if err := s.models.Activate(ctx, selected.ModelVersion); err != nil {
		return err
	}
	
	promoLog := domain.PolicyPromotionLog{
		PolicyValidationRunID: runID,
		Actor:                 "system",
		PreviousState:         selected.DecisionState,
		NextState:             domain.PolicyDecisionActive,
		Blockers:              nil,
		Notes:                 "reactivated by rollback",
	}

	if err := s.policyRepo.DemoteCurrentAndPromote(ctx, runID, "demoted by rollback to run "+fmt.Sprint(runID), promoLog); err != nil {
		if prevActiveVersion != "" {
			_ = s.models.Activate(context.Background(), prevActiveVersion)
		}
		return err
	}
	
	if s.eventRepo != nil {
		event := domain.SignalEvent{
			EventType:      domain.SignalEventPolicyRollback,
			ModelVersion:   selected.ModelVersion,
			IdempotencyKey: fmt.Sprintf("%s|%d", domain.SignalEventPolicyRollback, runID),
			Payload:        map[string]any{"run_id": runID},
		}
		saved, _ := s.eventRepo.Upsert(ctx, event)
		if s.notifier != nil {
			_ = s.notifier.Evaluate(ctx, saved)
		}
	}

	return nil
}
