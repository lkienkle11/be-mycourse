# Logic Flow Snapshot

## Application Startup
1. Load settings (`pkg/setting`).
2. Initialize primary DB (`models.Setup`).
3. Bind RBAC DB (`services.SetRBACDB`).
4. Optional CLI branch (`internal/appcli`) can short-circuit server startup.
5. Setup Supabase clients.
6. Setup Redis.
7. Optional SQL migration if `MIGRATE=1`.
8. Init system defaults (`config.InitSystem`).
9. Start queue consume placeholder.
10. Build router and serve HTTP.

## Auth Flow Logic
- Register: validate input -> check uniqueness -> create user -> send confirmation email.
- Login: validate credentials -> read/cache checks -> issue access/refresh tokens -> persist session map.
- Confirm: verify token -> transactionally confirm email + ensure learner role -> issue token pair.
- Refresh: validate refresh token + session ownership -> rotate token/session and permissions.

## RBAC Sync Logic
- Permission sync: upsert permissions from constants catalog.
- Role-permission sync: delete and rebuild mapping from constants matrix.
- Triggers: CLI commands, system API "sync-now", or scheduler jobs.

## Enforcement Logic
- Authentication gates run before handlers.
- Authorization decisions prefer JWT claim set; fallback to DB if claims absent.
- Error handling is centralized through middleware and errcode mapping.
