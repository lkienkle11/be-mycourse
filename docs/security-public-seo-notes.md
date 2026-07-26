# Security / public SEO notes (BE take-note)

_Last audited: 2026-07-25. **No BE code** in this phase — documentation intent only._

Companion to FE foundation docs:

- [`fe-mycourse/docs/seo-ranking-setup.md`](../../fe-mycourse/docs/seo-ranking-setup.md)
- [`fe-mycourse/docs/security-hardening-notes.md`](../../fe-mycourse/docs/security-hardening-notes.md)

## Intent

Future public SEO / storefront read APIs (if product approves them) must expose **published-only** data suitable for crawlers and OG/JSON-LD. Today there is **no** anonymous public course catalogue. Learner catalogue endpoints require authentication.

Do **not** invent new permissions, rate-limit quotas, or DTO fields in this note. Extend existing assets when implementation is approved.

## Existing BE assets to reuse later (B1–B10)

| # | Asset | Where | Notes for future public SEO |
| --- | --- | --- | --- |
| B1 | Gap: public storefront | [`modules.md`](./modules.md) Planned | No anonymous storefront routes yet. |
| B2 | Published list filter | `ListPublishedCourses` in learner course repo | Future public list DTO should reuse published semantics, not draft/admin shapes. |
| B3 | Preview filter | `filterPreviewOutline` | Teaser SEO outline vs full authenticated outline. |
| B4 | Publish path | `ApproveDraft`; `TopicCoursePublished` (not emitted yet) | Future cache invalidation hook when FE registers public cache profiles. |
| B5 | Rate limit | `internal/shared/ratelimit/` + NFR-1.1 | Extend existing tiers for crawler traffic — do not invent a parallel quota system. |
| B6 | Auth / CORS / cookie | `router.go`, `auth_jwt.go`, `csrf.go` | Public GET must stay cookie/Bearer-free for CDN cache; CORS must match FE origin. |
| B7 | Field matrix | [`return_types.md`](./return_types.md) | Future public DTO ≠ full `CourseDetail`; cite learner vs admin field differences. |
| B8 | Slug uniqueness | `ensureUniqueCourseSlug` | Canonical slug future; no public resolve-by-slug yet. |
| B9 | Media visibility | `canViewMediaFile` + `thumbnail_url` | OG images only from published public media. |
| B10 | `/me` cache-aside | auth `service_cache.go` | Pattern reference for a future catalogue cache — not implemented for courses. |

## Security expectations for a future public surface

- Published-only; never draft / in-review / rejection / collaborator-only fields.
- No PII, enrollment, progress, payment, or private ticket data in public DTOs.
- Cookie / CSRF / JWT middleware remain authoritative for private routes; robots/`noindex` on FE do not replace them.
- Align CORS allowed origins with the FE site origin (not API URL as “site”).
- TLS / HSTS stay on nginx (see [`deploy.md`](./deploy.md)).

## FE side (already scaffolding, unused)

FE will call future public GETs only through `serverRawFetch` + reviewed `publicCacheProfiles` (currently empty / fail-closed). See FE `seo-ranking-setup.md`.
