package service

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type ResearchArtifactsOverview struct {
	GeneratedAt        time.Time
	DatasetManifest    map[string]any
	ResearchSummary    map[string]any
	CalibrationSummary map[string]any
	GridMatrix         map[string]any
	SourcePaths        map[string]string
	Warnings           []string
}

type ResearchArtifactDocument struct {
	Key         string
	Title       string
	Path        string
	ContentType string
	Content     string
}

type ProductionPolicySnapshot struct {
	GeneratedAt       time.Time
	PolicyStatus      string
	ModelName         string
	ModelVersion      string
	ScenarioName      string
	CalibrationMethod string
	Threshold         float64
	Timeframe         string
	HorizonBars       int
	DatasetVersion    string
	FeatureSchema     string
	TrainRows         int
	ValidationRows    int
	TestRows          int
	Validation        ProductionPolicyMetrics
	Test              ProductionPolicyMetrics
	SourcePaths       map[string]string
	Warnings          []string
}

type ProductionPolicyMetrics struct {
	ActionableF1  float64
	Precision     float64
	Coverage      float64
	ActionableECE float64
}

type ResearchArtifactsService struct {
	dataRoot     string
	researchRoot string
}

func NewResearchArtifactsService(dataRoot string, researchRoot string) *ResearchArtifactsService {
	return &ResearchArtifactsService{
		dataRoot:     dataRoot,
		researchRoot: researchRoot,
	}
}

func (s *ResearchArtifactsService) LoadOverview(ctx context.Context) (ResearchArtifactsOverview, error) {
	overview := ResearchArtifactsOverview{
		GeneratedAt: time.Now().UTC(),
		SourcePaths: map[string]string{},
		Warnings:    []string{},
	}

	datasetPath, datasetManifest, datasetWarnings, err := findLatestJSON(
		ctx,
		filepath.Join(s.dataRoot, "datasets"),
		"manifest.json",
		func(payload map[string]any) bool {
			_, hasDatasetVersion := payload["dataset_version"]
			_, hasFeatureSchema := payload["feature_schema_version"]
			return hasDatasetVersion && hasFeatureSchema
		},
	)
	if err != nil {
		return ResearchArtifactsOverview{}, err
	}
	overview.Warnings = append(overview.Warnings, datasetWarnings...)
	if datasetPath != "" {
		overview.DatasetManifest = datasetManifest
		overview.SourcePaths["dataset_manifest"] = datasetPath
	}

	researchPath, researchSummary, researchWarnings, err := findLatestJSON(
		ctx,
		s.researchRoot,
		"summary.json",
		isResearchSummaryPayload,
	)
	if err != nil {
		return ResearchArtifactsOverview{}, err
	}
	overview.Warnings = append(overview.Warnings, researchWarnings...)
	if researchPath != "" {
		overview.ResearchSummary = researchSummary
		overview.SourcePaths["research_summary"] = researchPath
	}

	calibrationPath, calibrationSummary, calibrationWarnings, err := findLatestJSON(
		ctx,
		s.researchRoot,
		"summary.json",
		func(payload map[string]any) bool {
			_, hasMethods := payload["methods"]
			_, hasProductionCandidate := payload["production_candidate"]
			_, hasResearchCandidate := payload["research_candidate"]
			_, hasModelDir := payload["model_dir"]
			return hasMethods && hasProductionCandidate && hasResearchCandidate && hasModelDir
		},
	)
	if err != nil {
		return ResearchArtifactsOverview{}, err
	}
	overview.Warnings = append(overview.Warnings, calibrationWarnings...)
	if calibrationPath != "" {
		overview.CalibrationSummary = calibrationSummary
		overview.SourcePaths["calibration_summary"] = calibrationPath
	}

	gridMatrixPath, gridMatrix, gridWarnings, err := findLatestJSON(
		ctx,
		s.researchRoot,
		"timeframe_horizon_matrix.json",
		func(payload map[string]any) bool {
			_, hasGridName := payload["grid_name"]
			_, hasResults := payload["results"]
			return hasGridName && hasResults
		},
	)
	if err != nil {
		return ResearchArtifactsOverview{}, err
	}
	overview.Warnings = append(overview.Warnings, gridWarnings...)
	if gridMatrixPath != "" {
		overview.GridMatrix = gridMatrix
		overview.SourcePaths["grid_matrix"] = gridMatrixPath
	}

	return overview, nil
}

func isResearchSummaryPayload(payload map[string]any) bool {
	_, hasProductionCandidate := payload["production_candidate"]
	_, hasResearchCandidate := payload["research_candidate"]
	if !hasProductionCandidate || !hasResearchCandidate {
		return false
	}

	_, hasCalibrationMethods := payload["methods"]
	_, hasModelDir := payload["model_dir"]
	if hasCalibrationMethods || hasModelDir {
		return false
	}

	_, hasScenarios := payload["scenarios"]
	_, hasModels := payload["models"]
	_, hasModelGroup := payload["model_group"]
	return hasScenarios || hasModels || hasModelGroup
}

func (s *ResearchArtifactsService) LoadDocuments(ctx context.Context) ([]ResearchArtifactDocument, error) {
	overview, err := s.LoadOverview(ctx)
	if err != nil {
		return nil, err
	}

	documents := make([]ResearchArtifactDocument, 0, 5)

	appendJSONDocument := func(key string, title string, path string, payload map[string]any) error {
		if path == "" || payload == nil {
			return nil
		}
		content, err := prettyJSON(payload)
		if err != nil {
			return err
		}
		documents = append(documents, ResearchArtifactDocument{
			Key:         key,
			Title:       title,
			Path:        path,
			ContentType: "json",
			Content:     content,
		})
		return nil
	}

	if err := appendJSONDocument(
		"dataset_manifest",
		"Dataset manifest",
		overview.SourcePaths["dataset_manifest"],
		overview.DatasetManifest,
	); err != nil {
		return nil, err
	}
	if err := appendJSONDocument(
		"research_summary",
		"Research summary",
		overview.SourcePaths["research_summary"],
		overview.ResearchSummary,
	); err != nil {
		return nil, err
	}
	if err := appendJSONDocument(
		"calibration_summary",
		"Calibration summary",
		overview.SourcePaths["calibration_summary"],
		overview.CalibrationSummary,
	); err != nil {
		return nil, err
	}
	if err := appendJSONDocument(
		"grid_matrix",
		"Timeframe Horizon Matrix",
		overview.SourcePaths["grid_matrix"],
		overview.GridMatrix,
	); err != nil {
		return nil, err
	}

	for _, item := range []struct {
		key   string
		title string
		path  string
	}{
		{
			key:   "research_report",
			title: "Research report",
			path:  siblingReportPath(overview.SourcePaths["research_summary"]),
		},
		{
			key:   "calibration_report",
			title: "Calibration report",
			path:  siblingReportPath(overview.SourcePaths["calibration_summary"]),
		},
	} {
		if item.path == "" {
			continue
		}
		content, err := os.ReadFile(filepath.Clean(item.path))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		documents = append(documents, ResearchArtifactDocument{
			Key:         item.key,
			Title:       item.title,
			Path:        item.path,
			ContentType: "markdown",
			Content:     string(content),
		})
	}

	return documents, nil
}

func (s *ResearchArtifactsService) LoadProductionPolicy(ctx context.Context) (ProductionPolicySnapshot, error) {
	overview, err := s.LoadOverview(ctx)
	if err != nil {
		return ProductionPolicySnapshot{}, err
	}

	policy := ProductionPolicySnapshot{
		GeneratedAt:  overview.GeneratedAt,
		PolicyStatus: "incomplete",
		SourcePaths:  overview.SourcePaths,
		Warnings:     append([]string{}, overview.Warnings...),
	}

	dataset := overview.DatasetManifest
	policy.DatasetVersion = stringField(dataset, "dataset_version")
	policy.FeatureSchema = stringField(dataset, "feature_schema_version")
	policy.Timeframe = stringField(dataset, "timeframe")
	policy.HorizonBars = intField(dataset, "horizon_bars")
	policy.TrainRows = intField(mapField(dataset, "train_range"), "rows")
	policy.ValidationRows = intField(mapField(dataset, "val_range"), "rows")
	policy.TestRows = intField(mapField(dataset, "test_range"), "rows")

	researchCandidate := mapField(overview.ResearchSummary, "production_candidate")
	calibrationCandidate := mapField(overview.CalibrationSummary, "production_candidate")
	if researchCandidate == nil {
		policy.Warnings = append(policy.Warnings, "research production_candidate is missing")
	}
	if calibrationCandidate == nil {
		policy.Warnings = append(policy.Warnings, "calibration production_candidate is missing")
	}

	policy.ModelName = stringField(researchCandidate, "model_name")
	policy.ScenarioName = stringField(researchCandidate, "scenario_name")
	if !calibrationMatchesResearch(overview.CalibrationSummary, policy.ModelName) {
		if calibrationCandidate != nil {
			policy.Warnings = append(policy.Warnings, "calibration production_candidate does not match research production model")
		}
		calibrationCandidate = nil
	}

	policy.CalibrationMethod = stringField(calibrationCandidate, "method")
	policy.Threshold = floatField(calibrationCandidate, "selected_threshold")
	if policy.Threshold == 0 {
		policy.Threshold = floatField(researchCandidate, "selected_threshold")
	}
	policy.Validation = productionPolicyMetrics(mapField(calibrationCandidate, "validation"))
	policy.Test = productionPolicyMetrics(mapField(calibrationCandidate, "test"))

	researchGatePassed := boolField(mapField(researchCandidate, "validation_gate"), "passed")
	calibrationGatePassed := boolField(mapField(calibrationCandidate, "validation_gate"), "passed")
	switch {
	case researchCandidate == nil || calibrationCandidate == nil:
		policy.PolicyStatus = "incomplete"
	case researchGatePassed && calibrationGatePassed:
		policy.PolicyStatus = "production_candidate"
	case researchGatePassed:
		policy.PolicyStatus = "calibration_review"
	default:
		policy.PolicyStatus = "research_review"
	}

	return policy, nil
}

func calibrationMatchesResearch(calibrationSummary map[string]any, researchModelName string) bool {
	if calibrationSummary == nil || researchModelName == "" {
		return false
	}
	calibrationModelName := stringField(calibrationSummary, "model_name")
	if calibrationModelName == "" {
		return true
	}
	return calibrationModelName == researchModelName
}

func findLatestJSON(
	ctx context.Context,
	root string,
	fileName string,
	match func(payload map[string]any) bool,
) (string, map[string]any, []string, error) {
	warnings := []string{}
	if root == "" {
		return "", nil, append(warnings, "json root is not configured"), nil
	}

	rootInfo, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, append(warnings, "json root does not exist: "+root), nil
		}
		return "", nil, warnings, err
	}
	if !rootInfo.IsDir() {
		return "", nil, append(warnings, "json root is not a directory: "+root), nil
	}

	var (
		bestPath    string
		bestPayload map[string]any
		bestModTime time.Time
	)

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			warnings = append(warnings, "walk error: "+visitErr.Error())
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() || filepath.Base(path) != fileName {
			return nil
		}

		payload, loadErr := readJSONMap(path)
		if loadErr != nil {
			warnings = append(warnings, "skip invalid json: "+path)
			return nil
		}
		if !match(payload) {
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			warnings = append(warnings, "skip unreadable file info: "+path)
			return nil
		}
		if bestPath == "" || info.ModTime().After(bestModTime) {
			bestPath = path
			bestPayload = payload
			bestModTime = info.ModTime()
		}
		return nil
	})
	if walkErr != nil {
		return "", nil, warnings, walkErr
	}
	if bestPath == "" {
		warnings = append(warnings, "no matching json found under "+root)
	}
	return bestPath, bestPayload, warnings, nil
}

func readJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func prettyJSON(payload map[string]any) (string, error) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func siblingReportPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "report.md")
}

func productionPolicyMetrics(payload map[string]any) ProductionPolicyMetrics {
	signalMetrics := mapField(payload, "signal_metrics")
	probabilityMetrics := mapField(payload, "probability_metrics")

	return ProductionPolicyMetrics{
		ActionableF1:  firstFloatField(payload, signalMetrics, "actionable_f1"),
		Precision:     firstFloatField(payload, signalMetrics, "precision_actionable_signal"),
		Coverage:      firstFloatField(payload, signalMetrics, "signal_coverage"),
		ActionableECE: firstFloatField(payload, probabilityMetrics, "actionable_expected_calibration_error"),
	}
}

func firstFloatField(primary map[string]any, secondary map[string]any, key string) float64 {
	if value := floatField(primary, key); value != 0 {
		return value
	}
	return floatField(secondary, key)
}

func mapField(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return nil
	}
	value, ok := payload[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func stringField(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
}

func intField(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func floatField(payload map[string]any, key string) float64 {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func boolField(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	value, _ := payload[key].(bool)
	return value
}
