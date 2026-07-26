# Plan: Quota and usage reset controls

Phase: quota-usage-reset
Status: ready
Wave Count: 4
Execution Owner: work
Updated At: 2026-07-26

## Goal
Expose quota reset in the management panel by wiring the endpoint that already
exists, then add the usage-statistics reset path that does not exist yet.

## Wave mapping to the SPEC

The SPEC describes two waves, W1 and W2. They expand to four here so W1 has its
own gate and can ship on its own:

- SPEC **W1** → Waves 1–2 (implement, then gate and ship)
- SPEC **W2** → Waves 3–4 (backend, then UI and gate)

**Stop after Wave 2 is a valid outcome.** Waves 3–4 must not be started before
Wave 2's gate is clean.

## Inputs
- `.kit/planning/SPEC.md` — In Scope W1/W2 split, Out of Scope carve-out, Amendment 2026-07-25
- `quota-usage-reset-CONTEXT.md`
- `internal/api/handlers/management/quota.go:27-69` — the W1 endpoint contract
- `web/CLAUDE.md` — panel conventions
- `web/src/services/api/apiKeyUsage.ts` — service-wrapper precedent
- `web/src/components/quota/QuotaCard.tsx:145-153` — action precedent

## Wave 1
### T1 — Wire reset-quota into the quota cards
- type: implementation
- inputs:
  - `internal/api/handlers/management/quota.go` (read-only, contract source)
  - existing quota components and API services
- touches:
  - `web/src/services/api/quota.ts` (new)
  - `web/src/services/api/index.ts`
  - `web/src/components/quota/QuotaCard.tsx`
  - `web/src/components/quota/QuotaSection.tsx`
  - `web/src/components/quota/AllQuotaSection.tsx`
  - `web/src/components/quota/useQuotaLoader.ts`
  - `web/src/pages/QuotaPage.tsx`
  - `web/src/i18n/locales/en.json`
  - `web/src/i18n/locales/vi.json`
- avoid:
  - any Go file
  - `sdk/cliproxy/auth/`
  - the `ResetQuota` handler and its route
- steps:
  1. Add `quotaApi.resetQuota(authIndex: string)` posting
     `{"auth_index": authIndex}` to `/reset-quota` via `apiClient`, mirroring
     `apiKeyUsage.ts`; re-export it from `services/api/index.ts`.
  2. Add an optional `onResetQuota` action to `QuotaCard` beside the existing
     `onRefresh`, disabled with its own tooltip when `item.authIndex` is
     null/undefined — never send a coerced or empty `auth_index`.
  3. Gate the call behind an explicit confirm step; reset mutates live routing state.
  4. On success, refetch that card's quota through the existing loader rather than
     patching store state from the response's `models` array.
  5. Surface each documented failure distinctly: `400`, `404`, `500`, `503`.
  6. Add every new string to **both** `en.json` and `vi.json`.
- expected outputs:
  - a per-auth quota reset action in the panel, backed by the existing endpoint
- verification:
  - `cd web && bun run type-check && bun run lint`
  - `grep -rn 'reset-quota' web/src` returns the new service (it returns nothing today)
- stop if:
  - `authIndex` proves unavailable for most providers — that needs a backend change
    and breaks the frontend-only boundary the SPEC carve-out rests on
- escalate to:
  - to-plan phase

## Wave 2
### T2 — Gate and ship W1
- type: test
- inputs:
  - Wave 1
- touches:
  - phase run and validation evidence only
- avoid:
  - new source changes beyond fixing what the gate finds
- steps:
  1. `make build-web` then `make build`.
  2. Run the panel smoke the SPEC requires: start the server, open the quota page,
     trigger reset on one auth, and prove from server logs or the network trace that
     `POST /v0/management/reset-quota` was called with a stable `auth_index` and that
     the card reflected the result.
  3. Record the exact commands, exit codes, and smoke evidence.
  4. If no panel smoke can be run in this environment, say so explicitly — the gate
     then has no valid proof for this surface. Do not substitute Go tests.
- expected outputs:
  - clean W1 gate; W1 is shippable on its own
- verification:
  - `make build-web && make build && git diff --check`
  - smoke evidence attached to the run record
- stop if:
  - a gate fails twice after one targeted correction
- escalate to:
  - check

## Wave 3
### T3 — Add the usage-statistics reset endpoint
- type: implementation
- inputs:
  - the existing usage read paths: `GET /api-key-usage`, `GET /usage-queue`,
    `GET/PUT/PATCH /usage-statistics-enabled`
  - the usage counter store behind them
- touches:
  - `internal/api/handlers/management/` — new usage-reset handler and its test
  - `internal/api/server.go` — one route registration
  - the usage counter store implementation
- avoid:
  - database schema changes
  - the quota reset path from W1
  - auth, routing, providers, translators
- steps:
  1. Trace what actually backs the usage counters before designing the write path.
  2. Add the reset endpoint; make it idempotent and explicit about what it clears.
  3. Cover it with unit tests and an integration test, including a second call
     returning the same result.
- expected outputs:
  - a tested usage-statistics reset endpoint
- verification:
  - `go test ./internal/api/... && go vet ./internal/api/...`
- stop if:
  - a schema change is required — schema is a forbidden surface for this SPEC
- escalate to:
  - brainstorm

## Wave 4
### T4 — Surface usage reset in the panel and gate the phase
- type: implementation
- inputs:
  - Wave 3
- touches:
  - `web/src/services/api/` — usage reset wrapper
  - the usage/statistics page and its components
  - `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json`
- avoid:
  - reworking W1's shipped UI
- steps:
  1. Wire the new endpoint the same way W1 wired reset-quota: confirm step,
     distinct error states, refetch on success, both locale files.
  2. Run the full gate.
- expected outputs:
  - clean quota-usage-reset phase gate
- verification:
  - `go test ./... && go vet ./... && go build ./...`
  - `cd web && bun run type-check && bun run lint`
  - `make build-web && make build && git diff --check`
  - panel smoke for the usage reset action, same discipline as Wave 2
- stop if:
  - a gate fails twice after one targeted correction
- escalate to:
  - check

## Risks / Watch-fors
- `AuthFileItem.authIndex` is `?: string | number | null`. Coercing it produces a
  `400` the user cannot interpret; the action must be disabled instead.
- The reset response's `models` array reports which models were cleared, not their
  new quota values — it is not quota data.
- Reset acts on a live credential's routing state. No bulk reset, and no firing on
  first click.
- i18n keys added to only `en.json` will render as raw keys in Vietnamese.
- The SPEC treats Go tests alone as insufficient proof for web-visible surfaces.
  A missing smoke is a missing gate, not a passing one.
- W2 touches the usage store and queue; that risk must not leak backwards into W1.
