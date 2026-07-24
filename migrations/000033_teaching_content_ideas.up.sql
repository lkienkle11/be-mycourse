-- Add teaching_content_ideas to instructor applications and profiles
-- (required 50-500 Unicode code points on submit, not UTF-8 bytes. Approve copies via profile snapshot.)

ALTER TABLE instructor_applications
    ADD COLUMN IF NOT EXISTS teaching_content_ideas TEXT NOT NULL DEFAULT '';

ALTER TABLE instructor_profiles
    ADD COLUMN IF NOT EXISTS teaching_content_ideas TEXT NOT NULL DEFAULT '';
