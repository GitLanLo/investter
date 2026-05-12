package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

func RunAnalysis(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AnalysisRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}
		if req.AssetID == "" {
			req.AssetID = r.PathValue("id")
		}
		if req.AssetID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "asset_id is required", nil)
			return
		}

		var asOfTime time.Time
		if req.AsOfTime != "" {
			parsed, err := time.Parse(time.RFC3339, req.AsOfTime)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "validation_error", "as_of_time must be RFC3339", nil)
				return
			}
			asOfTime = parsed
		}

		userID := r.Context().Value("user_id").(int64)

		run, err := container.Services.Analysis.Run(r.Context(), service.RunAnalysisInput{
			AssetID:      req.AssetID,
			UserID:       userID,
			AsOfTime:     asOfTime,
			ModelVersion: req.ModelVersion,
			Timeframe:    req.Timeframe,
		})
		if err != nil {
			switch err {
			case service.ErrAssetNotFound:
				WriteError(w, http.StatusNotFound, "asset_not_found", err.Error(), nil)
			case service.ErrModelNotFound:
				WriteError(w, http.StatusNotFound, "model_not_found", err.Error(), nil)
			default:
				WriteError(w, http.StatusBadRequest, "analysis_run_failed", err.Error(), nil)
			}
			return
		}

		WriteJSON(w, http.StatusCreated, ToSignalDTO(run))
	}
}

func ListLatestSignals(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}

		runs, err := container.Services.Analysis.ListLatest(r.Context(), userID, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "signal_list_failed", err.Error(), nil)
			return
		}

		out := SignalsResponse{Items: make([]SignalDTO, 0, len(runs))}
		for _, r := range runs {
			out.Items = append(out.Items, ToSignalDTO(r))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func ListAssetSignals(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		assetID := r.PathValue("id")
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}

		runs, err := container.Services.Analysis.ListByAsset(r.Context(), userID, assetID, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "signal_list_failed", err.Error(), nil)
			return
		}

		out := SignalsResponse{Items: make([]SignalDTO, 0, len(runs))}
		for _, r := range runs {
			out.Items = append(out.Items, ToSignalDTO(r))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func ListSignalEvents(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}

		events, err := container.Services.Analysis.ListEvents(r.Context(), userID, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "event_list_failed", err.Error(), nil)
			return
		}

		out := make([]SignalEventDTO, 0, len(events))
		for _, e := range events {
			out = append(out, ToSignalEventDTO(e))
		}
		WriteJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func ListAssetSignalEvents(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		assetID := r.URL.Query().Get("asset_id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "missing_asset_id", "asset_id query parameter is required", nil)
			return
		}
		limit := 10
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}

		events, err := container.Services.Analysis.ListEventsByAsset(r.Context(), userID, assetID, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "event_list_failed", err.Error(), nil)
			return
		}

		out := make([]SignalEventDTO, 0, len(events))
		for _, e := range events {
			out = append(out, ToSignalEventDTO(e))
		}
		WriteJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}
