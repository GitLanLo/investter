package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type ModelRegistryService struct {
	repo repository.ModelRegistryRepository
}

func NewModelRegistryService(repo repository.ModelRegistryRepository) *ModelRegistryService {
	return &ModelRegistryService{repo: repo}
}

func (s *ModelRegistryService) List(ctx context.Context) ([]domain.ModelRegistryEntry, error) {
	return s.repo.List(ctx)
}

func (s *ModelRegistryService) GetActive(ctx context.Context) (domain.ModelRegistryEntry, error) {
	return s.repo.GetActive(ctx)
}

func (s *ModelRegistryService) GetByVersion(ctx context.Context, version string) (domain.ModelRegistryEntry, error) {
	if version == "" {
		return domain.ModelRegistryEntry{}, errors.New("model version must not be empty")
	}
	return s.repo.GetByVersion(ctx, version)
}

func (s *ModelRegistryService) Register(ctx context.Context, entry domain.ModelRegistryEntry) error {
	if entry.ModelVersion == "" {
		return errors.New("model version must not be empty")
	}
	if entry.ManifestPath == "" {
		return errors.New("manifest path must not be empty")
	}

	return s.repo.Register(ctx, entry)
}

func (s *ModelRegistryService) EnsureBootstrapActive(ctx context.Context, manifestPath string) (domain.ModelManifest, error) {
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return domain.ModelManifest{}, err
	}

	entry := domain.ModelRegistryEntry{
		ModelVersion:         manifest.ModelVersion,
		ModelType:            manifest.ModelType,
		Status:               domain.ModelStatusActive,
		Timeframe:            manifest.Timeframe,
		HorizonBars:          manifest.HorizonBars,
		FeatureSchemaVersion: manifest.FeatureSchemaVersion,
		ManifestPath:         manifestPath,
	}

	if err := s.Register(ctx, entry); err != nil {
		return domain.ModelManifest{}, err
	}

	return manifest, nil
}

func LoadManifest(path string) (domain.ModelManifest, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return domain.ModelManifest{}, err
	}

	var manifest domain.ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return domain.ModelManifest{}, err
	}

	if manifest.ModelVersion == "" {
		return domain.ModelManifest{}, errors.New("manifest.model_version must not be empty")
	}
	if manifest.ModelType == "" {
		return domain.ModelManifest{}, errors.New("manifest.model_type must not be empty")
	}
	if manifest.Timeframe == "" {
		return domain.ModelManifest{}, errors.New("manifest.timeframe must not be empty")
	}
	if manifest.HorizonBars <= 0 {
		return domain.ModelManifest{}, errors.New("manifest.horizon_bars must be positive")
	}
	if manifest.FeatureSchemaVersion == "" {
		return domain.ModelManifest{}, errors.New("manifest.feature_schema_version must not be empty")
	}
	if len(manifest.FeatureColumns) == 0 {
		return domain.ModelManifest{}, errors.New("manifest.feature_columns must not be empty")
	}
	if manifest.DecisionThreshold <= 0 || manifest.DecisionThreshold > 1 {
		return domain.ModelManifest{}, errors.New("manifest.decision_threshold must be in (0, 1]")
	}

	return manifest, nil
}
