-- Backfill compatibility for instructor profiles/applications on drifted DBs:
-- 1) ensure deleted_at for soft-delete scope
-- 2) normalize instructor_profiles to canonical id PK (some DBs use user_id PK only)

ALTER TABLE instructor_profiles
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;
ALTER TABLE instructor_applications
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;

ALTER TABLE instructor_profiles
    ADD COLUMN IF NOT EXISTS id BIGINT;

UPDATE instructor_profiles
SET id = user_id
WHERE id IS NULL;

CREATE SEQUENCE IF NOT EXISTS instructor_profiles_id_seq;

SELECT setval(
    'instructor_profiles_id_seq',
    GREATEST(COALESCE((SELECT MAX(id) FROM instructor_profiles), 0), 1)
);

ALTER TABLE instructor_profiles
    ALTER COLUMN id SET DEFAULT nextval('instructor_profiles_id_seq');

UPDATE instructor_profiles
SET id = nextval('instructor_profiles_id_seq')
WHERE id IS NULL;

ALTER TABLE instructor_profiles
    DROP CONSTRAINT IF EXISTS instructor_profiles_pkey;

ALTER TABLE instructor_profiles
    ALTER COLUMN id SET NOT NULL;

ALTER TABLE instructor_profiles
    ADD CONSTRAINT instructor_profiles_pkey PRIMARY KEY (id);

DROP INDEX IF EXISTS uix_instructor_profiles_user_active;
CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_profiles_user_active
    ON instructor_profiles (user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_instructor_profiles_user
    ON instructor_profiles (user_id);

DROP INDEX IF EXISTS uix_instructor_applications_user_active;
CREATE UNIQUE INDEX IF NOT EXISTS uix_instructor_applications_user_active
    ON instructor_applications (user_id)
    WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_instructor_applications_status;
CREATE INDEX IF NOT EXISTS idx_instructor_applications_status
    ON instructor_applications (review_status)
    WHERE deleted_at IS NULL;
