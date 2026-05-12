import React, { useEffect, useMemo, useState } from "react";
import { AlertCircle, ArrowUpRight, BrainCircuit, Check, Heart, Plus, Search, Trash2 } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { api, ApiError } from "../shared/api/client";

interface Asset {
  id: string;
  ticker: string;
  name: string;
  exchange: string;
  timeframe: string;
  instrument_uid: string;
  class_code: string;
  currency: string;
  model_supported: boolean;
}

interface WatchlistItem {
  asset_id: string;
  position: number;
  asset?: Asset;
}

interface WatchlistResponse {
  watchlist_id: number;
  name: string;
  items: WatchlistItem[];
}

interface AssetsResponse {
  items: Asset[];
}

interface Instrument {
  uid: string;
  ticker: string;
  name: string;
  exchange: string;
  class_code: string;
  instrument_type: string;
}

interface FreshnessItem {
  asset_id: string;
  data_fresh: boolean;
  last_candle_at?: string;
  last_price: number;
  price_change: number;
}

interface FreshnessResponse {
  items: FreshnessItem[];
}

type ViewMode = "watchlist" | "all";

function formatDate(value?: string) {
  if (!value) return "нет данных";
  return new Date(value).toLocaleString("ru-RU", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

function formatCurrency(value?: string) {
  if (!value) return "₽";
  return value.toUpperCase() === "RUB" ? "₽" : value.toUpperCase();
}

function marketLabel(asset: Asset) {
  return `${asset.class_code || asset.exchange || "MOEX"} · ${formatCurrency(asset.currency)}`;
}

interface NotificationRule {
  id: number;
  ticker?: string;
  is_enabled: boolean;
}

export default function WatchlistPage() {
  const [watchlist, setWatchlist] = useState<WatchlistResponse | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [freshness, setFreshness] = useState<FreshnessResponse | null>(null);
  const [rules, setRules] = useState<NotificationRule[]>([]);
  const [viewMode, setViewMode] = useState<ViewMode>("watchlist");
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<Instrument[]>([]);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [forecastingAsset, setForecastingAsset] = useState("");
  const [error, setError] = useState("");
  const [tokenRequired, setTokenRequired] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    fetchData();
  }, []);

  const watchlistByAsset = useMemo(() => {
    const out = new Map<string, WatchlistItem>();
    watchlist?.items.forEach((item) => out.set(item.asset_id, item));
    return out;
  }, [watchlist]);

  const watchlistByInstrument = useMemo(() => {
    const out = new Set<string>();
    watchlist?.items.forEach((item) => {
      if (item.asset?.instrument_uid) out.add(item.asset.instrument_uid);
      if (item.asset?.ticker) out.add(item.asset.ticker.toUpperCase());
      out.add(item.asset_id);
    });
    return out;
  }, [watchlist]);

  const freshnessByAsset = useMemo(() => {
    const out = new Map<string, FreshnessItem>();
    freshness?.items.forEach((item) => out.set(item.asset_id, item));
    return out;
  }, [freshness]);

  const activeRulesByTicker = useMemo(() => {
    const out = new Map<string, number>();
    rules.forEach((rule) => {
      if (rule.is_enabled && rule.ticker) {
        out.set(rule.ticker, (out.get(rule.ticker) || 0) + 1);
      }
    });
    return out;
  }, [rules]);

  const displayedAssets = useMemo(() => {
    if (viewMode === "watchlist") {
      return (watchlist?.items || [])
        .map((item) => item.asset)
        .filter((asset): asset is Asset => Boolean(asset));
    }
    return assets;
  }, [assets, viewMode, watchlist]);

  const fetchData = async () => {
    setError("");
    setLoading(true);
    try {
      const [watchlistData, assetsData, rulesData] = await Promise.all([
        api.get<WatchlistResponse>("/api/v1/watchlist"),
        api.get<AssetsResponse>("/api/v1/assets"),
        api.get<{ items: NotificationRule[] }>("/api/v1/alerts/rules").catch(() => ({ items: [] })),
      ]);
      setWatchlist(watchlistData);
      setAssets(assetsData.items || []);
      setRules(rulesData.items || []);

      // Загружаем freshness асинхронно, так как он может отвечать до 60 секунд
      api.get<FreshnessResponse>("/api/v1/watchlist/freshness")
        .then(setFreshness)
        .catch(() => setFreshness(null));

    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate("/login");
      } else {
        setError("Не удалось загрузить инструменты");
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;

    setError("");
    setTokenRequired(false);
    setSearching(true);
    try {
      const data = await api.get<{ items: Instrument[] }>(`/api/v1/instruments/search?query=${encodeURIComponent(searchQuery.trim())}`);
      setSearchResults(data.items || []);
    } catch (err) {
      if (err instanceof ApiError && err.code === "tinkoff_token_missing") {
        setTokenRequired(true);
        setError("Для поиска инструментов нужен Tinkoff API токен");
      } else {
        setError("Не удалось выполнить поиск");
      }
    } finally {
      setSearching(false);
    }
  };

  const openInstrument = async (instrumentUID: string) => {
    setError("");
    try {
      const asset = await api.post<Asset>(`/api/v1/instruments/${instrumentUID}/asset`, {});
      await fetchData();
      navigate(`/instruments/${asset.id}`);
    } catch (err) {
      if (err instanceof ApiError && err.code === "tinkoff_token_missing") {
        setTokenRequired(true);
        setError("Для открытия нового инструмента нужен Tinkoff API токен");
      } else {
        setError("Не удалось открыть инструмент");
      }
    }
  };

  const addToWatchlist = async (instrumentUID: string) => {
    setError("");
    try {
      await api.post("/api/v1/watchlist", { instrument_uid: instrumentUID });
      setSearchQuery("");
      setSearchResults([]);
      await fetchData();
      setViewMode("watchlist");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Не удалось добавить инструмент");
      }
    }
  };

  const addAssetToWatchlist = async (assetID: string) => {
    setError("");
    try {
      await api.post("/api/v1/watchlist", { asset_id: assetID });
      await fetchData();
    } catch (err) {
      setError("Не удалось добавить инструмент");
    }
  };

  const removeFromWatchlist = async (assetID: string) => {
    setError("");
    try {
      await api.delete(`/api/v1/watchlist/${assetID}`);
      await fetchData();
    } catch (err) {
      setError("Не удалось удалить инструмент");
    }
  };

  const runForecast = async (assetID: string) => {
    setError("");
    setForecastingAsset(assetID);
    try {
      await api.post(`/api/v1/assets/${assetID}/analysis/run`, {});
      navigate(`/instruments/${assetID}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось построить прогноз");
    } finally {
      setForecastingAsset("");
    }
  };

  if (loading) {
    return <div className="page"><div className="loading-panel">Загрузка инструментов...</div></div>;
  }

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <p className="eyebrow">Рынок</p>
          <h1>Инструменты</h1>
        </div>
        <div className="segmented-control" aria-label="Режим списка">
          <button type="button" className={viewMode === "watchlist" ? "active" : ""} onClick={() => setViewMode("watchlist")}>
            Мой список {watchlist?.items.length ?? 0}
          </button>
          <button type="button" className={viewMode === "all" ? "active" : ""} onClick={() => setViewMode("all")}>
            Весь рынок {assets.length}
          </button>
        </div>
      </header>

      {error && (
        <div className="error-message">
          <AlertCircle size={17} aria-hidden="true" />
          <span>{error}</span>
          {tokenRequired && <Link to="/settings/tinkoff">Открыть настройки</Link>}
        </div>
      )}

      <section className="panel search-panel">
        <form onSubmit={handleSearch} className="search-form">
          <div className="search-input">
            <Search size={18} aria-hidden="true" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Найти через Tinkoff: SBER, GAZP, OZON..."
            />
          </div>
          <button type="submit" className="btn primary" disabled={searching || !searchQuery.trim()}>
            {searching ? "Ищем..." : "Найти"}
          </button>
        </form>

        {searchResults.length > 0 && (
          <div className="search-results-list">
            {searchResults.map((res) => {
              const alreadyAdded = watchlistByInstrument.has(res.uid) || watchlistByInstrument.has(res.ticker.toUpperCase());
              return (
                <article key={res.uid} className="result-row">
                  <div>
                    <strong>{res.ticker}</strong>
                    <span>{res.name}</span>
                    <small>{res.class_code || res.exchange || res.instrument_type}</small>
                  </div>
                  <div className="row-actions">
                    <button type="button" className="btn small secondary" onClick={() => openInstrument(res.uid)}>
                      <ArrowUpRight size={15} aria-hidden="true" />
                      Открыть
                    </button>
                    <button type="button" className="btn small primary" onClick={() => addToWatchlist(res.uid)} disabled={alreadyAdded}>
                      {alreadyAdded ? <Check size={15} aria-hidden="true" /> : <Plus size={15} aria-hidden="true" />}
                      {alreadyAdded ? "В списке" : "Добавить"}
                    </button>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>

      <section className="panel table-panel market-table-panel">
        <div className="market-table-header">
          <div>
            <p className="eyebrow">{viewMode === "watchlist" ? "Ваш портфель мониторинга" : "Каталог всех бумаг"}</p>
            <h2>{viewMode === "watchlist" ? "Отслеживаемые бумаги" : "Все инструменты"}</h2>
          </div>
          <span>{displayedAssets.length} позиций</span>
        </div>
        <div className="responsive-table">
          <table className="data-table market-table">
            <thead>
              <tr>
                <th>Инструмент</th>
                <th>Цена</th>
                <th>Изм. (24ч)</th>
                <th>Мониторинг</th>
                <th>Прогноз</th>
                <th className="row-actions">Действия</th>
              </tr>
            </thead>
            <tbody>
              {displayedAssets.map((asset) => {
                const fresh = freshnessByAsset.get(asset.id);
                const inWatchlist = watchlistByAsset.has(asset.id);
                const activeAlerts = activeRulesByTicker.get(asset.ticker) || 0;

                const price = fresh?.last_price || 0;
                const change = fresh?.price_change || 0;
                const prevPrice = price - change;
                const changePct = prevPrice !== 0 ? (change / prevPrice) * 100 : 0;
                const changeClass = change >= 0 ? "positive" : "negative";

                return (
                  <tr key={asset.id}>
                    <td>
                      <Link to={`/instruments/${asset.id}`} className="instrument-cell">
                        <strong>{asset.ticker}</strong>
                        <span>{asset.name}</span>
                        <small>{marketLabel(asset)}</small>
                      </Link>
                    </td>
                    <td>
                      {fresh ? (
                        fresh.last_price > 0 ? (
                          <strong>{new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 }).format(fresh.last_price)}</strong>
                        ) : (
                          <span className="muted">—</span>
                        )
                      ) : (
                        <span className="loading-dots">...</span>
                      )}
                    </td>
                    <td>
                      {fresh ? (
                        fresh.last_price > 0 ? (
                          <em className={`change-pct ${fresh.price_change >= 0 ? "positive" : "negative"}`}>
                            {fresh.price_change >= 0 ? "+" : ""}
                            {((fresh.price_change / (fresh.last_price - fresh.price_change)) * 100).toFixed(2)}%
                          </em>
                        ) : (
                          <span className="muted">—</span>
                        )
                      ) : (
                        <span className="loading-dots">...</span>
                      )}
                    </td>
                    <td>
                      <Link to="/alerts" className={activeAlerts > 0 ? "status-chip success" : "status-chip"}>
                        {activeAlerts > 0 ? `Активен (${activeAlerts})` : "Нет правил"}
                      </Link>
                    </td>
                    <td className="forecast-cell">
                      <button type="button" className="btn small primary table-action-button" onClick={() => runForecast(asset.id)} disabled={forecastingAsset === asset.id || !fresh?.data_fresh}>
                        <BrainCircuit size={15} aria-hidden="true" />
                        {forecastingAsset === asset.id ? "Счет..." : "Прогноз"}
                      </button>
                    </td>
                    <td className="row-actions">
                      <div className="table-actions">
                        <Link className="btn small secondary table-action-button" to={`/instruments/${asset.id}`}>
                          <ArrowUpRight size={15} aria-hidden="true" />
                          Открыть
                        </Link>
                        {inWatchlist ? (
                          <button type="button" className="icon-button small danger-icon" onClick={() => removeFromWatchlist(asset.id)} aria-label="Убрать из списка">
                            <Trash2 size={15} aria-hidden="true" />
                          </button>
                        ) : (
                          <button type="button" className="icon-button small" onClick={() => addAssetToWatchlist(asset.id)} aria-label="Добавить в список">
                            <Heart size={15} aria-hidden="true" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
              {displayedAssets.length === 0 && (
                <tr>
                  <td colSpan={8} className="empty-cell">Нет бумаг в выбранном списке</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
