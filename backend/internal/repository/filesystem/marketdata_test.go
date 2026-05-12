package filesystem

import (
	"context"
	"testing"
	"time"

	"invest/backend/internal/domain"
)

func TestMarketDataRepositoryListCandlesDedupesTimestamp(t *testing.T) {
	repo := NewMarketDataRepository(t.TempDir())
	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)

	if err := repo.AppendCandles(context.Background(), "SBER", "5m", []domain.Candle{
		{Timestamp: ts, Open: 1, High: 2, Low: 1, Close: 1.5, Volume: 10, Ticker: "SBER", Timeframe: "5m", IngestedAt: ts.Add(time.Minute)},
	}); err != nil {
		t.Fatalf("AppendCandles first batch: %v", err)
	}
	if err := repo.AppendCandles(context.Background(), "SBER", "5m", []domain.Candle{
		{Timestamp: ts, Open: 3, High: 4, Low: 2, Close: 3.5, Volume: 20, Ticker: "SBER", Timeframe: "5m", IngestedAt: ts.Add(2 * time.Minute)},
	}); err != nil {
		t.Fatalf("AppendCandles second batch: %v", err)
	}

	candles, err := repo.ListCandles(context.Background(), "SBER", "5m", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Fatalf("ListCandles: %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("expected 1 deduped candle, got %d", len(candles))
	}
	if candles[0].Close != 3.5 || candles[0].Volume != 20 {
		t.Fatalf("expected latest ingested candle to win, got %+v", candles[0])
	}
}
