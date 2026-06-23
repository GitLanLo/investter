import React, { useEffect, useMemo, useState } from "react";
import { AlertTriangle, BellRing, CheckCircle2, Edit3, PauseCircle, PlayCircle, Plus, RefreshCw, RotateCcw, Save, Trash2 } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";

interface NotificationRule {
  id: number;
  ticker?: string;
  event_type: string;
  target_indicator?: string;
  operator?: string;
  threshold?: number;
  secondary_threshold?: number;
  severity: string;
  direction?: string;
  model_version?: string;
  is_enabled: boolean;
  cooldown_minutes: number;
  expires_at?: string;
  trigger_mode?: string;
  delivery_channels?: string[];
}

interface NotificationEvent {
  id: number;
  rule_id: number;
  signal_event_id?: number;
  ticker?: string;
  event_type: string;
  message: string;
  severity: string;
  delivery_status: string;
  created_at: string;
}

interface RuleFormState {
  ticker: string;
  event_type: string;
  severity: string;
  direction: string;
  threshold: string;
  operator?: string;
  cooldown_minutes: string;
  is_enabled: boolean;
  trigger_mode: string;
}

const supportedEventTypes = new Set([
  "decision_threshold_triggered",
  "classification_success",
  "inference_blocked_by_runtime",
  "policy_promotion",
  "price_level",
]);

const defaultRuleForm: RuleFormState = {
  ticker: "",
  event_type: "decision_threshold_triggered",
  severity: "warning",
  direction: "",
  threshold: "0.65",
  operator: ">",
  cooldown_minutes: "60",
  is_enabled: true,
  trigger_mode: "once",
};

function formatDate(value: string) {
  return new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

function eventLabel(value: string) {
  switch (value) {
    case "decision_threshold_triggered":
      return "Сильный прогноз";
    case "classification_success":
      return "Любой прогноз";
    case "inference_blocked_by_runtime":
      return "Ошибка расчета";
    case "policy_promotion":
      return "Смена ML-политики";
    case "price_level":
      return "Достижение цены";
    case "price":
      return "Рыночная цена";
    case "volume":
      return "Объем торгов";
    case "indicator":
      return "Тех. индикатор";
    default:
      return value || "Событие";
  }
}

function severityLabel(value: string) {
  switch (value) {
    case "critical":
      return "Критично";
    case "warning":
      return "Внимание";
    case "info":
      return "Инфо";
    default:
      return value || "Инфо";
  }
}

function directionLabel(value?: string) {
  if (value === "up") return "Рост";
  if (value === "down") return "Снижение";
  return "Любое";
}

function formatRuleValue(rule: NotificationRule) {
  if (rule.event_type === "inference_blocked_by_runtime" || rule.event_type === "policy_promotion") {
    return "не требуется";
  }
  if (rule.threshold === undefined || rule.threshold === null || Number.isNaN(rule.threshold)) {
    return "не задан";
  }
  if (rule.event_type === "price_level") {
    return `${rule.operator || ""} ${new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 }).format(rule.threshold)}`;
  }
  if (rule.event_type === "decision_threshold_triggered" || rule.event_type === "classification_success") {
    return `${Math.round(rule.threshold * 100)}%`;
  }
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 }).format(rule.threshold);
}

function isSupportedRule(rule: NotificationRule) {
  return supportedEventTypes.has(rule.event_type);
}

function normalizeProbability(raw: string) {
  const value = Number(raw.replace(",", "."));
  if (!Number.isFinite(value) || value <= 0) {
    return undefined;
  }
  if (value > 1 && value <= 100) {
    return value / 100;
  }
  return value;
}

function ruleToForm(rule: NotificationRule): RuleFormState {
  return {
    ticker: rule.ticker || "",
    event_type: rule.event_type,
    severity: rule.severity || "warning",
    direction: rule.direction || "",
    operator: rule.operator || ">",
    threshold: rule.threshold ? String(rule.threshold) : rule.event_type === "decision_threshold_triggered" ? "0.65" : "",
    cooldown_minutes: String(rule.cooldown_minutes ?? 60),
    is_enabled: rule.is_enabled,
    trigger_mode: rule.trigger_mode || "once",
  };
}

export default function AlertsPage() {
  const [rules, setRules] = useState<NotificationRule[]>([]);
  const [events, setEvents] = useState<NotificationEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editingRuleID, setEditingRuleID] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [ruleForm, setRuleForm] = useState<RuleFormState>(defaultRuleForm);
  const navigate = useNavigate();

  const supportedRules = useMemo(() => rules.filter(isSupportedRule), [rules]);
  const unsupportedRules = useMemo(() => rules.filter((rule) => !isSupportedRule(rule)), [rules]);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setRefreshing(true);
    setError("");
    try {
      const [rulesData, eventsData] = await Promise.all([
        api.get<{ items: NotificationRule[] }>("/api/v1/alerts/rules"),
        api.get<{ items: NotificationEvent[] }>("/api/v1/alerts/events?limit=30"),
      ]);
      setRules(rulesData.items || []);
      setEvents(eventsData.items || []);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate("/login");
      } else {
        setError(err instanceof ApiError ? err.message : "Не удалось загрузить уведомления");
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const updateForm = (patch: Partial<RuleFormState>) => {
    setRuleForm((current) => {
      const next = { ...current, ...patch };
      if (patch.event_type === "price_level" && !next.threshold) {
        next.threshold = "";
      } else if (patch.event_type === "decision_threshold_triggered" && !next.threshold) {
        next.threshold = "0.65";
      } else if (patch.event_type && patch.event_type !== "decision_threshold_triggered" && patch.event_type !== "price_level") {
        next.threshold = "";
      }
      return next;
    });
  };

  const buildRulePayload = () => {
    const eventType = ruleForm.event_type;
    const cooldown = Number.parseInt(ruleForm.cooldown_minutes, 10);
    if (!Number.isInteger(cooldown) || cooldown < 0) {
      throw new Error("Cooldown должен быть целым числом минут");
    }

    let threshold: number | undefined;
    if (eventType === "decision_threshold_triggered") {
      threshold = normalizeProbability(ruleForm.threshold);
      if (!threshold || threshold > 1) {
        throw new Error("Порог вероятности должен быть от 0 до 1 или от 1 до 100%");
      }
    } else if (eventType === "price_level") {
      threshold = Number(ruleForm.threshold.replace(",", "."));
      if (Number.isNaN(threshold) || threshold <= 0) {
        throw new Error("Целевая цена должна быть положительным числом");
      }
    }

    return {
      ticker: ruleForm.ticker.trim().toUpperCase() || undefined,
      event_type: eventType,
      operator: eventType === "price_level" ? ruleForm.operator : undefined,
      severity: ruleForm.severity,
      direction: ruleForm.direction || undefined,
      threshold,
      is_enabled: ruleForm.is_enabled,
      cooldown_minutes: cooldown,
      trigger_mode: ruleForm.trigger_mode,
      delivery_channels: ["app"],
    };
  };

  const resetForm = () => {
    setEditingRuleID(null);
    setRuleForm(defaultRuleForm);
    setSuccess("");
    setError("");
  };

  const saveRule = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    setSuccess("");
    setSaving(true);
    try {
      const payload = buildRulePayload();
      if (editingRuleID) {
        await api.patch(`/api/v1/alerts/rules/${editingRuleID}`, payload);
        setSuccess("Правило обновлено");
      } else {
        await api.post("/api/v1/alerts/rules", payload);
        setSuccess("Правило создано");
      }
      resetForm();
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить правило");
    } finally {
      setSaving(false);
    }
  };

  const editRule = (rule: NotificationRule) => {
    if (!isSupportedRule(rule)) {
      setError("Это market-правило создано старой формой и пока не исполняется. Его можно удалить или оставить до добавления market evaluator.");
      return;
    }
    setEditingRuleID(rule.id);
    setRuleForm(ruleToForm(rule));
    setSuccess("");
    setError("");
  };

  const toggleRule = async (rule: NotificationRule) => {
    setError("");
    setSuccess("");
    try {
      await api.patch(`/api/v1/alerts/rules/${rule.id}`, {
        ...rule,
        is_enabled: !rule.is_enabled,
        delivery_channels: rule.delivery_channels?.length ? rule.delivery_channels : ["app"],
        trigger_mode: rule.trigger_mode || "once",
      });
      setSuccess(rule.is_enabled ? "Правило поставлено на паузу" : "Правило включено");
      await fetchData();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось изменить правило");
    }
  };

  const deleteRule = async (id: number) => {
    if (!window.confirm("Удалить правило уведомлений?")) return;

    setError("");
    setSuccess("");
    try {
      await api.delete(`/api/v1/alerts/rules/${id}`);
      if (editingRuleID === id) {
        resetForm();
      }
      setSuccess("Правило удалено");
      await fetchData();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось удалить правило");
    }
  };

  if (loading) {
    return <div className="page"><div className="loading-panel">Загрузка оповещений...</div></div>;
  }

  return (
    <div className="page alerts-page">
      <header className="page-header">
        <div>
          <p className="eyebrow">Контроль</p>
          <h1>Уведомления по бумагам</h1>
        </div>
        <button type="button" className="icon-button" onClick={fetchData} disabled={refreshing} title="Обновить">
          <RefreshCw size={18} aria-hidden="true" />
        </button>
      </header>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      <section className="alerts-overview">
        <div className="alert-kpi">
          <span>Активные правила</span>
          <strong>{supportedRules.filter((rule) => rule.is_enabled).length}</strong>
        </div>
        <div className="alert-kpi">
          <span>События</span>
          <strong>{events.length}</strong>
        </div>
        <div className="alert-kpi">
          <span>Неисполняемые</span>
          <strong>{unsupportedRules.length}</strong>
        </div>
      </section>

      <section className="panel">
        <div className="panel-title-row">
          <div>
            <h2>{editingRuleID ? "Редактировать правило" : "Новое правило"}</h2>
            <p>Исполняются торговые уведомления, ML-прогнозы, ошибки расчета и события смены политики.</p>
          </div>
          <BellRing size={22} aria-hidden="true" />
        </div>
        <form onSubmit={saveRule} className="rule-form advanced-rule-form">
          <label className="form-group">
            <span>Тикер</span>
            <input
              type="text"
              placeholder="SBER или пусто"
              value={ruleForm.ticker}
              onChange={(event) => updateForm({ ticker: event.target.value.toUpperCase() })}
            />
          </label>
          <label className="form-group">
            <span>Событие</span>
            <select value={ruleForm.event_type} onChange={(event) => updateForm({ event_type: event.target.value })}>
              <optgroup label="Торговые">
                <option value="price_level">Достижение цены</option>
              </optgroup>
              <optgroup label="ML Прогнозы">
                <option value="decision_threshold_triggered">Сильный прогноз</option>
                <option value="classification_success">Любой прогноз</option>
                <option value="inference_blocked_by_runtime">Ошибка расчета</option>
                <option value="policy_promotion">Смена ML-политики</option>
              </optgroup>
            </select>
          </label>
          {ruleForm.event_type === "price_level" && (
            <label className="form-group">
              <span>Условие</span>
              <select value={ruleForm.operator} onChange={(event) => updateForm({ operator: event.target.value })}>
                <option value=">">Цена выше (&gt;)</option>
                <option value="<">Цена ниже (&lt;)</option>
                <option value=">=">Цена выше или равна (&ge;)</option>
                <option value="<=">Цена ниже или равна (&le;)</option>
              </select>
            </label>
          )}
          <label className="form-group">
            <span>Направление</span>
            <select value={ruleForm.direction} onChange={(event) => updateForm({ direction: event.target.value })}>
              <option value="">Любое</option>
              <option value="up">Рост</option>
              <option value="down">Снижение</option>
            </select>
          </label>
          <label className="form-group">
            <span>{ruleForm.event_type === "price_level" ? "Целевая цена" : "Порог"}</span>
            <input
              inputMode="decimal"
              placeholder={ruleForm.event_type === "price_level" ? "Например, 305.5" : (ruleForm.event_type === "decision_threshold_triggered" ? "0.65 или 65" : "не требуется")}
              value={ruleForm.threshold}
              disabled={!["decision_threshold_triggered", "price_level"].includes(ruleForm.event_type)}
              onChange={(event) => updateForm({ threshold: event.target.value })}
            />
          </label>
          <label className="form-group">
            <span>Важность</span>
            <select value={ruleForm.severity} onChange={(event) => updateForm({ severity: event.target.value })}>
              <option value="info">Инфо</option>
              <option value="warning">Внимание</option>
              <option value="critical">Критично</option>
            </select>
          </label>
          <label className="form-group">
            <span>Cooldown, мин</span>
            <input
              type="number"
              min="0"
              step="1"
              value={ruleForm.cooldown_minutes}
              onChange={(event) => updateForm({ cooldown_minutes: event.target.value })}
            />
          </label>
          <label className="toggle-row alert-toggle-row">
            <input
              type="checkbox"
              checked={ruleForm.is_enabled}
              onChange={(event) => updateForm({ is_enabled: event.target.checked })}
            />
            <span>Включено</span>
          </label>
          <div className="rule-form-actions">
            {editingRuleID && (
              <button type="button" className="btn secondary" onClick={resetForm}>
                <RotateCcw size={16} aria-hidden="true" />
                Сбросить
              </button>
            )}
            <button type="submit" className="btn primary" disabled={saving}>
              {editingRuleID ? <Save size={17} aria-hidden="true" /> : <Plus size={17} aria-hidden="true" />}
              {saving ? "Сохраняем..." : editingRuleID ? "Сохранить" : "Добавить"}
            </button>
          </div>
        </form>
      </section>

      <section className="rules-grid">
        {rules.map((rule) => {
          const supported = isSupportedRule(rule);
          return (
            <article key={rule.id} className={supported ? "rule-card" : "rule-card unsupported"}>
              <div className="rule-card-header">
                <div>
                  <span className={`status-chip ${rule.is_enabled ? "success" : "warning"}`}>{rule.is_enabled ? "Активно" : "Пауза"}</span>
                  {!supported && <span className="status-chip warning">Не исполняется</span>}
                  <h3>{rule.ticker || "Все бумаги"}</h3>
                  <p>{eventLabel(rule.event_type)}</p>
                </div>
                <div className="rule-card-actions">
                  <button type="button" className="icon-button small" onClick={() => editRule(rule)} title="Редактировать">
                    <Edit3 size={16} aria-hidden="true" />
                  </button>
                  <button type="button" className="icon-button small" onClick={() => toggleRule(rule)} title={rule.is_enabled ? "Пауза" : "Включить"}>
                    {rule.is_enabled ? <PauseCircle size={16} aria-hidden="true" /> : <PlayCircle size={16} aria-hidden="true" />}
                  </button>
                  <button type="button" className="icon-button small danger-icon" onClick={() => deleteRule(rule.id)} title="Удалить">
                    <Trash2 size={16} aria-hidden="true" />
                  </button>
                </div>
              </div>
              <dl>
                <div><dt>Направление</dt><dd>{directionLabel(rule.direction)}</dd></div>
                <div><dt>Порог</dt><dd>{formatRuleValue(rule)}</dd></div>
                <div><dt>Важность</dt><dd>{severityLabel(rule.severity)}</dd></div>
                <div><dt>Cooldown</dt><dd>{rule.cooldown_minutes ?? 0} мин</dd></div>
              </dl>
            </article>
          );
        })}
        {rules.length === 0 && <div className="empty-panel">Правил пока нет</div>}
      </section>

      <section className="panel">
        <div className="panel-title-row">
          <div>
            <h2>История</h2>
            <p>События появляются после запуска прогноза или системной проверки.</p>
          </div>
          <AlertTriangle size={22} aria-hidden="true" />
        </div>
        <ul className="events-list">
          {events.map((event) => (
            <li key={event.id} className={`event-item ${event.severity}`}>
              <CheckCircle2 size={18} aria-hidden="true" />
              <div>
                <strong>{event.ticker || eventLabel(event.event_type)}</strong>
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
