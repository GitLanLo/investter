import React, { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { ActionType, dispose, init, type Chart as KLineChart, type KLineData } from "klinecharts";
import {
  Bell,
  BookOpen,
  BrainCircuit,
  ChevronLeft,
  ChevronRight,
  Crosshair,
  Eraser,
  Grid3X3,
  Layers,
  LineChart,
  LocateFixed,
  Maximize2,
  MessageSquare,
  Minus,
  PanelRight,
  PencilLine,
  Plus,
  Ruler,
  Save,
  Settings2,
  Sigma,
  SlidersHorizontal,
  Square,
  TrendingUp,
  Undo2,
  Redo2,
  RefreshCw,
  Trash2,
} from "lucide-react";

export interface ChartCandle {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

interface ChartProps {
  data: ChartCandle[];
  height?: number | string;
  emptyText?: string;
  timeframe?: string;
  timeframeOptions?: Array<{ value: string; label: string }>;
  onTimeframeChange?: (value: string) => void;
  onRefresh?: () => void;
  refreshing?: boolean;
  storageKey?: string;
  instrumentLabel?: string;
  marketLabel?: string;
  inWatchlist?: boolean;
  onToggleWatchlist?: () => void;
  onCreateAlert?: () => void;
  onBuyInstrument?: () => void;
  buyInstrumentDisabled?: boolean;
  alerts?: Array<{ id: number; condition: string; action: string; active: boolean }>;
  onDeleteAlert?: (id: number) => void;
  watchlist?: Array<{ assetId: string; ticker: string; lastPrice: number; priceChange: number }>;
  onSelectAsset?: (assetId: string) => void;
  forecast?: {
    title: string;
    description: string;
    probability?: string;
    history?: Array<{ id?: number; date: string; state: string; direction: string; probability: string; model?: string }>;
    running?: boolean;
    disabled?: boolean;
    onRun?: () => void;
  };
}

type IndicatorName = "MA" | "EMA" | "BOLL" | "VOL" | "MACD" | "RSI";
type ChartType = "candle_solid" | "candle_stroke" | "ohlc" | "area";
type SidePanel = "forecast" | "watchlist" | "alerts" | "data";

interface SavedDrawing {
  name: string;
  points: Array<{ timestamp?: number; dataIndex?: number; value?: number }>;
}

interface ChartSettings {
  indicators: IndicatorName[];
  drawings: SavedDrawing[];
  chartType?: ChartType;
}

const DEFAULT_INDICATORS: IndicatorName[] = ["MA", "VOL"];
const PRICE_INDICATORS = new Set<IndicatorName>(["MA", "EMA", "BOLL"]);
const DRAWING_GROUP_ID = "user-drawing";
const INDICATOR_OPTIONS: Array<{ name: IndicatorName; label: string; pane: "price" | "volume" | "oscillator" }> = [
  { name: "MA", label: "MA", pane: "price" },
  { name: "EMA", label: "EMA", pane: "price" },
  { name: "BOLL", label: "BOLL", pane: "price" },
  { name: "VOL", label: "VOL", pane: "volume" },
  { name: "MACD", label: "MACD", pane: "oscillator" },
  { name: "RSI", label: "RSI", pane: "oscillator" },
];
const CHART_TYPE_OPTIONS: Array<{ value: ChartType; label: string; description: string }> = [
  { value: "candle_solid", label: "Японские свечи", description: "Основной режим для анализа цены" },
  { value: "candle_stroke", label: "Пустые свечи", description: "Меньше визуального шума" },
  { value: "ohlc", label: "Бары", description: "Компактное OHLC-представление" },
  { value: "area", label: "Область", description: "Тренд закрытия без деталей свечи" },
];
const DRAWING_MENU_TOOLS = [
  { name: "crosshair", label: "Курсор", icon: Crosshair },
  { name: "segment", label: "Тренд", icon: LineChart },
  { name: "horizontalStraightLine", label: "Уровень", icon: Ruler },
  { name: "priceChannelLine", label: "Канал", icon: Layers },
  { name: "fibonacciLine", label: "Fibo", icon: SlidersHorizontal },
  { name: "simpleAnnotation", label: "Заметка", icon: PencilLine },
];
const RANGE_OPTIONS = [
  { label: "1M", days: 31 },
  { label: "3M", days: 92 },
  { label: "6M", days: 183 },
  { label: "1Y", days: 366 },
  { label: "All", days: null },
];

function toKLineData(candle: ChartCandle): KLineData {
  return {
    timestamp: new Date(candle.timestamp).getTime(),
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
    volume: candle.volume,
  };
}

function fromKLineData(item?: KLineData | null): ChartCandle | null {
  if (!item) return null;
  return {
    timestamp: new Date(item.timestamp).toISOString(),
    open: Number(item.open),
    high: Number(item.high),
    low: Number(item.low),
    close: Number(item.close),
    volume: Number(item.volume || 0),
  };
}

function formatDate(value?: string) {
  if (!value) return "n/a";
  return new Date(value).toLocaleString("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatPrice(value: number) {
  if (!Number.isFinite(value)) return "n/a";
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 }).format(value);
}

function formatVolume(value: number) {
  if (!Number.isFinite(value)) return "0";
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 0 }).format(value);
}

export const Chart: React.FC<ChartProps> = ({
  data,
  height = 520,
  emptyText = "Нет свечей для выбранного инструмента",
  timeframe = "5m",
  timeframeOptions = [],
  onTimeframeChange,
  onRefresh,
  refreshing = false,
  storageKey = "default",
  instrumentLabel = "Инструмент",
  marketLabel = "MOEX",
  inWatchlist,
  onToggleWatchlist,
  onCreateAlert,
  onBuyInstrument,
  buyInstrumentDisabled = false,
  alerts = [],
  onDeleteAlert,
  watchlist = [],
  onSelectAsset,
  forecast,
}) => {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<KLineChart | null>(null);
  const indicatorPanesRef = useRef<Map<IndicatorName, string>>(new Map());
  const drawingsRef = useRef<Map<string, SavedDrawing>>(new Map());
  const [selected, setSelected] = useState<ChartCandle | null>(null);
  const [activeIndicators, setActiveIndicators] = useState<IndicatorName[]>(DEFAULT_INDICATORS);
  const [chartType, setChartType] = useState<ChartType>("candle_solid");
  const [openMenu, setOpenMenu] = useState<"chartType" | "timeframe" | "indicators" | "drawings" | null>(null);
  const [sidePanel, setSidePanel] = useState<SidePanel>("forecast");
  const [portalTarget, setPortalTarget] = useState<Element | null>(null);

  useEffect(() => {
    setPortalTarget(document.getElementById("chart-topbar-portal"));
  }, []);

  const klineData = useMemo(() => data.map(toKLineData), [data]);
  const chartStorageKey = `invest.chart.${storageKey}`;
  const fallbackSelected = data.length > 0 ? data[data.length - 1] : null;
  const selectedCandle = selected || fallbackSelected;
  const change = selectedCandle ? selectedCandle.close - selectedCandle.open : 0;
  const changePct = selectedCandle && selectedCandle.open !== 0 ? (change / selectedCandle.open) * 100 : 0;
  const range = selectedCandle ? selectedCandle.high - selectedCandle.low : 0;
  const rangePct = selectedCandle && selectedCandle.low !== 0 ? (range / selectedCandle.low) * 100 : 0;
  const changeClass = change >= 0 ? "positive" : "negative";

  const persistSettings = (indicators = activeIndicators, type = chartType) => {
    const settings: ChartSettings = {
      indicators,
      chartType: type,
      drawings: Array.from(drawingsRef.current.values()),
    };
    window.localStorage.setItem(chartStorageKey, JSON.stringify(settings));
  };

  const loadSettings = (): ChartSettings => {
    try {
      const raw = window.localStorage.getItem(chartStorageKey);
      if (!raw) return { indicators: DEFAULT_INDICATORS, drawings: [], chartType: "candle_solid" };
      const parsed = JSON.parse(raw) as Partial<ChartSettings>;
      return {
        indicators: (parsed.indicators || DEFAULT_INDICATORS).filter((name): name is IndicatorName =>
          INDICATOR_OPTIONS.some((option) => option.name === name),
        ),
        drawings: Array.isArray(parsed.drawings) ? parsed.drawings.filter((drawing) => drawing.name && Array.isArray(drawing.points)) : [],
        chartType: parsed.chartType || "candle_solid",
      };
    } catch {
      return { indicators: DEFAULT_INDICATORS, drawings: [], chartType: "candle_solid" };
    }
  };

  const createDrawing = (name: string, points?: SavedDrawing["points"]) => {
    const chart = chartRef.current;
    if (!chart) return;
    const overlay = {
      name,
      groupId: DRAWING_GROUP_ID,
      points,
      mode: "weak_magnet",
      modeSensitivity: 8,
      styles: {
        line: { color: "#2f6077", size: 2 },
        polygon: { color: "rgba(47, 96, 119, 0.12)" },
        text: { color: "#121b1f", size: 12, backgroundColor: "rgba(255, 255, 255, 0.9)" },
      },
      onDrawEnd: (event: any) => {
        if (event?.overlay?.id) {
          drawingsRef.current.set(event.overlay.id, { name: event.overlay.name, points: event.overlay.points || [] });
          persistSettings();
        }
        return true;
      },
      onPressedMoveEnd: (event: any) => {
        if (event?.overlay?.id) {
          drawingsRef.current.set(event.overlay.id, { name: event.overlay.name, points: event.overlay.points || [] });
          persistSettings();
        }
        return true;
      },
      onRemoved: (event: any) => {
        if (event?.overlay?.id) {
          drawingsRef.current.delete(event.overlay.id);
          persistSettings();
        }
        return true;
      },
    };
    const id = chart.createOverlay(overlay as any);
    if (typeof id === "string" && points?.length) {
      drawingsRef.current.set(id, { name, points });
    }
  };

  const applyIndicators = (chart: KLineChart, indicators: IndicatorName[]) => {
    indicatorPanesRef.current.forEach((paneId, name) => {
      if (!indicators.includes(name)) {
        chart.removeIndicator(paneId, name);
      }
    });
    indicatorPanesRef.current.clear();

    indicators.forEach((name) => {
      if (PRICE_INDICATORS.has(name)) {
        chart.createIndicator(name, false, { id: "candle_pane" });
        indicatorPanesRef.current.set(name, "candle_pane");
        return;
      }
      const paneId = `indicator_${name}`;
      chart.createIndicator(name, false, { id: paneId, height: name === "VOL" ? 96 : 118 });
      indicatorPanesRef.current.set(name, paneId);
    });
  };

  useEffect(() => {
    const container = chartContainerRef.current;
    if (!container) return;

    const chart = init(container, {
      locale: "ru-RU",
      timezone: "Europe/Moscow",
      styles: {
        grid: {
          horizontal: { color: "#edf1f0" },
          vertical: { color: "#f3f5f4" },
        },
        candle: {
          bar: {
            upColor: "#12835b",
            downColor: "#c24132",
            noChangeColor: "#66717a",
            upBorderColor: "#12835b",
            downBorderColor: "#c24132",
            noChangeBorderColor: "#66717a",
            upWickColor: "#12835b",
            downWickColor: "#c24132",
            noChangeWickColor: "#66717a",
          },
          tooltip: {
            showRule: "none",
            showType: "rect",
          },
        },
        xAxis: { axisLine: { color: "#dbe2e0" }, tickText: { color: "#66717a" } },
        yAxis: { axisLine: { color: "#dbe2e0" }, tickText: { color: "#66717a" } },
        crosshair: {
          horizontal: { line: { color: "#315c74", style: "dashed" }, text: { backgroundColor: "#315c74" } },
          vertical: { line: { color: "#315c74", style: "dashed" }, text: { backgroundColor: "#315c74" } },
        },
      },
    } as any);

    if (!chart) return;
    chartRef.current = chart;
    chart.setPriceVolumePrecision(4, 0);
    const settings = loadSettings();
    setActiveIndicators(settings.indicators);
    setChartType(settings.chartType || "candle_solid");
    chart.setStyles({ candle: { type: settings.chartType || "candle_solid" } } as any);
    drawingsRef.current.clear();
    applyIndicators(chart, settings.indicators);
    settings.drawings.forEach((drawing) => createDrawing(drawing.name, drawing.points));

    const handleCrosshairChange = (payload?: { kLineData?: KLineData }) => {
      const candle = fromKLineData(payload?.kLineData);
      if (candle) setSelected(candle);
    };

    chart.subscribeAction(ActionType.OnCrosshairChange, handleCrosshairChange);

    return () => {
      chart.unsubscribeAction(ActionType.OnCrosshairChange, handleCrosshairChange);
      chartRef.current = null;
      dispose(chart);
    };
  }, []);

  useEffect(() => {
    const settings = loadSettings();
    setActiveIndicators(settings.indicators);
    drawingsRef.current.clear();
    const chart = chartRef.current;
    if (!chart) return;
    chart.removeOverlay({ groupId: DRAWING_GROUP_ID });
    applyIndicators(chart, settings.indicators);
    settings.drawings.forEach((drawing) => createDrawing(drawing.name, drawing.points));
  }, [chartStorageKey]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    chart.applyNewData(klineData);
    chart.scrollToRealTime();
    setSelected(fallbackSelected);
  }, [fallbackSelected, klineData]);

  const scroll = (distance: number) => {
    chartRef.current?.scrollByDistance(distance, 160);
  };

  const zoom = (scale: number) => {
    const width = chartContainerRef.current?.clientWidth || 0;
    const zoomHeight = typeof height === "number" ? height : chartContainerRef.current?.clientHeight || 520;
    chartRef.current?.zoomAtCoordinate(scale, { x: Math.max(width / 2, 160), y: Math.floor(zoomHeight / 2) }, 160);
  };

  const toggleIndicator = (name: IndicatorName) => {
    const chart = chartRef.current;
    let next: IndicatorName[];

    if (PRICE_INDICATORS.has(name)) {
      // For price indicators, enforce single selection (radio-style among price indicators)
      if (activeIndicators.includes(name)) {
        next = activeIndicators.filter((item) => item !== name);
      } else {
        next = [...activeIndicators.filter((item) => !PRICE_INDICATORS.has(item)), name];
      }
    } else {
      // For pane indicators (VOL, MACD, RSI), allow multiple parallel indicators
      if (activeIndicators.includes(name)) {
        next = activeIndicators.filter((item) => item !== name);
      } else {
        next = [...activeIndicators, name];
      }
    }

    setActiveIndicators(next);
    if (chart) applyIndicators(chart, next);
    persistSettings(next);
  };

  const scrollToRange = (days: number | null) => {
    const chart = chartRef.current;
    if (!chart || klineData.length === 0) return;
    if (!days) {
      chart.scrollToDataIndex(0, 180);
      return;
    }
    const last = klineData[klineData.length - 1];
    const target = Number(last.timestamp) - days * 24 * 60 * 60 * 1000;
    chart.scrollToTimestamp(target, 180);
  };

  const clearDrawings = () => {
    chartRef.current?.removeOverlay({ groupId: DRAWING_GROUP_ID });
    drawingsRef.current.clear();
    persistSettings();
  };

  const changeChartType = (value: ChartType) => {
    setChartType(value);
    setOpenMenu(null);
    chartRef.current?.setStyles({ candle: { type: value } } as any);
    persistSettings(activeIndicators, value);
  };

  const activeChartType = CHART_TYPE_OPTIONS.find((option) => option.value === chartType) || CHART_TYPE_OPTIONS[0];
  const availableTimeframes = timeframeOptions.length > 0
    ? timeframeOptions
    : [
        { value: "1m", label: "1м" },
        { value: "5m", label: "5м" },
        { value: "15m", label: "15м" },
        { value: "1h", label: "1ч" },
        { value: "1d", label: "1д" },
      ];

  const topbarContent = (
    <>
      <div className="chart-topbar-actions" style={{ flex: 1 }}>
        <div className="chart-menu-root">
          <button type="button" className={openMenu === "chartType" ? "active" : ""} onClick={() => setOpenMenu(openMenu === "chartType" ? null : "chartType")}>
            Вид: {activeChartType.label}
          </button>
          {openMenu === "chartType" && (
            <div className="chart-popover chart-type-menu">
              {CHART_TYPE_OPTIONS.map((option) => (
                <button key={option.value} type="button" className={chartType === option.value ? "active" : ""} onClick={() => changeChartType(option.value)}>
                  <span>{option.label}</span>
                  <small>{option.description}</small>
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="chart-menu-root">
          <button type="button" className={openMenu === "timeframe" ? "active" : ""} onClick={() => setOpenMenu(openMenu === "timeframe" ? null : "timeframe")}>
            Период: {availableTimeframes.find((item) => item.value === timeframe)?.label || timeframe}
          </button>
          {openMenu === "timeframe" && (
            <div className="chart-popover chart-timeframe-menu">
              {availableTimeframes.map((item) => (
                <button
                  key={item.value}
                  type="button"
                  className={timeframe === item.value ? "active" : ""}
                  onClick={() => {
                    onTimeframeChange?.(item.value);
                    setOpenMenu(null);
                  }}
                >
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="chart-menu-root">
          <button type="button" className={openMenu === "indicators" ? "active" : ""} onClick={() => setOpenMenu(openMenu === "indicators" ? null : "indicators")}>
            <Sigma size={18} aria-hidden="true" />
            Индикаторы
          </button>
          {openMenu === "indicators" && (
            <div className="chart-popover chart-indicator-menu">
              {INDICATOR_OPTIONS.map((indicator) => (
                <button
                  key={indicator.name}
                  type="button"
                  className={activeIndicators.includes(indicator.name) ? "active" : ""}
                  onClick={() => toggleIndicator(indicator.name)}
                >
                  <span>{indicator.label}</span>
                  <small>{indicator.pane === "price" ? "На графике" : "Отдельная панель"}</small>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
      <div className="chart-window-actions">
        {onRefresh && (
          <button type="button" onClick={onRefresh} disabled={refreshing} title="Обновить свечи">
            <RefreshCw size={18} aria-hidden="true" />
            <span>{refreshing ? "Загрузка" : "Обновить"}</span>
          </button>
        )}
      </div>
    </>
  );

  return (
    <div className={`chart-wrap chart-terminal ${portalTarget ? 'portal-active' : ''}`}>
      {portalTarget ? createPortal(topbarContent, portalTarget) : (
        <div className="chart-terminal-topbar">
          {topbarContent}
        </div>
      )}

      <div className="chart-terminal-body">
        <aside className="chart-left-rail" aria-label="Инструменты рисования">
          {DRAWING_MENU_TOOLS.map((tool) => {
            const Icon = tool.icon;
            return (
              <button
                key={tool.name}
                type="button"
                className="drawing-tool-btn"
                title={tool.label}
                onClick={() => {
                  if (tool.name !== "crosshair") createDrawing(tool.name);
                }}
              >
                <Icon size={20} aria-hidden="true" />
              </button>
            );
          })}
          <div className="rail-spacer" style={{ flexGrow: 1 }} />
          <button type="button" className="drawing-tool-btn danger" title="Очистить разметку" onClick={clearDrawings}>
            <Eraser size={20} aria-hidden="true" />
          </button>
        </aside>

        <main className="chart-main-stage">
          <div className="chart-floating-legend">
            <div className="chart-legend-symbol">
              <strong>{instrumentLabel}</strong>
              <span>{marketLabel}</span>
            </div>
            <span>ОТКР <strong>{formatPrice(selectedCandle?.open ?? NaN)}</strong></span>
            <span>МАКС <strong>{formatPrice(selectedCandle?.high ?? NaN)}</strong></span>
            <span>МИН <strong>{formatPrice(selectedCandle?.low ?? NaN)}</strong></span>
            <span>ЗАКР <strong>{formatPrice(selectedCandle?.close ?? NaN)}</strong></span>
            <span className={changeClass}>{change >= 0 ? "+" : ""}{formatPrice(change)} ({changePct.toFixed(2)}%)</span>
          </div>

          <div className="chart-toolbar" aria-label="Управление графиком">
            <button type="button" onClick={() => scroll(180)} title="Прокрутить влево">
              <ChevronLeft size={16} aria-hidden="true" />
            </button>
            <button type="button" onClick={() => scroll(-180)} title="Прокрутить вправо">
              <ChevronRight size={16} aria-hidden="true" />
            </button>
            <button type="button" onClick={() => zoom(0.82)} title="Уменьшить масштаб">
              <Minus size={16} aria-hidden="true" />
            </button>
            <button type="button" onClick={() => zoom(1.18)} title="Увеличить масштаб">
              <Plus size={16} aria-hidden="true" />
            </button>
            <button type="button" onClick={() => chartRef.current?.scrollToRealTime(160)} title="К последней свече">
              <LocateFixed size={16} aria-hidden="true" />
            </button>
            {onBuyInstrument && (
              <button type="button" className="chart-toolbar-buy" onClick={onBuyInstrument} disabled={buyInstrumentDisabled} title="Купить инструмент">
                <TrendingUp size={16} aria-hidden="true" />
                <span>Купить</span>
              </button>
            )}
          </div>

          <div ref={chartContainerRef} className="chart-surface kline-chart-surface" />
          {data.length === 0 && <div className="chart-empty">{emptyText}</div>}
        </main>

        <aside className="chart-side-panel">
          {sidePanel === "forecast" && (
            <section className="chart-forecast-panel">
              <div className="chart-side-header">
                <strong>ML Прогноз</strong>
                <BrainCircuit size={18} aria-hidden="true" />
              </div>
              <div className="chart-forecast-hero">
                <span>AI модуль</span>
                <h3>{forecast?.title || "Прогноз еще не запускался"}</h3>
                <p>{forecast?.description || "Запустите прогноз по текущим свечам, чтобы оценить вероятность движения."}</p>
                {forecast?.probability && <strong>{forecast.probability}</strong>}
                <button type="button" onClick={forecast?.onRun} disabled={forecast?.running || forecast?.disabled}>
                  <BrainCircuit size={17} aria-hidden="true" />
                  {forecast?.running ? "Считаем..." : "Построить прогноз"}
                </button>
              </div>
              <div className="chart-forecast-history">
                <span>История</span>
                {(forecast?.history || []).map((item, index) => (
                  <article key={item.id ?? index}>
                    <div>
                      <strong>{item.direction || "n/a"}</strong>
                      <small>{item.date}</small>
                    </div>
                    <em>{item.probability}</em>
                  </article>
                ))}
                {(forecast?.history || []).length === 0 && <p>Истории прогнозов пока нет.</p>}
              </div>
            </section>
          )}
          {sidePanel === "watchlist" && (
            <section className="chart-watchlist-panel">
              <div className="chart-side-header">
                <strong>Список</strong>
                <button type="button" onClick={onToggleWatchlist} title={inWatchlist ? "Убрать из списка" : "Добавить в список"}>
                  {inWatchlist ? <Minus size={16} aria-hidden="true" /> : <Plus size={16} aria-hidden="true" />}
                </button>
              </div>
              <div className="chart-watchlist-columns">
                <span>Инструмент</span>
                <span>Посл. цена</span>
                <span>Изм. (24ч)</span>
              </div>
              <div className="chart-watchlist-list">
                {watchlist.map((item) => {
                  const change = item.priceChange;
                  const prevPrice = item.lastPrice - change;
                  const changePct = prevPrice !== 0 ? (change / prevPrice) * 100 : 0;
                  const cClass = change >= 0 ? "positive" : "negative";
                  const currentTicker = instrumentLabel.split(" · ")[0];
                  
                  return (
                    <div 
                      key={item.assetId} 
                      className={`chart-watchlist-row ${item.ticker === currentTicker ? "active" : ""}`}
                      onClick={() => onSelectAsset?.(item.assetId)}
                    >
                      <span>{item.ticker}</span>
                      <strong>{formatPrice(item.lastPrice)}</strong>
                      <em className={cClass}>{change >= 0 ? "+" : ""}{changePct.toFixed(2)}%</em>
                    </div>
                  );
                })}
                {watchlist.length === 0 && (
                  <p className="chart-side-empty">
                    Список пуст. Добавьте инструменты для быстрого доступа.
                  </p>
                )}
              </div>
            </section>
          )}
          {sidePanel === "alerts" && (
            <section className="chart-alerts-panel">
              <div className="chart-side-header has-list">
                <strong>Уведомления</strong>
                <button type="button" onClick={onCreateAlert} title="Создать уведомление">
                  <Plus size={16} aria-hidden="true" />
                </button>
              </div>

              {alerts && alerts.length > 0 ? (
                <div className="chart-alerts-list">
                  {alerts.map(alert => (
                    <article key={alert.id} className="chart-alert-item">
                      <div className="chart-alert-info">
                        <strong>{alert.condition}</strong>
                        <span>{alert.action}</span>
                      </div>
                      <button type="button" className="icon-button small danger-icon" onClick={() => onDeleteAlert?.(alert.id)}>
                        <Trash2 size={16} aria-hidden="true" />
                      </button>
                    </article>
                  ))}
                </div>
              ) : (
                <div className="chart-alerts-empty">
                  <div className="chart-alerts-icon">
                    <Bell size={64} strokeWidth={1} aria-hidden="true" />
                  </div>
                  <p>
                    Уведомления мгновенно уведомляют вас о выполнении заданных условий. Создайте уведомление и убедитесь сами.
                  </p>
                  <button type="button" className="btn primary" onClick={onCreateAlert}>Создать уведомление</button>
                </div>
              )}            </section>
          )}
          {sidePanel === "data" && (
            <section className="chart-data-window">
              <div className="chart-data-title">
                <BookOpen size={18} aria-hidden="true" />
                <strong>{instrumentLabel}</strong>
                <span>{timeframe}</span>
              </div>
              <dl>
                <div><dt>Дата</dt><dd>{formatDate(selectedCandle?.timestamp)}</dd></div>
                <div><dt>Цена откр.</dt><dd>{formatPrice(selectedCandle?.open ?? NaN)}</dd></div>
                <div><dt>Макс.</dt><dd>{formatPrice(selectedCandle?.high ?? NaN)}</dd></div>
                <div><dt>Мин.</dt><dd>{formatPrice(selectedCandle?.low ?? NaN)}</dd></div>
                <div><dt>Цена закрытия</dt><dd>{formatPrice(selectedCandle?.close ?? NaN)}</dd></div>
                <div><dt>Изменение</dt><dd className={changeClass}>{change >= 0 ? "+" : ""}{formatPrice(change)} ({changePct.toFixed(2)}%)</dd></div>
                <div><dt>Объем</dt><dd>{formatVolume(selectedCandle?.volume ?? 0)}</dd></div>
                <div><dt>Диапазон</dt><dd>{formatPrice(range)} ({rangePct.toFixed(2)}%)</dd></div>
              </dl>
            </section>
          )}
        </aside>

        <aside className="chart-right-rail" aria-label="Панели">
          <button type="button" className={sidePanel === "watchlist" ? "active" : ""} onClick={() => setSidePanel("watchlist")} title="Список"><BookOpen size={20} aria-hidden="true" /></button>
          <button type="button" className={sidePanel === "alerts" ? "active" : ""} onClick={() => setSidePanel("alerts")} title="Уведомления"><Bell size={20} aria-label="Уведомления" /></button>
          <button type="button" className={sidePanel === "forecast" ? "active" : ""} onClick={() => setSidePanel("forecast")} title="ML Прогноз"><BrainCircuit size={20} aria-hidden="true" /></button>
          <button type="button" className={sidePanel === "data" ? "active" : ""} onClick={() => setSidePanel("data")} title="Данные"><PanelRight size={20} aria-hidden="true" /></button>
        </aside>
      </div>
    </div>
  );
};
