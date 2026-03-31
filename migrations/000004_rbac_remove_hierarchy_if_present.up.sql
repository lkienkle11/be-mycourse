-- For databases that already applied the previous 000003 (role hierarchy), remove closure/parent and ensure user_permissions.
-- Idempotent: safe if 000003_rbac_flat already ran on a fresh install.

DROP TABLE IF EXISTS role_closure;

ALTER TABLE roles DROP COLUMN IF EXISTS parent_id;

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id VARCHAR(128) NOT NULL,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_user ON user_permissions (user_id);
