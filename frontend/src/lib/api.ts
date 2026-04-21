import { loadMockAssetWorkbench, mockWorkspaceShellData } from "./mock";
import type {
  AssetCard,
  AssetDTO,
  AssetWorkbenchData,
  AssetWorkbenchRequest,
  CalibrationMethodSnapshot,
  CandidateMetrics,
  CandidateSnapshot,
  CandleBar,
  CandleDTO,
  FactorDTO,
  FactorPoint,
  MLOverview,
  ArtifactDocument,
  ResearchOverviewDTO,
  ResearchDocumentsResponseDTO,
  ResearchDocumentDTO,
  ScenarioSnapshot,
  SignalCard,
  SignalDTO,
  WorkspaceShellData,
} from "./types";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";
const artifactPreviewLimit = 24000;

export function formatProbability(value: number | undefined): string {
  if (value === undefined || Number.isNaN(value)) {
    return "n/a";
  }
  return `${Math.round(value * 100)}%`;
}

export function formatMetric(value: number | undefined, digits = 3): string {
  if (value === undefined || Number.isNaN(value)) {
    return "n/a";
  }
  return value.toFixed(digits);
}

export async function loadWorkspaceShell(): Promise<WorkspaceShellData> {
  try {
    const [documents, assets, signals, overview] = await Promise.all([
      fetchJson<ResearchDocumentsResponseDTO>("/ml/research/documents"),
      fetchJson<{ items: AssetDTO[] }>("/assets"),
      fetchJson<{ items: SignalDTO[] }>("/signals/latest?limit=12"),
      fetchJson<ResearchOverviewDTO>("/ml/research/overview"),
    ]);

    return {
      generatedFrom: "api",
      assets: assets.items.map(mapAsset),
      latestSignals: signals.items.map(mapSignal),
      mlOverview: mapResearchOverview(overview),
      artifactDocuments: documents.items.map(mapResearchDocument),
    };
  } catch {
    return mockWorkspaceShellData;
  }
}

export async function loadAssetWorkbench(
  assetId: string,
  request: AssetWorkbenchRequest = {},
): Promise<AssetWorkbenchData> {
  try {
    const effectiveRequest = resolveWorkbenchRequest(request);
    const candleParams = new URLSearchParams();
    if (effectiveRequest.from) {
      candleParams.set("from", effectiveRequest.from);
    }
    if (effectiveRequest.to) {
      candleParams.set("to", effectiveRequest.to);
    }
    candleParams.set("limit", String(effectiveRequest.limit ?? 20000));

    const candlesResponse = await fetchJson<{ items: CandleDTO[] }>(
      `/assets/${assetId}/candles?${candleParams.toString()}`,
    );
    const candles = candlesResponse.items.map(mapCandle);

    const firstCandle = candles[0];
    const lastCandle = candles[candles.length - 1];
    const factorFrom = firstCandle?.timestamp ?? effectiveRequest.from;
    const factorTo = lastCandle?.timestamp ?? effectiveRequest.to;
    const factorQuery =
      factorFrom && factorTo
        ? `?from=${encodeURIComponent(factorFrom)}&to=${encodeURIComponent(factorTo)}`
        : "";
    const factorsResponse = await fetchJson<{ items: FactorDTO[] }>(
      `/assets/${assetId}/factors${factorQuery}`,
    );
    const signalHistoryResponse = await fetchJson<{ items: SignalDTO[] }>(
      `/assets/${assetId}/signals?limit=24`,
    );

    return {
      generatedFrom: "api",
      candles,
      factors: factorsResponse.items.map(mapFactor),
      signalHistory: signalHistoryResponse.items.map(mapSignal),
    };
  } catch {
    return loadMockAssetWorkbench(assetId, request);
  }
}

export async function runAnalysisForAsset(assetId: string, timeframe: string): Promise<SignalCard> {
  const response = await fetch(`${apiBaseUrl}/analysis/run`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      asset_id: assetId,
      model_version: "active",
      timeframe,
    }),
  });

  if (!response.ok) {
    throw new Error(`analysis run failed: ${response.status}`);
  }

  return mapSignal((await response.json()) as SignalDTO);
}

function mapAsset(dto: AssetDTO): AssetCard {
  return {
    id: dto.id,
    ticker: dto.ticker,
    name: dto.name,
    venue: dto.exchange,
    timeframe: dto.timeframe,
    active: dto.is_active,
  };
}

function mapSignal(dto: SignalDTO): SignalCard {
  const stableId =
    dto.id !== undefined ? String(dto.id) : `${dto.asset_id}:${dto.as_of_time}:${dto.model_version}`;
  return {
    id: stableId,
    assetId: dto.asset_id,
    direction: dto.signal_direction,
    state: dto.signal_state,
    probability: dto.signal_probability,
    threshold: dto.threshold,
    asOfTime: dto.as_of_time,
    timeframe: dto.timeframe,
    modelVersion: dto.model_version,
    horizonBars: dto.horizon_bars,
    classProbabilities: dto.class_probabilities,
  };
}

function mapCandle(dto: CandleDTO): CandleBar {
  return {
    timestamp: dto.timestamp,
    open: dto.open,
    high: dto.high,
    low: dto.low,
    close: dto.close,
    volume: dto.volume,
  };
}

function mapFactor(dto: FactorDTO): FactorPoint {
  return {
    factor: dto.factor,
    timestamp: dto.timestamp,
    close: dto.close,
  };
}

function mapResearchDocument(dto: ResearchDocumentDTO): ArtifactDocument {
  const content = dto.content ?? "";
  return {
    key: dto.key,
    title: dto.title,
    path: dto.path,
    contentType:
      dto.content_type === "markdown" || dto.content_type === "json" ? dto.content_type : "text",
    content:
      content.length > artifactPreviewLimit
        ? `${content.slice(0, artifactPreviewLimit)}\n\n... truncated in UI preview (${content.length} chars total)`
        : content,
  };
}

async function fetchJson<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`);
  if (!response.ok) {
    throw new Error(`request failed for ${path}: ${response.status}`);
  }
  return response.json() as Promise<T>;
}

function resolveWorkbenchRequest(request: AssetWorkbenchRequest): Required<AssetWorkbenchRequest> {
  const fallbackFrom = new Date();
  fallbackFrom.setUTCDate(fallbackFrom.getUTCDate() - 60);

  return {
    from: request.from ?? fallbackFrom.toISOString(),
    to: request.to ?? "",
    limit: request.limit ?? 20000,
  };
}

function mapResearchOverview(dto: ResearchOverviewDTO): MLOverview {
  return {
    generatedAt: dto.generated_at,
    warnings: dto.warnings ?? [],
    sourcePaths: dto.source_paths ?? {},
    dataset: mapDataset(dto.dataset_manifest),
    research: mapResearchSummary(dto.research_summary),
    calibration: mapCalibrationSummary(dto.calibration_summary),
  };
}

function mapDataset(raw: Record<string, unknown> | undefined) {
  if (!raw) {
    return undefined;
  }

  return {
    datasetVersion: stringValue(raw.dataset_version) ?? "unknown_dataset",
    featureSchemaVersion: stringValue(raw.feature_schema_version) ?? "unknown_schema",
    timeframe: stringValue(raw.timeframe) ?? "n/a",
    horizonBars: numberValue(raw.horizon_bars) ?? 0,
    tickers: stringArray(raw.tickers),
    trainRows: numberValue(recordValue(raw.train_range)?.rows) ?? 0,
    valRows: numberValue(recordValue(raw.val_range)?.rows) ?? 0,
    testRows: numberValue(recordValue(raw.test_range)?.rows) ?? 0,
  };
}

function mapResearchSummary(raw: Record<string, unknown> | undefined) {
  if (!raw) {
    return undefined;
  }

  return {
    researchCandidate: mapCandidate(recordValue(raw.research_candidate), "Research candidate"),
    productionCandidate: mapCandidate(recordValue(raw.production_candidate), "Production candidate"),
    scenarios: mapScenarioSnapshots(recordValue(raw.scenarios)),
  };
}

function mapCalibrationSummary(raw: Record<string, unknown> | undefined) {
  if (!raw) {
    return undefined;
  }

  return {
    researchCandidate: mapCandidate(recordValue(raw.research_candidate), "Calibration research"),
    productionCandidate: mapCandidate(recordValue(raw.production_candidate), "Calibration production"),
    methods: mapCalibrationMethods(recordValue(raw.methods)),
  };
}

function mapCandidate(raw: Record<string, unknown> | undefined, label: string): CandidateSnapshot | undefined {
  if (!raw) {
    return undefined;
  }

  return {
    label,
    modelName: stringValue(raw.model_name),
    scenarioName: stringValue(raw.scenario_name),
    method: stringValue(raw.method),
    selectionMode: stringValue(raw.selection_mode),
    selectedThreshold: numberValue(raw.selected_threshold),
    gatePassed: booleanValue(recordValue(raw.validation_gate)?.passed),
    validation: candidateMetricsFromCandidateBlock(recordValue(raw.validation)),
    test: candidateMetricsFromCandidateBlock(recordValue(raw.test)),
  };
}

function mapScenarioSnapshots(raw: Record<string, unknown> | undefined): ScenarioSnapshot[] {
  if (!raw) {
    return [];
  }

  return Object.entries(raw)
    .map(([name, value]) => {
      const scenario = recordValue(value);
      const bestModelName = stringValue(scenario?.best_model);
      const bestModel = bestModelName ? recordValue(recordValue(scenario?.models)?.[bestModelName]) : undefined;

      return {
        name,
        featureCount: numberValue(scenario?.feature_count) ?? 0,
        modelName: bestModelName,
        selectedThreshold: numberValue(bestModel?.selected_threshold) ?? numberValue(bestModel?.decision_threshold),
        validation: candidateMetricsFromDetailedSplit(recordValue(bestModel?.validation)),
        test: candidateMetricsFromDetailedSplit(recordValue(bestModel?.test)),
      };
    })
    .sort((left, right) => (right.validation.actionableF1 ?? 0) - (left.validation.actionableF1 ?? 0));
}

function mapCalibrationMethods(raw: Record<string, unknown> | undefined): CalibrationMethodSnapshot[] {
  if (!raw) {
    return [];
  }

  return Object.entries(raw)
    .map(([method, value]) => {
      const item = recordValue(value);
      return {
        method,
        available: booleanValue(item?.available) ?? false,
        selectedThreshold: numberValue(item?.selected_threshold),
        validation: candidateMetricsFromDetailedSplit(recordValue(item?.validation)),
        test: candidateMetricsFromDetailedSplit(recordValue(item?.test)),
      };
    })
    .sort((left, right) => (right.validation.actionableF1 ?? 0) - (left.validation.actionableF1 ?? 0));
}

function candidateMetricsFromCandidateBlock(raw: Record<string, unknown> | undefined): CandidateMetrics {
  if (!raw) {
    return {};
  }

  return {
    macroF1: numberValue(raw.macro_f1),
    balancedAccuracy: numberValue(raw.balanced_accuracy),
    actionableF1: numberValue(raw.actionable_f1),
    precision: numberValue(raw.precision_actionable_signal),
    coverage: numberValue(raw.signal_coverage),
    actionableEce: numberValue(raw.actionable_expected_calibration_error),
  };
}

function candidateMetricsFromDetailedSplit(raw: Record<string, unknown> | undefined): CandidateMetrics {
  if (!raw) {
    return {};
  }

  const signalMetrics = recordValue(raw.signal_metrics);
  const probabilityMetrics = recordValue(raw.probability_metrics);
  const classificationMetrics = recordValue(raw.metrics);

  return {
    macroF1: numberValue(classificationMetrics?.macro_f1) ?? numberValue(raw.macro_f1),
    balancedAccuracy: numberValue(classificationMetrics?.balanced_accuracy) ?? numberValue(raw.balanced_accuracy),
    actionableF1: numberValue(signalMetrics?.actionable_f1) ?? numberValue(raw.actionable_f1),
    precision:
      numberValue(signalMetrics?.precision_actionable_signal) ?? numberValue(raw.precision_actionable_signal),
    coverage: numberValue(signalMetrics?.signal_coverage) ?? numberValue(raw.signal_coverage),
    actionableEce:
      numberValue(probabilityMetrics?.actionable_expected_calibration_error) ??
      numberValue(raw.actionable_expected_calibration_error),
  };
}

function recordValue(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function numberValue(value: unknown): number | undefined {
  return typeof value === "number" ? value : undefined;
}

function booleanValue(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined;
}

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string");
}
