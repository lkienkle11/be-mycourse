-- Rename categories to course_topics and add nested child_topics JSONB tree.

ALTER TABLE categories RENAME TO course_topics;

ALTER INDEX idx_categories_created_by RENAME TO idx_course_topics_created_by;

ALTER INDEX idx_categories_image_file_id RENAME TO idx_course_topics_image_file_id;

ALTER TABLE course_topics
    ADD COLUMN IF NOT EXISTS child_topics JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE permissions SET permission_name = 'topic:read', updated_at = NOW() WHERE permission_id = 'P18';

UPDATE permissions SET permission_name = 'topic:create', updated_at = NOW() WHERE permission_id = 'P19';

UPDATE permissions SET permission_name = 'topic:update', updated_at = NOW() WHERE permission_id = 'P20';

UPDATE permissions SET permission_name = 'topic:delete', updated_at = NOW() WHERE permission_id = 'P21';

CREATE TABLE course_outcomes (
    id BIGSERIAL PRIMARY KEY,
    short_description VARCHAR(100) NOT NULL,
    description JSONB NOT NULL DEFAULT '[]'::jsonb,
    image_file_id UUID NULL REFERENCES media_files (id) ON DELETE SET NULL,
    status taxonomy_status NOT NULL DEFAULT 'ACTIVE',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_course_outcomes_created_by ON course_outcomes (created_by);

CREATE INDEX idx_course_outcomes_image_file_id ON course_outcomes (image_file_id);

CREATE TABLE course_skills (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    children JSONB NOT NULL DEFAULT '[]'::jsonb,
    status taxonomy_status NOT NULL DEFAULT 'ACTIVE',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_course_skills_created_by ON course_skills (created_by);

INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P30', 'course_outcome:read', '', NOW(), NOW()),
    ('P31', 'course_outcome:create', '', NOW(), NOW()),
    ('P32', 'course_outcome:update', '', NOW(), NOW()),
    ('P33', 'course_outcome:delete', '', NOW(), NOW()),
    ('P34', 'course_skill:read', '', NOW(), NOW()),
    ('P35', 'course_skill:create', '', NOW(), NOW()),
    ('P36', 'course_skill:update', '', NOW(), NOW()),
    ('P37', 'course_skill:delete', '', NOW(), NOW())
ON CONFLICT (permission_id) DO UPDATE
SET permission_name = EXCLUDED.permission_name,
    updated_at = NOW();

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P30', 'P31', 'P32', 'P33', 'P34', 'P35', 'P36', 'P37')
WHERE r.name IN ('sysadmin', 'admin')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P30', 'P34')
WHERE r.name IN ('instructor', 'learner')
ON CONFLICT DO NOTHING;
