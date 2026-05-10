ALTER TABLE users
ADD COLUMN IF NOT EXISTS registration_email_send_total INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.registration_email_send_total IS 'Successful registration confirmation emails while pending; not exposed in public JSON; reset on email confirm.';
