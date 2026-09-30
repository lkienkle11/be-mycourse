-- Index-only migration for the public "trending courses" / "popular instructors" catalog
-- endpoints and the authenticated "continue learning" endpoint (openspec change
-- add-home-catalog-apis). No column or table changes — courses had no index at all on
-- created_at, and course_enrollments had no index on user_id alone (only the composite
-- unique (course_id, user_id)).

-- Speeds up "trending courses": ORDER BY created_at DESC over published, non-trashed courses.
CREATE INDEX idx_courses_published_created_at
    ON courses (created_at DESC)
    WHERE deleted_at IS NULL AND trashed_at IS NULL AND current_published_version_id IS NOT NULL;

-- Speeds up "popular instructors": GROUP BY owner_user_id + MAX(created_at) tie-break, same filter.
CREATE INDEX idx_courses_owner_published_created_at
    ON courses (owner_user_id, created_at DESC)
    WHERE deleted_at IS NULL AND trashed_at IS NULL AND current_published_version_id IS NOT NULL;

-- Speeds up "continue learning": WHERE user_id = ? across a learner's enrollments.
CREATE INDEX idx_course_enrollments_user_active
    ON course_enrollments (user_id)
    WHERE deleted_at IS NULL;
