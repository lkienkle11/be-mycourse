-- granted_by_user_id IS NULL scopes this to rows the up migration actually created: every
-- app-created binding (RoleBindingService.Assign) always records the issuing principal there,
-- so a plain role/resource_type match would also delete real collaborator access added after
-- the backfill ran.
DELETE FROM authorization_role_bindings WHERE resource_type = 'course' AND role_name = 'EDITOR' AND granted_by_user_id IS NULL;
