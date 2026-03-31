-- Restore hierarchy schema only (no closure data backfill).
DROP TABLE IF EXISTS user_permissions;

ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES roles (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_roles_parent_id ON roles (parent_id);

CREATE TABLE IF NOT EXISTS role_closure (
    ancestor_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    descendant_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (ancestor_id, descendant_id)
);

CREATE INDEX IF NOT EXISTS idx_role_closure_desc ON role_closure (descendant_id);
CREATE INDEX IF NOT EXISTS idx_role_closure_anc ON role_closure (ancestor_id);
