package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	ID       string
	Name     string
	Contents string
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database handle is nil")
	}

	if err := ensureMigrationTable(ctx, db); err != nil {
		return err
	}

	applied, err := loadAppliedMigrationIDs(ctx, db)
	if err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, item := range migrations {
		if _, ok := applied[item.ID]; ok {
			continue
		}
		if err := applyMigration(ctx, db, item); err != nil {
			return fmt.Errorf("apply migration %s: %w", item.ID, err)
		}
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func loadAppliedMigrationIDs(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = struct{}{}
	}

	return applied, rows.Err()
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		body, err := migrationFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return nil, err
		}

		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		name := id
		if idx := strings.IndexByte(id, '_'); idx >= 0 && idx < len(id)-1 {
			name = id[idx+1:]
		}

		migrations = append(migrations, migration{
			ID:       id,
			Name:     name,
			Contents: string(body),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})
	return migrations, nil
}

func applyMigration(ctx context.Context, db *sql.DB, item migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, item.Contents); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
		item.ID,
		item.Name,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
