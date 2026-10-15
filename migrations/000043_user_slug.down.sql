DROP TABLE IF EXISTS user_slug_history;
DROP INDEX IF EXISTS uix_users_slug_active;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_slug_format;
ALTER TABLE users DROP COLUMN IF EXISTS slug;
