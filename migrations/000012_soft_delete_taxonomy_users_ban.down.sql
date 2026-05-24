ALTER TABLE users DROP COLUMN IF EXISTS banned_until;

DROP INDEX IF EXISTS uix_tags_slug_active;

ALTER TABLE tags DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE tags ADD CONSTRAINT tags_slug_key UNIQUE (slug);

DROP INDEX IF EXISTS uix_course_skills_slug_active;

ALTER TABLE course_skills DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE course_skills ADD CONSTRAINT course_skills_slug_key UNIQUE (slug);

DROP INDEX IF EXISTS idx_course_outcomes_deleted_at;

ALTER TABLE course_outcomes DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS uix_course_topics_slug_active;

ALTER TABLE course_topics DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE course_topics ADD CONSTRAINT course_topics_slug_key UNIQUE (slug);

DROP INDEX IF EXISTS uix_course_levels_slug_active;

ALTER TABLE course_levels DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE course_levels ADD CONSTRAINT course_levels_slug_key UNIQUE (slug);
