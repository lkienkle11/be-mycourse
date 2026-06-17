ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS trashed_at BIGINT NULL;

CREATE INDEX IF NOT EXISTS idx_courses_trashed_active
    ON courses (trashed_at)
    WHERE deleted_at IS NULL AND trashed_at IS NOT NULL;
