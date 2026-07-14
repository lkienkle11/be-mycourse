-- Reverse taxonomy multilingual schema. Leaves JSONB `translations` keys in place
-- (backward-compatible; parsers tolerate unknown fields).

DROP TABLE IF EXISTS tag_translations;
DROP TABLE IF EXISTS course_skill_translations;
DROP TABLE IF EXISTS course_outcome_translations;
DROP TABLE IF EXISTS course_topic_translations;
DROP TABLE IF EXISTS course_level_translations;

ALTER TABLE tags DROP COLUMN IF EXISTS row_version;
ALTER TABLE course_skills DROP COLUMN IF EXISTS row_version;
ALTER TABLE course_outcomes DROP COLUMN IF EXISTS row_version;
ALTER TABLE course_topics DROP COLUMN IF EXISTS row_version;
ALTER TABLE course_levels DROP COLUMN IF EXISTS row_version;
