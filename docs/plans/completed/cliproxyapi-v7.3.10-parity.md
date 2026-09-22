---
id: plan-20260921-v7310
type: plan
intake_id: intake-20260921-v7310
lane: high-risk
status: active
created: 2026-09-21
updated: 2026-09-21
---

# Plan: CLIProxyAPI v7.3.9..v7.3.10 targeted parity

## Outcome
- result: llmhub ports the user-approved v7.3.9..v7.3.10 capability slices (3 slices, 2 upstream commits, ~3.5k upstream LOC delta) as independently verifiable semantic ports behind existing kimi auth/executor, session/LCP, auth conductor, and translator interfaces — without wholesale merge, interactions-translator subsystem, or docs/test churn.
- success_signals:
  - Kimi.ai end-to-end: `--kimi-ai-login` and `GET /v0/management/kimi-ai-auth-url` run the device flow against `auth.kimi.ai`, save a token record typed `kimi-ai` with `domain`/`base_url`, and kimi-ai credentials route to `api.kimi.ai` for chat-completions, responses, and claude-messages paths.
  - Context compaction correctness: a compacted Claude Code conversation is matched by `MerklePrefixMatcher` as `node_kind=compaction` (not fork/subagent), keeps session affinity and parent linkage, and `NodeKind`/`IsCompaction`/`IsFork` appear in `SessionInfo`, `SessionTreeNode`, `ClientRequestMetadata`, and queued usage records.
  - Reasoning preservation: `ConvertOpenAIResponsesRequestToOpenAIChatCompletions` propagates the latest assistant reasoning onto consecutive tool-call turns, and emits `[reasoning unavailable]` when session reasoning is enabled but no reasoning exists.
  - Each accepted slice lands with focused tests citing its upstream commit(s) and local symbol.
  - `go test` on touched packages, `make build`, `git diff --check`, `gofmt` clean at gates.
  - Newer-than-v7.3.10 upstream delta is pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.10` at `a5ab69521f7b` / `refs/upstream-checkpoints/cliproxyapi/v7.3.10`; `scope_policy` (3 include / 4 exclude) is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.3.9..v7.3.10.json` — 79 paths (match 0, baseline 1, upstream-add-absent 5, diverged-absent 24, upstream-delete-present-local 0, semantic-review 49).
  - `docs/upstream/cliproxyapi-ledger-v7.3.9..v7.3.10.md` — 10 non-merge commits, every row disposed (3 adapt / 1 defer / 4 reject / 1 already-present / 1 superseded-locally).
  - `docs/plans/completed/cliproxyapi-v7.3.9-parity.md` — prior cycle; its slices must not regress (esp. Devin wave-2, response-model observability, codex duplex).
  - `CLAUDE.md` / `docs/WORKFLOW.md` — Postgres-authoritative store, SDK additive-only, no `web/**/*_test.go`, llmhub branding.
- rejected_alternatives:
  - Wholesale merge of the 79-path delta — diverged-absent/semantic-review dominate; upstream splits (`conductor_*.go`, `claude_client_detection.go`) map onto local monoliths (`conductor.go`, `claude_device_profile.go`, `claude_executor.go`).
  - Porting upstream's split conductor files — local `sdk/cliproxy/auth/conductor.go` + `home_*.go` + `cooldown_state.go` is a deliberate monolithic restructure; port behavior only.
  - Adopting upstream's `plausibleClaudeCLIVersion` patch-floor policy — local `shouldUpgradeClaudeDeviceProfile` (`claude_device_profile.go:186`) deliberately learns any newer version; superseded locally.
- requirements:
  - R1 [accepted]: **Kimi.ai domain in auth core** — `internal/auth/kimi/kimi.go` gains domain constants (`KimiAIDomain`, `KimiAIOAuthHost`, `KimiAIAPIBaseURL`), resolvers (`IsKimiAIDomain`, `IsKimiComDomain`, `NormalizeKimiDomain`, `ResolveKimiOAuthHost`, `ResolveKimiAPIBaseURL`, `ResolveKimiDomainFromAuth`, `IsKimiAIAuth`), `NewKimiAuthWithDomain`, `NewDeviceFlowClientWithDomainDeviceIDAndProxyURL`, `DeviceFlowClient.SetHTTPClient`; `KimiTokenStorage` gains `Domain`/`BaseURL`; `token.go` refresh/exchange endpoints follow resolved domain. | source: `a5ab69521f7b`
  - R2 [accepted]: **Kimi.ai login surfaces** — `cmd/server` `--kimi-ai-login` flag → `cmd.DoKimiAILogin`; `sdk/auth/kimi.go` `NewKimiAIAuthenticator`/`NewKimiAIDotAuthenticator`; management route `GET /kimi-ai-auth-url` + `RequestKimiAIToken` + `?domain`/`?channel` query on `/kimi-auth-url`; `auth_manager.go` and `sdk/auth/refresh_registry.go` register `kimi-ai`/`kimi.ai`; TUI oauth tab gains the Kimi.ai entry. | source: `a5ab69521f7b`
  - R3 [accepted]: **Kimi.ai runtime routing** — `helps/kimi_responses.go` gains `ResolveKimiBaseURL`/`ResolveKimiChatURL`/`ResolveKimiClaudeBaseURL`; `kimi_executor.go` routes chat/responses/claude-messages/count via resolvers and refreshes with domain-aware client + storage update; `watcher/synthesizer/file.go` preserves kimi `domain`/`base_url` attributes (+ `Auth.FileName`); `claude_signing.go` accepts `api.kimi.ai` and provider aliases; `internal/thinking` registers `kimi-ai`/`kimi.ai`/`kimi.com` in apply/strip/native maps; registry channel/model-updater/web-search-path accept the aliases; conductor/service provider-set normalization maps kimi aliases. | source: `a5ab69521f7b`
  - R4 [accepted]: **LCP context-compaction engine** — `sdk/cliproxy/session/lcp.go` gains `PrepareExt`, `EnvironmentDigest`, `MatchFingerprintsWithContext`, `BindFingerprintsWithContext`, `TouchFingerprintsWithContext`, `matchCompactionLocked`, `newLCPCompactionSessionID`, ancestor/overlap logic; `MerklePrefixMatch` carries `IsCompaction`; `sdk/cliproxy/executor/types.go` gains `IsCompactionMetadataKey`, `NodeKindMetadataKey`, `LCPTailFingerprintsMetadataKey`, `LCPEnvironmentDigestMetadataKey`. | source: `28100e54b9b5`
  - R5 [accepted]: **Node-kind metadata propagation** — `session/info.go` ExtractSessionInfo distinguishes compaction (NodeKind=`compaction`, AgentName=`main`) from fork (`fork`, `subagent`); `tree_compat.go` `SessionTreeNode` gains `NodeKind`/`IsFork`/`IsCompaction`; `internal/logging/requestmeta.go` + `internal/redisqueue/plugin.go` usage records gain the three fields; `internal/home` auth dispatch forwards `X-Node-Kind`; `sdk/cliproxy/auth/selector.go` `pickLCP`/`homeDispatchSessionIDs` consume the WithContext APIs + `providerFromSourceFormat`/`canonicalLCPProvider`/`sessionMetadataString`; conductor resets compaction/fork/node-kind metadata when canonical session is empty. | source: `28100e54b9b5`
  - R6 [accepted]: **Reasoning preservation across tool turns** — `internal/translator/openai/openai/responses/openai_openai-responses_request.go` gains `hasReasoningInSession` detection, `latestReasoningContent` carry-forward, `fallbackToolReasoning` ("[reasoning unavailable]"), `isUsableResponsesReasoning`; reasoning resets across non-assistant boundaries; merged/tool-call assistant messages inherit latest reasoning. | source: `40cc6489879a`
  - R7 [accepted]: **Final gate** — re-resolve latest upstream release (`upstream_sync.py sync --slug cliproxyapi`), refresh checkpoint; pin any newer-than-v7.3.10 delta as explicit follow-up in ledger + Progress — never silent scope growth. | source: skill contract

## Non-goals
- `cdfb79ef843f` gemini/antigravity *interactions* builtin tools — the `internal/translator/{gemini,antigravity}/interactions/` subsystem has no local equivalent (`internal/translator/common/antigravity_tools.go:11-13`); local builtin-tool support lives in the openai-responses paths.
- `83a4913aa49c` + `c52ca7bd4e0b` claude patch-floor native-passthrough — superseded locally; adopting upstream's same-major.minor floor is a separate product decision.
- `e56547f39d57` scheduler rebuild epoch guard — deferred until profiling shows rebuild cost (recorded in `scope_policy.exclude` as `scheduler-rebuild-epoch-guard-deferred-until-profiling`).
- `ddc3f731f45a` (STEERING.md docs — feature already present: `codex_websockets_duplex.go`, `sdk_config.go:65-70`), `33ae35d53a23` (README links), `563865e77adb` (test timing churn) — docs/test-only.
- Upstream file-split refactors, pluginhost, Home/gitstore, branding — standing exclusions.

## Approach and Risks
- approach: three semantic ports of `a5ab69521f7b` (kimi-ai domain), `28100e54b9b5` (LCP compaction + node-kind), `40cc6489879a` (reasoning tool-turn preservation). Kimi work is split: auth-core gate (p0) → runtime routing + login surfaces in parallel (p1/p2) → auth-layer alias wiring (p5). LCP work: engine first (p3) → propagation second (p6). p5 and p6 run sequentially because both own `sdk/cliproxy/auth/conductor.go`.
- upstream→local surface map (verified by grep):
  - `sdk/cliproxy/auth/conductor_{execution,selection,home,lifecycle,cooldown,refresh}.go` → local `conductor.go` + `home_dispatch.go`/`home_selection.go`/`cooldown_state.go`
  - `sdk/cliproxy/service_{auth,executors,models,plugins}.go` → local `sdk/cliproxy/service.go`
  - `internal/api/handlers/management/auth_files_provider_oauth.go` → local `auth_files.go` (`RequestKimiToken`, `server.go:879` route)
  - upstream `selector.go` LCP hunks → local `selector.go` (`pickLCP` :1338, `lcpAffinityNamespace` :1430, `sessionMetadataString` :1479, `extractSessionIDs` :1721); `homeDispatchSessionIDs` → `home_session_alias.go:143`
  - upstream `internal/translator/{gemini,antigravity}/interactions/` → no local equivalent (excluded)
- risks:
  - r1: kimi commit spans 43 upstream files; mapping split-file hunks onto monoliths can drop a call site → mitigate by grepping every upstream-touched function's local equivalent before task close; `conductor_executor_replace_test.go` (upstream-add-absent) hunk covers `executorLocked`-era behavior — evaluate before skipping.
  - r2: `lcp.go` diverged earlier; upstream diff may not apply cleanly → port `PrepareExt`/`WithContext` semantics against current local structures (`lcpNamespace`, `lcpGroup`, `bindLocked`/`touchLocked`/`matchLocked` signatures differ) rather than patching textually.
  - r3: kimi `Refresh` uses `helps.NewProxyAwareHTTPClient` + `client.SetHTTPClient` — both must exist or be ported; executor `CountTokens` delegates to `ClaudeExecutor.countTokensUpstream` — verify local signature.
  - r4: `ExtractSessionInfo` local version already sets `AgentName`/`ParentSessionID` for claude agents (`info.go:121-170`) — compaction/fork split must not regress that behavior.
- recovery: per-phase revert of touched files; upstream end-state preserved at `refs/upstream-checkpoints/cliproxyapi/v7.3.10`.

## Phases and Verification

### Wave 0 — sequential gate
- phase: `p0-kimi-auth-core` — story_id: `story-20260921-p0-kimi-auth-core` — status: `checked` — deps: none
  - goal: kimi.ai domain resolution exists in the auth core.
  - tasks:
    - p0-t1 `internal/auth/kimi/kimi.go`: domain constants (`KimiAIDomain`, `KimiAIOAuthHost`, `KimiAIAPIBaseURL`, export `KimiOAuthHost`), `IsKimiAIDomain`/`IsKimiComDomain`/`NormalizeKimiDomain`/`ResolveKimiOAuthHost`/`ResolveKimiAPIBaseURL`/`ResolveKimiDomainFromAuth`/`IsKimiAIAuth`, `NewKimiAuthWithDomain`, `NewDeviceFlowClientWithDomainDeviceIDAndProxyURL`, `DeviceFlowClient.SetHTTPClient`; `CreateTokenStorage` writes `Type`/`Domain`/`BaseURL`.
    - p0-t2 `internal/auth/kimi/token.go`: `KimiTokenStorage` gains `Domain`/`BaseURL`; refresh/token URLs resolve from domain.
    - p0-t3 port upstream `internal/auth/kimi/kimi_test.go` (upstream-add-absent) — adapt to local storage type.
  - surfaces: `internal/auth/kimi/**`
  - verify: `go test ./internal/auth/kimi/... -count=1` green; `go build ./...`.
  - stop condition: `ResolveKimiDomainFromAuth`/`NewKimiAuthWithDomain` missing → halt, re-review upstream `a5ab69521f7b` kimi.go.

### Wave 1 — parallel phases (file-disjoint)
- phase: `p1-kimi-runtime` — story_id: `story-20260921-p1-kimi-runtime` — status: `checked` — deps: `p0-kimi-auth-core`
  - goal: kimi-ai credentials route to `api.kimi.ai` on every executor path.
  - tasks:
    - p1-t1 `internal/runtime/executor/helps/kimi_responses.go`: `ResolveKimiBaseURL`/`ResolveKimiChatURL`/`ResolveKimiClaudeBaseURL` + metadata/attributes/base_url resolution order.
    - p1-t2 `internal/runtime/executor/kimi_executor.go`: Execute/ExecuteStream/CountTokens route via resolvers; `Refresh` uses `ResolveKimiDomainFromAuth` + proxy-aware client + storage Domain/BaseURL update; nil-safe `auth.Attributes`.
    - p1-t3 `internal/runtime/executor/claude_signing.go`: `isKimiAPIEndpoint` accepts `api.kimi.ai`; `isKimiMessagesUpstream` accepts `kimi-ai`/`kimi.ai`/`kimi.com` providers.
    - p1-t4 `internal/watcher/synthesizer/file.go`: set `Auth.FileName`; preserve kimi `domain`/`base_url` attributes + `ResolveKimiDomainFromAuth` normalization (+ upstream `file_test.go` hunks).
    - p1-t5 `internal/thinking/{apply.go,strip.go,provider/kimi/apply.go}`: register `kimi-ai`/`kimi.ai`/`kimi.com` appliers + extract/strip cases.
    - p1-t6 `internal/registry/{model_definitions,model_registry,model_updater}.go`: channel aliases, `responsesWebSearchProviderPathSupport`, `detectChangedProviders` rows (+ `model_updater_test.go`, `model_definitions_test.go` hunks).
    - p1-t7 port relevant `kimi_executor_test.go` + `kimi_responses_test.go` hunks.
  - surfaces: listed files only; avoid `sdk/cliproxy/**`, `internal/cmd/**`, `internal/api/**`, `cmd/**`.
  - verify: `go test ./internal/runtime/executor/... ./internal/thinking/... ./internal/registry/... ./internal/watcher/... -count=1`.
- phase: `p2-kimi-login-surfaces` — story_id: `story-20260921-p2-kimi-login` — status: `checked` — deps: `p0-kimi-auth-core`
  - goal: `--kimi-ai-login` and `/kimi-ai-auth-url` produce `kimi-ai` auth records.
  - tasks:
    - p2-t1 `sdk/auth/kimi.go`: domain/provider fields, `NewKimiAIAuthenticator`, `NewKimiAIDotAuthenticator`, metadata `domain`/`base_url`, file-prefix `kimi-ai`, storage Type override.
    - p2-t2 `internal/cmd/kimi_login.go`: `DoKimiAILogin` + `doKimiLoginWithProvider`; `internal/cmd/auth_manager.go` registers both authenticators.
    - p2-t3 `cmd/server/main.go`: `--kimi-ai-login` flag + `commandMode` + `argvFlagConsumesValue` entries.
    - p2-t4 `internal/api/handlers/management/auth_files.go`: `RequestKimiToken` gains `?domain`/`?channel` query → `requestKimiTokenWithDomain`; new `RequestKimiAIToken`; `internal/api/server.go:879` area registers `GET /kimi-ai-auth-url`; port `internal/api/server_kimi_oauth_test.go` (upstream-add-absent).
    - p2-t5 `internal/tui/oauth_tab.go`: Kimi (kimi.com)/(kimi.ai) rows + `kimi-ai-auth-url` providerKey case.
    - p2-t6 `sdk/auth/refresh_registry.go`: register `kimi-ai`, `kimi.ai` refresh leads (+ test hunks).
  - surfaces: listed files only; avoid `internal/auth/**`, `sdk/cliproxy/**`.
  - verify: `go test ./sdk/auth/... ./internal/cmd/... ./internal/api/... -count=1`; `go build ./cmd/server` shows `--kimi-ai-login` in `-h` output.
- phase: `p3-lcp-compaction-engine` — story_id: `story-20260921-p3-lcp-engine` — status: `checked` — deps: none
  - goal: `MerklePrefixMatcher` supports compaction continuation matching.
  - tasks:
    - p3-t1 `sdk/cliproxy/executor/types.go`: `IsCompactionMetadataKey`, `NodeKindMetadataKey`, `LCPTailFingerprintsMetadataKey`, `LCPEnvironmentDigestMetadataKey`.
    - p3-t2 `sdk/cliproxy/session/lcp.go`: `PrepareExt`, `EnvironmentDigest`, tail-fingerprint extraction, `MatchFingerprintsWithContext`/`BindFingerprintsWithContext`/`TouchFingerprintsWithContext`, `matchCompactionLocked`, `isAncestorSession`, `isCompactionOverlap`, `newLCPCompactionSessionID`, `calculateOverlap`; `MerklePrefixMatch.IsCompaction` (+ParentSessionID if absent); keep `Prepare`/`MatchFingerprints`/`Bind*` signatures for existing callers.
    - p3-t3 port upstream `sdk/cliproxy/session/lcp_test.go` hunks (+840 LOC upstream — port compaction/WithContext cases).
  - surfaces: `sdk/cliproxy/session/lcp.go`, `lcp_test.go`, `sdk/cliproxy/executor/types.go`; avoid `selector.go`, `conductor.go`.
  - verify: `go test ./sdk/cliproxy/session/... -count=1` green incl. new compaction cases.
- phase: `p4-reasoning-tool-turns` — story_id: `story-20260921-p4-reasoning` — status: `checked` — deps: none
  - goal: reasoning content survives consecutive tool-call turns in responses→chat conversion.
  - tasks:
    - p4-t1 `internal/translator/openai/openai/responses/openai_openai-responses_request.go`: `hasReasoningInSession` detection (reasoning.effort / reasoning_effort / reasoning object / input items), `latestReasoningContent` carry-forward, `fallbackToolReasoning` (`[reasoning unavailable]`), `isUsableResponsesReasoning`, reset on non-assistant boundary, apply to merged + tool-call assistant messages.
    - p4-t2 port upstream `openai_openai-responses_request_test.go` hunks (+147 LOC).
  - surfaces: the two files only.
  - verify: `go test ./internal/translator/openai/... -count=1`.

### Wave 2 — sequential (shared `sdk/cliproxy/auth` + `service.go` owner)
- phase: `p5-kimi-auth-wiring` — story_id: `story-20260921-p5-kimi-wiring` — status: `checked` — deps: `p1-kimi-runtime`, `p2-kimi-login-surfaces`
  - goal: kimi-ai/kimi.ai/kimi.com aliases normalize to kimi across auth selection and service plumbing.
  - tasks:
    - p5-t1 `sdk/cliproxy/auth/conductor.go`: providerSet/`normalizeProviders` expansions (sites :3347, :3392, :3475) map all four kimi spellings → `kimi`; `conductor_diagnostics.go:186` providerSet same treatment.
    - p5-t2 `sdk/cliproxy/auth/oauth_model_alias.go` kimi case + `auto_refresh_loop.go` kimi-ai lead registration; `sdk/cliproxy/auth/scheduler.go` kimi hunk if applicable to local scheduler keys.
    - p5-t3 `sdk/cliproxy/service.go:1066` providerSet normalization (all kimi spellings set each other); evaluate upstream `executorLocked` micro-fix — adopt only if local equivalent has the same unlocked-map access.
    - p5-t4 port `conductor_executor_replace_test.go`/`oauth_model_alias_test.go` hunks where behavior maps.
  - surfaces: `sdk/cliproxy/**`; avoid `selector.go` LCP functions, `session/**`.
  - verify: `go test ./sdk/cliproxy/... -count=1`.

### Wave 3 — sequential after p5 (same conductor.go surface) + p3
- phase: `p6-node-kind-propagation` — story_id: `story-20260921-p6-node-kind` — status: `checked` — deps: `p3-lcp-compaction-engine`, `p5-kimi-auth-wiring`
  - goal: compaction/fork/trunk metadata flows end-to-end: matcher → selector → conductor → home → usage.
  - tasks:
    - p6-t1 `sdk/cliproxy/session/info.go` `ExtractSessionInfo`: `IsCompaction`→NodeKind=`compaction`/AgentName=`main`; else fork/`subagent` (`info.go:121-170` region); `SessionInfo` gains `NodeKind`/`IsCompaction`.
    - p6-t2 `sdk/cliproxy/session/tree_compat.go`: `SessionTreeNode` + `RecordNode` gain `NodeKind`/`IsFork`/`IsCompaction`.
    - p6-t3 `internal/logging/requestmeta.go` `ClientRequestMetadata` + `internal/redisqueue/plugin.go` usage record/queue detail gain the three fields.
    - p6-t4 `internal/home/{client.go,requests.go}`: `X-Node-Kind` header → `authDispatchRequest.NodeKind` (+ test hunk).
    - p6-t5 `sdk/cliproxy/auth/selector.go`: `pickLCP` → `PrepareExt`/`MatchFingerprintsWithContext`, write `LCPTailFingerprints`/`LCPEnvironmentDigest`/compaction/fork/node-kind metadata; `Pick` clears the keys on fallback; add `providerFromSourceFormat`, `canonicalLCPProvider` if absent.
    - p6-t6 `sdk/cliproxy/auth/home_session_alias.go:143` `homeDispatchSessionIDs`: LCP matcher fallback when `primary==""` (bind `home-pending`, propagate parent/fork/compaction metadata); conductor reset of compaction/fork/node-kind when canonical session ID is empty (upstream `conductor_execution.go` hunk → local `conductor.go` Execute path).
    - p6-t7 port `selector_lcp_test.go` + `home_session_alias_test.go` hunks where behavior maps.
  - surfaces: `sdk/cliproxy/session/{info,tree_compat}.go`, `sdk/cliproxy/auth/{selector,home_session_alias,conductor}.go`, `internal/{logging,redisqueue,home}`.
  - verify: `go test ./sdk/cliproxy/... ./internal/home/... ./internal/redisqueue/... ./internal/logging/... -count=1`.

### Wave 4 — final gate
- phase: `p7-final-gate` — story_id: `story-20260921-p7-final-gate` — status: `checked` — deps: all above
  - goal: satisfy R7 and repo gates.
  - tasks:
    - p7-t1 `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi` — re-resolve latest; if >v7.3.10, pin delta as follow-up in ledger + Progress (do not absorb).
    - p7-t2 `go test ./... -count=1` (record pre-existing env failures), `make build`, `git diff --check`, `gofmt -l` on changed files.
    - p7-t3 update checkpoint/ledger; move plan to `docs/plans/completed/` after validation.
  - verify: all commands above exit clean or with documented pre-existing failures.

## Progress
- 2026-09-21 — checkpoint synced v7.3.9→v7.3.10 (`a5ab69521f7b`, 1 release, local baseline `1ccfcecdb773`); gap 79 paths (baseline 1 / add-absent 5 / diverged-absent 24 / semantic-review 49); ledger 10 commits dispositioned (3 adapt / 1 defer / 4 reject / 1 already-present / 1 superseded-locally); `scope_policy` recorded (3 include / 4 exclude). Next action: `to-plan`.
- 2026-09-21T15:00Z — phase `p0-kimi-auth-core` started; task=p0-start task_status=in-progress; surfaces `internal/auth/kimi/**`.
- 2026-09-21T15:10Z — task=p0-t1 task_status=DONE; kimi.go upstream v7.3.10 end-state ported (domain constants, resolvers, WithDomain constructors, SetHTTPClient, CreateTokenStorage Type/Domain/BaseURL, singleflight refresh); local masking headers preserved.
- 2026-09-21T15:10Z — task=p0-t2 task_status=DONE; token.go gained Domain/BaseURL + conditional Type/Domain/BaseURL defaults in SaveTokenToFile; local merge-before-create ordering kept.
- 2026-09-21T15:10Z — task=p0-t3 task_status=DONE; ported `kimi_test.go` (6 funcs) + `kimi_refresh_test.go` (singleflight helpers — earlier-cycle file absent locally).
- 2026-09-21T15:10Z — wave-0 summary: `go test ./internal/auth/kimi/...` green, `go build ./...` clean, `git diff --check` clean, gofmt clean; p0 gated → `checked`.
- 2026-09-21T18:00Z — wave-1 fanout: 4 background subagents launched (p1 `b538aa0f`, p2 `38eccb59`, p3 `a4e4ca73`, p4 `acffd88c`), surfaces file-disjoint per plan.
- 2026-09-21T18:40Z — task=p4-t1,t2 task_status=DONE; reasoning carry-forward + fallback + session detection ported; **deviation**: absorbed commit's parent-state machinery (messages-accumulation pipeline, `takePendingReasoningContent`, assistant-merge, `AlignOpenAIToolCallMessages` final flush) — local file predated upstream reasoning pipeline entirely; preserved local divergences (`NormalizeResponsesToolsForCodex`, `qualifyResponsesNamespaceToolName`, `appendChatTools`+additional_tools). 5 upstream tests ported verbatim.
- 2026-09-21T18:45Z — task=p2-t1..t6 task_status=DONE; sdk authenticators + `--kimi-ai-login` + `/kimi-ai-auth-url` + TUI + refresh-registry landed. Deviations: `argvFlagConsumesValue` lives in `cmd/server/discover.go` (local refactor); `commandMode` hunk N/A (no such symbol — Postgres-config startup); added missing `devin-login` to consumes-value list; preserved local `CompleteOAuthSessionsByProvider` + device-flow response fields. `oauth_sessions_test.go` providers hunk applied in-session (was outside worker surfaces).
- 2026-09-21T18:50Z — task=p3-t1..t3 task_status=DONE; `lcp.go` ported **byte-identical** to upstream v7.3.10 end-state modulo module path (verified via `diff` vs checkpoint ref); `types.go` +4 metadata keys; all 20 upstream tests ported verbatim, zero skips. Pre-existing gap noted: `LookupSession` absent locally (exists upstream since v7.3.9, no local callers — follow-up only if needed).
- 2026-09-21T18:55Z — task=p1-t1..t7 task_status=DONE; helps resolvers + executor routing/refresh (byte-parity `Refresh`, `NewProxyAwareHTTPClient` already existed) + claude_signing detection + watcher FileName/domain attrs + thinking×4 aliases + registry aliases + ported tests. Deviation: `claude_signing.go` got only `isKimiAPIEndpoint`/`isKimiMessagesUpstream` — upstream's `stripDefaultKimiClaudeCodeAttribution` machinery has no local counterpart/callers (diverged 94-line file); follow-up if attribution stripping is ever needed.
- 2026-09-21T18:55Z — wave-1 summary: union verify on combined tree — `go build ./...` clean; `go test` on all wave-1 packages green (executor 3.4s, helps 11.0s, sdk/cliproxy/auth regression 30.7s, api/cmd/auth/registry/watcher/thinking/session/translator all `ok`); `git diff --check` + `gofmt -l` clean. All four phases gated → `checked`. Next action: `p5-kimi-auth-wiring` (wave-2 sequential).
- 2026-09-21T19:20Z — task=p5-t1..t4 task_status=DONE (in-session, wave-2 sequential). Added `canonicalSchedulingProvider` (kimi.com→kimi, kimi.ai→kimi-ai) + `executorLocked` (kimi-alias fallback → "kimi") in conductor.go; `executorKeyFromAuth` kimi switch; all lock-held `m.executors[...]` reads → `executorLocked` (pickNextLegacy, pickNextMixedLegacy, refreshAuthForRequest, InjectCredentials, findAllAntigravityCreditsCandidateAuths, executorFor, warnLogAuthUnavailable, auto_refresh_loop); canonical provider keys in pickNext*/mixed/cooldown-wait/retryAllowed/diagnostics + scheduler (`pickSingle`, `normalizeProviderKeys`, `upsertAuthLocked`, `buildScheduledAuthMeta`); ported upstream `AvailableProviders`+`HasProviderAuth` (new local API — upstream tests require them); `oauth_model_alias` + service.go (4 authenticators in `newDefaultAuthManager`, executor/model-registration case expansion, refresh-callback kimi expansion). Tests: ported upstream `TestManager{RefreshAndLegacySelectionKimiAliases,SchedulerFastPathKimiAI,KimiDomainIsolation}` + kimi loop in `TestManagerExecutorReturnsRegisteredExecutor` + `OAuthModelAliasChannel` kimi loop — all green; `go test ./sdk/cliproxy/...` all ok; build/gofmt/diff-check clean.
- 2026-09-21T19:20Z — wave-2 deviations (recorded): upstream `requestToFormat`/`RequestAfterAuthInterceptor` machinery absent locally → hunk N/A; `baselineExecutorAuths` → local `registerHomeExecutors` registers executors directly, aliases covered by `executorLocked` fallback → N/A; `service_executor_registration_test.go` has no local counterpart → executor-alias coverage via `TestManagerExecutorReturnsRegisteredExecutor`; candidate loops keep local `candidate.Provider` basis wrapped in `canonicalSchedulingProvider` (upstream uses `executorKeyFromAuth` — preserves local compat-auth routing semantics); scheduler bucket keys canonicalized via `canonicalSchedulingProvider(auth.Provider)` (upstream uses `executorKeyFromAuth`); `newDefaultAuthManager` still lacks Devin/Kiro authenticators — pre-existing drift, not kimi scope.
- 2026-09-21T20:15Z — task=p7-t1 task_status=DONE; re-resolve moved upstream to `v7.3.11` (`ffe6ad3c5fcf`, 8 commits / 19 paths) — delta pinned as follow-up only: `docs/upstream/cliproxyapi-ledger-v7.3.10..v7.3.11.md` stub generated with blank dispositions + `pinned-follow-up` status; gap JSON `cliproxyapi-gap-v7.3.10..v7.3.11.json` written; NOT absorbed into this plan.
- 2026-09-21T20:15Z — task=p7-t2 task_status=DONE; `go test ./... -count=1` zero failures; `PATH=/usr/local/go/bin:$PATH make build` clean (web embed + `llmhub` binary); `git diff --check` clean; `gofmt -l` clean across all changed Go files.
- 2026-09-21T20:15Z — task=p7-t3 task_status=DONE; checkpoint.json `checkpoint`=v7.3.11 (latest resolved), v7.3.10 moved to `prior_checkpoints` with role `parity-completed-targeted-semantic-ports`; plan moved to `docs/plans/completed/`.
- 2026-09-21T20:10Z — task=p6-t1..t7 task_status=DONE (in-session, wave-3 sequential). `SessionInfo`/`SessionTreeNode`/`RecordNode` + `ClientRequestMetadata` + `queuedUsageDetail` gained `NodeKind`/`IsFork`/`IsCompaction`; `newAuthDispatchRequest` reads `X-Node-Kind`; `pickLCP`→`PrepareExt`/`MatchFingerprintsWithContext`/`BindFingerprintsWithContext` + `OnResult`→`PrepareExt`/`TouchFingerprintsWithContext` with tail-fingerprint/env-digest metadata write-through; `Pick` explicit-identity branch clears IsCompaction/NodeKind; `canonicalLCPProvider`+`providerFromSourceFormat` added; `homeDispatchSessionIDs` gained LCP matcher fallback (bind `home-pending`, fork/compaction metadata propagation) + `pickHomeDispatchSelection` seeds `SessionAffinityModelMetadataKey` + `X-Node-Kind` dispatch header; `reportHomeResult`→`updateSessionAffinity`; `syncMetadataSessionToContext` clears/propagates node-kind/fork/compaction and re-gained `util.WithSessionID`; `LookupSession` ported (test caller). Tests: 1 compaction test in `selector_lcp_test.go`, 3 tests in `home_session_alias_test.go`, `TestAuthDispatchRequestIncludesNodeKind` adapted to local 4-arg signature — all green; `LookupAffinity` alias test skipped (API absent locally).

## Decisions
- `2026-09-21` — p0: ported upstream v7.3.10 `kimi.go` wholesale incl. singleflight refresh (`kimiRefreshGroup`, `refreshTokenSingleFlight`) — local lacked it though it predates this range (v7.3.9-era parity gap absorbed); kept local masking headers (`X-Msh-Platform: cli-proxy-api`, `X-Msh-Version: 1.0.0`) and dropped the `buildinfo` import upstream uses. rationale: end-state port keeps refresh dedup keyed `tokenURL:refreshToken`, required for domain-aware isolation.
- `2026-09-21` — p4: absorbed the upstream reasoning pipeline's parent-state machinery (pending-reasoning buffer, assistant merge, tool-alignment flush) because local `openai_openai-responses_request.go` predated it — the fix's hunks had no anchor otherwise. rationale: matching upstream end-state for this function is cheaper than maintaining a divergent reasoning model; remaining upstream deltas in the same function (`text.format`→`response_format`, `input_image` detail, `mergeResponsesRequestChatTools`) pinned as follow-ups, not absorbed.
- `2026-09-21` — p1/p2: kimi `type`/`provider` canonical form is `kimi-ai` for the `.ai` domain (upstream convention), with `kimi.ai`/`kimi.com` accepted as aliases at every ingress (auth fields, watcher synthesis, thinking/registry maps); local extras preserved (`CompleteOAuthSessionsByProvider`, device-flow response fields, `devin-login` flag registration).
- `2026-09-21` — p6: ported `LookupSession` (pre-existing v7.3.9-era gap deliberately left in p3) because the ported `TestHomeDispatchSessionIDsMatchesLCPCompactionViaReportHomeResult` requires it as an assertion; kept `LookupAffinity` unported (never existed locally, no local callers) → skipped `TestSessionAffinitySelectorLookupAffinityProviderAlias`; restored `util.WithSessionID` wrapping in `syncMetadataSessionToContext` (local had the util + `devin_executor` consumer but the call site was dropped — drift restored to upstream end-state); `selectorMu` → local `m.mu.RLock` for selector read.

## Validation
- `2026-09-21T15:10Z` — phase: `p0-kimi-auth-core` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: upstream semantics ported wholesale (exact host lists in `IsKimiAIDomain`/`IsKimiComDomain`, singleflight flight-key `tokenURL:refreshToken` granularity) — reviewed against upstream end-state, not independently re-derived; singleflight refresh is a v7.3.9-era parity gap absorbed here (recorded in Decisions)
  - commands:
    - `/usr/local/go/bin/go test ./internal/auth/kimi/... -count=1` → ok, 6 test funcs green incl. `TestResolveKimiDomainFromAuth` (14 cases), `TestRefreshToken_KimiAIEndpoint`, `TestRefreshToken_CrossDomainIsolation`
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean
    - `/usr/local/go/bin/gofmt -l internal/auth/kimi/` → clean after `gofmt -w`
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, docs/upstream/cliproxyapi-checkpoint.json, docs/upstream/cliproxyapi-ledger-v7.3.9..v7.3.10.md
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: upstream domain-resolution semantics, singleflight refresh dedup correctness
- `2026-09-21T18:55Z` — phase: `p1-kimi-runtime` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session` (worker `b538aa0f` implemented; orchestrator verified union build/test)
  - judge_model: `SWE-2 Max`
  - proof_gaps: `kimi_executor.go` byte-parity claim verified by worker diff vs checkpoint ref, not re-derived in-session; `claude_signing.go` strip machinery intentionally unported (no local counterpart — recorded in Progress); watcher/thinking/registry alias coverage asserted by ported tests only
  - commands:
    - `/usr/local/go/bin/go build ./...` → clean
    - `/usr/local/go/bin/go test ./internal/runtime/executor/... ./internal/thinking/... ./internal/registry/... ./internal/watcher/... -count=1` → all `ok` (executor 3.4s, helps 11.0s)
    - union `go test` (all wave-1 packages) → all `ok`
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.10, a5ab69521f7b
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: executor URL-resolution precedence order on live kimi.ai traffic (unit-tested only)
- `2026-09-21T18:55Z` — phase: `p2-kimi-login-surfaces` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session` (worker `38eccb59`; orchestrator applied `oauth_sessions_test.go` providers hunk)
  - judge_model: `SWE-2 Max`
  - proof_gaps: `commandMode` hunk N/A locally (Postgres-config startup) — flag dispatch verified via `-h` output only; device-flow round-trip not exercised against real `auth.kimi.ai`
  - commands:
    - `/usr/local/go/bin/go test ./sdk/auth/... ./internal/cmd/... ./internal/api/... -count=1` → all `ok` incl. `TestKimiAndKimiAIOAuthRoutes`, `TestProviderRefreshLeads/{kimi,kimi-ai,kimi.ai}`, `TestGuardOAuthSessionPendingForSave`
    - `go build ./cmd/server` + `-h` → shows `-kimi-ai-login` "Login to Kimi.ai using OAuth" + `-kimi-login` "Login to Kimi (.com) using OAuth"
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.10, a5ab69521f7b
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live OAuth device flow against auth.kimi.ai
- `2026-09-21T18:55Z` — phase: `p3-lcp-compaction-engine` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session` (worker `a4e4ca73`)
  - judge_model: `SWE-2 Max`
  - proof_gaps: byte-identical claim verified by worker `diff` vs `refs/upstream-checkpoints/cliproxyapi/v7.3.10` — strongest possible same-session evidence; `LookupSession` remains absent (pre-existing gap, no local callers); `*WithContext` APIs have no callers until p6
  - commands:
    - `/usr/local/go/bin/go test ./sdk/cliproxy/session/... -count=1` → `ok` 0.36s, all 20 new tests green (`-v` verified)
    - `/usr/local/go/bin/go test ./sdk/cliproxy/auth/... -count=1` → `ok` 30.6s regression on existing selector/home_session callers
    - `git diff --check` → clean; `gofmt -l` → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.10, 28100e54b9b5
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: compaction matching on real Claude Code compacted transcripts (unit-tested only)
- `2026-09-21T18:55Z` — phase: `p4-reasoning-tool-turns` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session` (worker `acffd88c`)
  - judge_model: `SWE-2 Max`
  - proof_gaps: absorbed parent-state machinery reviewed as a unit (not line-by-line vs upstream); 7 additional upstream pre-commit reasoning tests not ported (follow-up); `input_image` detail passthrough + `mergeResponsesRequestChatTools` remain divergent (out of scope)
  - commands:
    - `/usr/local/go/bin/go test ./internal/translator/openai/... -count=1` → `ok` (responses 0.068s); 5 new tests pass via `-v -run`
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean; `gofmt -l` → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.10, 40cc6489879a
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: end-to-end reasoning rendering on a live model response
- `2026-09-21T19:25Z` — phase: `p5-kimi-auth-wiring` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: `AvailableProviders`/`HasProviderAuth` are new local API surface (ported because upstream tests exercise them — no local callers yet); candidate-match loops deliberately keep `candidate.Provider` basis rather than upstream `executorKeyFromAuth` (documented deviation); compat-auth (compat_name) routing unchanged
  - commands:
    - `/usr/local/go/bin/go build ./...` → clean
    - `/usr/local/go/bin/go test ./sdk/cliproxy/auth/ -run 'TestManager(ExecutorReturnsRegisteredExecutor|RefreshAndLegacySelectionKimiAliases|SchedulerFastPathKimiAI|KimiDomainIsolation)|TestOAuthModelAliasChannel_Kimi' -v -count=1` → 5/5 PASS incl. cross-domain isolation + fast-path Execute/ExecuteStream
    - `/usr/local/go/bin/go test ./sdk/cliproxy/... -count=1` → all `ok` (auth 30.7s full regression)
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.10, a5ab69521f7b
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: kimi-ai end-to-end scheduling under live traffic
- `2026-09-21T20:10Z` — phase: `p6-node-kind-propagation` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: `LookupAffinity` never existed locally (like p3's `LookupSession` pre-p6) → alias test skipped, API unported; `X-Node-Kind` header path exercised only via unit test on `newAuthDispatchRequest` (no live Home dispatch); local `authDispatchRequest` signature diverges from upstream (4 args vs 8 — parentSessionID/credentialPolicy/excludedAuthIDs come from other upstream machinery not present locally)
  - commands:
    - `/usr/local/go/bin/go test ./sdk/cliproxy/... ./internal/home/... ./internal/redisqueue/... ./internal/logging/... -count=1` → all `ok` (auth 30.7s full regression)
    - `/usr/local/go/bin/go test ./sdk/cliproxy/auth/ -run 'LCPCompaction|WithoutPresetProviderMetadata' -v -count=1` → 4/4 PASS (`MatchesLCPCompaction`, `ViaReportHomeResult`, `WithoutPresetProviderMetadata`, `LCPCompactionPreservesAffinityAndLineage`)
    - `/usr/local/go/bin/go test ./internal/home/ -run TestAuthDispatchRequestIncludesNodeKind -v -count=1` → PASS
    - `/usr/local/go/bin/go build ./...` → clean; `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.10, 28100e54b9b5
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: end-to-end compaction lineage under a live Home deployment
- `2026-09-21T20:15Z` — phase: `p7-final-gate` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: upstream advanced to `v7.3.11` during execution — new delta (8 commits / 19 paths) is pinned-follow-up with blank dispositions, not triaged; full-repo `go test` passed but no live-traffic exercise of kimi.ai or Home compaction lineage
  - commands:
    - `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi` → checkpoint v7.3.11 `ffe6ad3c5fcf`, ref written
    - `python3 .claude/skills/upstream/scripts/upstream_gap.py --slug cliproxyapi` → 19 paths (9 diverged-absent, 10 semantic-review), gap JSON written
    - `python3 .claude/skills/upstream/scripts/upstream_ledger.py --slug cliproxyapi --from v7.3.10 --to v7.3.11` → stub ledger written
    - `/usr/local/go/bin/go test ./... -count=1` → all `ok`/`no test files`, zero failures
    - `PATH=/usr/local/go/bin:$PATH make build` → clean (web embed + `llmhub` binary produced)
    - `git diff --check` → clean; `/usr/local/go/bin/gofmt -l` on all changed `.go` files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.10-parity.md, docs/upstream/cliproxyapi-checkpoint.json, docs/upstream/cliproxyapi-ledger-v7.3.9..v7.3.10.md, docs/upstream/cliproxyapi-ledger-v7.3.10..v7.3.11.md
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live OAuth/execution against auth.kimi.ai + api.kimi.ai; live Home compaction lineage; v7.3.11 delta intentionally untriaged

## Current State and Next Action
- active_phase: none — all phases `checked`, plan complete
- lifecycle_status: completed
- latest_run_id: none
- latest_trace_ids: none
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items:
  - follow-up (not blocking): `v7.3.10..v7.3.11` delta pinned untriaged (`docs/upstream/cliproxyapi-ledger-v7.3.10..v7.3.11.md`, 8 commits — gemini schema array fix, pluginapi usage fields, responses SSE telemetry filter + prewarm, cache-control hoist refactor, claude fallback-credit beta gate, xAI test fix); `LookupAffinity` absent locally; `claude_signing.go` strip machinery unported; 7 upstream pre-commit reasoning tests + `text.format`/`input_image`/`mergeResponsesRequestChatTools` deltas in responses translator
- exact_next_action: next `triage upstream` cycle starts at `v7.3.10..v7.3.11` pinned ledger
