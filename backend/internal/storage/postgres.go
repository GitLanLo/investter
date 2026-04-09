package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"invest/backend/internal/config"
)

func Open(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

func Check(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database handle is nil")
	}

	return db.PingContext(ctx)
}
