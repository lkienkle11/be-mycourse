DROP TABLE IF EXISTS instructor_application_skills;
DROP TABLE IF EXISTS instructor_application_topics;

ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience DROP DEFAULT;

ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience TYPE INT USING (
    CASE years_of_experience
        WHEN 'UNDER_1_YEAR' THEN 0
        WHEN 'ONE_TO_TWO_YEARS' THEN 1
        WHEN 'THREE_TO_FIVE_YEARS' THEN 3
        WHEN 'SIX_TO_TEN_YEARS' THEN 6
        WHEN 'OVER_TEN_YEARS' THEN 11
        ELSE 0
    END
);

ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience SET DEFAULT 0;
ALTER TABLE instructor_profiles ALTER COLUMN years_of_experience SET NOT NULL;

ALTER TABLE instructor_profiles
    DROP COLUMN IF EXISTS current_job_title_id,
    DROP COLUMN IF EXISTS current_company_id,
    DROP COLUMN IF EXISTS current_company_domain,
    DROP COLUMN IF EXISTS current_company_description,
    DROP COLUMN IF EXISTS current_company_location;

ALTER TABLE instructor_applications ALTER COLUMN years_of_experience DROP DEFAULT;

ALTER TABLE instructor_applications ALTER COLUMN years_of_experience TYPE INT USING (
    CASE years_of_experience
        WHEN 'UNDER_1_YEAR' THEN 0
        WHEN 'ONE_TO_TWO_YEARS' THEN 1
        WHEN 'THREE_TO_FIVE_YEARS' THEN 3
        WHEN 'SIX_TO_TEN_YEARS' THEN 6
        WHEN 'OVER_TEN_YEARS' THEN 11
        ELSE 0
    END
);

ALTER TABLE instructor_applications ALTER COLUMN years_of_experience SET DEFAULT 0;
ALTER TABLE instructor_applications ALTER COLUMN years_of_experience SET NOT NULL;

ALTER TABLE instructor_applications
    DROP COLUMN IF EXISTS current_job_title_id,
    DROP COLUMN IF EXISTS current_company_id,
    DROP COLUMN IF EXISTS current_company_domain,
    DROP COLUMN IF EXISTS current_company_description,
    DROP COLUMN IF EXISTS current_company_location,
    DROP COLUMN IF EXISTS submitted_at,
    DROP COLUMN IF EXISTS review_due_at,
    DROP COLUMN IF EXISTS returned_at,
    DROP COLUMN IF EXISTS rejection_count,
    DROP COLUMN IF EXISTS rejection_history;

DELETE FROM role_permissions
WHERE permission_id = 'P68';

DELETE FROM permissions
WHERE permission_id = 'P68';
