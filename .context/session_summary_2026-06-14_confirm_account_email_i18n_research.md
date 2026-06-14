# Session Summary — Discovery: confirm_account email i18n

> Saved: 2026-06-14
> Project: be-mycourse + fe-mycourse
> Phase: **Discovery only** (pre-implementation)

## Problem

Register confirmation email is always Vietnamese even when FE shows English. Root cause: hardcoded VI in `template/html/email/confirm_account.html` and `brevo/client.go` subject. `locale` already flows to confirm URL but not to email rendering.

## GitNexus research

| Symbol | Risk | d=1 callers to update |
|--------|------|----------------------|
| `RenderConfirmAccount` | LOW | `SendConfirmationEmail` |
| `SendConfirmationEmail` | LOW | `sendRegistrationEmail` |
| `sendRegistrationEmail` | LOW | internal only (`registerNewPending`, `registerResendPending`) |
| `normalizeRegisterLocale` | LOW | `Handler.Register` |

Index was stale (~29 commits); re-analyze required at close-out.

## Symbols reuse

- `RegisterRequest`, `Handler.Register`, `AuthService.Register`, `sendRegistrationEmail` rate-limit logic — **reuse**
- `normalizeRegisterLocale` — **reuse** (unchanged; feeds email i18n + confirm URL)

## Symbols change

- `template/languages/confirm_account/en.js`, `vi.js` — **new**
- `mailtmpl/i18n.go` — **new** (load JS, interpolate `{{key}}`)
- `mailtmpl/render.go` — accept `languageCode`, build localized `ConfirmAccountData`
- `confirm_account.html` — template variables only (no hardcoded copy)
- `brevo.SendConfirmationEmail` — add `languageCode` param, localized subject
- `sendRegistrationEmail` — pass locale/language to brevo
- `RegisterRequest` — **reuse** `locale` only (no `language_code`)

## FE scope

- `RegisterPayload`: send `locale` from `useLocale()` in signup + resend flows
- No other auth logic changes

## Docs gap

- `docs/modules/auth.md`, `docs/curl_api.md` — locale only documents link, not email language
- `docs/sequence_diagrams.md` — Register missing locale param
- FE `docs/flow.md`, `docs/api-overview.md`, `docs/screens.md` — still reference `locale` in register payload

## Files to touch (implementation)

**BE:** `template/languages/confirm_account/*.js`, `template/html/email/confirm_account.html`, `internal/shared/mailtmpl/*`, `internal/shared/brevo/client.go`, `internal/auth/application/service.go`, `internal/auth/delivery/dto.go`, `locale.go`, `handler.go`, docs, Postman artifact

**FE:** `src/api/callers/auth/auth.ts`, `src/actions/auth/auth-client.ts`, `login-content.tsx`, docs

## Quality gates (close-out)

BE: golangci-lint, check-architecture, check-dupl, check-layout, go test, go build, Postman regen, gitnexus analyze + detect_changes

FE: lint:biome, lint, build, quality:deps, gitnexus analyze + detect_changes
