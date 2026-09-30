-- Regenerate any existing courses.slug value that does not conform to the
-- new format (non-ASCII passthrough from the pre-existing utils.SlugifyName,
-- or any other pre-existing malformed value) so the CHECK constraint added
-- in 000040 cannot fail to apply. Scans EVERY row regardless of deleted_at
-- (the CHECK constraint applies table-wide, unlike the partial unique index
-- uix_courses_slug_active which only covers active rows).
--
-- Plain single-statement UPDATE, not a DO block or PL/pgSQL body: this
-- repo's migration runner splits files on every literal semicolon character
-- (see migrations/README.md), even one appearing inside a comment like this
-- one, so no comment line in this file may contain that character at all.
-- The replacement slug uses a 16-character lowercase hex suffix from md5()
-- over enough entropy (row id, wall-clock, and random()) that a collision is
-- astronomically unlikely, and md5 output is already confined to [0-9a-f],
-- so no further sanitizing is needed to satisfy the slug format. If a
-- collision with an active row's slug ever did occur, this UPDATE would fail
-- on uix_courses_slug_active and could simply be re-run.
UPDATE courses
SET slug = 'course-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = extract(epoch FROM now())::bigint
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';
