-- Soft-delete columns on taxonomy tables and time-limited ban on users.

ALTER TABLE course_levels ADD COLUMN deleted_at BIGINT NULL;

CREATE INDEX idx_course_levels_deleted_at ON course_levels (deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE course_levels DROP CONSTRAINT IF EXISTS course_levels_slug_key;

CREATE UNIQUE INDEX uix_course_levels_slug_active ON course_levels (slug) WHERE deleted_at IS NULL;

ALTER TABLE course_topics ADD COLUMN deleted_at BIGINT NULL;

CREATE INDEX idx_course_topics_deleted_at ON course_topics (deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE course_topics DROP CONSTRAINT IF EXISTS course_topics_slug_key;

ALTER TABLE course_topics DROP CONSTRAINT IF EXISTS categories_slug_key;

CREATE UNIQUE INDEX uix_course_topics_slug_active ON course_topics (slug) WHERE deleted_at IS NULL;

ALTER TABLE course_outcomes ADD COLUMN deleted_at BIGINT NULL;

CREATE INDEX idx_course_outcomes_deleted_at ON course_outcomes (deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE course_skills ADD COLUMN deleted_at BIGINT NULL;

CREATE INDEX idx_course_skills_deleted_at ON course_skills (deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE course_skills DROP CONSTRAINT IF EXISTS course_skills_slug_key;

CREATE UNIQUE INDEX uix_course_skills_slug_active ON course_skills (slug) WHERE deleted_at IS NULL;

ALTER TABLE tags ADD COLUMN deleted_at BIGINT NULL;

CREATE INDEX idx_tags_deleted_at ON tags (deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_slug_key;

CREATE UNIQUE INDEX uix_tags_slug_active ON tags (slug) WHERE deleted_at IS NULL;

ALTER TABLE users ADD COLUMN banned_until BIGINT NULL;
