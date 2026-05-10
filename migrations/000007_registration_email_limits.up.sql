ALTER TABLE users
ADD COLUMN IF NOT EXISTS registration_email_send_total INTEGER NOT NULL DEFAULT 0;

-- See migrations/README.md: embedded migrate splits on each semicolon — avoid semicolons inside COMMENT text.
COMMENT ON COLUMN users.registration_email_send_total IS 'Successful registration confirmation emails while pending. Not exposed in public JSON. Reset on email confirm.';
