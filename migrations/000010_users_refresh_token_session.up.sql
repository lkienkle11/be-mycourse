ALTER TABLE users
    ADD COLUMN IF NOT EXISTS refresh_token_session JSONB NOT NULL DEFAULT '{}';
