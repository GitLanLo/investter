package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/config"
	"invest/backend/internal/httpserver"
	"invest/backend/internal/service"
	"invest/backend/internal/storage"
)

func main() {
	cfg := config.Load()
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.Open(cfg)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	if err := storage.Migrate(context.Background(), db); err != nil {
		log.Fatalf("db migrate failed: %v", err)
	}

	container := app.NewContainer(db, cfg.MLDataRoot, cfg.MLResearchRoot)
	if err := container.Services.Assets.SeedDefaults(context.Background()); err != nil {
		log.Fatalf("asset seed failed: %v", err)
	}
	if _, err := container.Services.Models.EnsureBootstrapActive(
		context.Background(),
		filepath.Join(cfg.MLModelRoot, "baseline_stub_v1", "model_manifest.json"),
	); err != nil {
		log.Fatalf("bootstrap model init failed: %v", err)
	}
	if cfg.OutcomeSchedulerEnabled {
		service.NewOutcomeMaterializationScheduler(
			container.Services.Jobs,
			cfg.OutcomeSchedulerInterval,
			cfg.OutcomeSchedulerLimit,
			cfg.OutcomeSchedulerRunOnStart,
			log.Default(),
		).Start(rootCtx)
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddress(),
		Handler: httpserver.NewRouter(cfg, httpserver.Dependencies{
			DB:        db,
			Container: container,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("backend listening on %s", cfg.HTTPAddress())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	<-rootCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}

	log.Println("backend stopped")
}
