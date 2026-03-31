-- Flat RBAC: remove role parent/child and closure (replaced by user_roles + role_permissions + user_permissions).
-- On databases that never had hierarchy, DROP IF EXISTS / IF EXISTS are no-ops; user_permissions may already exist from 000001.

DROP TABLE IF EXISTS role_closure;

ALTER TABLE roles DROP COLUMN IF EXISTS parent_id;

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id VARCHAR(128) NOT NULL,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_user ON user_permissions (user_id);
