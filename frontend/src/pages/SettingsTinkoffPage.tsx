import React, { useEffect, useState } from "react";
import { CheckCircle2, KeyRound, ShieldCheck, Trash2 } from "lucide-react";
import { api, ApiError } from "../shared/api/client";

interface TinkoffStatus {
  connected: boolean;
  token_hint?: string;
  is_sandbox: boolean;
}

export default function SettingsTinkoffPage() {
  const [token, setToken] = useState("");
  const [isSandbox, setIsSandbox] = useState(false);
  const [status, setStatus] = useState<TinkoffStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    fetchStatus();
  }, []);

  const fetchStatus = async () => {
    try {
      const data = await api.get<TinkoffStatus>("/api/v1/me/tinkoff-token/status");
      setStatus(data);
      setIsSandbox(data.is_sandbox);
    } catch (err) {
      console.error("Failed to fetch Tinkoff status", err);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);

    try {
      await api.post("/api/v1/me/tinkoff-token", { token, is_sandbox: isSandbox });
      setSuccess("Токен сохранен");
      setToken("");
      fetchStatus();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось сохранить токен");
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async () => {
    setError("");
    setSuccess("");
    setLoading(true);

    try {
      await api.post("/api/v1/me/tinkoff-token/test", { token, is_sandbox: isSandbox });
      setSuccess("Токен прошел проверку");
    } catch (err) {
      setError(err instanceof ApiError ? `Токен не прошел проверку: ${err.message}` : "Не удалось проверить токен");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm("Удалить сохраненный Tinkoff токен?")) return;

    try {
      await api.delete("/api/v1/me/tinkoff-token");
      setSuccess("Токен удален");
      fetchStatus();
    } catch (err) {
      setError("Не удалось удалить токен");
    }
  };

  return (
    <div className="page">
      <header className="integration-hero">
        <div>
          <p className="eyebrow">Источник данных</p>
          <h1>Tinkoff Invest</h1>
          <p>Подключите токен, чтобы искать бумаги, загружать свечи и строить прогнозы по реальным данным.</p>
        </div>
        <span className={`connection-badge ${status?.connected ? "connected" : "disconnected"}`}>
          {status?.connected ? <CheckCircle2 size={16} aria-hidden="true" /> : <KeyRound size={16} aria-hidden="true" />}
          {status?.connected ? "Подключено" : "Не подключено"}
        </span>
      </header>

      <section className="integration-grid">
        <div className={`panel status-panel ${status?.connected ? "success-panel" : "warning-panel"}`}>
          <div className="panel-title-row">
            <div>
              <h2>{status?.connected ? "Токен активен" : "Токен не подключен"}</h2>
              <p>{status?.connected ? `Контур: ${status.is_sandbox ? "sandbox" : "production"}` : "Без токена поиск и загрузка свечей недоступны."}</p>
            </div>
            {status?.connected ? <CheckCircle2 size={24} aria-hidden="true" /> : <KeyRound size={24} aria-hidden="true" />}
          </div>
          <dl className="details-list">
            <div><dt>Режим</dt><dd>{status?.is_sandbox ? "Sandbox" : "Production"}</dd></div>
            <div><dt>Маска</dt><dd><code>{status?.token_hint || "нет"}</code></dd></div>
          </dl>
          {status?.connected && (
            <button type="button" className="btn danger" onClick={handleDelete}>
              <Trash2 size={17} aria-hidden="true" />
              Удалить токен
            </button>
          )}
        </div>

        <form className="panel settings-form" onSubmit={handleSave}>
          <div className="panel-title-row">
            <div>
              <h2>Обновить подключение</h2>
              <p>Для торговых свечей используйте production-токен.</p>
            </div>
            <ShieldCheck size={22} aria-hidden="true" />
          </div>
          {error && <div className="error-message">{error}</div>}
          {success && <div className="success-message">{success}</div>}

          <div className="form-group">
            <label>API токен</label>
            <input
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="t.XXXXX..."
              required={!status?.connected}
            />
          </div>

          <label className="toggle-row token-mode-toggle">
            <input type="checkbox" checked={isSandbox} onChange={(e) => setIsSandbox(e.target.checked)} />
            <span>Sandbox режим</span>
          </label>

          <div className="toolbar">
            <button type="button" className="btn secondary" onClick={handleTest} disabled={loading || !token}>
              Проверить
            </button>
            <button type="submit" className="btn primary" disabled={loading || !token}>
              {loading ? "Сохраняем..." : "Сохранить"}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
}
