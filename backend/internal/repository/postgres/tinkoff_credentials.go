package postgres

import (
	"context"
	"database/sql"
	"time"

	"invest/backend/internal/domain"
)

type TinkoffCredentialRepository struct {
	db *sql.DB
}

func NewTinkoffCredentialRepository(db *sql.DB) *TinkoffCredentialRepository {
	return &TinkoffCredentialRepository{db: db}
}

func (r *TinkoffCredentialRepository) Upsert(ctx context.Context, cred domain.UserTinkoffCredential) error {
	query := `
		INSERT INTO user_tinkoff_credentials (user_id, token_encrypted, token_nonce, token_hint, is_sandbox, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			token_encrypted = EXCLUDED.token_encrypted,
			token_nonce = EXCLUDED.token_nonce,
			token_hint = EXCLUDED.token_hint,
			is_sandbox = EXCLUDED.is_sandbox,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, cred.UserID, cred.TokenEncrypted, cred.TokenNonce, cred.TokenHint, cred.IsSandbox, now)
	return err
}

func (r *TinkoffCredentialRepository) GetByUserID(ctx context.Context, userID int64) (domain.UserTinkoffCredential, error) {
	query := `
		SELECT user_id, token_encrypted, token_nonce, token_hint, is_sandbox, created_at, updated_at
		FROM user_tinkoff_credentials
		WHERE user_id = $1
	`
	var cred domain.UserTinkoffCredential
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&cred.UserID, &cred.TokenEncrypted, &cred.TokenNonce, &cred.TokenHint, &cred.IsSandbox, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err != nil {
		return domain.UserTinkoffCredential{}, err
	}
	return cred, nil
}

func (r *TinkoffCredentialRepository) Delete(ctx context.Context, userID int64) error {
	query := `DELETE FROM user_tinkoff_credentials WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}
