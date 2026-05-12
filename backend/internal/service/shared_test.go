package service

import (
	"context"
	"database/sql"
	"time"

	"invest/backend/internal/domain"
)

// --- Shared Test Doubles ---

type MockAssetRepo struct {
	Assets map[string]domain.Asset
}

func (m *MockAssetRepo) List(ctx context.Context) ([]domain.Asset, error) {
	out := make([]domain.Asset, 0, len(m.Assets))
	for _, a := range m.Assets {
		out = append(out, a)
	}
	return out, nil
}

func (m *MockAssetRepo) GetByID(ctx context.Context, id string) (domain.Asset, error) {
	a, ok := m.Assets[id]
	if !ok {
		return domain.Asset{}, sql.ErrNoRows
	}
	return a, nil
}

func (m *MockAssetRepo) Upsert(ctx context.Context, asset domain.Asset) error {
	if m.Assets == nil {
		m.Assets = make(map[string]domain.Asset)
	}
	m.Assets[asset.ID] = asset
	return nil
}

type MockMarketDataRepo struct {
	Candles             map[string][]domain.Candle
	Factors             []domain.FactorBar
	LastListTimeframe   string
	LastAppendTimeframe string
}

func (m *MockMarketDataRepo) ListCandles(ctx context.Context, ticker string, timeframe string, from time.Time, to time.Time, limit int) ([]domain.Candle, error) {
	m.LastListTimeframe = timeframe
	cands := m.Candles[ticker]
	filtered := make([]domain.Candle, 0)
	for _, c := range cands {
		if !from.IsZero() && c.Timestamp.Before(from) {
			continue
		}
		if !to.IsZero() && c.Timestamp.After(to) {
			continue
		}
		filtered = append(filtered, c)
	}
	// Sort by time (newest first as expected by service)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (m *MockMarketDataRepo) ListFactors(ctx context.Context, timeframe string, from time.Time, to time.Time) ([]domain.FactorBar, error) {
	filtered := make([]domain.FactorBar, 0)
	for _, factor := range m.Factors {
		if factor.Timeframe != "" && factor.Timeframe != timeframe {
			continue
		}
		if !from.IsZero() && factor.Timestamp.Before(from) {
			continue
		}
		if !to.IsZero() && factor.Timestamp.After(to) {
			continue
		}
		filtered = append(filtered, factor)
	}
	return filtered, nil
}

func (m *MockMarketDataRepo) AppendCandles(ctx context.Context, ticker string, timeframe string, candles []domain.Candle) error {
	if m.Candles == nil {
		m.Candles = make(map[string][]domain.Candle)
	}
	m.LastAppendTimeframe = timeframe
	m.Candles[ticker] = append(m.Candles[ticker], candles...)
	return nil
}

func (m *MockMarketDataRepo) AppendFactors(ctx context.Context, alias string, timeframe string, factors []domain.FactorBar) error {
	m.Factors = append(m.Factors, factors...)
	return nil
}

type MockWatchlistRepo struct {
	Items []domain.WatchlistItem
}

func (m *MockWatchlistRepo) GetOrCreateByName(ctx context.Context, userID int64, name string) (domain.Watchlist, error) {
	return domain.Watchlist{ID: 1, Name: name, UserID: userID}, nil
}

func (m *MockWatchlistRepo) ListItems(ctx context.Context, watchlistID int64) ([]domain.WatchlistItem, error) {
	return m.Items, nil
}

func (m *MockWatchlistRepo) AddItem(ctx context.Context, watchlistID int64, assetID string, position int) error {
	return nil
}

func (m *MockWatchlistRepo) RemoveItem(ctx context.Context, watchlistID int64, assetID string) error {
	return nil
}

type MockJobRunRepo struct {
	Runs []domain.JobRun
	Seq  int64
}

func (m *MockJobRunRepo) Create(ctx context.Context, run domain.JobRun) (domain.JobRun, error) {
	m.Seq++
	run.ID = m.Seq
	run.StartedAt = time.Now().UTC()
	m.Runs = append(m.Runs, run)
	return run, nil
}

func (m *MockJobRunRepo) Finish(ctx context.Context, id int64, status string, payload map[string]any, errorMessage string) (domain.JobRun, error) {
	for i, r := range m.Runs {
		if r.ID == id {
			m.Runs[i].Status = status
			m.Runs[i].Payload = payload
			m.Runs[i].ErrorMessage = errorMessage
			m.Runs[i].FinishedAt = time.Now().UTC()
			return m.Runs[i], nil
		}
	}
	return domain.JobRun{}, ErrJobsUnavailable
}

func (m *MockJobRunRepo) ListLatest(ctx context.Context, limit int) ([]domain.JobRun, error) {
	if limit > len(m.Runs) {
		limit = len(m.Runs)
	}
	return m.Runs[:limit], nil
}

type MockSignalRepo struct {
	Signals map[string][]domain.SignalRun
	Latest  []domain.SignalRun
	Seq     int64
}

func (m *MockSignalRepo) Create(ctx context.Context, run domain.SignalRun) (domain.SignalRun, error) {
	m.Seq++
	run.ID = m.Seq
	m.Latest = append([]domain.SignalRun{run}, m.Latest...)
	return run, nil
}

func (m *MockSignalRepo) ListLatest(ctx context.Context, userID int64, limit int) ([]domain.SignalRun, error) {
	if limit <= 0 || limit > len(m.Latest) {
		limit = len(m.Latest)
	}
	return m.Latest[:limit], nil
}

func (m *MockSignalRepo) ListByAsset(ctx context.Context, userID int64, assetID string, limit int) ([]domain.SignalRun, error) {
	return m.Signals[assetID], nil
}

func (m *MockSignalRepo) ListByPolicySnapshot(ctx context.Context, userID int64, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalRun, error) {
	return nil, nil
}

type MockModelRepo struct {
	Active domain.ModelRegistryEntry
	Items  map[string]domain.ModelRegistryEntry
}

func (r *MockModelRepo) List(context.Context) ([]domain.ModelRegistryEntry, error) {
	out := make([]domain.ModelRegistryEntry, 0, len(r.Items))
	for _, item := range r.Items {
		out = append(out, item)
	}
	return out, nil
}

func (r *MockModelRepo) GetActive(context.Context) (domain.ModelRegistryEntry, error) {
	if r.Active.ModelVersion == "" {
		return domain.ModelRegistryEntry{}, sql.ErrNoRows
	}
	return r.Active, nil
}

func (r *MockModelRepo) GetByVersion(_ context.Context, version string) (domain.ModelRegistryEntry, error) {
	if version == r.Active.ModelVersion {
		return r.Active, nil
	}
	entry, ok := r.Items[version]
	if !ok {
		return domain.ModelRegistryEntry{}, sql.ErrNoRows
	}
	return entry, nil
}

func (r *MockModelRepo) Register(context.Context, domain.ModelRegistryEntry) error {
	return nil
}

func (r *MockModelRepo) Activate(_ context.Context, version string) error {
	if version == r.Active.ModelVersion {
		return nil
	}
	entry, ok := r.Items[version]
	if !ok {
		return sql.ErrNoRows
	}
	r.Active = entry
	r.Active.Status = domain.ModelStatusActive
	return nil
}

type MockSignalEventRepo struct {
	Events []domain.SignalEvent
}

func (m *MockSignalEventRepo) Create(ctx context.Context, event domain.SignalEvent) (domain.SignalEvent, error) {
	event.ID = int64(len(m.Events) + 1)
	m.Events = append(m.Events, event)
	return event, nil
}

func (m *MockSignalEventRepo) Upsert(ctx context.Context, event domain.SignalEvent) (domain.SignalEvent, error) {
	if event.IdempotencyKey != "" {
		for i := range m.Events {
			if m.Events[i].IdempotencyKey == event.IdempotencyKey {
				event.ID = m.Events[i].ID
				event.CreatedAt = m.Events[i].CreatedAt
				m.Events[i] = event
				return event, nil
			}
		}
	}
	event.ID = int64(len(m.Events) + 1)
	m.Events = append(m.Events, event)
	return event, nil
}

func (m *MockSignalEventRepo) ListLatest(ctx context.Context, userID int64, limit int) ([]domain.SignalEvent, error) {
	return m.Events, nil
}

func (m *MockSignalEventRepo) ListByAsset(ctx context.Context, userID int64, assetID string, limit int) ([]domain.SignalEvent, error) {
	var filtered []domain.SignalEvent
	for _, e := range m.Events {
		if e.Ticker == assetID {
			filtered = append(filtered, e)
		}
		if len(filtered) == limit {
			break
		}
	}
	return filtered, nil
}

type MockNotificationRepo struct {
	Rules  []domain.NotificationRule
	Events []domain.NotificationEvent
}

func (r *MockNotificationRepo) ListRules(ctx context.Context, userID int64) ([]domain.NotificationRule, error) {
	return r.Rules, nil
}
func (r *MockNotificationRepo) ListActiveRules(ctx context.Context) ([]domain.NotificationRule, error) {
	return r.Rules, nil
}
func (r *MockNotificationRepo) GetRuleByID(ctx context.Context, id int64, userID int64) (domain.NotificationRule, error) {
	return domain.NotificationRule{}, nil
}
func (r *MockNotificationRepo) CreateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	rule.ID = int64(len(r.Rules) + 1)
	r.Rules = append(r.Rules, rule)
	return rule, nil
}
func (r *MockNotificationRepo) UpdateRule(ctx context.Context, rule domain.NotificationRule) (domain.NotificationRule, error) {
	return rule, nil
}
func (r *MockNotificationRepo) DeleteRule(ctx context.Context, id int64, userID int64) error {
	return nil
}
func (r *MockNotificationRepo) CreateEvent(ctx context.Context, event domain.NotificationEvent) (domain.NotificationEvent, error) {
	if event.SignalEventID != nil {
		for _, existing := range r.Events {
			if existing.RuleID == event.RuleID && existing.SignalEventID != nil && *existing.SignalEventID == *event.SignalEventID {
				return existing, nil
			}
		}
	}
	event.ID = int64(len(r.Events) + 1)
	r.Events = append(r.Events, event)
	return event, nil
}
func (r *MockNotificationRepo) ListLatestEvents(ctx context.Context, userID int64, limit int) ([]domain.NotificationEvent, error) {
	return r.Events, nil
}
func (r *MockNotificationRepo) GetLatestEventForRule(ctx context.Context, ruleID int64) (domain.NotificationEvent, error) {
	return domain.NotificationEvent{}, nil
}

type MockPolicyRepo struct {
	items map[int64]domain.PolicyValidationRun
}

func (r *MockPolicyRepo) Create(context.Context, domain.PolicyValidationRun) (domain.PolicyValidationRun, error) {
	return domain.PolicyValidationRun{}, nil
}

func (r *MockPolicyRepo) ListLatest(_ context.Context, _ int) ([]domain.PolicyValidationRun, error) {
	out := make([]domain.PolicyValidationRun, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

func (r *MockPolicyRepo) UpdateDecisionState(_ context.Context, id int64, state string, _ string) (domain.PolicyValidationRun, error) {
	item, ok := r.items[id]
	if !ok {
		return domain.PolicyValidationRun{}, sql.ErrNoRows
	}
	item.DecisionState = state
	r.items[id] = item
	return item, nil
}

func (r *MockPolicyRepo) GetByID(ctx context.Context, id int64) (domain.PolicyValidationRun, error) {
	item, ok := r.items[id]
	if !ok {
		return domain.PolicyValidationRun{}, sql.ErrNoRows
	}
	return item, nil
}

func (r *MockPolicyRepo) DemoteCurrentAndPromote(ctx context.Context, targetRunID int64, demoteReason string, log domain.PolicyPromotionLog) error {
	for id, item := range r.items {
		if item.DecisionState == domain.PolicyDecisionActive || item.DecisionState == domain.PolicyDecisionPromoted {
			item.DecisionState = domain.PolicyDecisionArchived
			r.items[id] = item
		}
	}
	item := r.items[targetRunID]
	item.DecisionState = domain.PolicyDecisionActive
	r.items[targetRunID] = item
	return nil
}

func (r *MockPolicyRepo) LogPromotion(ctx context.Context, log domain.PolicyPromotionLog) error {
	return nil
}

type MockSignalOutcomeRepo struct {
	Outcomes []domain.SignalOutcome
}

func (m *MockSignalOutcomeRepo) Upsert(ctx context.Context, outcome domain.SignalOutcome) (domain.SignalOutcome, error) {
	m.Outcomes = append(m.Outcomes, outcome)
	return outcome, nil
}

func (m *MockSignalOutcomeRepo) ListByPolicySnapshot(ctx context.Context, userID int64, modelName string, calibrationMethod string, datasetVersion string, limit int) ([]domain.SignalOutcome, error) {
	return m.Outcomes, nil
}
