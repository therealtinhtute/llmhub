# ROADMAP: All post-v7.2.93 updates (full sync)

Entry Phase: `websocket-message-too-big`
Execution Mode: full

## Phases
- **auth-credential-concurrency** — Home + general credential concurrency — ✅ **done** (PR #2, gate `01KYCVQEX1N4W28X5SFB751THG`)
- **token-estimation** — Token state handling + improved counting + perf in executor — ✅ **done** (PR #2, gate `01KYCVQEX866578AD3EZAHY6FK`)
- **model-routing** — Codex Alpha Search + new model support + error handling — ❌ not started (no phase dir; Codex Alpha Search absent from code)
- **websocket-pluginhost** — WebSocket 1009 + race fixes + pluginhost — ⚠️ partial (1009 shipped via `US-016`; pluginhost absent by decision, gated as its own high-risk initiative)
- **docs-frontend** — AIUsage showcase + management panel UI + changelog + Docker improvements — ❌ not started
- **quota-usage-reset** — Quota/usage reset controls in the management panel — ❌ not started (added 2026-07-25, see SPEC Amendment)

Dependencies: credential/auth changes first, then executor/translator, then frontend/docs.

`quota-usage-reset` is **exempt from that ordering** — it depends on nothing in
`model-routing` or `websocket-pluginhost` and ships independently of `docs-frontend`.
Its W1 is frontend-only wiring of an endpoint that already exists and is already
tested; its W2 adds a new usage-reset backend path. Do not let W2 block W1.

`quota-usage-reset` is llmhub-custom, **not** upstream parity — it is the single
approved carve-out from this SPEC's YAGNI decision.

## Phase directories on disk are NOT this SPEC's scope

`ls .kit/planning/phases/` lists 7 directories belonging to **two** initiatives.
Only 2 are this SPEC's, distinguishable by PLAN timestamp:

- this SPEC (`Updated At: 2026-07-23`) — `auth-credential-concurrency`, `token-estimation`
- older `US-016` SPEC, `v7.2.49`→`v7.2.93`, already merged in `e4bf0f7` (`Updated At: 2026-07-22`) — `auth-count-reliability`, `model-display-compatibility`, `tool-protocol-parity`, `translator-content-fidelity`, `websocket-message-too-big`

Those 5 read `Status: ready` but their code is shipped; the field is stale. Do not
re-implement them, and do not rewrite this file to match the directory listing —
that would erase the record of the 3 phases never started.
