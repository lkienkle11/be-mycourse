ALTER TABLE course_levels DROP CONSTRAINT IF EXISTS chk_course_levels_slug_format;
ALTER TABLE course_topics DROP CONSTRAINT IF EXISTS chk_course_topics_slug_format;
ALTER TABLE course_skills DROP CONSTRAINT IF EXISTS chk_course_skills_slug_format;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS chk_tags_slug_format;
