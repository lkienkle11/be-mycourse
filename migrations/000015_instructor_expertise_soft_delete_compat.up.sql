-- Backfill compatibility for instructor expertise tables on drifted DBs:
-- 1) ensure deleted_at exists for soft-delete scope
-- 2) normalize legacy course_topic_id/course_skill_id -> topic_id/skill_id
-- 3) recreate active-only unique indexes

ALTER TABLE instructor_expertise_topics
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;
ALTER TABLE instructor_expertise_skills
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'instructor_expertise_topics' AND column_name = 'topic_id'
    ) THEN
        ALTER TABLE instructor_expertise_topics ADD COLUMN topic_id BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'instructor_expertise_topics' AND column_name = 'course_topic_id'
    ) THEN
        UPDATE instructor_expertise_topics
        SET topic_id = course_topic_id
        WHERE topic_id IS NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'instructor_expertise_skills' AND column_name = 'skill_id'
    ) THEN
        ALTER TABLE instructor_expertise_skills ADD COLUMN skill_id BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'instructor_expertise_skills' AND column_name = 'course_skill_id'
    ) THEN
        UPDATE instructor_expertise_skills
        SET skill_id = course_skill_id
        WHERE skill_id IS NULL;
    END IF;
END $$;

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
