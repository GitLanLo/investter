package service

import (
	"context"
	"database/sql"
	"errors"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

var ErrPolicyValidationUnavailable = errors.New("policy validation repository is not configured")
var ErrPolicyValidationRunNotFound = errors.New("policy validation run not found")
var ErrPolicyDecisionStateInvalid = errors.New("policy decision state is invalid")

type PolicyValidationService struct {
	repo     repository.PolicyValidationRunRepository
	research *ResearchArtifactsService
	signals  repository.SignalRunRepository
}

func NewPolicyValidationService(
	repo repository.PolicyValidationRunRepository,
	research *ResearchArtifactsService,
	signals ...repository.SignalRunRepository,
) *PolicyValidationService {
	var signalRepo repository.SignalRunRepository
	if len(signals) > 0 {
		signalRepo = signals[0]
	}
	return &PolicyValidationService{repo: repo, research: research, signals: signalRepo}
}

func (s *PolicyValidationService) CreateFromCurrentPolicy(ctx context.Context, notes string) (domain.PolicyValidationRun, error) {
	if s == nil || s.repo == nil || s.research == nil {
		return domain.PolicyValidationRun{}, ErrPolicyValidationUnavailable
	}

	policy, err := s.research.LoadProductionPolicy(ctx)
	if err != nil {
		return domain.PolicyValidationRun{}, err
	}

	decisionState := domain.PolicyDecisionCandidate
	if policy.PolicyStatus != "production_candidate" {
		decisionState = domain.PolicyDecisionBlocked
	}

	return s.repo.Create(ctx, domain.PolicyValidationRun{
		PolicyStatus:      policy.PolicyStatus,
		ModelName:         policy.ModelName,
		ScenarioName:      policy.ScenarioName,
		CalibrationMethod: policy.CalibrationMethod,
		Threshold:         policy.Threshold,
		DatasetVersion:    policy.DatasetVersion,
		Validation: domain.PolicyValidationMetrics{
			ActionableF1:  policy.Validation.ActionableF1,
			Precision:     policy.Validation.Precision,
			Coverage:      policy.Validation.Coverage,
			ActionableECE: policy.Validation.ActionableECE,
		},
		Test: domain.PolicyValidationMetrics{
			ActionableF1:  policy.Test.ActionableF1,
			Precision:     policy.Test.Precision,
			Coverage:      policy.Test.Coverage,
			ActionableECE: policy.Test.ActionableECE,
		},
		DecisionState: decisionState,
		Notes:         notes,
	})
}

func (s *PolicyValidationService) ListLatest(ctx context.Context, limit int) ([]domain.PolicyValidationRun, error) {
	if s == nil || s.repo == nil {
		return nil, ErrPolicyValidationUnavailable
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListLatest(ctx, limit)
}

func (s *PolicyValidationService) UpdateDecisionState(
	ctx context.Context,
	id int64,
	decisionState string,
	notes string,
) (domain.PolicyValidationRun, error) {
	if s == nil || s.repo == nil {
		return domain.PolicyValidationRun{}, ErrPolicyValidationUnavailable
	}
	if id <= 0 {
		return domain.PolicyValidationRun{}, ErrPolicyValidationRunNotFound
	}
	if !isValidPolicyDecisionState(decisionState) {
		return domain.PolicyValidationRun{}, ErrPolicyDecisionStateInvalid
	}
	run, err := s.repo.UpdateDecisionState(ctx, id, decisionState, notes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PolicyValidationRun{}, ErrPolicyValidationRunNotFound
		}
		return domain.PolicyValidationRun{}, err
	}
	return run, nil
}

func isValidPolicyDecisionState(decisionState string) bool {
	switch decisionState {
	case domain.PolicyDecisionCandidate,
		domain.PolicyDecisionShadowLive,
		domain.PolicyDecisionPromoted,
		domain.PolicyDecisionBlocked:
		return true
	default:
		return false
	}
}

func (s *PolicyValidationService) LoadShadowSummary(ctx context.Context, limit int) (domain.PolicyShadowSummary, error) {
	if s == nil || s.repo == nil || s.signals == nil {
		return domain.PolicyShadowSummary{}, ErrPolicyValidationUnavailable
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	runs, err := s.repo.ListLatest(ctx, 100)
	if err != nil {
		return domain.PolicyShadowSummary{}, err
	}

	var selected *domain.PolicyValidationRun
	for index := range runs {
		if runs[index].DecisionState == domain.PolicyDecisionShadowLive || runs[index].DecisionState == domain.PolicyDecisionPromoted {
			selected = &runs[index]
			break
		}
	}
	if selected == nil && len(runs) > 0 {
		selected = &runs[0]
	}
	if selected == nil {
		return domain.PolicyShadowSummary{}, ErrPolicyValidationRunNotFound
	}

	signals, err := s.signals.ListByPolicySnapshot(
		ctx,
		selected.ModelName,
		selected.CalibrationMethod,
		selected.DatasetVersion,
		limit,
	)
	if err != nil {
		return domain.PolicyShadowSummary{}, err
	}

	summary := domain.PolicyShadowSummary{
		ValidationRunID:   selected.ID,
		DecisionState:     selected.DecisionState,
		ModelName:         selected.ModelName,
		CalibrationMethod: selected.CalibrationMethod,
		Threshold:         selected.Threshold,
		DatasetVersion:    selected.DatasetVersion,
		SignalsTotal:      len(signals),
	}
	for _, signal := range signals {
		if signal.SignalState == domain.SignalStateActionable {
			summary.ActionableSignals++
		} else {
			summary.NoTradeSignals++
		}
		switch signal.SignalDirection {
		case domain.SignalDirectionUp:
			summary.UpSignals++
		case domain.SignalDirectionDown:
			summary.DownSignals++
		}
		if summary.LastSignalAt.IsZero() || signal.AsOfTime.After(summary.LastSignalAt) {
			summary.LastSignalAt = signal.AsOfTime
		}
		if summary.FirstSignalAt.IsZero() || signal.AsOfTime.Before(summary.FirstSignalAt) {
			summary.FirstSignalAt = signal.AsOfTime
		}
	}
	if summary.SignalsTotal > 0 {
		summary.ObservedCoverage = float64(summary.ActionableSignals) / float64(summary.SignalsTotal)
	}
	return summary, nil
}
