import { mockDashboardData } from "./mock";
import type { AssetCard, AssetDTO, DashboardData, SignalCard, SignalDTO } from "./types";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

function mapAsset(dto: AssetDTO): AssetCard {
  return {
    id: dto.id,
    ticker: dto.ticker,
    name: dto.name,
    venue: dto.exchange,
    timeframe: dto.timeframe,
    active: dto.is_active,
  };
}

function mapSignal(dto: SignalDTO): SignalCard {
  return {
    id: dto.id,
    assetId: dto.asset_id,
    direction: dto.signal_direction,
    state: dto.signal_state,
    probability: dto.signal_probability,
    threshold: dto.threshold,
    asOfTime: dto.as_of_time,
    timeframe: dto.timeframe,
    modelVersion: dto.model_version,
    classProbabilities: dto.class_probabilities,
  };
}

async function fetchJson<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`);
  if (!response.ok) {
    throw new Error(`request failed for ${path}: ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export async function loadDashboardData(): Promise<DashboardData> {
  try {
    const [assets, signals] = await Promise.all([
      fetchJson<AssetDTO[]>("/assets"),
      fetchJson<{ items: SignalDTO[] }>("/signals/latest?limit=5"),
    ]);

    return {
      generatedFrom: "api",
      assets: assets.map(mapAsset),
      latestSignals: signals.items.map(mapSignal),
    };
  } catch {
    return mockDashboardData;
  }
}

export function formatProbability(value: number): string {
  return `${Math.round(value * 100)}%`;
}
