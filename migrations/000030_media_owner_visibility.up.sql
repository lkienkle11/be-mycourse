-- Media ownership + visibility for per-user R2 prefixes and library access control.
ALTER TABLE media_files
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(16) NOT NULL DEFAULT 'private';

CREATE INDEX IF NOT EXISTS idx_media_files_user_id ON media_files (user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_media_files_visibility ON media_files (visibility)
    WHERE deleted_at IS NULL;
