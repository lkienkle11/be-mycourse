ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS uix_instructor_expertise_skills_user_skill;
DROP INDEX IF EXISTS uix_instructor_expertise_skills_user_skill;
ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS uix_instructor_expertise_topics_user_topic;
DROP INDEX IF EXISTS uix_instructor_expertise_topics_user_topic;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'instructor_expertise_topics' AND column_name = 'course_topic_id'
    ) THEN
        CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_topics_user_topic
            ON instructor_expertise_topics (user_id, course_topic_id);
    ELSE
        CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_topics_user_topic
            ON instructor_expertise_topics (user_id, topic_id);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'instructor_expertise_skills' AND column_name = 'course_skill_id'
    ) THEN
        CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_skills_user_skill
            ON instructor_expertise_skills (user_id, course_skill_id);
    ELSE
        CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_expertise_skills_user_skill
            ON instructor_expertise_skills (user_id, skill_id);
    END IF;
END $$;

ALTER TABLE instructor_expertise_skills DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE instructor_expertise_topics DROP COLUMN IF EXISTS deleted_at;
