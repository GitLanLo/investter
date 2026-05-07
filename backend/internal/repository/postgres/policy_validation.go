package postgres

import (
	"encoding/json"
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
			model_version,
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at
	`,
		run.PolicyStatus,
		run.ModelName,
		run.ModelVersion,
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

func (r *PolicyValidationRunRepository) GetByID(ctx context.Context, id int64) (domain.PolicyValidationRun, error) {
	var item domain.PolicyValidationRun
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			policy_status,
			model_name,
			model_version,
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
		WHERE id = $1
	`, id).Scan(
		&item.ID,
		&item.PolicyStatus,
		&item.ModelName,
		&item.ModelVersion,
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

func (r *PolicyValidationRunRepository) ListLatest(ctx context.Context, limit int) ([]domain.PolicyValidationRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			policy_status,
			model_name,
			model_version,
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
			&item.ModelVersion,
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
			model_version,
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
		&item.ModelVersion,
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

func (r *PolicyValidationRunRepository) LogPromotion(ctx context.Context, log domain.PolicyPromotionLog) error {
	var blockersJSON []byte
	if len(log.Blockers) > 0 {
		blockersJSON, _ = json.Marshal(log.Blockers)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO policy_promotion_log (
			policy_validation_run_id,
			actor,
			previous_state,
			next_state,
			blockers,
			notes
		) VALUES ($1, $2, $3, $4, $5, $6)
	`,
		log.PolicyValidationRunID,
		log.Actor,
		log.PreviousState,
		log.NextState,
		blockersJSON,
		log.Notes,
	)
	return err
}

func (r *PolicyValidationRunRepository) DemoteCurrentAndPromote(ctx context.Context, targetRunID int64, demoteReason string, promotionLog domain.PolicyPromotionLog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Demote all current active/promoted runs
	if _, err := tx.ExecContext(ctx, `
		UPDATE policy_validation_runs
		SET decision_state = $1,
		    notes = notes || $2
		WHERE decision_state IN ($3, $4)
	`, domain.PolicyDecisionArchived, "\n"+demoteReason, domain.PolicyDecisionActive, domain.PolicyDecisionPromoted); err != nil {
		return err
	}

	// 2. Promote target run
	if _, err := tx.ExecContext(ctx, `
		UPDATE policy_validation_runs
		SET decision_state = $1,
		    notes = notes || $2
		WHERE id = $3
	`, domain.PolicyDecisionActive, "\npromoted to active", targetRunID); err != nil {
		return err
	}

	// 3. Log promotion
	var blockersJSON []byte
	if len(promotionLog.Blockers) > 0 {
		blockersJSON, _ = json.Marshal(promotionLog.Blockers)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO policy_promotion_log (
			policy_validation_run_id,
			actor,
			previous_state,
			next_state,
			blockers,
			notes
		) VALUES ($1, $2, $3, $4, $5, $6)
	`,
		promotionLog.PolicyValidationRunID,
		promotionLog.Actor,
		promotionLog.PreviousState,
		promotionLog.NextState,
		blockersJSON,
		promotionLog.Notes,
	); err != nil {
		return err
	}

	return tx.Commit()
}
