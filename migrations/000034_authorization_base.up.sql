CREATE TABLE authorization_actions (
    action_name VARCHAR(100) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    boundary_permission VARCHAR(50) NOT NULL REFERENCES permissions (permission_name) ON UPDATE CASCADE ON DELETE RESTRICT,
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    PRIMARY KEY (action_name, resource_type)
);

CREATE TABLE authorization_grants (
    id UUID PRIMARY KEY,
    principal_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    action_name VARCHAR(100) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(128) NOT NULL,
    effect VARCHAR(8) NOT NULL,
    conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    valid_from BIGINT NULL,
    expires_at BIGINT NULL,
    granted_by_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    revoked_at BIGINT NULL,
    CONSTRAINT fk_authorization_grants_action
        FOREIGN KEY (action_name, resource_type)
        REFERENCES authorization_actions (action_name, resource_type)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT chk_authorization_grants_effect
        CHECK (effect IN ('ALLOW', 'DENY')),
    CONSTRAINT chk_authorization_grants_conditions
        CHECK (jsonb_typeof(conditions) = 'array'),
    CONSTRAINT chk_authorization_grants_validity
        CHECK (expires_at IS NULL OR valid_from IS NULL OR expires_at > valid_from)
);

CREATE INDEX idx_authorization_grants_decision
    ON authorization_grants (principal_user_id, resource_type, resource_id, action_name)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_authorization_grants_resource
    ON authorization_grants (resource_type, resource_id)
    WHERE revoked_at IS NULL;

-- authorization_role_actions is the role definition: which actions a role name currently
-- covers, for a resource type. Redefining it (insert/delete a row) takes effect immediately
-- for every existing holder of that role, since bindings below reference the role by name
-- and never store a copy of its action set.
CREATE TABLE authorization_role_actions (
    role_name VARCHAR(100) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    action_name VARCHAR(100) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    PRIMARY KEY (role_name, resource_type, action_name),
    CONSTRAINT fk_authorization_role_actions_action
        FOREIGN KEY (action_name, resource_type)
        REFERENCES authorization_actions (action_name, resource_type)
        ON UPDATE CASCADE ON DELETE RESTRICT
);

-- authorization_role_bindings is the role assignment: which principal holds which role on
-- which specific resource instance. Its actual actions are resolved by joining role_name
-- against authorization_role_actions at decision time, never stored here.
CREATE TABLE authorization_role_bindings (
    id UUID PRIMARY KEY,
    principal_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_name VARCHAR(100) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(128) NOT NULL,
    granted_by_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    revoked_at BIGINT NULL
);

CREATE INDEX idx_authorization_role_bindings_decision
    ON authorization_role_bindings (principal_user_id, resource_type, resource_id)
    WHERE revoked_at IS NULL;
