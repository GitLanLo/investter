package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

func GetProductionPolicy(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Research == nil {
			WriteError(w, http.StatusServiceUnavailable, "research_unavailable", "research service not configured", nil)
			return
		}
		policy, err := container.Services.Research.LoadProductionPolicy(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "production_policy_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToProductionPolicyDTO(policy))
	}
}

func ListPolicyValidationRuns(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}
		items, err := container.Services.Policy.ListLatest(r.Context(), limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "policy_list_failed", err.Error(), nil)
			return
		}
		out := make([]MLPolicyValidationRunDTO, 0, len(items))
		for _, item := range items {
			out = append(out, ToPolicyValidationRunDTO(item))
		}
		WriteJSON(w, http.StatusOK, MLPolicyValidationRunsResponse{Items: out})
	}
}

type policyValidationCreateRequest struct {
	Notes string `json:"notes"`
}

func CreatePolicyValidationRun(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		var req policyValidationCreateRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		run, err := container.Services.Policy.CreateFromCurrentPolicy(r.Context(), req.Notes)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "policy_create_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusCreated, ToPolicyValidationRunDTO(run))
	}
}

type policyValidationUpdateRequest struct {
	DecisionState string `json:"decision_state"`
	Notes         string `json:"notes"`
}

func UpdatePolicyValidationRun(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		var req policyValidationUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}
		run, err := container.Services.Policy.UpdateDecisionState(r.Context(), id, req.DecisionState, req.Notes)
		if err != nil {
			if errors.Is(err, service.ErrPolicyValidationRunNotFound) {
				WriteError(w, http.StatusNotFound, "policy_validation_run_not_found", err.Error(), nil)
				return
			}
			if errors.Is(err, service.ErrPolicyDecisionStateInvalid) {
				WriteError(w, http.StatusBadRequest, "policy_decision_state_invalid", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "policy_update_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToPolicyValidationRunDTO(run))
	}
}

func PromotePolicy(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		err := container.Services.Promotion.Promote(r.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrPromotionBlocked) {
				WriteError(w, http.StatusUnprocessableEntity, "promotion_blocked", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "promotion_failed", err.Error(), nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func RollbackPolicy(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		err := container.Services.Promotion.Rollback(r.Context(), id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "rollback_failed", err.Error(), nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func GetPolicyShadowSummary(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		summary, err := container.Services.Policy.LoadShadowSummary(r.Context(), 1000)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "shadow_summary_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToPolicyShadowSummaryDTO(summary))
	}
}

func GetPolicyOutcomes(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		summary, err := container.Services.Policy.LoadOutcomeSummary(r.Context(), 1000)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "outcome_summary_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToPolicyOutcomeSummaryDTO(summary))
	}
}

func MaterializePolicyOutcomes(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		limit := 1000
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}
		summary, err := container.Services.Policy.MaterializeOutcomes(r.Context(), limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "outcome_materialize_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToPolicyOutcomeSummaryDTO(summary))
	}
}

func GetPolicyOutcomeHistory(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Policy == nil {
			WriteError(w, http.StatusServiceUnavailable, "policy_unavailable", "policy service not configured", nil)
			return
		}
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil {
				limit = parsed
			}
		}
		records, err := container.Services.Policy.LoadOutcomeHistory(r.Context(), limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "outcome_history_failed", err.Error(), nil)
			return
		}
		out := MLPolicyOutcomeHistoryResponse{Items: make([]MLPolicyOutcomeRecordDTO, 0, len(records))}
		for _, record := range records {
			out.Items = append(out.Items, ToPolicyOutcomeRecordDTO(record))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}
