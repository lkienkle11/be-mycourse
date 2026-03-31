-- Stable human/API key stays in `code`; fast-moving check strings for middleware live in `code_check` (also unique).

ALTER TABLE permissions ADD COLUMN IF NOT EXISTS code_check VARCHAR(128);

UPDATE permissions SET code_check = code WHERE code_check IS NULL OR BTRIM(code_check) = '';

ALTER TABLE permissions ALTER COLUMN code_check SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uix_permissions_code_check ON permissions (code_check);
