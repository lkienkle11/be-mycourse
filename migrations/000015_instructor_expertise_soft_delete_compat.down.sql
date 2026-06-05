ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS uix_instructor_expertise_skills_user_skill;
DROP INDEX IF EXISTS uix_instructor_expertise_skills_user_skill;
ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS uix_instructor_expertise_topics_user_topic;
DROP INDEX IF EXISTS uix_instructor_expertise_topics_user_topic;

-- Avoid DO $$ blocks here because golang-migrate is configured with
-- MultiStatementEnabled, which can split PL/pgSQL blocks by semicolons.
CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_topics_user_topic
    ON instructor_expertise_topics (user_id, topic_id);
CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_skills_user_skill
    ON instructor_expertise_skills (user_id, skill_id);

ALTER TABLE instructor_expertise_skills DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE instructor_expertise_topics DROP COLUMN IF EXISTS deleted_at;
