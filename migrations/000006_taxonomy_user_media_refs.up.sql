-- Sub14: categories + users reference uploaded media by FK instead of raw URL strings.
-- Backfill: match legacy image_url / avatar_url to media_files.url or origin_url when possible.

ALTER TABLE categories
    ADD COLUMN image_file_id UUID NULL REFERENCES media_files (id) ON DELETE SET NULL;

ALTER TABLE users
    ADD COLUMN avatar_file_id UUID NULL REFERENCES media_files (id) ON DELETE SET NULL;

CREATE INDEX idx_categories_image_file_id ON categories (image_file_id);
CREATE INDEX idx_users_avatar_file_id ON users (avatar_file_id);

UPDATE categories c
SET image_file_id = mf.id
FROM media_files mf
WHERE c.image_url <> ''
  AND mf.deleted_at IS NULL
  AND (mf.url = c.image_url OR mf.origin_url = c.image_url);

UPDATE users u
SET avatar_file_id = mf.id
FROM media_files mf
WHERE u.avatar_url <> ''
  AND mf.deleted_at IS NULL
  AND (mf.url = u.avatar_url OR mf.origin_url = u.avatar_url);

ALTER TABLE categories DROP COLUMN image_url;
ALTER TABLE users DROP COLUMN avatar_url;
