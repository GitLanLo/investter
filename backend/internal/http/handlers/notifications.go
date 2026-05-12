package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"invest/backend/internal/app"
	"invest/backend/internal/domain"
)

func ListNotificationRules(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		if container.Services.Notifications == nil {
			WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "notification service is not configured", nil)
			return
		}

		rules, err := container.Services.Notifications.ListRules(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "rules_list_failed", err.Error(), nil)
			return
		}

		out := make([]NotificationRuleDTO, 0, len(rules))
		for _, rule := range rules {
			out = append(out, ToNotificationRuleDTO(rule))
		}
		WriteJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func CreateNotificationRule(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		if container.Services.Notifications == nil {
			WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "notification service is not configured", nil)
			return
		}

		var rule domain.NotificationRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}
		rule.UserID = userID

		saved, err := container.Services.Notifications.CreateRule(r.Context(), rule)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "rule_create_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusCreated, ToNotificationRuleDTO(saved))
	}
}

func UpdateNotificationRule(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		if container.Services.Notifications == nil {
			WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "notification service is not configured", nil)
			return
		}

		idRaw := r.PathValue("id")
		id, err := strconv.ParseInt(idRaw, 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_id", "id must be an integer", nil)
			return
		}

		var rule domain.NotificationRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}
		rule.ID = id
		rule.UserID = userID

		saved, err := container.Services.Notifications.UpdateRule(r.Context(), rule)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "rule_update_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, ToNotificationRuleDTO(saved))
	}
}

func DeleteNotificationRule(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		if container.Services.Notifications == nil {
			WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "notification service is not configured", nil)
			return
		}

		idRaw := r.PathValue("id")
		id, err := strconv.ParseInt(idRaw, 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_id", "id must be an integer", nil)
			return
		}

		if err := container.Services.Notifications.DeleteRule(r.Context(), id, userID); err != nil {
			WriteError(w, http.StatusInternalServerError, "rule_delete_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusNoContent, nil)
	}
}

func ListNotificationEvents(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		if container.Services.Notifications == nil {
			WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "notification service is not configured", nil)
			return
		}

		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}

		events, err := container.Services.Notifications.ListEvents(r.Context(), userID, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "events_list_failed", err.Error(), nil)
			return
		}

		out := make([]NotificationEventDTO, 0, len(events))
		for _, e := range events {
			out = append(out, ToNotificationEventDTO(e))
		}
		WriteJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}
