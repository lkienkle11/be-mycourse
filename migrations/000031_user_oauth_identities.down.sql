DROP TABLE IF EXISTS user_oauth_identities;

ALTER TABLE users DROP COLUMN IF EXISTS password_set_at;
