-- Taxonomy multilingual hybrid + optimistic lock (plan Phase B).
-- 1) row_version on five taxonomy roots
-- 2) five *_translations tables + locale='en' backfill from canonical columns
-- 3) JSONB tree patch: ensure translations.en.name on child_topics / children (idempotent)

-- ---------------------------------------------------------------------------
-- row_version (mirror course/media)
-- ---------------------------------------------------------------------------
ALTER TABLE course_levels
    ADD COLUMN IF NOT EXISTS row_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE course_topics
    ADD COLUMN IF NOT EXISTS row_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE course_outcomes
    ADD COLUMN IF NOT EXISTS row_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE course_skills
    ADD COLUMN IF NOT EXISTS row_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tags
    ADD COLUMN IF NOT EXISTS row_version BIGINT NOT NULL DEFAULT 1;

UPDATE course_levels SET row_version = 1 WHERE row_version = 0;
UPDATE course_topics SET row_version = 1 WHERE row_version = 0;
UPDATE course_outcomes SET row_version = 1 WHERE row_version = 0;
UPDATE course_skills SET row_version = 1 WHERE row_version = 0;
UPDATE tags SET row_version = 1 WHERE row_version = 0;

-- ---------------------------------------------------------------------------
-- translation tables
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS course_level_translations (
    id UUID PRIMARY KEY,
    course_level_id UUID NOT NULL REFERENCES course_levels (id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uix_course_level_translations_parent_locale UNIQUE (course_level_id, locale)
);
CREATE INDEX IF NOT EXISTS idx_course_level_translations_locale
    ON course_level_translations (locale);

CREATE TABLE IF NOT EXISTS course_topic_translations (
    id UUID PRIMARY KEY,
    topic_id UUID NOT NULL REFERENCES course_topics (id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uix_course_topic_translations_parent_locale UNIQUE (topic_id, locale)
);
CREATE INDEX IF NOT EXISTS idx_course_topic_translations_locale
    ON course_topic_translations (locale);

CREATE TABLE IF NOT EXISTS course_outcome_translations (
    id UUID PRIMARY KEY,
    outcome_id UUID NOT NULL REFERENCES course_outcomes (id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    short_description VARCHAR(100) NOT NULL,
    description JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uix_course_outcome_translations_parent_locale UNIQUE (outcome_id, locale)
);
CREATE INDEX IF NOT EXISTS idx_course_outcome_translations_locale
    ON course_outcome_translations (locale);

CREATE TABLE IF NOT EXISTS course_skill_translations (
    id UUID PRIMARY KEY,
    skill_id UUID NOT NULL REFERENCES course_skills (id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uix_course_skill_translations_parent_locale UNIQUE (skill_id, locale)
);
CREATE INDEX IF NOT EXISTS idx_course_skill_translations_locale
    ON course_skill_translations (locale);

CREATE TABLE IF NOT EXISTS tag_translations (
    id UUID PRIMARY KEY,
    tag_id UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uix_tag_translations_parent_locale UNIQUE (tag_id, locale)
);
CREATE INDEX IF NOT EXISTS idx_tag_translations_locale
    ON tag_translations (locale);

-- ---------------------------------------------------------------------------
-- backfill locale='en' from canonical columns (idempotent via NOT EXISTS)
-- ---------------------------------------------------------------------------
INSERT INTO course_level_translations (id, course_level_id, locale, name, created_at, updated_at)
SELECT gen_random_uuid(), cl.id, 'en', cl.name,
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM course_levels cl
WHERE NOT EXISTS (
    SELECT 1 FROM course_level_translations t
    WHERE t.course_level_id = cl.id AND t.locale = 'en'
);

INSERT INTO course_topic_translations (id, topic_id, locale, name, created_at, updated_at)
SELECT gen_random_uuid(), ct.id, 'en', ct.name,
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM course_topics ct
WHERE NOT EXISTS (
    SELECT 1 FROM course_topic_translations t
    WHERE t.topic_id = ct.id AND t.locale = 'en'
);

INSERT INTO course_outcome_translations (
    id, outcome_id, locale, short_description, description, created_at, updated_at
)
SELECT gen_random_uuid(), co.id, 'en', co.short_description, co.description,
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM course_outcomes co
WHERE NOT EXISTS (
    SELECT 1 FROM course_outcome_translations t
    WHERE t.outcome_id = co.id AND t.locale = 'en'
);

INSERT INTO course_skill_translations (id, skill_id, locale, name, created_at, updated_at)
SELECT gen_random_uuid(), cs.id, 'en', cs.name,
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM course_skills cs
WHERE NOT EXISTS (
    SELECT 1 FROM course_skill_translations t
    WHERE t.skill_id = cs.id AND t.locale = 'en'
);

INSERT INTO tag_translations (id, tag_id, locale, name, created_at, updated_at)
SELECT gen_random_uuid(), tg.id, 'en', tg.name,
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM tags tg
WHERE NOT EXISTS (
    SELECT 1 FROM tag_translations t
    WHERE t.tag_id = tg.id AND t.locale = 'en'
);

-- ---------------------------------------------------------------------------
-- JSONB tree: ensure translations.en.name (fail on invalid/empty nodes)
-- NOTE: golang-migrate splits on every semicolon — do NOT use plpgsql or DO blocks
-- with internal semicolons. Use LANGUAGE sql + $fn$ with zero semicolons inside the body.
-- ---------------------------------------------------------------------------

-- Fail fast on corrupt non-array trees (no DO block).
-- THEN branch must be non-constant so Postgres does not eagerly type-check a literal CAST.
SELECT CASE
    WHEN COUNT(*) > 0 THEN (
        ('migration 000032 course_topics non-array child_topics n=' || COUNT(*)::text)::integer
    )
    ELSE 0
END
FROM course_topics
WHERE child_topics IS NOT NULL
  AND jsonb_typeof(child_topics) <> 'array';

SELECT CASE
    WHEN COUNT(*) > 0 THEN (
        ('migration 000032 course_skills non-array children n=' || COUNT(*)::text)::integer
    )
    ELSE 0
END
FROM course_skills
WHERE children IS NOT NULL
  AND jsonb_typeof(children) <> 'array';

DROP FUNCTION IF EXISTS taxonomy_ensure_tree_en_translations(jsonb, text);
DROP FUNCTION IF EXISTS taxonomy_ensure_tree_en_translations(jsonb);

CREATE OR REPLACE FUNCTION taxonomy_ensure_tree_en_translations(nodes jsonb)
RETURNS jsonb
LANGUAGE sql
IMMUTABLE
AS $fn$
SELECT CASE
  WHEN nodes IS NULL THEN '[]'::jsonb
  WHEN jsonb_typeof(nodes) <> 'array' THEN (
    ('migration 000032 taxonomy tree must be a JSON array got ' || jsonb_typeof(nodes))::jsonb
  )
  ELSE (
    SELECT COALESCE(jsonb_agg(patched.elem ORDER BY patched.ord), '[]'::jsonb)
    FROM (
      SELECT
        t.ord,
        CASE
          WHEN jsonb_typeof(t.elem) <> 'object' THEN (
            ('migration 000032 taxonomy tree node must be an object got ' || jsonb_typeof(t.elem) || ' ord=' || t.ord::text)::jsonb
          )
          WHEN NULLIF(btrim(COALESCE(t.elem ->> 'id', '')), '') IS NULL THEN (
            ('migration 000032 taxonomy tree node missing id ord=' || t.ord::text)::jsonb
          )
          WHEN NULLIF(btrim(COALESCE(t.elem ->> 'name', '')), '') IS NULL THEN (
            ('migration 000032 taxonomy tree node has empty name ord=' || t.ord::text)::jsonb
          )
          ELSE (
            t.elem || jsonb_build_object(
              'translations',
              CASE
                WHEN (t.elem -> 'translations') IS NULL
                  OR jsonb_typeof(t.elem -> 'translations') <> 'object'
                THEN jsonb_build_object(
                  'en', jsonb_build_object('name', t.elem ->> 'name')
                )
                WHEN NULLIF(
                  btrim(COALESCE(t.elem #>> '{translations,en,name}', '')),
                  ''
                ) IS NULL
                THEN jsonb_set(
                  t.elem -> 'translations',
                  '{en}',
                  jsonb_build_object('name', t.elem ->> 'name'),
                  true
                )
                ELSE t.elem -> 'translations'
              END,
              'children',
              taxonomy_ensure_tree_en_translations(
                CASE
                  WHEN (t.elem -> 'children') IS NULL THEN '[]'::jsonb
                  WHEN jsonb_typeof(t.elem -> 'children') <> 'array' THEN (
                    ('migration 000032 taxonomy tree children must be a JSON array got ' || jsonb_typeof(t.elem -> 'children') || ' ord=' || t.ord::text)::jsonb
                  )
                  ELSE t.elem -> 'children'
                END
              )
            )
          )
        END AS elem
      FROM jsonb_array_elements(nodes) WITH ORDINALITY AS t(elem, ord)
    ) AS patched
  )
END
$fn$;

UPDATE course_topics
SET child_topics = taxonomy_ensure_tree_en_translations(child_topics)
WHERE child_topics IS NOT NULL;

UPDATE course_skills
SET children = taxonomy_ensure_tree_en_translations(children)
WHERE children IS NOT NULL;

DROP FUNCTION IF EXISTS taxonomy_ensure_tree_en_translations(jsonb);
