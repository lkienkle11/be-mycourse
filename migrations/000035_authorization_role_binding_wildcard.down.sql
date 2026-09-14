ALTER TABLE authorization_role_bindings
    DROP CONSTRAINT IF EXISTS chk_authorization_role_bindings_resource_id;
