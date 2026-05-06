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
  InstrumentCard,
  InstrumentDTO,
  MLOverview,
  ArtifactDocument,
  PolicyOutcomeSummary,
  PolicyOutcomeSummaryDTO,
  JobRun,
  JobRunDTO,
  JobSchedulerStatus,
  JobSchedulerStatusDTO,
  PolicyOutcomeRecord,
  PolicyOutcomeRecordDTO,
  PolicyValidationRun,
  PolicyValidationRunDTO,
  PolicyShadowSummary,
  PolicyShadowSummaryDTO,
  ProductionPolicyDTO,
  ProductionPolicySnapshot,
  ResearchOverviewDTO,
  ResearchDocumentsResponseDTO,
  ResearchDocumentDTO,
  ScenarioSnapshot,
  SignalCard,
  SignalDTO,
  WorkspaceShellData,
  FreshnessItem,
  FreshnessItemDTO,
  FreshnessSummary,
  FreshnessSummaryDTO,
  SchedulerInfo,
  SchedulersResponse,
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
    const [documents, assets, signals, overview, productionPolicy] = await Promise.all([
      fetchJson<ResearchDocumentsResponseDTO>("/ml/research/documents"),
      fetchJson<{ items: AssetDTO[] }>("/assets"),
      fetchJson<{ items: SignalDTO[] }>("/signals/latest?limit=12"),
      fetchJson<ResearchOverviewDTO>("/ml/research/overview"),
      fetchJson<ProductionPolicyDTO>("/ml/policy/production"),
    ]);
    const [validationRuns, shadowSummary, outcomeSummary, outcomeHistory, jobs, scheduler, freshness, schedulers] = await Promise.all([
      fetchJson<{ items: PolicyValidationRunDTO[] }>("/ml/policy/validation-runs?limit=8").catch(() => ({ items: [] })),
      fetchJson<PolicyShadowSummaryDTO>("/ml/policy/shadow-summary?limit=1000").catch(() => undefined),
      fetchJson<PolicyOutcomeSummaryDTO>("/ml/policy/outcomes?limit=1000").catch(() => undefined),
      fetchJson<{ items: PolicyOutcomeRecordDTO[] }>("/ml/policy/outcomes/history?limit=12").catch(() => ({ items: [] })),
      fetchJson<{ items: JobRunDTO[] }>("/jobs/runs?limit=12").catch(() => ({ items: [] })),
      fetchJson<JobSchedulerStatusDTO>("/jobs/scheduler").catch(() => undefined),
      fetchJson<FreshnessSummaryDTO>("/watchlist/freshness").catch(() => undefined),
      fetchJson<SchedulersResponse>("/jobs/schedulers").catch(() => undefined),
    ]);

    return {
      generatedFrom: "api",
      assets: assets.items.map(mapAsset),
      latestSignals: signals.items.map(mapSignal),
      mlOverview: mapResearchOverview(overview),
      productionPolicy: mapProductionPolicy(productionPolicy),
      policyValidationRuns: validationRuns.items.map(mapPolicyValidationRun),
      policyShadowSummary: shadowSummary ? mapPolicyShadowSummary(shadowSummary) : undefined,
      policyOutcomeSummary: outcomeSummary ? mapPolicyOutcomeSummary(outcomeSummary) : undefined,
      policyOutcomeHistory: outcomeHistory.items.map(mapPolicyOutcomeRecord),
      jobScheduler: scheduler ? mapJobSchedulerStatus(scheduler) : undefined,
      jobRuns: jobs.items.map(mapJobRun),
      artifactDocuments: documents.items.map(mapResearchDocument),
      freshness: freshness ? mapFreshnessSummary(freshness) : undefined,
      schedulers: schedulers?.schedulers ?? [],
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

export async function createPolicyValidationRun(notes: string): Promise<PolicyValidationRun> {
  const response = await fetch(`${apiBaseUrl}/ml/policy/validation-runs`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ notes }),
  });

  if (!response.ok) {
    throw new Error(`policy validation run failed: ${response.status}`);
  }

  return mapPolicyValidationRun((await response.json()) as PolicyValidationRunDTO);
}

export async function updatePolicyValidationRun(
  id: number,
  decisionState: string,
  notes: string,
): Promise<PolicyValidationRun> {
  const response = await fetch(`${apiBaseUrl}/ml/policy/validation-runs/${id}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      decision_state: decisionState,
      notes,
    }),
  });

  if (!response.ok) {
    throw new Error(`policy validation update failed: ${response.status}`);
  }

  return mapPolicyValidationRun((await response.json()) as PolicyValidationRunDTO);
}

export async function runOutcomeMaterializationJob(): Promise<JobRun> {
  const response = await fetch(`${apiBaseUrl}/jobs/outcomes/materialize?limit=1000`, {
    method: "POST",
  });

  if (!response.ok) {
    throw new Error(`outcome job failed: ${response.status}`);
  }

  return mapJobRun((await response.json()) as JobRunDTO);
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
    policy: dto.policy
      ? {
          policyStatus: dto.policy.policy_status,
          modelName: dto.policy.model_name,
          scenarioName: dto.policy.scenario_name,
          calibrationMethod: dto.policy.calibration_method,
          threshold: dto.policy.threshold,
          datasetVersion: dto.policy.dataset_version,
        }
      : undefined,
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

function mapProductionPolicy(dto: ProductionPolicyDTO): ProductionPolicySnapshot {
  return {
    generatedAt: dto.generated_at,
    status: dto.policy_status,
    modelName: dto.model_name,
    scenarioName: dto.scenario_name,
    calibrationMethod: dto.calibration_method,
    threshold: dto.threshold,
    timeframe: dto.timeframe,
    horizonBars: dto.horizon_bars,
    datasetVersion: dto.dataset_version,
    featureSchema: dto.feature_schema,
    trainRows: dto.train_rows,
    validationRows: dto.validation_rows,
    testRows: dto.test_rows,
    validation: mapProductionPolicyMetrics(dto.validation),
    test: mapProductionPolicyMetrics(dto.test),
    sourcePaths: dto.source_paths ?? {},
    warnings: dto.warnings ?? [],
  };
}

function mapProductionPolicyMetrics(raw: ProductionPolicyDTO["validation"]) {
  return {
    actionableF1: raw.actionable_f1,
    precision: raw.precision,
    coverage: raw.coverage,
    actionableEce: raw.actionable_ece,
  };
}

function mapPolicyValidationRun(dto: PolicyValidationRunDTO): PolicyValidationRun {
  return {
    id: dto.id,
    policyStatus: dto.policy_status,
    modelName: dto.model_name,
    scenarioName: dto.scenario_name,
    calibrationMethod: dto.calibration_method,
    threshold: dto.threshold,
    datasetVersion: dto.dataset_version,
    validation: mapProductionPolicyMetrics(dto.validation),
    test: mapProductionPolicyMetrics(dto.test),
    decisionState: dto.decision_state,
    notes: dto.notes,
    createdAt: dto.created_at,
  };
}

function mapPolicyShadowSummary(dto: PolicyShadowSummaryDTO): PolicyShadowSummary {
  return {
    validationRunId: dto.validation_run_id,
    decisionState: dto.decision_state,
    modelName: dto.model_name,
    calibrationMethod: dto.calibration_method,
    threshold: dto.threshold,
    datasetVersion: dto.dataset_version,
    signalsTotal: dto.signals_total,
    actionableSignals: dto.actionable_signals,
    noTradeSignals: dto.no_trade_signals,
    upSignals: dto.up_signals,
    downSignals: dto.down_signals,
    observedCoverage: dto.observed_coverage,
    firstSignalAt: dto.first_signal_at,
    lastSignalAt: dto.last_signal_at,
  };
}

function mapPolicyOutcomeSummary(dto: PolicyOutcomeSummaryDTO): PolicyOutcomeSummary {
  return {
    validationRunId: dto.validation_run_id,
    decisionState: dto.decision_state,
    modelName: dto.model_name,
    calibrationMethod: dto.calibration_method,
    threshold: dto.threshold,
    datasetVersion: dto.dataset_version,
    signalsTotal: dto.signals_total,
    actionableSignals: dto.actionable_signals,
    maturedSignals: dto.matured_signals,
    pendingSignals: dto.pending_signals,
    overduePendingSignals: dto.overdue_pending_signals,
    hitSignals: dto.hit_signals,
    missSignals: dto.miss_signals,
    realizedPrecision: dto.realized_precision,
    averageReturnPct: dto.average_return_pct,
    averageActionReturnPct: dto.average_action_return_pct,
    lastSignalAt: dto.last_signal_at,
    firstMaturedAt: dto.first_matured_at,
    lastMaturedAt: dto.last_matured_at,
    canPromote: dto.can_promote,
    promotionBlockers: (dto.promotion_blockers ?? []).map((item) => ({
      code: item.code,
      message: item.message,
    })),
  };
}

function mapPolicyOutcomeRecord(dto: PolicyOutcomeRecordDTO): PolicyOutcomeRecord {
  return {
    signalRunId: dto.signal_run_id,
    assetId: dto.asset_id,
    asOfTime: dto.as_of_time,
    signalState: dto.signal_state,
    signalDirection: dto.signal_direction,
    signalProbability: dto.signal_probability,
    timeframe: dto.timeframe,
    horizonBars: dto.horizon_bars,
    maturedAt: dto.matured_at,
    entryPrice: dto.entry_price,
    exitPrice: dto.exit_price,
    rawReturnPct: dto.raw_return_pct,
    actionReturnPct: dto.action_return_pct,
    isHit: dto.is_hit,
  };
}

function mapJobRun(dto: JobRunDTO): JobRun {
  return {
    id: dto.id,
    jobType: dto.job_type,
    status: dto.status,
    startedAt: dto.started_at,
    finishedAt: dto.finished_at,
    payload: dto.payload,
    errorMessage: dto.error_message,
  };
}

function mapJobSchedulerStatus(dto: JobSchedulerStatusDTO): JobSchedulerStatus {
  return {
    enabled: dto.enabled,
    interval: dto.interval,
    limit: dto.limit,
    runOnStart: dto.run_on_start,
  };
}

async function fetchJson<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`);
  if (!response.ok) {
    let errMsg = `request failed for ${path}: ${response.status}`;
    try {
      const errorData = await response.json();
      if (errorData && errorData.error && typeof errorData.error === "object") {
        const nestedMsg = errorData.error.message || errorData.error.code;
        if (nestedMsg) errMsg += ` - ${nestedMsg}`;
      } else if (errorData && errorData.message) {
        errMsg += ` - ${errorData.message}`;
      } else if (errorData && typeof errorData.error === "string") {
        errMsg += ` - ${errorData.error}`;
      }
    } catch {
      try {
        const textData = await response.text();
        if (textData) errMsg += ` - ${textData}`;
      } catch {}
    }
    throw new Error(errMsg);
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

export async function searchInstruments(query: string): Promise<InstrumentCard[]> {
  const response = await fetchJson<{ items: InstrumentDTO[] }>(`/instruments/search?query=${encodeURIComponent(query)}`);
  return response.items.map(mapInstrument);
}

export async function addInstrumentToWatchlist(instrumentUid: string, position: number = 0): Promise<void> {
  const response = await fetch(`${apiBaseUrl}/watchlist`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ instrument_uid: instrumentUid, position }),
  });
  if (!response.ok) {
    throw new Error(`failed to add instrument: ${response.status}`);
  }
}

function mapInstrument(dto: InstrumentDTO): InstrumentCard {
  return {
    uid: dto.uid,
    figi: dto.figi,
    ticker: dto.ticker,
    classCode: dto.class_code,
    isin: dto.isin,
    lot: dto.lot,
    currency: dto.currency,
    name: dto.name,
    exchange: dto.exchange,
    instrumentType: dto.instrument_type,
    apiTradeAvailable: dto.api_trade_available,
    first1MinCandleDate: dto.first_1min_candle_date,
    first1DayCandleDate: dto.first_1day_candle_date,
  };
}

// Sprint 6: Freshness & Watchlist Job API

export async function triggerWatchlistRefresh(): Promise<JobRun> {
  const response = await fetch(`${apiBaseUrl}/jobs/data-refresh`, { method: "POST" });
  if (!response.ok) throw new Error(`watchlist refresh failed: ${response.status}`);
  return mapJobRun((await response.json()) as JobRunDTO);
}

export async function triggerWatchlistSignalRefresh(): Promise<JobRun> {
  const response = await fetch(`${apiBaseUrl}/jobs/signals/run`, { method: "POST" });
  if (!response.ok) throw new Error(`watchlist signal refresh failed: ${response.status}`);
  return mapJobRun((await response.json()) as JobRunDTO);
}

export async function loadFreshness(): Promise<FreshnessSummary> {
  const dto = await fetchJson<FreshnessSummaryDTO>("/watchlist/freshness");
  return mapFreshnessSummary(dto);
}

export async function loadSchedulers(): Promise<SchedulerInfo[]> {
  const dto = await fetchJson<SchedulersResponse>("/jobs/schedulers");
  return dto.schedulers;
}

function mapFreshnessSummary(dto: FreshnessSummaryDTO): FreshnessSummary {
  return {
    generatedAt: dto.generated_at,
    totalItems: dto.total_items,
    freshData: dto.fresh_data,
    staleData: dto.stale_data,
    freshSignals: dto.fresh_signals,
    staleSignals: dto.stale_signals,
    watchlistOnly: dto.watchlist_only,
    items: dto.items.map(mapFreshnessItem),
  };
}

function mapFreshnessItem(dto: FreshnessItemDTO): FreshnessItem {
  return {
    assetId: dto.asset_id,
    ticker: dto.ticker,
    name: dto.name,
    lastCandleAt: dto.last_candle_at,
    lastSignalAt: dto.last_signal_at,
    dataFresh: dto.data_fresh,
    signalFresh: dto.signal_fresh,
    staleReason: dto.stale_reason,
    modelSupported: dto.model_supported,
  };
}
