export type AssetDTO = {
  id: string;
  ticker: string;
  name: string;
  exchange: string;
  timeframe: string;
  is_active: boolean;
};

export type InstrumentDTO = {
  uid: string;
  figi: string;
  ticker: string;
  class_code: string;
  isin: string;
  lot: number;
  currency: string;
  name: string;
  exchange: string;
  instrument_type: string;
  api_trade_available: boolean;
  first_1min_candle_date?: string;
  first_1day_candle_date?: string;
};

export type SignalDTO = {
  id?: number | string;
  asset_id: string;
  model_version: string;
  as_of_time: string;
  signal_state: string;
  signal_direction: string;
  signal_probability: number;
  class_probabilities: Record<string, number>;
  threshold: number;
  timeframe: string;
  horizon_bars: number;
  created_at?: string;
  policy?: SignalPolicyDTO;
};

export type SignalPolicyDTO = {
  policy_status: string;
  model_name: string;
  scenario_name: string;
  calibration_method: string;
  threshold: number;
  dataset_version: string;
};

export type CandleDTO = {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
};

export type FactorDTO = {
  factor: string;
  timestamp: string;
  close: number;
};

export type ResearchOverviewDTO = {
  generated_at: string;
  dataset_manifest?: Record<string, unknown>;
  research_summary?: Record<string, unknown>;
  calibration_summary?: Record<string, unknown>;
  source_paths?: Record<string, string>;
  warnings?: string[];
};

export type ResearchDocumentDTO = {
  key: string;
  title: string;
  path: string;
  content_type: string;
  content: string;
};

export type ResearchDocumentsResponseDTO = {
  generated_at: string;
  items: ResearchDocumentDTO[];
};

export type ProductionPolicyDTO = {
  generated_at: string;
  policy_status: string;
  model_name: string;
  scenario_name: string;
  calibration_method: string;
  threshold: number;
  timeframe: string;
  horizon_bars: number;
  dataset_version: string;
  feature_schema: string;
  train_rows: number;
  validation_rows: number;
  test_rows: number;
  validation: ProductionPolicyMetricsDTO;
  test: ProductionPolicyMetricsDTO;
  source_paths?: Record<string, string>;
  warnings?: string[];
};

export type ProductionPolicyMetricsDTO = {
  actionable_f1: number;
  precision: number;
  coverage: number;
  actionable_ece: number;
};

export type PolicyValidationRunDTO = {
  id: number;
  policy_status: string;
  model_name: string;
  scenario_name: string;
  calibration_method: string;
  threshold: number;
  dataset_version: string;
  validation: ProductionPolicyMetricsDTO;
  test: ProductionPolicyMetricsDTO;
  decision_state: string;
  notes: string;
  created_at: string;
};

export type PolicyShadowSummaryDTO = {
  validation_run_id: number;
  decision_state: string;
  model_name: string;
  calibration_method: string;
  threshold: number;
  dataset_version: string;
  signals_total: number;
  actionable_signals: number;
  no_trade_signals: number;
  up_signals: number;
  down_signals: number;
  observed_coverage: number;
  first_signal_at?: string;
  last_signal_at?: string;
};

export type PolicyOutcomeSummaryDTO = {
  validation_run_id: number;
  decision_state: string;
  model_name: string;
  calibration_method: string;
  threshold: number;
  dataset_version: string;
  signals_total: number;
  actionable_signals: number;
  matured_signals: number;
  pending_signals: number;
  overdue_pending_signals: number;
  hit_signals: number;
  miss_signals: number;
  realized_precision: number;
  average_return_pct: number;
  average_action_return_pct: number;
  last_signal_at?: string;
  first_matured_at?: string;
  last_matured_at?: string;
  can_promote: boolean;
  promotion_blockers?: PolicyPromotionBlockerDTO[];
};

export type PolicyOutcomeRecordDTO = {
  signal_run_id: number;
  asset_id: string;
  as_of_time: string;
  signal_state: string;
  signal_direction: string;
  signal_probability: number;
  timeframe: string;
  horizon_bars: number;
  matured_at: string;
  entry_price: number;
  exit_price: number;
  raw_return_pct: number;
  action_return_pct: number;
  is_hit: boolean;
};

export type PolicyPromotionBlockerDTO = {
  code: string;
  message: string;
};

export type JobRunDTO = {
  id: number;
  job_type: string;
  status: string;
  started_at: string;
  finished_at?: string;
  payload: Record<string, unknown>;
  error_message?: string;
};

export type JobSchedulerStatusDTO = {
  enabled: boolean;
  interval: string;
  limit: number;
  run_on_start: boolean;
};

export type AssetCard = {
  id: string;
  ticker: string;
  name: string;
  venue: string;
  timeframe: string;
  active: boolean;
};

export type InstrumentCard = {
  uid: string;
  figi: string;
  ticker: string;
  classCode: string;
  isin: string;
  lot: number;
  currency: string;
  name: string;
  exchange: string;
  instrumentType: string;
  apiTradeAvailable: boolean;
  first1MinCandleDate?: string;
  first1DayCandleDate?: string;
};

export type SignalCard = {
  id: string;
  assetId: string;
  direction: string;
  state: string;
  probability: number;
  threshold: number;
  asOfTime: string;
  timeframe: string;
  modelVersion: string;
  horizonBars: number;
  classProbabilities: Record<string, number>;
  policy?: SignalPolicySnapshot;
};

export type SignalPolicySnapshot = {
  policyStatus: string;
  modelName: string;
  scenarioName: string;
  calibrationMethod: string;
  threshold: number;
  datasetVersion: string;
};

export type CandleBar = {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
};

export type FactorPoint = {
  factor: string;
  timestamp: string;
  close: number;
};

export type CandidateMetrics = {
  macroF1?: number;
  balancedAccuracy?: number;
  actionableF1?: number;
  precision?: number;
  coverage?: number;
  actionableEce?: number;
};

export type CandidateSnapshot = {
  label: string;
  modelName?: string;
  scenarioName?: string;
  method?: string;
  selectionMode?: string;
  selectedThreshold?: number;
  gatePassed?: boolean;
  validation: CandidateMetrics;
  test: CandidateMetrics;
};

export type ScenarioSnapshot = {
  name: string;
  featureCount: number;
  modelName?: string;
  selectedThreshold?: number;
  validation: CandidateMetrics;
  test: CandidateMetrics;
};

export type CalibrationMethodSnapshot = {
  method: string;
  available: boolean;
  selectedThreshold?: number;
  validation: CandidateMetrics;
  test: CandidateMetrics;
};

export type DatasetSnapshot = {
  datasetVersion: string;
  featureSchemaVersion: string;
  timeframe: string;
  horizonBars: number;
  tickers: string[];
  trainRows: number;
  valRows: number;
  testRows: number;
};

export type MLOverview = {
  generatedAt?: string;
  warnings: string[];
  sourcePaths: Record<string, string>;
  dataset?: DatasetSnapshot;
  research?: {
    researchCandidate?: CandidateSnapshot;
    productionCandidate?: CandidateSnapshot;
    scenarios: ScenarioSnapshot[];
  };
  calibration?: {
    researchCandidate?: CandidateSnapshot;
    productionCandidate?: CandidateSnapshot;
    methods: CalibrationMethodSnapshot[];
  };
};

export type ProductionPolicySnapshot = {
  generatedAt: string;
  status: string;
  modelName: string;
  scenarioName: string;
  calibrationMethod: string;
  threshold: number;
  timeframe: string;
  horizonBars: number;
  datasetVersion: string;
  featureSchema: string;
  trainRows: number;
  validationRows: number;
  testRows: number;
  validation: CandidateMetrics;
  test: CandidateMetrics;
  sourcePaths: Record<string, string>;
  warnings: string[];
};

export type PolicyValidationRun = {
  id: number;
  policyStatus: string;
  modelName: string;
  scenarioName: string;
  calibrationMethod: string;
  threshold: number;
  datasetVersion: string;
  validation: CandidateMetrics;
  test: CandidateMetrics;
  decisionState: string;
  notes: string;
  createdAt: string;
};

export type PolicyShadowSummary = {
  validationRunId: number;
  decisionState: string;
  modelName: string;
  calibrationMethod: string;
  threshold: number;
  datasetVersion: string;
  signalsTotal: number;
  actionableSignals: number;
  noTradeSignals: number;
  upSignals: number;
  downSignals: number;
  observedCoverage: number;
  firstSignalAt?: string;
  lastSignalAt?: string;
};

export type PolicyOutcomeSummary = {
  validationRunId: number;
  decisionState: string;
  modelName: string;
  calibrationMethod: string;
  threshold: number;
  datasetVersion: string;
  signalsTotal: number;
  actionableSignals: number;
  maturedSignals: number;
  pendingSignals: number;
  overduePendingSignals: number;
  hitSignals: number;
  missSignals: number;
  realizedPrecision: number;
  averageReturnPct: number;
  averageActionReturnPct: number;
  lastSignalAt?: string;
  firstMaturedAt?: string;
  lastMaturedAt?: string;
  canPromote: boolean;
  promotionBlockers: PolicyPromotionBlocker[];
};

export type PolicyPromotionBlocker = {
  code: string;
  message: string;
};

export type PolicyOutcomeRecord = {
  signalRunId: number;
  assetId: string;
  asOfTime: string;
  signalState: string;
  signalDirection: string;
  signalProbability: number;
  timeframe: string;
  horizonBars: number;
  maturedAt: string;
  entryPrice: number;
  exitPrice: number;
  rawReturnPct: number;
  actionReturnPct: number;
  isHit: boolean;
};

export type JobRun = {
  id: number;
  jobType: string;
  status: string;
  startedAt: string;
  finishedAt?: string;
  payload: Record<string, unknown>;
  errorMessage?: string;
};

export type JobSchedulerStatus = {
  enabled: boolean;
  interval: string;
  limit: number;
  runOnStart: boolean;
};

export type ArtifactDocument = {
  key: string;
  title: string;
  path: string;
  contentType: "json" | "markdown" | "text";
  content: string;
};

export type WorkspaceShellData = {
  assets: AssetCard[];
  latestSignals: SignalCard[];
  mlOverview: MLOverview;
  productionPolicy: ProductionPolicySnapshot;
  policyValidationRuns: PolicyValidationRun[];
  policyShadowSummary?: PolicyShadowSummary;
  policyOutcomeSummary?: PolicyOutcomeSummary;
  policyOutcomeHistory: PolicyOutcomeRecord[];
  jobScheduler?: JobSchedulerStatus;
  jobRuns: JobRun[];
  artifactDocuments: ArtifactDocument[];
  generatedFrom: "api" | "mock";
};

export type AssetWorkbenchData = {
  candles: CandleBar[];
  factors: FactorPoint[];
  signalHistory: SignalCard[];
  generatedFrom: "api" | "mock";
};

export type AssetWorkbenchRequest = {
  from?: string;
  to?: string;
  limit?: number;
};
