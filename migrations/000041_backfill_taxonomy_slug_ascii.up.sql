-- Regenerate any existing taxonomy root slug value that does not conform to
-- the new format (non-ASCII passthrough from utils.SlugifyName, or any
-- other pre-existing malformed value) across all 4 tables, so the CHECK
-- constraints added in 000042 cannot fail to apply. Scans EVERY row
-- regardless of deleted_at (the CHECK constraint applies table-wide, unlike
-- the partial unique indexes which only cover active rows) -- mirrors
-- courses' own 000039 backfill exactly, same reasoning.
--
-- Plain UPDATE statements, not a DO block: this repo's migration runner
-- splits files on every literal semicolon character, including inside
-- comments -- see migrations/README.md.
--
-- created_at/updated_at on these 4 tables were originally TIMESTAMPTZ
-- (migrations 000002/000009) but migration 000011_audit_timestamps_bigint
-- converted them to BIGINT Unix-epoch-seconds, same as courses.updated_at --
-- confirmed empirically against the real dev DB (a first draft of this
-- migration using NOW() failed live with "column \"updated_at\" is of type
-- bigint but expression is of type timestamp with time zone", corrected
-- here before ever landing). Use the same
-- EXTRACT(EPOCH FROM NOW())::BIGINT expression courses' own 000039 uses.
--
-- The replacement slug uses a 16-character lowercase hex suffix from md5()
-- over enough entropy (row id, wall-clock, and random()) that a collision
-- is astronomically unlikely, and md5 output is already confined to
-- [0-9a-f], so no further sanitizing is needed to satisfy the slug format.
UPDATE course_levels
SET slug = 'level-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';

UPDATE course_topics
SET slug = 'topic-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';

UPDATE course_skills
SET slug = 'skill-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';

UPDATE tags
SET slug = 'tag-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';
