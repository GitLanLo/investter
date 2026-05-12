package storage

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMigrateAppliesPendingMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	// Expect migrations 001 to 016
	for i := 1; i <= 16; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO schema_migrations").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
	}

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ExpectationsWereMet: %v", err)
	}
}

func TestMigrateSkipsAppliedMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"version"})
	applied := []string{
		"001_init", "002_policy_validation_runs", "003_signal_policy_snapshot",
		"004_signal_outcomes", "005_instrument_catalog", "006_add_model_version_to_policy_validation_runs",
		"007_signal_events", "008_notifications", "009_policy_promotion_log",
		"010_users", "011_tinkoff_credentials", "012_user_scoped_data",
		"013_refresh_tokens", "014_fix_notification_rules_v2_user_id", "015_fix_signal_events_user_id",
		"016_user_access_control",
	}
	for _, id := range applied {
		rows.AddRow(id)
	}

	mock.ExpectQuery("SELECT version FROM schema_migrations").WillReturnRows(rows)

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ExpectationsWereMet: %v", err)
	}
}
