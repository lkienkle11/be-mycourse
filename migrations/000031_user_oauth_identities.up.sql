-- External OAuth/OIDC identities (Google, X, future providers) + local-password marker on users.
-- Time columns on user_oauth_identities use BIGINT epoch seconds to align with the users table.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_set_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN users.password_set_at IS
    'When the user last set a local MyCourse password, NULL means OAuth-only or pending register.';

UPDATE users
SET password_set_at = to_timestamp(created_at)
WHERE password_set_at IS NULL
  AND email_confirmed = true
  AND hash_password IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_oauth_identities (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider        VARCHAR(32) NOT NULL,
    provider_sub    VARCHAR(255) NOT NULL,
    provider_email  VARCHAR(255) NULL,
    linked_at       BIGINT NOT NULL,
    last_login_at   BIGINT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      BIGINT NOT NULL,
    updated_at      BIGINT NOT NULL,

    CONSTRAINT uix_oauth_provider_sub UNIQUE (provider, provider_sub)
);

CREATE INDEX IF NOT EXISTS idx_oauth_identities_user_id
    ON user_oauth_identities (user_id);

CREATE INDEX IF NOT EXISTS idx_oauth_identities_provider_email
    ON user_oauth_identities (provider, lower(provider_email))
    WHERE provider_email IS NOT NULL;

COMMENT ON TABLE user_oauth_identities IS
    'External OAuth/OIDC identities linked to app users.';
COMMENT ON COLUMN user_oauth_identities.provider IS
    'Stable slug: google, x, or another provider slug.';
COMMENT ON COLUMN user_oauth_identities.provider_sub IS
    'Provider stable subject (Google OIDC sub, X user id, and similar).';
COMMENT ON COLUMN user_oauth_identities.metadata IS
    'Non-authoritative snapshot: name, picture url, channel, user_agent.';
