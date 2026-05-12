package postgres

import (
	"context"
	"database/sql"
	"time"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, userID, token, expiresAt)
	return err
}

func (r *AuthRepository) GetRefreshToken(ctx context.Context, token string) (int64, time.Time, error) {
	query := `
		SELECT user_id, expires_at
		FROM refresh_tokens
		WHERE token = $1
	`
	var userID int64
	var expiresAt time.Time
	err := r.db.QueryRowContext(ctx, query, token).Scan(&userID, &expiresAt)
	return userID, expiresAt, err
}

func (r *AuthRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *AuthRepository) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}
