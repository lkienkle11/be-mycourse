-- Instructor dashboard: phone, applications, profiles, expertise, tickets.

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(32) NOT NULL DEFAULT '';

CREATE TABLE instructor_applications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    review_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    rejection_reason TEXT NOT NULL DEFAULT '',
    headline VARCHAR(255) NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    years_of_experience INT NOT NULL DEFAULT 0,
    current_job_title VARCHAR(255) NOT NULL DEFAULT '',
    current_company VARCHAR(255) NOT NULL DEFAULT '',
    cv_file_id VARCHAR(64) NOT NULL DEFAULT '',
    linkedin_url TEXT NOT NULL DEFAULT '',
    github_url TEXT NOT NULL DEFAULT '',
    portfolio_links JSONB NOT NULL DEFAULT '[]',
    certificates JSONB NOT NULL DEFAULT '[]',
    intro_video_file_id VARCHAR(64) NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE UNIQUE INDEX uix_instructor_applications_user_active ON instructor_applications (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_applications_status ON instructor_applications (review_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_applications_user ON instructor_applications (user_id);

CREATE TABLE instructor_profiles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    headline VARCHAR(255) NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    years_of_experience INT NOT NULL DEFAULT 0,
    current_job_title VARCHAR(255) NOT NULL DEFAULT '',
    current_company VARCHAR(255) NOT NULL DEFAULT '',
    cv_file_id VARCHAR(64) NOT NULL DEFAULT '',
    linkedin_url TEXT NOT NULL DEFAULT '',
    github_url TEXT NOT NULL DEFAULT '',
    portfolio_links JSONB NOT NULL DEFAULT '[]',
    certificates JSONB NOT NULL DEFAULT '[]',
    intro_video_file_id VARCHAR(64) NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE UNIQUE INDEX uix_instructor_profiles_user_active ON instructor_profiles (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_profiles_user ON instructor_profiles (user_id);

CREATE TABLE instructor_expertise_topics (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES course_topics (id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE UNIQUE INDEX uix_instructor_expertise_topics_user_topic ON instructor_expertise_topics (user_id, topic_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_expertise_topics_user ON instructor_expertise_topics (user_id);

CREATE TABLE instructor_expertise_skills (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES course_skills (id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE UNIQUE INDEX uix_instructor_expertise_skills_user_skill ON instructor_expertise_skills (user_id, skill_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_expertise_skills_user ON instructor_expertise_skills (user_id);

CREATE TABLE instructor_tickets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    subject VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE INDEX idx_instructor_tickets_user ON instructor_tickets (user_id);
CREATE INDEX idx_instructor_tickets_status ON instructor_tickets (status) WHERE deleted_at IS NULL;

CREATE TABLE instructor_ticket_messages (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES instructor_tickets (id) ON DELETE CASCADE,
    author_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body TEXT NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE INDEX idx_instructor_ticket_messages_ticket ON instructor_ticket_messages (ticket_id);

INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P41', 'instructor_roster:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P42', 'instructor_roster:create', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P43', 'instructor_roster:delete', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P44', 'instructor_application:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P45', 'instructor_application:create', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P46', 'instructor_application:update', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P47', 'instructor_application:delete', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P48', 'instructor_application:approve', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P49', 'instructor_application:reject', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P50', 'instructor_profile:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P51', 'instructor_profile:create', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P52', 'instructor_profile:update', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P53', 'instructor_profile:delete', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P54', 'instructor_expertise:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P55', 'instructor_expertise:create', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P56', 'instructor_expertise:update', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P57', 'instructor_expertise:delete', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P58', 'instructor_ticket:close', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('sysadmin', 'admin')
  AND p.permission_id BETWEEN 'P41' AND 'P58'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P45', 'P47', 'P49', 'P55', 'P56', 'P57', 'P58')
WHERE r.name = 'instructor'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id = 'P45'
WHERE r.name = 'learner'
ON CONFLICT DO NOTHING;
