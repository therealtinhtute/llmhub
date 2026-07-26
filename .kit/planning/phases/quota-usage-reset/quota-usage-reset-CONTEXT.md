# Context: Quota and usage reset controls

Phase: quota-usage-reset
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low (W1) / medium (W2)
Expected Proof: unit, integration, manual-check, command-output

## Goal

Make quota reset reachable from the management panel by wiring the endpoint that
already exists, then add the usage-statistics reset path that does not exist at all.

## Scope Boundary

### Allowed Surfaces

W1 — frontend only:
- `web/src/services/api/quota.ts` (new)
- `web/src/services/api/index.ts` (add one re-export)
- `web/src/components/quota/` — `QuotaCard.tsx`, `QuotaSection.tsx`, `AllQuotaSection.tsx`, `useQuotaLoader.ts`
- `web/src/pages/QuotaPage.tsx`
- `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json`
- `web/src/stores/useQuotaStore.ts` (only if the reset result must invalidate cached quota)

W2 — adds backend:
- `internal/api/handlers/management/` — usage reset handler
- `internal/api/server.go` — one route registration
- the usage counter store reached by `GET /api-key-usage` and `GET /usage-queue`
- Go tests alongside the above
- the same `web/` surfaces as W1, for its UI

### Forbidden Surfaces

- `sdk/cliproxy/auth/conductor.go` and anything under `sdk/cliproxy/auth/` — `Manager.ResetQuota` already exists and is already tested; this phase consumes it, never edits it
- `internal/api/handlers/management/quota.go` `ResetQuota` handler — W1 wires it as-is; changing its contract turns a frontend job into a backend job
- credential concurrency, lifecycle release, token estimation — the two phases merged in PR #2
- provider additions, model routing, translators, executors
- database schema
- installer, Docker, release, branding

## Spec Hooks

- SPEC In Scope — "Quota and usage reset controls in the management panel", W1/W2 split
- SPEC Out of Scope carve-out — "Frontend-only changes without backend parity — **except** `quota-usage-reset` W1"
- SPEC Key Decisions — the YAGNI carve-out; this phase is llmhub-custom, not upstream parity
- SPEC Validation Expectations — web-visible surfaces need an API or browser smoke, not just Go tests

## Locked Decisions

- **The reset-quota endpoint contract is fixed and W1 conforms to it.** Verified in
  `internal/api/handlers/management/quota.go:27-69`:
  - request `POST /v0/management/reset-quota` with body `{"auth_index": "<string>"}`
  - success `200 {"status":"ok","auth_index":"<string>","models":[...]}`
  - failures `400` invalid body or empty `auth_index`, `404` auth not found,
    `500` reset failed, `503` core auth manager unavailable
  W1 must render all four failure cases, not just the happy path.
- **`auth_index` is optional on the client type and must be treated as possibly absent.**
  `web/src/types/authFile.ts:29` declares `authIndex?: string | number | null`. When it
  is null/undefined the reset action is disabled with a distinct reason — never send an
  empty or coerced `auth_index`, because the server answers `400` and the user sees a
  meaningless error.
- **Reset is destructive enough to confirm.** It clears quota/cooldown routing state and
  resumes registry routing for a live credential. Require an explicit confirm step rather
  than firing on first click.
- **After a successful reset the card must re-read, not guess.** The response carries
  `models`, but the card's own quota state is loaded separately; refetch through the
  existing loader rather than patching store state from the response.
- **W2 is a new write path, not a wiring job.** The only usage endpoints today are
  `GET/PUT/PATCH /usage-statistics-enabled`, `GET /api-key-usage`, `GET /usage-queue` —
  all read-or-toggle. W2 owns designing the write path and its idempotency.
- **W1 ships without W2.** They are separate waves specifically so the button is not held
  hostage to the backend work.

## Assumptions

- The panel's own conventions apply, per `web/CLAUDE.md`: one Axios wrapper per API domain
  under `services/api/` re-exported from `index.ts`; every user-visible string goes through
  `useTranslation()` with keys added to **both** `en.json` and `vi.json`; state in Zustand.
- `apiClient` already carries the management key as `Authorization: Bearer` after login,
  so the new service needs no auth handling of its own.
- The existing `onRefresh` path on `QuotaCard` (line 149) is the model to imitate for an
  action that mutates then reloads.
- **Unverified**: whether every provider's quota card can supply `authIndex`. If some
  provider populates it and others do not, W1 covers only the ones that do — see Escalate If.

## Canonical Refs

- `.kit/planning/SPEC.md` — In Scope entry, Out of Scope carve-out, Amendment 2026-07-25
- `.kit/planning/ROADMAP.md` — phase 6, and why it is exempt from the frontend-last ordering
- `internal/api/handlers/management/quota.go:27` — `ResetQuota` handler, the W1 contract
- `internal/api/server.go:643` — `mgmt.POST("/reset-quota", s.mgmt.ResetQuota)`
- `sdk/cliproxy/auth/conductor.go:3409` — `Manager.ResetQuota`, read-only reference
- `web/CLAUDE.md` — panel conventions
- `web/src/components/quota/QuotaCard.tsx:145-153` — the `onRefresh` action precedent
- `web/src/services/api/apiKeyUsage.ts` — the smallest service-wrapper example
- `plans/cliproxyapi-upstream-parity-2026-07-02.md` — "Conflicts With llmhub Custom" explains
  why this endpoint is llmhub-shaped and upstream's file-backed equivalent was not ported

## Rejected Options

- **Reset from the auth-files page instead of the quota cards** — the user reads quota
  state on the quota cards, so that is where the corrective action belongs; a second
  location would split the mental model.
- **Bulk "reset all" control** — larger blast radius on live credentials for no stated
  need. Per-auth only.
- **Patching quota state from the reset response's `models` array** — that array reports
  which models were cleared, not their new quota values; treating it as quota data would
  display numbers that were never measured.
- **Extending the existing `ResetQuota` handler to also reset usage counters** — collapses
  two different blast radii into one endpoint and would make W1 depend on W2.
- **Doing W2 first because it is "more complete"** — W1 delivers user-visible value with an
  already-tested backend; sequencing it behind new backend work is pure delay.

## Deferred Ideas

- Scheduled or automatic quota reset.
- Audit log of who reset which credential and when.
- Surfacing the returned `models` list as a detailed post-reset report.
- Reset controls in the TUI (`internal/tui/`).

## Escalate If

- `authIndex` turns out to be absent for most providers — the phase then needs a backend
  change to expose it, which breaks the W1 frontend-only boundary and the SPEC carve-out
  that justifies it. Route back to `to-plan`.
- W2's design requires a database schema change — schema is a forbidden surface for this
  SPEC. Route back to `brainstorm`.
- Resetting quota is found to have side effects beyond clearing cooldown and resuming
  routing (for example touching persisted credentials). Stop and route to the user.
- A panel smoke cannot be run in the working environment — the SPEC requires one for this
  surface, so the gate would have no valid proof. Say so rather than substituting Go tests.
