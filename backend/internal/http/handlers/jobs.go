package handlers

import (
	"net/http"
	"strconv"

	"invest/backend/internal/app"
	"invest/backend/internal/config"
)

func ListJobRuns(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Jobs == nil {
			WriteError(w, http.StatusServiceUnavailable, "jobs_unavailable", "jobs service is not configured", nil)
			return
		}
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}
		items, err := container.Services.Jobs.ListLatest(r.Context(), limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "jobs_list_failed", err.Error(), nil)
			return
		}
		out := make([]JobRunDTO, 0, len(items))
		for _, item := range items {
			out = append(out, ToJobRunDTO(item))
		}
		WriteJSON(w, http.StatusOK, JobRunsResponse{Items: out})
	}
}

func RunDataRefreshJob(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.WatchlistRefresh == nil {
			WriteError(w, http.StatusServiceUnavailable, "watchlist_refresh_unavailable", "watchlist refresh service is not configured", nil)
			return
		}
		limit := 50
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}
		userID := r.Context().Value("user_id").(int64)
		item, err := container.Services.WatchlistRefresh.RunWatchlistRefreshForUser(r.Context(), userID, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "watchlist_refresh_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToJobRunDTO(item))
	}
}

func RunSignalRefreshJob(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.WatchlistRefresh == nil {
			WriteError(w, http.StatusServiceUnavailable, "watchlist_signal_refresh_unavailable", "watchlist signal refresh service is not configured", nil)
			return
		}
		userID := r.Context().Value("user_id").(int64)
		item, err := container.Services.WatchlistRefresh.RunWatchlistSignalRefreshForUser(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "watchlist_signal_refresh_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToJobRunDTO(item))
	}
}

func RunOutcomeMaterializeJob(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Jobs == nil {
			WriteError(w, http.StatusServiceUnavailable, "jobs_unavailable", "jobs service is not configured", nil)
			return
		}
		limit := 1000
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}
		item, err := container.Services.Jobs.RunOutcomeMaterialization(r.Context(), limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "outcome_job_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToJobRunDTO(item))
	}
}

func GetJobSchedulerStatus(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, JobSchedulerStatusDTO{
			Enabled:    cfg.OutcomeSchedulerEnabled,
			Interval:   cfg.OutcomeSchedulerInterval.String(),
			Limit:      cfg.OutcomeSchedulerLimit,
			RunOnStart: cfg.OutcomeSchedulerRunOnStart,
		})
	}
}

func GetSchedulers(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		watchlistRefreshEnabled := cfg.WatchlistRefreshSchedulerEnabled && cfg.TinkoffInvestToken != ""
		WriteJSON(w, http.StatusOK, SchedulersResponse{
			Schedulers: []SchedulerInfoDTO{
				{
					Name:       "outcomes_materialize",
					Enabled:    cfg.OutcomeSchedulerEnabled,
					Interval:   cfg.OutcomeSchedulerInterval.String(),
					Limit:      cfg.OutcomeSchedulerLimit,
					RunOnStart: cfg.OutcomeSchedulerRunOnStart,
				},
				{
					Name:     "watchlist_data_refresh",
					Enabled:  watchlistRefreshEnabled,
					Interval: cfg.WatchlistRefreshInterval.String(),
					Limit:    cfg.WatchlistRefreshLimit,
				},
				{
					Name:     "watchlist_signal_refresh",
					Enabled:  cfg.WatchlistSignalSchedulerEnabled,
					Interval: cfg.WatchlistSignalInterval.String(),
				},
			},
		})
	}
}
