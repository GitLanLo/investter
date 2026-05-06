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
	watchlistRepo repository.WatchlistRepository
	assetRepo     repository.AssetRepository
	marketData    repository.MarketDataRepository
	jobRepo       repository.JobRunRepository
	signalRepo    repository.SignalRunRepository
	analysis      *AnalysisService
	instruments   InstrumentService
	logger        *log.Logger
	factorSpecs   []FactorRefreshSpec
}

func NewWatchlistRefreshService(
	watchlistRepo repository.WatchlistRepository,
	assetRepo repository.AssetRepository,
	marketData repository.MarketDataRepository,
	jobRepo repository.JobRunRepository,
	signalRepo repository.SignalRunRepository,
	analysis *AnalysisService,
	instruments InstrumentService,
	logger *log.Logger,
) *WatchlistRefreshService {
	return &WatchlistRefreshService{
		watchlistRepo: watchlistRepo,
		assetRepo:     assetRepo,
		marketData:    marketData,
		jobRepo:       jobRepo,
		signalRepo:    signalRepo,
		analysis:      analysis,
		instruments:   instruments,
		logger:        logger,
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

// GetFreshness returns the freshness summary for all watchlist instruments.
func (s *WatchlistRefreshService) GetFreshness(ctx context.Context) (FreshnessSummary, error) {
	watchlist, err := s.watchlistRepo.GetOrCreateByName(ctx, "default")
	if err != nil {
		return FreshnessSummary{}, fmt.Errorf("watchlist init failed: %w", err)
	}

	wlItems, err := s.watchlistRepo.ListItems(ctx, watchlist.ID)
	if err != nil {
		return FreshnessSummary{}, fmt.Errorf("watchlist items failed: %w", err)
	}

	now := time.Now().UTC()
	summary := FreshnessSummary{
		GeneratedAt: now,
		TotalItems:  len(wlItems),
		Items:       make([]FreshnessItem, 0, len(wlItems)),
	}

	for _, wlItem := range wlItems {
		asset, err := s.assetRepo.GetByID(ctx, wlItem.AssetID)
		if err != nil {
			summary.Items = append(summary.Items, FreshnessItem{
				AssetID:     wlItem.AssetID,
				DataFresh:   false,
				SignalFresh: false,
				StaleReason: "asset not found",
			})
			summary.StaleData++
			summary.StaleSignals++
			continue
		}

		fi := FreshnessItem{
			AssetID:        asset.ID,
			Ticker:         asset.Ticker,
			Name:           asset.Name,
			ModelSupported: asset.ModelSupported,
		}

		// Check market status
		marketOpen := true
		if s.instruments != nil && asset.Exchange != "" {
			open, err := s.instruments.IsMarketOpen(ctx, asset.Exchange)
			if err == nil {
				marketOpen = open
			}
		}

		// Check candle data freshness
		candles, _ := s.marketData.ListCandles(ctx, asset.Ticker, asset.Timeframe, time.Time{}, time.Time{}, 1)
		if len(candles) > 0 {
			lastCandle := candles[0].Timestamp
			fi.LastCandleAt = &lastCandle
			if now.Sub(lastCandle) < dataFreshnessThreshold {
				fi.DataFresh = true
				summary.FreshData++
			} else {
				if marketOpen {
					fi.StaleReason = "candle data older than " + dataFreshnessThreshold.String()
					summary.StaleData++
				} else {
					fi.DataFresh = true // Consider fresh if market is closed
					fi.StaleReason = "market closed"
					summary.FreshData++
				}
			}
		} else {
			if marketOpen {
				fi.StaleReason = "no candle data available"
				summary.StaleData++
			} else {
				fi.DataFresh = true
				fi.StaleReason = "no data; market closed"
				summary.FreshData++
			}
		}

		// Check signal freshness
		if !asset.ModelSupported {
			fi.SignalFresh = false
			if fi.StaleReason == "" {
				fi.StaleReason = "watchlist-only (no ML signal)"
			} else {
				fi.StaleReason += "; watchlist-only (no ML signal)"
			}
			summary.WatchlistOnly++
			summary.StaleSignals++
		} else {
			signals, _ := s.signalRepo.ListByAsset(ctx, asset.ID, 1)
			if len(signals) > 0 {
				lastSignal := signals[0].CreatedAt
				fi.LastSignalAt = &lastSignal
				if now.Sub(lastSignal) < signalFreshnessThreshold {
					fi.SignalFresh = true
					summary.FreshSignals++
				} else {
					if fi.StaleReason != "" {
						fi.StaleReason += "; "
					}
					fi.StaleReason += "signal older than " + signalFreshnessThreshold.String()
					summary.StaleSignals++
				}
			} else {
				if fi.StaleReason != "" {
					fi.StaleReason += "; "
				}
				fi.StaleReason += "no signals generated"
				summary.StaleSignals++
			}
		}

		summary.Items = append(summary.Items, fi)
	}

	return summary, nil
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

// RunWatchlistRefresh refreshes market data for all watchlist instruments.
// This is a bridge/manual runner that simulates a data refresh by checking if
// the instrument has candle data. In a full implementation this would call
// the T-Bank data ingest adapter.
func (s *WatchlistRefreshService) RunWatchlistRefresh(ctx context.Context, limit int) (domain.JobRun, error) {
	if s.jobRepo == nil {
		return domain.JobRun{}, ErrJobsUnavailable
	}

	run, err := s.jobRepo.Create(ctx, domain.JobRun{
		JobType: JobTypeWatchlistRefresh,
		Status:  domain.JobStatusRunning,
		Payload: map[string]any{"limit": limit},
	})
	if err != nil {
		return domain.JobRun{}, err
	}

	result, jobErr := s.doWatchlistRefresh(ctx, limit)

	payload := map[string]any{
		"limit":             limit,
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

func (s *WatchlistRefreshService) doWatchlistRefresh(ctx context.Context, limit int) (WatchlistRefreshResult, error) {
	watchlist, err := s.watchlistRepo.GetOrCreateByName(ctx, "default")
	if err != nil {
		return WatchlistRefreshResult{}, fmt.Errorf("watchlist init: %w", err)
	}

	items, err := s.watchlistRepo.ListItems(ctx, watchlist.ID)
	if err != nil {
		return WatchlistRefreshResult{}, fmt.Errorf("watchlist list: %w", err)
	}

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	result := WatchlistRefreshResult{TotalInstruments: len(items)}
	factorTimeframes := make(map[string]struct{})

	for _, wlItem := range items {
		asset, err := s.assetRepo.GetByID(ctx, wlItem.AssetID)
		if err != nil {
			result.Failed++
			result.FailedAssetIDs = append(result.FailedAssetIDs, wlItem.AssetID)
			continue
		}

		if asset.Timeframe != "" {
			factorTimeframes[asset.Timeframe] = struct{}{}
		}

		// 1. Fetch and append new candles
		if s.instruments != nil && asset.Ticker != "" {
			inst, err := s.resolveAssetInstrument(ctx, asset)
			if err != nil {
				result.Failed++
				result.FailedAssetIDs = append(result.FailedAssetIDs, asset.ID)
				if s.logger != nil {
					s.logger.Printf("watchlist refresh: %s instrument resolve failed: %v", asset.Ticker, err)
				}
				continue
			}
			asset = mergeAssetInstrument(asset, inst)

			// Find last candle to determine start time
			lastCands, _ := s.marketData.ListCandles(ctx, asset.Ticker, asset.Timeframe, time.Time{}, time.Time{}, 1)
			from := time.Now().UTC().Add(-24 * time.Hour) // Default to last 24h if no data
			if len(lastCands) > 0 {
				from = lastCands[0].Timestamp.Add(time.Second)
			}

			// Don't request if from is too close to now
			if time.Since(from) > time.Minute {
				newCands, err := s.instruments.GetCandles(ctx, inst.UID, asset.Timeframe, from, time.Now().UTC())
				if err != nil {
					if s.logger != nil {
						s.logger.Printf("watchlist refresh: %s candles fetch failed: %v", asset.Ticker, err)
					}
					// We continue metadata update even if candles fail, but mark as failed for result
					result.Failed++
					result.FailedAssetIDs = append(result.FailedAssetIDs, asset.ID)
				} else if len(newCands) > 0 {
					for i := range newCands {
						newCands[i].Ticker = asset.Ticker
						newCands[i].Timeframe = asset.Timeframe
					}
					if err := s.marketData.AppendCandles(ctx, asset.Ticker, asset.Timeframe, newCands); err != nil {
						if s.logger != nil {
							s.logger.Printf("watchlist refresh: %s candles append failed: %v", asset.Ticker, err)
						}
						result.Failed++
						result.FailedAssetIDs = append(result.FailedAssetIDs, asset.ID)
					}
				}
			}

			// 2. Update asset metadata
			if err := s.assetRepo.Upsert(ctx, asset); err != nil {
				if s.logger != nil {
					s.logger.Printf("watchlist refresh: asset %s upsert failed: %v", asset.ID, err)
				}
			}
		}

		result.Refreshed++
		if s.logger != nil {
			s.logger.Printf("watchlist refresh: %s (%s) processed", asset.Ticker, asset.ID)
		}
	}

	s.refreshFactorStatus(ctx, &result, factorTimeframes)

	return result, nil
}

func (s *WatchlistRefreshService) refreshFactorStatus(
	ctx context.Context,
	result *WatchlistRefreshResult,
	timeframes map[string]struct{},
) {
	if result == nil || s.marketData == nil {
		return
	}

	now := time.Now().UTC()
	s.refreshConfiguredFactors(ctx, result, timeframes, now)

	for timeframe := range timeframes {
		factors, err := s.marketData.ListFactors(ctx, timeframe, now.Add(-24*time.Hour), now)
		if err != nil {
			if s.logger != nil {
				s.logger.Printf("watchlist refresh: factor freshness check failed for timeframe=%s: %v", timeframe, err)
			}
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

func (s *WatchlistRefreshService) resolveAssetInstrument(ctx context.Context, asset domain.Asset) (domain.TinkoffInstrument, error) {
	if asset.InstrumentUID != "" {
		return s.instruments.GetInstrumentByUID(ctx, asset.InstrumentUID)
	}
	items, err := s.instruments.FindInstrument(ctx, asset.Ticker)
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
) {
	if s.instruments == nil || len(s.factorSpecs) == 0 {
		return
	}

	for _, spec := range s.factorSpecs {
		if _, ok := timeframes[spec.Timeframe]; !ok && len(timeframes) > 0 {
			continue
		}

		from := now.Add(-24 * time.Hour)
		factors, err := s.marketData.ListFactors(ctx, spec.Timeframe, time.Time{}, time.Time{})
		if err != nil {
			result.FailedFactors++
			result.FailedFactorsIDs = append(result.FailedFactorsIDs, spec.Alias)
			continue
		}
		for _, factor := range factors {
			if factor.Factor == spec.Alias && factor.Timestamp.After(from) {
				from = factor.Timestamp.Add(time.Second)
			}
		}
		if now.Sub(from) <= time.Minute {
			continue
		}

		inst, err := s.resolveFactorInstrument(ctx, spec)
		if err != nil {
			result.FailedFactors++
			result.FailedFactorsIDs = append(result.FailedFactorsIDs, spec.Alias)
			if s.logger != nil {
				s.logger.Printf("watchlist refresh: factor %s instrument resolve failed: %v", spec.Alias, err)
			}
			continue
		}

		candles, err := s.instruments.GetCandles(ctx, inst.UID, spec.Timeframe, from, now)
		if err != nil {
			result.FailedFactors++
			result.FailedFactorsIDs = append(result.FailedFactorsIDs, spec.Alias)
			if s.logger != nil {
				s.logger.Printf("watchlist refresh: factor %s candles fetch failed: %v", spec.Alias, err)
			}
			continue
		}
		if len(candles) == 0 {
			continue
		}

		factorBars := make([]domain.FactorBar, 0, len(candles))
		for _, candle := range candles {
			factorBars = append(factorBars, domain.FactorBar{
				Timestamp:  candle.Timestamp,
				Factor:     spec.Alias,
				Open:       candle.Open,
				High:       candle.High,
				Low:        candle.Low,
				Close:      candle.Close,
				Volume:     candle.Volume,
				Timeframe:  spec.Timeframe,
				Source:     candle.Source,
				IngestedAt: candle.IngestedAt,
			})
		}
		if err := s.marketData.AppendFactors(ctx, spec.Alias, spec.Timeframe, factorBars); err != nil {
			result.FailedFactors++
			result.FailedFactorsIDs = append(result.FailedFactorsIDs, spec.Alias)
			if s.logger != nil {
				s.logger.Printf("watchlist refresh: factor %s append failed: %v", spec.Alias, err)
			}
			continue
		}
		result.FactorsRefreshed++
	}
}

func (s *WatchlistRefreshService) resolveFactorInstrument(ctx context.Context, spec FactorRefreshSpec) (domain.TinkoffInstrument, error) {
	items, err := s.instruments.FindInstrument(ctx, spec.Ticker)
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

	result, jobErr := s.doSignalRefresh(ctx)

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

func (s *WatchlistRefreshService) doSignalRefresh(ctx context.Context) (WatchlistSignalResult, error) {
	watchlist, err := s.watchlistRepo.GetOrCreateByName(ctx, "default")
	if err != nil {
		return WatchlistSignalResult{}, fmt.Errorf("watchlist init: %w", err)
	}

	items, err := s.watchlistRepo.ListItems(ctx, watchlist.ID)
	if err != nil {
		return WatchlistSignalResult{}, fmt.Errorf("watchlist list: %w", err)
	}

	result := WatchlistSignalResult{TotalInstruments: len(items)}

	for _, wlItem := range items {
		asset, err := s.assetRepo.GetByID(ctx, wlItem.AssetID)
		if err != nil {
			result.Skipped++
			result.SkippedAssetIDs = append(result.SkippedAssetIDs, wlItem.AssetID)
			continue
		}

		if !asset.ModelSupported {
			result.Skipped++
			result.SkippedAssetIDs = append(result.SkippedAssetIDs, asset.ID)
			if s.logger != nil {
				s.logger.Printf("watchlist signal: %s skipped (watchlist-only, model_supported=false)", asset.Ticker)
			}
			continue
		}

		result.ModelSupported++

		// Check data availability: need candle data to generate signals
		candles, _ := s.marketData.ListCandles(ctx, asset.Ticker, asset.Timeframe, time.Time{}, time.Time{}, 1)
		if len(candles) == 0 {
			result.Skipped++
			result.SkippedAssetIDs = append(result.SkippedAssetIDs, asset.ID)
			if s.logger != nil {
				s.logger.Printf("watchlist signal: %s skipped (no candle data)", asset.Ticker)
			}
			continue
		}

		// Run analysis to generate latest signal
		if s.analysis != nil {
			_, err := s.analysis.Run(ctx, RunAnalysisInput{
				AssetID:      asset.ID,
				ModelVersion: "active",
				Timeframe:    asset.Timeframe,
			})
			if err != nil {
				result.Failed++
				result.FailedAssetIDs = append(result.FailedAssetIDs, asset.ID)
				if s.logger != nil {
					s.logger.Printf("watchlist signal: %s failed: %v", asset.Ticker, err)
				}
				continue
			}
			result.SignalsGenerated++
			if s.logger != nil {
				s.logger.Printf("watchlist signal: %s signal generated", asset.Ticker)
			}
		}
	}

	return result, nil
}
