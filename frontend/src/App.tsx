import { startTransition, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import type {
  PointerEvent as ReactPointerEvent,
  WheelEvent as ReactWheelEvent,
} from "react";
import {
  createPolicyValidationRun,
  formatMetric,
  formatProbability,
  loadAssetWorkbench,
  loadWorkspaceShell,
  runOutcomeMaterializationJob,
  runAnalysisForAsset,
  updatePolicyValidationRun,
} from "./lib/api";
import type {
  ArtifactDocument,
  AssetCard,
  AssetWorkbenchData,
  CalibrationMethodSnapshot,
  CandidateSnapshot,
  CandleBar,
  FactorPoint,
  PolicyOutcomeSummary,
  PolicyOutcomeRecord,
  PolicyValidationRun,
  JobRun,
  JobSchedulerStatus,
  PolicyShadowSummary,
  ProductionPolicySnapshot,
  ScenarioSnapshot,
  SignalCard,
  WorkspaceShellData,
} from "./lib/types";

const shellNav = ["Overview", "Signal Lab", "Research Board", "Policy Gate", "Artifact Feed"];
const factorPalette = ["#d06931", "#1e7b89", "#5b7c2d", "#8d4fd1"];
const emptyAssets: AssetCard[] = [];
const emptySignals: SignalCard[] = [];
const emptyArtifactDocuments: ArtifactDocument[] = [];
const emptyCandles: CandleBar[] = [];
const emptyFactors: FactorPoint[] = [];
const chartTimeframeOptions = [
  { key: "5m", label: "5m", minutes: 5 },
  { key: "15m", label: "15m", minutes: 15 },
  { key: "1h", label: "1h", minutes: 60 },
  { key: "4h", label: "4h", minutes: 240 },
  { key: "1d", label: "1d", minutes: 1440 },
] as const;
type ChartTimeframe = (typeof chartTimeframeOptions)[number]["key"];
type InitialViewState = {
  chartOnly: boolean;
  assetId: string | null;
  chartTimeframe: ChartTimeframe;
};
type FocusedChartRange = {
  from: string;
  to: string;
  bars: number;
  source: "selection" | "viewport";
};

export function App() {
  const [initialViewState] = useState<InitialViewState>(() => readInitialViewState());
  const [shellData, setShellData] = useState<WorkspaceShellData | null>(null);
  const [workbenchData, setWorkbenchData] = useState<AssetWorkbenchData | null>(null);
  const [selectedAssetId, setSelectedAssetId] = useState<string | null>(initialViewState.assetId);
  const [selectedChartTimeframe, setSelectedChartTimeframe] = useState<ChartTimeframe>(
    initialViewState.chartTimeframe,
  );
  const [focusedRange, setFocusedRange] = useState<FocusedChartRange | null>(null);
  const [selectedArtifactKey, setSelectedArtifactKey] = useState<string | null>(null);
  const [loadingAsset, setLoadingAsset] = useState(false);
  const [runPending, setRunPending] = useState(false);
  const [policyValidationPending, setPolicyValidationPending] = useState(false);
  const [policyOutcomeJobPending, setPolicyOutcomeJobPending] = useState(false);
  const [policyTransitionPendingId, setPolicyTransitionPendingId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const deferredAssetId = useDeferredValue(selectedAssetId);

  useEffect(() => {
    let active = true;

    void loadWorkspaceShell().then((nextData) => {
      if (!active) {
        return;
      }

      startTransition(() => {
        setShellData(nextData);
        setSelectedAssetId((current) => {
          if (current && nextData.assets.some((asset) => asset.id === current)) {
            return current;
          }
          return nextData.latestSignals[0]?.assetId ?? nextData.assets[0]?.id ?? null;
        });
        setActionError(null);
      });
    });

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!deferredAssetId) {
      return;
    }

    let active = true;
    setLoadingAsset(true);
    void loadAssetWorkbench(deferredAssetId)
      .then((nextData) => {
        if (!active) {
          return;
        }
        startTransition(() => {
          setWorkbenchData(nextData);
        });
      })
      .finally(() => {
        if (active) {
          setLoadingAsset(false);
        }
      });

    return () => {
      active = false;
    };
  }, [deferredAssetId]);

  useEffect(() => {
    if (!initialViewState.chartOnly) {
      return;
    }

    const nextUrl = buildChartWindowUrl(selectedAssetId, selectedChartTimeframe);
    window.history.replaceState(null, "", nextUrl);
  }, [initialViewState.chartOnly, selectedAssetId, selectedChartTimeframe]);

  useEffect(() => {
    setFocusedRange(null);
  }, [selectedAssetId, selectedChartTimeframe]);

  const assets = shellData?.assets ?? emptyAssets;
  const signals = shellData?.latestSignals ?? emptySignals;
  const selectedAsset =
    assets.find((asset) => asset.id === selectedAssetId) ?? assets[0] ?? null;
  const selectedSignal =
    signals.find((signal) => signal.assetId === selectedAsset?.id) ?? signals[0] ?? null;
  const overview = shellData?.mlOverview;
  const productionPolicy = shellData?.productionPolicy;
  const policyValidationRuns = shellData?.policyValidationRuns ?? [];
  const policyShadowSummary = shellData?.policyShadowSummary;
  const policyOutcomeSummary = shellData?.policyOutcomeSummary;
  const policyOutcomeHistory = shellData?.policyOutcomeHistory ?? [];
  const jobScheduler = shellData?.jobScheduler;
  const jobRuns = shellData?.jobRuns ?? [];
  const artifactDocuments = shellData?.artifactDocuments ?? emptyArtifactDocuments;
  const baseCandles = workbenchData?.candles ?? emptyCandles;
  const baseFactors = workbenchData?.factors ?? emptyFactors;
  const selectedSignalHistory = workbenchData?.signalHistory ?? emptySignals;
  const visibleCandles = useMemo(
    () => aggregateCandles(baseCandles, selectedChartTimeframe),
    [baseCandles, selectedChartTimeframe],
  );
  const visibleFactors = useMemo(
    () => aggregateFactors(baseFactors, selectedChartTimeframe),
    [baseFactors, selectedChartTimeframe],
  );
  const focusedFactors = useMemo(
    () => filterFactorsByRange(visibleFactors, focusedRange),
    [visibleFactors, focusedRange],
  );
  const visibleGaps = countTradingGaps(visibleCandles, selectedChartTimeframe);
  const selectedArtifact =
    artifactDocuments.find((item) => item.key === selectedArtifactKey) ?? artifactDocuments[0] ?? null;

  useEffect(() => {
    if (!artifactDocuments.length) {
      setSelectedArtifactKey(null);
      return;
    }
    if (!selectedArtifactKey || !artifactDocuments.some((item) => item.key === selectedArtifactKey)) {
      setSelectedArtifactKey(artifactDocuments[0].key);
    }
  }, [artifactDocuments, selectedArtifactKey]);

  async function handleRunAnalysis() {
    if (!selectedAsset) {
      return;
    }

    setRunPending(true);
    setActionError(null);
    try {
      const nextSignal = await runAnalysisForAsset(selectedAsset.id, selectedAsset.timeframe);
      startTransition(() => {
        setShellData((current) => {
          if (!current) {
            return current;
          }
          const filtered = current.latestSignals.filter((item) => item.assetId !== nextSignal.assetId);
          return {
            ...current,
            generatedFrom: "api",
            latestSignals: [nextSignal, ...filtered].slice(0, 12),
          };
        });
        setWorkbenchData((current) => {
          if (!current) {
            return current;
          }
          const filtered = current.signalHistory.filter((item) => item.id !== nextSignal.id);
          return {
            ...current,
            generatedFrom: "api",
            signalHistory: [nextSignal, ...filtered].slice(0, 24),
          };
        });
      });
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "analysis run failed");
    } finally {
      setRunPending(false);
    }
  }

  async function handleCreatePolicyValidationRun() {
    if (!productionPolicy || shellData?.generatedFrom === "mock") {
      return;
    }

    setPolicyValidationPending(true);
    setActionError(null);
    try {
      const nextRun = await createPolicyValidationRun(
        `GUI validation snapshot for ${productionPolicy.modelName || "current policy"}`,
      );
      startTransition(() => {
        setShellData((current) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            generatedFrom: "api",
            policyValidationRuns: [nextRun, ...current.policyValidationRuns.filter((item) => item.id !== nextRun.id)].slice(0, 8),
          };
        });
      });
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "policy validation run failed");
    } finally {
      setPolicyValidationPending(false);
    }
  }

  async function handleUpdatePolicyValidationRun(run: PolicyValidationRun, decisionState: string) {
    if (shellData?.generatedFrom === "mock") {
      return;
    }

    setPolicyTransitionPendingId(run.id);
    setActionError(null);
    try {
      const updatedRun = await updatePolicyValidationRun(
        run.id,
        decisionState,
        `GUI decision: ${decisionState} on ${new Date().toISOString()}`,
      );
      startTransition(() => {
        setShellData((current) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            generatedFrom: "api",
            policyValidationRuns: current.policyValidationRuns.map((item) =>
              item.id === updatedRun.id ? updatedRun : item,
            ),
          };
        });
      });
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "policy validation update failed");
    } finally {
      setPolicyTransitionPendingId(null);
    }
  }

  async function handleRunOutcomeMaterializationJob() {
    if (shellData?.generatedFrom === "mock") {
      return;
    }

    setPolicyOutcomeJobPending(true);
    setActionError(null);
    try {
      const nextJob = await runOutcomeMaterializationJob();
      const refreshedShell = await loadWorkspaceShell();
      startTransition(() => {
        setShellData({
          ...refreshedShell,
          generatedFrom: "api",
          jobRuns: [nextJob, ...refreshedShell.jobRuns.filter((item) => item.id !== nextJob.id)].slice(0, 6),
        });
      });
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "outcome materialization failed");
    } finally {
      setPolicyOutcomeJobPending(false);
    }
  }

  function handleOpenChartWindow() {
    const chartUrl = buildChartWindowUrl(selectedAsset?.id ?? selectedAssetId, selectedChartTimeframe);
    window.open(chartUrl, "_blank", "noopener,noreferrer,width=1480,height=980");
  }

  if (initialViewState.chartOnly) {
    return (
      <ChartWindowPage
        assets={assets}
        selectedAsset={selectedAsset}
        selectedAssetId={selectedAssetId}
        selectedSignal={selectedSignal}
        loadingAsset={loadingAsset}
        candles={visibleCandles}
        factors={focusedFactors}
        visibleGaps={visibleGaps}
        selectedChartTimeframe={selectedChartTimeframe}
        onSelectAsset={setSelectedAssetId}
        onChartTimeframeSelect={setSelectedChartTimeframe}
        onFocusRangeChange={setFocusedRange}
      />
    );
  }

  return (
    <div className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Sprint 2 operator GUI</p>
          <h1>invest ML control room</h1>
          <p className="topbar-text">
            One workspace for signal execution, research snapshots, candle context and calibration
            artifacts.
          </p>
        </div>
        <nav className="nav">
          {shellNav.map((item) => (
            <a key={item} href={`#${item.toLowerCase().replace(/\s+/g, "-")}`}>
              {item}
            </a>
          ))}
        </nav>
      </header>

      <main className="layout">
        <section className="hero card" id="overview">
          <div className="hero-copy">
            <p className="eyebrow">ML workspace</p>
            <h2>Research, runtime and market structure in a single operator view.</h2>
            <p className="hero-text">
              The shell now works as a real ML deck: pick an asset, inspect the latest candle tape,
              compare factor curves, rerun analysis and audit the current research and calibration
              candidates.
            </p>
            <div className="hero-grid">
              <Metric label="Universe assets" value={String(assets.length || 5)} />
              <Metric
                label="Deploy path"
                value={overview?.calibration?.productionCandidate?.method ?? "platt"}
              />
              <Metric
                label="Dataset split"
                value={overview?.dataset ? `${overview.dataset.trainRows}/${overview.dataset.valRows}/${overview.dataset.testRows}` : "n/a"}
              />
              <Metric
                label="Runtime"
                value={shellData?.generatedFrom === "mock" ? "fallback" : "live"}
                tone={shellData?.generatedFrom === "mock" ? "warn" : "good"}
              />
            </div>
          </div>

          <div className="hero-side">
            <div className="section-heading compact">
              <div>
                <p className="eyebrow">Asset switcher</p>
                <h3>{selectedAsset?.ticker ?? "SBER"} workbench</h3>
              </div>
              <span className="section-note">{selectedAsset?.timeframe ?? "5m"}</span>
            </div>

            <div className="asset-rail">
              {assets.map((asset) => (
                <button
                  key={asset.id}
                  className={`asset-pill ${asset.id === selectedAsset?.id ? "asset-pill-active" : ""}`}
                  type="button"
                  onClick={() => setSelectedAssetId(asset.id)}
                >
                  <strong>{asset.ticker}</strong>
                  <span>{asset.venue}</span>
                </button>
              ))}
            </div>

            {overview?.warnings?.length ? (
              <div className="warning-strip">
                {overview.warnings.slice(0, 2).map((warning) => (
                  <p key={warning}>{warning}</p>
                ))}
              </div>
            ) : null}
          </div>
        </section>

        <section className="card chart-stage" id="signal-lab">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Signal lab</p>
              <h3>{selectedAsset?.ticker ?? "SBER"} candle workbench</h3>
            </div>
            <span className="section-note">
              {loadingAsset ? "loading market tape" : `${visibleCandles.length} bars · ${selectedChartTimeframe} chart`}
            </span>
          </div>
          <div className="chart-toolbar">
            <ChartToolbarControls
              selectedChartTimeframe={selectedChartTimeframe}
              onChartTimeframeSelect={setSelectedChartTimeframe}
              onOpenChartWindow={handleOpenChartWindow}
            />
            <p className="chart-note">
              Drag across the chart to inspect any span. OHLCV aggregates over the selected range,
              session gaps stay visible, wheel zooms, and arrows pan the viewport.
            </p>
          </div>
          <CandleChart
            candles={visibleCandles}
            timeframe={selectedChartTimeframe}
            signal={selectedSignal}
            onFocusRangeChange={setFocusedRange}
          />
          <div className="chart-footer">
            <ChartMetric
              label="Last close"
              value={formatPrice(visibleCandles[visibleCandles.length - 1]?.close)}
            />
            <ChartMetric label="Window range" value={formatRange(visibleCandles)} wrapValue />
            <ChartMetric label="Trading gaps" value={String(visibleGaps)} />
            <ChartMetric label="Feed" value={workbenchData?.generatedFrom ?? shellData?.generatedFrom ?? "loading"} />
          </div>
        </section>

        <section className="card signal-console">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Execution panel</p>
              <h3>{selectedSignal?.assetId ?? selectedAsset?.ticker ?? "SBER"} live signal</h3>
            </div>
            <span className={`badge badge-${badgeTone(selectedSignal?.direction)}`}>
              {selectedSignal ? selectedSignal.direction : "pending"}
            </span>
          </div>

          <div className="signal-hero">
            <p className="signal-probability">{formatProbability(selectedSignal?.probability)}</p>
            <p className="signal-meta">
              threshold {formatProbability(selectedSignal?.threshold)} · {selectedSignal?.timeframe ?? "5m"} ·{" "}
              {selectedSignal?.modelVersion ?? overview?.research?.productionCandidate?.modelName ?? "loading"}
            </p>
          </div>

          <div className="policy-compare-strip">
            <Metric
              label="Signal"
              value={formatProbability(selectedSignal?.threshold)}
              tone={thresholdDeltaTone(selectedSignal, productionPolicy)}
            />
            <Metric
              label="Policy"
              value={formatProbability(selectedSignal?.policy?.threshold ?? productionPolicy?.threshold)}
              tone={thresholdDeltaTone(selectedSignal, productionPolicy)}
            />
            <Metric
              label="State"
              value={formatPolicyStatus(selectedSignal?.policy?.policyStatus ?? productionPolicy?.status)}
              tone={(selectedSignal?.policy?.policyStatus ?? productionPolicy?.status) === "production_candidate" ? "good" : "warn"}
            />
          </div>

          <div className="console-actions">
            <button
              className="action-button"
              type="button"
              onClick={() => {
                void handleRunAnalysis();
              }}
              disabled={!selectedAsset || runPending || shellData?.generatedFrom === "mock"}
            >
              {runPending ? "Running analysis..." : "Run analysis"}
            </button>
            <p className="console-note">
              {shellData?.generatedFrom === "mock"
                ? "Backend offline, UI is showing local fallback data."
                : selectedSignal
                  ? `Last run ${formatDate(selectedSignal.asOfTime)}`
                  : "Signal runtime ready"}
            </p>
            {actionError ? <p className="console-error">{actionError}</p> : null}
          </div>

          <div className="probability-stack">
            {selectedSignal
              ? Object.entries(selectedSignal.classProbabilities).map(([label, value]) => (
                  <ProbabilityBar key={label} label={formatClassLabel(label)} value={value} />
                ))
              : null}
          </div>
        </section>

        <section className="card factors-panel">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Cross-asset context</p>
              <h3>Factor strip</h3>
            </div>
            <span className="section-note">
              {focusedRange?.source === "selection" ? "selection synced" : "viewport synced"} ·{" "}
              {groupFactors(focusedFactors).length} factors
            </span>
          </div>
          <FactorChart factors={focusedFactors} />
        </section>

        <section className="card intelligence-panel" id="research-board">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Model intelligence</p>
              <h3>Research and deployment candidates</h3>
            </div>
            <span className="section-note">
              {overview?.sourcePaths.research_summary ? baseName(overview.sourcePaths.research_summary) : "no summary"}
            </span>
          </div>
          <div className="candidate-grid">
            <ProductionPolicyCard policy={productionPolicy} />
            <CandidateCard title="Research best" candidate={overview?.research?.researchCandidate} />
            <CandidateCard title="Production gate" candidate={overview?.research?.productionCandidate} />
            <CandidateCard title="Calibration gate" candidate={overview?.calibration?.productionCandidate} />
          </div>
        </section>

        <section className="card policy-validation-panel" id="policy-gate">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Policy gate</p>
              <h3>Validation run ledger</h3>
            </div>
            <span className="section-note">{policyValidationRuns.length} snapshots</span>
          </div>
          <div className="policy-validation-layout">
            <div className="policy-validation-copy">
              <p>
                Persist the current production policy snapshot before shadow/live promotion. Each
                row stores validation and test gates with operator notes.
              </p>
              <PolicyShadowSummaryCard summary={policyShadowSummary} />
              <PolicyOutcomeSummaryCard summary={policyOutcomeSummary} />
              <button
                className="action-button"
                type="button"
                onClick={() => {
                  void handleCreatePolicyValidationRun();
                }}
                disabled={!productionPolicy || policyValidationPending || shellData?.generatedFrom === "mock"}
              >
                {policyValidationPending ? "Saving snapshot..." : "Save policy snapshot"}
              </button>
              <button
                className="action-button"
                type="button"
                onClick={() => {
                  void handleRunOutcomeMaterializationJob();
                }}
                disabled={policyOutcomeJobPending || shellData?.generatedFrom === "mock"}
              >
                {policyOutcomeJobPending ? "Running outcome job..." : "Materialize outcomes"}
              </button>
              <PolicySchedulerCard scheduler={jobScheduler} />
              <PolicyOutcomeHistoryCard items={policyOutcomeHistory} />
              <PolicyJobsCard jobs={jobRuns} />
            </div>
            <div className="policy-run-list">
              {policyValidationRuns.length ? (
                policyValidationRuns.map((run) => (
                  <PolicyValidationRunRow
                    key={run.id}
                    run={run}
                    outcomeSummary={policyOutcomeSummary}
                    disabled={shellData?.generatedFrom === "mock" || policyTransitionPendingId === run.id}
                    onDecision={(decisionState) => {
                      void handleUpdatePolicyValidationRun(run, decisionState);
                    }}
                  />
                ))
              ) : (
                <p className="empty-note">No policy validation runs yet.</p>
              )}
            </div>
          </div>
        </section>

        <section className="card dataset-panel">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Dataset pack</p>
              <h3>Training context</h3>
            </div>
            <span className="section-note">
              {overview?.dataset?.datasetVersion ?? "dataset unavailable"}
            </span>
          </div>

          <div className="dataset-grid">
            <Metric label="Feature schema" value={overview?.dataset?.featureSchemaVersion ?? "n/a"} />
            <Metric label="Timeframe" value={overview?.dataset?.timeframe ?? "n/a"} />
            <Metric label="Horizon bars" value={String(overview?.dataset?.horizonBars ?? 0)} />
            <Metric label="Tickers" value={String(overview?.dataset?.tickers.length ?? 0)} />
          </div>

          <div className="split-stack">
            <SplitBar label="Train" value={overview?.dataset?.trainRows ?? 0} total={totalRows(overview)} />
            <SplitBar label="Validation" value={overview?.dataset?.valRows ?? 0} total={totalRows(overview)} />
            <SplitBar label="Test" value={overview?.dataset?.testRows ?? 0} total={totalRows(overview)} />
          </div>
        </section>

        <section className="card artifact-board" id="artifact-feed">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Artifact feed</p>
              <h3>Latest research documents</h3>
            </div>
            <span className="section-note">{artifactDocuments.length} docs</span>
          </div>
          <div className="artifact-browser">
            <div className="artifact-list">
              {artifactDocuments.map((document) => (
                <button
                  key={document.key}
                  className={`artifact-row ${document.key === selectedArtifact?.key ? "artifact-row-active" : ""}`}
                  type="button"
                  onClick={() => setSelectedArtifactKey(document.key)}
                >
                  <strong>{document.title}</strong>
                  <span>{baseName(document.path)}</span>
                </button>
              ))}
            </div>
            <ArtifactPreview document={selectedArtifact} />
          </div>
        </section>

        <section className="card calibration-board">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Calibration audit</p>
              <h3>Method comparison</h3>
            </div>
            <span className="section-note">
              {overview?.sourcePaths.calibration_summary
                ? baseName(overview.sourcePaths.calibration_summary)
                : "no summary"}
            </span>
          </div>
          <div className="method-list">
            {(overview?.calibration?.methods ?? []).map((method) => (
              <CalibrationMethodCard key={method.method} method={method} />
            ))}
          </div>
        </section>

        <section className="card signal-feed">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Signal history</p>
              <h3>{selectedAsset?.ticker ?? "Asset"} runtime history</h3>
            </div>
            <span className="section-note">{selectedSignalHistory.length} runs</span>
          </div>
          <div className="signal-list">
            {selectedSignalHistory.map((signal, index) => (
              <SignalHistoryRow key={signal.id} signal={signal} active={index === 0} />
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}

function ChartWindowPage({
  assets,
  selectedAsset,
  selectedAssetId,
  selectedSignal,
  loadingAsset,
  candles,
  factors,
  visibleGaps,
  selectedChartTimeframe,
  onSelectAsset,
  onChartTimeframeSelect,
  onFocusRangeChange,
}: {
  assets: AssetCard[];
  selectedAsset: AssetCard | null;
  selectedAssetId: string | null;
  selectedSignal?: SignalCard;
  loadingAsset: boolean;
  candles: CandleBar[];
  factors: FactorPoint[];
  visibleGaps: number;
  selectedChartTimeframe: ChartTimeframe;
  onSelectAsset: (assetId: string) => void;
  onChartTimeframeSelect: (timeframe: ChartTimeframe) => void;
  onFocusRangeChange: (range: FocusedChartRange | null) => void;
}) {
  const windowSummary = summarizeRange(candles);

  return (
    <div className="chart-window-shell">
      <header className="chart-window-header">
        <div>
          <p className="eyebrow">Detached chart window</p>
          <h1>{selectedAsset?.ticker ?? "Asset"} chart board</h1>
          <p className="topbar-text">
            Expanded candle view with mouse range selection, zoom and pan, without the rest of the
            operator workspace.
          </p>
        </div>
        <div className="chart-window-actions">
          <a className="secondary-button" href={window.location.pathname}>
            Open full workspace
          </a>
        </div>
      </header>

      <section className="card chart-window-card">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Chart window</p>
            <h3>{selectedAsset?.ticker ?? "SBER"} detached workbench</h3>
          </div>
          <span className="section-note">
            {loadingAsset ? "loading candles" : `${candles.length} bars · ${selectedChartTimeframe} chart`}
          </span>
        </div>

        <div className="asset-rail asset-rail-compact">
          {assets.map((asset) => (
            <button
              key={asset.id}
              className={`asset-pill ${asset.id === selectedAssetId ? "asset-pill-active" : ""}`}
              type="button"
              onClick={() => onSelectAsset(asset.id)}
            >
              <strong>{asset.ticker}</strong>
              <span>{asset.venue}</span>
            </button>
          ))}
        </div>

        <div className="chart-window-overview">
          <ChartMetric label="Open" value={formatPrice(windowSummary?.open)} />
          <ChartMetric label="High" value={formatPrice(windowSummary?.high)} />
          <ChartMetric label="Low" value={formatPrice(windowSummary?.low)} />
          <ChartMetric label="Close" value={formatPrice(windowSummary?.close)} />
          <ChartMetric label="Move" value={formatSignedPercent(windowSummary?.deltaPct)} />
          <ChartMetric label="Volume" value={formatCompactInteger(windowSummary?.volume)} />
        </div>

        <div className="chart-toolbar">
          <ChartToolbarControls
            selectedChartTimeframe={selectedChartTimeframe}
            onChartTimeframeSelect={onChartTimeframeSelect}
          />
          <p className="chart-note">
            Detached mode keeps the same chart engine, but gives the candle area the whole viewport
            for range brushing.
          </p>
        </div>

        <CandleChart
          candles={candles}
          timeframe={selectedChartTimeframe}
          signal={selectedSignal}
          onFocusRangeChange={onFocusRangeChange}
        />

        <div className="chart-footer">
          <ChartMetric label="Last close" value={formatPrice(candles[candles.length - 1]?.close)} />
          <ChartMetric label="Window range" value={formatRange(candles)} wrapValue />
          <ChartMetric label="Trading gaps" value={String(visibleGaps)} />
          <ChartMetric label="Feed" value="chart window" />
        </div>
      </section>

      <section className="card chart-window-factors">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Cross-asset context</p>
            <h3>Factor strip</h3>
          </div>
          <span className="section-note">{groupFactors(factors).length} synced factors</span>
        </div>
        <FactorChart factors={factors} />
      </section>
    </div>
  );
}

function ChartToolbarControls({
  selectedChartTimeframe,
  onChartTimeframeSelect,
  onOpenChartWindow,
}: {
  selectedChartTimeframe: ChartTimeframe;
  onChartTimeframeSelect: (timeframe: ChartTimeframe) => void;
  onOpenChartWindow?: () => void;
}) {
  return (
    <div className="chart-control-row">
      <div className="timeframe-switch">
        <span className="timeframe-switch-label">Chart timeframe</span>
        <div className="range-switch">
          {chartTimeframeOptions.map((option) => (
            <button
              key={option.key}
              className={`range-pill ${selectedChartTimeframe === option.key ? "range-pill-active" : ""}`}
              type="button"
              onClick={() => onChartTimeframeSelect(option.key)}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>
      {onOpenChartWindow ? (
        <button className="secondary-button" type="button" onClick={onOpenChartWindow}>
          Open chart window
        </button>
      ) : null}
    </div>
  );
}

function Metric({
  label,
  value,
  tone = "neutral",
}: {
  label: string;
  value: string;
  tone?: "neutral" | "warn" | "good";
}) {
  return (
    <div className={`metric metric-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function ChartMetric({
  label,
  value,
  wrapValue = false,
}: {
  label: string;
  value: string;
  wrapValue?: boolean;
}) {
  return (
    <div className="chart-metric">
      <span>{label}</span>
      <strong className={`chart-metric-value ${wrapValue ? "chart-metric-value-wrap" : ""}`}>{value}</strong>
    </div>
  );
}

function ProbabilityBar({ label, value }: { label: string; value: number }) {
  return (
    <div className="probability-bar">
      <div className="probability-labels">
        <span>{label}</span>
        <strong>{formatProbability(value)}</strong>
      </div>
      <div className="probability-track">
        <div className="probability-fill" style={{ width: `${Math.max(8, value * 100)}%` }} />
      </div>
    </div>
  );
}

function CandidateCard({ title, candidate }: { title: string; candidate?: CandidateSnapshot }) {
  return (
    <article className="candidate-card">
      <p className="candidate-title">{title}</p>
      <h4>
        {candidate?.method
          ? `${candidate.method} calibration`
          : `${candidate?.scenarioName ?? "n/a"} / ${candidate?.modelName ?? "n/a"}`}
      </h4>
      <p className="candidate-meta">
        {candidate?.selectionMode ?? "n/a"} · threshold {formatProbability(candidate?.selectedThreshold)}
      </p>
      <div className="candidate-metrics">
        <Metric label="Val act F1" value={formatMetric(candidate?.validation.actionableF1, 3)} />
        <Metric label="Val precision" value={formatMetric(candidate?.validation.precision, 3)} />
        <Metric label="Val coverage" value={formatMetric(candidate?.validation.coverage, 3)} />
        <Metric label="Val ECE" value={formatMetric(candidate?.validation.actionableEce, 3)} />
      </div>
      {candidate?.gatePassed !== undefined ? (
        <p className="candidate-gate">
          Gate {candidate.gatePassed ? "passed" : "fallback"} · test act F1{" "}
          {formatMetric(candidate.test.actionableF1, 3)}
        </p>
      ) : null}
    </article>
  );
}

function ProductionPolicyCard({ policy }: { policy?: ProductionPolicySnapshot }) {
  return (
    <article className="candidate-card production-policy-card">
      <p className="candidate-title">Sprint 3 policy</p>
      <h4>{policy?.status ?? "loading policy"}</h4>
      <p className="candidate-meta">
        {policy
          ? `${policy.scenarioName || "n/a"} / ${policy.modelName || "n/a"} · ${policy.calibrationMethod || "n/a"}`
          : "Production policy endpoint is loading."}
      </p>
      <div className="candidate-metrics">
        <Metric label="Threshold" value={formatProbability(policy?.threshold)} />
        <Metric label="Val F1" value={formatMetric(policy?.validation.actionableF1, 3)} />
        <Metric label="Val ECE" value={formatMetric(policy?.validation.actionableEce, 3)} />
        <Metric label="Coverage" value={formatMetric(policy?.validation.coverage, 3)} />
      </div>
      <p className="candidate-gate">
        {policy
          ? `${policy.datasetVersion || "dataset n/a"} · ${policy.timeframe || "n/a"} · horizon ${policy.horizonBars || 0}`
          : "Waiting for backend policy snapshot."}
      </p>
    </article>
  );
}

function ScenarioCard({ scenario }: { scenario: ScenarioSnapshot }) {
  return (
    <article className="scenario-card">
      <div className="scenario-top">
        <div>
          <p className="scenario-name">{scenario.name}</p>
          <p className="scenario-meta">
            {scenario.modelName ?? "n/a"} · {scenario.featureCount} features · threshold{" "}
            {formatProbability(scenario.selectedThreshold)}
          </p>
        </div>
        <strong>{formatMetric(scenario.validation.actionableF1, 3)}</strong>
      </div>
      <div className="scenario-stats">
        <span>Val precision {formatMetric(scenario.validation.precision, 3)}</span>
        <span>Coverage {formatMetric(scenario.validation.coverage, 3)}</span>
        <span>ECE {formatMetric(scenario.validation.actionableEce, 3)}</span>
      </div>
    </article>
  );
}

function CalibrationMethodCard({ method }: { method: CalibrationMethodSnapshot }) {
  return (
    <article className="method-card">
      <div className="scenario-top">
        <div>
          <p className="scenario-name">{method.method}</p>
          <p className="scenario-meta">
            {method.available ? "available" : "unavailable"} · threshold{" "}
            {formatProbability(method.selectedThreshold)}
          </p>
        </div>
        <strong>{formatMetric(method.validation.actionableF1, 3)}</strong>
      </div>
      <div className="scenario-stats">
        <span>Val precision {formatMetric(method.validation.precision, 3)}</span>
        <span>Coverage {formatMetric(method.validation.coverage, 3)}</span>
        <span>ECE {formatMetric(method.validation.actionableEce, 3)}</span>
      </div>
    </article>
  );
}

function SplitBar({ label, value, total }: { label: string; value: number; total: number }) {
  const width = total > 0 ? Math.max(6, (value / total) * 100) : 0;

  return (
    <div className="split-row">
      <div className="probability-labels">
        <span>{label}</span>
        <strong>{value}</strong>
      </div>
      <div className="probability-track">
        <div className="probability-fill" style={{ width: `${width}%` }} />
      </div>
    </div>
  );
}

function SignalHistoryRow({ signal, active }: { signal: SignalCard; active: boolean }) {
  return (
    <div className={`signal-row ${active ? "signal-row-active" : ""}`}>
      <div>
        <p className="signal-row-title">
          {signal.direction} · {formatProbability(signal.probability)}
        </p>
        <p className="signal-row-meta">
          {formatDate(signal.asOfTime)} · threshold {formatProbability(signal.threshold)} · {signal.modelVersion}
        </p>
      </div>
      <strong>{signal.state}</strong>
    </div>
  );
}

function PolicyValidationRunRow({
  run,
  outcomeSummary,
  disabled,
  onDecision,
}: {
  run: PolicyValidationRun;
  outcomeSummary?: PolicyOutcomeSummary;
  disabled: boolean;
  onDecision: (decisionState: string) => void;
}) {
  const nextDecisionActions = policyDecisionActions(run.decisionState, run.id, outcomeSummary);
  return (
    <article className="policy-run-row">
      <div>
        <p className="signal-row-title">
          {run.modelName} · {run.calibrationMethod}
        </p>
        <p className="signal-row-meta">
          {run.datasetVersion} · threshold {formatProbability(run.threshold)} · {formatDate(run.createdAt)}
        </p>
        {run.notes ? <p className="policy-run-note">{run.notes}</p> : null}
        {outcomeSummary?.validationRunId === run.id && outcomeSummary.promotionBlockers.length ? (
          <div className="policy-blocker-list">
            {outcomeSummary.promotionBlockers.map((blocker) => (
              <p key={blocker.code} className="policy-run-note">
                {blocker.message}
              </p>
            ))}
          </div>
        ) : null}
        {nextDecisionActions.length ? (
          <div className="policy-run-actions">
            {nextDecisionActions.map((action) => (
              <button
                key={action.state}
                className="policy-run-action"
                type="button"
                onClick={() => onDecision(action.state)}
                disabled={disabled}
              >
                {action.label}
              </button>
            ))}
          </div>
        ) : null}
      </div>
      <div className="policy-run-metrics">
        <Metric label="Decision" value={run.decisionState} tone={run.decisionState === "blocked" ? "warn" : "good"} />
        <Metric label="Val F1" value={formatMetric(run.validation.actionableF1, 3)} />
        <Metric label="Val ECE" value={formatMetric(run.validation.actionableEce, 3)} />
        <Metric label="Test precision" value={formatMetric(run.test.precision, 3)} />
      </div>
    </article>
  );
}

function PolicyShadowSummaryCard({ summary }: { summary?: PolicyShadowSummary }) {
  if (!summary) {
    return (
      <div className="shadow-summary-card">
        <p className="shadow-summary-title">Shadow monitor</p>
        <p className="empty-note">No persisted policy signal history yet.</p>
      </div>
    );
  }

  return (
    <div className="shadow-summary-card">
      <div>
        <p className="shadow-summary-title">Shadow monitor</p>
        <p className="signal-row-meta">
          run #{summary.validationRunId} · {summary.decisionState} · {summary.datasetVersion}
        </p>
      </div>
      <div className="shadow-summary-grid">
        <Metric label="Signals" value={String(summary.signalsTotal)} />
        <Metric label="Actionable" value={String(summary.actionableSignals)} />
        <Metric label="Coverage" value={formatProbability(summary.observedCoverage)} />
        <Metric label="Up / Down" value={`${summary.upSignals}/${summary.downSignals}`} />
      </div>
      <p className="signal-row-meta">
        {summary.firstSignalAt && summary.lastSignalAt
          ? `${formatDate(summary.firstSignalAt)} - ${formatDate(summary.lastSignalAt)}`
          : "Waiting for first persisted signal"}
      </p>
    </div>
  );
}

function PolicyOutcomeSummaryCard({ summary }: { summary?: PolicyOutcomeSummary }) {
  if (!summary) {
    return (
      <div className="shadow-summary-card">
        <p className="shadow-summary-title">Forward outcomes</p>
        <p className="empty-note">No realized policy outcomes yet.</p>
      </div>
    );
  }

  return (
    <div className="shadow-summary-card">
      <div>
        <p className="shadow-summary-title">Forward outcomes</p>
        <p className="signal-row-meta">
          run #{summary.validationRunId} · {summary.maturedSignals} matured · {summary.pendingSignals} pending
        </p>
      </div>
      <div className="shadow-summary-grid">
        <Metric label="Precision" value={formatProbability(summary.realizedPrecision)} />
        <Metric label="Hits / Misses" value={`${summary.hitSignals}/${summary.missSignals}`} />
        <Metric label="Avg return" value={formatSignedPercent(summary.averageReturnPct)} />
        <Metric label="Action return" value={formatSignedPercent(summary.averageActionReturnPct)} />
        <Metric label="Overdue pending" value={String(summary.overduePendingSignals)} tone={summary.overduePendingSignals ? "warn" : "good"} />
        <Metric label="Promote" value={summary.canPromote ? "ready" : "blocked"} tone={summary.canPromote ? "good" : "warn"} />
        <Metric label="Blockers" value={String(summary.promotionBlockers.length)} tone={summary.promotionBlockers.length ? "warn" : "good"} />
      </div>
      {summary.promotionBlockers.length ? (
        <div className="policy-blocker-list">
          {summary.promotionBlockers.map((blocker) => (
            <p key={blocker.code} className="policy-run-note">
              {blocker.message}
            </p>
          ))}
        </div>
      ) : null}
      <p className="signal-row-meta">
        {summary.firstMaturedAt && summary.lastMaturedAt
          ? `${formatDate(summary.firstMaturedAt)} - ${formatDate(summary.lastMaturedAt)}`
          : "Waiting for completed horizons"}
        {summary.lastSignalAt ? ` · latest signal ${formatDate(summary.lastSignalAt)}` : ""}
      </p>
    </div>
  );
}

function PolicyJobsCard({ jobs }: { jobs: JobRun[] }) {
  if (!jobs.length) {
    return (
      <div className="shadow-summary-card">
        <p className="shadow-summary-title">Recent jobs</p>
        <p className="empty-note">No materialization jobs yet.</p>
      </div>
    );
  }

  return (
    <div className="shadow-summary-card">
      <div>
        <p className="shadow-summary-title">Recent jobs</p>
        <p className="signal-row-meta">{jobs.length} tracked backend runs</p>
      </div>
      <div className="policy-job-list">
        {jobs.map((job) => (
          <div key={job.id} className="policy-job-row">
            <div>
              <p className="signal-row-title">
                {job.jobType} · {job.status}
              </p>
              <p className="signal-row-meta">
                {formatDate(job.startedAt)}
                {job.finishedAt ? ` → ${formatDate(job.finishedAt)}` : ""}
              </p>
            </div>
            <div className="policy-job-meta">
              <span>matured {formatUnknownMetric(job.payload.matured_signals)}</span>
              <span>pending {formatUnknownMetric(job.payload.pending_signals)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function PolicySchedulerCard({ scheduler }: { scheduler?: JobSchedulerStatus }) {
  if (!scheduler) {
    return (
      <div className="shadow-summary-card">
        <p className="shadow-summary-title">Scheduler</p>
        <p className="empty-note">Scheduler status unavailable.</p>
      </div>
    );
  }

  return (
    <div className="shadow-summary-card">
      <div>
        <p className="shadow-summary-title">Scheduler</p>
        <p className="signal-row-meta">
          {scheduler.enabled ? "automatic materialization enabled" : "manual materialization only"}
        </p>
      </div>
      <div className="shadow-summary-grid">
        <Metric label="Enabled" value={scheduler.enabled ? "on" : "off"} tone={scheduler.enabled ? "good" : "warn"} />
        <Metric label="Interval" value={scheduler.interval} />
        <Metric label="Batch limit" value={String(scheduler.limit)} />
        <Metric label="Run on start" value={scheduler.runOnStart ? "yes" : "no"} tone={scheduler.runOnStart ? "good" : "neutral"} />
      </div>
    </div>
  );
}

function PolicyOutcomeHistoryCard({ items }: { items: PolicyOutcomeRecord[] }) {
  if (!items.length) {
    return (
      <div className="shadow-summary-card">
        <p className="shadow-summary-title">Outcome audit</p>
        <p className="empty-note">No matured outcomes yet.</p>
      </div>
    );
  }

  return (
    <div className="shadow-summary-card">
      <div>
        <p className="shadow-summary-title">Outcome audit</p>
        <p className="signal-row-meta">{items.length} recent matured signals</p>
      </div>
      <div className="policy-outcome-list">
        {items.map((item) => (
          <div key={`${item.signalRunId}:${item.maturedAt}`} className="policy-outcome-row">
            <div>
              <p className="signal-row-title">
                {item.assetId} · {item.signalDirection} · {item.isHit ? "hit" : "miss"}
              </p>
              <p className="signal-row-meta">
                {formatDate(item.asOfTime)} → {formatDate(item.maturedAt)} · {item.timeframe} · horizon{" "}
                {item.horizonBars}
              </p>
            </div>
            <div className="policy-outcome-meta">
              <span>{formatProbability(item.signalProbability)}</span>
              <span>{formatSignedPercent(item.actionReturnPct)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function policyDecisionActions(
  decisionState: string,
  runId?: number,
  outcomeSummary?: PolicyOutcomeSummary,
) {
  switch (decisionState) {
    case "candidate":
      return [
        { state: "shadow_live", label: "Start shadow" },
        { state: "blocked", label: "Block" },
      ];
    case "shadow_live":
      return [
        { state: "promoted", label: "Promote" },
        { state: "blocked", label: "Block" },
      ].filter((action) => {
        if (action.state !== "promoted") {
          return true;
        }
        const matchesRun = outcomeSummary?.validationRunId === runId;
        return matchesRun ? outcomeSummary?.canPromote === true : false;
      });
    case "blocked":
      return [{ state: "candidate", label: "Reopen" }];
    default:
      return [];
  }
}

function CandleChart({
  candles,
  timeframe,
  signal,
  onFocusRangeChange,
}: {
  candles: CandleBar[];
  timeframe: ChartTimeframe;
  signal?: SignalCard;
  onFocusRangeChange?: (range: FocusedChartRange | null) => void;
}) {
  const chartScrollRef = useRef<HTMLDivElement | null>(null);
  const initialWindowSize = initialViewportBars(candles.length, timeframe);
  const [hoverIndex, setHoverIndex] = useState<number | null>(candles.length ? candles.length - 1 : null);
  const [windowSize, setWindowSize] = useState(initialWindowSize);
  const [windowEndIndex, setWindowEndIndex] = useState(candles.length - 1);
  const [selectedRange, setSelectedRange] = useState<{ startIndex: number; endIndex: number } | null>(null);
  const [brushState, setBrushState] = useState<{
    pointerId: number;
    anchorIndex: number;
    currentIndex: number;
    startClientX: number;
  } | null>(null);

  useEffect(() => {
    setWindowSize(initialViewportBars(candles.length, timeframe));
    setWindowEndIndex(candles.length - 1);
    setHoverIndex(candles.length ? candles.length - 1 : null);
    setSelectedRange(null);
    setBrushState(null);
  }, [candles, timeframe]);

  const width = 980;
  const height = 380;
  const chartCanvasWidth = chartScrollCanvasWidth(candles.length, timeframe);
  const padX = 36;
  const padY = 24;
  const priceBottomY = 286;
  const volumeTopY = 308;
  const volumeBottomY = 360;
  const viewport = clampViewport(windowSize, windowEndIndex, candles.length);
  const normalizedWindowSize = Math.max(viewport.endIndex - viewport.startIndex + 1, 0);
  const visibleCandles = candles.slice(viewport.startIndex, viewport.endIndex + 1);
  const lows = visibleCandles.map((item) => item.low);
  const highs = visibleCandles.map((item) => item.high);
  const minPrice = lows.length ? Math.min(...lows) : 0;
  const maxPrice = highs.length ? Math.max(...highs) : 1;
  const span = Math.max(maxPrice - minPrice, 0.01);
  const xPositions = buildTimeScalePositions(
    visibleCandles.map((item) => item.timestamp),
    { width: width - 52, padX },
  );
  const minPixelGap = minimumAdjacentGap(xPositions);
  const candleWidth = Math.max(0.8, Math.min(10, minPixelGap * 0.72));
  const maxVolume = Math.max(...visibleCandles.map((item) => item.volume), 1);
  const priceTicks = [0, 0.25, 0.5, 0.75, 1].map((ratio) => ({
    ratio,
    y: padY + (priceBottomY - padY) * ratio,
    value: maxPrice - span * ratio,
  }));
  const absoluteActiveIndex = Math.round(
    clampNumber(hoverIndex ?? viewport.endIndex, viewport.startIndex, viewport.endIndex),
  );
  const activeIndex = Math.min(
    Math.max(absoluteActiveIndex - viewport.startIndex, 0),
    Math.max(visibleCandles.length - 1, 0),
  );
  const activeCandle = visibleCandles[activeIndex];
  const activeX = xPositions[activeIndex] ?? padX;
  const gapBands = buildGapBands(visibleCandles, xPositions, timeframe);
  const effectiveRange = brushState
    ? normalizeIndexRange(brushState.anchorIndex, brushState.currentIndex)
    : selectedRange;
  const inspectedCandles = effectiveRange
    ? candles.slice(effectiveRange.startIndex, effectiveRange.endIndex + 1)
    : activeCandle
      ? [activeCandle]
      : [];
  const inspectedSummary = summarizeCandles(inspectedCandles);
  const summaryCandles = effectiveRange ? inspectedCandles : visibleCandles;
  const rangeSummary = summarizeRange(summaryCandles);
  const summaryGaps = countTradingGaps(summaryCandles, timeframe);
  const selectionBand = effectiveRange
    ? buildSelectionBand(effectiveRange, viewport.startIndex, xPositions, candleWidth)
    : null;
  const signalMarker = buildSignalMarker(
    signal,
    visibleCandles,
    timeframe,
    xPositions,
    width,
    minPrice,
    span,
    priceBottomY,
    padX,
    padY,
  );
  const canZoomIn = visibleCandles.length > Math.min(96, candles.length);
  const canZoomOut = visibleCandles.length < candles.length;
  const canPanLeft = viewport.startIndex > 0;
  const canPanRight = viewport.endIndex < candles.length - 1;

  useEffect(() => {
    const scrollNode = chartScrollRef.current;
    if (!scrollNode) {
      return;
    }
    window.requestAnimationFrame(() => {
      scrollNode.scrollLeft = scrollNode.scrollWidth - scrollNode.clientWidth;
    });
  }, [chartCanvasWidth, timeframe, candles.length]);

  useEffect(() => {
    if (!onFocusRangeChange) {
      return;
    }
    if (!summaryCandles.length) {
      onFocusRangeChange(null);
      return;
    }
    onFocusRangeChange({
      from: summaryCandles[0].timestamp,
      to: summaryCandles[summaryCandles.length - 1].timestamp,
      bars: summaryCandles.length,
      source: effectiveRange ? "selection" : "viewport",
    });
  }, [
    effectiveRange ? "selection" : "viewport",
    onFocusRangeChange,
    summaryCandles.length,
    summaryCandles[0]?.timestamp,
    summaryCandles[summaryCandles.length - 1]?.timestamp,
  ]);

  if (!candles.length) {
    return <div className="chart-empty">No candle data loaded.</div>;
  }

  const closeLine = visibleCandles
    .map((candle, index) => {
      const x = xPositions[index];
      const y = scaleYInRange(candle.close, minPrice, span, padY, priceBottomY);
      return `${x},${y}`;
    })
    .join(" ");

  function clearSelection() {
    setSelectedRange(null);
    setBrushState(null);
  }

  function absoluteIndexFromEvent(event: ReactPointerEvent<SVGSVGElement>) {
    const svg = event.currentTarget;
    const rect = svg.getBoundingClientRect();
    const relativeX = ((event.clientX - rect.left) / rect.width) * width;
    return viewport.startIndex + findClosestXIndex(xPositions, relativeX);
  }

  function handlePointerMove(event: ReactPointerEvent<SVGSVGElement>) {
    setHoverIndex(absoluteIndexFromEvent(event));
  }

  function applyZoom(nextSize: number, anchorAbsoluteIndex: number) {
    if (!candles.length) {
      return;
    }
    clearSelection();
    const boundedAnchorIndex = Math.round(clampNumber(anchorAbsoluteIndex, 0, candles.length - 1));
    const boundedSize = Math.max(
      Math.min(nextSize, maxViewportBars(candles.length)),
      Math.min(24, candles.length),
    );
    const anchorOffset = Math.min(
      Math.max(boundedAnchorIndex - viewport.startIndex, 0),
      Math.max(visibleCandles.length - 1, 0),
    );
    const anchorRatio = visibleCandles.length > 1 ? anchorOffset / (visibleCandles.length - 1) : 1;
    const nextStart = Math.round(boundedAnchorIndex - anchorRatio * Math.max(boundedSize - 1, 0));
    const nextClampedStart = Math.min(Math.max(nextStart, 0), Math.max(candles.length - boundedSize, 0));
    setWindowSize(boundedSize);
    setWindowEndIndex(nextClampedStart + boundedSize - 1);
    setHoverIndex(boundedAnchorIndex);
  }

  function handleWheel(event: ReactWheelEvent<SVGSVGElement>) {
    if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
      return;
    }
    event.preventDefault();
    const direction = event.deltaY < 0 ? "in" : "out";
    const nextSize = direction === "in"
      ? Math.floor(visibleCandles.length * 0.75)
      : Math.ceil(visibleCandles.length * 1.25);
    applyZoom(nextSize, absoluteActiveIndex);
  }

  function shiftWindow(direction: "left" | "right") {
    clearSelection();
    const step = Math.max(Math.floor(visibleCandles.length * 0.25), 24);
    const nextEnd = direction === "left" ? viewport.endIndex - step : viewport.endIndex + step;
    const nextViewport = clampViewport(normalizedWindowSize, nextEnd, candles.length);
    setWindowSize(normalizedWindowSize);
    setWindowEndIndex(nextViewport.endIndex);
    setHoverIndex(nextViewport.endIndex);
  }

  function handlePointerDown(event: ReactPointerEvent<SVGSVGElement>) {
    if (event.button !== 0) {
      return;
    }
    const anchorIndex = absoluteIndexFromEvent(event);
    event.currentTarget.setPointerCapture(event.pointerId);
    setHoverIndex(anchorIndex);
    setBrushState({
      pointerId: event.pointerId,
      anchorIndex,
      currentIndex: anchorIndex,
      startClientX: event.clientX,
    });
  }

  function handlePointerUp(event: ReactPointerEvent<SVGSVGElement>) {
    if (brushState && brushState.pointerId === event.pointerId) {
      const nextIndex = absoluteIndexFromEvent(event);
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId);
      }
      setBrushState(null);
      setHoverIndex(nextIndex);
      if (Math.abs(event.clientX - brushState.startClientX) < 5) {
        setSelectedRange(null);
        return;
      }
      setSelectedRange(normalizeIndexRange(brushState.anchorIndex, nextIndex));
    }
  }

  return (
    <div className="chart-shell">
      <div className="chart-inspector">
        <div className="chart-inspector-copy">
          <strong className={`chart-inspector-title ${effectiveRange ? "chart-inspector-title-range" : ""}`}>
            {effectiveRange ? formatSelectionLabel(inspectedCandles) : formatDate(activeCandle?.timestamp)}
          </strong>
          <div className="chart-inspector-meta">
            <span className="chart-inspector-note">
              {effectiveRange
                ? `${inspectedCandles.length} bars selected`
                : "Hover a candle or drag to select a range"}
            </span>
            <span className="chart-inspector-tag">{timeframe}</span>
          </div>
        </div>
        <div className="inspector-stats">
          <InspectorStat label="Open" value={formatPrice(inspectedSummary?.open)} />
          <InspectorStat label="High" value={formatPrice(inspectedSummary?.high)} />
          <InspectorStat label="Low" value={formatPrice(inspectedSummary?.low)} />
          <InspectorStat label="Close" value={formatPrice(inspectedSummary?.close)} />
          <InspectorStat label="Volume" value={formatCompactInteger(inspectedSummary?.volume)} />
        </div>
      </div>
      <div className="chart-zoom-bar">
        <div className="zoom-actions">
          <button className="zoom-button" type="button" onClick={() => shiftWindow("left")} disabled={!canPanLeft}>
            ←
          </button>
          <button className="zoom-button" type="button" onClick={() => applyZoom(Math.floor(visibleCandles.length * 0.75), absoluteActiveIndex)} disabled={!canZoomIn}>
            +
          </button>
          <button className="zoom-button" type="button" onClick={() => applyZoom(Math.ceil(visibleCandles.length * 1.25), absoluteActiveIndex)} disabled={!canZoomOut}>
            −
          </button>
          <button className="zoom-button" type="button" onClick={() => shiftWindow("right")} disabled={!canPanRight}>
            →
          </button>
          <button
            className="zoom-button"
            type="button"
            onClick={() => applyZoom(initialViewportBars(candles.length, timeframe), candles.length - 1)}
            disabled={visibleCandles.length === initialViewportBars(candles.length, timeframe) && viewport.endIndex === candles.length - 1}
          >
            reset
          </button>
          {effectiveRange ? (
            <button className="zoom-button" type="button" onClick={clearSelection}>
              clear range
            </button>
          ) : null}
        </div>
        <p className="zoom-note">
          scale {visibleCandles.length} / {candles.length} bars, vertical wheel to zoom, horizontal scroll to move
        </p>
      </div>
      <div className="chart-scroll-frame">
        <div className="chart-scroll-viewport" ref={chartScrollRef} tabIndex={0} aria-label="Scrollable candlestick chart">
          <div className="chart-scroll-canvas" style={{ width: chartCanvasWidth }}>
            <svg
              className={`chart-svg ${brushState ? "chart-svg-selecting" : ""}`}
              viewBox={`0 0 ${width} ${height}`}
              preserveAspectRatio="none"
              role="img"
              aria-label="candlestick chart"
              onPointerMove={(event) => {
                if (brushState && brushState.pointerId === event.pointerId) {
                  const nextIndex = absoluteIndexFromEvent(event);
                  setHoverIndex(nextIndex);
                  setBrushState((current) =>
                    current
                      ? {
                          ...current,
                          currentIndex: nextIndex,
                        }
                      : current,
                  );
                  return;
                }
                handlePointerMove(event);
              }}
              onPointerDown={handlePointerDown}
              onPointerLeave={() => {
                if (!brushState) {
                  setHoverIndex(viewport.endIndex);
                }
              }}
              onPointerUp={handlePointerUp}
              onPointerCancel={(event) => {
                if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                  event.currentTarget.releasePointerCapture(event.pointerId);
                }
                setBrushState(null);
              }}
              onWheel={handleWheel}
            >
              {priceTicks.map((tick) => (
                <g key={tick.ratio}>
                  <line x1={padX} x2={width - padX - 48} y1={tick.y} y2={tick.y} className="chart-grid-line" />
                </g>
              ))}
            <line x1={padX} x2={width - padX - 48} y1={volumeTopY} y2={volumeTopY} className="chart-volume-divider" />
            {gapBands.map((band) => (
              <rect
                key={`${band.startX}-${band.endX}`}
                x={band.startX}
                y={padY}
                width={Math.max(band.endX - band.startX, 1)}
                height={priceBottomY - padY}
                className="chart-gap-band"
              />
            ))}
            {selectionBand ? (
              <rect
                x={selectionBand.startX}
                y={padY}
                width={Math.max(selectionBand.endX - selectionBand.startX, 1)}
                height={volumeBottomY - padY}
                className="chart-selection-band"
              />
            ) : null}
            {visibleCandles.map((candle, index) => {
              const x = xPositions[index];
              const tone = candle.close >= candle.open ? "up" : "down";
              const volumeHeight = Math.max(2, (candle.volume / maxVolume) * (volumeBottomY - volumeTopY));
              return (
                <rect
                  key={`${candle.timestamp}-volume`}
                  x={x - candleWidth / 2}
                  y={volumeBottomY - volumeHeight}
                  width={candleWidth}
                  height={volumeHeight}
                  rx={Math.min(candleWidth / 2, 2)}
                  className={`chart-volume-bar chart-volume-bar-${tone}`}
                />
              );
            })}
            {signalMarker ? (
              <g className={`chart-signal-marker chart-signal-marker-${signalMarker.tone}`}>
                <rect
                  x={signalMarker.rectX}
                  y={signalMarker.rectY}
                  width={signalMarker.width}
                  height={signalMarker.height}
                  rx={signalMarker.height / 2}
                  className="chart-signal-pill"
                />
                <text
                  x={signalMarker.rectX + signalMarker.width / 2}
                  y={signalMarker.rectY + signalMarker.height / 2 + 1}
                  className="chart-signal-text"
                >
                  {signalMarker.label}
                </text>
                <line
                  x1={signalMarker.anchorX}
                  x2={signalMarker.anchorX}
                  y1={signalMarker.rectY + signalMarker.height}
                  y2={signalMarker.anchorY}
                  className="chart-signal-stem"
                />
              </g>
            ) : null}
            <polyline points={closeLine} className="chart-close-line" />
            {visibleCandles.map((candle, index) => {
              const x = xPositions[index];
              const wickTop = scaleYInRange(candle.high, minPrice, span, padY, priceBottomY);
              const wickBottom = scaleYInRange(candle.low, minPrice, span, padY, priceBottomY);
              const openY = scaleYInRange(candle.open, minPrice, span, padY, priceBottomY);
              const closeY = scaleYInRange(candle.close, minPrice, span, padY, priceBottomY);
              const bodyTop = Math.min(openY, closeY);
              const bodyHeight = Math.max(Math.abs(openY - closeY), 2);
              const tone = candle.close >= candle.open ? "up" : "down";

              return (
                <g key={candle.timestamp}>
                  <line x1={x} x2={x} y1={wickTop} y2={wickBottom} className={`wick wick-${tone}`} />
                  <rect
                    x={x - candleWidth / 2}
                    y={bodyTop}
                    width={candleWidth}
                    height={bodyHeight}
                    rx={3}
                    className={`candle candle-${tone}`}
                  />
                </g>
              );
            })}
            {selectionBand ? (
              <>
                <line
                  x1={selectionBand.startX}
                  x2={selectionBand.startX}
                  y1={padY}
                  y2={volumeBottomY}
                  className="chart-selection-edge"
                />
                <line
                  x1={selectionBand.endX}
                  x2={selectionBand.endX}
                  y1={padY}
                  y2={volumeBottomY}
                  className="chart-selection-edge"
                />
              </>
            ) : (
              <line x1={activeX} x2={activeX} y1={padY} y2={volumeBottomY} className="chart-crosshair" />
            )}
            </svg>
          </div>
        </div>
        <div className="chart-y-axis-overlay" aria-hidden="true">
          {priceTicks.map((tick) => (
            <span
              key={tick.ratio}
              className="chart-y-axis-tick"
              style={{ top: `${(tick.y / height) * 100}%` }}
            >
              {formatPrice(tick.value)}
            </span>
          ))}
        </div>
      </div>
      <div className="chart-axis">
        <span>{formatAxisDate(visibleCandles[0]?.timestamp)}</span>
        <span>{formatAxisDate(visibleCandles[Math.floor(visibleCandles.length / 2)]?.timestamp)}</span>
        <span>{formatAxisDate(visibleCandles[visibleCandles.length - 1]?.timestamp)}</span>
      </div>
      <div className="chart-summary-grid">
        <ChartSummaryCard
          label="Scope"
          value={effectiveRange ? "Selection" : "Viewport"}
          meta={`${summaryCandles.length} bars`}
        />
        <ChartSummaryCard
          label="Net move"
          value={formatSignedPrice(rangeSummary?.delta)}
          meta={formatSignedPercent(rangeSummary?.deltaPct)}
          tone={toneFromDelta(rangeSummary?.delta)}
        />
        <ChartSummaryCard
          label="Range"
          value={formatRange(summaryCandles)}
          meta={`${formatPrice(rangeSummary?.low)} low · ${formatPrice(rangeSummary?.high)} high`}
        />
        <ChartSummaryCard
          label="Avg volume"
          value={formatCompactInteger(rangeSummary?.averageVolume)}
          meta={`${formatCompactInteger(rangeSummary?.volume)} total`}
        />
        <ChartSummaryCard
          label="Trading gaps"
          value={String(summaryGaps)}
          meta={effectiveRange ? "inside selection" : "inside viewport"}
        />
        <ChartSummaryCard
          label="Live signal"
          value={signal ? `${signal.direction} ${formatProbability(signal.probability)}` : "n/a"}
          meta={
            signal
              ? `threshold ${formatProbability(signal.threshold)} · ${signalMarker ? "on chart" : "off chart"}`
              : "runtime unavailable"
          }
          tone={badgeTone(signal?.direction)}
        />
      </div>
    </div>
  );
}

function InspectorStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="inspector-stat">
      <span className="inspector-stat-label">{label}</span>
      <strong className="inspector-stat-value">{value}</strong>
    </div>
  );
}

function ChartSummaryCard({
  label,
  value,
  meta,
  tone = "neutral",
}: {
  label: string;
  value: string;
  meta: string;
  tone?: "neutral" | "up" | "down" | "none";
}) {
  return (
    <div className={`chart-summary-card chart-summary-card-${tone}`}>
      <span className="chart-summary-label">{label}</span>
      <strong className="chart-summary-value">{value}</strong>
      <span className="chart-summary-meta">{meta}</span>
    </div>
  );
}

function ArtifactPreview({ document }: { document: ArtifactDocument | null }) {
  if (!document) {
    return <div className="chart-empty">No artifact documents loaded.</div>;
  }

  return (
    <article className="artifact-preview">
      <div className="artifact-preview-header">
        <div>
          <p className="artifact-preview-title">{document.title}</p>
          <p className="artifact-preview-path">{document.path}</p>
        </div>
        <span className="artifact-preview-badge">{document.contentType}</span>
      </div>
      <pre className="artifact-preview-content">{document.content}</pre>
    </article>
  );
}

function FactorChart({ factors }: { factors: FactorPoint[] }) {
  const grouped = groupFactors(factors);
  if (!grouped.length) {
    return <div className="chart-empty">No factor data loaded.</div>;
  }

  const width = 640;
  const height = 220;
  const padX = 18;
  const padY = 16;
  const xPositions = buildTimeScalePositions(
    grouped[0]?.points.map((point) => point.timestamp) ?? [],
    { width, padX },
  );

  return (
    <div className="factor-shell">
      <svg className="factor-svg" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="factor strip chart">
        {[0, 0.5, 1].map((ratio) => {
          const y = padY + (height - padY * 2) * ratio;
          return <line key={ratio} x1={padX} x2={width - padX} y1={y} y2={y} className="chart-grid-line" />;
        })}
        {grouped.map((series, index) => {
          const closes = series.points.map((point) => point.close);
          const min = Math.min(...closes);
          const max = Math.max(...closes);
          const span = Math.max(max - min, 0.0001);
          const points = series.points
            .map((point, pointIndex) => {
              const x = xPositions[pointIndex] ?? padX;
              const ratio = (point.close - min) / span;
              const y = height - padY - ratio * (height - padY * 2);
              return `${x},${y}`;
            })
            .join(" ");

          return (
            <polyline
              key={series.factor}
              points={points}
              className="factor-line"
              style={{ stroke: factorPalette[index % factorPalette.length] }}
            />
          );
        })}
      </svg>
      <div className="factor-legend">
        {grouped.map((series, index) => (
          <div key={series.factor} className="factor-chip">
            <span
              className="factor-swatch"
              style={{ backgroundColor: factorPalette[index % factorPalette.length] }}
            />
            <strong>{series.factor}</strong>
            <span>{formatPrice(series.points[series.points.length - 1]?.close)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function groupFactors(factors: FactorPoint[]) {
  const grouped = new Map<string, FactorPoint[]>();
  for (const point of factors) {
    const bucket = grouped.get(point.factor) ?? [];
    bucket.push(point);
    grouped.set(point.factor, bucket);
  }

  return Array.from(grouped.entries()).map(([factor, points]) => ({
    factor,
    points: [...points].sort((left, right) => left.timestamp.localeCompare(right.timestamp)),
  }));
}

function filterFactorsByRange(factors: FactorPoint[], range: FocusedChartRange | null) {
  if (!range) {
    return factors;
  }

  return factors.filter((point) => point.timestamp >= range.from && point.timestamp <= range.to);
}

function aggregateCandles(candles: CandleBar[], timeframe: ChartTimeframe) {
  const timeframeMs = timeframeToMs(timeframe);
  const baseTimeframeMs = timeframeToMs("5m");
  if (!candles.length || timeframeMs <= baseTimeframeMs) {
    return candles;
  }

  const buckets = new Map<number, CandleBar[]>();
  for (const candle of candles) {
    const bucketStart = floorTimestamp(Date.parse(candle.timestamp), timeframeMs);
    const bucket = buckets.get(bucketStart) ?? [];
    bucket.push(candle);
    buckets.set(bucketStart, bucket);
  }

  return Array.from(buckets.entries())
    .sort(([left], [right]) => left - right)
    .map(([bucketStart, bucket]) => {
      const ordered = [...bucket].sort((left, right) => left.timestamp.localeCompare(right.timestamp));
      return {
        timestamp: new Date(bucketStart).toISOString(),
        open: ordered[0]?.open ?? 0,
        high: Math.max(...ordered.map((item) => item.high)),
        low: Math.min(...ordered.map((item) => item.low)),
        close: ordered[ordered.length - 1]?.close ?? 0,
        volume: ordered.reduce((sum, item) => sum + item.volume, 0),
      };
    });
}

function aggregateFactors(factors: FactorPoint[], timeframe: ChartTimeframe) {
  const timeframeMs = timeframeToMs(timeframe);
  const baseTimeframeMs = timeframeToMs("5m");
  if (!factors.length || timeframeMs <= baseTimeframeMs) {
    return factors;
  }

  const buckets = new Map<string, FactorPoint[]>();
  for (const point of factors) {
    const bucketStart = floorTimestamp(Date.parse(point.timestamp), timeframeMs);
    const bucketKey = `${point.factor}:${bucketStart}`;
    const bucket = buckets.get(bucketKey) ?? [];
    bucket.push(point);
    buckets.set(bucketKey, bucket);
  }

  return Array.from(buckets.entries())
    .map(([bucketKey, bucket]) => {
      const [factor, bucketStart] = bucketKey.split(":");
      const ordered = [...bucket].sort((left, right) => left.timestamp.localeCompare(right.timestamp));
      return {
        factor,
        timestamp: new Date(Number(bucketStart)).toISOString(),
        close: ordered[ordered.length - 1]?.close ?? 0,
      };
    })
    .sort((left, right) => {
      if (left.timestamp === right.timestamp) {
        return left.factor.localeCompare(right.factor);
      }
      return left.timestamp.localeCompare(right.timestamp);
    });
}

function readInitialViewState(): InitialViewState {
  const params = new URLSearchParams(window.location.search);
  const timeframeParam = params.get("tf");
  const timeframeKeys = new Set<ChartTimeframe>(chartTimeframeOptions.map((option) => option.key));
  const chartTimeframe = timeframeKeys.has((timeframeParam ?? "") as ChartTimeframe)
    ? ((timeframeParam ?? "5m") as ChartTimeframe)
    : "5m";

  return {
    chartOnly: params.get("view") === "chart",
    assetId: params.get("asset"),
    chartTimeframe,
  };
}

function buildChartWindowUrl(assetId: string | null, chartTimeframe: ChartTimeframe) {
  const params = new URLSearchParams();
  params.set("view", "chart");
  if (assetId) {
    params.set("asset", assetId);
  }
  params.set("tf", chartTimeframe);
  return `${window.location.pathname}?${params.toString()}`;
}

function scaleY(value: number, min: number, span: number, height: number, pad: number) {
  return height - pad - ((value - min) / span) * (height - pad * 2);
}

function scaleYInRange(value: number, min: number, span: number, top: number, bottom: number) {
  return bottom - ((value - min) / span) * (bottom - top);
}

function buildTimeScalePositions(
  timestamps: string[],
  { width, padX }: { width: number; padX: number },
) {
  if (!timestamps.length) {
    return [];
  }
  const start = Date.parse(timestamps[0]);
  const end = Date.parse(timestamps[timestamps.length - 1]);
  const span = Math.max(end - start, 1);

  return timestamps.map((timestamp) => {
    const ratio = (Date.parse(timestamp) - start) / span;
    return padX + ratio * (width - padX * 2);
  });
}

function findClosestXIndex(positions: number[], x: number) {
  let bestIndex = 0;
  let bestDistance = Number.POSITIVE_INFINITY;

  for (const [index, position] of positions.entries()) {
    const distance = Math.abs(position - x);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = index;
    }
  }

  return bestIndex;
}

function findClosestTimestampIndex(candles: CandleBar[], timestampMs: number) {
  let bestIndex = 0;
  let bestDistance = Number.POSITIVE_INFINITY;

  for (const [index, candle] of candles.entries()) {
    const distance = Math.abs(Date.parse(candle.timestamp) - timestampMs);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = index;
    }
  }

  return bestIndex;
}

function buildGapBands(candles: CandleBar[], xPositions: number[], timeframe: ChartTimeframe) {
  const bands: Array<{ startX: number; endX: number }> = [];
  const gapThresholdMs = gapThresholdForTimeframe(timeframe);
  for (let index = 1; index < candles.length; index += 1) {
    const previous = Date.parse(candles[index - 1].timestamp);
    const current = Date.parse(candles[index].timestamp);
    if (current - previous > gapThresholdMs) {
      bands.push({
        startX: xPositions[index - 1],
        endX: xPositions[index],
      });
    }
  }
  return bands;
}

function normalizeIndexRange(left: number, right: number) {
  return {
    startIndex: Math.min(left, right),
    endIndex: Math.max(left, right),
  };
}

function summarizeCandles(candles: CandleBar[]) {
  if (!candles.length) {
    return undefined;
  }

  return {
    open: candles[0]?.open,
    high: Math.max(...candles.map((item) => item.high)),
    low: Math.min(...candles.map((item) => item.low)),
    close: candles[candles.length - 1]?.close,
    volume: candles.reduce((sum, item) => sum + item.volume, 0),
  };
}

function summarizeRange(candles: CandleBar[]) {
  if (!candles.length) {
    return undefined;
  }

  const summary = summarizeCandles(candles);
  if (!summary) {
    return undefined;
  }

  const delta = summary.close - summary.open;
  const deltaPct = summary.open !== 0 ? delta / summary.open : undefined;
  return {
    ...summary,
    bars: candles.length,
    delta,
    deltaPct,
    averageVolume: summary.volume / candles.length,
  };
}

function buildSelectionBand(
  selection: { startIndex: number; endIndex: number },
  viewportStartIndex: number,
  xPositions: number[],
  candleWidth: number,
) {
  const startIndex = selection.startIndex - viewportStartIndex;
  const endIndex = selection.endIndex - viewportStartIndex;
  if (startIndex < 0 || endIndex >= xPositions.length) {
    return null;
  }

  return {
    startX: xPositions[startIndex] - candleWidth / 2,
    endX: xPositions[endIndex] + candleWidth / 2,
  };
}

function buildSignalMarker(
  signal: SignalCard | undefined,
  candles: CandleBar[],
  timeframe: ChartTimeframe,
  xPositions: number[],
  width: number,
  minPrice: number,
  span: number,
  priceBottomY: number,
  padX: number,
  padY: number,
) {
  if (!signal || !candles.length) {
    return null;
  }

  const targetTimestamp = Date.parse(signal.asOfTime);
  if (Number.isNaN(targetTimestamp)) {
    return null;
  }

  const index = findClosestTimestampIndex(candles, targetTimestamp);
  const candleTimestamp = Date.parse(candles[index]?.timestamp ?? "");
  const maxDistanceMs = Math.max(timeframeToMs(timeframe) * 2, 30 * 60 * 1000);
  if (Number.isNaN(candleTimestamp) || Math.abs(candleTimestamp - targetTimestamp) > maxDistanceMs) {
    return null;
  }

  const label = `${signal.direction.toUpperCase()} ${formatProbability(signal.probability)}`;
  const anchorX = xPositions[index] ?? padX;
  const anchorY = Math.max(
    scaleYInRange(candles[index]?.high ?? candles[index]?.close ?? 0, minPrice, span, padY, priceBottomY) - 4,
    padY + 22,
  );
  const pillHeight = 24;
  const pillWidth = Math.max(76, label.length * 7.4 + 18);
  const rectX = clampNumber(anchorX - pillWidth / 2, padX, width - padX - pillWidth);
  const rectY = clampNumber(anchorY - 34, padY + 4, priceBottomY - pillHeight - 6);

  return {
    label,
    tone: badgeTone(signal.direction),
    anchorX,
    anchorY,
    rectX,
    rectY,
    width: pillWidth,
    height: pillHeight,
  };
}

function clampViewport(windowSize: number, windowEndIndex: number, length: number) {
  const normalizedSize = Math.max(Math.min(windowSize, length), Math.min(24, length));
  const normalizedEnd = Math.min(Math.max(windowEndIndex, normalizedSize - 1), length - 1);
  const startIndex = Math.max(normalizedEnd - normalizedSize + 1, 0);
  return {
    startIndex,
    endIndex: startIndex + normalizedSize - 1,
  };
}

function maxViewportBars(length: number) {
  return Math.min(length, 1440);
}

function initialViewportBars(length: number, timeframe: ChartTimeframe) {
  if (!length) {
    return 0;
  }

  const targetByTimeframe: Record<ChartTimeframe, number> = {
    "5m": 180,
    "15m": 180,
    "1h": 160,
    "4h": 140,
    "1d": 120,
  };

  return Math.min(length, targetByTimeframe[timeframe], maxViewportBars(length));
}

function chartScrollCanvasWidth(length: number, timeframe: ChartTimeframe) {
  const bars = initialViewportBars(length, timeframe);
  return Math.max(1320, Math.ceil(bars * 8 + 120));
}

function minimumAdjacentGap(positions: number[]) {
  if (positions.length < 2) {
    return 10;
  }

  let minGap = Number.POSITIVE_INFINITY;
  for (let index = 1; index < positions.length; index += 1) {
    minGap = Math.min(minGap, positions[index] - positions[index - 1]);
  }
  return Number.isFinite(minGap) ? Math.max(minGap, 0.8) : 10;
}

function countTradingGaps(candles: CandleBar[], timeframe: ChartTimeframe) {
  let total = 0;
  const gapThresholdMs = gapThresholdForTimeframe(timeframe);
  for (let index = 1; index < candles.length; index += 1) {
    const previous = Date.parse(candles[index - 1].timestamp);
    const current = Date.parse(candles[index].timestamp);
    if (current - previous > gapThresholdMs) {
      total += 1;
    }
  }
  return total;
}

function timeframeToMs(timeframe: ChartTimeframe) {
  const option = chartTimeframeOptions.find((item) => item.key === timeframe);
  return (option?.minutes ?? 5) * 60_000;
}

function gapThresholdForTimeframe(timeframe: ChartTimeframe) {
  return Math.max(60 * 60 * 1000, timeframeToMs(timeframe) * 3);
}

function floorTimestamp(timestampMs: number, timeframeMs: number) {
  return Math.floor(timestampMs / timeframeMs) * timeframeMs;
}

function badgeTone(direction: string | undefined) {
  if (direction === "up") {
    return "up";
  }
  if (direction === "down") {
    return "down";
  }
  return "none";
}

function formatClassLabel(value: string) {
  return value.split("_").join(" ");
}

function thresholdDeltaTone(signal: SignalCard | null | undefined, policy: ProductionPolicySnapshot | undefined) {
  const policyThreshold = signal?.policy?.threshold ?? policy?.threshold;
  if (signal?.threshold === undefined || policyThreshold === undefined) {
    return "neutral";
  }
  return Math.abs(signal.threshold - policyThreshold) <= 0.001 ? "good" : "warn";
}

function formatPolicyStatus(status: string | undefined) {
  if (!status) {
    return "n/a";
  }
  const labels: Record<string, string> = {
    production_candidate: "candidate",
    calibration_review: "calibration",
    research_review: "research",
    incomplete: "incomplete",
  };
  return labels[status] ?? status.replace(/_/g, " ");
}

function formatDate(value: string | undefined) {
  if (!value) {
    return "n/a";
  }
  return new Date(value).toLocaleString("ru-RU", {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatSelectionLabel(candles: CandleBar[]) {
  if (!candles.length) {
    return "n/a";
  }
  if (candles.length === 1) {
    return formatDate(candles[0]?.timestamp);
  }
  return `${formatDate(candles[0]?.timestamp)} - ${formatDate(candles[candles.length - 1]?.timestamp)}`;
}

function formatAxisDate(value: string | undefined) {
  if (!value) {
    return "n/a";
  }
  return new Date(value).toLocaleString("ru-RU", {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatPrice(value: number | undefined) {
  if (value === undefined || Number.isNaN(value)) {
    return "n/a";
  }
  return value.toFixed(2);
}

function formatSignedPrice(value: number | undefined) {
  if (value === undefined || Number.isNaN(value)) {
    return "n/a";
  }
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  return `${sign}${Math.abs(value).toFixed(2)}`;
}

function formatSignedPercent(value: number | undefined) {
  if (value === undefined || Number.isNaN(value)) {
    return "n/a";
  }
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  return `${sign}${(Math.abs(value) * 100).toFixed(2)}%`;
}

function formatCompactInteger(value: number | undefined) {
  if (value === undefined || Number.isNaN(value)) {
    return "n/a";
  }
  return new Intl.NumberFormat("ru-RU", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);
}

function formatUnknownMetric(value: unknown) {
  return typeof value === "number" ? String(value) : "n/a";
}

function formatRange(candles: CandleBar[]) {
  if (!candles.length) {
    return "n/a";
  }
  const low = Math.min(...candles.map((item) => item.low));
  const high = Math.max(...candles.map((item) => item.high));
  return `${low.toFixed(2)} / ${high.toFixed(2)}`;
}

function toneFromDelta(value: number | undefined): "up" | "down" | "none" {
  if (value === undefined || Number.isNaN(value) || value === 0) {
    return "none";
  }
  return value > 0 ? "up" : "down";
}

function clampNumber(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}

function totalRows(overview: WorkspaceShellData["mlOverview"] | undefined) {
  return (
    (overview?.dataset?.trainRows ?? 0) +
    (overview?.dataset?.valRows ?? 0) +
    (overview?.dataset?.testRows ?? 0)
  );
}

function baseName(path: string) {
  const normalized = path.split("\\").join("/");
  const parts = normalized.split("/");
  return parts[parts.length - 1] ?? path;
}
