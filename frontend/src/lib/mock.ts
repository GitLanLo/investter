import type { DashboardData } from "./types";

export const mockDashboardData: DashboardData = {
  generatedFrom: "mock",
  assets: [
    { id: "SBER", ticker: "SBER", name: "Sberbank", venue: "MOEX", timeframe: "5m", active: true },
    { id: "GAZP", ticker: "GAZP", name: "Gazprom", venue: "MOEX", timeframe: "5m", active: true },
    { id: "LKOH", ticker: "LKOH", name: "Lukoil", venue: "MOEX", timeframe: "5m", active: true }
  ],
  latestSignals: [
    {
      id: 1,
      assetId: "SBER",
      direction: "up",
      state: "actionable",
      probability: 0.74,
      threshold: 0.65,
      asOfTime: "2026-04-09T09:55:00Z",
      timeframe: "5m",
      modelVersion: "baseline_stub_v1",
      classProbabilities: { up_signal: 0.74, down_signal: 0.13, no_trade: 0.13 }
    },
    {
      id: 2,
      assetId: "GAZP",
      direction: "down",
      state: "watch",
      probability: 0.58,
      threshold: 0.65,
      asOfTime: "2026-04-09T09:55:00Z",
      timeframe: "5m",
      modelVersion: "baseline_stub_v1",
      classProbabilities: { up_signal: 0.16, down_signal: 0.58, no_trade: 0.26 }
    }
  ]
};
