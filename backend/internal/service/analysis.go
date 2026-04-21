package service

import (
	"context"
	"database/sql"
	"errors"
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
}

func NewAnalysisService(
	assetRepo repository.AssetRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
) *AnalysisService {
	return &AnalysisService{
		assetRepo:  assetRepo,
		modelRepo:  modelRepo,
		signalRepo: signalRepo,
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

	return s.signalRepo.Create(ctx, run)
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
	var (
		entry domain.ModelRegistryEntry
		err   error
	)

	if version == "" || version == "active" {
		entry, err = s.modelRepo.GetActive(ctx)
	} else {
		entry, err = s.modelRepo.GetByVersion(ctx, version)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ModelRegistryEntry{}, ErrModelNotFound
		}
		return domain.ModelRegistryEntry{}, err
	}

	return entry, nil
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
