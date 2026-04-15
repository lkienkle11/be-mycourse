-- Core schema + RBAC seed in one migration.
-- permissions.code = stable dot-separated key. action = colon-separated JWT or RequirePermission string.

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(128) NOT NULL,
    action VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uix_permissions_code UNIQUE (code),
    CONSTRAINT uix_permissions_action UNIQUE (action)
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

INSERT INTO permissions (code, action, description, created_at, updated_at)
VALUES
    ('user.create', 'user:create', '', NOW(), NOW()),
    ('user.delete', 'user:delete', '', NOW(), NOW()),
    ('user.read', 'user:read', '', NOW(), NOW()),
    ('user.update', 'user:update', '', NOW(), NOW()),
    ('course.create', 'course:create', '', NOW(), NOW()),
    ('course.delete', 'course:delete', '', NOW(), NOW()),
    ('course.read', 'course:read', '', NOW(), NOW()),
    ('course.update', 'course:update', '', NOW(), NOW()),
    ('user_admin.create', 'user_admin:create', '', NOW(), NOW()),
    ('user_admin.delete', 'user_admin:delete', '', NOW(), NOW()),
    ('user_admin.read', 'user_admin:read', '', NOW(), NOW()),
    ('user_admin.update', 'user_admin:update', '', NOW(), NOW());

INSERT INTO roles (name, description, created_at, updated_at)
VALUES
    ('sysadmin', 'System-wide administration', NOW(), NOW()),
    ('admin', 'Business administration', NOW(), NOW()),
    ('instructor', 'Manage and teach courses', NOW(), NOW()),
    ('learner', 'Consume learning content', NOW(), NOW());

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'sysadmin';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
INNER JOIN permissions p ON p.code IN (
    'user.read',
    'user.create',
    'user.update',
    'user.delete',
    'course.read',
    'course.create',
    'course.update',
    'course.delete'
)
WHERE r.name = 'admin';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
INNER JOIN permissions p ON p.code IN (
    'user.read',
    'course.read',
    'course.create',
    'course.update'
)
WHERE r.name = 'instructor';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
INNER JOIN permissions p ON p.code IN (
    'user.read',
    'course.read'
)
WHERE r.name = 'learner';
