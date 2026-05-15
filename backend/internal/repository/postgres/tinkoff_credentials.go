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

func (r *TinkoffCredentialRepository) CreateBrokerConnection(ctx context.Context, connection domain.BrokerConnection) (domain.BrokerConnection, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	defer tx.Rollback()

	if connection.IsActive {
		if _, err := tx.ExecContext(ctx, `UPDATE broker_connections SET is_active = FALSE, updated_at = NOW() WHERE user_id = $1`, connection.UserID); err != nil {
			return domain.BrokerConnection{}, err
		}
	}

	query := `
		INSERT INTO broker_connections (user_id, name, token_encrypted, token_nonce, token_hint, is_sandbox, is_active, last_sync_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, name, token_encrypted, token_nonce, token_hint, is_sandbox, is_active, last_sync_at, created_at, updated_at
	`
	var out domain.BrokerConnection
	err = tx.QueryRowContext(
		ctx,
		query,
		connection.UserID,
		connection.Name,
		connection.TokenEncrypted,
		connection.TokenNonce,
		connection.TokenHint,
		connection.IsSandbox,
		connection.IsActive,
		connection.LastSyncAt,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.Name,
		&out.TokenEncrypted,
		&out.TokenNonce,
		&out.TokenHint,
		&out.IsSandbox,
		&out.IsActive,
		&out.LastSyncAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.BrokerConnection{}, err
	}
	return out, nil
}

func (r *TinkoffCredentialRepository) UpdateBrokerConnection(ctx context.Context, connection domain.BrokerConnection) (domain.BrokerConnection, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	defer tx.Rollback()

	if connection.IsActive {
		if _, err := tx.ExecContext(ctx, `UPDATE broker_connections SET is_active = FALSE, updated_at = NOW() WHERE user_id = $1 AND id <> $2`, connection.UserID, connection.ID); err != nil {
			return domain.BrokerConnection{}, err
		}
	}

	query := `
		UPDATE broker_connections
		SET name = $3,
		    token_encrypted = $4,
		    token_nonce = $5,
		    token_hint = $6,
		    is_sandbox = $7,
		    is_active = $8,
		    last_sync_at = $9,
		    updated_at = NOW()
		WHERE user_id = $1 AND id = $2
		RETURNING id, user_id, name, token_encrypted, token_nonce, token_hint, is_sandbox, is_active, last_sync_at, created_at, updated_at
	`
	var out domain.BrokerConnection
	err = tx.QueryRowContext(
		ctx,
		query,
		connection.UserID,
		connection.ID,
		connection.Name,
		connection.TokenEncrypted,
		connection.TokenNonce,
		connection.TokenHint,
		connection.IsSandbox,
		connection.IsActive,
		connection.LastSyncAt,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.Name,
		&out.TokenEncrypted,
		&out.TokenNonce,
		&out.TokenHint,
		&out.IsSandbox,
		&out.IsActive,
		&out.LastSyncAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.BrokerConnection{}, err
	}
	return out, nil
}

func (r *TinkoffCredentialRepository) ListBrokerConnections(ctx context.Context, userID int64) ([]domain.BrokerConnection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, token_encrypted, token_nonce, token_hint, is_sandbox, is_active, last_sync_at, created_at, updated_at
		FROM broker_connections
		WHERE user_id = $1
		ORDER BY is_active DESC, id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.BrokerConnection
	for rows.Next() {
		var item domain.BrokerConnection
		if err := scanBrokerConnection(rows.Scan, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *TinkoffCredentialRepository) GetBrokerConnection(ctx context.Context, userID int64, connectionID int64) (domain.BrokerConnection, error) {
	var item domain.BrokerConnection
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, token_encrypted, token_nonce, token_hint, is_sandbox, is_active, last_sync_at, created_at, updated_at
		FROM broker_connections
		WHERE user_id = $1 AND id = $2
	`, userID, connectionID).Scan(
		&item.ID,
		&item.UserID,
		&item.Name,
		&item.TokenEncrypted,
		&item.TokenNonce,
		&item.TokenHint,
		&item.IsSandbox,
		&item.IsActive,
		&item.LastSyncAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	return item, nil
}

func (r *TinkoffCredentialRepository) GetActiveBrokerConnection(ctx context.Context, userID int64) (domain.BrokerConnection, error) {
	var item domain.BrokerConnection
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, token_encrypted, token_nonce, token_hint, is_sandbox, is_active, last_sync_at, created_at, updated_at
		FROM broker_connections
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY id
		LIMIT 1
	`, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.Name,
		&item.TokenEncrypted,
		&item.TokenNonce,
		&item.TokenHint,
		&item.IsSandbox,
		&item.IsActive,
		&item.LastSyncAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	return item, nil
}

func (r *TinkoffCredentialRepository) SetActiveBrokerConnection(ctx context.Context, userID int64, connectionID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `UPDATE broker_connections SET is_active = FALSE, updated_at = NOW() WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	_ = result

	res, err := tx.ExecContext(ctx, `UPDATE broker_connections SET is_active = TRUE, updated_at = NOW() WHERE user_id = $1 AND id = $2`, userID, connectionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *TinkoffCredentialRepository) DeleteBrokerConnection(ctx context.Context, userID int64, connectionID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var wasActive bool
	err = tx.QueryRowContext(ctx, `DELETE FROM broker_connections WHERE user_id = $1 AND id = $2 RETURNING is_active`, userID, connectionID).Scan(&wasActive)
	if err != nil {
		return err
	}
	if wasActive {
		_, err = tx.ExecContext(ctx, `
			UPDATE broker_connections
			SET is_active = TRUE, updated_at = NOW()
			WHERE id = (
				SELECT id FROM broker_connections
				WHERE user_id = $1
				ORDER BY id
				LIMIT 1
			)
		`, userID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *TinkoffCredentialRepository) SaveBrokerAccountSelection(ctx context.Context, selection domain.BrokerAccountSelection) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO broker_account_selections (user_id, connection_id, account_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET connection_id = EXCLUDED.connection_id,
		    account_id = EXCLUDED.account_id,
		    updated_at = NOW()
	`, selection.UserID, selection.ConnectionID, selection.AccountID)
	return err
}

func (r *TinkoffCredentialRepository) GetBrokerAccountSelection(ctx context.Context, userID int64) (domain.BrokerAccountSelection, error) {
	var item domain.BrokerAccountSelection
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id, connection_id, account_id, updated_at
		FROM broker_account_selections
		WHERE user_id = $1
	`, userID).Scan(&item.UserID, &item.ConnectionID, &item.AccountID, &item.UpdatedAt)
	if err != nil {
		return domain.BrokerAccountSelection{}, err
	}
	return item, nil
}

type scanFunc func(dest ...any) error

func scanBrokerConnection(scan scanFunc, item *domain.BrokerConnection) error {
	return scan(
		&item.ID,
		&item.UserID,
		&item.Name,
		&item.TokenEncrypted,
		&item.TokenNonce,
		&item.TokenHint,
		&item.IsSandbox,
		&item.IsActive,
		&item.LastSyncAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
}
