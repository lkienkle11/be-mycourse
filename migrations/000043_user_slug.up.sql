-- User slug: a stable, URL-safe public identifier for every account.
-- The column is nullable on purpose: existing users are backfilled by the
-- operator command CLI_BACKFILL_USER_SLUGS (SQL cannot transliterate display
-- names), and 000044 applies NOT NULL only after that backfill has finished.
-- A NULL never conflicts in the unique index and passes the CHECK, so this
-- migration applies cleanly on a table that still holds un-backfilled rows.
ALTER TABLE users ADD COLUMN slug VARCHAR(255);

ALTER TABLE users
    ADD CONSTRAINT chk_users_slug_format
    CHECK (slug IS NULL OR (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'));

CREATE UNIQUE INDEX uix_users_slug_active ON users (slug) WHERE deleted_at IS NULL;

-- Former slugs stay reserved so an old public link never resolves to another
-- person. Rows disappear only when the owning user is hard deleted.
CREATE TABLE user_slug_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    slug VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    CONSTRAINT uq_user_slug_history_slug UNIQUE (slug),
    CONSTRAINT chk_user_slug_history_slug_format
        CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$')
);

CREATE INDEX idx_user_slug_history_user_id ON user_slug_history (user_id);
