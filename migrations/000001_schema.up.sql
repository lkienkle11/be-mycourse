-- Core schema + RBAC seed (single migration).
-- permissions.permission_id = stable catalog PK (e.g. P1). permission_name = JWT / RequirePermission string (resource:action).

CREATE TABLE permissions (
    permission_id VARCHAR(10) PRIMARY KEY,
    permission_name VARCHAR(50) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uix_permissions_permission_name UNIQUE (permission_name)
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
    permission_id VARCHAR(10) NOT NULL REFERENCES permissions (permission_id) ON UPDATE CASCADE ON DELETE CASCADE,
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
    permission_id VARCHAR(10) NOT NULL REFERENCES permissions (permission_id) ON UPDATE CASCADE ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

CREATE INDEX idx_user_permissions_user ON user_permissions (user_id);

INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P1', 'profile:read', '', NOW(), NOW()),
    ('P2', 'profile:update', '', NOW(), NOW()),
    ('P3', 'profile:delete', '', NOW(), NOW()),
    ('P4', 'profile:create', '', NOW(), NOW()),
    ('P5', 'course:read', '', NOW(), NOW()),
    ('P6', 'course:update', '', NOW(), NOW()),
    ('P7', 'course:delete', '', NOW(), NOW()),
    ('P8', 'course:create', '', NOW(), NOW()),
    ('P9', 'course_instructor:read', '', NOW(), NOW()),
    ('P10', 'user:read', '', NOW(), NOW()),
    ('P11', 'user:update', '', NOW(), NOW()),
    ('P12', 'user:delete', '', NOW(), NOW()),
    ('P13', 'user:create', '', NOW(), NOW());

INSERT INTO roles (name, description, created_at, updated_at)
VALUES
    ('sysadmin', 'System-wide administration', NOW(), NOW()),
    ('admin', 'Business administration', NOW(), NOW()),
    ('instructor', 'Manage and teach courses', NOW(), NOW()),
    ('learner', 'Consume learning content', NOW(), NOW());

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'sysadmin';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN (
    'P1',
    'P2',
    'P3',
    'P4',
    'P5',
    'P6',
    'P7',
    'P8',
    'P10',
    'P11',
    'P12',
    'P13'
)
WHERE r.name = 'admin';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P1', 'P5', 'P6', 'P7', 'P9', 'P10')
WHERE r.name = 'instructor';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P1', 'P5', 'P10')
WHERE r.name = 'learner';
