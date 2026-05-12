package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type MarketDataService struct {
	assetRepo   repository.AssetRepository
	repo        repository.MarketDataRepository
	signalRepo  repository.SignalRunRepository
	instruments InstrumentService
}

func NewMarketDataService(
	assetRepo repository.AssetRepository,
	repo repository.MarketDataRepository,
	signalRepo repository.SignalRunRepository,
	instruments InstrumentService,
) *MarketDataService {
	return &MarketDataService{
		assetRepo:   assetRepo,
		repo:        repo,
		signalRepo:  signalRepo,
		instruments: instruments,
	}
}

func (s *MarketDataService) Refresh(ctx context.Context, userID int64, assetID string, token string, isSandbox bool) error {
	return s.RefreshTimeframe(ctx, userID, assetID, token, isSandbox, "")
}

func (s *MarketDataService) RefreshTimeframe(ctx context.Context, userID int64, assetID string, token string, isSandbox bool, timeframe string) error {
	asset, err := s.loadAsset(ctx, assetID)
	if err != nil {
		return err
	}

	if s.instruments == nil {
		return errors.New("instruments service not available")
	}

	if asset.Timeframe == "" {
		asset.Timeframe = "5m"
	}
	requestedTimeframe, err := normalizeMarketTimeframe(timeframe, asset.Timeframe)
	if err != nil {
		return err
	}

	instruments := InstrumentServiceForSandbox(s.instruments, isSandbox)
	
	var inst domain.TinkoffInstrument
	if asset.InstrumentUID == "" {
		resolved, err := s.resolveInstrument(ctx, instruments, token, asset)
		if err != nil {
			return err
		}
		inst = resolved
		asset = mergeAssetInstrument(asset, inst)
		if err := s.assetRepo.Upsert(ctx, asset); err != nil {
			return err
		}
	} else {
		inst.UID = asset.InstrumentUID
		inst.Ticker = asset.Ticker
	}

	// We look back 30 days to find the last candle. Reading all history is too slow for large datasets.
	lookbackStart := time.Now().UTC().Add(-30 * 24 * time.Hour)
	lastCands, err := s.repo.ListCandles(ctx, asset.Ticker, requestedTimeframe, lookbackStart, time.Time{}, 1)
	if err != nil {
		return err
	}
	from := time.Now().UTC().Add(-initialCandleRefreshLookback(requestedTimeframe))
	if len(lastCands) > 0 {
		// We use the exact timestamp of the last candle to allow fetching updates for incomplete candles.
		// Deduplication logic will handle overwriting the existing candle with newer data.
		from = lastCands[0].Timestamp
	}

	if time.Since(from) <= time.Minute {
		return nil
	}

	newCands, err := instruments.GetCandles(ctx, token, inst.UID, requestedTimeframe, from, time.Now().UTC())
	if err != nil {
		return err
	}

	if len(newCands) > 0 {
		for i := range newCands {
			newCands[i].Ticker = asset.Ticker
			newCands[i].Timeframe = requestedTimeframe
		}
		return s.repo.AppendCandles(ctx, asset.Ticker, requestedTimeframe, newCands)
	}

	return nil
}

func (s *MarketDataService) resolveInstrument(
	ctx context.Context,
	instruments InstrumentService,
	token string,
	asset domain.Asset,
) (domain.TinkoffInstrument, error) {
	if instruments == nil {
		return domain.TinkoffInstrument{}, errors.New("instruments service not available")
	}
	if asset.InstrumentUID != "" {
		return instruments.GetInstrumentByUID(ctx, token, asset.InstrumentUID)
	}
	items, err := instruments.FindInstrument(ctx, token, asset.Ticker)
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}
	for _, item := range items {
		if !strings.EqualFold(item.Ticker, asset.Ticker) {
			continue
		}
		if asset.ClassCode != "" && !strings.EqualFold(item.ClassCode, asset.ClassCode) {
			continue
		}
		if item.UID == "" {
			continue
		}
		return item, nil
	}
	return domain.TinkoffInstrument{}, fmt.Errorf("instrument not found: ticker=%s class_code=%s", asset.Ticker, asset.ClassCode)
}

func initialCandleRefreshLookback(timeframe string) time.Duration {
	switch timeframe {
	case "1m":
		return 24 * time.Hour
	case "5m", "15m":
		return 7 * 24 * time.Hour
	case "1h":
		return 30 * 24 * time.Hour
	case "1d":
		return 365 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func normalizeMarketTimeframe(value string, fallback string) (string, error) {
	timeframe := strings.TrimSpace(value)
	if timeframe == "" {
		timeframe = strings.TrimSpace(fallback)
	}
	if timeframe == "" {
		timeframe = "5m"
	}
	switch timeframe {
	case "1m", "5m", "15m", "1h", "1d":
		return timeframe, nil
	default:
		return "", fmt.Errorf("unsupported timeframe: %s", timeframe)
	}
}

func (s *MarketDataService) ListCandles(
	ctx context.Context,
	assetID string,
	from time.Time,
	to time.Time,
	limit int,
) (domain.Asset, []domain.Candle, error) {
	return s.ListCandlesTimeframe(ctx, assetID, "", from, to, limit)
}

func (s *MarketDataService) ListCandlesTimeframe(
	ctx context.Context,
	assetID string,
	timeframe string,
	from time.Time,
	to time.Time,
	limit int,
) (domain.Asset, []domain.Candle, error) {
	asset, err := s.loadAsset(ctx, assetID)
	if err != nil {
		return domain.Asset{}, nil, err
	}
	requestedTimeframe, err := normalizeMarketTimeframe(timeframe, asset.Timeframe)
	if err != nil {
		return domain.Asset{}, nil, err
	}
	asset.Timeframe = requestedTimeframe

	if limit <= 0 {
		limit = 500
	}
	if limit > 20000 {
		limit = 20000
	}

	items, err := s.repo.ListCandles(ctx, asset.Ticker, requestedTimeframe, from, to, limit)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	return asset, items, nil
}

func (s *MarketDataService) ListFactors(
	ctx context.Context,
	assetID string,
	from time.Time,
	to time.Time,
) (domain.Asset, []domain.FactorBar, error) {
	asset, err := s.loadAsset(ctx, assetID)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	items, err := s.repo.ListFactors(ctx, asset.Timeframe, from, to)
	if err != nil {
		return domain.Asset{}, nil, err
	}

	return asset, items, nil
}

func (s *MarketDataService) GetIndicators(
	ctx context.Context,
	assetID string,
	from time.Time,
	to time.Time,
) (map[string][]IndicatorValue, error) {
	return s.GetIndicatorsTimeframe(ctx, assetID, "", from, to)
}

func (s *MarketDataService) GetIndicatorsTimeframe(
	ctx context.Context,
	assetID string,
	timeframe string,
	from time.Time,
	to time.Time,
) (map[string][]IndicatorValue, error) {
	_, candles, err := s.ListCandlesTimeframe(ctx, assetID, timeframe, from, to, 2000)
	if err != nil {
		return nil, err
	}

	if len(candles) == 0 {
		return nil, nil
	}

	res := make(map[string][]IndicatorValue)
	res["ema_20"] = CalculateEMA(candles, 20)
	res["ema_50"] = CalculateEMA(candles, 50)
	res["rsi_14"] = CalculateRSI(candles, 14)
	res["atr_14"] = CalculateATR(candles, 14)

	return res, nil
}

type AssetSummary struct {
	Asset        domain.Asset      `json:"asset"`
	LastPrice    float64           `json:"last_price"`
	PriceChange  float64           `json:"price_change"`
	LastSignal   *domain.SignalRun `json:"last_signal,omitempty"`
	DataFresh    bool              `json:"data_fresh"`
	LastCandleAt *time.Time        `json:"last_candle_at,omitempty"`
}

func (s *MarketDataService) GetSummary(ctx context.Context, assetID string) (AssetSummary, error) {
	return s.GetSummaryTimeframe(ctx, assetID, "")
}

func (s *MarketDataService) GetSummaryTimeframe(ctx context.Context, assetID string, timeframe string) (AssetSummary, error) {
	asset, err := s.loadAsset(ctx, assetID)
	if err != nil {
		return AssetSummary{}, err
	}
	requestedTimeframe, err := normalizeMarketTimeframe(timeframe, asset.Timeframe)
	if err != nil {
		return AssetSummary{}, err
	}
	asset.Timeframe = requestedTimeframe

	ticker := strings.TrimSpace(asset.Ticker)
	if ticker == "" {
		ticker = strings.TrimSpace(asset.ID)
	}

	// Use 60-day window for robustness
	lookbackStart := time.Now().UTC().Add(-60 * 24 * time.Hour)
	candles, err := s.repo.ListCandles(ctx, ticker, requestedTimeframe, lookbackStart, time.Time{}, 2)
	if err != nil {
		return AssetSummary{Asset: asset}, nil // Return asset even if candles fail
	}

	summary := AssetSummary{Asset: asset}
	if len(candles) > 0 {
		latest := candles[len(candles)-1]
		summary.LastPrice = latest.Close
		summary.LastCandleAt = &latest.Timestamp

		// Calculate 24h change
		dayAgo := latest.Timestamp.Add(-24 * time.Hour)
		// Search in a wider window around 24h ago to be more robust (24h +/- 12h)
		prevCands, _ := s.repo.ListCandles(ctx, ticker, requestedTimeframe, dayAgo.Add(-12*time.Hour), dayAgo.Add(12*time.Hour), 100)
		if len(prevCands) > 0 {
			// Find the candle closest to dayAgo
			best := prevCands[0]
			bestDiff := best.Timestamp.Sub(dayAgo).Abs()
			for _, c := range prevCands {
				diff := c.Timestamp.Sub(dayAgo).Abs()
				if diff < bestDiff {
					best = c
					bestDiff = diff
				}
			}
			summary.PriceChange = latest.Close - best.Close
		} else if len(candles) > 1 {
			// Fallback to previous candle if 24h ago not found
			previous := candles[len(candles)-2]
			summary.PriceChange = latest.Close - previous.Close
		}

		summary.DataFresh = time.Since(latest.Timestamp) < 4*time.Hour
	}

	return summary, nil
}

func (s *MarketDataService) loadAsset(ctx context.Context, assetID string) (domain.Asset, error) {
	if assetID == "" {
		return domain.Asset{}, errors.New("assetID must not be empty")
	}

	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Asset{}, ErrAssetNotFound
		}
		return domain.Asset{}, err
	}

	return asset, nil
}
