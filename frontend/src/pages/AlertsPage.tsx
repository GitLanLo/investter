import React, { useEffect, useState } from "react";
import { AlertTriangle, BellRing, Plus, Trash2 } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";

interface NotificationRule {
  id: number;
  ticker?: string;
  event_type: string;
  severity: string;
  direction?: string;
  threshold?: number;
  is_enabled: boolean;
  cooldown_minutes: number;
}

interface NotificationEvent {
  id: number;
  ticker?: string;
  message: string;
  severity: string;
  created_at: string;
}

function formatDate(value: string) {
  return new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

function formatRuleValue(rule: NotificationRule) {
  if (!rule.threshold) return "не задан";
  
  if (rule.event_type === "decision_threshold_triggered" || rule.event_type === "classification_success") {
    return `${(rule.threshold * 100).toFixed(0)}%`;
  }
  
  return rule.threshold.toString();
}

function eventLabel(value: string) {
  switch (value) {
    case "decision_threshold_triggered":
      return "Сильный прогноз";
    case "classification_success":
      return "Любой прогноз";
    case "inference_blocked_by_runtime":
      return "Ошибка расчета";
    case "price":
      return "Рыночная цена";
    case "volume":
      return "Объем торгов";
    case "indicator":
      return "Тех. индикатор";
    default:
      return value;
  }
}

export default function AlertsPage() {
  const [rules, setRules] = useState<NotificationRule[]>([]);
  const [events, setEvents] = useState<NotificationEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [newRule, setNewRule] = useState<Partial<NotificationRule>>({
    event_type: "decision_threshold_triggered",
    severity: "info",
    is_enabled: true,
    cooldown_minutes: 60,
  });
  const navigate = useNavigate();

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [rulesData, eventsData] = await Promise.all([
        api.get<{ items: NotificationRule[] }>("/api/v1/alerts/rules"),
        api.get<{ items: NotificationEvent[] }>("/api/v1/alerts/events?limit=20"),
      ]);
      setRules(rulesData.items || []);
      setEvents(eventsData.items || []);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate("/login");
      } else {
        setError("Не удалось загрузить уведомления");
      }
    } finally {
      setLoading(false);
    }
  };

  const createRule = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await api.post("/api/v1/alerts/rules", newRule);
      setNewRule({
        event_type: "decision_threshold_triggered",
        severity: "info",
        is_enabled: true,
        cooldown_minutes: 60,
      });
      fetchData();
    } catch (err) {
      setError("Не удалось создать правило");
    }
  };

  const deleteRule = async (id: number) => {
    setError("");
    try {
      await api.delete(`/api/v1/alerts/rules/${id}`);
      fetchData();
    } catch (err) {
      setError("Не удалось удалить правило");
    }
  };

  if (loading) {
    return <div className="page"><div className="loading-panel">Загрузка оповещений...</div></div>;
  }

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <p className="eyebrow">Контроль</p>
          <h1>Уведомления по бумагам</h1>
        </div>
      </header>

      {error && <div className="error-message">{error}</div>}

      <section className="alerts-overview">
        <div className="alert-kpi">
          <span>Активные правила</span>
          <strong>{rules.filter((rule) => rule.is_enabled).length}</strong>
        </div>
        <div className="alert-kpi">
          <span>События</span>
          <strong>{events.length}</strong>
        </div>
        <div className="alert-kpi">
          <span>Критичные</span>
          <strong>{events.filter((event) => event.severity === "critical").length}</strong>
        </div>
      </section>

      <section className="panel">
        <div className="panel-title-row">
          <div>
            <h2>Новое правило</h2>
            <p>Настройте уведомление только по важным прогнозам.</p>
          </div>
          <BellRing size={22} aria-hidden="true" />
        </div>
        <form onSubmit={createRule} className="rule-form">
          <input
            type="text"
            placeholder="Тикер или пусто"
            value={newRule.ticker || ""}
            onChange={(e) => setNewRule({ ...newRule, ticker: e.target.value.toUpperCase() })}
          />
          <select value={newRule.event_type} onChange={(e) => setNewRule({ ...newRule, event_type: e.target.value })}>
            <option value="decision_threshold_triggered">Сильный прогноз</option>
            <option value="classification_success">Любой прогноз</option>
            <option value="inference_blocked_by_runtime">Ошибка расчета</option>
          </select>
          <select value={newRule.direction || ""} onChange={(e) => setNewRule({ ...newRule, direction: e.target.value })}>
            <option value="">Любое направление</option>
            <option value="up">Рост</option>
            <option value="down">Снижение</option>
          </select>
          <input
            type="number"
            step="0.01"
            placeholder="Порог 0.65"
            value={newRule.threshold ?? ""}
            onChange={(e) => setNewRule({ ...newRule, threshold: e.target.value === "" ? undefined : parseFloat(e.target.value) })}
          />
          <button type="submit" className="btn primary">
            <Plus size={17} aria-hidden="true" />
            Добавить
          </button>
        </form>
      </section>

      <section className="rules-grid">
        {rules.map((rule) => (
          <article key={rule.id} className="rule-card">
            <div>
              <span className={`status-chip ${rule.is_enabled ? "success" : "warning"}`}>{rule.is_enabled ? "Активно" : "Пауза"}</span>
              <h3>{rule.ticker || "Все бумаги"}</h3>
              <p>{eventLabel(rule.event_type)}</p>
            </div>
            <dl>
              <div><dt>Направление</dt><dd>{rule.direction === "up" ? "Рост" : rule.direction === "down" ? "Снижение" : "Любое"}</dd></div>
              <div><dt>Порог</dt><dd>{formatRuleValue(rule)}</dd></div>
            </dl>
            <button type="button" className="icon-button danger-icon" onClick={() => deleteRule(rule.id)} aria-label="Удалить правило">
              <Trash2 size={17} aria-hidden="true" />
            </button>
          </article>
        ))}
        {rules.length === 0 && <div className="empty-panel">Правил пока нет</div>}
      </section>

      <section className="panel">
        <div className="panel-title-row">
          <div>
            <h2>История</h2>
            <p>Последние уведомления по прогнозам.</p>
          </div>
          <AlertTriangle size={22} aria-hidden="true" />
        </div>
        <ul className="events-list">
          {events.map((event) => (
            <li key={event.id} className={`event-item ${event.severity}`}>
              <div>
                <strong>{event.ticker || "Система"}</strong>
                <span>{event.message}</span>
              </div>
              <time>{formatDate(event.created_at)}</time>
            </li>
          ))}
          {events.length === 0 && <li className="empty-list-item">Событий пока нет</li>}
        </ul>
      </section>
    </div>
  );
}
