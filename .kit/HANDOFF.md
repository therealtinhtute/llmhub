---
id: 01KYEMS1CE457HW3GRPEXDGWPF
type: handoff
phase: quota-usage-reset
lane: normal
run_id: none
check_id: none
created: 2026-07-26
updated: 2026-07-26
session-date: 2026-07-26
branch: feat/quota-usage-reset
status: planned
continuity-mode: full-harness
active-phase: quota-usage-reset
last-updated: 2026-07-26 07:40
---

# Session Handoff — feat/quota-usage-reset

## Current State

**Branch**: `feat/quota-usage-reset`, created off `master` at `566d5e7`, **not pushed**, no upstream set
**Status**: planned — CONTEXT + PLAN written and committed; zero implementation
**Continuity Mode**: full-harness (ROADMAP + phase CONTEXT/PLAN + harness DB all present; this phase has no RUN and no CHECK yet, which is correct for a planning session)
**Active Phase**: `quota-usage-reset` (ROADMAP phase 6)
**Last Commit**: `4e9976d` — `chore(planning): amend SPEC and plan quota-usage-reset phase`
**Working Tree**: clean at time of writing except this file and the `Lane:` line added to the phase CONTEXT

`master` is at `566d5e7` and in sync with `origin/master`. The previous phase's
merged branch is gone locally and on the remote.

## What We're Building

`POST /v0/management/reset-quota` has been live since `US-015` with **no way to
invoke it from the management panel**. This phase closes that gap, then adds the
usage-statistics reset path that does not exist at all.

It is **llmhub-custom, not upstream parity** — `Manager.ResetQuota` was written
against llmhub's Postgres-owned runtime, and upstream's file-backed equivalent was
deliberately not ported. It is the **single approved carve-out** from this SPEC's
YAGNI decision; do not treat it as licence to widen scope elsewhere.

## Continuity Anchors

**Latest Cook Run**: none for this phase. The most recent run in the repo is
`.kit/runs/work/20260725-1210-token-estimation.md` (ULID `01KYCVPE9326H17R0KV7DHC2Z4`) and it belongs to the previous, already-merged phase.
**Latest Check Verdict**: none for this phase. Previous phase's verdict was
approve-with-requests (ULID `01KYCVQEX866578AD3EZAHY6FK`).
**Story**: `01KYEMC12NZ7SKJ6M99KQCCFT0` (`quota-usage-reset`, status `planned`)
**Active Context**: `.kit/planning/phases/quota-usage-reset/quota-usage-reset-CONTEXT.md`
**Active Plan**: `.kit/planning/phases/quota-usage-reset/quota-usage-reset-PLAN.md`

**Proof / Drift Notes**:
- `zharness resume --json` → `readiness: in-progress`, `drift: []`
- `zharness validate --json` → 3 findings, **all pre-existing zharness tool gaps**, none introduced by this session's artifacts (see Blockers)
- `zharness query phases` returns **3 stories against 6 ROADMAP phases** — `model-routing`, `websocket-pluginhost`, and `docs-frontend` have never been storied. Not an error; those phases were never planned.
- Nothing in this phase has been built or tested. There is no gate evidence because there is nothing yet to gate.

## Progress This Session

### Completed ✓
- Cleaned the base: `master` fast-forwarded to `566d5e7`, merged branch `feat/credential-concurrency-token-estimation` deleted locally and on the remote.
- Amended the locked SPEC to add `quota-usage-reset` — recorded as an `## Amendment — 2026-07-25` section rather than a rewrite, with the YAGNI carve-out, the grep evidence for the gap, and corrections to two wrong prior claims.
- Rewrote `.kit/planning/ROADMAP.md`'s phase list to carry per-phase status + evidence (5 → 6 phases), and added a section explaining that the 7 directories under `phases/` belong to **two** initiatives.
- Created story `01KYEMC12NZ7SKJ6M99KQCCFT0` and pointed `current_phase` at it.
- Wrote the phase CONTEXT (scope boundary, 6 locked decisions, 5 rejected options, 4 escalate triggers) and the 4-wave PLAN.
- Corrected the previous handoff entity's `open_items` in the harness DB — see Key Decisions #2.
- Committed all of the above as `4e9976d` (11 files, +597/−11).

### In Progress ⏳
- Nothing mid-flight. No code has been touched.

### Not Started
- **Every wave of this phase.** Wave 1 has not begun.
- The 3 unplanned ROADMAP phases (`model-routing`, `websocket-pluginhost`, `docs-frontend`).
- The two code follow-ups carried from the previous phase (`closeErr`, typeless content blocks).
- Advancing the stale `Status: ready` field on the 7 older PLAN files.
- Updating `plans/cliproxyapi-upstream-parity-2026-07-02.md` with `US-016` / PR #2 evidence.

## Key Decisions

1. **The SPEC's 2 waves expand to 4 in the PLAN, specifically so W1 can ship alone.**
   SPEC W1 → PLAN Waves 1–2 (implement, then gate and ship). SPEC W2 → Waves 3–4.
   **Stopping after Wave 2 is a valid, complete outcome.** Waves 3–4 must not start
   before Wave 2's gate is clean.
2. **The previous handoff entity carried three claims I had already disproved**, most
   dangerously "ROADMAP.md is stale, rewrite it to match the phases/ directory". That
   instruction would have erased the record of 3 phases never started, and `resume`
   feeds `open_items` straight to the next session. Corrected in the DB via changeset
   `01KYEMQ6KJ2YGNX4QTEN50PS2Q` rather than left for someone to act on.
3. **`entry_phase` was set to `quota-usage-reset` by mistake** in changeset
   `01KYEMCA59A2YFW725KEHXW5NK` and restored to `auth-credential-concurrency` by
   `01KYEMMHVZNBP0RMXWEEXJFBQ3`. DB and `workflow-state.yml` now agree.
4. **Lane is `normal`, not the SPEC's `high-risk`.** W1 touches no auth, credential,
   routing, or executor path — it is a frontend call to an endpoint that already
   exists and is already tested. W2 raises this to medium blast radius but still
   does not enter the high-risk surfaces.
5. **The existing `ResetQuota` handler is a forbidden surface.** W1 conforms to its
   contract as-is. Changing it would turn a frontend job into a backend job and
   invalidate the Out-of-Scope carve-out that permits W1 to be frontend-only.
6. **No bulk "reset all".** Per-auth only — reset acts on a live credential's routing
   state, and a bulk control multiplies that blast radius for no stated need.
7. **Planning artifacts were committed on a branch, not on `master`**, per the standing
   rule against committing directly to the default branch. They reach `master` when
   this phase's PR merges.

## Blockers & Issues

No blockers on starting Wave 1. One unknown that could invalidate the plan, plus
carried-over items:

### BLOCKED_CONTEXT (potential) — `authIndex` may not be populated for every provider
- **Issue**: `web/src/types/authFile.ts:29` declares `authIndex?: string | number | null`
  — optional **and** nullable. The endpoint requires a non-empty `auth_index` and
  answers `400` otherwise. If most providers do not populate it, W1 cannot stay
  frontend-only, and the SPEC carve-out that justifies W1 collapses.
- **Needed**: verify which providers populate `authIndex` before writing code.
- **Next**: this is Wave 1's first real question. If it fails, route back to `to-plan` —
  do not quietly add a backend change to a frontend-only wave.

### BLOCKED_VERIFICATION (anticipated) — the gate needs a panel smoke, not Go tests
- **Issue**: the SPEC states that for this web-visible surface, static Go tests are
  **not** sufficient proof; an API or browser smoke is required.
- **Needed**: a runnable panel against a server with at least one auth credential.
- **Next**: if that cannot be arranged in the working environment, say so explicitly.
  A missing smoke is a missing gate, not a passing one. Do not substitute Go tests.

### 5 harness contract violations — all tool gaps, none an artifact error
- `SPEC->PLAN: not_yet_implemented` — PLAN artifacts carry no `spec_id` field; zharness
  self-reports this.
- `plan_id "none" is not a valid ULID` ×2 — `.kit/harness.db` has no `plans` table and
  no `plan create` command.
- **New this session**: `CHECK->HANDOFF: run_id "none" is not a valid ULID` and the same
  for `check_id`. The handoff playbook explicitly permits `none` when the phase has no
  run or check yet — which is exactly this phase's state — but `validate` rejects it.
  The validator and the playbook disagree; the playbook is right.
- **Next**: report upstream. Do **not** mint local ULIDs and do **not** point `run_id`
  at the previous phase's run to silence the validator — those artifacts belong to
  `token-estimation`, and a green validate bought with a false cross-link is worse
  than a visible gap.

### Two code follow-ups from the previous phase, still open
- `sdk/cliproxy/executionregistry/registry.go:50` — `closeErr` is declared and read at
  406/437 but never assigned, so `Close()` always returns `nil`. Either assign it or
  drop the field; the misleading contract is the whole problem.
- `internal/runtime/executor/helps/claude_input_tokens.go:199` — the `case "":` fallback
  serializes typeless content blocks whole, bypassing the media-exclusion branch at 197.
  Defensive only; the Claude API always supplies `type`.

## Technical Context

**The W1 contract, verified in `internal/api/handlers/management/quota.go:27-69`:**
- `POST /v0/management/reset-quota`, body `{"auth_index": "<string>"}`
- `200 {"status":"ok","auth_index":"<string>","models":[...]}`
- `400` invalid body or empty `auth_index` · `404` auth not found · `500` reset failed
  · `503` core auth manager unavailable

All four failure cases must be rendered distinctly. The `models` array reports which
models were **cleared** — it is not quota data, and patching store state from it would
display numbers that were never measured. Refetch through the existing loader instead.

**Key Files**:
- `internal/api/handlers/management/quota.go:27` — the handler (read-only for this phase)
- `internal/api/server.go:643` — `mgmt.POST("/reset-quota", s.mgmt.ResetQuota)`
- `sdk/cliproxy/auth/conductor.go:3409` — `Manager.ResetQuota` (forbidden surface)
- `web/src/components/quota/QuotaCard.tsx:145-153` — the `onRefresh` action precedent;
  today this is the card's **only** action
- `web/src/services/api/apiKeyUsage.ts` — smallest service-wrapper example to copy
- `web/src/services/api/index.ts` — 15 `export *` lines; the new `quota.ts` goes here
- `web/src/stores/useQuotaStore.ts` — per-provider `Record<string, XQuotaState>` + `clearQuotaCache()`
- `web/src/components/quota/` — `AllQuotaSection.tsx`, `QuotaSection.tsx`, `useQuotaLoader.ts`,
  `quotaConfigs.ts`, `quotaStyles.ts`, `useGridColumns.ts`, `index.ts`

**Panel conventions** (`web/CLAUDE.md`): one Axios wrapper per API domain under
`services/api/` re-exported from `index.ts`; hash routing; `@/` aliases `src/`; Zustand
stores; **every user-visible string needs a key in BOTH `en.json` and `vi.json`** — keys
added only to `en.json` render as raw keys in Vietnamese.

**Commands**: `cd web && bun run type-check && bun run lint` · `make build-web` ·
`make build` · `bun dev` for hot reload against a server on `:9090`.

**Configuration**:
- `GOCACHE=/private/tmp/llmhub-go-cache` was used for gate commands in the prior session
- `LLMHUB_POSTGRES_TEST_DSN` unset → 2 `internal/store` tests skip
- `staticcheck` not installed

## Next Steps

1. **→ START HERE: verify `authIndex` availability, then implement PLAN Wave 1 / T1.**
   First check which providers populate `AuthFileItem.authIndex` (`web/src/types/authFile.ts:29`
   declares it optional and nullable). If it holds, add `web/src/services/api/quota.ts`
   with `quotaApi.resetQuota(authIndex)` posting `{"auth_index": authIndex}`, export it
   from `services/api/index.ts`, add an `onResetQuota` action to `QuotaCard` beside
   `onRefresh` — disabled with its own tooltip when `authIndex` is absent, behind a
   confirm step, refetching via the loader on success — and add every string to both
   locale files. Expected: `cd web && bun run type-check && bun run lint` clean, and
   `grep -rn 'reset-quota' web/src` returning the new service (it returns nothing today).
2. **Gate and ship W1 (PLAN Wave 2)** — `make build-web && make build && git diff --check`,
   plus the panel smoke proving the action calls the endpoint with a stable `auth_index`.
   Then open the PR. **Stopping here is a complete outcome.**
3. **Only after Wave 2 is green: PLAN Waves 3–4** — the usage-statistics reset endpoint.
   Trace what actually backs the usage counters before designing the write path; today
   the only usage endpoints are `GET/PUT/PATCH /usage-statistics-enabled`,
   `GET /api-key-usage`, `GET /usage-queue`, all read-or-toggle.
4. **Plan the 3 remaining ROADMAP phases** — the SPEC is not fully covered. `model-routing`
   is next by the dependency rule (executor/translator tier), then `docs-frontend`.
   `websocket-pluginhost` needs a scope decision first: its pluginhost half is a
   separately-gated high-risk initiative deferred as Slice 5 in the parity plan.
5. **Close the two code follow-ups** — `closeErr` first (a misleading API contract), then
   the typeless-content-block hardening.

## Notes

- **Do not rewrite `.kit/planning/ROADMAP.md` to match `ls .kit/planning/phases/`.** The
  7 directories belong to two initiatives, distinguishable by PLAN `Updated At:` —
  `2026-07-23` is this SPEC (`auth-credential-concurrency`, `token-estimation`),
  `2026-07-22` is the older `US-016` SPEC whose code already shipped in `e4bf0f7`.
  Rewriting would erase the record of the 3 phases never started.
- The 5 `US-016` phases read `Status: ready` but their code is shipped. **Do not
  re-implement them**; only the status field is stale.
- ULID alphabet is Crockford base32, 26 chars, `0123456789ABCDEFGHJKMNPQRSTVWXYZ` —
  **no I, L, O, or U**. Generate them with `zharness id`, don't type them.
- **The DB is the source of truth for ULIDs, not the files.** `zharness run create` mints
  its own, which have differed from hand-assigned ones. `audit` stays green while file
  and DB diverge — a quiet trap. Always take the ULID the CLI returns.
- `zharness resume --facts` rejects certain phrases in `facts.risks[].action` (`"git status"`
  was rejected). Reword rather than fight it.
- Prior context that still holds: PR #1 (`feat/model-display-name-propagation`) was closed
  without merge; `origin/feat/model-display-name-propagation` is still visible in
  `git log --decorate`.
- Previous handoff (PR #2, phase `token-estimation`, entity `01KYCYCA4CY9FJE3TKVHDTH4VM`)
  is preserved in git history at commit `4e9976d` if its detail is needed.

---

*Generated by handoff on 2026-07-26 07:40*
