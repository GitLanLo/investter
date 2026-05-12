package httpserver

import (
	"context"
	"database/sql"
	"net/http"

	"invest/backend/internal/app"
	"invest/backend/internal/config"
	"invest/backend/internal/domain"
	"invest/backend/internal/http/handlers"
	"invest/backend/internal/storage"
)

type Dependencies struct {
	DB        *sql.DB
	Container app.Container
}

func NewRouter(cfg config.Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	v1 := "/api/v1"

	auth := authMiddleware(deps.Container.Services.Auth, cfg.AppEnv)

	// Health & Ready (Public)
	mux.HandleFunc("GET /health", handlers.Health(cfg))
	mux.HandleFunc("GET /ready", handlers.Ready(deps.DB, cfg))
	mux.HandleFunc("GET "+v1+"/health", handlers.Health(cfg))
	mux.HandleFunc("GET "+v1+"/ready", handlers.Ready(deps.DB, cfg))

	// Auth (Public)
	mux.HandleFunc("POST "+v1+"/auth/register", handlers.Register(deps.Container))
	mux.HandleFunc("POST "+v1+"/auth/login", handlers.Login(deps.Container))
	mux.HandleFunc("POST "+v1+"/auth/refresh", handlers.Refresh(deps.Container))
	mux.HandleFunc("POST "+v1+"/auth/logout", handlers.Logout(deps.Container))

	// Protected Routes
	protected := http.NewServeMux()

	registerUserRoutes(protected, v1, deps.Container)
	registerAssetRoutes(protected, v1, deps.Container)
	registerInstrumentRoutes(protected, v1, deps.Container)
	registerWatchlistRoutes(protected, v1, deps.Container)
	registerAnalysisRoutes(protected, v1, deps.Container)
	registerNotificationRoutes(protected, v1, deps.Container)
	adminML := http.NewServeMux()
	registerAdminRoutes(adminML, v1, deps.Container)
	protected.Handle(v1+"/admin/ml/", requirePermission(domain.PermissionMLAdmin)(adminML))

	adminUsers := http.NewServeMux()
	registerAdminUserRoutes(adminUsers, v1, deps.Container)
	protected.Handle(v1+"/admin/users", requirePermission(domain.PermissionUserAdmin)(adminUsers))
	protected.Handle(v1+"/admin/users/", requirePermission(domain.PermissionUserAdmin)(adminUsers))
	registerJobRoutes(protected, v1, deps.Container, cfg)

	mux.Handle(v1+"/", auth(protected))

	// Legacy / Compatibility redirect or index
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		handlers.WriteJSON(w, http.StatusOK, map[string]any{
			"service":  "invest-backend",
			"version":  "v1",
			"api_root": v1,
			"links": map[string]string{
				"health": "/health",
				"ready":  "/ready",
			},
		})
	})

	return recoveryMiddleware(loggingMiddleware(corsMiddleware(mux)))
}

func registerUserRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/me", handlers.Me(c))
	mux.HandleFunc("GET "+v1+"/me/tinkoff-token/status", handlers.GetTinkoffTokenStatus(c))
	mux.HandleFunc("POST "+v1+"/me/tinkoff-token", handlers.SaveTinkoffToken(c))
	mux.HandleFunc("POST "+v1+"/me/tinkoff-token/test", handlers.TestTinkoffToken(c))
	mux.HandleFunc("DELETE "+v1+"/me/tinkoff-token", handlers.DeleteTinkoffToken(c))
}

func registerAssetRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/assets", handlers.ListAssets(c))
	mux.HandleFunc("GET "+v1+"/assets/{id}/candles", handlers.GetAssetCandles(c))
	mux.HandleFunc("GET "+v1+"/assets/{id}/factors", handlers.GetAssetFactors(c))
	mux.HandleFunc("GET "+v1+"/assets/{id}/indicators", handlers.GetAssetIndicators(c))
	mux.HandleFunc("GET "+v1+"/assets/{id}/summary", handlers.GetAssetSummary(c))
	mux.HandleFunc("POST "+v1+"/assets/{id}/refresh", handlers.RefreshAsset(c))
}

func registerInstrumentRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/instruments/search", handlers.SearchInstruments(c))
	mux.HandleFunc("GET "+v1+"/instruments/{uid}", handlers.GetInstrument(c))
	mux.HandleFunc("POST "+v1+"/instruments/{uid}/asset", handlers.EnsureInstrumentAsset(c))
}

func registerWatchlistRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/watchlist", handlers.GetWatchlist(c))
	mux.HandleFunc("POST "+v1+"/watchlist", handlers.UpsertWatchlistItem(c))
	mux.HandleFunc("DELETE "+v1+"/watchlist/{asset_id}", handlers.RemoveWatchlistItem(c))
	mux.HandleFunc("GET "+v1+"/watchlist/freshness", handlers.GetWatchlistFreshness(c))
}

func registerAnalysisRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("POST "+v1+"/assets/{id}/analysis/run", handlers.RunAnalysis(c))
	mux.HandleFunc("GET "+v1+"/assets/{id}/signals", handlers.ListAssetSignals(c))
	mux.HandleFunc("GET "+v1+"/signals/latest", handlers.ListLatestSignals(c))
	mux.HandleFunc("GET "+v1+"/signals/events", handlers.ListAssetSignalEvents(c))
}

func registerNotificationRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/alerts/rules", handlers.ListNotificationRules(c))
	mux.HandleFunc("POST "+v1+"/alerts/rules", handlers.CreateNotificationRule(c))
	mux.HandleFunc("PATCH "+v1+"/alerts/rules/{id}", handlers.UpdateNotificationRule(c))
	mux.HandleFunc("DELETE "+v1+"/alerts/rules/{id}", handlers.DeleteNotificationRule(c))
	mux.HandleFunc("GET "+v1+"/alerts/events", handlers.ListNotificationEvents(c))
	mux.HandleFunc("GET "+v1+"/alerts/signal-events", handlers.ListSignalEvents(c))
}

func registerAdminRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/admin/ml/models/active", handlers.GetActiveModel(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/models/{version}", handlers.GetModelByVersion(c))
	mux.HandleFunc("POST "+v1+"/admin/ml/models/{version}/activate", handlers.ActivateModel(c))

	mux.HandleFunc("GET "+v1+"/admin/ml/research/overview", handlers.GetResearchOverview(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/research/documents", handlers.GetResearchDocuments(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/policy/production", handlers.GetProductionPolicy(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/policy/validation-runs", handlers.ListPolicyValidationRuns(c))
	mux.HandleFunc("POST "+v1+"/admin/ml/policy/validation-runs", handlers.CreatePolicyValidationRun(c))
	mux.HandleFunc("PATCH "+v1+"/admin/ml/policy/validation-runs/{id}", handlers.UpdatePolicyValidationRun(c))
	mux.HandleFunc("POST "+v1+"/admin/ml/policy/validation-runs/{id}/promote", handlers.PromotePolicy(c))
	mux.HandleFunc("POST "+v1+"/admin/ml/policy/validation-runs/{id}/rollback", handlers.RollbackPolicy(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/policy/shadow-summary", handlers.GetPolicyShadowSummary(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/policy/outcomes", handlers.GetPolicyOutcomes(c))
	mux.HandleFunc("POST "+v1+"/admin/ml/policy/outcomes", handlers.MaterializePolicyOutcomes(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/policy/outcomes/history", handlers.GetPolicyOutcomeHistory(c))
	mux.HandleFunc("GET "+v1+"/admin/ml/monitoring/summary", handlers.GetMonitoringSummary(c))
}

func registerAdminUserRoutes(mux *http.ServeMux, v1 string, c app.Container) {
	mux.HandleFunc("GET "+v1+"/admin/users", handlers.ListUsers(c))
	mux.HandleFunc("PATCH "+v1+"/admin/users/{id}/access", handlers.UpdateUserAccess(c))
}

func registerJobRoutes(mux *http.ServeMux, v1 string, c app.Container, cfg config.Config) {
	mux.HandleFunc("GET "+v1+"/jobs/runs", handlers.ListJobRuns(c))
	mux.HandleFunc("GET "+v1+"/jobs/scheduler", handlers.GetJobSchedulerStatus(cfg))
	mux.HandleFunc("GET "+v1+"/jobs/schedulers", handlers.GetSchedulers(cfg))
	mux.HandleFunc("POST "+v1+"/jobs/data-refresh", handlers.RunDataRefreshJob(c))
	mux.HandleFunc("POST "+v1+"/jobs/signals/run", handlers.RunSignalRefreshJob(c))
	mux.HandleFunc("POST "+v1+"/jobs/outcomes/materialize", handlers.RunOutcomeMaterializeJob(c))
}

func (d Dependencies) Check(ctx context.Context) error {
	return storage.Check(ctx, d.DB)
}
