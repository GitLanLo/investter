package domain

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Permissions  []string  `json:"permissions"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

const (
	UserRoleUser       = "user"
	UserRoleAdmin      = "admin"
	UserRoleSuperAdmin = "super_admin"

	PermissionMLAdmin      = "ml_admin"
	PermissionUserAdmin    = "user_admin"
	PermissionMarketAccess = "market_access"
)

type Asset struct {
	ID                  string
	Ticker              string
	Name                string
	Exchange            string
	Timeframe           string
	IsActive            bool
	Figi                string
	InstrumentUID       string
	ClassCode           string
	InstrumentType      string
	Lot                 int32
	Currency            string
	APITradeAvailable   bool
	First1MinCandleDate *time.Time
	First1DayCandleDate *time.Time
	ModelSupported      bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ModelFamily               string             `json:"model_family"`
	TaskType                  string             `json:"task_type"`
	Task                      string             `json:"task"`
	Classes                   []string           `json:"classes"`
	Timeframe                 string             `json:"timeframe"`
	InputTimeframe            string             `json:"input_timeframe"`
	HorizonBars               int                `json:"horizon_bars"`
	PredictionHorizonBars     int                `json:"prediction_horizon_bars"`
	FeatureSchemaVersion      string             `json:"feature_schema_version"`
	FeatureColumns            []string           `json:"feature_columns"`
	FeatureOrder              []string           `json:"feature_order"`
	NormalizationArtifactPath string             `json:"normalization_artifact_path"`
	Normalization             string             `json:"normalization"`
	ExportFormat              string             `json:"export_format"`
	ModelArtifactPath         string             `json:"model_artifact_path"`
	ArtifactSHA256            string             `json:"artifact_sha256"`
	Metrics                   map[string]float64 `json:"metrics"`
	DecisionThreshold         float64            `json:"decision_threshold"`
	Threshold                 float64            `json:"threshold"`
	Calibration               string             `json:"calibration"`
	CreatedAt                 time.Time          `json:"created_at"`
	SourceDatasetVersion      string             `json:"source_dataset_version"`
	InputWindowBars           int                `json:"input_window_bars"`
	InputTensorShape          []int              `json:"input_tensor_shape"`
	RuntimeStatus             string             `json:"runtime_status,omitempty"`
}

type SignalClassProbabilities struct {
	Up      float64 `json:"up"`
	Down    float64 `json:"down"`
	NoTrade float64 `json:"no_trade"`
}

type SignalRun struct {
	ID                 int64
	AssetID            string
	UserID             int64
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

type SignalOutcome struct {
	ID              int64
	SignalRunID     int64
	AssetID         string
	MaturedAt       time.Time
	EntryPrice      float64
	ExitPrice       float64
	RawReturnPct    float64
	ActionReturnPct float64
	IsHit           bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SignalEvent struct {
	ID             int64          `json:"id"`
	UserID         int64          `json:"user_id,omitempty"`
	SignalRunID    *int64         `json:"signal_run_id,omitempty"`
	EventType      string         `json:"event_type"`
	ModelVersion   string         `json:"model_version"`
	Ticker         string         `json:"ticker,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

type NotificationRule struct {
	ID                 int64      `json:"id"`
	UserID             int64      `json:"user_id"`
	Ticker             string     `json:"ticker,omitempty"`
	EventType          string     `json:"event_type"` // e.g., "price", "indicator", "signal"
	TargetIndicator    string     `json:"target_indicator,omitempty"` // e.g., "RSI", "EMA"
	Operator           string     `json:"operator,omitempty"`         // e.g., ">", "<", "cross_up", "cross_down"
	Threshold          float64    `json:"threshold,omitempty"`
	SecondaryThreshold float64    `json:"secondary_threshold,omitempty"` // for channels/ranges
	Severity           string     `json:"severity"`
	Direction          string     `json:"direction,omitempty"`
	ModelVersion       string     `json:"model_version,omitempty"`
	IsEnabled          bool       `json:"is_enabled"`
	CooldownMinutes    int        `json:"cooldown_minutes"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	TriggerMode        string     `json:"trigger_mode"` // "once", "always", "not_more_than_n", "once_per_day"
	DeliveryChannels   []string   `json:"delivery_channels"` // "app", "email", "telegram", "browser"
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type NotificationEvent struct {
	ID               int64          `json:"id"`
	RuleID           int64          `json:"rule_id"`
	SignalEventID    *int64         `json:"signal_event_id,omitempty"`
	EventType        string         `json:"event_type"`
	Severity         string         `json:"severity"`
	ModelVersion     string         `json:"model_version"`
	Ticker           string         `json:"ticker,omitempty"`
	Message          string         `json:"message"`
	Payload          map[string]any `json:"payload,omitempty"`
	DeliveryStatus   string         `json:"delivery_status"`
	DeliveryAttempts int            `json:"delivery_attempts"`
	CreatedAt        time.Time      `json:"created_at"`
}

const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

const (
	SignalEventClassificationSuccess      = "classification_success"
	SignalEventInferenceBlockedByRuntime  = "inference_blocked_by_runtime"
	SignalEventDecisionThresholdTriggered = "decision_threshold_triggered"
	SignalEventPolicyPromotion            = "policy_promotion"
	SignalEventPolicyRollback             = "policy_rollback"
	SignalEventModelActivated             = "model_activated"
)

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
	ModelVersion      string
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

type PolicyPromotionLog struct {
	ID                    int64     `json:"id"`
	PolicyValidationRunID int64     `json:"policy_validation_run_id"`
	Actor                 string    `json:"actor"`
	PreviousState         string    `json:"previous_state"`
	NextState             string    `json:"next_state"`
	Blockers              []string  `json:"blockers,omitempty"`
	Notes                 string    `json:"notes,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
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

type PolicyOutcomeSummary struct {
	ValidationRunID        int64
	DecisionState          string
	ModelName              string
	CalibrationMethod      string
	Threshold              float64
	DatasetVersion         string
	SignalsTotal           int
	ActionableSignals      int
	MaturedSignals         int
	PendingSignals         int
	OverduePendingSignals  int
	HitSignals             int
	MissSignals            int
	RealizedPrecision      float64
	AverageReturnPct       float64
	AverageActionReturnPct float64
	LastSignalAt           time.Time
	FirstMaturedAt         time.Time
	LastMaturedAt          time.Time
	CanPromote             bool
	PromotionBlockers      []PolicyPromotionBlocker
}

type PolicyOutcomeRecord struct {
	SignalRunID       int64
	AssetID           string
	AsOfTime          time.Time
	SignalState       string
	SignalDirection   string
	SignalProbability float64
	Timeframe         string
	HorizonBars       int
	MaturedAt         time.Time
	EntryPrice        float64
	ExitPrice         float64
	RawReturnPct      float64
	ActionReturnPct   float64
	IsHit             bool
}

type PolicyPromotionBlocker struct {
	Code    string
	Message string
}

type JobRun struct {
	ID           int64
	JobType      string
	Status       string
	StartedAt    time.Time
	FinishedAt   time.Time
	Payload      map[string]any
	ErrorMessage string
}

const (
	ModelStatusActive   = "active"
	ModelStatusInactive = "inactive"
	ModelStatusArchived = "archived"

	SignalStateActionable = "actionable"
	SignalStateNoTrade    = "no_trade"

	SignalDirectionUp   = "up"
	SignalDirectionDown = "down"
	SignalDirectionNone = "none"

	PolicyDecisionCandidate  = "candidate"
	PolicyDecisionShadowLive = "shadow_live"
	PolicyDecisionApproved   = "approved"
	PolicyDecisionActive     = "active"
	PolicyDecisionBlocked    = "blocked"
	PolicyDecisionArchived   = "archived"
	PolicyDecisionPromoted   = "promoted" // Legacy alias for Active

	JobStatusRunning   = "running"
	JobStatusSucceeded = "succeeded"
	JobStatusFailed    = "failed"
)
