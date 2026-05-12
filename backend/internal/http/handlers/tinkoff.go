package handlers

import (
	"encoding/json"
	"net/http"

	"invest/backend/internal/app"
)

type tinkoffTokenRequest struct {
	Token     string `json:"token"`
	IsSandbox bool   `json:"is_sandbox"`
}

type tinkoffStatusResponse struct {
	Connected bool   `json:"connected"`
	TokenHint string `json:"token_hint,omitempty"`
	IsSandbox bool   `json:"is_sandbox"`
}

func GetTinkoffTokenStatus(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)

		hint, isSandbox, connected, err := container.Services.TinkoffCredentials.GetHint(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "tinkoff_status_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, tinkoffStatusResponse{
			Connected: connected,
			TokenHint: hint,
			IsSandbox: isSandbox,
		})
	}
}

func SaveTinkoffToken(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)

		var req tinkoffTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}

		if req.Token == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "token is required", nil)
			return
		}

		err := container.Services.TinkoffCredentials.SaveToken(r.Context(), userID, req.Token, req.IsSandbox)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "tinkoff_token_save_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func TestTinkoffToken(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req tinkoffTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}

		if req.Token == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "token is required", nil)
			return
		}

		err := container.Services.TinkoffCredentials.TestToken(r.Context(), req.Token, req.IsSandbox)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "tinkoff_token_invalid", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, map[string]any{"status": "valid"})
	}
}

func DeleteTinkoffToken(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)

		err := container.Services.TinkoffCredentials.DeleteToken(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "tinkoff_token_delete_failed", err.Error(), nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
