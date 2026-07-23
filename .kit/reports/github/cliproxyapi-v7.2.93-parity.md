---
title: CLIProxyAPI v7.2.93 parity audit
description: Evidence-based comparison of llmhub HEAD against CLIProxyAPI changes after v7.2.49
status: active
created: 2026-07-21
tags: [github, CLIProxyAPI, upstream-parity]
---

# CLIProxyAPI v7.2.93 parity audit

## Baseline

- Local: `2960b690bf89232b8a23c5b8823fbe0ca831347f`.
- Upstream: `router-for-me/CLIProxyAPI` release `v7.2.93`, published 2026-07-21, commit `01f387f4`.
- Previous local audit baseline: upstream `v7.2.49`.
- Compared range: `v7.2.49...v7.2.93` — 197 commits: 84 feature, 55 fix, 11 performance, 3 refactor, 3 test, plus merge/docs/chore changes.
- GitHub compare file output is capped at 300 changed files; the range touches at least 300 files.
- The local repository has no configured upstream remote. Research used the local `librarian` skill workflow with `gh` API and selectively cached evidence under `.kit/cache/github/router-for-me/CLIProxyAPI/`.

## Existing local constraints

The existing targeted-backport policy remains correct:

- Keep Postgres as runtime source of truth.
- Keep Amp, Kiro, embedded management UI, and local installers.
- Do not import upstream `pluginhost/pluginstore`, release, Docker, sponsor, or branding churn.
- Do not replace whole auth, service, watcher, storage, or logging packages.

Binding local references:

- `docs/stories/high-risk/US-015-upstream-parity-backport/design.md:17-48`
- `docs/stories/high-risk/US-015-upstream-parity-backport/overview.md:3-35`
- `plans/cliproxyapi-upstream-parity-2026-07-02.md:125-175`

## Highest-value missing or partial behavior

| Priority | Capability | Local status | Evidence | Recommendation |
|---|---|---|---|---|
| P0 | WebSocket close 1009 propagation without credential fallback | Missing | `internal/runtime/executor/codex_websockets_executor.go:333-341`, `:572-585`; upstream mapping: `.kit/cache/github/router-for-me/CLIProxyAPI/internal/runtime/executor/codex_websockets_executor.go:932-962` | Backport first as a narrow correctness fix. |
| P0 | Generic `count_tokens` endpoint errors must not disable valid models | Missing | `sdk/cliproxy/auth/conductor.go:1537-1560`; Amp shares this path at `internal/api/modules/amp/routes.go:312-324` | Add availability-neutral handling in the shared conductor. |
| P0 | Quota cooldown escalation once per window plus bounded jitter | Partial | `sdk/cliproxy/auth/conductor.go:2294-2305`, `:2397-2417`, `:2994-3041`; upstream jitter: `.kit/cache/github/router-for-me/CLIProxyAPI/sdk/cliproxy/auth/conductor.go:3672-3702` | Adapt the upstream algorithm while preserving local cooldown persistence. |
| P0 | HTTP 400 `invalid_grant` suspension | Partial | `sdk/cliproxy/auth/conductor.go:2797-2806`, `:4447-4471`; upstream recognition/suspension: `.kit/cache/github/router-for-me/CLIProxyAPI/sdk/cliproxy/auth/conductor.go:3791-3798` | Recognize structured and textual `invalid_grant`; keep existing Postgres state flow. |
| P0 | Codex/xAI custom, additional, and namespaced tools | Missing/partial | `internal/translator/codex/claude/codex_claude_request.go:254-286`; `internal/runtime/executor/xai_executor.go:547-556`, `:708-799`; upstream namespace logic: `.kit/cache/github/router-for-me/CLIProxyAPI/internal/translator/openai/openai/responses/openai_openai-responses_tools.go:17-83`, `:236-328` | Backport as protocol-correctness work with round-trip fixtures. |
| P0 | Filter internal xAI `x_search` traces | Missing | `internal/runtime/executor/xai_executor.go:35-44`, `:161-179`, `:376-405`; upstream xAI search/tool logic: `.kit/cache/github/router-for-me/CLIProxyAPI/internal/runtime/executor/xai_executor.go:1939-2098` | Prevent clients from re-executing provider-internal search calls. |
| P1 | Structured Claude tool outputs in OpenAI Responses | Partial | `internal/translator/claude/openai/responses/claude_openai-responses_request.go:364-371`; other paths already preserve arrays/objects at `internal/translator/claude/openai/chat-completions/claude_openai_request.go:390-437` | Reuse the existing structured-content behavior instead of flattening with `.String()`. |
| P1 | OpenAI file data and MIME normalization across translators | Partial | Claude path: `internal/translator/claude/openai/responses/claude_openai-responses_request.go:253-270`; missing in Gemini/Codex paths at `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go:198-210`, `internal/translator/codex/openai/chat-completions/codex_openai_request.go:407-428`; upstream helper: `.kit/cache/github/router-for-me/CLIProxyAPI/internal/translator/common/file_data.go:10-39` | Add one shared helper and apply it surgically. |
| P1 | OpenAI Responses output indexing | Partial | `internal/translator/codex/claude/codex_claude_response.go:24-35`, `:163-207`; upstream emits stable indices at `.kit/cache/github/router-for-me/CLIProxyAPI/internal/translator/openai/openai/responses/openai_openai-responses_response.go:322-360` | Use upstream `output_index` when present; retain local fallback indexing. |
| P1 | New provider/model compatibility | Missing/partial | Local lacks Grok 4.5, GPT-5.6, Kimi K3 and Gemini production IDs in `internal/registry/models/models.json`; upstream examples at `.kit/cache/github/router-for-me/CLIProxyAPI/internal/registry/models/models.json:1433-1463`, `:2175-2192`, `:2424-2430` | Port only with coupled executor/header/normalization behavior and tests. |
| P1 | Configurable display names across APIs | Partial | Registry carries display names at `internal/registry/model_registry.go:33`, `:1161-1176`, but configured model types and OpenAI output drop them at `internal/config/config.go:424`, `sdk/api/handlers/openai/openai_handlers.go:70`; upstream config fields at `.kit/cache/github/router-for-me/CLIProxyAPI/internal/config/config.go:501-621` | Complete the existing local capability across all model-list APIs. |

## Useful but deferred

| Capability | Decision | Reason |
|---|---|---|
| Shared request-time refresh after 401 | Defer to a dedicated auth slice | High value, but Kiro already refreshes/retries locally at `internal/runtime/executor/kiro_executor.go:459-473` and `:522-536`; an unqualified port can double-refresh/replay. |
| OAuth session cancellation | Defer to a management/auth slice | Requires a pre-persistence guard around local `saveTokenRecord` so cancellation also protects Postgres writes. |
| Native xAI API-key configuration | Defer | Executor support is partial, but management/config/watcher integration is a separate public auth feature. |
| Kimi K3 | Couple with native thinking and model normalization | Adding registry metadata alone would advertise behavior the executor does not yet implement correctly. |
| GPT-5.6 variants | Couple with per-model header overrides and search routing | Metadata-only backport can produce invalid upstream requests. |
| Service-tier usage persistence | Defer | Requires API, usage record, Postgres/Redis queue, and management contract changes. |
| Deferred request-body logging and CPA trace IDs | Defer | Local Postgres mode deliberately disables file request logging; upstream logging should not be imported wholesale. |
| Encrypted xAI reasoning replay | Defer | Local has no replay cache, so there is no cross-caller cache leak; this is functional expansion rather than a current security repair. |
| Translator micro-optimizations | Defer pending benchmark | Do not port performance changes without showing a local regression or benchmark win. |

## Explicitly rejected from this backport

- Google Interactions: upstream commit `8b9c4da2` adds a new public protocol/API surface across handlers, auth selection, thinking, executors, and many translator families. Upstream handler evidence: `.kit/cache/github/router-for-me/CLIProxyAPI/sdk/api/handlers/gemini/interactions_handlers.go:26-118`. Treat it as a separate initiative, not parity maintenance.
- Remote Codex model catalog refresh: upstream performs immediate outbound fetch and repeats every three hours from external URLs at `.kit/cache/github/router-for-me/CLIProxyAPI/internal/registry/codex_client_models_updater.go:15-83`. Keep the embedded catalog and existing explicit fetch command to preserve deterministic/offline behavior and avoid an undeclared runtime network dependency.
- Upstream plugin system and related Redis/Home synchronization.
- Release, Docker, FreeBSD, sponsorship, logo, and README churn.

## Verification evidence

Focused local baseline completed successfully:

```text
go test ./sdk/api/handlers/openai ./sdk/cliproxy/auth ./internal/translator/... ./internal/runtime/executor ./internal/registry ./internal/api
```

All selected packages passed; packages without tests reported `[no test files]`.

## Implementation status

### Phase 1 — WebSocket close 1009

Approved on 2026-07-23 against the exact six-file product fingerprint.

- Evidence: `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P1-CLOSE/r01/`.
- Patch SHA-256: `081c6029e14044651368ed6cc1a212148d791d0694fd7836228138c025d14427`.
- Scratch forward application matches the tested tree; reverse application
  matches local baseline `2960b690bf89232b8a23c5b8823fbe0ca831347f`.
- All three focused suites and the combined executor/auth/OpenAI-handler suite
  passed.
- The product fingerprint was unchanged after testing.
- Independent high-risk review: `APPROVED`, zero verified Critical or Major
  findings.

Implemented behavior:

- observed Gorilla close code 1009 maps to a typed request-scoped 413-equivalent
  `message_too_big` error;
- credential accounting, cooldown, refresh, and fallback are skipped for that
  typed classification;
- ordinary non-1009 and unobserved writer errors retain existing retry behavior;
- Responses WebSocket rollback preserves transcript, output, replay, and tool
  cache state.

Story evidence:
`docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`.

### Phase 2 — Auth and CountTokens reliability

Approved on 2026-07-23 after two serial implementation slices and the combined
Phase 2 gate.

- Cooldown/jitter evidence:
  `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P2-S1-cooldown-jitter/r01/`.
- Error-classification evidence:
  `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P2-S2-error-classification/r04/`.
- Reviewed patch SHA-256 values:
  - P2-S1: `7e17ef3ab124b1fa713285fb984ac45722926121e4a2489fd4572beb068af875`;
  - P2-S2: `0f0b7c79e4627bc956a333ce64604dc4ef6a8357324be7915655cc660e0325ec`.
- Both independent slice reviews returned `APPROVED` with zero verified Critical
  or Major findings.
- P2-S2 `r01` and `r02` were rejected during review. A later audit found that
  frozen `r03` retained the structured-JSON fallback defect despite an
  inconsistent approval artifact. The stale `r03` tree was reversed from main;
  corrected `r04` was reconstructed, tested, reviewed, and applied from immutable
  result tree `98d750898d239533918d092276faf9520a64adb1`.
- Affected-package tests, the full Go suite, `go vet ./...`, `go build ./...`, and
  `git diff --check` passed on the combined main tree.

Implemented behavior:

- quota backoff advances once per recovery window and retry jitter remains inside
  the configured maximum;
- HTTP 400/401 `invalid_grant` uses exact structured or bounded textual matching,
  receives the existing 30-minute suspension, and permits credential fallback;
- unrelated HTTP 400 errors and longer identifiers remain request-invalid;
- generic CountTokens endpoint 404 responses record counters/hooks/persistence
  without hiding the model;
- explicit nested, wrapped, or joined `model_not_found` remains
  availability-changing;
- Amp inherits the conductor policy without a production route fork, while Kiro
  `disableCooling` behavior remains unchanged.

Decision record:
`docs/decisions/0010-auth-cooldown-and-error-classification.md`.

### Phase 3 — Translator content fidelity

Approved on 2026-07-23 after three independently reviewed implementation
slices and the combined Phase 3 gate.

- File-data evidence:
  `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P3-S1-file-data-normalization/r03/`.
- Claude tool-result evidence:
  `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P3-S2-claude-tool-result-content/r03/`.
- Codex-to-Claude index evidence:
  `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P3-S3-codex-claude-output-indices/r04/`.
- Reviewed patch SHA-256 values:
  - P3-S1: `1e0dae5ab9290d2a16ce81727fb0040fb5a6f6eca591d02f77e9a2761dbaca7d`;
  - P3-S2: `c7c0dff343dc6af9e00ec9858258286e3c19612b25c31bd92ddfabcd0b27c2f4`;
  - P3-S3: `9e1e4ccd0ef3a03b83b1726448f11cb53819eb56e874765eef46ec14faec62cc`.
- All three final slice reviews returned `APPROVED` with no findings.
- Earlier immutable revisions remain preserved with their rejection evidence:
  P3-S1 `r02`; P3-S2 `r01` and `r02`; P3-S3 `r01`, `r02`, and `r03`.
- The six affected translator packages, full Go suite, `go vet ./...`,
  `go build ./...`, and `git diff --check` passed on the combined main tree.

Implemented behavior:

- one shared parser normalizes raw and data-URL OpenAI file content for Gemini,
  Gemini CLI, and Codex without silently discarding invalid structured items;
- Claude tool results preserve valid structured arrays and empty arrays, validate
  native image/document/search/tool-reference shapes, accept document string or
  text/image-array content, and compact arbitrary or invalid values to JSON text;
- Codex-to-Claude streams retain provider `output_index` values, use sequential
  fallback only when absent, close active block lifecycles before transitions,
  and ignore delayed completions for retired reasoning or function items.

Decision record:
`docs/decisions/0011-translator-structured-output-and-index-fidelity.md`.

### Phase 4 — Tool protocol parity

Implemented on 2026-07-23 and accepted by the final combined repository gate.

Implemented behavior:

- request-scoped Codex and xAI declaration tables preserve original tool type,
  namespace, and name while using flattened provider names;
- `mcp__` names remain byte-stable and effective-name collisions fail before
  HTTP or WebSocket network activity with code `tool_name_collision`;
- custom tools round-trip through custom call IDs and custom input event families;
- fragmented Chat function names and arguments are classified only when identity
  is unambiguous, with deterministic terminal fallback;
- only a complete JSON object containing exactly one string `input` member is
  unwrapped as compatibility input;
- provider-internal xAI search traces are filtered without hiding exact declared
  namespaced tools, and output indexes remain compact.

The reviewed P4-S2 xAI patch remains at
`.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r03/`
with SHA-256
`869464e623c79acc03a143c3a1bbcca460c5fe127bc9a1654bbe61908861ba93`.
P4-S1 reached final implementation revision `r06`; the user-directed accelerated
workflow skipped a new immutable freeze and independent closure review.

Decision record:
`docs/decisions/0012-tool-namespace-and-x-search-lifecycle.md`.

### Phase 5 — Model and display compatibility

Implemented on 2026-07-23 and accepted by the final combined repository gate.

Implemented behavior:

- added Grok 4.5 static metadata;
- added selected Gemini 3 and 3.1 production IDs without removing preview IDs;
- added optional configured display names for Claude, Codex, Gemini,
  Vertex-compatible, OpenAI-compatible, and OAuth alias models;
- retained display names through sanitization, registration, prefix clones,
  Codex built-in replacement, and OAuth alias forks;
- exposed presentation fields through OpenAI, Codex client, Claude, Gemini, and
  Gemini CLI listings without changing IDs, routing, upstream names, providers,
  or auth selection.

GPT-5.6 and Kimi K3 remain deferred because their correct support requires
coupled executor behavior. Google Interactions, plugin systems, xAI API-key
management, encrypted replay, Kiro tool formatting, and unrelated expansion
remain excluded.

Decision record:
`docs/decisions/0013-model-display-name-presentation-contract.md`.

### Final combined gate

The first full-suite run exposed three xAI identity regressions. The corrected
implementation captures declarations before lossy namespace flattening, merges
unique translated fallback declarations, matches translated tool choices by
effective identity, and makes namespace qualification idempotent. The three
focused regressions then passed, followed by:

```text
go test ./...       pass
go vet ./...        pass
go build ./...      pass
git diff --check    pass
```

No web build or live-provider smoke test was run. All parity changes remain local
and uncommitted; no push, PR, merge, release, or publication occurred.

## Research limits

- GitHub code-search quota was exhausted after ten searches, but core API quota remained available. Selected files were fetched through repository contents and commit APIs.
- GitHub Compare exposes at most 300 changed files, so the report does not claim an exhaustive changed-file count.
- No live provider credentials were used; provider behavior must be proven through deterministic fixtures and optional operator-owned smoke tests during implementation.

## Sources

- Upstream repository: https://github.com/router-for-me/CLIProxyAPI
- Latest audited release: https://github.com/router-for-me/CLIProxyAPI/releases/tag/v7.2.93
- Official management documentation: https://help.router-for.me/management/api
