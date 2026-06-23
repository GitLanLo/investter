package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

const (
	JobTypeWatchlistRefresh       = "watchlist_refresh"
	JobTypeWatchlistSignalRefresh = "watchlist_signal_refresh"
)

// WatchlistRefreshService handles the live loop for refreshing watchlist instrument data
// and generating signals for model-supported instruments.
type WatchlistRefreshService struct {
	watchlistRepo      repository.WatchlistRepository
	assetRepo          repository.AssetRepository
	marketData         repository.MarketDataRepository
	jobRepo            repository.JobRunRepository
	signalRepo         repository.SignalRunRepository
	analysis           *AnalysisService
	instruments        InstrumentService
	tinkoffCredentials *TinkoffCredentialService
	systemToken        string
	logger             *log.Logger
	notifier           *NotificationService
	factorSpecs        []FactorRefreshSpec
}

func NewWatchlistRefreshService(
	watchlistRepo repository.WatchlistRepository,
	assetRepo repository.AssetRepository,
	marketData repository.MarketDataRepository,
	jobRepo repository.JobRunRepository,
	signalRepo repository.SignalRunRepository,
	analysis *AnalysisService,
	instruments InstrumentService,
	tinkoffCredentials *TinkoffCredentialService,
	systemToken string,
	logger *log.Logger,
	notifier *NotificationService,
) *WatchlistRefreshService {
	return &WatchlistRefreshService{
		watchlistRepo:      watchlistRepo,
		assetRepo:          assetRepo,
		marketData:         marketData,
		jobRepo:            jobRepo,
		signalRepo:         signalRepo,
		analysis:           analysis,
		instruments:        instruments,
		tinkoffCredentials: tinkoffCredentials,
		systemToken:        systemToken,
		logger:             logger,
		notifier:           notifier,
	}
}

func (s *WatchlistRefreshService) WithFactorSpecs(specs []FactorRefreshSpec) *WatchlistRefreshService {
	if s == nil {
		return s
	}
	s.factorSpecs = append([]FactorRefreshSpec(nil), specs...)
	return s
}

type FactorRefreshSpec struct {
	Alias     string
	Ticker    string
	Timeframe string
	ClassCode string
}

type universeConfigFile struct {
	Factors []struct {
		Alias     string `json:"alias"`
		Ticker    string `json:"ticker"`
		Timeframe string `json:"timeframe"`
		ClassCode string `json:"class_code"`
	} `json:"factors"`
}

func LoadUniverseFactorSpecs(path string) ([]FactorRefreshSpec, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload universeConfigFile
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	specs := make([]FactorRefreshSpec, 0, len(payload.Factors))
	for _, item := range payload.Factors {
		if item.Alias == "" || item.Ticker == "" || item.Timeframe == "" {
			continue
		}
		specs = append(specs, FactorRefreshSpec{
			Alias:     item.Alias,
			Ticker:    item.Ticker,
			Timeframe: item.Timeframe,
			ClassCode: item.ClassCode,
		})
	}
	return specs, nil
}

// FreshnessItem represents the freshness state of a single watchlist instrument.
type FreshnessItem struct {
	AssetID        string     `json:"asset_id"`
	Ticker         string     `json:"ticker"`
	Name           string     `json:"name"`
	LastCandleAt   *time.Time `json:"last_candle_at,omitempty"`
	LastSignalAt   *time.Time `json:"last_signal_at,omitempty"`
	DataFresh      bool       `json:"data_fresh"`
	SignalFresh    bool       `json:"signal_fresh"`
	StaleReason    string     `json:"stale_reason,omitempty"`
	ModelSupported bool       `json:"model_supported"`
	LastPrice      float64    `json:"last_price"`
	PriceChange    float64    `json:"price_change"`
}

// FreshnessSummary is the response for the freshness endpoint.
type FreshnessSummary struct {
	GeneratedAt   time.Time       `json:"generated_at"`
	TotalItems    int             `json:"total_items"`
	FreshData     int             `json:"fresh_data"`
	StaleData     int             `json:"stale_data"`
	FreshSignals  int             `json:"fresh_signals"`
	StaleSignals  int             `json:"stale_signals"`
	WatchlistOnly int             `json:"watchlist_only"`
	Items         []FreshnessItem `json:"items"`
}

// dataFreshnessThreshold defines the maximum age of candle data before it's considered stale.
const dataFreshnessThreshold = 2 * time.Hour

// signalFreshnessThreshold defines the maximum age of a signal before it's considered stale.
const signalFreshnessThreshold = 4 * time.Hour

func (s *WatchlistRefreshService) GetFreshness(ctx context.Context) (FreshnessSummary, error) {
	assets, err := s.assetRepo.List(ctx)
	if err != nil {
		return FreshnessSummary{}, fmt.Errorf("assets list failed: %w", err)
	}

	return s.getFreshnessForAssets(ctx, assets, s.instruments, s.systemToken)
}

func (s *WatchlistRefreshService) getFreshnessForAssets(
	ctx context.Context,
	assets []domain.Asset,
	instruments InstrumentService,
	token string,
) (FreshnessSummary, error) {
	now := time.Now().UTC()
	lookbackStart := now.Add(-30 * 24 * time.Hour)
	summary := FreshnessSummary{
		GeneratedAt: now,
		TotalItems:  len(assets),
		Items:       make([]FreshnessItem, len(assets)),
	}

	type result struct {
		index int
		item  FreshnessItem
	}
	resChan := make(chan result, len(assets))

	for i, asset := range assets {
		go func(idx int, a domain.Asset) {
			fi := FreshnessItem{
				AssetID:        a.ID,
				Ticker:         a.Ticker,
				Name:           a.Name,
				ModelSupported: a.ModelSupported,
			}

			ticker := strings.TrimSpace(a.Ticker)
			if ticker == "" {
				ticker = strings.TrimSpace(a.ID)
			}
			timeframe := strings.TrimSpace(a.Timeframe)
			if timeframe == "" {
				timeframe = "5m"
			}

			timeframesToTry := []string{timeframe, "1h", "1d", "15m", "1m"}
			var candles []domain.Candle
			var usedTimeframe string

			for _, tf := range timeframesToTry {
				if tf == "" {
					continue
				}
				// Use a shorter timeout for individual candle lookups to prevent total request hang
				lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				c, _ := s.marketData.ListCandles(lookupCtx, ticker, tf, lookbackStart, time.Time{}, 2)
				cancel()
				if len(c) > 0 {
					candles = c
					usedTimeframe = tf
					break
				}
			}

			if len(candles) > 0 {
				latest := candles[len(candles)-1]
				fi.LastCandleAt = &latest.Timestamp
				fi.LastPrice = latest.Close

				// Calculate 24h change using the same timeframe that had data
				dayAgo := latest.Timestamp.Add(-24 * time.Hour)
				// Small timeout for change calculation too
				changeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				prevCands, _ := s.marketData.ListCandles(changeCtx, ticker, usedTimeframe, dayAgo.Add(-12*time.Hour), dayAgo.Add(12*time.Hour), 100)
				cancel()
				if len(prevCands) > 0 {
					best := prevCands[0]
					bestDiff := best.Timestamp.Sub(dayAgo).Abs()
					for _, c := range prevCands {
						diff := c.Timestamp.Sub(dayAgo).Abs()
						if diff < bestDiff {
							best = c
							bestDiff = diff
						}
					}
					fi.PriceChange = latest.Close - best.Close
				} else if len(candles) > 1 {
					fi.PriceChange = latest.Close - candles[0].Close
				}

				if now.Sub(latest.Timestamp) < dataFreshnessThreshold {
					fi.DataFresh = true
				} else {
					fi.StaleReason = "candle data older than " + dataFreshnessThreshold.String()
					if instruments != nil && token != "" && a.Exchange != "" {
						marketOpen, err := instruments.IsMarketOpen(ctx, token, a.Exchange)
						if err == nil && !marketOpen {
							fi.DataFresh = true
							fi.StaleReason = "market closed; " + fi.StaleReason
						}
					}
				}
			} else {
				fi.StaleReason = "no candle data available in any timeframe"
			}
			resChan <- result{idx, fi}
		}(i, asset)
	}

	for i := 0; i < len(assets); i++ {
		res := <-resChan
		summary.Items[res.index] = res.item
		if res.item.DataFresh {
			summary.FreshData++
		} else {
			summary.StaleData++
		}
	}

	return summary, nil
}

func (s *WatchlistRefreshService) GetFreshnessForUser(ctx context.Context, userID int64) (FreshnessSummary, error) {
	watchlist, err := s.watchlistRepo.GetOrCreateByName(ctx, userID, "default")
	if err != nil {
		return FreshnessSummary{}, fmt.Errorf("watchlist init failed: %w", err)
	}

	wlItems, err := s.watchlistRepo.ListItems(ctx, watchlist.ID)
	if err != nil {
		return FreshnessSummary{}, fmt.Errorf("watchlist items failed: %w", err)
	}

	now := time.Now().UTC()
	wlAssetIDs := make(map[string]struct{})
	assets := make([]domain.Asset, 0, len(wlItems))
	for _, item := range wlItems {
		wlAssetIDs[item.AssetID] = struct{}{}
		asset, err := s.assetRepo.GetByID(ctx, item.AssetID)
		if err != nil {
			return FreshnessSummary{}, fmt.Errorf("watchlist asset %s failed: %w", item.AssetID, err)
		}
		assets = append(assets, asset)
	}

	instruments, token := s.userInstrumentClient(ctx, userID)
	summary, err := s.getFreshnessForAssets(ctx, assets, instruments, token)
	if err != nil {
		return summary, err
	}

	for i := range summary.Items {
		item := &summary.Items[i]
		if _, ok := wlAssetIDs[item.AssetID]; ok && item.ModelSupported {
			signals, _ := s.signalRepo.ListByAsset(ctx, userID, item.AssetID, 1)
			if len(signals) > 0 {
				lastSignal := signals[0].CreatedAt
				item.LastSignalAt = &lastSignal
				if now.Sub(lastSignal) < signalFreshnessThreshold {
					item.SignalFresh = true
					summary.FreshSignals++
				} else {
					summary.StaleSignals++
				}
			} else {
				summary.StaleSignals++
			}
		} else if _, ok := wlAssetIDs[item.AssetID]; ok {
			summary.WatchlistOnly++
			summary.StaleSignals++
		}
	}

	return summary, nil
}

func (s *WatchlistRefreshService) userInstrumentClient(ctx context.Context, userID int64) (InstrumentService, string) {
	if s == nil {
		return nil, ""
	}
	if userID <= 0 || s.tinkoffCredentials == nil {
		return s.instruments, s.systemToken
	}
	token, isSandbox, err := s.tinkoffCredentials.GetToken(ctx, userID)
	if err != nil || token == "" {
		return s.instruments, s.systemToken
	}
	return InstrumentServiceForSandbox(s.instruments, isSandbox), token
}

// WatchlistRefreshResult holds the results from a watchlist refresh job.
type WatchlistRefreshResult struct {
	TotalInstruments int      `json:"total_instruments"`
	Refreshed        int      `json:"refreshed"`
	Failed           int      `json:"failed"`
	FactorsChecked   int      `json:"factors_checked"`
	FactorsRefreshed int      `json:"factors_refreshed"`
	FreshFactors     int      `json:"fresh_factors"`
	StaleFactors     int      `json:"stale_factors"`
	FailedFactors    int      `json:"failed_factors"`
	FailedAssetIDs   []string `json:"failed_asset_ids,omitempty"`
	FailedFactorsIDs []string `json:"failed_factors,omitempty"`
}

// RunWatchlistRefresh refreshes market data for all watchlist instruments (or all assets if userID=0).
func (s *WatchlistRefreshService) RunWatchlistRefresh(ctx context.Context, limit int) (domain.JobRun, error) {
	return s.RunWatchlistRefreshForUser(ctx, 0, limit)
}

func (s *WatchlistRefreshService) RunWatchlistRefreshForUser(ctx context.Context, userID int64, limit int) (domain.JobRun, error) {
	if s.jobRepo == nil {
		return domain.JobRun{}, ErrJobsUnavailable
	}

	run, err := s.jobRepo.Create(ctx, domain.JobRun{
		JobType: JobTypeWatchlistRefresh,
		Status:  domain.JobStatusRunning,
		Payload: map[string]any{"limit": limit, "user_id": userID},
	})
	if err != nil {
		return domain.JobRun{}, err
	}

	result, jobErr := s.doWatchlistRefresh(ctx, userID, limit)

	payload := map[string]any{
		"limit":             limit,
		"user_id":           userID,
		"total_instruments": result.TotalInstruments,
		"refreshed":         result.Refreshed,
		"failed":            result.Failed,
		"factors_checked":   result.FactorsChecked,
		"factors_refreshed": result.FactorsRefreshed,
		"fresh_factors":     result.FreshFactors,
		"stale_factors":     result.StaleFactors,
		"failed_factors":    result.FailedFactors,
	}
	if len(result.FailedAssetIDs) > 0 {
		payload["failed_asset_ids"] = result.FailedAssetIDs
	}
	if len(result.FailedFactorsIDs) > 0 {
		payload["failed_factor_aliases"] = result.FailedFactorsIDs
	}

	status := domain.JobStatusSucceeded
	errMsg := ""
	if jobErr != nil {
		status = domain.JobStatusFailed
		errMsg = jobErr.Error()
	} else if result.Failed > 0 || result.FailedFactors > 0 {
		status = domain.JobStatusFailed
		errMsg = fmt.Sprintf("%d items failed; %d factors failed", result.Failed, result.FailedFactors)
	}

	return s.jobRepo.Finish(ctx, run.ID, status, payload, errMsg)
}

func (s *WatchlistRefreshService) doWatchlistRefresh(ctx context.Context, userID int64, limit int) (WatchlistRefreshResult, error) {
	var assetIDs []string
	instruments, token, err := s.refreshInstrumentClient(ctx, userID)
	if err != nil {
		return WatchlistRefreshResult{}, err
	}

	if userID > 0 {
		watchlist, err := s.watchlistRepo.GetOrCreateByName(ctx, userID, "default")
		if err != nil {
			return WatchlistRefreshResult{}, fmt.Errorf("watchlist init: %w", err)
		}
		items, err := s.watchlistRepo.ListItems(ctx, watchlist.ID)
		if err != nil {
			return WatchlistRefreshResult{}, fmt.Errorf("watchlist list: %w", err)
		}
		for _, it := range items {
			assetIDs = append(assetIDs, it.AssetID)
		}
	} else {
		// System-wide update for all active assets
		assets, err := s.assetRepo.List(ctx)
		if err != nil {
			return WatchlistRefreshResult{}, fmt.Errorf("assets list: %w", err)
		}
		for _, a := range assets {
			if a.IsActive {
				assetIDs = append(assetIDs, a.ID)
			}
		}
	}

	if limit > 0 && len(assetIDs) > limit {
		assetIDs = assetIDs[:limit]
	}

	result := WatchlistRefreshResult{TotalInstruments: len(assetIDs)}
	factorTimeframes := make(map[string]struct{})

	// Parallelize with concurrency limit
	type assetResult struct {
		assetID   string
		failed    bool
		timeframe string
	}
	resChan := make(chan assetResult, len(assetIDs))
	sem := make(chan struct{}, 5) // Concurrency limit of 5

	for _, id := range assetIDs {
		go func(assetID string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			res := assetResult{assetID: assetID}
			defer func() { resChan <- res }()

			// Per-asset context with timeout
			assetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			asset, err := s.assetRepo.GetByID(assetCtx, assetID)
			if err != nil {
				res.failed = true
				return
			}

			if asset.Timeframe != "" {
				res.timeframe = asset.Timeframe
			}

			if instruments != nil && asset.Ticker != "" {
				inst, err := s.resolveAssetInstrument(assetCtx, instruments, token, asset)
				if err != nil {
					res.failed = true
					return
				}
				asset = mergeAssetInstrument(asset, inst)

				lookbackStart := time.Now().UTC().Add(-30 * 24 * time.Hour)
				lastCands, _ := s.marketData.ListCandles(assetCtx, asset.Ticker, asset.Timeframe, lookbackStart, time.Time{}, 1)
				from := time.Now().UTC().Add(-24 * time.Hour)
				if len(lastCands) > 0 {
					from = lastCands[0].Timestamp
				}

				if time.Since(from) > time.Minute {
					newCands, err := instruments.GetCandles(assetCtx, token, inst.UID, asset.Timeframe, from, time.Now().UTC())
					if err != nil {
						res.failed = true
						if s.logger != nil {
							s.logger.Printf("watchlist refresh: asset %s GetCandles failed: %v", asset.ID, err)
						}
					} else if len(newCands) > 0 {
						for i := range newCands {
							newCands[i].Ticker = asset.Ticker
							newCands[i].Timeframe = asset.Timeframe
						}
						if err := s.marketData.AppendCandles(assetCtx, asset.Ticker, asset.Timeframe, newCands); err != nil {
							res.failed = true
						} else if s.notifier != nil {
							latestCandle := newCands[len(newCands)-1]
							_ = s.notifier.EvaluatePriceAlerts(context.Background(), asset.Ticker, latestCandle.Close)
						}
					}
				}

				if err := s.assetRepo.Upsert(assetCtx, asset); err != nil {
					if s.logger != nil {
						s.logger.Printf("watchlist refresh: asset %s upsert failed: %v", asset.ID, err)
					}
				}
			}
		}(id)
	}

	for i := 0; i < len(assetIDs); i++ {
		res := <-resChan
		if res.failed {
			result.Failed++
			result.FailedAssetIDs = append(result.FailedAssetIDs, res.assetID)
		} else {
			result.Refreshed++
		}
		if res.timeframe != "" {
			factorTimeframes[res.timeframe] = struct{}{}
		}
	}

	s.refreshFactorStatus(ctx, &result, factorTimeframes, instruments, token)

	return result, nil
}

func (s *WatchlistRefreshService) refreshInstrumentClient(ctx context.Context, userID int64) (InstrumentService, string, error) {
	if s == nil || s.instruments == nil {
		return nil, "", nil
	}
	if userID <= 0 {
		if s.systemToken == "" {
			return nil, "", ErrTinkoffUnavailable
		}
		return s.instruments, s.systemToken, nil
	}
	if s.tinkoffCredentials == nil {
		if s.systemToken == "" {
			return nil, "", ErrTinkoffUnavailable
		}
		return s.instruments, s.systemToken, nil
	}
	token, isSandbox, err := s.tinkoffCredentials.GetToken(ctx, userID)
	if err != nil || token == "" {
		return nil, "", ErrTinkoffUnavailable
	}
	return InstrumentServiceForSandbox(s.instruments, isSandbox), token, nil
}

func (s *WatchlistRefreshService) refreshFactorStatus(
	ctx context.Context,
	result *WatchlistRefreshResult,
	timeframes map[string]struct{},
	instruments InstrumentService,
	token string,
) {
	if result == nil || s.marketData == nil {
		return
	}

	now := time.Now().UTC()
	s.refreshConfiguredFactors(ctx, result, timeframes, now, instruments, token)

	for timeframe := range timeframes {
		listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		factors, err := s.marketData.ListFactors(listCtx, timeframe, now.Add(-24*time.Hour), now)
		cancel()
		if err != nil {
			result.StaleFactors++
			continue
		}

		latestByAlias := make(map[string]time.Time)
		for _, factor := range factors {
			result.FactorsChecked++
			latest := latestByAlias[factor.Factor]
			if factor.Timestamp.After(latest) {
				latestByAlias[factor.Factor] = factor.Timestamp
			}
		}

		for _, latest := range latestByAlias {
			if now.Sub(latest) < dataFreshnessThreshold {
				result.FreshFactors++
			} else {
				result.StaleFactors++
			}
		}
	}
}

func (s *WatchlistRefreshService) resolveAssetInstrument(
	ctx context.Context,
	instruments InstrumentService,
	token string,
	asset domain.Asset,
) (domain.TinkoffInstrument, error) {
	if instruments == nil {
		return domain.TinkoffInstrument{}, ErrTinkoffUnavailable
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

func mergeAssetInstrument(asset domain.Asset, inst domain.TinkoffInstrument) domain.Asset {
	asset.Name = inst.Name
	asset.Figi = inst.Figi
	asset.Ticker = inst.Ticker
	asset.ClassCode = inst.ClassCode
	asset.Exchange = inst.Exchange
	asset.InstrumentUID = inst.UID
	asset.InstrumentType = inst.InstrumentType
	asset.Lot = inst.Lot
	asset.Currency = inst.Currency
	asset.APITradeAvailable = inst.APITradeAvailable
	if inst.First1MinCandleDate != nil {
		asset.First1MinCandleDate = inst.First1MinCandleDate
	}
	if inst.First1DayCandleDate != nil {
		asset.First1DayCandleDate = inst.First1DayCandleDate
	}
	return asset
}

func (s *WatchlistRefreshService) refreshConfiguredFactors(
	ctx context.Context,
	result *WatchlistRefreshResult,
	timeframes map[string]struct{},
	now time.Time,
	instruments InstrumentService,
	token string,
) {
	if instruments == nil || token == "" || len(s.factorSpecs) == 0 {
		return
	}

	type factorRes struct {
		alias     string
		refreshed bool
		failed    bool
	}
	resChan := make(chan factorRes, len(s.factorSpecs))

	for _, spec := range s.factorSpecs {
		go func(sp FactorRefreshSpec) {
			res := factorRes{alias: sp.Alias}
			defer func() { resChan <- res }()

			if _, ok := timeframes[sp.Timeframe]; !ok && len(timeframes) > 0 {
				return
			}

			factorCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			lookbackStart := now.Add(-30 * 24 * time.Hour)
			from := lookbackStart
			factors, err := s.marketData.ListFactors(factorCtx, sp.Timeframe, lookbackStart, time.Time{})
			if err != nil {
				res.failed = true
				return
			}
			for _, factor := range factors {
				if factor.Factor == sp.Alias && factor.Timestamp.After(from) {
					from = factor.Timestamp
				}
			}
			if now.Sub(from) <= time.Minute {
				return
			}

			inst, err := s.resolveFactorInstrument(factorCtx, instruments, token, sp)
			if err != nil {
				res.failed = true
				return
			}

			candles, err := instruments.GetCandles(factorCtx, token, inst.UID, sp.Timeframe, from, now)
			if err != nil {
				res.failed = true
				return
			}
			if len(candles) == 0 {
				return
			}

			factorBars := make([]domain.FactorBar, 0, len(candles))
			for _, candle := range candles {
				factorBars = append(factorBars, domain.FactorBar{
					Timestamp:  candle.Timestamp,
					Factor:     sp.Alias,
					Open:       candle.Open,
					High:       candle.High,
					Low:        candle.Low,
					Close:      candle.Close,
					Volume:     candle.Volume,
					Timeframe:  sp.Timeframe,
					Source:     candle.Source,
					IngestedAt: candle.IngestedAt,
				})
			}
			if err := s.marketData.AppendFactors(factorCtx, sp.Alias, sp.Timeframe, factorBars); err != nil {
				res.failed = true
				return
			}
			res.refreshed = true
		}(spec)
	}

	for i := 0; i < len(s.factorSpecs); i++ {
		res := <-resChan
		if res.failed {
			result.FailedFactors++
			result.FailedFactorsIDs = append(result.FailedFactorsIDs, res.alias)
		} else if res.refreshed {
			result.FactorsRefreshed++
		}
	}
}

func (s *WatchlistRefreshService) resolveFactorInstrument(
	ctx context.Context,
	instruments InstrumentService,
	token string,
	spec FactorRefreshSpec,
) (domain.TinkoffInstrument, error) {
	if instruments == nil {
		return domain.TinkoffInstrument{}, ErrTinkoffUnavailable
	}
	items, err := instruments.FindInstrument(ctx, token, spec.Ticker)
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}
	for _, item := range items {
		if !strings.EqualFold(item.Ticker, spec.Ticker) {
			continue
		}
		if spec.ClassCode != "" && !strings.EqualFold(item.ClassCode, spec.ClassCode) {
			continue
		}
		if item.UID == "" {
			continue
		}
		return item, nil
	}
	return domain.TinkoffInstrument{}, fmt.Errorf("factor instrument not found: ticker=%s class_code=%s", spec.Ticker, spec.ClassCode)
}

// WatchlistSignalResult holds the results from a watchlist signal refresh job.
type WatchlistSignalResult struct {
	TotalInstruments int      `json:"total_instruments"`
	ModelSupported   int      `json:"model_supported"`
	SignalsGenerated int      `json:"signals_generated"`
	Skipped          int      `json:"skipped"`
	Failed           int      `json:"failed"`
	FailedAssetIDs   []string `json:"failed_asset_ids,omitempty"`
	SkippedAssetIDs  []string `json:"skipped_asset_ids,omitempty"`
}

// RunWatchlistSignalRefresh generates latest signals for model-supported watchlist instruments.
func (s *WatchlistRefreshService) RunWatchlistSignalRefresh(ctx context.Context) (domain.JobRun, error) {
	return s.RunWatchlistSignalRefreshForUser(ctx, 0)
}

func (s *WatchlistRefreshService) RunWatchlistSignalRefreshForUser(ctx context.Context, userID int64) (domain.JobRun, error) {
	if s.jobRepo == nil {
		return domain.JobRun{}, ErrJobsUnavailable
	}

	run, err := s.jobRepo.Create(ctx, domain.JobRun{
		JobType: JobTypeWatchlistSignalRefresh,
		Status:  domain.JobStatusRunning,
		Payload: map[string]any{},
	})
	if err != nil {
		return domain.JobRun{}, err
	}

	result, jobErr := s.doSignalRefresh(ctx, userID)

	payload := map[string]any{
		"total_instruments": result.TotalInstruments,
		"model_supported":   result.ModelSupported,
		"signals_generated": result.SignalsGenerated,
		"skipped":           result.Skipped,
		"failed":            result.Failed,
	}
	if len(result.FailedAssetIDs) > 0 {
		payload["failed_asset_ids"] = result.FailedAssetIDs
	}
	if len(result.SkippedAssetIDs) > 0 {
		payload["skipped_asset_ids"] = result.SkippedAssetIDs
	}

	status := domain.JobStatusSucceeded
	errMsg := ""
	if jobErr != nil {
		status = domain.JobStatusFailed
		errMsg = jobErr.Error()
	} else if result.Failed > 0 {
		status = domain.JobStatusFailed
		errMsg = fmt.Sprintf("%d items failed", result.Failed)
	}

	return s.jobRepo.Finish(ctx, run.ID, status, payload, errMsg)
}

func (s *WatchlistRefreshService) doSignalRefresh(ctx context.Context, userID int64) (WatchlistSignalResult, error) {
	var assetIDs []string

	if userID > 0 {
		watchlist, err := s.watchlistRepo.GetOrCreateByName(ctx, userID, "default")
		if err != nil {
			return WatchlistSignalResult{}, fmt.Errorf("watchlist init: %w", err)
		}
		items, err := s.watchlistRepo.ListItems(ctx, watchlist.ID)
		if err != nil {
			return WatchlistSignalResult{}, fmt.Errorf("watchlist list: %w", err)
		}
		for _, it := range items {
			assetIDs = append(assetIDs, it.AssetID)
		}
	} else {
		// System-wide signal refresh for all model-supported assets
		assets, err := s.assetRepo.List(ctx)
		if err != nil {
			return WatchlistSignalResult{}, fmt.Errorf("assets list: %w", err)
		}
		for _, a := range assets {
			if a.ModelSupported && a.IsActive {
				assetIDs = append(assetIDs, a.ID)
			}
		}
	}

	result := WatchlistSignalResult{TotalInstruments: len(assetIDs)}

	type signalResult struct {
		assetID         string
		modelSupported  bool
		signalGenerated bool
		failed          bool
		skipped         bool
	}
	resChan := make(chan signalResult, len(assetIDs))
	sem := make(chan struct{}, 3) // Signal generation is heavier, so lower concurrency

	for _, id := range assetIDs {
		go func(assetID string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			res := signalResult{assetID: assetID}
			defer func() { resChan <- res }()

			assetCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()

			asset, err := s.assetRepo.GetByID(assetCtx, assetID)
			if err != nil {
				res.skipped = true
				return
			}

			if !asset.ModelSupported {
				res.skipped = true
				return
			}

			res.modelSupported = true

			lookbackStart := time.Now().UTC().Add(-30 * 24 * time.Hour)
			candles, _ := s.marketData.ListCandles(assetCtx, asset.Ticker, asset.Timeframe, lookbackStart, time.Time{}, 1)
			if len(candles) == 0 {
				res.skipped = true
				return
			}

			if s.analysis != nil {
				_, err := s.analysis.Run(assetCtx, RunAnalysisInput{
					AssetID:      asset.ID,
					UserID:       userID,
					ModelVersion: "active",
					Timeframe:    asset.Timeframe,
				})
				if err != nil {
					res.failed = true
					if s.logger != nil {
						s.logger.Printf("watchlist signal refresh: asset %s analysis failed: %v", asset.ID, err)
					}
					return
				}
				res.signalGenerated = true
			}
		}(id)
	}

	for i := 0; i < len(assetIDs); i++ {
		res := <-resChan
		if res.modelSupported {
			result.ModelSupported++
		}
		if res.signalGenerated {
			result.SignalsGenerated++
		}
		if res.failed {
			result.Failed++
			result.FailedAssetIDs = append(result.FailedAssetIDs, res.assetID)
		}
		if res.skipped {
			result.Skipped++
			result.SkippedAssetIDs = append(result.SkippedAssetIDs, res.assetID)
		}
	}

	return result, nil
}
