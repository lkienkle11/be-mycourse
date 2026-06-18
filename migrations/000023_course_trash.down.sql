DROP INDEX IF EXISTS idx_courses_trashed_active;

ALTER TABLE courses
    DROP COLUMN IF EXISTS trashed_at;
