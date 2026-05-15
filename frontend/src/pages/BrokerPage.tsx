import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { Activity, BriefcaseBusiness, Clock3, History, ListChecks, Plus, RefreshCw, Search, Send, Settings2, Trash2, WalletCards } from "lucide-react";
import { Link } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";

interface MoneyValue {
  currency: string;
  units: number;
  nano: number;
  amount: number;
}

interface BrokerConnection {
  id: number;
  name: string;
  token_hint: string;
  is_sandbox: boolean;
  is_active: boolean;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
}

interface BrokerAccount {
  id: string;
  type: string;
  name: string;
  status: string;
  opened_date?: string;
  closed_date?: string;
  access_level: string;
}

interface PortfolioPosition {
  figi: string;
  instrument_type: string;
  quantity: number;
  average_position_price: MoneyValue;
  expected_yield: number;
  current_price: MoneyValue;
  blocked: boolean;
  blocked_lots: number;
  position_uid: string;
  instrument_uid: string;
  ticker: string;
  class_code: string;
  daily_yield: MoneyValue;
}

interface BrokerPortfolio {
  account_id: string;
  total_amount_portfolio: MoneyValue;
  total_amount_shares: MoneyValue;
  total_amount_bonds: MoneyValue;
  total_amount_etf: MoneyValue;
  total_amount_currencies: MoneyValue;
  total_amount_futures: MoneyValue;
  total_amount_options: MoneyValue;
  expected_yield: number;
  daily_yield: MoneyValue;
  daily_yield_relative: number;
  positions: PortfolioPosition[];
}

interface SecurityPosition {
  figi: string;
  instrument_uid: string;
  ticker: string;
  class_code: string;
  blocked: number;
  balance: number;
}

interface BrokerPositions {
  account_id: string;
  money: MoneyValue[];
  blocked: MoneyValue[];
  securities: SecurityPosition[];
  futures: SecurityPosition[];
  options: SecurityPosition[];
  limits_loading_in_progress: boolean;
}

interface BrokerOperation {
  cursor: string;
  broker_account_id: string;
  id: string;
  name: string;
  date?: string;
  type: string;
  description: string;
  state: string;
  instrument_uid: string;
  quantity_done: number;
}

interface BrokerOperationsPage {
  has_next: boolean;
  next_cursor: string;
  items: BrokerOperation[];
}

interface BrokerOrder {
  order_id: string;
  order_request_id?: string;
  execution_report_status: string;
  lots_requested: number;
  lots_executed: number;
  lots_left: number;
  initial_order_price: MoneyValue;
  executed_order_price: MoneyValue;
  initial_security_price: MoneyValue;
  direction: string;
  order_type: string;
  instrument_uid: string;
  figi: string;
  created_at?: string;
}

interface Instrument {
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
}

interface ConnectionsResponse {
  items: BrokerConnection[];
}

interface AccountsResponse {
  connection: BrokerConnection;
  items: BrokerAccount[];
}

interface PortfolioResponse {
  connection: BrokerConnection;
  account_id: string;
  portfolio: BrokerPortfolio;
}

interface PositionsResponse {
  connection: BrokerConnection;
  account_id: string;
  positions: BrokerPositions;
}

interface OperationsResponse {
  connection: BrokerConnection;
  account_id: string;
  page: BrokerOperationsPage;
}

interface OrdersResponse {
  connection: BrokerConnection;
  account_id: string;
  items: BrokerOrder[];
}

interface OrderResponse {
  connection: BrokerConnection;
  account_id: string;
  order: BrokerOrder;
}

interface SandboxAccountResponse {
  connection: BrokerConnection;
  account_id: string;
}

interface SandboxPayInResponse {
  connection: BrokerConnection;
  account_id: string;
}

const emptyMoney: MoneyValue = { currency: "rub", units: 0, nano: 0, amount: 0 };

function formatMoney(value?: MoneyValue) {
  if (!value) return "—";
  const currency = (value.currency || "rub").toUpperCase();
  const amount = Number.isFinite(value.amount) ? value.amount : value.units + value.nano / 1_000_000_000;
  if (/^[A-Z]{3}$/.test(currency)) {
    return new Intl.NumberFormat("ru-RU", { style: "currency", currency, maximumFractionDigits: 2 }).format(amount);
  }
  return `${new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(amount)} ${currency}`;
}

function formatNumber(value?: number) {
  if (value === undefined || Number.isNaN(value)) return "—";
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 }).format(value);
}

function formatPercent(value?: number) {
  if (value === undefined || Number.isNaN(value)) return "—";
  return `${new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(value * 100)}%`;
}

function formatDate(value?: string) {
  if (!value) return "—";
  return new Date(value).toLocaleString("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
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

function selectedAccount(accounts: BrokerAccount[], accountID: string) {
  return accounts.find((account) => account.id === accountID) || null;
}

function isTradeablePortfolioPosition(position: PortfolioPosition) {
  return Boolean(position.instrument_uid || position.figi) && position.instrument_type !== "currency";
}

export default function BrokerPage() {
  const [connections, setConnections] = useState<BrokerConnection[]>([]);
  const [accounts, setAccounts] = useState<BrokerAccount[]>([]);
  const [selectedConnectionID, setSelectedConnectionID] = useState(0);
  const [selectedAccountID, setSelectedAccountID] = useState("");
  const [portfolio, setPortfolio] = useState<BrokerPortfolio | null>(null);
  const [positions, setPositions] = useState<BrokerPositions | null>(null);
  const [operations, setOperations] = useState<BrokerOperation[]>([]);
  const [orders, setOrders] = useState<BrokerOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [snapshotLoading, setSnapshotLoading] = useState(false);
  const [switching, setSwitching] = useState(false);
  const [orderSubmitting, setOrderSubmitting] = useState(false);
  const [sandboxSubmitting, setSandboxSubmitting] = useState<"create" | "payin" | "">("");
  const [sandboxAmount, setSandboxAmount] = useState("1000000");
  const [instrumentQuery, setInstrumentQuery] = useState("");
  const [instrumentResults, setInstrumentResults] = useState<Instrument[]>([]);
  const [instrumentSearching, setInstrumentSearching] = useState(false);
  const [selectedInstrument, setSelectedInstrument] = useState<Instrument | null>(null);
  const [orderForm, setOrderForm] = useState({
    instrument_id: "",
    direction: "BUY",
    order_type: "LIMIT",
    quantity: "1",
    price: "",
  });
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [orderPanelOpen, setOrderPanelOpen] = useState(false);

  const activeConnection = useMemo(
    () => connections.find((item) => item.id === selectedConnectionID) || connections.find((item) => item.is_active) || null,
    [connections, selectedConnectionID],
  );

  const currentAccount = useMemo(() => selectedAccount(accounts, selectedAccountID), [accounts, selectedAccountID]);

  const loadConnections = async () => {
    setError("");
    setLoading(true);
    try {
      const data = await api.get<ConnectionsResponse>("/api/v1/broker/connections");
      setConnections(data.items || []);
      const preferred = data.items.find((item) => item.is_active) || data.items[0];
      setSelectedConnectionID(preferred?.id || 0);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось загрузить подключения");
    } finally {
      setLoading(false);
    }
  };

  const loadAccounts = async (connectionID: number) => {
    if (!connectionID) return;
    setAccountsLoading(true);
    setError("");
    try {
      const data = await api.get<AccountsResponse>(`/api/v1/broker/accounts?connection_id=${connectionID}`);
      setAccounts(data.items || []);
      setConnections((current) => current.map((item) => (item.id === data.connection.id ? data.connection : item)));
      const openAccount = data.items.find((account) => account.status === "ACCOUNT_STATUS_OPEN") || data.items[0];
      const nextAccountID = data.items.some((account) => account.id === selectedAccountID) ? selectedAccountID : openAccount?.id || "";
      setSelectedAccountID(nextAccountID);
      if (nextAccountID) {
        await api.post("/api/v1/broker/context", { connection_id: connectionID, account_id: nextAccountID }).catch(() => undefined);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось загрузить счета");
      setAccounts([]);
      setSelectedAccountID("");
    } finally {
      setAccountsLoading(false);
    }
  };

  const loadSnapshot = async (connectionID: number, accountID: string) => {
    if (!connectionID || !accountID) return;
    setSnapshotLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({ connection_id: String(connectionID), account_id: accountID });
      const operationParams = new URLSearchParams(params);
      operationParams.set("limit", "30");
      operationParams.set("from", new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString());
      operationParams.set("to", new Date().toISOString());

      const [portfolioData, positionsData, operationsData, ordersData] = await Promise.all([
        api.get<PortfolioResponse>(`/api/v1/broker/portfolio?${params.toString()}`),
        api.get<PositionsResponse>(`/api/v1/broker/positions?${params.toString()}`),
        api.get<OperationsResponse>(`/api/v1/broker/operations?${operationParams.toString()}`),
        api.get<OrdersResponse>(`/api/v1/broker/orders?${params.toString()}`),
      ]);

      setPortfolio(portfolioData.portfolio);
      setPositions(positionsData.positions);
      setOperations(operationsData.page.items || []);
      setOrders(ordersData.items || []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось загрузить данные счета");
    } finally {
      setSnapshotLoading(false);
    }
  };

  useEffect(() => {
    loadConnections();
  }, []);

  useEffect(() => {
    if (selectedConnectionID) {
      loadAccounts(selectedConnectionID);
    }
  }, [selectedConnectionID]);

  useEffect(() => {
    if (selectedConnectionID && selectedAccountID) {
      loadSnapshot(selectedConnectionID, selectedAccountID);
    }
  }, [selectedConnectionID, selectedAccountID]);

  const handleConnectionChange = async (connectionID: number) => {
    if (!connectionID || connectionID === selectedConnectionID) return;
    setSwitching(true);
    setSelectedConnectionID(connectionID);
    setSelectedAccountID("");
    setPortfolio(null);
    setPositions(null);
    setOperations([]);
    setOrders([]);
    try {
      await api.patch(`/api/v1/broker/connections/${connectionID}/active`);
      setConnections((items) => items.map((item) => ({ ...item, is_active: item.id === connectionID })));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось переключить подключение");
    } finally {
      setSwitching(false);
    }
  };

  const handleAccountChange = async (accountID: string) => {
    setSelectedAccountID(accountID);
    if (!selectedConnectionID || !accountID) return;
    await api.post("/api/v1/broker/context", { connection_id: selectedConnectionID, account_id: accountID }).catch(() => undefined);
  };

  const refresh = async () => {
    if (!selectedConnectionID) {
      await loadConnections();
      return;
    }
    await loadAccounts(selectedConnectionID);
    if (selectedAccountID) {
      await loadSnapshot(selectedConnectionID, selectedAccountID);
    }
  };

  const updateOrderForm = (field: keyof typeof orderForm, value: string) => {
    if (field === "instrument_id") {
      setSelectedInstrument(null);
    }
    setOrderForm((current) => ({ ...current, [field]: value }));
  };

  const selectInstrumentForOrder = (instrument: Instrument, direction = orderForm.direction) => {
    setSelectedInstrument(instrument);
    setInstrumentQuery([instrument.ticker, instrument.name].filter(Boolean).join(" "));
    setInstrumentResults([]);
    setOrderForm((current) => ({
      ...current,
      direction,
      instrument_id: instrument.uid || instrument.figi,
    }));
  };

  const selectPositionForOrder = (position: PortfolioPosition, direction: "BUY" | "SELL") => {
    const instrumentID = position.instrument_uid || position.figi;
    if (!instrumentID) return;

    selectInstrumentForOrder({
      uid: position.instrument_uid,
      figi: position.figi,
      ticker: position.ticker || instrumentID,
      class_code: position.class_code,
      isin: "",
      lot: 1,
      currency: position.current_price?.currency || "",
      name: position.ticker || position.instrument_type || instrumentID,
      exchange: position.class_code,
      instrument_type: position.instrument_type,
      api_trade_available: true,
    }, direction);
    setOrderPanelOpen(true);
  };

  const searchOrderInstrument = async () => {
    if (!instrumentQuery.trim()) return;

    setError("");
    setInstrumentSearching(true);
    try {
      const data = await api.get<{ items: Instrument[] }>(`/api/v1/instruments/search?query=${encodeURIComponent(instrumentQuery.trim())}`);
      setInstrumentResults(data.items || []);
      if (!data.items?.length) {
        setError("Инструменты по запросу не найдены");
      }
    } catch (err) {
      if (err instanceof ApiError && err.code === "tinkoff_token_missing") {
        setError("Для поиска инструментов нужен активный T-Invest токен");
      } else {
        setError(err instanceof ApiError ? err.message : "Не удалось выполнить поиск инструмента");
      }
    } finally {
      setInstrumentSearching(false);
    }
  };

  const parseSandboxAmount = () => {
    const amount = Number(sandboxAmount.replace(",", "."));
    if (!Number.isFinite(amount) || amount <= 0) {
      setError("Сумма пополнения должна быть положительным числом");
      return null;
    }
    return amount;
  };

  const createSandboxAccount = async () => {
    if (!selectedConnectionID || !activeConnection?.is_sandbox) return;
    const amount = parseSandboxAmount();
    if (amount === null) return;
    if (!window.confirm(`Создать sandbox-счет и пополнить его на ${sandboxAmount} RUB?`)) return;

    setError("");
    setSuccess("");
    setSandboxSubmitting("create");
    try {
      const response = await api.post<SandboxAccountResponse>("/api/v1/broker/sandbox/accounts", {
        connection_id: selectedConnectionID,
        initial_balance: String(amount),
      });
      await loadAccounts(selectedConnectionID);
      setSelectedAccountID(response.account_id);
      await api.post("/api/v1/broker/context", { connection_id: selectedConnectionID, account_id: response.account_id }).catch(() => undefined);
      await loadSnapshot(selectedConnectionID, response.account_id);
      setSuccess(`Sandbox-счет создан: ${response.account_id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось создать sandbox-счет");
    } finally {
      setSandboxSubmitting("");
    }
  };

  const payInSandboxAccount = async () => {
    if (!selectedConnectionID || !selectedAccountID || !activeConnection?.is_sandbox) return;
    const amount = parseSandboxAmount();
    if (amount === null) return;
    if (!window.confirm(`Пополнить текущий sandbox-счет на ${sandboxAmount} RUB?`)) return;

    setError("");
    setSuccess("");
    setSandboxSubmitting("payin");
    try {
      const response = await api.post<SandboxPayInResponse>("/api/v1/broker/sandbox/pay-in", {
        connection_id: selectedConnectionID,
        account_id: selectedAccountID,
        amount: String(amount),
      });
      await loadSnapshot(selectedConnectionID, selectedAccountID);
      setSuccess(`Sandbox-счет пополнен: ${response.account_id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось пополнить sandbox-счет");
    } finally {
      setSandboxSubmitting("");
    }
  };

  const placeOrder = async (event: FormEvent) => {
    event.preventDefault();
    if (!selectedConnectionID || !selectedAccountID) return;

    const quantity = Number(orderForm.quantity);
    if (!Number.isInteger(quantity) || quantity <= 0) {
      setError("Количество должно быть положительным числом лотов");
      return;
    }
    if (orderForm.order_type === "LIMIT" && !orderForm.price.trim()) {
      setError("Для лимитной заявки нужна цена");
      return;
    }

    const directionLabel = orderForm.direction === "BUY" ? "покупку" : "продажу";
    const typeLabel = orderForm.order_type === "LIMIT" ? "лимитную" : "рыночную";
    const priceLabel = orderForm.order_type === "LIMIT" ? ` по цене ${orderForm.price}` : "";
    const environmentLabel = activeConnection?.is_sandbox ? "sandbox" : "production";
    const confirmed = window.confirm(`Выставить ${typeLabel} заявку на ${directionLabel}: ${orderForm.instrument_id}, ${quantity} лот(ов)${priceLabel}? Контур: ${environmentLabel}.`);
    if (!confirmed) return;

    setError("");
    setSuccess("");
    setOrderSubmitting(true);
    try {
      const response = await api.post<OrderResponse>("/api/v1/broker/orders", {
        connection_id: selectedConnectionID,
        account_id: selectedAccountID,
        instrument_id: orderForm.instrument_id.trim(),
        quantity,
        price: orderForm.order_type === "LIMIT" ? orderForm.price.trim() : "",
        direction: orderForm.direction,
        order_type: orderForm.order_type,
      });
      setSuccess(`Заявка отправлена: ${response.order.order_id || response.order.order_request_id}`);
      setOrderForm((current) => ({ ...current, quantity: "1", price: "" }));
      await loadSnapshot(selectedConnectionID, selectedAccountID);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось выставить заявку");
    } finally {
      setOrderSubmitting(false);
    }
  };

  const cancelOrder = async (orderID: string) => {
    if (!selectedConnectionID || !selectedAccountID || !orderID) return;
    if (!window.confirm(`Отменить заявку ${orderID}?`)) return;

    setError("");
    setSuccess("");
    try {
      const params = new URLSearchParams({ connection_id: String(selectedConnectionID), account_id: selectedAccountID });
      await api.delete(`/api/v1/broker/orders/${encodeURIComponent(orderID)}?${params.toString()}`);
      setSuccess(`Заявка ${orderID} отправлена на отмену`);
      await loadSnapshot(selectedConnectionID, selectedAccountID);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось отменить заявку");
    }
  };

  if (loading) {
    return <div className="page"><div className="loading-panel">Загрузка брокерского кабинета...</div></div>;
  }

  if (!connections.length) {
    return (
      <div className="page broker-page">
        <header className="page-header">
          <div>
            <p className="eyebrow">Брокерский клиент</p>
            <h1>Счета T-Invest</h1>
          </div>
        </header>
        <section className="empty-panel broker-empty">
          <WalletCards size={34} aria-hidden="true" />
          <strong>Нет подключений</strong>
          <span>Добавьте T-Invest токен, чтобы увидеть счета и портфель.</span>
          <Link to="/settings/tinkoff" className="btn primary">Добавить подключение</Link>
        </section>
      </div>
    );
  }

  const total = portfolio?.total_amount_portfolio || emptyMoney;
  const shares = portfolio?.total_amount_shares || emptyMoney;
  const bondsAndEtfAmount = (portfolio?.total_amount_bonds?.amount || 0) + (portfolio?.total_amount_etf?.amount || 0);
  const environmentLabel = activeConnection?.is_sandbox ? "Sandbox" : "Production";

  return (
    <div className="page broker-page">
      <header className="page-header broker-header">
        <div>
          <p className="eyebrow">Брокерский счет</p>
          <h1>{currentAccount?.name || "Портфель T-Invest"}</h1>
          <p className="page-lead">
            {currentAccount ? `${accountTypeLabel(currentAccount.type)} · ${statusLabel(currentAccount.status)} · ${environmentLabel}` : "Выберите счет для просмотра портфеля"}
          </p>
        </div>
        <div className="broker-header-actions">
          <button type="button" className={settingsOpen ? "btn primary" : "btn secondary"} onClick={() => setSettingsOpen((value) => !value)}>
            <Settings2 size={17} aria-hidden="true" />
            Настройки счета
          </button>
          <button type="button" className={orderPanelOpen ? "btn primary" : "btn secondary"} onClick={() => setOrderPanelOpen((value) => !value)} disabled={!selectedAccountID}>
            <Send size={17} aria-hidden="true" />
            Новая заявка
          </button>
          <button type="button" className="btn secondary" onClick={refresh} disabled={accountsLoading || snapshotLoading}>
            <RefreshCw size={17} aria-hidden="true" />
            Обновить
          </button>
        </div>
      </header>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      {settingsOpen && (
        <section className="panel broker-settings-panel">
          <div className="broker-context-card">
            <span className={activeConnection?.is_sandbox ? "status-chip success" : "status-chip warning"}>{environmentLabel}</span>
            <div>
              <strong>{activeConnection?.name || "Подключение"}</strong>
              <small>{activeConnection?.token_hint || "token"} · {currentAccount?.name || selectedAccountID || "счет не выбран"}</small>
            </div>
          </div>

          <div className="broker-controls">
            <label>
              <span>Подключение</span>
              <select value={selectedConnectionID || ""} onChange={(event) => handleConnectionChange(Number(event.target.value))} disabled={switching}>
                {connections.map((connection) => (
                  <option key={connection.id} value={connection.id}>
                    {connection.name} · {connection.is_sandbox ? "sandbox" : "prod"}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span>Счет</span>
              <select value={selectedAccountID} onChange={(event) => handleAccountChange(event.target.value)} disabled={accountsLoading || !accounts.length}>
                {accounts.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.name || account.id}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <div className="account-list broker-settings-accounts">
            {accounts.map((account) => (
              <button
                type="button"
                key={account.id}
                className={account.id === selectedAccountID ? "account-card active" : "account-card"}
                onClick={() => handleAccountChange(account.id)}
              >
                <strong>{account.name || accountTypeLabel(account.type)}</strong>
                <span>{accountTypeLabel(account.type)}</span>
                <small>{statusLabel(account.status)} · {account.opened_date ? formatDate(account.opened_date) : account.id}</small>
              </button>
            ))}
          </div>

          {activeConnection?.is_sandbox && (
            <div className="sandbox-actions">
              <label className="form-group">
                <span>Сумма пополнения, RUB</span>
                <input inputMode="decimal" value={sandboxAmount} onChange={(event) => setSandboxAmount(event.target.value)} />
              </label>
              <button type="button" className="btn secondary" onClick={createSandboxAccount} disabled={sandboxSubmitting !== ""}>
                <Plus size={16} aria-hidden="true" />
                {sandboxSubmitting === "create" ? "Создаем..." : "Создать sandbox-счет"}
              </button>
              <button type="button" className="btn primary" onClick={payInSandboxAccount} disabled={sandboxSubmitting !== "" || !selectedAccountID}>
                <WalletCards size={16} aria-hidden="true" />
                {sandboxSubmitting === "payin" ? "Пополняем..." : "Пополнить счет"}
              </button>
            </div>
          )}
        </section>
      )}

      <section className="broker-summary-grid">
        <article className="stat-card broker-stat-card">
          <span>Стоимость портфеля</span>
          <strong>{formatMoney(total)}</strong>
          <small>{environmentLabel} · {activeConnection?.token_hint}</small>
        </article>
        <article className="stat-card broker-stat-card">
          <span>Акции</span>
          <strong>{formatMoney(shares)}</strong>
          <small>{portfolio?.positions?.length || 0} позиций в портфеле</small>
        </article>
        <article className="stat-card broker-stat-card">
          <span>Облигации и фонды</span>
          <strong>{formatMoney({ ...emptyMoney, amount: bondsAndEtfAmount })}</strong>
          <small>ETF: {formatMoney(portfolio?.total_amount_etf)}</small>
        </article>
        <article className="stat-card broker-stat-card">
          <span>Дневной результат</span>
          <strong className={(portfolio?.daily_yield?.amount || 0) >= 0 ? "positive" : "negative"}>{formatMoney(portfolio?.daily_yield)}</strong>
          <small>{formatPercent(portfolio?.daily_yield_relative)}</small>
        </article>
      </section>

      {orderPanelOpen && (
        <form className="panel broker-order-form broker-order-drawer" onSubmit={placeOrder}>
          <div className="panel-title-row">
            <div>
              <h2>Новая заявка</h2>
              <p>{environmentLabel} · {currentAccount?.name || selectedAccountID}</p>
            </div>
            <Send size={22} aria-hidden="true" />
          </div>
          <div className="order-form-grid">
            <div className="form-group instrument-picker">
              <label>Поиск инструмента</label>
              <div className="instrument-search-line">
                <Search size={17} aria-hidden="true" />
                <input
                  value={instrumentQuery}
                  onChange={(event) => setInstrumentQuery(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") {
                      event.preventDefault();
                      searchOrderInstrument();
                    }
                  }}
                  placeholder="SBER, GAZP, OZON..."
                />
                <button type="button" className="btn small secondary" onClick={searchOrderInstrument} disabled={instrumentSearching || !instrumentQuery.trim()}>
                  {instrumentSearching ? "Ищем..." : "Найти"}
                </button>
              </div>
              {instrumentResults.length > 0 && (
                <div className="instrument-results-list">
                  {instrumentResults.slice(0, 6).map((instrument) => (
                    <button
                      type="button"
                      key={instrument.uid || instrument.figi}
                      className="instrument-result-row"
                      onClick={() => selectInstrumentForOrder(instrument)}
                    >
                      <strong>{instrument.ticker || instrument.uid}</strong>
                      <span>{instrument.name || instrument.instrument_type}</span>
                      <small>{instrument.class_code || instrument.exchange || instrument.currency?.toUpperCase()} · лот {instrument.lot || 1}</small>
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div className="form-group">
              <label>Инструмент UID / FIGI</label>
              <input value={orderForm.instrument_id} onChange={(event) => updateOrderForm("instrument_id", event.target.value)} placeholder="BBG004730N88" required />
              {selectedInstrument && (
                <small className="form-hint">{selectedInstrument.ticker} · {selectedInstrument.class_code || selectedInstrument.exchange || selectedInstrument.currency?.toUpperCase()}</small>
              )}
            </div>
            <div className="form-group">
              <label>Операция</label>
              <select value={orderForm.direction} onChange={(event) => updateOrderForm("direction", event.target.value)}>
                <option value="BUY">Покупка</option>
                <option value="SELL">Продажа</option>
              </select>
            </div>
            <div className="form-group">
              <label>Тип</label>
              <select value={orderForm.order_type} onChange={(event) => updateOrderForm("order_type", event.target.value)}>
                <option value="LIMIT">Лимитная</option>
                <option value="MARKET">Рыночная</option>
              </select>
            </div>
            <div className="form-group">
              <label>Лоты</label>
              <input type="number" min="1" step="1" value={orderForm.quantity} onChange={(event) => updateOrderForm("quantity", event.target.value)} required />
            </div>
            {orderForm.order_type === "LIMIT" && (
              <div className="form-group">
                <label>Цена за инструмент</label>
                <input inputMode="decimal" value={orderForm.price} onChange={(event) => updateOrderForm("price", event.target.value)} placeholder="0.00" required />
              </div>
            )}
          </div>
          <div className="toolbar">
            <button type="submit" className={activeConnection?.is_sandbox ? "btn primary" : "btn danger"} disabled={orderSubmitting || snapshotLoading || !selectedAccountID}>
              <Send size={17} aria-hidden="true" />
              {orderSubmitting ? "Отправляем..." : "Выставить заявку"}
            </button>
          </div>
        </form>
      )}

      <section className="panel broker-main-panel broker-positions-panel">
        <div className="panel-title-row">
          <div>
            <h2>Позиции портфеля</h2>
            <p>{snapshotLoading ? "Загружаем данные счета..." : `${portfolio?.positions?.length || 0} позиций · ${formatMoney(total)}`}</p>
          </div>
          <BriefcaseBusiness size={22} aria-hidden="true" />
        </div>
        <div className="responsive-table">
          <table className="data-table broker-positions-table">
            <thead>
              <tr>
                <th>Инструмент</th>
                <th>Тип</th>
                <th>Кол-во</th>
                <th>Цена</th>
                <th>Доход</th>
                <th className="row-actions">Действия</th>
              </tr>
            </thead>
            <tbody>
              {(portfolio?.positions || []).map((position) => (
                <tr key={position.position_uid || position.instrument_uid || position.figi}>
                  <td>
                    <span className="instrument-cell">
                      <strong>{position.ticker || position.instrument_uid || position.figi}</strong>
                      <small>{position.class_code || position.instrument_uid}</small>
                    </span>
                  </td>
                  <td>{position.instrument_type || "—"}</td>
                  <td>{formatNumber(position.quantity)}</td>
                  <td>{formatMoney(position.current_price)}</td>
                  <td className={position.expected_yield >= 0 ? "positive" : "negative"}>{formatNumber(position.expected_yield)}</td>
                  <td className="row-actions trade-action-cell">
                    {isTradeablePortfolioPosition(position) && (
                      <>
                        <button type="button" className="btn small secondary" onClick={() => selectPositionForOrder(position, "BUY")}>Купить</button>
                        <button type="button" className="btn small secondary" onClick={() => selectPositionForOrder(position, "SELL")}>Продать</button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
              {!portfolio?.positions?.length && (
                <tr>
                  <td colSpan={6} className="empty-cell">Позиции не найдены</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="broker-secondary-grid broker-account-info-grid">
        <section className="panel">
          <div className="panel-title-row">
            <div>
              <h2>Деньги</h2>
              <p>{positions?.limits_loading_in_progress ? "Лимиты обновляются" : "Свободные остатки"}</p>
            </div>
            <Activity size={22} aria-hidden="true" />
          </div>
          <div className="money-strip">
            {(positions?.money || []).map((item) => (
              <div key={item.currency}>
                <span>{item.currency.toUpperCase()}</span>
                <strong>{formatMoney(item)}</strong>
              </div>
            ))}
            {!positions?.money?.length && <span className="empty-list-item">Нет денежных остатков</span>}
          </div>
        </section>

        <section className="panel">
          <div className="panel-title-row">
            <div>
              <h2>Активные заявки</h2>
              <p>{orders.length ? `${orders.length} заявок` : "Нет активных заявок"}</p>
            </div>
            <ListChecks size={22} aria-hidden="true" />
          </div>
          <div className="responsive-table">
            <table className="data-table compact">
              <thead>
                <tr>
                  <th>Инструмент</th>
                  <th>Статус</th>
                  <th>Лоты</th>
                  <th>Цена</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => (
                  <tr key={order.order_id}>
                    <td>{order.instrument_uid || order.figi}</td>
                    <td><span className="status-chip">{statusLabel(order.execution_report_status)}</span></td>
                    <td>{order.lots_executed}/{order.lots_requested}</td>
                    <td>{formatMoney(order.initial_order_price)}</td>
                    <td className="row-actions">
                      <button type="button" className="icon-button small danger-icon" onClick={() => cancelOrder(order.order_id)} title="Отменить заявку">
                        <Trash2 size={16} aria-hidden="true" />
                      </button>
                    </td>
                  </tr>
                ))}
                {!orders.length && (
                  <tr>
                    <td colSpan={5} className="empty-cell">Активных заявок нет</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <section className="broker-secondary-grid broker-account-info-grid">
        <section className="panel">
          <div className="panel-title-row">
            <div>
              <h2>Бумаги</h2>
              <p>Остатки по счету</p>
            </div>
            <Activity size={22} aria-hidden="true" />
          </div>
          <div className="responsive-table">
            <table className="data-table compact">
              <thead>
                <tr>
                  <th>Бумага</th>
                  <th>Баланс</th>
                  <th>Блок</th>
                </tr>
              </thead>
              <tbody>
                {(positions?.securities || []).slice(0, 8).map((item) => (
                  <tr key={item.instrument_uid || item.figi}>
                    <td>{item.ticker || item.instrument_uid || item.figi}</td>
                    <td>{formatNumber(item.balance)}</td>
                    <td>{formatNumber(item.blocked)}</td>
                  </tr>
                ))}
                {!positions?.securities?.length && (
                  <tr>
                    <td colSpan={3} className="empty-cell">Нет бумаг</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>

        <section className="panel">
          <div className="panel-title-row">
            <div>
              <h2>Операции за 30 дней</h2>
              <p>Последние операции по выбранному счету</p>
            </div>
            <History size={22} aria-hidden="true" />
          </div>
          <div className="operations-list">
            {operations.map((operation) => (
              <article key={operation.id || operation.cursor} className="operation-row">
                <Clock3 size={18} aria-hidden="true" />
                <div>
                  <strong>{operation.name || operation.description || operation.type}</strong>
                  <span>{operation.description || operation.type}</span>
                </div>
                <time>{formatDate(operation.date)}</time>
              </article>
            ))}
            {!operations.length && <div className="empty-panel small-empty">Операций за период нет</div>}
          </div>
        </section>
      </section>
    </div>
  );
}
