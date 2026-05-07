package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

var (
	ErrAssetNotFound = errors.New("asset not found")
	ErrModelNotFound = errors.New("model not found")
)

type RunAnalysisInput struct {
	AssetID      string
	AsOfTime     time.Time
	ModelVersion string
	Timeframe    string
}

type AnalysisService struct {
	assetRepo  repository.AssetRepository
	modelRepo  repository.ModelRegistryRepository
	signalRepo repository.SignalRunRepository
	eventRepo  repository.SignalEventRepository
	notifier   *NotificationService
	research   *ResearchArtifactsService
}

func NewAnalysisService(
	assetRepo repository.AssetRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
	eventRepo repository.SignalEventRepository,
	notifier *NotificationService,
	research ...*ResearchArtifactsService,
) *AnalysisService {
	var researchService *ResearchArtifactsService
	if len(research) > 0 {
		researchService = research[0]
	}
	return &AnalysisService{
		assetRepo:  assetRepo,
		modelRepo:  modelRepo,
		signalRepo: signalRepo,
		eventRepo:  eventRepo,
		notifier:   notifier,
		research:   researchService,
	}
}

func (s *AnalysisService) Run(ctx context.Context, input RunAnalysisInput) (domain.SignalRun, error) {
	if input.AssetID == "" {
		return domain.SignalRun{}, errors.New("assetID must not be empty")
	}

	asset, err := s.assetRepo.GetByID(ctx, input.AssetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SignalRun{}, ErrAssetNotFound
		}
		return domain.SignalRun{}, err
	}

	entry, err := s.resolveModel(ctx, input.ModelVersion)
	if err != nil {
		return domain.SignalRun{}, err
	}

	manifest, err := LoadManifest(entry.ManifestPath)
	if err != nil {
		return domain.SignalRun{}, err
	}

	if manifest.RuntimeStatus != RuntimeStatusAvailable {
		if s.eventRepo != nil {
			event := domain.SignalEvent{
				EventType:      domain.SignalEventInferenceBlockedByRuntime,
				ModelVersion:   entry.ModelVersion,
				Ticker:         asset.Ticker,
				IdempotencyKey: fmt.Sprintf("%s|%s|%s|%s", domain.SignalEventInferenceBlockedByRuntime, entry.ModelVersion, asset.Ticker, input.AsOfTime.UTC().Format(time.RFC3339)),
				Payload:        map[string]any{"runtime_status": manifest.RuntimeStatus},
			}
			saved, _ := s.eventRepo.Upsert(ctx, event)
			if s.notifier != nil {
				_ = s.notifier.Evaluate(ctx, saved)
			}
		}
		return domain.SignalRun{}, ErrModelRuntimeBlocked
	}

	if input.Timeframe != "" && input.Timeframe != manifest.Timeframe {
		return domain.SignalRun{}, errors.New("requested timeframe does not match active model timeframe")
	}

	asOfTime := input.AsOfTime.UTC()
	if asOfTime.IsZero() {
		asOfTime = time.Now().UTC().Truncate(time.Minute)
	}

	probabilities := deterministicProbabilities(asset.ID, asOfTime, manifest.ModelVersion)
	state, direction, signalProbability := classifySignal(probabilities, manifest.DecisionThreshold)

	run := domain.SignalRun{
		AssetID:            asset.ID,
		ModelVersion:       manifest.ModelVersion,
		AsOfTime:           asOfTime,
		SignalState:        state,
		SignalDirection:    direction,
		SignalProbability:  signalProbability,
		ClassProbabilities: probabilities,
		Threshold:          manifest.DecisionThreshold,
		Timeframe:          manifest.Timeframe,
		HorizonBars:        manifest.HorizonBars,
	}
	if policy := s.loadSignalPolicySnapshot(ctx); policy != nil {
		run.Policy = policy
	}

	res, err := s.signalRepo.Create(ctx, run)
	if err == nil && s.eventRepo != nil {
		eventType := domain.SignalEventClassificationSuccess
		if res.SignalState == domain.SignalStateActionable {
			eventType = domain.SignalEventDecisionThresholdTriggered
		}
		event := domain.SignalEvent{
			SignalRunID:    &res.ID,
			EventType:      eventType,
			ModelVersion:   res.ModelVersion,
			Ticker:         asset.Ticker,
			IdempotencyKey: fmt.Sprintf("%s|%s|%s|%s", eventType, asset.Ticker, res.AsOfTime.UTC().Format(time.RFC3339), res.ModelVersion),
			Payload: map[string]any{
				"state":       res.SignalState,
				"direction":   res.SignalDirection,
				"probability": res.SignalProbability,
			},
		}
		saved, upsertErr := s.eventRepo.Upsert(ctx, event)
		if upsertErr == nil && s.notifier != nil {
			_ = s.notifier.Evaluate(ctx, saved)
		}
	}

	return res, err
}

func (s *AnalysisService) ListLatest(ctx context.Context, limit int) ([]domain.SignalRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.signalRepo.ListLatest(ctx, limit)
}

func (s *AnalysisService) ListEvents(ctx context.Context, limit int) ([]domain.SignalEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.eventRepo.ListLatest(ctx, limit)
}

func (s *AnalysisService) ListEventsByAsset(ctx context.Context, assetID string, limit int) ([]domain.SignalEvent, error) {
	if assetID == "" {
		return nil, errors.New("assetID must not be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.eventRepo.ListByAsset(ctx, assetID, limit)
}

func (s *AnalysisService) ListByAsset(ctx context.Context, assetID string, limit int) ([]domain.SignalRun, error) {
	if assetID == "" {
		return nil, errors.New("assetID must not be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	return s.signalRepo.ListByAsset(ctx, assetID, limit)
}

func (s *AnalysisService) resolveModel(ctx context.Context, version string) (domain.ModelRegistryEntry, error) {
	if version != "" && version != "active" {
		entry, err := s.modelRepo.GetByVersion(ctx, version)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ModelRegistryEntry{}, ErrModelNotFound
			}
			return domain.ModelRegistryEntry{}, err
		}
		return entry, nil
	}

	// Try active model first
	entry, err := s.modelRepo.GetActive(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ModelRegistryEntry{}, ErrModelNotFound
		}
		return domain.ModelRegistryEntry{}, err
	}

	// Check if active is available
	manifest, err := LoadManifest(entry.ManifestPath)
	if err == nil && manifest.RuntimeStatus == RuntimeStatusAvailable {
		return entry, nil
	}

	// Fallback: search for latest available model in registry
	items, err := s.modelRepo.List(ctx)
	if err != nil {
		return entry, nil // Return original entry (which might be blocked) if list fails
	}

	for _, item := range items {
		m, err := LoadManifest(item.ManifestPath)
		if err != nil {
			continue
		}
		// Fallback must match timeframe and horizon if they are known or fixed by entry
		if m.RuntimeStatus == RuntimeStatusAvailable &&
			m.Timeframe == entry.Timeframe &&
			m.HorizonBars == entry.HorizonBars {
			return item, nil
		}
	}

	return entry, nil
}

func (s *AnalysisService) loadSignalPolicySnapshot(ctx context.Context) *domain.SignalPolicySnapshot {
	if s.research == nil {
		return nil
	}
	policy, err := s.research.LoadProductionPolicy(ctx)
	if err != nil || policy.PolicyStatus == "" {
		return nil
	}
	return &domain.SignalPolicySnapshot{
		PolicyStatus:      policy.PolicyStatus,
		ModelName:         policy.ModelName,
		ScenarioName:      policy.ScenarioName,
		CalibrationMethod: policy.CalibrationMethod,
		Threshold:         policy.Threshold,
		DatasetVersion:    policy.DatasetVersion,
	}
}

func deterministicProbabilities(assetID string, asOfTime time.Time, modelVersion string) domain.SignalClassProbabilities {
	seed := deterministicSeed(assetID, asOfTime, modelVersion)
	rng := rand.New(rand.NewSource(seed))

	dominantIndex := rng.Intn(3)
	raw := []float64{
		0.15 + rng.Float64()*0.25,
		0.15 + rng.Float64()*0.25,
		0.15 + rng.Float64()*0.25,
	}
	raw[dominantIndex] += 1.75 + rng.Float64()*0.75

	total := raw[0] + raw[1] + raw[2]
	return domain.SignalClassProbabilities{
		Up:      raw[0] / total,
		Down:    raw[1] / total,
		NoTrade: raw[2] / total,
	}
}

func classifySignal(
	probabilities domain.SignalClassProbabilities,
	threshold float64,
) (state string, direction string, signalProbability float64) {
	signalProbability = probabilities.Up
	direction = domain.SignalDirectionUp
	winningProbability := probabilities.Up
	winningClass := domain.SignalDirectionUp

	if probabilities.Down > signalProbability {
		signalProbability = probabilities.Down
		direction = domain.SignalDirectionDown
	}
	if probabilities.Down > winningProbability {
		winningProbability = probabilities.Down
		winningClass = domain.SignalDirectionDown
	}
	if probabilities.NoTrade > winningProbability {
		winningProbability = probabilities.NoTrade
		winningClass = domain.SignalStateNoTrade
	}

	if winningClass == domain.SignalDirectionUp && probabilities.Up >= threshold {
		return domain.SignalStateActionable, domain.SignalDirectionUp, signalProbability
	}
	if winningClass == domain.SignalDirectionDown && probabilities.Down >= threshold {
		return domain.SignalStateActionable, domain.SignalDirectionDown, signalProbability
	}

	return domain.SignalStateNoTrade, domain.SignalDirectionNone, signalProbability
}

func deterministicSeed(assetID string, asOfTime time.Time, modelVersion string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(assetID))
	_, _ = hasher.Write([]byte("|"))
	_, _ = hasher.Write([]byte(asOfTime.UTC().Format(time.RFC3339)))
	_, _ = hasher.Write([]byte("|"))
	_, _ = hasher.Write([]byte(modelVersion))
	return int64(hasher.Sum64())
}
