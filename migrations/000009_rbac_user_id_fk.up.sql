-- Change user_roles.user_id and user_permissions.user_id from VARCHAR(128) to BIGINT
-- and add FK constraints referencing users(id).
-- Safe on empty tables (no existing user assignments on a fresh install).

-- user_roles
ALTER TABLE user_roles DROP CONSTRAINT user_roles_pkey;
ALTER TABLE user_roles ALTER COLUMN user_id TYPE BIGINT USING 0;
ALTER TABLE user_roles ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);
ALTER TABLE user_roles
    ADD CONSTRAINT fk_user_roles_user
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

-- user_permissions
ALTER TABLE user_permissions DROP CONSTRAINT user_permissions_pkey;
ALTER TABLE user_permissions ALTER COLUMN user_id TYPE BIGINT USING 0;
ALTER TABLE user_permissions ADD CONSTRAINT user_permissions_pkey PRIMARY KEY (user_id, permission_id);
ALTER TABLE user_permissions
    ADD CONSTRAINT fk_user_permissions_user
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
