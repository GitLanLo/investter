import type {
  AssetWorkbenchData,
  AssetWorkbenchRequest,
  CandleBar,
  FactorPoint,
  WorkspaceShellData,
} from "./types";

const mockAssets = [
  { id: "SBER", ticker: "SBER", name: "Sberbank", venue: "MOEX", timeframe: "5m", active: true },
  { id: "GAZP", ticker: "GAZP", name: "Gazprom", venue: "MOEX", timeframe: "5m", active: true },
  { id: "LKOH", ticker: "LKOH", name: "Lukoil", venue: "MOEX", timeframe: "5m", active: true },
  { id: "NVTK", ticker: "NVTK", name: "Novatek", venue: "MOEX", timeframe: "5m", active: true },
  { id: "MOEX", ticker: "MOEX", name: "Moscow Exchange", venue: "MOEX", timeframe: "5m", active: true },
];

export const mockWorkspaceShellData: WorkspaceShellData = {
  generatedFrom: "mock",
  assets: mockAssets,
  latestSignals: [
    {
      id: "SBER:1",
      assetId: "SBER",
      direction: "up",
      state: "actionable",
      probability: 0.74,
      threshold: 0.3,
      asOfTime: "2026-04-10T09:55:00Z",
      timeframe: "5m",
      modelVersion: "sprint2_hgb_platt_v1",
      horizonBars: 12,
      classProbabilities: { up: 0.74, down: 0.14, no_trade: 0.12 },
      policy: {
        policyStatus: "production_candidate",
        modelName: "hgb_multiclass",
        scenarioName: "core_price_volume_only",
        calibrationMethod: "platt",
        threshold: 0.3,
        datasetVersion: "mvp_live_wf_20260409",
      },
    },
    {
      id: "GAZP:1",
      assetId: "GAZP",
      direction: "down",
      state: "actionable",
      probability: 0.61,
      threshold: 0.3,
      asOfTime: "2026-04-10T09:55:00Z",
      timeframe: "5m",
      modelVersion: "sprint2_hgb_platt_v1",
      horizonBars: 12,
      classProbabilities: { up: 0.21, down: 0.61, no_trade: 0.18 },
      policy: {
        policyStatus: "production_candidate",
        modelName: "hgb_multiclass",
        scenarioName: "core_price_volume_only",
        calibrationMethod: "platt",
        threshold: 0.3,
        datasetVersion: "mvp_live_wf_20260409",
      },
    },
    {
      id: "LKOH:1",
      assetId: "LKOH",
      direction: "none",
      state: "no_trade",
      probability: 0.42,
      threshold: 0.3,
      asOfTime: "2026-04-10T09:55:00Z",
      timeframe: "5m",
      modelVersion: "sprint2_hgb_platt_v1",
      horizonBars: 12,
      classProbabilities: { up: 0.28, down: 0.3, no_trade: 0.42 },
      policy: {
        policyStatus: "production_candidate",
        modelName: "hgb_multiclass",
        scenarioName: "core_price_volume_only",
        calibrationMethod: "platt",
        threshold: 0.3,
        datasetVersion: "mvp_live_wf_20260409",
      },
    },
  ],
  mlOverview: {
    generatedAt: "2026-04-10T10:00:00Z",
    warnings: [],
    sourcePaths: {
      dataset_manifest: "data/datasets/dataset_version=mvp_live_wf_20260409/manifest.json",
      research_summary: "artifacts/research/mvp_live_wf_20260409/ablation_sprint2_richer_modelpack/summary.json",
      calibration_summary: "artifacts/research/mvp_live_wf_20260409/calibration_audit_hgb_core_price_volume/summary.json",
      grid_matrix: "artifacts/research/sprint7/timeframe_horizon_matrix.json",
    },
    dataset: {
      datasetVersion: "mvp_live_wf_20260409",
      featureSchemaVersion: "feature_v1",
      timeframe: "5m",
      horizonBars: 12,
      tickers: ["GAZP", "LKOH", "MOEX", "NVTK", "SBER"],
      trainRows: 7247,
      valRows: 1535,
      testRows: 1540,
    },
    gridMatrix: {
      status: "completed",
      gridName: "timeframe_horizon_matrix",
      timeframes: ["5m", "15m", "1h"],
      horizons: [6, 12, 24],
      expectedResultCount: 9,
      completedResultCount: 9,
      results: [
        {
          status: "completed",
          timeframe: "15m",
          horizon: 12,
          datasetVersion: "sprint7_15m_h12_20260506",
          rows: 10322,
          valActionableF1: 0.34,
          valPrecision: 0.43,
          valCoverage: 0.31,
          valEce: 0.18,
        },
        {
          status: "completed",
          timeframe: "5m",
          horizon: 12,
          datasetVersion: "sprint7_5m_h12_20260506",
          rows: 30966,
          valActionableF1: 0.29,
          valPrecision: 0.36,
          valCoverage: 0.39,
          valEce: 0.24,
        },
      ],
    },
    research: {
      researchCandidate: {
        label: "Research candidate",
        scenarioName: "no_regime",
        modelName: "hgb_multiclass",
        selectionMode: "research_candidate",
        selectedThreshold: 0.55,
        validation: {
          macroF1: 0.457,
          balancedAccuracy: 0.471,
          actionableF1: 0.309,
          precision: 0.407,
          coverage: 0.33,
          actionableEce: 0.336,
        },
        test: {
          macroF1: 0.442,
          balancedAccuracy: 0.463,
          actionableF1: 0.278,
          precision: 0.34,
          coverage: 0.29,
          actionableEce: 0.403,
        },
      },
      productionCandidate: {
        label: "Production candidate",
        scenarioName: "core_price_volume_only",
        modelName: "hgb_multiclass",
        selectionMode: "production_candidate",
        selectedThreshold: 0.55,
        gatePassed: true,
        validation: {
          macroF1: 0.438,
          balancedAccuracy: 0.447,
          actionableF1: 0.271,
          precision: 0.388,
          coverage: 0.289,
          actionableEce: 0.298,
        },
        test: {
          macroF1: 0.402,
          balancedAccuracy: 0.406,
          actionableF1: 0.212,
          precision: 0.27,
          coverage: 0.271,
          actionableEce: 0.426,
        },
      },
      scenarios: [
        scenario("no_regime", 53, 0.55, 0.309, 0.407, 0.33, 0.336, 0.278, 0.34, 0.29, 0.403),
        scenario("full", 57, 0.55, 0.291, 0.411, 0.295, 0.328, 0.252, 0.289, 0.287, 0.399),
        scenario("core_price_volume_only", 15, 0.55, 0.271, 0.388, 0.289, 0.298, 0.212, 0.27, 0.271, 0.426),
        scenario("no_cross_asset", 51, 0.55, 0.281, 0.356, 0.351, 0.408, 0.267, 0.332, 0.29, 0.395),
      ],
    },
    calibration: {
      researchCandidate: {
        label: "Calibration research",
        method: "platt",
        selectionMode: "research_candidate",
        selectedThreshold: 0.3,
        validation: {
          actionableF1: 0.366,
          precision: 0.366,
          coverage: 0.54,
          actionableEce: 0.01,
        },
        test: {
          actionableF1: 0.269,
          precision: 0.252,
          coverage: 0.479,
          actionableEce: 0.115,
        },
      },
      productionCandidate: {
        label: "Calibration production",
        method: "platt",
        selectionMode: "production_candidate",
        selectedThreshold: 0.3,
        gatePassed: true,
        validation: {
          actionableF1: 0.366,
          precision: 0.366,
          coverage: 0.54,
          actionableEce: 0.01,
        },
        test: {
          actionableF1: 0.269,
          precision: 0.252,
          coverage: 0.479,
          actionableEce: 0.115,
        },
      },
      methods: [
        calibrationMethod("platt", true, 0.3, 0.366, 0.366, 0.54, 0.01, 0.269, 0.252, 0.479, 0.115),
        calibrationMethod("identity", true, 0.3, 0.366, 0.366, 0.54, 0.221, 0.269, 0.252, 0.479, 0.348),
        calibrationMethod("isotonic", true, 0.3, 0.362, 0.371, 0.513, 0.0, 0.271, 0.259, 0.458, 0.117),
      ],
    },
  },
  productionPolicy: {
    generatedAt: "2026-04-10T10:05:00Z",
    status: "production_candidate",
    modelName: "hgb_multiclass",
    scenarioName: "core_price_volume_only",
    calibrationMethod: "platt",
    threshold: 0.3,
    timeframe: "5m",
    horizonBars: 12,
    datasetVersion: "mvp_live_wf_20260409",
    featureSchema: "feature_v1",
    trainRows: 7247,
    validationRows: 1535,
    testRows: 1540,
    validation: {
      actionableF1: 0.366,
      precision: 0.366,
      coverage: 0.54,
      actionableEce: 0.01,
    },
    test: {
      actionableF1: 0.269,
      precision: 0.252,
      coverage: 0.479,
      actionableEce: 0.115,
    },
    sourcePaths: {
      research_summary: "artifacts/research/mvp_live_wf_20260409/ablation_sprint2_richer_modelpack/summary.json",
      calibration_summary: "artifacts/research/mvp_live_wf_20260409/calibration_audit_hgb_core_price_volume/summary.json",
    },
    warnings: [],
  },
  policyValidationRuns: [
    {
      id: 1,
      policyStatus: "production_candidate",
      modelName: "hgb_multiclass",
      scenarioName: "core_price_volume_only",
      calibrationMethod: "platt",
      threshold: 0.3,
      datasetVersion: "mvp_live_wf_20260409",
      validation: {
        actionableF1: 0.366,
        precision: 0.366,
        coverage: 0.54,
        actionableEce: 0.01,
      },
      test: {
        actionableF1: 0.269,
        precision: 0.252,
        coverage: 0.479,
        actionableEce: 0.115,
      },
      decisionState: "candidate",
      notes: "mock shadow candidate",
      createdAt: "2026-04-10T10:10:00Z",
    },
  ],
  policyShadowSummary: {
    validationRunId: 1,
    decisionState: "candidate",
    modelName: "hgb_multiclass",
    calibrationMethod: "platt",
    threshold: 0.3,
    datasetVersion: "mvp_live_wf_20260409",
    signalsTotal: 24,
    actionableSignals: 11,
    noTradeSignals: 13,
    upSignals: 6,
    downSignals: 5,
    observedCoverage: 11 / 24,
    firstSignalAt: "2026-04-10T08:00:00Z",
    lastSignalAt: "2026-04-10T10:10:00Z",
  },
  policyOutcomeSummary: {
    validationRunId: 1,
    decisionState: "candidate",
    modelName: "hgb_multiclass",
    calibrationMethod: "platt",
    threshold: 0.3,
    datasetVersion: "mvp_live_wf_20260409",
    signalsTotal: 24,
    actionableSignals: 11,
    maturedSignals: 18,
    pendingSignals: 6,
    overduePendingSignals: 2,
    hitSignals: 7,
    missSignals: 4,
    realizedPrecision: 7 / 11,
    averageReturnPct: 0.0011,
    averageActionReturnPct: 0.0024,
    lastSignalAt: "2026-04-10T10:15:00Z",
    firstMaturedAt: "2026-04-10T09:00:00Z",
    lastMaturedAt: "2026-04-10T10:45:00Z",
    canPromote: false,
    promotionBlockers: [
      {
        code: "insufficient_matured_signals",
        message: "Need more matured actionable signals before promotion.",
      },
      {
        code: "overdue_pending_outcomes",
        message: "Some pending outcomes are past their maturity window and likely indicate missing candles or stale data.",
      },
    ],
  },
  policyOutcomeHistory: [
    {
      signalRunId: 14,
      assetId: "SBER",
      asOfTime: "2026-04-10T09:45:00Z",
      signalState: "actionable",
      signalDirection: "up",
      signalProbability: 0.71,
      timeframe: "5m",
      horizonBars: 12,
      maturedAt: "2026-04-10T10:45:00Z",
      entryPrice: 301.5,
      exitPrice: 302.9,
      rawReturnPct: 0.0046,
      actionReturnPct: 0.0046,
      isHit: true,
    },
    {
      signalRunId: 13,
      assetId: "LKOH",
      asOfTime: "2026-04-10T09:30:00Z",
      signalState: "actionable",
      signalDirection: "down",
      signalProbability: 0.66,
      timeframe: "5m",
      horizonBars: 12,
      maturedAt: "2026-04-10T10:30:00Z",
      entryPrice: 7214,
      exitPrice: 7241,
      rawReturnPct: 0.0037,
      actionReturnPct: -0.0037,
      isHit: false,
    },
  ],
  jobScheduler: {
    enabled: true,
    interval: "15m0s",
    limit: 1000,
    runOnStart: true,
  },
  jobRuns: [
    {
      id: 1,
      jobType: "outcomes_materialize",
      status: "succeeded",
      startedAt: "2026-04-10T10:20:00Z",
      finishedAt: "2026-04-10T10:20:02Z",
      payload: {
        limit: 1000,
        validation_run_id: 1,
        matured_signals: 18,
        pending_signals: 6,
        can_promote: false,
        promotion_blocker_count: 1,
      },
    },
  ],
  artifactDocuments: [
    {
      key: "dataset_manifest",
      title: "Dataset manifest",
      path: "data/datasets/dataset_version=mvp_live_wf_20260409/manifest.json",
      contentType: "json",
      content: JSON.stringify(
        {
          dataset_version: "mvp_live_wf_20260409",
          feature_schema_version: "feature_v1",
          timeframe: "5m",
          horizon_bars: 12,
          tickers: ["GAZP", "LKOH", "MOEX", "NVTK", "SBER"],
          train_range: { rows: 7247 },
          val_range: { rows: 1535 },
          test_range: { rows: 1540 },
        },
        null,
        2,
      ),
    },
    {
      key: "research_summary",
      title: "Research summary",
      path: "artifacts/research/mvp_live_wf_20260409/ablation_sprint2_richer_modelpack/summary.json",
      contentType: "json",
      content: JSON.stringify(
        {
          research_candidate: {
            scenario_name: "no_regime",
            model_name: "hgb_multiclass",
            selected_threshold: 0.55,
          },
          production_candidate: {
            scenario_name: "core_price_volume_only",
            model_name: "hgb_multiclass",
            selected_threshold: 0.55,
          },
        },
        null,
        2,
      ),
    },
    {
      key: "research_report",
      title: "Research report",
      path: "artifacts/research/mvp_live_wf_20260409/ablation_sprint2_richer_modelpack/report.md",
      contentType: "markdown",
      content: [
        "# Research report",
        "",
        "## Winner",
        "",
        "- research candidate: no_regime / hgb_multiclass",
        "- production candidate: core_price_volume_only / hgb_multiclass",
        "",
        "## Validation",
        "",
        "- actionable_f1: 0.309",
        "- precision: 0.407",
        "- coverage: 0.330",
      ].join("\n"),
    },
    {
      key: "calibration_summary",
      title: "Calibration summary",
      path: "artifacts/research/mvp_live_wf_20260409/calibration_audit_hgb_core_price_volume/summary.json",
      contentType: "json",
      content: JSON.stringify(
        {
          production_candidate: {
            method: "platt",
            selected_threshold: 0.3,
          },
          methods: {
            platt: { available: true },
            identity: { available: true },
          },
        },
        null,
        2,
      ),
    },
    {
      key: "calibration_report",
      title: "Calibration report",
      path: "artifacts/research/mvp_live_wf_20260409/calibration_audit_hgb_core_price_volume/report.md",
      contentType: "markdown",
      content: [
        "# Calibration audit",
        "",
        "## Outcome",
        "",
        "- platt chosen as production candidate",
        "- threshold retuned to 0.30",
        "- actionable ECE improved materially on validation",
      ].join("\n"),
    },
  ],
  freshness: {
    generatedAt: "2026-04-10T10:25:00Z",
    totalItems: 5,
    freshData: 3,
    staleData: 2,
    freshSignals: 2,
    staleSignals: 3,
    watchlistOnly: 2,
    items: [
      { assetId: "SBER", ticker: "SBER", name: "Sberbank", dataFresh: true, signalFresh: true, modelSupported: true },
      { assetId: "GAZP", ticker: "GAZP", name: "Gazprom", dataFresh: true, signalFresh: true, modelSupported: true },
      { assetId: "LKOH", ticker: "LKOH", name: "Lukoil", dataFresh: true, signalFresh: false, staleReason: "no signals generated", modelSupported: true },
      { assetId: "NVTK", ticker: "NVTK", name: "Novatek", dataFresh: false, signalFresh: false, staleReason: "no candle data available; watchlist-only (no ML signal)", modelSupported: false },
      { assetId: "MOEX", ticker: "MOEX", name: "Moscow Exchange", dataFresh: false, signalFresh: false, staleReason: "no candle data available; watchlist-only (no ML signal)", modelSupported: false },
    ],
  },
  schedulers: [
    { name: "outcome_materialize", enabled: true, interval: "15m0s", limit: 1000 },
    { name: "watchlist_refresh", enabled: false, interval: "30m0s", limit: 50 },
    { name: "watchlist_signal_refresh", enabled: false, interval: "30m0s" },
  ],
};

const mockWorkbenchByAsset = new Map<string, AssetWorkbenchData>(
  mockAssets.map((asset, index) => [
    asset.id,
    {
      generatedFrom: "mock",
      candles: buildMockCandles(index),
      factors: buildMockFactors(index),
      signalHistory: buildMockSignalHistory(asset.id, index),
    },
  ]),
);

function selectMockWorkbench(assetId: string, request: AssetWorkbenchRequest): AssetWorkbenchData {
  const base = mockWorkbenchByAsset.get(assetId) ?? {
    generatedFrom: "mock",
    candles: buildMockCandles(0),
    factors: buildMockFactors(0),
    signalHistory: buildMockSignalHistory(assetId, 0),
  };

  const candles = sliceCandles(base.candles, request);
  const factors = sliceFactors(base.factors, candles, request);

  return {
    generatedFrom: "mock",
    candles,
    factors,
    signalHistory: base.signalHistory,
  };
}

export function loadMockAssetWorkbench(assetId: string, request: AssetWorkbenchRequest = {}): AssetWorkbenchData {
  return selectMockWorkbench(assetId, request);
}

function buildMockCandles(seed: number): CandleBar[] {
  const start = Date.parse("2026-04-10T07:00:00Z");
  const base = 285 + seed * 18;
  const candles: CandleBar[] = [];
  let lastClose = base;

  for (let index = 0; index < 42; index += 1) {
    const drift = Math.sin((index + seed * 2) / 4) * 1.1 + Math.cos((index + seed) / 7) * 0.8;
    const open = lastClose;
    const close = open + drift;
    const high = Math.max(open, close) + 0.6 + (index % 3) * 0.15;
    const low = Math.min(open, close) - 0.55 - (index % 4) * 0.12;
    candles.push({
      timestamp: new Date(start + index * 5 * 60_000).toISOString(),
      open: roundPrice(open),
      high: roundPrice(high),
      low: roundPrice(low),
      close: roundPrice(close),
      volume: 95_000 + seed * 22_000 + index * 1_700,
    });
    lastClose = close;
  }

  return candles;
}

function buildMockFactors(seed: number): FactorPoint[] {
  const candles = buildMockCandles(seed);
  const points: FactorPoint[] = [];
  for (const [index, candle] of candles.entries()) {
    points.push({
      factor: "usdrub",
      timestamp: candle.timestamp,
      close: roundPrice(91.6 + seed * 0.1 + Math.sin(index / 6) * 0.45),
    });
    points.push({
      factor: "brent",
      timestamp: candle.timestamp,
      close: roundPrice(82.1 + seed * 0.2 + Math.cos(index / 5) * 0.9),
    });
    points.push({
      factor: "rtsi",
      timestamp: candle.timestamp,
      close: roundPrice(1120 + seed * 8 + Math.sin(index / 7) * 18),
    });
  }
  return points;
}

function buildMockSignalHistory(assetId: string, seed: number) {
  const baseTime = Date.parse("2026-04-10T09:55:00Z");
  const directions = ["up", "down", "none", "up", "none", "down"];

  return directions.map((direction, index) => {
    const probabilityBase = direction === "up" ? 0.68 : direction === "down" ? 0.62 : 0.44;
    const probability = Math.max(0.34, Math.min(0.86, probabilityBase + seed * 0.01 - index * 0.015));
    return {
      id: `${assetId}:${index + 1}`,
      assetId,
      direction,
      state: direction === "none" ? "no_trade" : "actionable",
      probability: roundProbability(probability),
      threshold: 0.3,
      asOfTime: new Date(baseTime - index * 30 * 60_000).toISOString(),
      timeframe: "5m",
      modelVersion: "sprint2_hgb_platt_v1",
      horizonBars: 12,
      classProbabilities:
        direction === "up"
          ? { up: roundProbability(probability), down: 0.16, no_trade: roundProbability(1 - probability - 0.16) }
          : direction === "down"
            ? { up: 0.18, down: roundProbability(probability), no_trade: roundProbability(1 - probability - 0.18) }
            : { up: 0.24, down: 0.22, no_trade: roundProbability(0.54) },
    };
  });
}

function sliceCandles(candles: CandleBar[], request: AssetWorkbenchRequest) {
  const fromMs = request.from ? Date.parse(request.from) : Number.NEGATIVE_INFINITY;
  const toMs = request.to ? Date.parse(request.to) : Number.POSITIVE_INFINITY;
  const filtered = candles.filter((candle) => {
    const ts = Date.parse(candle.timestamp);
    return ts >= fromMs && ts <= toMs;
  });

  if (request.limit && filtered.length > request.limit) {
    return filtered.slice(filtered.length - request.limit);
  }
  return filtered;
}

function sliceFactors(factors: FactorPoint[], candles: CandleBar[], request: AssetWorkbenchRequest) {
  const from = candles[0]?.timestamp ?? request.from;
  const to = candles[candles.length - 1]?.timestamp ?? request.to;
  if (!from || !to) {
    return factors;
  }
  return factors.filter((point) => point.timestamp >= from && point.timestamp <= to);
}

function scenario(
  name: string,
  featureCount: number,
  selectedThreshold: number,
  valActionableF1: number,
  valPrecision: number,
  valCoverage: number,
  valEce: number,
  testActionableF1: number,
  testPrecision: number,
  testCoverage: number,
  testEce: number,
) {
  return {
    name,
    featureCount,
    modelName: "hgb_multiclass",
    selectedThreshold,
    validation: {
      actionableF1: valActionableF1,
      precision: valPrecision,
      coverage: valCoverage,
      actionableEce: valEce,
    },
    test: {
      actionableF1: testActionableF1,
      precision: testPrecision,
      coverage: testCoverage,
      actionableEce: testEce,
    },
  };
}

function calibrationMethod(
  method: string,
  available: boolean,
  selectedThreshold: number,
  valActionableF1: number,
  valPrecision: number,
  valCoverage: number,
  valEce: number,
  testActionableF1: number,
  testPrecision: number,
  testCoverage: number,
  testEce: number,
) {
  return {
    method,
    available,
    selectedThreshold,
    validation: {
      actionableF1: valActionableF1,
      precision: valPrecision,
      coverage: valCoverage,
      actionableEce: valEce,
    },
    test: {
      actionableF1: testActionableF1,
      precision: testPrecision,
      coverage: testCoverage,
      actionableEce: testEce,
    },
  };
}

function roundPrice(value: number): number {
  return Math.round(value * 100) / 100;
}

function roundProbability(value: number): number {
  return Math.round(value * 1000) / 1000;
}
