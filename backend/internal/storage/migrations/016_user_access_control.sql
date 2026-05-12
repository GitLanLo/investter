ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(32) NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '["market_access"]'::jsonb;

UPDATE users
SET role = 'super_admin',
    permissions = '["market_access", "ml_admin", "user_admin"]'::jsonb
WHERE id = (SELECT MIN(id) FROM users)
  AND role = 'user';

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
