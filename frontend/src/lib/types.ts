export type AssetDTO = {
  id: string;
  ticker: string;
  name: string;
  exchange: string;
  timeframe: string;
  is_active: boolean;
};

export type SignalDTO = {
  id?: number | string;
  asset_id: string;
  model_version: string;
  as_of_time: string;
  signal_state: string;
  signal_direction: string;
  signal_probability: number;
  class_probabilities: Record<string, number>;
  threshold: number;
  timeframe: string;
  horizon_bars: number;
  created_at?: string;
};

export type AssetCard = {
  id: string;
  ticker: string;
  name: string;
  venue: string;
  timeframe: string;
  active: boolean;
};

export type SignalCard = {
  id: string;
  assetId: string;
  direction: string;
  state: string;
  probability: number;
  threshold: number;
  asOfTime: string;
  timeframe: string;
  modelVersion: string;
  classProbabilities: Record<string, number>;
};

export type DashboardData = {
  assets: AssetCard[];
  latestSignals: SignalCard[];
  generatedFrom: "api" | "mock";
};
