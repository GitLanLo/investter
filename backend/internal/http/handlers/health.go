package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"invest/backend/internal/config"
	"invest/backend/internal/storage"
)

type healthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Env       string            `json:"env"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

func Health(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Service:   "invest-api",
			Env:       cfg.AppEnv,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func Ready(db *sql.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ready"
		checks := make(map[string]string)
		
		if err := storage.Check(r.Context(), db); err != nil {
			status = "unready"
			checks["postgres"] = err.Error()
		} else {
			checks["postgres"] = "ok"
		}

		code := http.StatusOK
		if status != "ready" {
			code = http.StatusServiceUnavailable
		}

		WriteJSON(w, code, healthResponse{
			Status:    status,
			Service:   "invest-api",
			Env:       cfg.AppEnv,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checks,
		})
	}
}
