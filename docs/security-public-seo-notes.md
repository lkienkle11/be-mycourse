# Security / public SEO notes (BE take-note)

_Last audited: 2026-09-27 — `openspec/changes/add-home-catalog-apis` implemented the course + instructor public catalog (B1, B2, B10 below); see that update per-row. Prior: 2026-07-25 (documentation intent only, no BE code at that time)._

Companion to FE foundation docs:

- [`fe-mycourse/docs/seo-ranking-setup.md`](../../fe-mycourse/docs/seo-ranking-setup.md)
- [`fe-mycourse/docs/security-hardening-notes.md`](../../fe-mycourse/docs/security-hardening-notes.md)

## Intent

Public SEO / storefront read APIs must expose **published-only** data suitable for crawlers and OG/JSON-LD. As of 2026-09-27, `GET /catalog/courses/trending` and `GET /catalog/instructors/popular` implement this for course and instructor listings (no auth). Learner catalogue endpoints (`learner-courses*`) still require authentication by design — only the 2 catalog routes above are public.

Do **not** invent new permissions, rate-limit quotas, or DTO fields in this note. Extend existing assets when implementation is approved.

## Existing BE assets to reuse later (B1–B10)

| # | Asset | Where | Notes for future public SEO |
| --- | --- | --- | --- |
| B1 | Gap: public storefront | [`modules.md`](./modules.md) | **Implemented** — `GET /catalog/courses/trending`, `GET /catalog/instructors/popular` (no auth). Payment/checkout, and price/rating fields, remain planned. |
| B2 | Published list filter | `internal/course/infra/repo_catalog.go`'s `ListTrendingCourses` | **Implemented** — reuses `ListPublishedCourses`'s published/non-trashed filter shape, adds its own `TrendingCourseItem` published-only projection (not `CourseDetail`/`CourseListItem`). |
| B3 | Preview filter | `filterPreviewOutline` | Teaser SEO outline vs full authenticated outline. |
| B4 | Publish path | `ApproveDraft`; `TopicCoursePublished` (not emitted yet) | Still **not implemented** — the new trending-courses/popular-instructors caches (B10) rely on their 5 min TTL only, not a publish-triggered invalidation hook. Future cache invalidation hook when FE registers public cache profiles. |
| B5 | Rate limit | `internal/shared/ratelimit/` + NFR-1.1 | Extend existing tiers for crawler traffic — do not invent a parallel quota system. |
| B6 | Auth / CORS / cookie | `router.go`, `auth_jwt.go`, `csrf.go` | Public GET must stay cookie/Bearer-free for CDN cache; CORS must match FE origin. |
| B7 | Field matrix | [`return_types.md`](./return_types.md) | Future public DTO ≠ full `CourseDetail`; cite learner vs admin field differences. |
| B8 | Slug uniqueness | `ensureUniqueCourseSlug` | Canonical slug future; no public resolve-by-slug yet. |
| B9 | Media visibility | `canViewMediaFile` + `thumbnail_url` | OG images only from published public media. |
| B10 | `/me` cache-aside | auth `service_cache.go` | **Extended** — new generic `internal/shared/cache/json_cache.go` (`GetJSON`/`SetJSON`) follows the same fail-open, TTL-based pattern, used by the trending-courses and popular-instructors catalog services (5 min TTL each). |

## Security expectations for a future public surface

- Published-only; never draft / in-review / rejection / collaborator-only fields.
- No PII, enrollment, progress, payment, or private ticket data in public DTOs.
- Cookie / CSRF / JWT middleware remain authoritative for private routes; robots/`noindex` on FE do not replace them.
- Align CORS allowed origins with the FE site origin (not API URL as “site”).
- TLS / HSTS stay on nginx (see [`deploy.md`](./deploy.md)).

## FE side (already scaffolding, unused)

FE will call future public GETs only through `serverRawFetch` + reviewed `publicCacheProfiles` (currently empty / fail-closed). See FE `seo-ranking-setup.md`.
