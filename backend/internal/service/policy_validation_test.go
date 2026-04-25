package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"invest/backend/internal/domain"
)

type mockAssetRepo struct {
	asset domain.Asset
}

func (m *mockAssetRepo) GetByID(_ context.Context, id string) (domain.Asset, error) {
	if m.asset.ID == id {
		return m.asset, nil
	}
	return domain.Asset{}, sql.ErrNoRows
}
func (m *mockAssetRepo) List(context.Context) ([]domain.Asset, error) { return nil, nil }
func (m *mockAssetRepo) Upsert(context.Context, domain.Asset) error   { return nil }

type mockMarketDataRepo struct {
	candles []domain.Candle
}

func (m *mockMarketDataRepo) ListCandles(
	_ context.Context,
	_ string,
	_ string,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.Candle, error) {
	filtered := []domain.Candle{}
	for _, c := range m.candles {
		if !c.Timestamp.Before(from) && !c.Timestamp.After(to) {
			filtered = append(filtered, c)
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}
func (m *mockMarketDataRepo) ListFactors(context.Context, string, time.Time, time.Time) ([]domain.FactorBar, error) {
	return nil, nil
}

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func TestRealizedSignalOutcome(t *testing.T) {
	assetRepo := &mockAssetRepo{
		asset: domain.Asset{ID: "SBER", Ticker: "SBER", Timeframe: "5m"},
	}

	tests := []struct {
		name            string
		signal          domain.SignalRun
		candles         []domain.Candle
		expectMatured   bool
		expectHit       bool
		expectReturnPct float64
	}{
		{
			name: "up hit",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionUp,
			},
			candles: []domain.Candle{
				{Timestamp: mustParseTime("2026-04-22T10:00:00Z"), Close: 100},
				{Timestamp: mustParseTime("2026-04-22T10:05:00Z"), Close: 101},
				{Timestamp: mustParseTime("2026-04-22T10:10:00Z"), Close: 105},
			},
			expectMatured:   true,
			expectHit:       true,
			expectReturnPct: 0.05,
		},
		{
			name: "up miss",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionUp,
			},
			candles: []domain.Candle{
				{Timestamp: mustParseTime("2026-04-22T10:00:00Z"), Close: 100},
				{Timestamp: mustParseTime("2026-04-22T10:05:00Z"), Close: 99},
				{Timestamp: mustParseTime("2026-04-22T10:10:00Z"), Close: 95},
			},
			expectMatured:   true,
			expectHit:       false,
			expectReturnPct: -0.05,
		},
		{
			name: "down hit",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionDown,
			},
			candles: []domain.Candle{
				{Timestamp: mustParseTime("2026-04-22T10:00:00Z"), Close: 100},
				{Timestamp: mustParseTime("2026-04-22T10:05:00Z"), Close: 99},
				{Timestamp: mustParseTime("2026-04-22T10:10:00Z"), Close: 90},
			},
			expectMatured:   true,
			expectHit:       true,
			expectReturnPct: -0.10,
		},
		{
			name: "down miss",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionDown,
			},
			candles: []domain.Candle{
				{Timestamp: mustParseTime("2026-04-22T10:00:00Z"), Close: 100},
				{Timestamp: mustParseTime("2026-04-22T10:05:00Z"), Close: 101},
				{Timestamp: mustParseTime("2026-04-22T10:10:00Z"), Close: 102},
			},
			expectMatured:   true,
			expectHit:       false,
			expectReturnPct: 0.02,
		},
		{
			name: "no-trade maturity",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionNone,
			},
			candles: []domain.Candle{
				{Timestamp: mustParseTime("2026-04-22T10:00:00Z"), Close: 100},
				{Timestamp: mustParseTime("2026-04-22T10:05:00Z"), Close: 101},
				{Timestamp: mustParseTime("2026-04-22T10:10:00Z"), Close: 100}, // same price
			},
			expectMatured:   true,
			expectHit:       true,
			expectReturnPct: 0.00,
		},
		{
			name: "pending horizon",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionUp,
			},
			candles: []domain.Candle{
				{Timestamp: mustParseTime("2026-04-22T10:00:00Z"), Close: 100},
				{Timestamp: mustParseTime("2026-04-22T10:05:00Z"), Close: 101},
				// missing the second bar
			},
			expectMatured: false,
		},
		{
			name: "missing candles",
			signal: domain.SignalRun{
				AssetID:         "SBER",
				Timeframe:       "5m",
				HorizonBars:     2,
				AsOfTime:        mustParseTime("2026-04-22T10:00:00Z"),
				SignalDirection: domain.SignalDirectionUp,
			},
			candles: []domain.Candle{
				// No candles at all
			},
			expectMatured: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &PolicyValidationService{}
			svc = svc.WithOutcomeData(assetRepo, &mockMarketDataRepo{candles: tc.candles})

			outcome, ok, err := svc.realizedSignalOutcome(context.Background(), tc.signal)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tc.expectMatured {
				t.Fatalf("expected matured=%v, got %v", tc.expectMatured, ok)
			}
			if ok {
				if outcome.Hit != tc.expectHit {
					t.Errorf("expected hit=%v, got %v", tc.expectHit, outcome.Hit)
				}
				if outcome.ReturnPct != tc.expectReturnPct {
					t.Errorf("expected return_pct=%v, got %v", tc.expectReturnPct, outcome.ReturnPct)
				}
			}
		})
	}
}
