---
id: 01KY1T7GGY3DBEW2NAAV620VCK
type: spec
phase: none
lane: high-risk
intake_id: 01KY1T9KV4F9NKFP1APTK2M7ZG
created: 2026-07-21
updated: 2026-07-22
---

# SPEC: CLIProxyAPI v7.2.93 targeted parity backport

Status: locked
Input Type: maintenance
Lane: high-risk
Risk Flags: auth, external-systems, public-contract, existing-behavior, multi-domain
Affected Surfaces: api, provider, db, docs
Downstream: to-plan full
Updated At: 2026-07-22

## Source Mode

files

## Source Inputs

- Approved implementation plan: `.kit/plans/2026-07-21-cliproxyapi-v7.2.93-backport/plan.md`
- Upstream parity evidence: `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- Previous parity decisions: `plans/cliproxyapi-upstream-parity-2026-07-02.md`
- Existing high-risk story: `docs/stories/high-risk/US-015-upstream-parity-backport/`
- User execution intent: implement the approved five-phase plan without committing.

## Scenario

refine existing spec

## Goal

Backport the highest-value correctness and compatibility behavior from `router-for-me/CLIProxyAPI` release range `v7.2.49...v7.2.93` into llmhub while preserving local Postgres runtime ownership, Amp routes, Kiro behavior, embedded management UI, and installer/release architecture.

## Users / Actors

- AI coding clients using OpenAI, Claude, Gemini, Codex, xAI, Amp, and provider-specific protocol routes.
- Operators managing OAuth credentials, cooldown state, model availability, and model catalogs through llmhub.
- Maintainers reviewing and releasing targeted upstream parity changes.

## Requirements

1. Map upstream WebSocket close code `1009` to a structured request-scoped `message_too_big` error equivalent to HTTP 413.
2. A mapped WebSocket request-size error must not mark, cool down, refresh, or fall back provider credentials.
3. Quota backoff must escalate at most once inside an active cooldown window.
4. Cooldown wait jitter must not exceed `min(wait/4, 2s)` and total wait must not exceed `max-retry-interval`.
5. Structured or textual OAuth `invalid_grant`, including HTTP 400 responses, must enter a 30-minute suspension path without classifying unrelated 400 errors as auth failures.
6. A generic unsupported `count_tokens` endpoint error must be availability-neutral, while explicit structured or nested `model_not_found` remains availability-changing.
7. OpenAI file input must normalize raw base64 and data URLs through one shared pure helper and preserve or infer the correct MIME type.
8. OpenAI Responses tool-output arrays that map entirely to valid Claude tool-result content blocks must remain structured when translated to Claude; arbitrary objects and arrays that are not valid Claude content blocks must be serialized as compact JSON text, matching canonical upstream behavior and Claude's schema.
9. Provider-supplied output indices must be preserved when present, with current sequential indexing retained as fallback.
10. Codex translations must support custom tools, `additional_tools`, and namespaced tools without short-name collisions.
11. xAI execution must promote `additional_tools` to top-level tools, preserve namespace routing and choices, and avoid double-qualifying existing `mcp__` names.
12. Provider-internal xAI `x_search` traces must not be exposed as client-executable tool calls unless the client declared the corresponding tool.
13. Add Grok 4.5 metadata using upstream capability limits.
14. Add Gemini production model IDs while retaining existing preview IDs for compatibility.
15. Configured model display names must propagate consistently through OpenAI, Claude, Gemini, Gemini CLI, and Codex client model listings without changing routing IDs.
16. Every implementation slice must include deterministic unit or integration fixtures and be independently mergeable and revertible.
17. Full Go tests and web/binary builds must pass before the work is considered complete.
18. No commit, push, pull request, release, or external publication is authorized by this work request.

## Boundaries

### In Scope

- WebSocket 1009 request-scoped propagation.
- Shared auth/count-token reliability fixes that require no data migration.
- Translator content fidelity for structured output, file/MIME normalization, and output indexing.
- Codex/xAI custom, additional, namespace, and internal-search tool correctness.
- Grok 4.5, Gemini production IDs alongside previews, and configured display-name propagation.
- Focused and full verification plus parity/story evidence updates.

### Out of Scope

- Google Interactions protocol, routes, executors, translators, configuration, or public API.
- Upstream pluginhost/pluginstore, Redis/Home plugin synchronization, or plugin management UI.
- Runtime remote refresh of the Codex model catalog.
- GPT-5.6 Sol/Terra/Luna and their required header/search behavior.
- Kimi K3 and its required native thinking, `[1m]` normalization, Claude-path normalization, and token counting.
- Shared request-time 401 refresh and Kiro double-refresh prevention design.
- OAuth session cancellation and its Postgres pre-persistence guard.
- Native xAI API-key configuration and management endpoints.
- Service-tier persistence, deferred request-body logging, CPA trace IDs, encrypted xAI reasoning replay, or performance changes without local benchmark evidence.
- Release, Docker, FreeBSD, sponsor, branding, logo, and README churn.
- Database schema migrations, destructive data operations, credential overwrites, or auth-ID migrations.

## Constraints

- Use targeted adaptations; never wholesale-replace auth, service, watcher, storage, logging, or translator subsystems.
- Postgres remains authoritative for runtime config, auth records, cooldown snapshots, and usage data.
- Amp routes and Kiro executor/auth/quota/usage behavior must remain supported.
- Translator helpers must remain pure and have no storage, network, or config side effects.
- Error classification belongs at executor/auth boundaries, not duplicated in HTTP handlers.
- Existing preview model IDs remain discoverable when production IDs are added.
- Each phase must be independently testable, mergeable, and revertible.
- No new API key, external account, or third-party service is required for deterministic verification.
- Do not commit changes.

## Acceptance Criteria

- Synthetic WebSocket close 1009 produces a structured 413-style error with no credential fallback.
- Concurrent 429 fixtures inside one cooldown window advance backoff once; jitter stays inside both configured bounds.
- HTTP 400 `invalid_grant` suspends, while unrelated HTTP 400 errors do not.
- Generic CountTokens 404 leaves model availability unchanged; explicit `model_not_found` changes availability.
- Raw base64 and data URL fixtures produce equivalent provider payloads and correct MIME types.
- Valid Claude tool-result content arrays survive OpenAI Responses conversion as structured arrays; arbitrary objects and invalid/mixed arrays use deterministic compact JSON text fallback.
- Interleaved output events retain stable provider indices.
- Custom/additional/namespaced tools round-trip through selected streaming and non-streaming Codex/xAI paths without collisions or double qualification.
- xAI internal search traces are filtered unless client-declared.
- Grok 4.5 and both Gemini production and preview IDs are discoverable.
- Configured display names are consistent across all specified model listing APIs while IDs remain unchanged.
- `go test ./...`, `make build-web`, `make build`, and `git diff --check` pass.
- Documentation records implemented, deferred, and rejected upstream items with exact verification evidence.

## Validation Expectations

- Unit tests for mapped WebSocket errors and auth retry/fallback classification.
- Concurrency-aware unit tests for cooldown escalation and bounded jitter.
- Auth fixtures for structured/textual `invalid_grant` and CountTokens endpoint/model error distinction.
- Translator fixture tests for raw base64, data URLs, valid structured Claude content arrays, deterministic compact-JSON fallback for arbitrary values, output indices, custom tools, additional tools, namespaces, and xAI search filtering.
- Registry/config/API tests for model IDs and display-name propagation.
- Focused package tests after each implementation slice.
- Full Go suite plus web and binary builds after all slices.
- Optional live smoke tests may use operator-owned credentials, but deterministic tests are the completion proof.

## Dependencies / Assumptions

- Upstream evidence is pinned to CLIProxyAPI `v7.2.93` at commit `01f387f4`.
- Local implementation baseline is `2960b690bf89232b8a23c5b8823fbe0ca831347f` plus the planning/report artifacts created for this task.
- Local translator and executor interfaces remain close enough to adapt upstream algorithms and fixtures without importing upstream architecture.
- If an upstream implementation cannot fit the current local interface, the externally observable behavior will be implemented behind the local interface rather than widening the import boundary.

## Key Decisions

- Chosen: targeted five-slice backport ordered by correctness and dependency risk.
- Chosen on 2026-07-22: match canonical upstream and Claude schema for tool-result content — preserve arrays only when every member maps to a valid Claude content block; serialize arbitrary objects and invalid/mixed arrays as compact JSON text.
- Rejected: wholesale upstream merge because it would conflict with Postgres runtime ownership, Amp, Kiro, embedded management, local installers, and excluded plugin/release systems.
- Rejected: P0-only patch because it would leave known high-impact tool protocol and safe model/display-name compatibility gaps unresolved.
- Rejected: complete parity through `v7.2.93` because Google Interactions, plugins, runtime remote refresh, GPT-5.6, Kimi K3, logging, and replay features require separate product or architecture decisions.
- Rejected: runtime Codex catalog refresh because it adds periodic undeclared outbound network access and weakens deterministic/offline behavior.

## Open Questions

- None. Scope, behavior, exclusions, verification, rollback, and execution intent were approved before this SPEC lock.

## Deferred Ideas

- Dedicated Google Interactions initiative.
- GPT-5.6 with coupled model-header and search-routing support.
- Kimi K3 with native thinking and full model normalization.
- Shared request-time 401 refresh with an explicit Kiro bypass/ownership contract.
- Postgres-safe OAuth cancellation.
- Native xAI API-key configuration and management.
- Service-tier persistence and provider cost attribution.
- Benchmark-driven translator performance work.

## Ambiguity Report

- Goal clarity: complete; upstream baseline and local outcome are pinned.
- Scope clarity: complete; five included slices and explicit exclusions are locked.
- Constraints clarity: complete; Postgres, Amp, Kiro, architecture boundaries, rollback, and no-commit rule are explicit.
- Acceptance clarity: complete; every requirement has deterministic proof and final build commands.
