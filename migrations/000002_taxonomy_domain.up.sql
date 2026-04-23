CREATE TYPE taxonomy_status AS ENUM ('ACTIVE', 'INACTIVE');

CREATE TABLE course_levels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    status taxonomy_status NOT NULL DEFAULT 'ACTIVE',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_course_levels_created_by ON course_levels (created_by);

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    image_url VARCHAR(512) NOT NULL DEFAULT '',
    status taxonomy_status NOT NULL DEFAULT 'ACTIVE',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_created_by ON categories (created_by);

CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    status taxonomy_status NOT NULL DEFAULT 'ACTIVE',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_created_by ON tags (created_by);

INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P14', 'course_level:read', '', NOW(), NOW()),
    ('P15', 'course_level:create', '', NOW(), NOW()),
    ('P16', 'course_level:update', '', NOW(), NOW()),
    ('P17', 'course_level:delete', '', NOW(), NOW()),
    ('P18', 'category:read', '', NOW(), NOW()),
    ('P19', 'category:create', '', NOW(), NOW()),
    ('P20', 'category:update', '', NOW(), NOW()),
    ('P21', 'category:delete', '', NOW(), NOW()),
    ('P22', 'tag:read', '', NOW(), NOW()),
    ('P23', 'tag:create', '', NOW(), NOW()),
    ('P24', 'tag:update', '', NOW(), NOW()),
    ('P25', 'tag:delete', '', NOW(), NOW())
ON CONFLICT (permission_id) DO UPDATE
SET permission_name = EXCLUDED.permission_name,
    updated_at = NOW();
