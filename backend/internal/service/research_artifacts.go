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
		func(payload map[string]any) bool {
			_, hasScenarios := payload["scenarios"]
			_, hasProductionCandidate := payload["production_candidate"]
			_, hasResearchCandidate := payload["research_candidate"]
			return hasScenarios && hasProductionCandidate && hasResearchCandidate
		},
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

	return overview, nil
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
