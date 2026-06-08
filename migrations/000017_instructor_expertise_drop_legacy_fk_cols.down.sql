-- Restore legacy course_topic_id/course_skill_id columns for rollback.

ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS instructor_expertise_topics_topic_id_fkey;
ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS instructor_expertise_skills_skill_id_fkey;

ALTER TABLE instructor_expertise_topics
    ALTER COLUMN topic_id DROP NOT NULL;
ALTER TABLE instructor_expertise_skills
    ALTER COLUMN skill_id DROP NOT NULL;

ALTER TABLE instructor_expertise_topics
    ADD COLUMN IF NOT EXISTS course_topic_id BIGINT;
UPDATE instructor_expertise_topics
SET course_topic_id = topic_id
WHERE course_topic_id IS NULL
  AND topic_id IS NOT NULL;
ALTER TABLE instructor_expertise_topics
    ALTER COLUMN course_topic_id SET NOT NULL;
ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS instructor_expertise_topics_course_topic_id_fkey;
ALTER TABLE instructor_expertise_topics
    ADD CONSTRAINT instructor_expertise_topics_course_topic_id_fkey
    FOREIGN KEY (course_topic_id) REFERENCES course_topics (id) ON DELETE CASCADE;

ALTER TABLE instructor_expertise_skills
    ADD COLUMN IF NOT EXISTS course_skill_id BIGINT;
UPDATE instructor_expertise_skills
SET course_skill_id = skill_id
WHERE course_skill_id IS NULL
  AND skill_id IS NOT NULL;
ALTER TABLE instructor_expertise_skills
    ALTER COLUMN course_skill_id SET NOT NULL;
ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS instructor_expertise_skills_course_skill_id_fkey;
ALTER TABLE instructor_expertise_skills
    ADD CONSTRAINT instructor_expertise_skills_course_skill_id_fkey
    FOREIGN KEY (course_skill_id) REFERENCES course_skills (id) ON DELETE CASCADE;
