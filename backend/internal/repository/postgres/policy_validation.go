package postgres

import (
	"context"
	"database/sql"

	"invest/backend/internal/domain"
)

type PolicyValidationRunRepository struct {
	db *sql.DB
}

func NewPolicyValidationRunRepository(db *sql.DB) *PolicyValidationRunRepository {
	return &PolicyValidationRunRepository{db: db}
}

func (r *PolicyValidationRunRepository) Create(ctx context.Context, run domain.PolicyValidationRun) (domain.PolicyValidationRun, error) {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO policy_validation_runs (
			policy_status,
			model_name,
			scenario_name,
			calibration_method,
			threshold,
			dataset_version,
			validation_actionable_f1,
			validation_precision,
			validation_coverage,
			validation_actionable_ece,
			test_actionable_f1,
			test_precision,
			test_coverage,
			test_actionable_ece,
			decision_state,
			notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at
	`,
		run.PolicyStatus,
		run.ModelName,
		run.ScenarioName,
		run.CalibrationMethod,
		run.Threshold,
		run.DatasetVersion,
		run.Validation.ActionableF1,
		run.Validation.Precision,
		run.Validation.Coverage,
		run.Validation.ActionableECE,
		run.Test.ActionableF1,
		run.Test.Precision,
		run.Test.Coverage,
		run.Test.ActionableECE,
		run.DecisionState,
		run.Notes,
	).Scan(&run.ID, &run.CreatedAt)
	if err != nil {
		return domain.PolicyValidationRun{}, err
	}
	return run, nil
}

func (r *PolicyValidationRunRepository) ListLatest(ctx context.Context, limit int) ([]domain.PolicyValidationRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			policy_status,
			model_name,
			scenario_name,
			calibration_method,
			threshold,
			dataset_version,
			validation_actionable_f1,
			validation_precision,
			validation_coverage,
			validation_actionable_ece,
			test_actionable_f1,
			test_precision,
			test_coverage,
			test_actionable_ece,
			decision_state,
			notes,
			created_at
		FROM policy_validation_runs
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.PolicyValidationRun{}
	for rows.Next() {
		var item domain.PolicyValidationRun
		if err := rows.Scan(
			&item.ID,
			&item.PolicyStatus,
			&item.ModelName,
			&item.ScenarioName,
			&item.CalibrationMethod,
			&item.Threshold,
			&item.DatasetVersion,
			&item.Validation.ActionableF1,
			&item.Validation.Precision,
			&item.Validation.Coverage,
			&item.Validation.ActionableECE,
			&item.Test.ActionableF1,
			&item.Test.Precision,
			&item.Test.Coverage,
			&item.Test.ActionableECE,
			&item.DecisionState,
			&item.Notes,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PolicyValidationRunRepository) UpdateDecisionState(
	ctx context.Context,
	id int64,
	decisionState string,
	notes string,
) (domain.PolicyValidationRun, error) {
	var item domain.PolicyValidationRun
	err := r.db.QueryRowContext(ctx, `
		UPDATE policy_validation_runs
		SET decision_state = $2,
		    notes = $3
		WHERE id = $1
		RETURNING
			id,
			policy_status,
			model_name,
			scenario_name,
			calibration_method,
			threshold,
			dataset_version,
			validation_actionable_f1,
			validation_precision,
			validation_coverage,
			validation_actionable_ece,
			test_actionable_f1,
			test_precision,
			test_coverage,
			test_actionable_ece,
			decision_state,
			notes,
			created_at
	`, id, decisionState, notes).Scan(
		&item.ID,
		&item.PolicyStatus,
		&item.ModelName,
		&item.ScenarioName,
		&item.CalibrationMethod,
		&item.Threshold,
		&item.DatasetVersion,
		&item.Validation.ActionableF1,
		&item.Validation.Precision,
		&item.Validation.Coverage,
		&item.Validation.ActionableECE,
		&item.Test.ActionableF1,
		&item.Test.Precision,
		&item.Test.Coverage,
		&item.Test.ActionableECE,
		&item.DecisionState,
		&item.Notes,
		&item.CreatedAt,
	)
	if err != nil {
		return domain.PolicyValidationRun{}, err
	}
	return item, nil
}
