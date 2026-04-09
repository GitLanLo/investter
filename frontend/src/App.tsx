import { startTransition, useEffect, useState } from "react";
import { formatProbability, loadDashboardData } from "./lib/api";
import type { DashboardData, SignalCard } from "./lib/types";

const shellNav = ["Dashboard", "Asset Details", "Watchlist", "Rules"];

export function App() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    void loadDashboardData()
      .then((nextData) => {
        if (!active) {
          return;
        }
        startTransition(() => {
          setData(nextData);
          setError(null);
        });
      })
      .catch((nextError: Error) => {
        if (!active) {
          return;
        }
        setError(nextError.message);
      });

    return () => {
      active = false;
    };
  }, []);

  const assets = data?.assets ?? [];
  const signals = data?.latestSignals ?? [];
  const topSignal = signals[0];

  return (
    <div className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Sprint 1 frontend prep</p>
          <h1>invest workspace shell</h1>
        </div>
        <nav className="nav">
          {shellNav.map((item) => (
            <a key={item} href={`#${item.toLowerCase().replace(/\s+/g, "-")}`}>
              {item}
            </a>
          ))}
        </nav>
      </header>

      <main className="layout">
        <section className="hero card" id="dashboard">
          <div className="hero-copy">
            <p className="eyebrow">Signal workspace</p>
            <h2>One shell for raw feed, model output and operator actions.</h2>
            <p className="hero-text">
              The app shell already understands backend DTOs, shows latest signal state, and stays usable
              even when the backend is offline by falling back to local mock data.
            </p>
          </div>
          <div className="hero-grid">
            <Metric label="Universe assets" value={String(assets.length || 5)} />
            <Metric label="Latest signal" value={topSignal ? topSignal.assetId : "SBER"} />
            <Metric label="Feed source" value={data?.generatedFrom ?? "loading"} />
            <Metric label="Backend" value={error ? "offline" : "ready"} tone={error ? "warn" : "good"} />
          </div>
        </section>

        <section className="card spotlight" id="asset-details">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Asset spotlight</p>
              <h3>{topSignal ? topSignal.assetId : "SBER"} detail card</h3>
            </div>
            {topSignal ? <span className={`badge badge-${topSignal.direction}`}>{topSignal.direction}</span> : null}
          </div>
          <div className="spotlight-grid">
            <div className="signal-hero">
              <p className="signal-probability">
                {topSignal ? formatProbability(topSignal.probability) : "74%"}
              </p>
              <p className="signal-meta">
                threshold {topSignal ? formatProbability(topSignal.threshold) : "65%"} ·{" "}
                {topSignal?.timeframe ?? "5m"} · {topSignal?.modelVersion ?? "baseline_stub_v1"}
              </p>
            </div>
            <div className="probability-stack">
              {(topSignal ? Object.entries(topSignal.classProbabilities) : []).map(([label, value]) => (
                <ProbabilityBar key={label} label={label} value={value} />
              ))}
            </div>
          </div>
        </section>

        <section className="card" id="watchlist">
          <div className="section-heading">
            <div>
              <p className="eyebrow">MVP watchlist</p>
              <h3>Universe tiles</h3>
            </div>
            <span className="section-note">wired to `/assets`</span>
          </div>
          <div className="asset-grid">
            {assets.map((asset) => (
              <article className="asset-tile" key={asset.id}>
                <div className="asset-tile-row">
                  <strong>{asset.ticker}</strong>
                  <span>{asset.timeframe}</span>
                </div>
                <p>{asset.name}</p>
                <div className="asset-tile-row subtle">
                  <span>{asset.venue}</span>
                  <span>{asset.active ? "active" : "paused"}</span>
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className="card" id="rules">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Contract pack</p>
              <h3>Frontend prep deliverables</h3>
            </div>
            <span className="section-note">Sprint 1 handoff</span>
          </div>
          <div className="doc-grid">
            <DocCard
              title="Screen map"
              body="Dashboard, asset details, watchlist and rule settings are fixed as the main routes."
            />
            <DocCard
              title="Wireframes"
              body="Low-fidelity flows are documented so Sprint 2 can move straight into typed implementation."
            />
            <DocCard
              title="API/domain mapping"
              body="Backend DTOs are translated into frontend domain cards in a dedicated mapping layer."
            />
          </div>
        </section>

        <section className="card signal-feed">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Latest signals</p>
              <h3>Signal rail</h3>
            </div>
            <span className="section-note">{data?.generatedFrom === "mock" ? "mock fallback" : "live API"}</span>
          </div>
          <div className="signal-list">
            {signals.map((signal) => (
              <SignalRow key={signal.id} signal={signal} />
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}

function Metric({
  label,
  value,
  tone = "neutral",
}: {
  label: string;
  value: string;
  tone?: "neutral" | "warn" | "good";
}) {
  return (
    <div className={`metric metric-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function ProbabilityBar({ label, value }: { label: string; value: number }) {
  return (
    <div className="probability-bar">
      <div className="probability-labels">
        <span>{label}</span>
        <strong>{formatProbability(value)}</strong>
      </div>
      <div className="probability-track">
        <div className="probability-fill" style={{ width: `${Math.max(6, value * 100)}%` }} />
      </div>
    </div>
  );
}

function SignalRow({ signal }: { signal: SignalCard }) {
  return (
    <article className="signal-row">
      <div>
        <p className="signal-row-title">
          {signal.assetId} · {signal.direction}
        </p>
        <p className="signal-row-meta">
          {signal.timeframe} · {signal.modelVersion} · {new Date(signal.asOfTime).toLocaleString("ru-RU")}
        </p>
      </div>
      <strong>{formatProbability(signal.probability)}</strong>
    </article>
  );
}

function DocCard({ title, body }: { title: string; body: string }) {
  return (
    <article className="doc-card">
      <h4>{title}</h4>
      <p>{body}</p>
    </article>
  );
}
