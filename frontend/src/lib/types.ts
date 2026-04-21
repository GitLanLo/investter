export type AssetDTO = {
  id: string;
  ticker: string;
  name: string;
  exchange: string;
  timeframe: string;
  is_active: boolean;
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

export type AssetCard = {
  id: string;
  ticker: string;
  name: string;
  venue: string;
  timeframe: string;
  active: boolean;
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
