-- authorization_role_bindings.resource_id already accepts the reserved sentinel value '*'
-- (domain.WildcardResourceID), meaning "every resource of this binding's resource_type", read
-- by GormGrantRepository.roleExpandedGrants. This constraint makes that convention
-- database-enforced instead of relying only on application-level discipline: every resource_id
-- must be either the literal '*' or a UUID-shaped string, matching how every other resource ID
-- in this schema is a UUID.
ALTER TABLE authorization_role_bindings
    ADD CONSTRAINT chk_authorization_role_bindings_resource_id
    CHECK (
        resource_id = '*'
        OR resource_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
    );
