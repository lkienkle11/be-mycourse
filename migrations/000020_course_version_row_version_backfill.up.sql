-- GORM inserts row_version=0 on create, overriding the column DEFAULT 1.
-- Normalize legacy rows so optimistic-lock basic-info saves accept expected_row_version >= 1.
UPDATE course_versions
SET row_version = 1
WHERE row_version = 0;
