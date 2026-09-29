-- Database-level backstop so courses.slug can never be NULL, empty, or
-- malformed even if application-layer validation has a bug. Requires
-- migration 000039 to have already regenerated any pre-existing
-- non-conforming row, or this ALTER TABLE fails to apply.
ALTER TABLE courses
    ADD CONSTRAINT chk_courses_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');
