---
id: 01KY1T7GGY3DBEW2NAAV620VCK
type: spec
phase: websocket-message-too-big
lane: high-risk
intake_id: 01KY1T9KV4F9NKFP1APTK2M7ZG
created: 2026-07-23
updated: 2026-07-25
---

# SPEC: Implement all post-v7.2.93 updates from CLIProxyAPI

Status: locked
Input Type: new-initiative
Lane: high-risk
Risk Flags: external-systems, public-contract, existing-behavior, multi-domain
Affected Surfaces: api, provider, db, docs, frontend

## Source Mode
files

## Source Inputs
- All changes between CLIProxyAPI v7.2.93 and v7.2.96 (credential concurrency, token handling, model routing, docs, etc.)

## Goal
Improve and implement **all** updates that happened in upstream after v7.2.93 into llmhub, covering both Go backend and React frontend, while preserving existing behavior.

## In Scope
- Credential concurrency (Home/general)
- Token estimation/state handling and perf improvements
- Model routing updates (Codex Alpha Search + new models)
- Translator/executor behavior fixes (empty responses, output indexing, MIME types, WebSocket 1009)
- Docs and management panel UI updates
- Maintain existing provider behavior (no breaking changes)
- Quota and usage reset controls in the management panel — **llmhub-custom, not
  upstream parity** (see Amendment 2026-07-25). Two waves:
  - W1: wire the existing `POST /v0/management/reset-quota` to a panel action
    (frontend only; backend already shipped in `US-015`)
  - W2: usage-statistics reset — new backend endpoint plus UI (no endpoint exists today)

## Out of Scope
- Unrelated new features
- Full upstream merge beyond listed gaps
- Frontend-only changes without backend parity — **except** `quota-usage-reset` W1,
  which wires an endpoint that already exists and is already tested (Amendment 2026-07-25)

## Key Decisions
- Full post-v7.2.93 coverage (not targeted slice)
- Phased implementation (auth/credential first, then executor/translator, then frontend)
- YAGNI: only implement what is in the upstream diff list
- **YAGNI carve-out (2026-07-25)**: `quota-usage-reset` is deliberately outside the
  upstream diff list. It is the only approved exception; do not treat it as licence
  to widen scope elsewhere.

## Amendment — 2026-07-25

This SPEC was `Status: locked` when amended. Recorded rather than rewritten so the
original intent stays auditable.

**Added**: phase `quota-usage-reset` (ROADMAP phase 6).

**Why it is not upstream parity**: `POST /v0/management/reset-quota` and
`Manager.ResetQuota` are llmhub-custom, added in `US-015` and designed against
llmhub's Postgres-owned runtime — upstream's equivalent is file-backed and was
deliberately not ported (`plans/cliproxyapi-upstream-parity-2026-07-02.md`,
"Conflicts With llmhub Custom"). Wiring it to the panel therefore closes an
llmhub gap, not an upstream one.

**Gap this closes**: the endpoint has been live since `US-015` with **no way to
invoke it from the panel**. Verified 2026-07-25 — `grep -rn 'reset-quota\|resetQuota' web/src`
returns nothing; there is no `web/src/services/api/quota.ts` among the 16 API
service files; every `reset*` symbol in `web/src` is a read-only display field in
`web/src/types/quota.ts` (`resetTime`, `reset_at`, `resetLabel`, `resets_at`); and
`QuotaCard.tsx:149` exposes only `onRefresh`.

**Usage reset is a genuinely new feature**, not a wiring job: the only usage
endpoints are `GET/PUT/PATCH /usage-statistics-enabled`, `GET /api-key-usage`, and
`GET /usage-queue` — all read-or-toggle. W2 must add the write path, and will touch
the usage store / Redis queue, so it carries real risk that W1 does not. Ship W1
independently; do not let W2 block it.

**Two prior claims in this session's records were wrong and are corrected here**:
this SPEC was never "fully covered" (only 2 of 5 ROADMAP phases were done), and
`.kit/planning/ROADMAP.md` was never stale — it is this SPEC's real plan.

## Deferred Ideas
- End-to-end integration tests for new token logic
- Backend migration script for credential changes

## Validation Expectations
- `go test ./...` and `make build-web` + `make build` pass
- Parity report updated with new evidence
- Frontend UI matches new model display/name changes
- No regression in existing providers
- `quota-usage-reset`: because this is a web-visible surface, static Go tests are
  **not** sufficient proof — the parity plan's Verification Matrix requires an
  API or browser smoke for the changed endpoint/page. Each wave needs:
  - W1 — `bun run type-check` + a panel smoke proving the action calls
    `POST /v0/management/reset-quota` with a stable `auth_index` and reflects the result
  - W2 — Go unit + integration tests for the new usage-reset endpoint, plus the same
    smoke discipline for its UI

## Next Step
Run `/to-plan` after approval.
