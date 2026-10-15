-- Release 2 migration: apply only after the user slug backfill command has run
-- and SELECT COUNT(*) FROM users WHERE slug IS NULL returns 0. It fails on
-- purpose when any row is still NULL.
ALTER TABLE users ALTER COLUMN slug SET NOT NULL;
