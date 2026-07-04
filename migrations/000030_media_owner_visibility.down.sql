DROP INDEX IF EXISTS idx_media_files_visibility;
DROP INDEX IF EXISTS idx_media_files_user_id;

ALTER TABLE media_files
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS user_id;
