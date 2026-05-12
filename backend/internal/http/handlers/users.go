package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

type updateUserAccessRequest struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func ListUsers(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := container.Services.Auth.ListUsers(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "users_list_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"items": users})
	}
}

func UpdateUserAccess(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			WriteError(w, http.StatusBadRequest, "invalid_user_id", "user id must be a positive integer", nil)
			return
		}
		var req updateUserAccessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}
		actorID, _ := r.Context().Value("user_id").(int64)
		user, err := container.Services.Auth.UpdateUserAccess(r.Context(), actorID, id, req.Role, req.Permissions)
		if err != nil {
			if errors.Is(err, service.ErrForbidden) {
				WriteError(w, http.StatusForbidden, "forbidden", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "user_access_update_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, user)
	}
}
