import React, { useCallback, useEffect, useState } from "react";
import { ShoppingCart, WifiOff, X } from "lucide-react";
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

interface BrokerConnection {
  id: number;
  name: string;
  token_hint: string;
  is_sandbox: boolean;
  is_active: boolean;
}

interface BrokerAccount {
  id: string;
  type: string;
  name: string;
  status: string;
}

interface MoneyValue {
  currency: string;
  units: number;
  nano: number;
  amount: number;
}

interface BrokerOrder {
  order_id: string;
  order_request_id?: string;
  execution_report_status: string;
  lots_requested: number;
  lots_executed: number;
  lots_left: number;
  initial_order_price?: MoneyValue;
  direction: string;
  order_type: string;
  instrument_uid: string;
  figi: string;
  created_at?: string;
}

interface ConnectionsResponse {
  items: BrokerConnection[];
}

interface AccountsResponse {
  connection: BrokerConnection;
  items: BrokerAccount[];
}

interface OrderResponse {
  connection: BrokerConnection;
  account_id: string;
  order: BrokerOrder;
}

interface OrdersResponse {
  connection: BrokerConnection;
  account_id: string;
  items: BrokerOrder[];
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

function accountTypeLabel(type: string) {
  const map: Record<string, string> = {
    ACCOUNT_TYPE_TINKOFF: "Брокерский",
    ACCOUNT_TYPE_TINKOFF_IIS: "ИИС",
    ACCOUNT_TYPE_INVEST_BOX: "Инвесткопилка",
  };
  return map[type] || type || "Счет";
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    ACCOUNT_STATUS_OPEN: "Открыт",
    ACCOUNT_STATUS_CLOSED: "Закрыт",
    EXECUTION_REPORT_STATUS_FILL: "Исполнена",
    EXECUTION_REPORT_STATUS_NEW: "Новая",
    EXECUTION_REPORT_STATUS_CANCELLED: "Отменена",
    EXECUTION_REPORT_STATUS_REJECTED: "Отклонена",
    EXECUTION_REPORT_STATUS_PARTIALLYFILL: "Частично",
  };
  return map[status] || status || "—";
}

function formatMoneyAmount(value: number, currency?: string) {
  if (!Number.isFinite(value)) return "—";
  const code = (currency || "rub").toUpperCase();
  if (/^[A-Z]{3}$/.test(code)) {
    return new Intl.NumberFormat("ru-RU", { style: "currency", currency: code, maximumFractionDigits: 4 }).format(value);
  }
  return `${new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 }).format(value)} ${code}`;
}

function formatMoneyValue(value?: MoneyValue) {
  if (!value) return "—";
  const amount = Number.isFinite(value.amount) ? value.amount : value.units + value.nano / 1_000_000_000;
  return formatMoneyAmount(amount, value.currency);
}

function orderDirectionLabel(value: string) {
  if (value === "ORDER_DIRECTION_SELL" || value === "SELL") return "Продажа";
  return "Покупка";
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
type TradeDirection = "BUY" | "SELL";

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
  const [isTradeModalOpen, setIsTradeModalOpen] = useState(false);
  const [tradeLoading, setTradeLoading] = useState(false);
  const [tradeSubmitting, setTradeSubmitting] = useState(false);
  const [tradeOrdersLoading, setTradeOrdersLoading] = useState(false);
  const [tradeConnections, setTradeConnections] = useState<BrokerConnection[]>([]);
  const [tradeAccounts, setTradeAccounts] = useState<BrokerAccount[]>([]);
  const [tradeOrders, setTradeOrders] = useState<BrokerOrder[]>([]);
  const [tradeConnectionID, setTradeConnectionID] = useState(0);
  const [tradeAccountID, setTradeAccountID] = useState("");
  const [tradeDirection, setTradeDirection] = useState<TradeDirection>("BUY");
  const [tradeOrderType, setTradeOrderType] = useState<"LIMIT" | "MARKET">("LIMIT");
  const [tradeQuantity, setTradeQuantity] = useState("1");
  const [tradePrice, setTradePrice] = useState("");
  const [tradeError, setTradeError] = useState("");
  const [tradeSuccess, setTradeSuccess] = useState("");
  const [tradeAuthRequired, setTradeAuthRequired] = useState(false);
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

  const loadOlderCandles = useCallback(async (timestamp: number) => {
    if (!id) return undefined;
    try {
      const date = new Date(timestamp);
      const toParam = date.toISOString();
      const params = `timeframe=${encodeURIComponent(timeframe)}`;
      const res = await api.get<CandlesResponse>(`/api/v1/assets/${id}/candles?${params}&to=${toParam}&limit=800`);
      if (res.items && res.items.length > 0) {
        setCandles((prev) => {
          const merged = [...res.items, ...prev];
          return dedupeCandles(merged);
        });
        return res.items;
      }
    } catch (err) {
      console.error("Failed to load older candles", err);
    }
    return undefined;
  }, [id, timeframe]);

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

  const activeTradeConnection = tradeConnections.find((connection) => connection.id === tradeConnectionID) || null;

  const loadTradeOrders = async (connectionID: number, accountID: string) => {
    if (!connectionID || !accountID) {
      setTradeOrders([]);
      return;
    }

    setTradeOrdersLoading(true);
    try {
      const params = new URLSearchParams({ connection_id: String(connectionID), account_id: accountID });
      const data = await api.get<OrdersResponse>(`/api/v1/broker/orders?${params.toString()}`);
      setTradeOrders(data.items || []);
    } catch {
      setTradeOrders([]);
    } finally {
      setTradeOrdersLoading(false);
    }
  };

  const loadTradeAccounts = async (connectionID: number, preferredAccountID = "") => {
    if (!connectionID) {
      setTradeAccounts([]);
      setTradeAccountID("");
      setTradeOrders([]);
      return;
    }

    const data = await api.get<AccountsResponse>(`/api/v1/broker/accounts?connection_id=${connectionID}`);
    const items = data.items || [];
    setTradeAccounts(items);
    setTradeConnections((current) => current.map((item) => (item.id === data.connection.id ? data.connection : item)));
    const openAccount = items.find((account) => account.status === "ACCOUNT_STATUS_OPEN") || items[0];
    const nextAccountID = items.some((account) => account.id === preferredAccountID) ? preferredAccountID : openAccount?.id || "";
    setTradeAccountID(nextAccountID);
    await loadTradeOrders(connectionID, nextAccountID);
  };

  const openTradeTicket = async (direction: TradeDirection = "BUY") => {
    if (!summary) return;
    setIsTradeModalOpen(true);
    setTradeError("");
    setTradeSuccess("");
    setTradeAuthRequired(false);
    setTradeDirection(direction);
    setTradeQuantity("1");
    setTradeOrderType("LIMIT");
    setTradePrice(summary.last_price ? String(summary.last_price) : "");
    setTradeLoading(true);
    try {
      const connectionsData = await api.get<ConnectionsResponse>("/api/v1/broker/connections");
      const connections = connectionsData.items || [];
      setTradeConnections(connections);
      const preferred = connections.find((connection) => connection.is_active) || connections[0];
      setTradeConnectionID(preferred?.id || 0);
      if (preferred) {
        await loadTradeAccounts(preferred.id, tradeAccountID);
      }
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401) {
          setTradeAuthRequired(true);
          setTradeError("Нужно войти в аккаунт, чтобы загрузить брокерские счета");
        } else {
          setTradeError(err.message);
        }
      } else {
        setTradeError("Не удалось загрузить брокерские счета");
      }
    } finally {
      setTradeLoading(false);
    }
  };

  const changeTradeConnection = async (connectionID: number) => {
    setTradeConnectionID(connectionID);
    setTradeError("");
    setTradeSuccess("");
    setTradeLoading(true);
    try {
      await loadTradeAccounts(connectionID);
    } catch (err) {
      setTradeError(err instanceof ApiError ? err.message : "Не удалось загрузить счета подключения");
    } finally {
      setTradeLoading(false);
    }
  };

  const changeTradeAccount = async (accountID: string) => {
    setTradeAccountID(accountID);
    setTradeError("");
    setTradeSuccess("");
    await loadTradeOrders(tradeConnectionID, accountID);
  };

  const cancelTradeOrder = async (orderID: string) => {
    if (!tradeConnectionID || !tradeAccountID || !orderID) return;
    if (!window.confirm(`Отменить заявку ${orderID}?`)) return;

    setTradeError("");
    setTradeSuccess("");
    try {
      const params = new URLSearchParams({ connection_id: String(tradeConnectionID), account_id: tradeAccountID });
      await api.delete(`/api/v1/broker/orders/${encodeURIComponent(orderID)}?${params.toString()}`);
      setTradeSuccess(`Заявка ${orderID} отправлена на отмену`);
      await loadTradeOrders(tradeConnectionID, tradeAccountID);
    } catch (err) {
      setTradeError(err instanceof ApiError ? err.message : "Не удалось отменить заявку");
    }
  };

  const submitTradeOrder = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!summary) return;

    const instrumentID = summary.asset.instrument_uid || summary.asset.figi;
    if (!instrumentID) {
      setTradeError("У инструмента нет UID/FIGI для выставления заявки");
      return;
    }

    const quantity = Number(tradeQuantity);
    if (!Number.isInteger(quantity) || quantity <= 0) {
      setTradeError("Количество должно быть положительным числом лотов");
      return;
    }
    if (tradeOrderType === "LIMIT" && !tradePrice.trim()) {
      setTradeError("Для лимитной заявки нужна цена");
      return;
    }
    if (!tradeConnectionID || !tradeAccountID) {
      setTradeError("Выберите брокерский счет");
      return;
    }

    const lot = summary.asset.lot || 1;
    const priceLabel = tradeOrderType === "LIMIT" ? ` по цене ${tradePrice}` : "";
    const environment = activeTradeConnection?.is_sandbox ? "sandbox" : "production";
    const actionLabel = tradeDirection === "BUY" ? "Купить" : "Продать";
    const confirmed = window.confirm(`${actionLabel} ${summary.asset.ticker}: ${quantity} лот(ов), ${quantity * lot} шт.${priceLabel}? Контур: ${environment}.`);
    if (!confirmed) return;

    setTradeSubmitting(true);
    setTradeError("");
    setTradeSuccess("");
    try {
      const response = await api.post<OrderResponse>("/api/v1/broker/orders", {
        connection_id: tradeConnectionID,
        account_id: tradeAccountID,
        instrument_id: instrumentID,
        quantity,
        price: tradeOrderType === "LIMIT" ? tradePrice.trim() : "",
        direction: tradeDirection,
        order_type: tradeOrderType,
      });
      await api.post("/api/v1/broker/context", { connection_id: tradeConnectionID, account_id: tradeAccountID }).catch(() => undefined);
      setTradeSuccess(`Заявка отправлена: ${response.order.order_id || response.order.order_request_id}`);
      await loadTradeOrders(tradeConnectionID, tradeAccountID);
    } catch (err) {
      setTradeError(err instanceof ApiError ? err.message : "Не удалось выставить заявку");
    } finally {
      setTradeSubmitting(false);
    }
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
        onBuyInstrument={() => openTradeTicket("BUY")}
        buyInstrumentDisabled={!(summary.asset.instrument_uid || summary.asset.figi) || summary.asset.instrument_type === "currency"}
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
        emptyText={hasToken ? "" : "Подключите Tinkoff токен в настройках"}
      />

      <AlertModal
        isOpen={isAlertModalOpen}
        ticker={summary.asset.ticker}
        onClose={() => setIsAlertModalOpen(false)}
        onSubmit={submitAlert}
      />

      {isTradeModalOpen && (
        <div className="modal-backdrop" onClick={() => setIsTradeModalOpen(false)}>
          <form className="modal-content trade-ticket-modal" onSubmit={submitTradeOrder} onClick={(event) => event.stopPropagation()}>
            <div className="modal-header">
              <h2><ShoppingCart size={20} aria-hidden="true" /> Заявка <strong>{summary.asset.ticker}</strong></h2>
              <button type="button" className="modal-close" onClick={() => setIsTradeModalOpen(false)} title="Закрыть"><X size={20} aria-hidden="true" /></button>
            </div>

            <div className="modal-body">
              {tradeError && <div className="error-message">{tradeError}</div>}
              {tradeSuccess && <div className="success-message">{tradeSuccess}</div>}

              <div className="trade-ticket-summary">
                <div>
                  <span>Инструмент</span>
                  <strong>{summary.asset.name}</strong>
                  <small>{summary.asset.class_code || summary.asset.exchange || "MOEX"} · лот {summary.asset.lot || 1}</small>
                </div>
                <div>
                  <span>Последняя цена</span>
                  <strong>{formatMoneyAmount(summary.last_price, summary.asset.currency)}</strong>
                  <small>{formatDate(summary.last_candle_at)}</small>
                </div>
              </div>

              {tradeLoading ? (
                <div className="loading-panel small-empty">Загружаем счета...</div>
              ) : tradeAuthRequired ? (
                <div className="empty-panel small-empty">
                  <span>Сессия не активна. Войдите в аккаунт, чтобы выставлять заявки.</span>
                  <Link to="/login" className="btn primary">Войти</Link>
                </div>
              ) : tradeConnections.length === 0 ? (
                <div className="empty-panel small-empty">
                  <span>Нет broker-подключений.</span>
                  <Link to="/settings/tinkoff" className="btn primary">Добавить T-Invest токен</Link>
                </div>
              ) : (
                <div className="trade-ticket-grid">
                  <label className="form-group">
                    <span>Подключение</span>
                    <select value={tradeConnectionID || ""} onChange={(event) => changeTradeConnection(Number(event.target.value))}>
                      {tradeConnections.map((connection) => (
                        <option key={connection.id} value={connection.id}>
                          {connection.name} · {connection.is_sandbox ? "sandbox" : "prod"}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="form-group">
                    <span>Счет</span>
                    <select value={tradeAccountID} onChange={(event) => changeTradeAccount(event.target.value)} disabled={!tradeAccounts.length}>
                      {tradeAccounts.map((account) => (
                        <option key={account.id} value={account.id}>
                          {account.name || account.id} · {accountTypeLabel(account.type)} · {statusLabel(account.status)}
                        </option>
                      ))}
                    </select>
                  </label>
                  <div className="form-group trade-direction-group">
                    <span>Операция</span>
                    <div className="trade-direction-control">
                      <button type="button" className={tradeDirection === "BUY" ? "active buy" : "buy"} onClick={() => setTradeDirection("BUY")}>Купить</button>
                      <button type="button" className={tradeDirection === "SELL" ? "active sell" : "sell"} onClick={() => setTradeDirection("SELL")}>Продать</button>
                    </div>
                  </div>
                  <label className="form-group">
                    <span>Тип заявки</span>
                    <select value={tradeOrderType} onChange={(event) => setTradeOrderType(event.target.value as "LIMIT" | "MARKET")}>
                      <option value="LIMIT">Лимитная</option>
                      <option value="MARKET">Рыночная</option>
                    </select>
                  </label>
                  <label className="form-group">
                    <span>Лоты</span>
                    <input type="number" min="1" step="1" value={tradeQuantity} onChange={(event) => setTradeQuantity(event.target.value)} />
                  </label>
                  {tradeOrderType === "LIMIT" && (
                    <label className="form-group">
                      <span>Цена</span>
                      <input inputMode="decimal" value={tradePrice} onChange={(event) => setTradePrice(event.target.value)} />
                    </label>
                  )}
                </div>
              )}

              {tradeConnections.length > 0 && (
                <section className="trade-orders-panel">
                  <div className="panel-title-row">
                    <div>
                      <h3>Активные заявки</h3>
                      <p>{tradeOrdersLoading ? "Обновляем..." : tradeOrders.length ? `${tradeOrders.length} заявок по счету` : "Нет активных заявок"}</p>
                    </div>
                  </div>
                  <div className="trade-orders-list">
                    {tradeOrders.map((order) => (
                      <article key={order.order_id} className="trade-order-row">
                        <div>
                          <strong>{order.instrument_uid === summary.asset.instrument_uid || order.figi === summary.asset.figi ? summary.asset.ticker : order.instrument_uid || order.figi}</strong>
                          <span>{orderDirectionLabel(order.direction)} · {statusLabel(order.execution_report_status)}</span>
                        </div>
                        <div>
                          <strong>{order.lots_executed}/{order.lots_requested}</strong>
                          <span>{formatMoneyValue(order.initial_order_price)}</span>
                        </div>
                        <button type="button" className="icon-button small danger-icon" onClick={() => cancelTradeOrder(order.order_id)} title="Отменить заявку">
                          <X size={16} aria-hidden="true" />
                        </button>
                      </article>
                    ))}
                    {!tradeOrders.length && <div className="empty-list-item">Активных заявок нет</div>}
                  </div>
                </section>
              )}
            </div>

            <div className="modal-footer">
              <button type="button" className="btn secondary" onClick={() => setIsTradeModalOpen(false)}>Закрыть</button>
              <button type="submit" className={tradeDirection === "SELL" && !activeTradeConnection?.is_sandbox ? "btn danger" : "btn primary"} disabled={tradeSubmitting || tradeLoading || !tradeConnections.length || !tradeAccountID}>
                <ShoppingCart size={17} aria-hidden="true" />
                {tradeSubmitting ? "Отправляем..." : tradeDirection === "BUY" ? "Купить" : "Продать"}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
