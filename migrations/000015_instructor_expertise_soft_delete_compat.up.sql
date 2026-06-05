-- Backfill compatibility for instructor expertise tables on drifted DBs:
-- 1) ensure deleted_at exists for soft-delete scope
-- 2) normalize legacy course_topic_id/course_skill_id -> topic_id/skill_id
-- 3) recreate active-only unique indexes

ALTER TABLE instructor_expertise_topics
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;
ALTER TABLE instructor_expertise_skills
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;

-- Avoid DO $$ blocks because golang-migrate is configured with
-- MultiStatementEnabled, which can split PL/pgSQL blocks by semicolons.
ALTER TABLE instructor_expertise_topics
    ADD COLUMN IF NOT EXISTS topic_id BIGINT;
UPDATE instructor_expertise_topics
SET topic_id = (to_jsonb(instructor_expertise_topics) ->> 'course_topic_id')::BIGINT
WHERE topic_id IS NULL
  AND (to_jsonb(instructor_expertise_topics) ->> 'course_topic_id') IS NOT NULL;

ALTER TABLE instructor_expertise_skills
    ADD COLUMN IF NOT EXISTS skill_id BIGINT;
UPDATE instructor_expertise_skills
SET skill_id = (to_jsonb(instructor_expertise_skills) ->> 'course_skill_id')::BIGINT
WHERE skill_id IS NULL
  AND (to_jsonb(instructor_expertise_skills) ->> 'course_skill_id') IS NOT NULL;

ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS uix_instructor_expertise_topics_user_topic;
DROP INDEX IF EXISTS uix_instructor_expertise_topics_user_topic;

ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS uix_instructor_expertise_skills_user_skill;
DROP INDEX IF EXISTS uix_instructor_expertise_skills_user_skill;

CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_topics_user_topic
    ON instructor_expertise_topics (user_id, topic_id)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_skills_user_skill
    ON instructor_expertise_skills (user_id, skill_id)
    WHERE deleted_at IS NULL;
