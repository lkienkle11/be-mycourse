# Session — course admin granular permissions (2026-06-17)

## Done
- Migration `000024_course_admin_permissions`: P59–P66
- `permissions.go`, `roles_permission.go`, `routes.go` — granular guards (not `admin:modify`)
- Docs: `database.md`, `router.md`, `modules/course.md`, `curl_api.md`, `requirements.md`, `data-flow.md`

## Permissions
| ID | Name | Route |
|----|------|-------|
| P59 | course_review:read | GET pending |
| P60 | course_review:approve | POST approve |
| P61 | course_review:reject | POST reject |
| P62 | course_catalog:read | GET courses |
| P63 | course_catalog:trash | POST trash |
| P64 | course_trash:read | GET trash |
| P65 | course_trash:restore | POST restore |
| P66 | course_trash:delete | DELETE permanent |

## Deploy note
Run `MIGRATE=1` (or apply `000024`) then re-login so JWT includes P59–P66.

## Quality
- `make test-all` PASS
