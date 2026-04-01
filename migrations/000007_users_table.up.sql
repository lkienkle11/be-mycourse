-- Custom users table for email+password authentication.
-- user_code is a UUID (populated as UUIDv7 at the application layer).
-- deleted_at enables GORM soft-delete.

CREATE TABLE IF NOT EXISTS users (
    id                    BIGSERIAL    PRIMARY KEY,
    user_code             UUID         NOT NULL DEFAULT gen_random_uuid(),
    email                 VARCHAR(255) NOT NULL,
    hash_password         VARCHAR(255) NOT NULL,
    display_name          VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url            TEXT         NOT NULL DEFAULT '',
    is_disable            BOOLEAN      NOT NULL DEFAULT FALSE,
    email_confirmed       BOOLEAN      NOT NULL DEFAULT FALSE,
    confirmation_token    VARCHAR(128),
    confirmation_sent_at  TIMESTAMPTZ,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,
    CONSTRAINT uix_users_email      UNIQUE (email),
    CONSTRAINT uix_users_user_code  UNIQUE (user_code)
);

CREATE INDEX IF NOT EXISTS idx_users_email             ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_user_code         ON users (user_code);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at        ON users (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_confirm_token     ON users (confirmation_token) WHERE confirmation_token IS NOT NULL;
