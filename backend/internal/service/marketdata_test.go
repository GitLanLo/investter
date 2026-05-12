package service

import (
	"context"
	"testing"
	"time"

	"invest/backend/internal/domain"
)

func TestMarketDataRefreshResolvesMissingInstrumentUIDAndAppendsCandles(t *testing.T) {
	now := time.Now().UTC()
	assetRepo := &MockAssetRepo{Assets: map[string]domain.Asset{
		"SBER": {
			ID:        "SBER",
			Ticker:    "SBER",
			Name:      "Sberbank",
			ClassCode: "TQBR",
			Exchange:  "MOEX",
			Timeframe: "5m",
			Currency:  "rub",
			Lot:       10,
			Figi:      "old-figi",
			IsActive:  true,
		},
	}}
	marketData := &MockMarketDataRepo{Candles: map[string][]domain.Candle{}}
	instruments := &marketDataInstrumentMock{
		findResults: []domain.TinkoffInstrument{{
			UID:               "uid-sber",
			Figi:              "figi-sber",
			Ticker:            "SBER",
			Name:              "Sberbank PJSC",
			ClassCode:         "TQBR",
			Exchange:          "MOEX",
			InstrumentType:    "share",
			Lot:               10,
			Currency:          "rub",
			APITradeAvailable: true,
		}},
		candles: []domain.Candle{{
			Timestamp: now.Add(-time.Hour),
			Open:      310,
			High:      312,
			Low:       309,
			Close:     311,
			Volume:    1000,
		}},
	}

	svc := NewMarketDataService(assetRepo, marketData, nil, instruments)
	if err := svc.Refresh(context.Background(), 7, "SBER", "token", false); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if instruments.findQuery != "SBER" {
		t.Fatalf("expected ticker search for SBER, got %q", instruments.findQuery)
	}
	if instruments.candlesUID != "uid-sber" {
		t.Fatalf("expected candles request by resolved uid, got %q", instruments.candlesUID)
	}
	if !instruments.candlesFrom.Before(now.Add(-6 * 24 * time.Hour)) {
		t.Fatalf("expected initial 5m refresh lookback of about 7 days, got %s", instruments.candlesFrom)
	}

	updated := assetRepo.Assets["SBER"]
	if updated.InstrumentUID != "uid-sber" || updated.Figi != "figi-sber" || updated.Name != "Sberbank PJSC" {
		t.Fatalf("asset metadata was not merged from Tinkoff: %+v", updated)
	}

	got := marketData.Candles["SBER"]
	if len(got) != 1 {
		t.Fatalf("expected 1 appended candle, got %d", len(got))
	}
	if got[0].Ticker != "SBER" || got[0].Timeframe != "5m" {
		t.Fatalf("expected appended candle ticker/timeframe to be set, got %+v", got[0])
	}
}

func TestMarketDataRefreshTimeframeUsesRequestedIntervalWithoutChangingAssetDefault(t *testing.T) {
	now := time.Now().UTC()
	assetRepo := &MockAssetRepo{Assets: map[string]domain.Asset{
		"SBER": {
			ID:            "SBER",
			Ticker:        "SBER",
			Name:          "Sberbank",
			ClassCode:     "TQBR",
			Exchange:      "MOEX",
			Timeframe:     "5m",
			Currency:      "rub",
			InstrumentUID: "uid-sber",
			IsActive:      true,
		},
	}}
	marketData := &MockMarketDataRepo{Candles: map[string][]domain.Candle{}}
	instruments := &marketDataInstrumentMock{
		findResults: []domain.TinkoffInstrument{{
			UID:               "uid-sber",
			Figi:              "figi-sber",
			Ticker:            "SBER",
			Name:              "Sberbank",
			ClassCode:         "TQBR",
			Exchange:          "MOEX",
			InstrumentType:    "share",
			Currency:          "rub",
			APITradeAvailable: true,
		}},
		candles: []domain.Candle{{
			Timestamp: now.Add(-time.Hour),
			Open:      310,
			High:      312,
			Low:       309,
			Close:     311,
			Volume:    1000,
		}},
	}

	svc := NewMarketDataService(assetRepo, marketData, nil, instruments)
	if err := svc.RefreshTimeframe(context.Background(), 7, "SBER", "token", false, "15m"); err != nil {
		t.Fatalf("RefreshTimeframe() error = %v", err)
	}

	if instruments.candlesTimeframe != "15m" {
		t.Fatalf("expected Tinkoff candles request for 15m, got %q", instruments.candlesTimeframe)
	}
	if marketData.LastListTimeframe != "15m" || marketData.LastAppendTimeframe != "15m" {
		t.Fatalf("expected storage to use 15m, list=%q append=%q", marketData.LastListTimeframe, marketData.LastAppendTimeframe)
	}
	if assetRepo.Assets["SBER"].Timeframe != "5m" {
		t.Fatalf("expected stored asset default timeframe to stay 5m, got %q", assetRepo.Assets["SBER"].Timeframe)
	}

	asset, _, err := svc.ListCandlesTimeframe(context.Background(), "SBER", "15m", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Fatalf("ListCandlesTimeframe() error = %v", err)
	}
	if asset.Timeframe != "15m" {
		t.Fatalf("expected response asset timeframe to reflect selected interval, got %q", asset.Timeframe)
	}
}

type marketDataInstrumentMock struct {
	findResults      []domain.TinkoffInstrument
	candles          []domain.Candle
	findQuery        string
	candlesUID       string
	candlesTimeframe string
	candlesFrom      time.Time
}

func (m *marketDataInstrumentMock) FindInstrument(_ context.Context, _ string, query string) ([]domain.TinkoffInstrument, error) {
	m.findQuery = query
	return m.findResults, nil
}

func (m *marketDataInstrumentMock) GetInstrumentByUID(_ context.Context, _ string, uid string) (domain.TinkoffInstrument, error) {
	for _, item := range m.findResults {
		if item.UID == uid {
			return item, nil
		}
	}
	return domain.TinkoffInstrument{}, ErrAssetNotFound
}

func (m *marketDataInstrumentMock) GetCandles(_ context.Context, _ string, uid string, timeframe string, from time.Time, _ time.Time) ([]domain.Candle, error) {
	m.candlesUID = uid
	m.candlesTimeframe = timeframe
	m.candlesFrom = from
	return m.candles, nil
}

func (m *marketDataInstrumentMock) IsMarketOpen(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}
