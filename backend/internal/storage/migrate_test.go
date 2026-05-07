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
	mock.ExpectBegin()
	mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS assets.*CREATE INDEX IF NOT EXISTS idx_job_runs_job_type_started_at").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("001_init", "init").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS policy_validation_runs.*CREATE INDEX IF NOT EXISTS idx_policy_validation_runs_decision_state").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("002_policy_validation_runs", "policy_validation_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)ALTER TABLE signal_runs.*CREATE INDEX IF NOT EXISTS idx_signal_runs_policy_status").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("003_signal_policy_snapshot", "signal_policy_snapshot").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS signal_outcomes.*CREATE INDEX IF NOT EXISTS idx_signal_outcomes_matured_at").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("004_signal_outcomes", "signal_outcomes").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)ALTER TABLE assets ADD COLUMN IF NOT EXISTS figi.*CREATE INDEX IF NOT EXISTS idx_assets_figi").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("005_instrument_catalog", "instrument_catalog").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)ALTER TABLE policy_validation_runs ADD COLUMN model_version").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("006_add_model_version_to_policy_validation_runs", "add_model_version_to_policy_validation_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)ALTER TABLE signal_events ADD COLUMN IF NOT EXISTS model_version.*CREATE INDEX IF NOT EXISTS idx_signal_events_model_version").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("007_signal_events", "signal_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS notification_rules_v2.*CREATE INDEX IF NOT EXISTS idx_notification_rules_v2_event_type").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("008_notifications", "notifications").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS policy_promotion_log.*CREATE INDEX IF NOT EXISTS idx_policy_promotion_log_created_at").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs("009_policy_promotion_log", "policy_promotion_log").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

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
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow("001_init").
			AddRow("002_policy_validation_runs").
			AddRow("003_signal_policy_snapshot").
			AddRow("004_signal_outcomes").
			AddRow("005_instrument_catalog").
			AddRow("006_add_model_version_to_policy_validation_runs").
			AddRow("007_signal_events").
			AddRow("008_notifications").
			AddRow("009_policy_promotion_log"))

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ExpectationsWereMet: %v", err)
	}
}
