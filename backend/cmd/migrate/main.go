package main

import (
	"context"
	"log"
	"time"

	"invest/backend/internal/config"
	"invest/backend/internal/storage"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	db, err := storage.Open(cfg)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := storage.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}

	log.Println("migrations applied")
}
