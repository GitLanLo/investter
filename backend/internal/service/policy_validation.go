package service

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

var ErrPolicyValidationUnavailable = errors.New("policy validation repository is not configured")
var ErrPolicyValidationRunNotFound = errors.New("policy validation run not found")
var ErrPolicyDecisionStateInvalid = errors.New("policy decision state is invalid")

const (
	minPromotionMaturedSignals = 20
	minPromotionPrecision      = 0.55
	maxForwardSignalStaleness  = 72 * time.Hour
)

var policyValidationNow = func() time.Time {
	return time.Now().UTC()
}

func SetPolicyValidationNowForTest(now time.Time) func() {
	previous := policyValidationNow
	policyValidationNow = func() time.Time {
		return now.UTC()
	}
	return func() {
		policyValidationNow = previous
	}
}

type PolicyValidationService struct {
	repo     repository.PolicyValidationRunRepository
	research *ResearchArtifactsService
	signals  repository.SignalRunRepository
	assets   repository.AssetRepository
	market   repository.MarketDataRepository
	outcomes repository.SignalOutcomeRepository
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

func (s *PolicyValidationService) WithOutcomeData(
	assets repository.AssetRepository,
	market repository.MarketDataRepository,
	outcomes ...repository.SignalOutcomeRepository,
) *PolicyValidationService {
	s.assets = assets
	s.market = market
	if len(outcomes) > 0 {
		s.outcomes = outcomes[0]
	}
	return s
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
		ModelVersion:      policy.ModelVersion,
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
		domain.PolicyDecisionApproved,
		domain.PolicyDecisionActive,
		domain.PolicyDecisionPromoted,
		domain.PolicyDecisionBlocked,
		domain.PolicyDecisionArchived:
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

	selected, signals, err := s.loadSelectedPolicySignals(ctx, limit)
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

func (s *PolicyValidationService) LoadOutcomeSummaryByID(ctx context.Context, runID int64, limit int) (domain.PolicyOutcomeSummary, error) {
	if s == nil || s.repo == nil || s.signals == nil || s.assets == nil || s.market == nil {
		return domain.PolicyOutcomeSummary{}, ErrPolicyValidationUnavailable
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	selected, err := s.repo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PolicyOutcomeSummary{}, ErrPolicyValidationRunNotFound
		}
		return domain.PolicyOutcomeSummary{}, err
	}

	signals, err := s.signals.ListByPolicySnapshot(
		ctx,
		selected.ModelName,
		selected.CalibrationMethod,
		selected.DatasetVersion,
		limit,
	)
	if err != nil {
		return domain.PolicyOutcomeSummary{}, err
	}

	persistedOutcomes, err := s.loadPersistedOutcomeIndex(ctx, selected, limit)
	if err != nil {
		return domain.PolicyOutcomeSummary{}, err
	}

	summary := domain.PolicyOutcomeSummary{
		ValidationRunID:   selected.ID,
		DecisionState:     selected.DecisionState,
		ModelName:         selected.ModelName,
		CalibrationMethod: selected.CalibrationMethod,
		Threshold:         selected.Threshold,
		DatasetVersion:    selected.DatasetVersion,
		SignalsTotal:      len(signals),
	}
	now := policyValidationNow()
	var returnPctSum float64
	var actionReturnPctSum float64
	for _, signal := range signals {
		if signal.SignalState == domain.SignalStateActionable {
			summary.ActionableSignals++
		}
		if summary.LastSignalAt.IsZero() || signal.AsOfTime.After(summary.LastSignalAt) {
			summary.LastSignalAt = signal.AsOfTime
		}

		if persisted, ok := persistedOutcomes[signal.ID]; ok {
			accumulateOutcome(&summary, signal, persisted, &returnPctSum, &actionReturnPctSum)
			continue
		}

		outcome, ok, err := s.realizedSignalOutcome(ctx, signal)
		if err != nil {
			return domain.PolicyOutcomeSummary{}, err
		}
		if !ok {
			summary.PendingSignals++
			if signalOutcomeOverdue(signal, now) {
				summary.OverduePendingSignals++
			}
			continue
		}
		accumulateOutcome(&summary, signal, toPersistedOutcome(signal, outcome), &returnPctSum, &actionReturnPctSum)
	}
	if summary.HitSignals+summary.MissSignals > 0 {
		summary.RealizedPrecision = float64(summary.HitSignals) / float64(summary.HitSignals+summary.MissSignals)
		summary.AverageActionReturnPct = actionReturnPctSum / float64(summary.HitSignals+summary.MissSignals)
	}
	if summary.MaturedSignals > 0 {
		summary.AverageReturnPct = returnPctSum / float64(summary.MaturedSignals)
	}
	summary.PromotionBlockers = buildPromotionBlockers(summary, now)
	summary.CanPromote = len(summary.PromotionBlockers) == 0
	return summary, nil
}

func (s *PolicyValidationService) LoadOutcomeSummary(ctx context.Context, limit int) (domain.PolicyOutcomeSummary, error) {
	if s == nil || s.repo == nil || s.signals == nil || s.assets == nil || s.market == nil {
		return domain.PolicyOutcomeSummary{}, ErrPolicyValidationUnavailable
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	selected, signals, err := s.loadSelectedPolicySignals(ctx, limit)
	if err != nil {
		return domain.PolicyOutcomeSummary{}, err
	}
	persistedOutcomes, err := s.loadPersistedOutcomeIndex(ctx, selected, limit)
	if err != nil {
		return domain.PolicyOutcomeSummary{}, err
	}

	summary := domain.PolicyOutcomeSummary{
		ValidationRunID:   selected.ID,
		DecisionState:     selected.DecisionState,
		ModelName:         selected.ModelName,
		CalibrationMethod: selected.CalibrationMethod,
		Threshold:         selected.Threshold,
		DatasetVersion:    selected.DatasetVersion,
		SignalsTotal:      len(signals),
	}
	now := policyValidationNow()
	var returnPctSum float64
	var actionReturnPctSum float64
	for _, signal := range signals {
		if signal.SignalState == domain.SignalStateActionable {
			summary.ActionableSignals++
		}
		if summary.LastSignalAt.IsZero() || signal.AsOfTime.After(summary.LastSignalAt) {
			summary.LastSignalAt = signal.AsOfTime
		}

		if persisted, ok := persistedOutcomes[signal.ID]; ok {
			accumulateOutcome(&summary, signal, persisted, &returnPctSum, &actionReturnPctSum)
			continue
		}

		outcome, ok, err := s.realizedSignalOutcome(ctx, signal)
		if err != nil {
			return domain.PolicyOutcomeSummary{}, err
		}
		if !ok {
			summary.PendingSignals++
			if signalOutcomeOverdue(signal, now) {
				summary.OverduePendingSignals++
			}
			continue
		}
		accumulateOutcome(&summary, signal, toPersistedOutcome(signal, outcome), &returnPctSum, &actionReturnPctSum)
	}
	if summary.HitSignals+summary.MissSignals > 0 {
		summary.RealizedPrecision = float64(summary.HitSignals) / float64(summary.HitSignals+summary.MissSignals)
		summary.AverageActionReturnPct = actionReturnPctSum / float64(summary.HitSignals+summary.MissSignals)
	}
	if summary.MaturedSignals > 0 {
		summary.AverageReturnPct = returnPctSum / float64(summary.MaturedSignals)
	}
	summary.PromotionBlockers = buildPromotionBlockers(summary, now)
	summary.CanPromote = len(summary.PromotionBlockers) == 0
	return summary, nil
}

func (s *PolicyValidationService) MaterializeOutcomes(ctx context.Context, limit int) (domain.PolicyOutcomeSummary, error) {
	if s == nil || s.repo == nil || s.signals == nil || s.assets == nil || s.market == nil || s.outcomes == nil {
		return domain.PolicyOutcomeSummary{}, ErrPolicyValidationUnavailable
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	_, signals, err := s.loadSelectedPolicySignals(ctx, limit)
	if err != nil {
		return domain.PolicyOutcomeSummary{}, err
	}
	for _, signal := range signals {
		outcome, ok, err := s.realizedSignalOutcome(ctx, signal)
		if err != nil {
			return domain.PolicyOutcomeSummary{}, err
		}
		if !ok {
			continue
		}
		if _, err := s.outcomes.Upsert(ctx, toPersistedOutcome(signal, outcome)); err != nil {
			return domain.PolicyOutcomeSummary{}, err
		}
	}

	return s.LoadOutcomeSummary(ctx, limit)
}

func (s *PolicyValidationService) LoadOutcomeHistory(ctx context.Context, limit int) ([]domain.PolicyOutcomeRecord, error) {
	if s == nil || s.repo == nil || s.signals == nil || s.assets == nil || s.market == nil {
		return nil, ErrPolicyValidationUnavailable
	}
	if limit <= 0 || limit > 5000 {
		limit = 100
	}

	selected, signals, err := s.loadSelectedPolicySignals(ctx, limit)
	if err != nil {
		return nil, err
	}
	persistedOutcomes, err := s.loadPersistedOutcomeIndex(ctx, selected, limit)
	if err != nil {
		return nil, err
	}

	records := make([]domain.PolicyOutcomeRecord, 0, len(signals))
	for _, signal := range signals {
		if persisted, ok := persistedOutcomes[signal.ID]; ok {
			records = append(records, toOutcomeRecord(signal, persisted))
			continue
		}

		outcome, ok, err := s.realizedSignalOutcome(ctx, signal)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		records = append(records, toOutcomeRecord(signal, toPersistedOutcome(signal, outcome)))
	}

	return records, nil
}

func (s *PolicyValidationService) loadSelectedPolicySignals(
	ctx context.Context,
	limit int,
) (domain.PolicyValidationRun, []domain.SignalRun, error) {
	runs, err := s.repo.ListLatest(ctx, 100)
	if err != nil {
		return domain.PolicyValidationRun{}, nil, err
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
		return domain.PolicyValidationRun{}, nil, ErrPolicyValidationRunNotFound
	}

	signals, err := s.signals.ListByPolicySnapshot(
		ctx,
		selected.ModelName,
		selected.CalibrationMethod,
		selected.DatasetVersion,
		limit,
	)
	if err != nil {
		return domain.PolicyValidationRun{}, nil, err
	}
	return *selected, signals, nil
}

type realizedSignalOutcome struct {
	EntryPrice      float64
	ExitPrice       float64
	MaturedAt       time.Time
	ReturnPct       float64
	ActionReturnPct float64
	Hit             bool
}

func (s *PolicyValidationService) realizedSignalOutcome(
	ctx context.Context,
	signal domain.SignalRun,
) (realizedSignalOutcome, bool, error) {
	if signal.HorizonBars <= 0 {
		return realizedSignalOutcome{}, false, nil
	}

	asset, err := s.assets.GetByID(ctx, signal.AssetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return realizedSignalOutcome{}, false, ErrAssetNotFound
		}
		return realizedSignalOutcome{}, false, err
	}

	timeframeDuration := timeframeToDuration(signal.Timeframe)
	if timeframeDuration <= 0 {
		timeframeDuration = timeframeToDuration(asset.Timeframe)
	}
	if timeframeDuration <= 0 {
		timeframeDuration = 5 * time.Minute
	}
	timeframeValue := signal.Timeframe
	if timeframeValue == "" {
		timeframeValue = asset.Timeframe
	}
	to := signal.AsOfTime.UTC().Add(timeframeDuration * time.Duration(signal.HorizonBars+2))
	candles, err := s.market.ListCandles(ctx, asset.Ticker, timeframeValue, signal.AsOfTime.UTC(), to, signal.HorizonBars+2)
	if err != nil {
		return realizedSignalOutcome{}, false, err
	}
	if len(candles) <= signal.HorizonBars {
		return realizedSignalOutcome{}, false, nil
	}

	entry := candles[0]
	exit := candles[signal.HorizonBars]
	if entry.Close == 0 {
		return realizedSignalOutcome{}, false, nil
	}

	returnPct := (exit.Close - entry.Close) / entry.Close
	actionReturnPct := 0.0
	hit := false
	switch signal.SignalDirection {
	case domain.SignalDirectionUp:
		actionReturnPct = returnPct
		hit = returnPct > 0
	case domain.SignalDirectionDown:
		actionReturnPct = -returnPct
		hit = returnPct < 0
	default:
		hit = math.Abs(returnPct) == 0
	}

	return realizedSignalOutcome{
		EntryPrice:      entry.Close,
		ExitPrice:       exit.Close,
		MaturedAt:       exit.Timestamp,
		ReturnPct:       returnPct,
		ActionReturnPct: actionReturnPct,
		Hit:             hit,
	}, true, nil
}

func (s *PolicyValidationService) loadPersistedOutcomeIndex(
	ctx context.Context,
	selected domain.PolicyValidationRun,
	limit int,
) (map[int64]domain.SignalOutcome, error) {
	index := map[int64]domain.SignalOutcome{}
	if s.outcomes == nil {
		return index, nil
	}
	items, err := s.outcomes.ListByPolicySnapshot(
		ctx,
		selected.ModelName,
		selected.CalibrationMethod,
		selected.DatasetVersion,
		limit,
	)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		index[item.SignalRunID] = item
	}
	return index, nil
}

func toPersistedOutcome(signal domain.SignalRun, outcome realizedSignalOutcome) domain.SignalOutcome {
	return domain.SignalOutcome{
		SignalRunID:     signal.ID,
		AssetID:         signal.AssetID,
		MaturedAt:       outcome.MaturedAt,
		EntryPrice:      outcome.EntryPrice,
		ExitPrice:       outcome.ExitPrice,
		RawReturnPct:    outcome.ReturnPct,
		ActionReturnPct: outcome.ActionReturnPct,
		IsHit:           outcome.Hit,
	}
}

func toOutcomeRecord(signal domain.SignalRun, outcome domain.SignalOutcome) domain.PolicyOutcomeRecord {
	return domain.PolicyOutcomeRecord{
		SignalRunID:       signal.ID,
		AssetID:           signal.AssetID,
		AsOfTime:          signal.AsOfTime,
		SignalState:       signal.SignalState,
		SignalDirection:   signal.SignalDirection,
		SignalProbability: signal.SignalProbability,
		Timeframe:         signal.Timeframe,
		HorizonBars:       signal.HorizonBars,
		MaturedAt:         outcome.MaturedAt,
		EntryPrice:        outcome.EntryPrice,
		ExitPrice:         outcome.ExitPrice,
		RawReturnPct:      outcome.RawReturnPct,
		ActionReturnPct:   outcome.ActionReturnPct,
		IsHit:             outcome.IsHit,
	}
}

func accumulateOutcome(
	summary *domain.PolicyOutcomeSummary,
	signal domain.SignalRun,
	outcome domain.SignalOutcome,
	returnPctSum *float64,
	actionReturnPctSum *float64,
) {
	summary.MaturedSignals++
	*returnPctSum += outcome.RawReturnPct
	if signal.SignalState == domain.SignalStateActionable {
		*actionReturnPctSum += outcome.ActionReturnPct
		if outcome.IsHit {
			summary.HitSignals++
		} else {
			summary.MissSignals++
		}
	}
	if summary.FirstMaturedAt.IsZero() || outcome.MaturedAt.Before(summary.FirstMaturedAt) {
		summary.FirstMaturedAt = outcome.MaturedAt
	}
	if summary.LastMaturedAt.IsZero() || outcome.MaturedAt.After(summary.LastMaturedAt) {
		summary.LastMaturedAt = outcome.MaturedAt
	}
}

func buildPromotionBlockers(summary domain.PolicyOutcomeSummary, now time.Time) []domain.PolicyPromotionBlocker {
	blockers := make([]domain.PolicyPromotionBlocker, 0, 5)
	actionableMatured := summary.HitSignals + summary.MissSignals
	if actionableMatured < minPromotionMaturedSignals {
		blockers = append(blockers, domain.PolicyPromotionBlocker{
			Code:    "insufficient_matured_signals",
			Message: "Need more matured actionable signals before promotion.",
		})
	}
	if summary.PendingSignals > 0 {
		blockers = append(blockers, domain.PolicyPromotionBlocker{
			Code:    "pending_signal_horizons",
			Message: "Some shadow/live signals have not completed their forecast horizon yet.",
		})
	}
	if summary.OverduePendingSignals > 0 {
		blockers = append(blockers, domain.PolicyPromotionBlocker{
			Code:    "overdue_pending_outcomes",
			Message: "Some pending outcomes are past their maturity window and likely indicate missing candles or stale data.",
		})
	}
	if actionableMatured > 0 && summary.RealizedPrecision < minPromotionPrecision {
		blockers = append(blockers, domain.PolicyPromotionBlocker{
			Code:    "realized_precision_below_threshold",
			Message: "Realized precision is below the minimum promotion threshold.",
		})
	}
	if !summary.LastSignalAt.IsZero() && now.Sub(summary.LastSignalAt) > maxForwardSignalStaleness {
		blockers = append(blockers, domain.PolicyPromotionBlocker{
			Code:    "stale_forward_validation",
			Message: "The latest shadow/live policy signal is stale, so forward validation is no longer current.",
		})
	}
	return blockers
}

func signalOutcomeOverdue(signal domain.SignalRun, now time.Time) bool {
	timeframeDuration := timeframeToDuration(signal.Timeframe)
	if timeframeDuration <= 0 {
		timeframeDuration = 5 * time.Minute
	}
	if signal.HorizonBars <= 0 {
		return false
	}
	expectedMaturity := signal.AsOfTime.UTC().Add(timeframeDuration * time.Duration(signal.HorizonBars))
	return now.After(expectedMaturity.Add(timeframeDuration))
}

func timeframeToDuration(timeframe string) time.Duration {
	switch timeframe {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 0
	}
}
