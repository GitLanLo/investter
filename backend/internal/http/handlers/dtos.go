package handlers

type MLResearchOverviewResponse struct {
	GeneratedAt        string            `json:"generated_at"`
	DatasetManifest    map[string]any    `json:"dataset_manifest,omitempty"`
	ResearchSummary    map[string]any    `json:"research_summary,omitempty"`
	CalibrationSummary map[string]any    `json:"calibration_summary,omitempty"`
	GridMatrix         map[string]any    `json:"grid_matrix,omitempty"`
	SourcePaths        map[string]string `json:"source_paths"`
	Warnings           []string          `json:"warnings"`
}

type MLResearchDocumentDTO struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type MLResearchDocumentsResponse struct {
	GeneratedAt string                  `json:"generated_at"`
	Items       []MLResearchDocumentDTO `json:"items"`
}

type MLProductionPolicyResponse struct {
	GeneratedAt       string                       `json:"generated_at"`
	PolicyStatus      string                       `json:"policy_status"`
	ModelName         string                       `json:"model_name"`
	ModelVersion      string                       `json:"model_version,omitempty"`
	ScenarioName      string                       `json:"scenario_name"`
	CalibrationMethod string                       `json:"calibration_method"`
	Threshold         float64                      `json:"threshold"`
	Timeframe         string                       `json:"timeframe"`
	HorizonBars       int                          `json:"horizon_bars"`
	DatasetVersion    string                       `json:"dataset_version"`
	FeatureSchema     string                       `json:"feature_schema"`
	TrainRows         int                          `json:"train_rows"`
	ValidationRows    int                          `json:"validation_rows"`
	TestRows          int                          `json:"test_rows"`
	Validation        MLProductionPolicyMetricsDTO `json:"validation"`
	Test              MLProductionPolicyMetricsDTO `json:"test"`
	SourcePaths       map[string]string            `json:"source_paths"`
	Warnings          []string                     `json:"warnings"`
}

type AssetDTO struct {
	ID                  string `json:"id"`
	Ticker              string `json:"ticker"`
	Name                string `json:"name"`
	Exchange            string `json:"exchange"`
	Timeframe           string `json:"timeframe"`
	IsActive            bool   `json:"is_active"`
	Figi                string `json:"figi"`
	InstrumentUID       string `json:"instrument_uid"`
	ClassCode           string `json:"class_code"`
	InstrumentType      string `json:"instrument_type"`
	Lot                 int32  `json:"lot"`
	Currency            string `json:"currency"`
	APITradeAvailable   bool   `json:"api_trade_available"`
	First1MinCandleDate string `json:"first_1min_candle_date,omitempty"`
	First1DayCandleDate string `json:"first_1day_candle_date,omitempty"`
	ModelSupported      bool   `json:"model_supported"`
}

type AssetsResponse struct {
	Items []AssetDTO `json:"items"`
}

type InstrumentDTO struct {
	UID                 string `json:"uid"`
	Figi                string `json:"figi"`
	Ticker              string `json:"ticker"`
	ClassCode           string `json:"class_code"`
	Isin                string `json:"isin"`
	Lot                 int32  `json:"lot"`
	Currency            string `json:"currency"`
	Name                string `json:"name"`
	Exchange            string `json:"exchange"`
	InstrumentType      string `json:"instrument_type"`
	APITradeAvailable   bool   `json:"api_trade_available"`
	First1MinCandleDate string `json:"first_1min_candle_date,omitempty"`
	First1DayCandleDate string `json:"first_1day_candle_date,omitempty"`
}

type CandleDTO struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
}

type FactorDTO struct {
	Factor    string  `json:"factor"`
	Timestamp string  `json:"timestamp"`
	Close     float64 `json:"close"`
}

type WatchlistResponse struct {
	WatchlistID int64              `json:"watchlist_id"`
	Name        string             `json:"name"`
	Items       []WatchlistItemDTO `json:"items"`
}

type WatchlistItemDTO struct {
	AssetID  string    `json:"asset_id"`
	Position int       `json:"position"`
	Asset    *AssetDTO `json:"asset,omitempty"`
}

type WatchlistUpsertRequest struct {
	AssetID       string `json:"asset_id"`
	InstrumentUID string `json:"instrument_uid"`
	Position      int    `json:"position"`
}

type AnalysisRunRequest struct {
	AssetID      string `json:"asset_id"`
	AsOfTime     string `json:"as_of_time"`
	ModelVersion string `json:"model_version"`
	Timeframe    string `json:"timeframe"`
}

type SignalsResponse struct {
	Items []SignalDTO `json:"items"`
}

type SignalDTO struct {
	ID                 *int64             `json:"id,omitempty"`
	AssetID            string             `json:"asset_id"`
	AsOfTime           string             `json:"as_of_time"`
	SignalState        string             `json:"signal_state"`
	SignalDirection    string             `json:"signal_direction"`
	SignalProbability  float64            `json:"signal_probability"`
	ClassProbabilities map[string]float64 `json:"class_probabilities,omitempty"`
	Threshold          float64            `json:"threshold"`
	Timeframe          string             `json:"timeframe"`
	ModelVersion       string             `json:"model_version"`
	HorizonBars        int                `json:"horizon_bars"`
	Policy             *SignalPolicyDTO   `json:"policy,omitempty"`
}

type SignalPolicyDTO struct {
	PolicyStatus      string  `json:"policy_status"`
	ModelName         string  `json:"model_name"`
	ScenarioName      string  `json:"scenario_name"`
	CalibrationMethod string  `json:"calibration_method"`
	Threshold         float64 `json:"threshold"`
	DatasetVersion    string  `json:"dataset_version"`
}

type SignalEventDTO struct {
	ID             int64          `json:"id"`
	SignalRunID    *int64         `json:"signal_run_id,omitempty"`
	EventType      string         `json:"event_type"`
	ModelVersion   string         `json:"model_version"`
	Ticker         string         `json:"ticker,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type NotificationRuleDTO struct {
	ID                 int64    `json:"id"`
	Ticker             string   `json:"ticker,omitempty"`
	EventType          string   `json:"event_type"`
	TargetIndicator    string   `json:"target_indicator,omitempty"`
	Operator           string   `json:"operator,omitempty"`
	Threshold          float64  `json:"threshold"`
	SecondaryThreshold float64  `json:"secondary_threshold,omitempty"`
	Severity           string   `json:"severity"`
	Direction          string   `json:"direction,omitempty"`
	ModelVersion       string   `json:"model_version,omitempty"`
	IsEnabled          bool     `json:"is_enabled"`
	CooldownMinutes    int      `json:"cooldown_minutes"`
	ExpiresAt          string   `json:"expires_at,omitempty"`
	TriggerMode        string   `json:"trigger_mode"`
	DeliveryChannels   []string `json:"delivery_channels"`
}

type JobRunDTO struct {
	ID           int64          `json:"id"`
	JobType      string         `json:"job_type"`
	Status       string         `json:"status"`
	StartedAt    string         `json:"started_at"`
	FinishedAt   string         `json:"finished_at,omitempty"`
	Payload      map[string]any `json:"payload"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type JobRunsResponse struct {
	Items []JobRunDTO `json:"items"`
}

type JobSchedulerStatusDTO struct {
	Enabled    bool   `json:"enabled"`
	Interval   string `json:"interval"`
	Limit      int    `json:"limit"`
	RunOnStart bool   `json:"run_on_start"`
}

type SchedulerInfoDTO struct {
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Interval   string `json:"interval"`
	Limit      int    `json:"limit,omitempty"`
	RunOnStart bool   `json:"run_on_start,omitempty"`
}

type SchedulersResponse struct {
	Schedulers []SchedulerInfoDTO `json:"schedulers"`
}

type MLModelManifestDTO struct {
	ModelVersion              string             `json:"model_version"`
	ModelType                 string             `json:"model_type"`
	ModelFamily               string             `json:"model_family"`
	Classes                   []string           `json:"classes"`
	Timeframe                 string             `json:"timeframe"`
	HorizonBars               int                `json:"horizon_bars"`
	FeatureSchemaVersion      string             `json:"feature_schema_version"`
	FeatureColumns            []string           `json:"feature_columns"`
	NormalizationArtifactPath string             `json:"normalization_artifact_path"`
	ExportFormat              string             `json:"export_format"`
	ModelArtifactPath         string             `json:"model_artifact_path"`
	ArtifactSHA256            string             `json:"artifact_sha256"`
	Metrics                   map[string]float64 `json:"metrics"`
	DecisionThreshold         float64            `json:"decision_threshold"`
	Calibration               string             `json:"calibration"`
	CreatedAt                 string             `json:"created_at"`
	SourceDatasetVersion      string             `json:"source_dataset_version"`
	InputWindowBars           int                `json:"input_window_bars,omitempty"`
	InputTensorShape          []int              `json:"input_tensor_shape,omitempty"`
	RuntimeStatus             string             `json:"runtime_status"`
}

type MLProductionPolicyMetricsDTO struct {
	ActionableF1  float64 `json:"actionable_f1"`
	Precision     float64 `json:"precision"`
	Coverage      float64 `json:"coverage"`
	ActionableECE float64 `json:"actionable_ece"`
}

type MLPolicyValidationRunDTO struct {
	ID                int64                        `json:"id"`
	PolicyStatus      string                       `json:"policy_status"`
	ModelName         string                       `json:"model_name"`
	ModelVersion      string                       `json:"model_version"`
	ScenarioName      string                       `json:"scenario_name"`
	CalibrationMethod string                       `json:"calibration_method"`
	Threshold         float64                      `json:"threshold"`
	DatasetVersion    string                       `json:"dataset_version"`
	Validation        MLProductionPolicyMetricsDTO `json:"validation"`
	Test              MLProductionPolicyMetricsDTO `json:"test"`
	DecisionState     string                       `json:"decision_state"`
	Notes             string                       `json:"notes"`
	CreatedAt         string                       `json:"created_at"`
}

type MLPolicyValidationRunsResponse struct {
	Items []MLPolicyValidationRunDTO `json:"items"`
}

type PolicyPromotionBlockerDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MLPolicyShadowSummaryDTO struct {
	ValidationRunID   int64   `json:"validation_run_id"`
	DecisionState     string  `json:"decision_state"`
	ModelName         string  `json:"model_name"`
	CalibrationMethod string  `json:"calibration_method"`
	Threshold         float64 `json:"threshold"`
	DatasetVersion    string  `json:"dataset_version"`
	SignalsTotal      int     `json:"signals_total"`
	ActionableSignals int     `json:"actionable_signals"`
	NoTradeSignals    int     `json:"no_trade_signals"`
	UpSignals         int     `json:"up_signals"`
	DownSignals       int     `json:"down_signals"`
	ObservedCoverage  float64 `json:"observed_coverage"`
	FirstSignalAt     string  `json:"first_signal_at,omitempty"`
	LastSignalAt      string  `json:"last_signal_at,omitempty"`
}

type MLPolicyOutcomeSummaryDTO struct {
	ValidationRunID        int64                       `json:"validation_run_id"`
	DecisionState          string                      `json:"decision_state"`
	ModelName              string                      `json:"model_name"`
	CalibrationMethod      string                      `json:"calibration_method"`
	Threshold              float64                     `json:"threshold"`
	DatasetVersion         string                      `json:"dataset_version"`
	SignalsTotal           int                         `json:"signals_total"`
	ActionableSignals      int                         `json:"actionable_signals"`
	MaturedSignals         int                         `json:"matured_signals"`
	PendingSignals         int                         `json:"pending_signals"`
	OverduePendingSignals  int                         `json:"overdue_pending_signals"`
	HitSignals             int                         `json:"hit_signals"`
	MissSignals            int                         `json:"miss_signals"`
	RealizedPrecision      float64                     `json:"realized_precision"`
	AverageReturnPct       float64                     `json:"average_return_pct"`
	AverageActionReturnPct float64                     `json:"average_action_return_pct"`
	LastSignalAt           string                      `json:"last_signal_at,omitempty"`
	FirstMaturedAt         string                      `json:"first_matured_at,omitempty"`
	LastMaturedAt          string                      `json:"last_matured_at,omitempty"`
	CanPromote             bool                        `json:"can_promote"`
	PromotionBlockers      []PolicyPromotionBlockerDTO `json:"promotion_blockers"`
}

type MLPolicyOutcomeRecordDTO struct {
	SignalRunID       int64   `json:"signal_run_id"`
	AssetID           string  `json:"asset_id"`
	AsOfTime          string  `json:"as_of_time"`
	SignalState       string  `json:"signal_state"`
	SignalDirection   string  `json:"signal_direction"`
	SignalProbability float64 `json:"signal_probability"`
	Timeframe         string  `json:"timeframe"`
	HorizonBars       int     `json:"horizon_bars"`
	MaturedAt         string  `json:"matured_at"`
	EntryPrice        float64 `json:"entry_price"`
	ExitPrice         float64 `json:"exit_price"`
	RawReturnPct      float64 `json:"raw_return_pct"`
	ActionReturnPct   float64 `json:"action_return_pct"`
	IsHit             bool    `json:"is_hit"`
}

type MLPolicyOutcomeHistoryResponse struct {
	Items []MLPolicyOutcomeRecordDTO `json:"items"`
}

type NotificationEventDTO struct {
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
	CreatedAt        string         `json:"created_at"`
}
