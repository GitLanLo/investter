import React, { useEffect, useMemo, useState } from "react";
import { Activity, BrainCircuit, DatabaseZap, RefreshCw, Rocket, ShieldCheck, Users } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";

interface AdminData {
  monitoring: any;
  activeModel: any;
  research: any;
  validationRuns: any;
  jobs: any;
  users: UserAccess[];
}

interface UserAccess {
  id: number;
  email: string;
  role: string;
  permissions: string[];
  created_at: string;
}

type AdminTab = "overview" | "models" | "policy" | "users";

const PERMISSIONS = [
  { value: "market_access", label: "Рынок" },
  { value: "ml_admin", label: "ML" },
  { value: "user_admin", label: "Пользователи" },
];

function formatDate(value?: string) {
  if (!value) return "n/a";
  return new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

function metric(value?: number) {
  return Number.isFinite(value) ? value!.toFixed(3) : "n/a";
}

function percent(value?: number) {
  return Number.isFinite(value) ? `${(value! * 100).toFixed(1)}%` : "n/a";
}

export default function AdminMLPage() {
  const [data, setData] = useState<AdminData | null>(null);
  const [activeTab, setActiveTab] = useState<AdminTab>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setError("");
    try {
      const [monitoring, activeModel, research, validationRuns, jobs, users] = await Promise.all([
        api.get<any>("/api/v1/admin/ml/monitoring/summary"),
        api.get<any>("/api/v1/admin/ml/models/active").catch(() => null),
        api.get<any>("/api/v1/admin/ml/research/overview"),
        api.get<any>("/api/v1/admin/ml/policy/validation-runs?limit=10"),
        api.get<any>("/api/v1/jobs/runs?limit=10"),
        api.get<{ items: UserAccess[] }>("/api/v1/admin/users").catch(() => ({ items: [] })),
      ]);

      setData({ monitoring, activeModel, research, validationRuns, jobs, users: users.items || [] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate("/login");
      } else if (err instanceof ApiError && err.status === 403) {
        setError("Недостаточно прав для панели администрирования");
      } else {
        setError("Не удалось загрузить панель администрирования");
      }
    } finally {
      setLoading(false);
    }
  };

  const promotePolicy = async (id: number) => {
    setError("");
    try {
      await api.post(`/api/v1/admin/ml/policy/validation-runs/${id}/promote`, {});
      fetchData();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось промотировать политику");
    }
  };

  const updateUserAccess = async (user: UserAccess, patch: Partial<UserAccess>) => {
    const next = { ...user, ...patch };
    setError("");
    try {
      await api.patch(`/api/v1/admin/users/${user.id}/access`, {
        role: next.role,
        permissions: next.permissions,
      });
      fetchData();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось обновить права пользователя");
    }
  };

  const togglePermission = (user: UserAccess, permission: string) => {
    const has = user.permissions.includes(permission);
    const permissions = has ? user.permissions.filter((item) => item !== permission) : [...user.permissions, permission];
    updateUserAccess(user, { permissions });
  };

  const validationItems = data?.validationRuns?.items || [];
  const jobItems = data?.jobs?.items || [];
  const monitoringItems = data?.monitoring?.items || [];
  const gridRows = useMemo(() => {
    const rows = data?.research?.grid_matrix?.results || data?.research?.grid_matrix?.items || [];
    return Array.isArray(rows) ? rows.slice(0, 6) : [];
  }, [data]);

  if (loading) {
    return <div className="page"><div className="loading-panel">Загрузка панели...</div></div>;
  }

  return (
    <div className="page admin-console-page">
      <header className="admin-hero">
        <div>
          <p className="eyebrow">Администрирование</p>
          <h1>ML Control Center</h1>
          <p>Модели, качество, jobs и доступы пользователей в одном рабочем контуре.</p>
        </div>
        <button type="button" className="btn secondary" onClick={fetchData}>
          <RefreshCw size={17} aria-hidden="true" />
          Обновить
        </button>
      </header>

      {error && <div className="error-message">{error}</div>}

      <section className="admin-kpi-grid">
        <div className="admin-kpi-card">
          <span>Runtime</span>
          <strong>{data?.activeModel?.runtime_status || "n/a"}</strong>
          <small>{data?.activeModel?.model_version || "нет активной модели"}</small>
        </div>
        <div className="admin-kpi-card">
          <span>Monitoring</span>
          <strong>{data?.monitoring?.status || "unknown"}</strong>
          <small>{monitoringItems.length} health items</small>
        </div>
        <div className="admin-kpi-card">
          <span>Research grid</span>
          <strong>{data?.research?.grid_matrix ? "ready" : "missing"}</strong>
          <small>{gridRows.length || 0} вариантов в матрице</small>
        </div>
        <div className="admin-kpi-card">
          <span>Users</span>
          <strong>{data?.users.length || 0}</strong>
          <small>роль и permissions</small>
        </div>
      </section>

      <div className="admin-tabs" role="tablist" aria-label="Разделы администрирования">
        <button type="button" className={activeTab === "overview" ? "active" : ""} onClick={() => setActiveTab("overview")}>Обзор</button>
        <button type="button" className={activeTab === "models" ? "active" : ""} onClick={() => setActiveTab("models")}>Модели</button>
        <button type="button" className={activeTab === "policy" ? "active" : ""} onClick={() => setActiveTab("policy")}>Policy</button>
        <button type="button" className={activeTab === "users" ? "active" : ""} onClick={() => setActiveTab("users")}>Доступы</button>
      </div>

      {activeTab === "overview" && (
        <div className="admin-grid">
          <section className="panel admin-card">
            <div className="panel-title-row">
              <div>
                <h2>Состояние системы</h2>
                <p>Ошибки данных, runtime и уведомлений.</p>
              </div>
              <Activity size={22} aria-hidden="true" />
            </div>
            <ul className="monitoring-items">
              {monitoringItems.map((item: any, idx: number) => (
                <li key={idx} className={item.severity || item.status}>
                  <strong>{item.name || item.item}:</strong> {item.message}
                </li>
              ))}
              {monitoringItems.length === 0 && <li>Нет элементов мониторинга</li>}
            </ul>
          </section>

          <section className="panel admin-card table-panel">
            <div className="panel-title-row">
              <div>
                <h2>Последние jobs</h2>
                <p>Refresh, signals, outcomes и плановые операции.</p>
              </div>
              <DatabaseZap size={22} aria-hidden="true" />
            </div>
            <AdminJobsTable jobs={jobItems} />
          </section>
        </div>
      )}

      {activeTab === "models" && (
        <div className="admin-grid">
          <section className="panel admin-card">
            <div className="panel-title-row">
              <div>
                <h2>Активная runtime модель</h2>
                <p>То, что реально используется кнопкой прогноза.</p>
              </div>
              <BrainCircuit size={22} aria-hidden="true" />
            </div>
            <dl className="details-list">
              <div><dt>Версия</dt><dd>{data?.activeModel?.model_version || "n/a"}</dd></div>
              <div><dt>Семейство</dt><dd>{data?.activeModel?.model_family || data?.activeModel?.model_type || "n/a"}</dd></div>
              <div><dt>Runtime</dt><dd><span className={`status-chip ${data?.activeModel?.runtime_status === "available" ? "success" : "warning"}`}>{data?.activeModel?.runtime_status || "n/a"}</span></dd></div>
              <div><dt>Timeframe</dt><dd>{data?.activeModel?.timeframe || "n/a"}</dd></div>
              <div><dt>Horizon</dt><dd>{data?.activeModel?.horizon_bars || "n/a"}</dd></div>
            </dl>
          </section>

          <section className="panel admin-card table-panel">
            <div className="panel-title-row">
              <div>
                <h2>Research matrix</h2>
                <p>Сравнимые кандидаты по timeframe/horizon.</p>
              </div>
              <BrainCircuit size={22} aria-hidden="true" />
            </div>
            <div className="responsive-table">
              <table className="data-table compact">
                <thead>
                  <tr><th>TF</th><th>Horizon</th><th>Macro F1</th><th>Balanced acc</th></tr>
                </thead>
                <tbody>
                  {gridRows.map((row: any, idx: number) => (
                    <tr key={idx}>
                      <td>{row.timeframe || row.tf || "n/a"}</td>
                      <td>{row.horizon || row.horizon_bars || "n/a"}</td>
                      <td>{metric(row.macro_f1 ?? row.f1_macro)}</td>
                      <td>{metric(row.balanced_accuracy)}</td>
                    </tr>
                  ))}
                  {gridRows.length === 0 && <tr><td colSpan={4} className="empty-cell">Матрица не найдена</td></tr>}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      )}

      {activeTab === "policy" && (
        <section className="panel table-panel">
          <div className="panel-title-row">
            <div>
              <h2>Policy promotion</h2>
              <p>Ручной контроль решений перед включением в production.</p>
            </div>
            <Rocket size={22} aria-hidden="true" />
          </div>
          <div className="responsive-table">
            <table className="data-table compact">
              <thead>
                <tr><th>Версия</th><th>State</th><th>F1 val</th><th>Precision</th><th>Coverage</th><th></th></tr>
              </thead>
              <tbody>
                {validationItems.map((run: any) => (
                  <tr key={run.id}>
                    <td>{run.model_version}</td>
                    <td><span className={`status-chip ${run.decision_state}`}>{run.decision_state}</span></td>
                    <td>{metric(run.validation?.actionable_f1)}</td>
                    <td>{percent(run.validation?.precision)}</td>
                    <td>{percent(run.validation?.coverage)}</td>
                    <td className="row-actions">
                      {run.decision_state !== "active" && <button type="button" className="btn small secondary" onClick={() => promotePolicy(run.id)}>Promote</button>}
                    </td>
                  </tr>
                ))}
                {validationItems.length === 0 && <tr><td colSpan={6} className="empty-cell">Validation runs пока нет</td></tr>}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {activeTab === "users" && (
        <section className="panel table-panel">
          <div className="panel-title-row">
            <div>
              <h2>Пользователи и доступы</h2>
              <p>Главный админ выдает доступ к рынку, ML панели и управлению пользователями.</p>
            </div>
            <Users size={22} aria-hidden="true" />
          </div>
          <div className="responsive-table">
            <table className="data-table compact">
              <thead>
                <tr><th>Email</th><th>Роль</th><th>Permissions</th><th>Создан</th></tr>
              </thead>
              <tbody>
                {(data?.users || []).map((user) => (
                  <tr key={user.id}>
                    <td>{user.email}</td>
                    <td>
                      <select value={user.role} onChange={(event) => updateUserAccess(user, { role: event.target.value })}>
                        <option value="user">user</option>
                        <option value="admin">admin</option>
                        <option value="super_admin">super_admin</option>
                      </select>
                    </td>
                    <td>
                      <div className="permission-pills">
                        {PERMISSIONS.map((permission) => (
                          <button
                            key={permission.value}
                            type="button"
                            className={user.permissions.includes(permission.value) ? "active" : ""}
                            onClick={() => togglePermission(user, permission.value)}
                          >
                            <ShieldCheck size={13} aria-hidden="true" />
                            {permission.label}
                          </button>
                        ))}
                      </div>
                    </td>
                    <td>{formatDate(user.created_at)}</td>
                  </tr>
                ))}
                {(data?.users || []).length === 0 && <tr><td colSpan={4} className="empty-cell">Пользователей нет</td></tr>}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}

function AdminJobsTable({ jobs }: { jobs: any[] }) {
  return (
    <div className="responsive-table">
      <table className="data-table compact">
        <thead>
          <tr><th>Job</th><th>Статус</th><th>Завершен</th></tr>
        </thead>
        <tbody>
          {jobs.map((job: any) => (
            <tr key={job.id}>
              <td>{job.job_type}</td>
              <td><span className={`status-chip ${job.status}`}>{job.status}</span></td>
              <td>{formatDate(job.finished_at)}</td>
            </tr>
          ))}
          {jobs.length === 0 && <tr><td colSpan={3} className="empty-cell">Jobs пока нет</td></tr>}
        </tbody>
      </table>
    </div>
  );
}
