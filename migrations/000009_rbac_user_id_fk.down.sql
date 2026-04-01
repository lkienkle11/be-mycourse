ALTER TABLE user_permissions DROP CONSTRAINT IF EXISTS fk_user_permissions_user;
ALTER TABLE user_roles      DROP CONSTRAINT IF EXISTS fk_user_roles_user;

ALTER TABLE user_permissions DROP CONSTRAINT user_permissions_pkey;
ALTER TABLE user_permissions ALTER COLUMN user_id TYPE VARCHAR(128) USING '';
ALTER TABLE user_permissions ADD CONSTRAINT user_permissions_pkey PRIMARY KEY (user_id, permission_id);

ALTER TABLE user_roles DROP CONSTRAINT user_roles_pkey;
ALTER TABLE user_roles ALTER COLUMN user_id TYPE VARCHAR(128) USING '';
ALTER TABLE user_roles ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);
