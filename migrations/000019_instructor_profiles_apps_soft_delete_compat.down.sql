DROP INDEX IF EXISTS idx_instructor_applications_status;
CREATE INDEX IF NOT EXISTS idx_instructor_applications_review_status
    ON instructor_applications (review_status);

DROP INDEX IF EXISTS uix_instructor_applications_user_active;

DROP INDEX IF EXISTS idx_instructor_profiles_user;
DROP INDEX IF EXISTS uix_instructor_profiles_user_active;

ALTER TABLE instructor_profiles
    DROP CONSTRAINT IF EXISTS instructor_profiles_pkey;

ALTER TABLE instructor_profiles
    DROP COLUMN IF EXISTS id;

ALTER TABLE instructor_profiles
    ADD CONSTRAINT instructor_profiles_pkey PRIMARY KEY (user_id);

ALTER TABLE instructor_applications DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE instructor_profiles DROP COLUMN IF EXISTS deleted_at;

DROP SEQUENCE IF EXISTS instructor_profiles_id_seq;
