ALTER TABLE users ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';
ALTER TABLE categories ADD COLUMN image_url VARCHAR(512) NOT NULL DEFAULT '';

UPDATE users u SET avatar_url = COALESCE(mf.url, '')
FROM media_files mf
WHERE u.avatar_file_id IS NOT NULL AND mf.id = u.avatar_file_id AND mf.deleted_at IS NULL;

UPDATE categories c SET image_url = COALESCE(mf.url, '')
FROM media_files mf
WHERE c.image_file_id IS NOT NULL AND mf.id = c.image_file_id AND mf.deleted_at IS NULL;

DROP INDEX IF EXISTS idx_users_avatar_file_id;
DROP INDEX IF EXISTS idx_categories_image_file_id;

ALTER TABLE users DROP COLUMN IF EXISTS avatar_file_id;
ALTER TABLE categories DROP COLUMN IF EXISTS image_file_id;
