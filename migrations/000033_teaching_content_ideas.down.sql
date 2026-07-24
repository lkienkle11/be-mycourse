ALTER TABLE instructor_applications
    DROP COLUMN IF EXISTS teaching_content_ideas;

ALTER TABLE instructor_profiles
    DROP COLUMN IF EXISTS teaching_content_ideas;
