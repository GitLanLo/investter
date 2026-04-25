package service

import (
	"context"
	"fmt"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

var ErrJobsUnavailable = fmt.Errorf("jobs service is not configured")

const JobTypeOutcomesMaterialize = "outcomes_materialize"

type JobService struct {
	repo   repository.JobRunRepository
	policy *PolicyValidationService
}

func NewJobService(repo repository.JobRunRepository, policy *PolicyValidationService) *JobService {
	return &JobService{repo: repo, policy: policy}
}

func (s *JobService) ListLatest(ctx context.Context, limit int) ([]domain.JobRun, error) {
	if s == nil || s.repo == nil {
		return nil, ErrJobsUnavailable
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListLatest(ctx, limit)
}

func (s *JobService) RunOutcomeMaterialization(ctx context.Context, limit int) (domain.JobRun, error) {
	if s == nil || s.repo == nil || s.policy == nil {
		return domain.JobRun{}, ErrJobsUnavailable
	}

	run, err := s.repo.Create(ctx, domain.JobRun{
		JobType: JobTypeOutcomesMaterialize,
		Status:  domain.JobStatusRunning,
		Payload: map[string]any{"limit": limit},
	})
	if err != nil {
		return domain.JobRun{}, err
	}

	summary, err := s.policy.MaterializeOutcomes(ctx, limit)
	if err != nil {
		failedRun, finishErr := s.repo.Finish(ctx, run.ID, domain.JobStatusFailed, map[string]any{"limit": limit}, err.Error())
		if finishErr == nil {
			return failedRun, err
		}
		return domain.JobRun{}, err
	}

	payload := map[string]any{
		"limit":                   limit,
		"validation_run_id":       summary.ValidationRunID,
		"signals_total":           summary.SignalsTotal,
		"actionable_signals":      summary.ActionableSignals,
		"matured_signals":         summary.MaturedSignals,
		"pending_signals":         summary.PendingSignals,
		"overdue_pending_signals": summary.OverduePendingSignals,
		"hit_signals":             summary.HitSignals,
		"miss_signals":            summary.MissSignals,
		"realized_precision":      summary.RealizedPrecision,
		"average_return_pct":      summary.AverageReturnPct,
		"action_return_pct":       summary.AverageActionReturnPct,
		"can_promote":             summary.CanPromote,
		"promotion_blocker_count": len(summary.PromotionBlockers),
	}
	return s.repo.Finish(ctx, run.ID, domain.JobStatusSucceeded, payload, "")
}
