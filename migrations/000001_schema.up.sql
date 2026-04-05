-- Core schema: flat RBAC + users (replaces former multi-step chain).
-- permissions.code = stable dot-separated key. code_check = colon-separated JWT or RequirePermission string.

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(128) NOT NULL,
    code_check VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uix_permissions_code UNIQUE (code),
    CONSTRAINT uix_permissions_code_check UNIQUE (code_check)
);

CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uix_roles_name UNIQUE (name)
);

CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_code UUID NOT NULL DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    hash_password VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    is_disable BOOLEAN NOT NULL DEFAULT FALSE,
    email_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    confirmation_token VARCHAR(128),
    confirmation_sent_at TIMESTAMPTZ,
    refresh_token_session JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uix_users_email UNIQUE (email),
    CONSTRAINT uix_users_user_code UNIQUE (user_code)
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_user_code ON users (user_code);
CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_confirm_token ON users (confirmation_token) WHERE confirmation_token IS NOT NULL;

CREATE TABLE user_roles (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles (user_id);

CREATE TABLE user_permissions (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

CREATE INDEX idx_user_permissions_user ON user_permissions (user_id);
