package handlers

import (
	"errors"
	"net/http"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

func GetActiveModel(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entry, err := container.Services.Models.GetActive(r.Context())
		if err != nil {
			WriteError(w, http.StatusNotFound, "active_model_not_found", err.Error(), nil)
			return
		}
		manifest, err := service.LoadManifest(entry.ManifestPath)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "manifest_load_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToModelManifestDTO(manifest))
	}
}

func GetModelByVersion(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := r.PathValue("version")
		entry, err := container.Services.Models.GetByVersion(r.Context(), version)
		if err != nil {
			WriteError(w, http.StatusNotFound, "model_not_found", err.Error(), nil)
			return
		}
		manifest, err := service.LoadManifest(entry.ManifestPath)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "manifest_load_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToModelManifestDTO(manifest))
	}
}

func ActivateModel(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := r.PathValue("version")
		err := container.Services.Models.Activate(r.Context(), version)
		if err != nil {
			if errors.Is(err, service.ErrModelRuntimeBlocked) {
				WriteError(w, http.StatusForbidden, "model_runtime_blocked", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "model_activation_failed", err.Error(), nil)
			return
		}

		entry, err := container.Services.Models.GetActive(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "active_model_lookup_failed", err.Error(), nil)
			return
		}
		manifest, err := service.LoadManifest(entry.ManifestPath)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "manifest_load_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToModelManifestDTO(manifest))
	}
}

func GetResearchOverview(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Research == nil {
			WriteError(w, http.StatusServiceUnavailable, "research_unavailable", "research service not configured", nil)
			return
		}
		overview, err := container.Services.Research.LoadOverview(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "research_overview_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, ToResearchOverviewDTO(overview))
	}
}

func GetResearchDocuments(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Research == nil {
			WriteError(w, http.StatusServiceUnavailable, "research_unavailable", "research service not configured", nil)
			return
		}
		docs, err := container.Services.Research.LoadDocuments(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "research_docs_failed", err.Error(), nil)
			return
		}
		out := MLResearchDocumentsResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Items:       make([]MLResearchDocumentDTO, 0, len(docs)),
		}
		for _, doc := range docs {
			out.Items = append(out.Items, ToResearchDocumentDTO(doc))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func GetMonitoringSummary(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.Monitoring == nil {
			WriteError(w, http.StatusServiceUnavailable, "monitoring_unavailable", "monitoring service not configured", nil)
			return
		}
		summary, err := container.Services.Monitoring.GetSummary(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "monitoring_summary_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, summary)
	}
}
