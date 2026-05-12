package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"invest/backend/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	query := `
		INSERT INTO users (email, password_hash, role, permissions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, role, permissions
	`
	now := time.Now().UTC()
	if user.Role == "" {
		user.Role = domain.UserRoleUser
	}
	if len(user.Permissions) == 0 {
		user.Permissions = []string{domain.PermissionMarketAccess}
	}
	perms, err := json.Marshal(user.Permissions)
	if err != nil {
		return domain.User{}, err
	}
	var rawPermissions []byte
	err = r.db.QueryRowContext(ctx, query, user.Email, user.PasswordHash, user.Role, perms, now, now).Scan(&user.ID, &user.Role, &rawPermissions)
	if err != nil {
		return domain.User{}, err
	}
	user.Permissions = decodePermissions(rawPermissions)
	user.CreatedAt = now
	user.UpdatedAt = now
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	query := `
		SELECT id, email, password_hash, COALESCE(role, 'user'), COALESCE(permissions, '["market_access"]'::jsonb), created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user domain.User
	var rawPermissions []byte
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &rawPermissions, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	user.Permissions = decodePermissions(rawPermissions)
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	query := `
		SELECT id, email, password_hash, COALESCE(role, 'user'), COALESCE(permissions, '["market_access"]'::jsonb), created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user domain.User
	var rawPermissions []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &rawPermissions, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	user.Permissions = decodePermissions(rawPermissions)
	return user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, email, password_hash, COALESCE(role, 'user'), COALESCE(permissions, '["market_access"]'::jsonb), created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var user domain.User
		var rawPermissions []byte
		if err := rows.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &rawPermissions, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		user.Permissions = decodePermissions(rawPermissions)
		out = append(out, user)
	}
	return out, rows.Err()
}

func (r *UserRepository) UpdateAccess(ctx context.Context, id int64, role string, permissions []string) (domain.User, error) {
	if role == "" {
		role = domain.UserRoleUser
	}
	perms, err := json.Marshal(permissions)
	if err != nil {
		return domain.User{}, err
	}
	var user domain.User
	var rawPermissions []byte
	err = r.db.QueryRowContext(ctx, `
		UPDATE users
		SET role = $2, permissions = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, password_hash, role, permissions, created_at, updated_at
	`, id, role, perms).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &rawPermissions, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return domain.User{}, err
	}
	user.Permissions = decodePermissions(rawPermissions)
	return user, nil
}

func decodePermissions(raw []byte) []string {
	if len(raw) == 0 {
		return []string{domain.PermissionMarketAccess}
	}
	var permissions []string
	if err := json.Unmarshal(raw, &permissions); err != nil || len(permissions) == 0 {
		return []string{domain.PermissionMarketAccess}
	}
	return permissions
}
