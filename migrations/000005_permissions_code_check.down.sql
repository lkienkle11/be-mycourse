DROP INDEX IF EXISTS uix_permissions_code_check;

ALTER TABLE permissions DROP COLUMN IF EXISTS code_check;
