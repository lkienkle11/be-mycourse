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
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION taxonomy_ensure_tree_en_translations(nodes jsonb, path text DEFAULT '$')
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    result jsonb := '[]'::jsonb;
    elem jsonb;
    i int;
    node_id text;
    node_name text;
    translations jsonb;
    en_name text;
    children jsonb;
    child_path text;
BEGIN
    IF nodes IS NULL THEN
        RETURN '[]'::jsonb;
    END IF;
    IF jsonb_typeof(nodes) <> 'array' THEN
        RAISE EXCEPTION 'taxonomy tree at % must be a JSON array, got %', path, jsonb_typeof(nodes);
    END IF;

    FOR i IN 0 .. COALESCE(jsonb_array_length(nodes), 0) - 1 LOOP
        elem := nodes -> i;
        child_path := path || '[' || i || ']';

        IF jsonb_typeof(elem) <> 'object' THEN
            RAISE EXCEPTION 'taxonomy tree node at % must be an object', child_path;
        END IF;

        node_id := elem ->> 'id';
        IF node_id IS NULL OR btrim(node_id) = '' THEN
            RAISE EXCEPTION 'taxonomy tree node at % missing id', child_path;
        END IF;

        node_name := elem ->> 'name';
        IF node_name IS NULL OR btrim(node_name) = '' THEN
            RAISE EXCEPTION 'taxonomy tree node at % has empty name', child_path;
        END IF;

        translations := elem -> 'translations';
        IF translations IS NULL OR jsonb_typeof(translations) <> 'object' THEN
            translations := '{}'::jsonb;
        END IF;

        en_name := translations #>> '{en,name}';
        IF en_name IS NULL OR btrim(en_name) = '' THEN
            translations := jsonb_set(translations, '{en}', jsonb_build_object('name', node_name), true);
        END IF;

        children := elem -> 'children';
        IF children IS NULL THEN
            children := '[]'::jsonb;
        END IF;
        children := taxonomy_ensure_tree_en_translations(children, child_path || '.children');

        elem := elem || jsonb_build_object('translations', translations, 'children', children);
        result := result || jsonb_build_array(elem);
    END LOOP;

    RETURN result;
END;
$$;

-- Fail fast on corrupt non-array trees before rewriting (do not silently skip).
DO $$
DECLARE
    bad RECORD;
BEGIN
    FOR bad IN
        SELECT id, jsonb_typeof(child_topics) AS kind
        FROM course_topics
        WHERE child_topics IS NOT NULL
          AND jsonb_typeof(child_topics) <> 'array'
    LOOP
        RAISE EXCEPTION
            'course_topics.id=% has non-array child_topics (jsonb_typeof=%)',
            bad.id, bad.kind;
    END LOOP;

    FOR bad IN
        SELECT id, jsonb_typeof(children) AS kind
        FROM course_skills
        WHERE children IS NOT NULL
          AND jsonb_typeof(children) <> 'array'
    LOOP
        RAISE EXCEPTION
            'course_skills.id=% has non-array children (jsonb_typeof=%)',
            bad.id, bad.kind;
    END LOOP;
END;
$$;

UPDATE course_topics
SET child_topics = taxonomy_ensure_tree_en_translations(child_topics, 'course_topics.child_topics')
WHERE child_topics IS NOT NULL;

UPDATE course_skills
SET children = taxonomy_ensure_tree_en_translations(children, 'course_skills.children')
WHERE children IS NOT NULL;

DROP FUNCTION IF EXISTS taxonomy_ensure_tree_en_translations(jsonb, text);
