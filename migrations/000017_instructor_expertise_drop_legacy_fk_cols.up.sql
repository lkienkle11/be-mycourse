-- Finalize instructor expertise junction schema on drifted DBs:
-- 1) backfill canonical topic_id/skill_id from legacy columns when present
-- 2) drop legacy course_topic_id/course_skill_id columns (still NOT NULL on some envs)
-- 3) enforce NOT NULL + FK on canonical columns

UPDATE instructor_expertise_topics
SET topic_id = (to_jsonb(instructor_expertise_topics) ->> 'course_topic_id')::BIGINT
WHERE topic_id IS NULL
  AND (to_jsonb(instructor_expertise_topics) ->> 'course_topic_id') IS NOT NULL;

UPDATE instructor_expertise_skills
SET skill_id = (to_jsonb(instructor_expertise_skills) ->> 'course_skill_id')::BIGINT
WHERE skill_id IS NULL
  AND (to_jsonb(instructor_expertise_skills) ->> 'course_skill_id') IS NOT NULL;

ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS instructor_expertise_topics_course_topic_id_fkey;
ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS instructor_expertise_skills_course_skill_id_fkey;

ALTER TABLE instructor_expertise_topics DROP COLUMN IF EXISTS course_topic_id;
ALTER TABLE instructor_expertise_skills DROP COLUMN IF EXISTS course_skill_id;

ALTER TABLE instructor_expertise_topics
    ALTER COLUMN topic_id SET NOT NULL;
ALTER TABLE instructor_expertise_skills
    ALTER COLUMN skill_id SET NOT NULL;

ALTER TABLE instructor_expertise_topics
    DROP CONSTRAINT IF EXISTS instructor_expertise_topics_topic_id_fkey;
ALTER TABLE instructor_expertise_topics
    ADD CONSTRAINT instructor_expertise_topics_topic_id_fkey
    FOREIGN KEY (topic_id) REFERENCES course_topics (id) ON DELETE CASCADE;

ALTER TABLE instructor_expertise_skills
    DROP CONSTRAINT IF EXISTS instructor_expertise_skills_skill_id_fkey;
ALTER TABLE instructor_expertise_skills
    ADD CONSTRAINT instructor_expertise_skills_skill_id_fkey
    FOREIGN KEY (skill_id) REFERENCES course_skills (id) ON DELETE CASCADE;
