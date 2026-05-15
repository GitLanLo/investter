import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { CheckCircle2, KeyRound, Plus, RadioTower, ShieldCheck, Trash2 } from "lucide-react";
import { Link } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";

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

interface ConnectionsResponse {
  items: BrokerConnection[];
}

function formatDate(value?: string) {
  if (!value) return "нет данных";
  return new Date(value).toLocaleString("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function SettingsTinkoffPage() {
  const [connections, setConnections] = useState<BrokerConnection[]>([]);
  const [name, setName] = useState("");
  const [token, setToken] = useState("");
  const [isSandbox, setIsSandbox] = useState(false);
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const activeConnection = useMemo(() => connections.find((item) => item.is_active) || null, [connections]);

  useEffect(() => {
    fetchConnections();
  }, []);

  const fetchConnections = async () => {
    try {
      const data = await api.get<ConnectionsResponse>("/api/v1/broker/connections");
      setConnections(data.items || []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось загрузить подключения");
    }
  };

  const handleCreate = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);

    try {
      await api.post<BrokerConnection>("/api/v1/broker/connections", {
        name,
        token,
        is_sandbox: isSandbox,
      });
      setSuccess("Подключение добавлено");
      setName("");
      setToken("");
      await fetchConnections();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось сохранить подключение");
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async () => {
    setError("");
    setSuccess("");
    setTesting(true);
    try {
      await api.post("/api/v1/me/tinkoff-token/test", { token, is_sandbox: isSandbox });
      setSuccess("Токен прошел проверку");
    } catch (err) {
      setError(err instanceof ApiError ? `Токен не прошел проверку: ${err.message}` : "Не удалось проверить токен");
    } finally {
      setTesting(false);
    }
  };

  const handleActivate = async (connectionID: number) => {
    setError("");
    setSuccess("");
    try {
      await api.patch(`/api/v1/broker/connections/${connectionID}/active`);
      setSuccess("Активное подключение переключено");
      await fetchConnections();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось переключить подключение");
    }
  };

  const handleDelete = async (connection: BrokerConnection) => {
    if (!window.confirm(`Удалить подключение "${connection.name}"?`)) return;

    setError("");
    setSuccess("");
    try {
      await api.delete(`/api/v1/broker/connections/${connection.id}`);
      setSuccess("Подключение удалено");
      await fetchConnections();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось удалить подключение");
    }
  };

  return (
    <div className="page">
      <header className="integration-hero">
        <div>
          <p className="eyebrow">T-Invest API</p>
          <h1>Подключения</h1>
          <p>Добавьте один или несколько токенов T-Invest. Активное подключение используется для поиска инструментов, загрузки свечей и брокерского кабинета.</p>
        </div>
        <span className={`connection-badge ${connections.length ? "connected" : "disconnected"}`}>
          {connections.length ? <CheckCircle2 size={16} aria-hidden="true" /> : <KeyRound size={16} aria-hidden="true" />}
          {connections.length ? `${connections.length} подключ.` : "Не подключено"}
        </span>
      </header>

      <section className="integration-grid">
        <div className={`panel status-panel ${activeConnection ? "success-panel" : "warning-panel"}`}>
          <div className="panel-title-row">
            <div>
              <h2>{activeConnection ? "Активное подключение" : "Подключений нет"}</h2>
              <p>{activeConnection ? `${activeConnection.name} · ${activeConnection.is_sandbox ? "sandbox" : "production"}` : "Добавьте токен, чтобы работать с API T-Invest."}</p>
            </div>
            {activeConnection ? <CheckCircle2 size={24} aria-hidden="true" /> : <KeyRound size={24} aria-hidden="true" />}
          </div>
          <dl className="details-list">
            <div><dt>Режим</dt><dd>{activeConnection?.is_sandbox ? "Sandbox" : activeConnection ? "Production" : "—"}</dd></div>
            <div><dt>Маска</dt><dd><code>{activeConnection?.token_hint || "нет"}</code></dd></div>
            <div><dt>Последняя синхронизация</dt><dd>{formatDate(activeConnection?.last_sync_at)}</dd></div>
          </dl>
          {activeConnection && (
            <Link to="/broker" className="btn primary">
              <RadioTower size={17} aria-hidden="true" />
              Открыть счета
            </Link>
          )}
        </div>

        <form className="panel settings-form" onSubmit={handleCreate}>
          <div className="panel-title-row">
            <div>
              <h2>Новое подключение</h2>
              <p>Токен хранится зашифрованным; в интерфейсе показывается только маска.</p>
            </div>
            <ShieldCheck size={22} aria-hidden="true" />
          </div>
          {error && <div className="error-message">{error}</div>}
          {success && <div className="success-message">{success}</div>}

          <div className="form-group">
            <label>Название</label>
            <input value={name} onChange={(event) => setName(event.target.value)} placeholder="Основной счет" />
          </div>

          <div className="form-group">
            <label>API токен</label>
            <input
              type="password"
              value={token}
              onChange={(event) => setToken(event.target.value)}
              placeholder="t.XXXXX..."
              required
            />
          </div>

          <label className="toggle-row token-mode-toggle">
            <input type="checkbox" checked={isSandbox} onChange={(event) => setIsSandbox(event.target.checked)} />
            <span>Sandbox режим</span>
          </label>

          <div className="toolbar">
            <button type="button" className="btn secondary" onClick={handleTest} disabled={testing || !token}>
              Проверить
            </button>
            <button type="submit" className="btn primary" disabled={loading || !token}>
              <Plus size={17} aria-hidden="true" />
              {loading ? "Добавляем..." : "Добавить"}
            </button>
          </div>
        </form>
      </section>

      <section className="panel connections-panel">
        <div className="panel-title-row">
          <div>
            <h2>Сохраненные подключения</h2>
            <p>{connections.length ? "Можно переключать активное подключение без повторного ввода токена." : "Список пуст."}</p>
          </div>
          <KeyRound size={22} aria-hidden="true" />
        </div>
        <div className="connection-list">
          {connections.map((connection) => (
            <article key={connection.id} className={connection.is_active ? "connection-card active" : "connection-card"}>
              <div>
                <strong>{connection.name}</strong>
                <span>{connection.is_sandbox ? "Sandbox" : "Production"} · <code>{connection.token_hint}</code></span>
                <small>Синхронизация: {formatDate(connection.last_sync_at)}</small>
              </div>
              <div className="connection-actions">
                {!connection.is_active && (
                  <button type="button" className="btn secondary small" onClick={() => handleActivate(connection.id)}>
                    Сделать активным
                  </button>
                )}
                {connection.is_active && <span className="status-chip success">Активно</span>}
                <button type="button" className="icon-button small danger-icon" onClick={() => handleDelete(connection)} title="Удалить">
                  <Trash2 size={16} aria-hidden="true" />
                </button>
              </div>
            </article>
          ))}
          {!connections.length && <div className="empty-panel small-empty">Подключений пока нет</div>}
        </div>
      </section>
    </div>
  );
}
