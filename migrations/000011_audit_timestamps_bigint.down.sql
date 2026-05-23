-- Revert audit columns from BIGINT Unix seconds back to TIMESTAMPTZ.

ALTER TABLE system_privileged_users
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at);

ALTER TABLE system_app_config
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE media_pending_cloud_cleanup
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE media_files
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at),
  ALTER COLUMN deleted_at TYPE TIMESTAMPTZ USING to_timestamp(deleted_at);

ALTER TABLE tags
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE course_skills
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE course_outcomes
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE course_topics
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE course_levels
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE users
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at),
  ALTER COLUMN deleted_at TYPE TIMESTAMPTZ USING to_timestamp(deleted_at);

ALTER TABLE roles
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);

ALTER TABLE permissions
  ALTER COLUMN created_at DROP DEFAULT,
  ALTER COLUMN updated_at DROP DEFAULT,
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING to_timestamp(created_at),
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING to_timestamp(updated_at);
