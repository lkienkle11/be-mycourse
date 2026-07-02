-- Instructor application feature: state machine fields, company snapshot columns,
-- application expertise junction tables, years_of_experience enum codes, P68 submit_blocked.

INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P68', 'instructor_application:submit_blocked', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'instructor'
  AND p.permission_id = 'P68'
ON CONFLICT DO NOTHING;

ALTER TABLE instructor_applications
    ADD COLUMN IF NOT EXISTS current_job_title_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS current_company_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS current_company_domain VARCHAR(255),
    ADD COLUMN IF NOT EXISTS current_company_description TEXT,
    ADD COLUMN IF NOT EXISTS current_company_location VARCHAR(255),
    ADD COLUMN IF NOT EXISTS submitted_at BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS review_due_at BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS returned_at BIGINT,
    ADD COLUMN IF NOT EXISTS rejection_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rejection_history JSONB NOT NULL DEFAULT '[]';

ALTER TABLE instructor_applications ALTER COLUMN years_of_experience DROP DEFAULT;

ALTER TABLE instructor_applications ALTER COLUMN years_of_experience TYPE VARCHAR(32) USING (
    CASE
        WHEN years_of_experience <= 0 THEN 'UNDER_1_YEAR'
        WHEN years_of_experience <= 2 THEN 'ONE_TO_TWO_YEARS'
        WHEN years_of_experience <= 5 THEN 'THREE_TO_FIVE_YEARS'
        WHEN years_of_experience <= 10 THEN 'SIX_TO_TEN_YEARS'
        ELSE 'OVER_TEN_YEARS'
    END
);

ALTER TABLE instructor_applications ALTER COLUMN years_of_experience SET DEFAULT 'UNDER_1_YEAR';
ALTER TABLE instructor_applications ALTER COLUMN years_of_experience SET NOT NULL;

UPDATE instructor_applications
SET current_job_title_id = 'custom:' || lower(regexp_replace(trim(current_job_title), '[^a-zA-Z0-9]+', '-', 'g'))
WHERE current_job_title_id IS NULL
  AND trim(current_job_title) <> '';

UPDATE instructor_applications
SET current_job_title_id = 'custom:unknown'
WHERE current_job_title_id IS NULL
   OR trim(current_job_title_id) = '';

ALTER TABLE instructor_applications
    ALTER COLUMN current_job_title_id SET NOT NULL;

ALTER TABLE instructor_profiles
    ADD COLUMN IF NOT EXISTS current_job_title_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS current_company_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS current_company_domain VARCHAR(255),
    ADD COLUMN IF NOT EXISTS current_company_description TEXT,
    ADD COLUMN IF NOT EXISTS current_company_location VARCHAR(255);

ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience DROP DEFAULT;

ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience TYPE VARCHAR(32) USING (
    CASE
        WHEN years_of_experience <= 0 THEN 'UNDER_1_YEAR'
        WHEN years_of_experience <= 2 THEN 'ONE_TO_TWO_YEARS'
        WHEN years_of_experience <= 5 THEN 'THREE_TO_FIVE_YEARS'
        WHEN years_of_experience <= 10 THEN 'SIX_TO_TEN_YEARS'
        ELSE 'OVER_TEN_YEARS'
    END
);

ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience SET DEFAULT 'UNDER_1_YEAR';
ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience SET NOT NULL;

UPDATE instructor_profiles
SET current_job_title_id = 'custom:' || lower(regexp_replace(trim(current_job_title), '[^a-zA-Z0-9]+', '-', 'g'))
WHERE current_job_title_id IS NULL
  AND trim(current_job_title) <> '';

UPDATE instructor_profiles
SET current_job_title_id = 'custom:unknown'
WHERE current_job_title_id IS NULL
   OR trim(current_job_title_id) = '';

ALTER TABLE instructor_profiles
    ALTER COLUMN current_job_title_id SET NOT NULL;

CREATE TABLE instructor_application_topics (
    id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES instructor_applications (id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES course_topics (id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE UNIQUE INDEX uix_instructor_application_topics_app_topic ON instructor_application_topics (application_id, topic_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_application_topics_app ON instructor_application_topics (application_id);

CREATE TABLE instructor_application_skills (
    id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES instructor_applications (id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES course_skills (id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT
);

CREATE UNIQUE INDEX uix_instructor_application_skills_app_skill ON instructor_application_skills (application_id, skill_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instructor_application_skills_app ON instructor_application_skills (application_id);
