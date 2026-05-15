CREATE TABLE IF NOT EXISTS broker_connections (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_encrypted BYTEA NOT NULL,
    token_nonce BYTEA NOT NULL,
    token_hint VARCHAR(50) NOT NULL,
    is_sandbox BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_broker_connections_active_user
    ON broker_connections(user_id)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS idx_broker_connections_user_id
    ON broker_connections(user_id);

INSERT INTO broker_connections (
    user_id,
    name,
    token_encrypted,
    token_nonce,
    token_hint,
    is_sandbox,
    is_active,
    created_at,
    updated_at
)
SELECT
    user_id,
    'Основное подключение',
    token_encrypted,
    token_nonce,
    token_hint,
    COALESCE(is_sandbox, FALSE),
    TRUE,
    created_at,
    updated_at
FROM user_tinkoff_credentials
WHERE NOT EXISTS (
    SELECT 1
    FROM broker_connections bc
    WHERE bc.user_id = user_tinkoff_credentials.user_id
);

CREATE TABLE IF NOT EXISTS broker_account_selections (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    connection_id BIGINT NOT NULL REFERENCES broker_connections(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
