package domain

import "time"

type Asset struct {
	ID        string
	Ticker    string
	Name      string
	Exchange  string
	Timeframe string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Candle struct {
	Timestamp  time.Time
	Open       float64
	High       float64
	Low        float64
	Close      float64
	Volume     int64
	Ticker     string
	Timeframe  string
	Source     string
	IngestedAt time.Time
}

type FactorBar struct {
	Timestamp  time.Time
	Factor     string
	Open       float64
	High       float64
	Low        float64
	Close      float64
	Volume     int64
	Timeframe  string
	Source     string
	IngestedAt time.Time
}

type Watchlist struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WatchlistItem struct {
	ID          int64
	WatchlistID int64
	AssetID     string
	Position    int
	CreatedAt   time.Time
}

type ModelRegistryEntry struct {
	ID                   int64
	ModelVersion         string
	ModelType            string
	Status               string
	Timeframe            string
	HorizonBars          int
	FeatureSchemaVersion string
	ManifestPath         string
	CreatedAt            time.Time
}

type ModelManifest struct {
	ModelVersion              string             `json:"model_version"`
	ModelType                 string             `json:"model_type"`
	TaskType                  string             `json:"task_type"`
	Classes                   []string           `json:"classes"`
	Timeframe                 string             `json:"timeframe"`
	HorizonBars               int                `json:"horizon_bars"`
	FeatureSchemaVersion      string             `json:"feature_schema_version"`
	FeatureColumns            []string           `json:"feature_columns"`
	NormalizationArtifactPath string             `json:"normalization_artifact_path"`
	ExportFormat              string             `json:"export_format"`
	ModelArtifactPath         string             `json:"model_artifact_path"`
	Metrics                   map[string]float64 `json:"metrics"`
	DecisionThreshold         float64            `json:"decision_threshold"`
	CreatedAt                 time.Time          `json:"created_at"`
}

type SignalClassProbabilities struct {
	Up      float64 `json:"up"`
	Down    float64 `json:"down"`
	NoTrade float64 `json:"no_trade"`
}

type SignalRun struct {
	ID                 int64
	AssetID            string
	ModelVersion       string
	AsOfTime           time.Time
	SignalState        string
	SignalDirection    string
	SignalProbability  float64
	ClassProbabilities SignalClassProbabilities
	Threshold          float64
	Timeframe          string
	HorizonBars        int
	Policy             *SignalPolicySnapshot
	CreatedAt          time.Time
}

type SignalPolicySnapshot struct {
	PolicyStatus      string
	ModelName         string
	ScenarioName      string
	CalibrationMethod string
	Threshold         float64
	DatasetVersion    string
}

type PolicyValidationRun struct {
	ID                int64
	PolicyStatus      string
	ModelName         string
	ScenarioName      string
	CalibrationMethod string
	Threshold         float64
	DatasetVersion    string
	Validation        PolicyValidationMetrics
	Test              PolicyValidationMetrics
	DecisionState     string
	Notes             string
	CreatedAt         time.Time
}

type PolicyValidationMetrics struct {
	ActionableF1  float64
	Precision     float64
	Coverage      float64
	ActionableECE float64
}

type PolicyShadowSummary struct {
	ValidationRunID   int64
	DecisionState     string
	ModelName         string
	CalibrationMethod string
	Threshold         float64
	DatasetVersion    string
	SignalsTotal      int
	ActionableSignals int
	NoTradeSignals    int
	UpSignals         int
	DownSignals       int
	ObservedCoverage  float64
	FirstSignalAt     time.Time
	LastSignalAt      time.Time
}

const (
	ModelStatusActive = "active"

	SignalStateActionable = "actionable"
	SignalStateNoTrade    = "no_trade"

	SignalDirectionUp   = "up"
	SignalDirectionDown = "down"
	SignalDirectionNone = "none"

	PolicyDecisionCandidate  = "candidate"
	PolicyDecisionShadowLive = "shadow_live"
	PolicyDecisionPromoted   = "promoted"
	PolicyDecisionBlocked    = "blocked"
)
