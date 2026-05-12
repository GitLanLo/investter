import React, { useCallback, useEffect, useState } from "react";
import { WifiOff, X } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";
import { Chart, type ChartCandle } from "../shared/ui/Chart";
import { AlertModal } from "../entities/alerts/ui/AlertModal";

interface AssetSummary {
  asset: {
    id: string;
    ticker: string;
    name: string;
    exchange: string;
    timeframe: string;
    currency: string;
    figi?: string;
    instrument_uid?: string;
    class_code?: string;
    instrument_type?: string;
    lot?: number;
    model_supported: boolean;
  };
  last_price: number;
  price_change: number;
  data_fresh: boolean;
  last_candle_at?: string;
}

interface Candle {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

interface CandlesResponse {
  timeframe?: string;
  items: Candle[];
}

interface Signal {
  id?: number;
  signal_direction: string;
  signal_state: string;
  signal_probability: number;
  as_of_time: string;
  model_version: string;
}

interface SignalsResponse {
  items: Signal[];
}

interface TinkoffStatus {
  connected: boolean;
}

function formatDate(value?: string) {
  if (!value) return "нет данных";
  return new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

function signalText(signal?: Signal) {
  if (!signal) return "Прогноз еще не запускался";
  if (signal.signal_state === "no_trade") return "Сделок не требуется";
  if (signal.signal_direction === "up") return "Ожидается рост";
  if (signal.signal_direction === "down") return "Ожидается снижение";
  return signal.signal_state || "Прогноз готов";
}

function directionText(value: string) {
  if (value === "up") return "Рост";
  if (value === "down") return "Снижение";
  if (value === "flat") return "Боковик";
  return value || "n/a";
}

function dedupeCandles(items: Candle[]): ChartCandle[] {
  const byTime = new Map<number, ChartCandle>();
  items.forEach((c) => {
    const time = new Date(c.timestamp).getTime();
    byTime.set(time, {
      timestamp: c.timestamp,
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
      volume: c.volume,
    });
  });
  return Array.from(byTime.values()).sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
}

const TIMEFRAMES = [
  { value: "1m", label: "1м" },
  { value: "5m", label: "5м" },
  { value: "15m", label: "15м" },
  { value: "1h", label: "1ч" },
  { value: "1d", label: "1д" },
] as const;

type Timeframe = typeof TIMEFRAMES[number]["value"];

export default function InstrumentPage() {
  const { id } = useParams();
  const [summary, setSummary] = useState<AssetSummary | null>(null);
  const [candles, setCandles] = useState<ChartCandle[]>([]);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [watchlistStatus, setWatchlistStatus] = useState<any[]>([]);
  const [hasToken, setHasToken] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [analysisRunning, setAnalysisRunning] = useState(false);
  const [isAlertModalOpen, setIsAlertModalOpen] = useState(false);
  const [timeframe, setTimeframeState] = useState<Timeframe>(() => {
    return (localStorage.getItem("invest.timeframe") as Timeframe) || "5m";
  });

  const setTimeframe = (val: Timeframe) => {
    setTimeframeState(val);
    localStorage.setItem("invest.timeframe", val);
  };
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const [inWatchlist, setInWatchlist] = useState(false);
  const [instrumentAlerts, setInstrumentAlerts] = useState<Array<{ id: number; condition: string; action: string; active: boolean }>>([]);

  const applyData = (
    summaryData: AssetSummary, 
    candlesData: CandlesResponse, 
    signalsData: SignalsResponse, 
    watchlistData?: { items: { asset_id: string }[] }, 
    alertsData?: { items: any[] },
    freshnessData?: { items: any[] }
  ) => {
    setSummary(summaryData);
    setCandles(dedupeCandles(candlesData.items));
    setSignals(signalsData.items || []);
    if (watchlistData && summaryData?.asset?.id) {
      setInWatchlist(watchlistData.items.some((item) => item.asset_id === summaryData.asset.id));
    }
    if (freshnessData) {
      setWatchlistStatus(freshnessData.items);
    }
    if (alertsData && summaryData?.asset?.ticker) {
      const formattedAlerts = alertsData.items
        .filter((rule) => rule.ticker === summaryData.asset.ticker)
        .map((rule) => {
          let condition = "Неизвестно";
          let action = rule.is_enabled ? "Активно" : "Отключено";
          
          if (rule.event_type === "decision_threshold_triggered") {
            condition = `Сильный прогноз: ${rule.direction === "up" ? "Рост" : rule.direction === "down" ? "Снижение" : "Любой"}`;
            if (rule.threshold) condition += ` (>${(rule.threshold * 100).toFixed(0)}%)`;
          } else if (rule.event_type === "classification_success") {
            condition = "Любой прогноз";
          } else if (rule.event_type === "inference_blocked_by_runtime") {
            condition = "Ошибка расчета";
          } else if (rule.event_type === "price") {
            condition = `Цена ${rule.operator || ">"} ${rule.threshold}`;
          } else {
            condition = rule.event_type;
          }
          
          return { id: rule.id, condition, action, active: rule.is_enabled };
        });
      setInstrumentAlerts(formattedAlerts);
    }
  };

  const fetchData = useCallback(async (allowAutoRefresh = true) => {
    if (!id) return;

    setLoading(true);
    setError("");
    try {
      const params = `timeframe=${encodeURIComponent(timeframe)}`;
      const [summaryData, candlesData, signalsData, tokenStatus, watchlistData, alertsData, freshnessData] = await Promise.all([
        api.get<AssetSummary>(`/api/v1/assets/${id}/summary?${params}`),
        api.get<CandlesResponse>(`/api/v1/assets/${id}/candles?${params}&limit=800`),
        api.get<SignalsResponse>(`/api/v1/assets/${id}/signals?limit=10`),
        api.get<TinkoffStatus>("/api/v1/me/tinkoff-token/status").catch(() => ({ connected: false })),
        api.get<{ items: { asset_id: string }[] }>("/api/v1/watchlist").catch(() => undefined),
        api.get<{ items: any[] }>("/api/v1/alerts/rules").catch(() => undefined),
        api.get<{ items: any[] }>("/api/v1/watchlist/freshness").catch(() => undefined),
      ]);

      applyData(summaryData, candlesData, signalsData, watchlistData, alertsData, freshnessData);
      setHasToken(tokenStatus.connected);

      const needsRefresh = tokenStatus.connected && allowAutoRefresh && (candlesData.items.length === 0 || !summaryData.data_fresh);
      if (needsRefresh) {
        setRefreshing(true);
        try {
          await api.post(`/api/v1/assets/${id}/refresh`, { timeframe });
          await fetchData(false);
        } catch (err) {
          if (err instanceof ApiError) {
            setError(`Ошибка обновления: ${err.message}`);
          } else {
            setError("Не удалось автоматически загрузить свежие свечи из Tinkoff");
          }
        } finally {
          setRefreshing(false);
        }
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate("/login");
      } else {
        setError("Не удалось загрузить данные инструмента");
      }
    } finally {
      setLoading(false);
    }
  }, [id, navigate, timeframe]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const runAnalysis = async () => {
    if (!id) return;
    setError("");
    setAnalysisRunning(true);
    try {
      await api.post(`/api/v1/assets/${id}/analysis/run`, {});
      await fetchData(false);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(`Не удалось запустить анализ: ${err.message}`);
      } else {
        setError("Не удалось запустить анализ");
      }
    } finally {
      setAnalysisRunning(false);
    }
  };

  const refreshData = async () => {
    if (!id) return;
    setError("");
    setRefreshing(true);
    try {
      await api.post(`/api/v1/assets/${id}/refresh`, { timeframe });
      await fetchData(false);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "tinkoff_token_missing") {
          setError("Для загрузки свечей нужен Tinkoff API токен");
        } else {
          setError(`Ошибка загрузки свечей: ${err.message}`);
        }
      } else {
        setError("Не удалось загрузить свечи из Tinkoff");
      }
    } finally {
      setRefreshing(false);
    }
  };

  const toggleWatchlist = async () => {
    if (!summary?.asset?.id) return;
    try {
      if (inWatchlist) {
        await api.delete(`/api/v1/watchlist/${summary.asset.id}`);
        setInWatchlist(false);
      } else {
        await api.post("/api/v1/watchlist", { asset_id: summary.asset.id });
        setInWatchlist(true);
      }
    } catch (err) {
      setError("Не удалось обновить список желаемых");
    }
  };

  const openAlertModal = () => {
    setIsAlertModalOpen(true);
  };

  const submitAlert = async (rule: any) => {
    try {
      await api.post("/api/v1/alerts/rules", rule);
      setIsAlertModalOpen(false);
      await fetchData(false);
    } catch (err) {
      setError("Не удалось создать оповещение");
    }
  };

  const deleteAlert = async (id: number) => {
    try {
      await api.delete(`/api/v1/alerts/rules/${id}`);
      await fetchData(false);
    } catch (err) {
      setError("Не удалось удалить оповещение");
    }
  };

  if (loading && !summary) {
    return <div className="page"><div className="loading-panel">Загрузка инструмента...</div></div>;
  }
  if (!summary) {
    return <div className="page"><div className="empty-panel">Инструмент не найден</div></div>;
  }

  const lastSignal = signals[0];
  const forecastProbability = lastSignal ? `${(lastSignal.signal_probability * 100).toFixed(1)}%` : undefined;

  return (
    <div className="instrument-workspace-page">
      {error && (
        <div className="error-message workspace-error">
          <WifiOff size={17} aria-hidden="true" />
          <span>{error}</span>
          {!hasToken && <Link to="/settings/tinkoff">Подключить токен</Link>}
        </div>
      )}

      <Chart
        data={candles}
        timeframe={timeframe}
        timeframeOptions={TIMEFRAMES.map((item) => ({ value: item.value, label: item.label }))}
        onTimeframeChange={(value) => setTimeframe(value as Timeframe)}
        onRefresh={refreshData}
        refreshing={refreshing}
        storageKey={`${summary.asset.id}:${timeframe}`}
        instrumentLabel={`${summary.asset.ticker} · ${summary.asset.name}`}
        marketLabel={summary.asset.class_code || summary.asset.exchange || "MOEX"}
        inWatchlist={inWatchlist}
        onToggleWatchlist={toggleWatchlist}
        onCreateAlert={openAlertModal}
        alerts={instrumentAlerts}
        onDeleteAlert={deleteAlert}
        watchlist={watchlistStatus.map((item) => ({
          assetId: item.asset_id,
          ticker: item.ticker,
          lastPrice: item.last_price,
          priceChange: item.price_change,
        }))}
        onSelectAsset={(assetId) => navigate(`/instruments/${assetId}`)}
        forecast={{
          title: signalText(lastSignal),
          description: lastSignal
            ? `${forecastProbability} уверенность · ${formatDate(lastSignal.as_of_time)}`
            : "Запустите прогноз по свежим свечам, чтобы оценить текущую бумагу.",
          probability: forecastProbability,
          running: analysisRunning,
          disabled: candles.length === 0,
          onRun: runAnalysis,
          history: signals.map((signal) => ({
            id: signal.id,
            date: formatDate(signal.as_of_time),
            state: signal.signal_state,
            direction: directionText(signal.signal_direction || signal.signal_state),
            probability: `${(signal.signal_probability * 100).toFixed(1)}%`,
            model: signal.model_version,
          })),
        }}
        emptyText={hasToken ? "Нажмите «Обновить», чтобы получить данные из Tinkoff" : "Подключите Tinkoff токен в настройках"}
      />

      <AlertModal
        isOpen={isAlertModalOpen}
        ticker={summary.asset.ticker}
        onClose={() => setIsAlertModalOpen(false)}
        onSubmit={submitAlert}
      />
    </div>
  );
}
