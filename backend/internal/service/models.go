package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

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

var ErrModelRuntimeBlocked = errors.New("model activation blocked: runtime is metadata_only or unavailable")

func (s *ModelRegistryService) Activate(ctx context.Context, version string) error {
	entry, err := s.repo.GetByVersion(ctx, version)
	if err != nil {
		return err
	}

	manifest, err := LoadManifest(entry.ManifestPath)
	if err != nil {
		return err
	}

	if manifest.RuntimeStatus != RuntimeStatusAvailable {
		return ErrModelRuntimeBlocked
	}

	return s.repo.Activate(ctx, version)
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

const (
	RuntimeStatusAvailable    = "available"
	RuntimeStatusMetadataOnly = "metadata_only"
	RuntimeStatusUnavailable  = "unavailable"
)

func LoadManifest(path string) (domain.ModelManifest, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return domain.ModelManifest{}, err
	}

	var manifest domain.ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		normalized, ok := normalizeManifestCreatedAt(data)
		if !ok {
			return domain.ModelManifest{}, err
		}
		if retryErr := json.Unmarshal(normalized, &manifest); retryErr != nil {
			return domain.ModelManifest{}, err
		}
	}

	// Handle aliases and defaults
	if manifest.ModelType == "" && manifest.ModelFamily != "" {
		manifest.ModelType = manifest.ModelFamily
	}
	if manifest.ModelFamily == "" && manifest.ModelType != "" {
		manifest.ModelFamily = manifest.ModelType
	}
	if manifest.TaskType == "" && manifest.Task != "" {
		manifest.TaskType = manifest.Task
	}
	if manifest.Task == "" && manifest.TaskType != "" {
		manifest.Task = manifest.TaskType
	}
	if manifest.Timeframe == "" && manifest.InputTimeframe != "" {
		manifest.Timeframe = manifest.InputTimeframe
	}
	if manifest.InputTimeframe == "" && manifest.Timeframe != "" {
		manifest.InputTimeframe = manifest.Timeframe
	}
	if manifest.HorizonBars == 0 && manifest.PredictionHorizonBars != 0 {
		manifest.HorizonBars = manifest.PredictionHorizonBars
	}
	if manifest.PredictionHorizonBars == 0 && manifest.HorizonBars != 0 {
		manifest.PredictionHorizonBars = manifest.HorizonBars
	}
	if len(manifest.FeatureColumns) == 0 && len(manifest.FeatureOrder) != 0 {
		manifest.FeatureColumns = manifest.FeatureOrder
	}
	if len(manifest.FeatureOrder) == 0 && len(manifest.FeatureColumns) != 0 {
		manifest.FeatureOrder = manifest.FeatureColumns
	}
	if manifest.DecisionThreshold == 0 && manifest.Threshold != 0 {
		manifest.DecisionThreshold = manifest.Threshold
	}
	if manifest.Threshold == 0 && manifest.DecisionThreshold != 0 {
		manifest.Threshold = manifest.DecisionThreshold
	}
	if manifest.NormalizationArtifactPath == "" && manifest.Normalization != "" {
		manifest.NormalizationArtifactPath = manifest.Normalization
	}
	if manifest.Normalization == "" && manifest.NormalizationArtifactPath != "" {
		manifest.Normalization = manifest.NormalizationArtifactPath
	}

	// Reporting runtime status
	if manifest.ExportFormat == "onnx" || manifest.ExportFormat == "torchscript" {
		// Currently backend does not have native inference for these
		manifest.RuntimeStatus = RuntimeStatusMetadataOnly
	} else if manifest.ExportFormat == "joblib" || manifest.ExportFormat == "pkl" || manifest.ExportFormat == "stub" {
		// Tabular baseline might be available if using sidecar or native implementation
		// For Sprint 8, we assume metadata_only for neural, and maybe available for tabular baseline
		if manifest.ModelType == "hgb_multiclass" || manifest.ModelType == "stub_deterministic_v1" {
			manifest.RuntimeStatus = RuntimeStatusAvailable
		} else {
			manifest.RuntimeStatus = RuntimeStatusMetadataOnly
		}
	} else {
		manifest.RuntimeStatus = RuntimeStatusUnavailable
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

func normalizeManifestCreatedAt(data []byte) ([]byte, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}

	createdRaw, ok := raw["created_at"]
	if !ok {
		return nil, false
	}

	var createdAt string
	if err := json.Unmarshal(createdRaw, &createdAt); err != nil || createdAt == "" {
		return nil, false
	}
	if _, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		return nil, false
	}

	layouts := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, createdAt, time.UTC)
		if err != nil {
			continue
		}
		encoded, err := json.Marshal(parsed.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return nil, false
		}
		raw["created_at"] = encoded
		normalized, err := json.Marshal(raw)
		if err != nil {
			return nil, false
		}
		return normalized, true
	}

	return nil, false
}
