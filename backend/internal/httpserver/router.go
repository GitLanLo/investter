package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/config"
	"invest/backend/internal/domain"
	"invest/backend/internal/service"
	"invest/backend/internal/storage"
)

type healthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Env       string            `json:"env"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

type Dependencies struct {
	DB        *sql.DB
	Container app.Container
}

type errorPayload struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type assetsResponse struct {
	Items []assetDTO `json:"items"`
}

type assetDTO struct {
	ID        string `json:"id"`
	Ticker    string `json:"ticker"`
	Name      string `json:"name"`
	Exchange  string `json:"exchange"`
	Timeframe string `json:"timeframe"`
	IsActive  bool   `json:"is_active"`
}

type candlesResponse struct {
	AssetID   string      `json:"asset_id"`
	Timeframe string      `json:"timeframe"`
	Items     []candleDTO `json:"items"`
}

type candleDTO struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
}

type factorsResponse struct {
	AssetID string      `json:"asset_id"`
	Items   []factorDTO `json:"items"`
}

type factorDTO struct {
	Factor    string  `json:"factor"`
	Timestamp string  `json:"timestamp"`
	Close     float64 `json:"close"`
}

type watchlistResponse struct {
	WatchlistID int64              `json:"watchlist_id"`
	Name        string             `json:"name"`
	Items       []watchlistItemDTO `json:"items"`
}

type watchlistItemDTO struct {
	AssetID  string `json:"asset_id"`
	Position int    `json:"position"`
}

type watchlistUpsertRequest struct {
	AssetID  string `json:"asset_id"`
	Position int    `json:"position"`
}

type analysisRunRequest struct {
	AssetID      string `json:"asset_id"`
	AsOfTime     string `json:"as_of_time"`
	ModelVersion string `json:"model_version"`
	Timeframe    string `json:"timeframe"`
}

type signalsResponse struct {
	Items []signalDTO `json:"items"`
}

type signalDTO struct {
	AssetID            string                 `json:"asset_id"`
	AsOfTime           string                 `json:"as_of_time"`
	SignalState        string                 `json:"signal_state"`
	SignalDirection    string                 `json:"signal_direction"`
	SignalProbability  float64                `json:"signal_probability"`
	ClassProbabilities signalProbabilitiesDTO `json:"class_probabilities"`
	ModelVersion       string                 `json:"model_version"`
	Threshold          float64                `json:"threshold"`
	Timeframe          string                 `json:"timeframe"`
	HorizonBars        int                    `json:"horizon_bars"`
	Policy             *signalPolicyDTO       `json:"policy,omitempty"`
}

type signalPolicyDTO struct {
	PolicyStatus      string  `json:"policy_status"`
	ModelName         string  `json:"model_name"`
	ScenarioName      string  `json:"scenario_name"`
	CalibrationMethod string  `json:"calibration_method"`
	Threshold         float64 `json:"threshold"`
	DatasetVersion    string  `json:"dataset_version"`
}

type signalProbabilitiesDTO struct {
	Up      float64 `json:"up"`
	Down    float64 `json:"down"`
	NoTrade float64 `json:"no_trade"`
}

type mlResearchOverviewResponse struct {
	GeneratedAt        string            `json:"generated_at"`
	DatasetManifest    map[string]any    `json:"dataset_manifest,omitempty"`
	ResearchSummary    map[string]any    `json:"research_summary,omitempty"`
	CalibrationSummary map[string]any    `json:"calibration_summary,omitempty"`
	SourcePaths        map[string]string `json:"source_paths,omitempty"`
	Warnings           []string          `json:"warnings,omitempty"`
}

type mlResearchDocumentsResponse struct {
	GeneratedAt string                  `json:"generated_at"`
	Items       []mlResearchDocumentDTO `json:"items"`
}

type mlResearchDocumentDTO struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type mlProductionPolicyResponse struct {
	GeneratedAt       string                       `json:"generated_at"`
	PolicyStatus      string                       `json:"policy_status"`
	ModelName         string                       `json:"model_name"`
	ScenarioName      string                       `json:"scenario_name"`
	CalibrationMethod string                       `json:"calibration_method"`
	Threshold         float64                      `json:"threshold"`
	Timeframe         string                       `json:"timeframe"`
	HorizonBars       int                          `json:"horizon_bars"`
	DatasetVersion    string                       `json:"dataset_version"`
	FeatureSchema     string                       `json:"feature_schema"`
	TrainRows         int                          `json:"train_rows"`
	ValidationRows    int                          `json:"validation_rows"`
	TestRows          int                          `json:"test_rows"`
	Validation        mlProductionPolicyMetricsDTO `json:"validation"`
	Test              mlProductionPolicyMetricsDTO `json:"test"`
	SourcePaths       map[string]string            `json:"source_paths,omitempty"`
	Warnings          []string                     `json:"warnings,omitempty"`
}

type mlProductionPolicyMetricsDTO struct {
	ActionableF1  float64 `json:"actionable_f1"`
	Precision     float64 `json:"precision"`
	Coverage      float64 `json:"coverage"`
	ActionableECE float64 `json:"actionable_ece"`
}

type mlPolicyValidationRunsResponse struct {
	Items []mlPolicyValidationRunDTO `json:"items"`
}

type mlPolicyValidationRunCreateRequest struct {
	Notes string `json:"notes"`
}

type mlPolicyValidationRunUpdateRequest struct {
	DecisionState string `json:"decision_state"`
	Notes         string `json:"notes"`
}

type mlPolicyValidationRunDTO struct {
	ID                int64                        `json:"id"`
	PolicyStatus      string                       `json:"policy_status"`
	ModelName         string                       `json:"model_name"`
	ScenarioName      string                       `json:"scenario_name"`
	CalibrationMethod string                       `json:"calibration_method"`
	Threshold         float64                      `json:"threshold"`
	DatasetVersion    string                       `json:"dataset_version"`
	Validation        mlProductionPolicyMetricsDTO `json:"validation"`
	Test              mlProductionPolicyMetricsDTO `json:"test"`
	DecisionState     string                       `json:"decision_state"`
	Notes             string                       `json:"notes"`
	CreatedAt         string                       `json:"created_at"`
}

type mlPolicyShadowSummaryDTO struct {
	ValidationRunID   int64   `json:"validation_run_id"`
	DecisionState     string  `json:"decision_state"`
	ModelName         string  `json:"model_name"`
	CalibrationMethod string  `json:"calibration_method"`
	Threshold         float64 `json:"threshold"`
	DatasetVersion    string  `json:"dataset_version"`
	SignalsTotal      int     `json:"signals_total"`
	ActionableSignals int     `json:"actionable_signals"`
	NoTradeSignals    int     `json:"no_trade_signals"`
	UpSignals         int     `json:"up_signals"`
	DownSignals       int     `json:"down_signals"`
	ObservedCoverage  float64 `json:"observed_coverage"`
	FirstSignalAt     string  `json:"first_signal_at,omitempty"`
	LastSignalAt      string  `json:"last_signal_at,omitempty"`
}

func NewRouter(cfg config.Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Service:   "backend",
			Env:       cfg.AppEnv,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		if err := storage.Check(ctx, deps.DB); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, healthResponse{
				Status:    "unready",
				Service:   "backend",
				Env:       cfg.AppEnv,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Checks: map[string]string{
					"postgres": err.Error(),
				},
			})
			return
		}

		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ready",
			Service:   "backend",
			Env:       cfg.AppEnv,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks: map[string]string{
				"postgres": "ok",
			},
		})
	})

	mux.HandleFunc("/assets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		items, err := deps.Container.Services.Assets.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "assets_list_failed", err.Error(), nil)
			return
		}

		out := assetsResponse{Items: make([]assetDTO, 0, len(items))}
		for _, item := range items {
			out.Items = append(out.Items, assetDTO{
				ID:        item.ID,
				Ticker:    item.Ticker,
				Name:      item.Name,
				Exchange:  item.Exchange,
				Timeframe: item.Timeframe,
				IsActive:  item.IsActive,
			})
		}

		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/assets/{id}/candles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		from, err := parseOptionalRFC3339(r, "from")
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "from must be RFC3339", nil)
			return
		}
		to, err := parseOptionalRFC3339(r, "to")
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "to must be RFC3339", nil)
			return
		}
		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			writeError(w, http.StatusBadRequest, "validation_error", "to must be greater than or equal to from", nil)
			return
		}

		limit := 500
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", "limit must be an integer", nil)
				return
			}
			limit = parsed
		}

		asset, items, err := deps.Container.Services.MarketData.ListCandles(
			r.Context(),
			r.PathValue("id"),
			from,
			to,
			limit,
		)
		if err != nil {
			switch err {
			case service.ErrAssetNotFound:
				writeError(w, http.StatusNotFound, "asset_not_found", err.Error(), nil)
			default:
				writeError(w, http.StatusInternalServerError, "candles_list_failed", err.Error(), nil)
			}
			return
		}

		out := candlesResponse{
			AssetID:   asset.ID,
			Timeframe: asset.Timeframe,
			Items:     make([]candleDTO, 0, len(items)),
		}
		for _, item := range items {
			out.Items = append(out.Items, candleDTO{
				Timestamp: item.Timestamp.UTC().Format(time.RFC3339),
				Open:      item.Open,
				High:      item.High,
				Low:       item.Low,
				Close:     item.Close,
				Volume:    item.Volume,
			})
		}

		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/assets/{id}/signals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", "limit must be an integer", nil)
				return
			}
			limit = parsed
		}

		items, err := deps.Container.Services.Analysis.ListByAsset(
			r.Context(),
			r.PathValue("id"),
			limit,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "signals_list_failed", err.Error(), nil)
			return
		}

		policy := currentSignalPolicyDTO(r.Context(), deps)
		out := signalsResponse{Items: make([]signalDTO, 0, len(items))}
		for _, item := range items {
			out.Items = append(out.Items, toSignalDTO(item, policy))
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/assets/{id}/factors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		from, err := parseOptionalRFC3339(r, "from")
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "from must be RFC3339", nil)
			return
		}
		to, err := parseOptionalRFC3339(r, "to")
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "to must be RFC3339", nil)
			return
		}
		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			writeError(w, http.StatusBadRequest, "validation_error", "to must be greater than or equal to from", nil)
			return
		}

		asset, items, err := deps.Container.Services.MarketData.ListFactors(
			r.Context(),
			r.PathValue("id"),
			from,
			to,
		)
		if err != nil {
			switch err {
			case service.ErrAssetNotFound:
				writeError(w, http.StatusNotFound, "asset_not_found", err.Error(), nil)
			default:
				writeError(w, http.StatusInternalServerError, "factors_list_failed", err.Error(), nil)
			}
			return
		}

		out := factorsResponse{
			AssetID: asset.ID,
			Items:   make([]factorDTO, 0, len(items)),
		}
		for _, item := range items {
			out.Items = append(out.Items, factorDTO{
				Factor:    item.Factor,
				Timestamp: item.Timestamp.UTC().Format(time.RFC3339),
				Close:     item.Close,
			})
		}

		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/watchlist", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			watchlist, err := deps.Container.Services.Watchlist.EnsureDefault(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "watchlist_init_failed", err.Error(), nil)
				return
			}
			items, err := deps.Container.Services.Watchlist.ListItems(r.Context(), watchlist.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "watchlist_list_failed", err.Error(), nil)
				return
			}

			out := watchlistResponse{
				WatchlistID: watchlist.ID,
				Name:        watchlist.Name,
				Items:       make([]watchlistItemDTO, 0, len(items)),
			}
			for _, item := range items {
				out.Items = append(out.Items, watchlistItemDTO{
					AssetID:  item.AssetID,
					Position: item.Position,
				})
			}

			writeJSON(w, http.StatusOK, out)
		case http.MethodPost:
			var req watchlistUpsertRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
				return
			}
			if req.AssetID == "" {
				writeError(w, http.StatusBadRequest, "validation_error", "asset_id is required", nil)
				return
			}

			watchlist, err := deps.Container.Services.Watchlist.EnsureDefault(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "watchlist_init_failed", err.Error(), nil)
				return
			}

			if err := deps.Container.Services.Watchlist.AddItem(r.Context(), watchlist.ID, req.AssetID, req.Position); err != nil {
				writeError(w, http.StatusBadRequest, "watchlist_add_failed", err.Error(), nil)
				return
			}

			items, err := deps.Container.Services.Watchlist.ListItems(r.Context(), watchlist.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "watchlist_list_failed", err.Error(), nil)
				return
			}

			out := watchlistResponse{
				WatchlistID: watchlist.ID,
				Name:        watchlist.Name,
				Items:       make([]watchlistItemDTO, 0, len(items)),
			}
			for _, item := range items {
				out.Items = append(out.Items, watchlistItemDTO{
					AssetID:  item.AssetID,
					Position: item.Position,
				})
			}

			writeJSON(w, http.StatusCreated, out)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/analysis/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}

		var req analysisRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}
		if req.AssetID == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "asset_id is required", nil)
			return
		}

		var asOfTime time.Time
		if req.AsOfTime != "" {
			parsed, err := time.Parse(time.RFC3339, req.AsOfTime)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", "as_of_time must be RFC3339", nil)
				return
			}
			asOfTime = parsed
		}

		run, err := deps.Container.Services.Analysis.Run(r.Context(), service.RunAnalysisInput{
			AssetID:      req.AssetID,
			AsOfTime:     asOfTime,
			ModelVersion: req.ModelVersion,
			Timeframe:    req.Timeframe,
		})
		if err != nil {
			switch err {
			case service.ErrAssetNotFound:
				writeError(w, http.StatusNotFound, "asset_not_found", err.Error(), nil)
			case service.ErrModelNotFound:
				writeError(w, http.StatusNotFound, "model_not_found", err.Error(), nil)
			default:
				writeError(w, http.StatusBadRequest, "analysis_run_failed", err.Error(), nil)
			}
			return
		}

		writeJSON(w, http.StatusCreated, toSignalDTO(run, currentSignalPolicyDTO(r.Context(), deps)))
	})

	mux.HandleFunc("/signals/latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", "limit must be an integer", nil)
				return
			}
			limit = parsed
		}

		items, err := deps.Container.Services.Analysis.ListLatest(r.Context(), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "signals_list_failed", err.Error(), nil)
			return
		}

		policy := currentSignalPolicyDTO(r.Context(), deps)
		out := signalsResponse{Items: make([]signalDTO, 0, len(items))}
		for _, item := range items {
			out.Items = append(out.Items, toSignalDTO(item, policy))
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/ml/research/overview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if deps.Container.Services.Research == nil {
			writeError(w, http.StatusServiceUnavailable, "research_overview_unavailable", "research overview service is not configured", nil)
			return
		}

		overview, err := deps.Container.Services.Research.LoadOverview(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "research_overview_failed", err.Error(), nil)
			return
		}

		writeJSON(w, http.StatusOK, mlResearchOverviewResponse{
			GeneratedAt:        overview.GeneratedAt.UTC().Format(time.RFC3339),
			DatasetManifest:    overview.DatasetManifest,
			ResearchSummary:    overview.ResearchSummary,
			CalibrationSummary: overview.CalibrationSummary,
			SourcePaths:        overview.SourcePaths,
			Warnings:           overview.Warnings,
		})
	})

	mux.HandleFunc("/ml/research/documents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if deps.Container.Services.Research == nil {
			writeError(w, http.StatusServiceUnavailable, "research_documents_unavailable", "research documents service is not configured", nil)
			return
		}

		items, err := deps.Container.Services.Research.LoadDocuments(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "research_documents_failed", err.Error(), nil)
			return
		}

		out := mlResearchDocumentsResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Items:       make([]mlResearchDocumentDTO, 0, len(items)),
		}
		for _, item := range items {
			out.Items = append(out.Items, mlResearchDocumentDTO{
				Key:         item.Key,
				Title:       item.Title,
				Path:        item.Path,
				ContentType: item.ContentType,
				Content:     item.Content,
			})
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/ml/policy/production", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if deps.Container.Services.Research == nil {
			writeError(w, http.StatusServiceUnavailable, "production_policy_unavailable", "research policy service is not configured", nil)
			return
		}

		policy, err := deps.Container.Services.Research.LoadProductionPolicy(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "production_policy_failed", err.Error(), nil)
			return
		}

		writeJSON(w, http.StatusOK, toProductionPolicyDTO(policy))
	})

	mux.HandleFunc("/ml/policy/validation-runs", func(w http.ResponseWriter, r *http.Request) {
		if deps.Container.Services.Policy == nil {
			writeError(w, http.StatusServiceUnavailable, "policy_validation_unavailable", "policy validation service is not configured", nil)
			return
		}

		switch r.Method {
		case http.MethodGet:
			limit := 20
			if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
				parsed, err := strconv.Atoi(rawLimit)
				if err != nil {
					writeError(w, http.StatusBadRequest, "validation_error", "limit must be an integer", nil)
					return
				}
				limit = parsed
			}

			items, err := deps.Container.Services.Policy.ListLatest(r.Context(), limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "policy_validation_list_failed", err.Error(), nil)
				return
			}
			out := mlPolicyValidationRunsResponse{Items: make([]mlPolicyValidationRunDTO, 0, len(items))}
			for _, item := range items {
				out.Items = append(out.Items, toPolicyValidationRunDTO(item))
			}
			writeJSON(w, http.StatusOK, out)
		case http.MethodPost:
			var req mlPolicyValidationRunCreateRequest
			if r.Body != nil {
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
					return
				}
			}

			item, err := deps.Container.Services.Policy.CreateFromCurrentPolicy(r.Context(), req.Notes)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "policy_validation_create_failed", err.Error(), nil)
				return
			}
			writeJSON(w, http.StatusCreated, toPolicyValidationRunDTO(item))
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/ml/policy/validation-runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		if deps.Container.Services.Policy == nil {
			writeError(w, http.StatusServiceUnavailable, "policy_validation_unavailable", "policy validation service is not configured", nil)
			return
		}
		if r.Method != http.MethodPatch {
			writeMethodNotAllowed(w, http.MethodPatch)
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "validation_error", "id must be a positive integer", nil)
			return
		}

		var req mlPolicyValidationRunUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}

		item, err := deps.Container.Services.Policy.UpdateDecisionState(r.Context(), id, req.DecisionState, req.Notes)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrPolicyDecisionStateInvalid):
				writeError(w, http.StatusBadRequest, "validation_error", "decision_state is invalid", nil)
			case errors.Is(err, service.ErrPolicyValidationRunNotFound):
				writeError(w, http.StatusNotFound, "policy_validation_run_not_found", err.Error(), nil)
			default:
				writeError(w, http.StatusInternalServerError, "policy_validation_update_failed", err.Error(), nil)
			}
			return
		}
		writeJSON(w, http.StatusOK, toPolicyValidationRunDTO(item))
	})

	mux.HandleFunc("/ml/policy/shadow-summary", func(w http.ResponseWriter, r *http.Request) {
		if deps.Container.Services.Policy == nil {
			writeError(w, http.StatusServiceUnavailable, "policy_validation_unavailable", "policy validation service is not configured", nil)
			return
		}
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		limit := 1000
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", "limit must be an integer", nil)
				return
			}
			limit = parsed
		}

		summary, err := deps.Container.Services.Policy.LoadShadowSummary(r.Context(), limit)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrPolicyValidationRunNotFound):
				writeError(w, http.StatusNotFound, "policy_validation_run_not_found", err.Error(), nil)
			default:
				writeError(w, http.StatusInternalServerError, "policy_shadow_summary_failed", err.Error(), nil)
			}
			return
		}
		writeJSON(w, http.StatusOK, toPolicyShadowSummaryDTO(summary))
	})

	return loggingMiddleware(corsMiddleware(mux))
}

func parseOptionalRFC3339(r *http.Request, key string) (time.Time, error) {
	rawValue := r.URL.Query().Get(key)
	if rawValue == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, rawValue)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	details map[string]any,
) {
	writeJSON(w, status, errorPayload{
		Error: errorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", methods[0])
		if len(methods) > 1 {
			w.Header()["Allow"] = methods
		}
	}
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", nil)
}

func currentSignalPolicyDTO(ctx context.Context, deps Dependencies) *signalPolicyDTO {
	if deps.Container.Services.Research == nil {
		return nil
	}
	policy, err := deps.Container.Services.Research.LoadProductionPolicy(ctx)
	if err != nil || policy.PolicyStatus == "" {
		return nil
	}
	return &signalPolicyDTO{
		PolicyStatus:      policy.PolicyStatus,
		ModelName:         policy.ModelName,
		ScenarioName:      policy.ScenarioName,
		CalibrationMethod: policy.CalibrationMethod,
		Threshold:         policy.Threshold,
		DatasetVersion:    policy.DatasetVersion,
	}
}

func toSignalDTO(run domain.SignalRun, policy *signalPolicyDTO) signalDTO {
	if run.Policy != nil {
		policy = &signalPolicyDTO{
			PolicyStatus:      run.Policy.PolicyStatus,
			ModelName:         run.Policy.ModelName,
			ScenarioName:      run.Policy.ScenarioName,
			CalibrationMethod: run.Policy.CalibrationMethod,
			Threshold:         run.Policy.Threshold,
			DatasetVersion:    run.Policy.DatasetVersion,
		}
	}
	return signalDTO{
		AssetID:           run.AssetID,
		AsOfTime:          run.AsOfTime.UTC().Format(time.RFC3339),
		SignalState:       run.SignalState,
		SignalDirection:   run.SignalDirection,
		SignalProbability: run.SignalProbability,
		ClassProbabilities: signalProbabilitiesDTO{
			Up:      run.ClassProbabilities.Up,
			Down:    run.ClassProbabilities.Down,
			NoTrade: run.ClassProbabilities.NoTrade,
		},
		ModelVersion: run.ModelVersion,
		Threshold:    run.Threshold,
		Timeframe:    run.Timeframe,
		HorizonBars:  run.HorizonBars,
		Policy:       policy,
	}
}

func toProductionPolicyDTO(policy service.ProductionPolicySnapshot) mlProductionPolicyResponse {
	return mlProductionPolicyResponse{
		GeneratedAt:       policy.GeneratedAt.UTC().Format(time.RFC3339),
		PolicyStatus:      policy.PolicyStatus,
		ModelName:         policy.ModelName,
		ScenarioName:      policy.ScenarioName,
		CalibrationMethod: policy.CalibrationMethod,
		Threshold:         policy.Threshold,
		Timeframe:         policy.Timeframe,
		HorizonBars:       policy.HorizonBars,
		DatasetVersion:    policy.DatasetVersion,
		FeatureSchema:     policy.FeatureSchema,
		TrainRows:         policy.TrainRows,
		ValidationRows:    policy.ValidationRows,
		TestRows:          policy.TestRows,
		Validation:        toProductionPolicyMetricsDTO(policy.Validation),
		Test:              toProductionPolicyMetricsDTO(policy.Test),
		SourcePaths:       policy.SourcePaths,
		Warnings:          policy.Warnings,
	}
}

func toProductionPolicyMetricsDTO(metrics service.ProductionPolicyMetrics) mlProductionPolicyMetricsDTO {
	return mlProductionPolicyMetricsDTO{
		ActionableF1:  metrics.ActionableF1,
		Precision:     metrics.Precision,
		Coverage:      metrics.Coverage,
		ActionableECE: metrics.ActionableECE,
	}
}

func toPolicyValidationRunDTO(run domain.PolicyValidationRun) mlPolicyValidationRunDTO {
	return mlPolicyValidationRunDTO{
		ID:                run.ID,
		PolicyStatus:      run.PolicyStatus,
		ModelName:         run.ModelName,
		ScenarioName:      run.ScenarioName,
		CalibrationMethod: run.CalibrationMethod,
		Threshold:         run.Threshold,
		DatasetVersion:    run.DatasetVersion,
		Validation: mlProductionPolicyMetricsDTO{
			ActionableF1:  run.Validation.ActionableF1,
			Precision:     run.Validation.Precision,
			Coverage:      run.Validation.Coverage,
			ActionableECE: run.Validation.ActionableECE,
		},
		Test: mlProductionPolicyMetricsDTO{
			ActionableF1:  run.Test.ActionableF1,
			Precision:     run.Test.Precision,
			Coverage:      run.Test.Coverage,
			ActionableECE: run.Test.ActionableECE,
		},
		DecisionState: run.DecisionState,
		Notes:         run.Notes,
		CreatedAt:     run.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toPolicyShadowSummaryDTO(summary domain.PolicyShadowSummary) mlPolicyShadowSummaryDTO {
	out := mlPolicyShadowSummaryDTO{
		ValidationRunID:   summary.ValidationRunID,
		DecisionState:     summary.DecisionState,
		ModelName:         summary.ModelName,
		CalibrationMethod: summary.CalibrationMethod,
		Threshold:         summary.Threshold,
		DatasetVersion:    summary.DatasetVersion,
		SignalsTotal:      summary.SignalsTotal,
		ActionableSignals: summary.ActionableSignals,
		NoTradeSignals:    summary.NoTradeSignals,
		UpSignals:         summary.UpSignals,
		DownSignals:       summary.DownSignals,
		ObservedCoverage:  summary.ObservedCoverage,
	}
	if !summary.FirstSignalAt.IsZero() {
		out.FirstSignalAt = summary.FirstSignalAt.UTC().Format(time.RFC3339)
	}
	if !summary.LastSignalAt.IsZero() {
		out.LastSignalAt = summary.LastSignalAt.UTC().Format(time.RFC3339)
	}
	return out
}
