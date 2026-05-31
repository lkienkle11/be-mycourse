ALTER TABLE system_privileged_users
  ADD COLUMN machine_secret TEXT NOT NULL DEFAULT '';
