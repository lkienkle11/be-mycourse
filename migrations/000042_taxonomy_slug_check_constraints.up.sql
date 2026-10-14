ALTER TABLE course_levels
    ADD CONSTRAINT chk_course_levels_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');

ALTER TABLE course_topics
    ADD CONSTRAINT chk_course_topics_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');

ALTER TABLE course_skills
    ADD CONSTRAINT chk_course_skills_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');

ALTER TABLE tags
    ADD CONSTRAINT chk_tags_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');
