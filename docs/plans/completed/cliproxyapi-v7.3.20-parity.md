---
id: plan-20260927-v7320
type: plan
intake_id: intake-20260927-v7320
lane: high-risk
status: active
created: 2026-09-27
updated: 2026-09-28
---

# Plan: CLIProxyAPI v7.3.15..v7.3.20 targeted parity

## Outcome
- result: llmhub ports the user-approved v7.3.15..v7.3.20 capability slices (21 requirement slices, 33 upstream commits) as independently verifiable semantic ports behind existing registry, executor, auth-conductor, session, logging, management, and translator interfaces — without wholesale merge, external plugin host, or upstream file-split refactors.
- success_signals:
  - Gemini/Antigravity translators: function responses containing JSON-schema `$ref` are stringified, not emitted as raw objects; duplicate/unsanitized tool-call IDs are deduplicated with numeric suffixes while responses still match by raw ID; function-response turns split/merge so responses immediately follow their calls for Claude models; summary visibility honors `reasoning.summary`/`generate_summary` config instead of unconditional auto.
  - Claude translators: adjacent text blocks merge into one assistant message; tool names are sanitized and parameterless tools get a fallback schema.
  - Codex executor: HTTP, compact, and WebSocket requests send `X-Codex-Routing-Hint: model=<resolved-base-model>[;tier=<service_tier>]` on OAuth (not API-key) requests, operator `header:` rules and `override_header` keep precedence; `service_tier` maps fast/priority→priority, ultrafast preserved; compat models strip `author`/`recipient`/internal metadata even with multi-agent optimization off.
  - Claude executor: adaptive-thinking requests on progress-display models gain `thinking.display=updates` and the binding beta, matching upstream post-`4332048d` behavior.
  - xAI executor: `reasoning.effort` survives to upstream (thinking pipeline owns validation); only `stop` is deleted in the request sanitizer.
  - Auth conductor: credential-scoped 429 cooldowns compute deadlines from credential quota state, not per-model state; disabled credentials with `invalid_grant` stop rescheduling; enabled ones back off 1m→30m.
  - Server: `Stop` closes listeners and the HTTP server immediately instead of hanging on open SSE streams; listener/config access is mutex-guarded.
  - Logging/session: request IDs come from an atomic counter, log files get unique sequence suffixes, management log-by-ID picks the newest match; session extraction indexes top-level fields once instead of rescanning multi-megabyte payloads.
  - Devin catalog: `devin_models.json` carries the upstream 56-model set (incl. `claude-opus-5-5`, `gpt-6-*`, `gemini-3-1-pro`, `grok-4-7`, `swe-1-7-lightning`); `swe-1-6-slow` survives as builtin even when remote refresh omits it; grok Devin models report `owned_by: xai`; `fetchDevinModelsFromRemote` keeps the request context alive until the body is read; `splitDevinUID` normalization lands.
  - Meta: minted-key subscription fields (`subs_tier_name`, `subs_tier_id`, `is_subs_active`, `has_payment_method`) persist into auth metadata on login and refresh; `X-Client-Id` header is well-formed and sent; Meta 404 cooldown honors dynamic retry-after.
  - Usage: `usage.Record` carries `RequestID`/`TraceID`; redisqueue payloads emit `execution_id`/`trace_id` preserving legacy `request_id`.
  - File auths: `file_priority` attribute marks file-sourced priorities so refreshes cannot clobber them; watcher retries stat on atomic replace for known auth files.
  - configuration_update: `ModelInfo.SupportConfigurationUpdate` + `CodexModel.support-configuration-update` reach resolved model info; `ApplyRequestThinking` uses `req.Payload` as source with `normalizedUpdatesChanged` propagation.
  - Gemini signatures: no synthetic `skip_thought_signature_validator` is attached when no genuine signature exists.
  - TUI: `--management-base-url` flag and `remote-management.base-url` config select a remote management endpoint in client mode.
  - Each accepted slice lands with focused tests citing its upstream commit(s) and local symbol.
  - `go test` on touched packages, `make build`, `git diff --check`, `gofmt` clean at gates.
  - Newer-than-v7.3.20 upstream delta is pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.20` at `ef9e71201e0c` / `refs/upstream-checkpoints/cliproxyapi/v7.3.20`; `scope_policy` (21 include / 3 exclude) is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.3.15..v7.3.20.json` — 205 paths (match 0, baseline 8, upstream-add-absent 27, diverged-absent 85, upstream-delete-present-local 0, semantic-review 85).
  - `docs/upstream/cliproxyapi-ledger-v7.3.15..v7.3.20.md` — 39 non-merge commits across 5 releases, dispositions recorded.
  - `docs/plans/completed/cliproxyapi-v7.3.15-parity.md` and prior parity plans — their slices must not regress; standing exclusions still hold.
  - `CLAUDE.md` / `docs/WORKFLOW.md` — Postgres-authoritative store, SDK additive-only, no `web/**/*_test.go`, llmhub branding.
- rejected_alternatives:
  - Wholesale merge of the 205-path delta — diverged-absent/semantic-review dominate; upstream splits (`claude_executor_cloaking.go`, `codex_executor_request.go`, `conductor_*.go`, `sdk/pluginapi`) map onto local consolidated files and absent subsystems.
  - Restoring the external plugin host to absorb plugin commits — llmhub deliberately removed it (`internal/pluginhost` is a type-alias shim over `sdk/api/handlers`); rejected as a group.
  - Porting upstream's beta-sequence helper allowlist — local detects title/probe helpers by body shape (`isClaudeTitleHelperRequest`, `helps/claude_diagnostics.go:400`); only the thinking-display residual is ported.
- requirements:
  - R1 [accepted]: **Gemini function-response `$ref` stringify** — `internal/translator/common/gemini.go` gains `ContainsJSONRef`, `SetGeminiFunctionResponseRaw`, `SetGeminiFunctionResponseResult`; applied at every `response.result`/`functionResponse.response` write in antigravity claude/interactions and gemini translators. | source: `5af6cd7582ee` (v7.3.16)
  - R2 [accepted]: **Antigravity tool-ID sanitize + collision dedup** — `openai→antigravity` chat-completions sanitizes tool-call IDs via `SanitizeClaudeToolID`, dedups with `_N` suffixes, and matches `functionResponse`s by raw ID. | source: `580df95a946b` (v7.3.16)
  - R3 [accepted]: **Claude→Responses adjacent text merge** — consecutive `text` content blocks reuse one assistant message output item in stream and non-stream paths of `claude_openai-responses_response.go`. | source: `781a203b99fa` (v7.3.16)
  - R4 [accepted]: **Gemini function-response turn adjacency** — `common/gemini.go` gains `ContentHasGeminiFunctionCall` + `SplitGeminiFunctionResponseTurns`; `antigravity_claude_request.go` splits function-response turns and merges adjacent user turns for Claude models (`util.IsClaudeModel` added to `internal/util/claude_model.go`). | source: `3de5709db14c` (v7.3.19)
  - R5 [accepted]: **Claude tool-name sanitize + fallback schema** — `SanitizeClaudeFunctionName` applied to gemini→claude tool_use names and claude-bound tool declarations; parameterless tools receive a fallback empty-object schema. | source: `75b854eb2e63` (v7.3.17)
  - R6 [accepted]: **Antigravity reasoning summary config** — `enableAntigravityResponsesThinkingSummary` uses `thinking.ExtractSummaryConfig(inputRawJSON, "openai-response")`; defaults to enabled/auto only when effort set and summary unspecified. | source: `6dea3dfa13a9` (v7.3.19)
  - R7 [accepted]: **Codex routing hint** — `X-Codex-Routing-Hint` = `model=<resolved baseModel>[;tier=<service_tier>]` on Execute/ExecuteStream/compact/WebSocket for non-API-key auths; operator `header:` rules win, `override_header` applies last; forwarded hints replaced. | source: `b97f71da0cf0` `dd3b657b5d92` (v7.3.17)
  - R8 [accepted]: **Codex service_tier request translation** — `normalizeCodexServiceTier` maps `fast`/`priority`→`priority`, preserves `ultrafast`, omits absent/default/unsupported in `codex_openai_request.go`. | source: `be4b28f16568` (v7.3.19)
  - R9 [accepted]: **Codex multi-agent compat strip** — `RewriteCodexMultiAgentV2Input` accepts compat flag; compat mode strips `author`/`recipient`/`internal_chat_message_metadata_passthrough` from `agent_message` and message items even when optimization disabled. | source: `3b2882b72eee` (v7.3.19)
  - R10 [accepted]: **Claude 2.1.280 thinking-display residual** — progress-display models default `thinking.display=updates` when caller didn't set it; `thinking-binding-controls` beta triggers on adaptive thinking for those models; port the post-`4332048d` net state only. | source: `f7c738dad38b` (v7.3.17), net of `4332048dda19` (v7.3.18)
  - R11 [accepted]: **xAI reasoning.effort cleanup** — delete `sanitizeXAIResponsesBody` reasoning-effort stripping; sanitizer only removes `stop`; thinking handled before payload overrides. | source: `984aee8df05e` `6d1d74b30d77` (v7.3.17)
  - R12 [accepted]: **Credential-scoped quota isolation** — `MarkResult` 429 path computes credential cooldown from credential quota state (`credentialNext`/`quotaForFailure`), retains `credential_quota` reason, sets `auth.Quota.BackoffLevel`. | source: `aafa4e9549f7` (v7.3.16)
  - R13 [accepted]: **invalid_grant scheduling** — disabled+`invalid_grant` auths unschedule (`hasDisabledInvalidGrantFailure` in refresh check + dirty cleanup); enabled ones back off exponentially 1m→30m; detection accepts status 0/400/401. | source: `58461c8f476a` (v7.3.18)
  - R14 [accepted]: **Immediate server stop** — `Server.Stop` closes listeners + `server.Close()` instead of `Shutdown`; `listenerMu`/`cfgMu` guard `muxBaseListener`/`cfg`; fresh shutdown timeout on exit. | source: `759c57c2612d` (v7.3.17)
  - R15 [accepted]: **Request-ID collision fix** — `GenerateRequestID` uses atomic incrementing 8-hex counter; request log writer uses unique filename with numeric sequence fallback; `GetRequestLogByID` selects newest match by mtime+sequence (`logFileIsNewer`/`parseLogMetadata`). | source: `e6fcfa34bad1` (v7.3.16)
  - R16 [accepted]: **Session-extraction indexing** — `sessionObject` indexes top-level fields once in `hasExplicitSession`/`session info` paths; no rescan per candidate key; duplicate-key support for nested paths. | source: `1dffddf4ac8c` (v7.3.17)
  - R17 [accepted]: **Lifecycle clone avoidance** — read-only lifecycle/interceptor hooks share the request payload; clone only when a hook mutates body (local equivalents: `sdk/api/handlers/handlers.go:635-637,789-791,820`). | source: `c9a06ba6f297` (v7.3.17, non-plugin parts)
  - R18 [accepted]: **Devin catalog sync** — `devin_models.json` matches upstream v7.3.20 catalog; grok Devin models emit `owned_by: xai` (catalog + `cmd/fetch_devin_models`); `swe-1-6-slow` injected as builtin through catalog retrieval/lookup/dynamic update; `splitDevinUID` + thinking-level merge/rank normalization in `internal/registry/devin_models.go` + `helps/devin_models.go`; `fetchDevinModelsFromRemote` cancels after body read and closes bodies on all paths. | source: `65459cace01f` `377747b8b661` `60325f472a2e` `fc2c98d1ca2a` `74c1ebd37f47` (v7.3.16–v7.3.19)
  - R19 [accepted]: **Meta subscription metadata + client-id** — login and refresh persist `subs_tier_name`/`subs_tier_id`/`is_subs_active`/`has_payment_method` into auth metadata; `X-Client-Id` header well-formed (fix `meta_executor.go:57`); Meta 404 honors dynamic retry-after with shortened default cooldown. | source: `7d888bf3cac8` (v7.3.16), `9bdde54b59d1` (v7.3.17)
  - R20 [accepted]: **Usage request/trace IDs** — `usage.Record` gains `RequestID` (per-attempt execution UUID) + `TraceID` (inbound request ID); `UsageReporter` generates/links them; redisqueue emits `execution_id`/`trace_id` preserving `request_id`. | source: `2fe9932bb9d1` (v7.3.16, non-plugin parts)
  - R21 [accepted]: **File-priority preservation + watcher stat retry** — `sdk/cliproxy/auth/priority.go` (`AttributeFilePriority`, `ApplyAuthPriorityMetadata`) wired into filestore read + watcher synthesizer + `syncAuthFilePriorityAttribute`; watcher retries stat (3×25ms) for known auth files before treating rename/remove as deletion. | source: `c404af96ebac` (v7.3.16, non-plugin parts), `5530cbd2a5b0` (v7.3.18, watcher part)
  - R22 [accepted]: **configuration_update support** — `ModelInfo.SupportConfigurationUpdate` (JSON-internal), `CodexModel.support-configuration-update` config flag, `resolvedCodexOAuthModelInfoMetadataKey` attachment incl. `attachResolvedExecutionModelInfo` on restore paths, `ApplyRequestThinking`/`ApplyThinkingWithSourceAndSummary` source=`req.Payload` + `normalizedUpdatesChanged` propagation; eligible Codex models marked in `models.json`. | source: `ed980be34b99` (v7.3.18, non-plugin parts)
  - R23 [accepted]: **Gemini unsigned-signature guard** — `gemini_openai-responses_request.go` attaches `skip_thought_signature_validator` only when a genuine raw signature exists (non-empty check before `openAIResponsesGeminiThoughtSignature`). | source: `107d7f76c9b4` (v7.3.17, translator part)
  - R24 [accepted]: **TUI remote management base URL** — `-management-base-url` flag + `remote-management.base-url` config; TUI client mode uses resolved URL (flag > config > localhost default); config diff watcher covers the new key. | source: `ef9e71201e0c` (v7.3.20)
  - R25 [accepted]: **Final gate** — re-resolve latest upstream release (`upstream_sync.py sync --slug cliproxyapi`), refresh checkpoint; pin any newer-than-v7.3.20 delta as explicit follow-up in ledger + Progress — never silent scope growth. | source: skill contract

## Non-goals
- External plugin host surface — `855a722349ff` (HTTP operation bridge), `6cd86fb272b4` (scheduler terminal rejection), and the pluginhost halves of `c404af96ebac`, `5530cbd2a5b0`, `2fe9932bb9d1`, `ed980be34b99`, `107d7f76c9b4`, `c9a06ba6f297`. llmhub removed the plugin host; `internal/pluginhost` is a type-alias shim. Recorded in `scope_policy.exclude` as `plugin-host-surface`.
- `c7f358d4d46f` Session_id replay header — `already-present`: `antigravity_reasoning_replay.go:303` already probes `Session_id`/`Session-Id`/`X-Session-ID`; residual EqualFold scan is trivial. Recorded as `session-id-replay-header`.
- `4332048dda19` direct-Messages passthrough removal — `superseded-locally`: local has no such bypass; `ShouldCloak` auto-mode cloaks non-`claude-cli` callers by default. Recorded as `direct-messages-passthrough-bypass`.
- Debug-log dedup/baseline logging — `8222e328c02e`, `b4e5cc019d07`, `12245ad38884`: observability-only; pull in when replay/thinking log noise is a reported problem.
- Upstream file-split refactors, `internal/redisqueue` server-side plugin machinery beyond usage fields, branding/docs/gofmt churn — standing exclusions from prior parity cycles.
- The `response_model_rewriter.go` log-noise half of `107d7f76c9b4` — local conductor implements model rewriting inline (`lnForAuth`); cosmetic-only delta.

## Approach and Risks
- approach: twenty-one semantic ports grouped into nineteen phases across six waves. Devin catalog first (Wave 0: mechanical, product-visible). Wave 1 parallelizes the six file-disjoint translator/TUI slices. Wave 2 parallelizes executor + auth-conductor slices. Wave 3 parallelizes infra slices (server stop, logging, session, meta, usage, file-priority, hook clones). Wave 4 lands `configuration_update` alone — it changes `ApplyRequestThinking` signature and `thinking/apply.go`, so it runs after executor/thinking surfaces settle. Wave 5 is the upstream final gate.
- upstream→local surface map (verified during triage):
  - upstream `internal/runtime/executor/claude_executor_{cloaking,request,stream}.go` → local `internal/runtime/executor/claude_executor.go` (consolidated; `applyCloaking` :3579, beta constants :1330-1340, `claudeCodeCLIBetas` :1485+)
  - upstream `codex_executor_{execute,request,stream}.go` + `codex_websockets_{execute,stream}.go` → local `codex_executor.go` (`applyCodexHeaders` :1256), `codex_websockets_executor.go` (`applyCodexWebsocketHeaders` :1532)
  - upstream `sdk/cliproxy/auth/conductor_{cooldown,refresh,selection}.go` → local `sdk/cliproxy/auth/conductor.go` (`MarkResult` 429 path :3864-3904) + `auto_refresh_loop.go` (:365)
  - upstream `internal/pluginhost/*` → no local equivalent (shim only); plugin halves of multi-surface commits are non-goals
  - upstream `internal/config/config_types.go` → local `internal/config/config.go` (CodexModel and remote-management fields)
  - upstream `internal/api/handlers/management/auth_files_{fields,provider_oauth}.go` → local `auth_files.go` (`syncAuthFilePriorityAttribute` :2102) + `auth_files_meta_oauth.go`
- risks:
  - r1: `claude_executor.go` fingerprint code is dense and cloak-sensitive — port `thinking.display`/binding-beta gates verbatim; wrong gating changes the upstream-visible fingerprint.
  - r2: `Server.Stop` switching to `Close()` drops in-flight SSE responses — match upstream semantics (immediate close on stop only; normal request path untouched).
  - r3: `conductor.go` cooldown math is scheduler-critical — port `credentialNext`/`quotaForFailure` separation exactly; regression-test `credential_quota` reason retention and sibling propagation.
  - r4: R22 `ApplyRequestThinking` signature change ripples to ~10 executor call sites — settle those files first (deps on p7/p8/p9), and keep the variadic `normalizedUpdatesChanged` form to minimize churn.
  - r5: `devin_models.json` upstream restructure (aggregation vs per-variant entries) — adopt upstream catalog verbatim, then verify local-only IDs (`claude-haiku-4-5`, `claude-sonnet-4-5`, `gemini-3-flash`, `gpt-4-1`) are upstream-renamed (`MODEL_*` placeholders / aggregated variants) not dropped.
- recovery: per-phase revert of touched files; upstream end-state preserved at `refs/upstream-checkpoints/cliproxyapi/v7.3.20`.

## Phases and Verification

### Wave 0 — sequential gate
- phase: `p0-devin-catalog` — story_id: `story-20260928-p0-devin-catalog` — status: `checked` — deps: none
  - goal: Devin catalog + machinery matches upstream v7.3.20 (R18).
  - tasks:
    - p0-t1 `internal/registry/models/devin_models.json`: sync to upstream v7.3.20 catalog verbatim (`git show refs/upstream-checkpoints/cliproxyapi/v7.3.20:internal/registry/models/devin_models.json`); grok Devin models carry `owned_by: xai` (`377747b8b661`).
    - p0-t2 `internal/registry/devin_models.go`: port builtin-model injection so `swe-1-6-slow` survives remote refresh (`60325f472a2e`); port `splitDevinUID` + thinking-level merge/rank + display-name cleaning (`fc2c98d1ca2a`); register builtin in `model_definitions.go`.
    - p0-t3 `cmd/fetch_devin_models/main.go` + `internal/runtime/executor/helps/devin_models.go`: port UID splitting/normalization and `owned_by` mapping (`377747b8b661`, `fc2c98d1ca2a`).
    - p0-t4 `internal/registry/devin_models_updater.go`: keep request context until body read, close body on all paths (`74c1ebd37f47`).
    - p0-t5 port upstream `devin_models_test.go` + `helps/devin_models_test.go` additions adapted to local types.
  - surfaces: `internal/registry/devin_models*.go`, `internal/registry/models/devin_models.json`, `internal/registry/model_definitions.go`, `cmd/fetch_devin_models/`, `internal/runtime/executor/helps/devin_models*.go`
  - verify: `go test ./internal/registry/... ./internal/runtime/executor/helps/... ./cmd/fetch_devin_models/... -count=1` green; `python3 -c "import json; json.load(open('internal/registry/models/devin_models.json'))"` valid; catalog diff vs upstream tag empty except intentional local-only extras (justify each).
  - stop condition: local-only Devin ID has no upstream rename mapping → halt, list unmapped IDs.

### Wave 1 — parallel phases (translators + TUI, file-disjoint)
- phase: `p1-gemini-common` — story_id: `story-20260928-p1-gemini-common` — status: `checked` — deps: none
  - goal: `$ref` stringify + function-response turn adjacency helpers (R1, R4).
  - tasks:
    - p1-t1 `internal/translator/common/gemini.go`: add `ContainsJSONRef`, `SetGeminiFunctionResponseRaw`, `SetGeminiFunctionResponseResult`, `ContentHasGeminiFunctionCall`, `SplitGeminiFunctionResponseTurns` per `5af6cd7582ee`+`3de5709db14c`.
    - p1-t2 `internal/util/claude_model.go`: add `IsClaudeModel` per `3de5709db14c`.
    - p1-t3 call sites: `antigravity/claude/antigravity_claude_request.go` (result writes :675-710, contents merge :591), `antigravity/interactions/interactions_antigravity_request.go` (:511), gemini-side equivalents — replace `sjson.SetRawBytes` result writes with the new helpers; Claude-model branch uses `SplitGeminiFunctionResponseTurns`+`MergeAdjacentGeminiUserContents`.
    - p1-t4 port upstream test shapes for `$ref` stringify and turn splitting.
  - surfaces: `internal/translator/common/`, `internal/translator/antigravity/{claude,interactions}/`, `internal/util/claude_model*.go`, gemini translator call sites
  - verify: `go test ./internal/translator/... ./internal/util/... -count=1` green; focused test: function response with `$ref` arrives stringified upstream.
  - stop condition: a call site writes `response.result` through a helper not covered by upstream diff → halt, re-diff `5af6cd7582ee` full path list.

- phase: `p2-claude-responses` — story_id: `story-20260928-p2-claude-responses` — status: `checked` — deps: none
  - goal: adjacent text blocks share one assistant message (R3).
  - tasks:
    - p2-t1 `internal/translator/claude/openai/responses/claude_openai-responses_response.go`: skip `finalizeAssistantMessage` when `typ=="text"` (stream :515); reuse `activeMessageItem` for text in non-stream (~:1030) per `781a203b99fa`.
    - p2-t2 port upstream regression test (multi-text-block response yields single message item).
  - surfaces: `internal/translator/claude/openai/responses/claude_openai-responses_response*.go`
  - verify: `go test ./internal/translator/claude/... -count=1` green.
  - stop condition: local stream state machine lacks `finalizeAssistantMessage` boundary semantics → halt, map blocks→items flow first.

- phase: `p3-claude-tools` — story_id: `story-20260928-p3-claude-tools` — status: `checked` — deps: none
  - goal: sanitized tool names + fallback schema for parameterless tools (R5).
  - tasks:
    - p3-t1 `internal/util/`: add `SanitizeClaudeFunctionName` (+ `MapSanitizedFunctionName` if absent) per `75b854eb2e63`.
    - p3-t2 `internal/translator/claude/gemini/claude_gemini_request.go`: sanitize tool_use names (:240); inject fallback `{}`/empty-object input schema for parameterless tool declarations.
    - p3-t3 same sanitization at other claude-bound tool name writes the upstream commit touched.
  - surfaces: `internal/util/`, `internal/translator/claude/gemini/`, other files in `75b854eb2e63` diff
  - verify: `go test ./internal/translator/claude/... ./internal/util/... -count=1` green.
  - stop condition: sanitizer collides with `SanitizeClaudeToolID` semantics → halt, keep both with distinct doc comments.

- phase: `p4-antigravity-openai` — story_id: `story-20260928-p4-antigravity-openai` — status: `checked` — deps: none
  - goal: tool-ID dedup by raw ID + summary-config extraction (R2, R6).
  - tasks:
    - p4-t1 `internal/translator/antigravity/openai/chat-completions/antigravity_openai_request.go`: rawID/sanitized-ID split with `_N` dedup; `functionResponse` matches by raw ID per `580df95a946b`.
    - p4-t2 `internal/translator/antigravity/openai/responses/antigravity_openai-responses_request.go`: `enableAntigravityResponsesThinkingSummary` uses `thinking.ExtractSummaryConfig` + unspecified→enabled/auto default per `6dea3dfa13a9`.
  - surfaces: `internal/translator/antigravity/openai/**`
  - verify: `go test ./internal/translator/antigravity/... ./internal/thinking/... -count=1` green.
  - stop condition: local `thinking.ExtractSummaryConfig` lacks `openai-response` format support → halt, add format branch first.

- phase: `p5-gemini-resp-guard` — story_id: `story-20260928-p5-gemini-resp-guard` — status: `checked` — deps: none
  - goal: no synthetic bypass signature on unsigned parts (R23).
  - tasks:
    - p5-t1 `internal/translator/gemini/openai/responses/gemini_openai-responses_request.go`: `signature` only set when `rawSignature` non-empty (:409 area) per `107d7f76c9b4` translator part; skip attaching `skip_thought_signature_validator` to visibleText/thought parts without genuine signatures.
  - surfaces: `internal/translator/gemini/openai/responses/`
  - verify: `go test ./internal/translator/gemini/... -count=1` green.
  - stop condition: other sites emit the constant unconditionally → halt, audit all `geminiResponsesThoughtSignature` writes.

- phase: `p6-tui-remote` — story_id: `story-20260928-p6-tui-remote` — status: `checked` — deps: none
  - goal: remote management base URL for TUI client mode (R24).
  - tasks:
    - p6-t1 `internal/config/config.go` + `config.example.yaml`: `remote-management.base-url` field per `ef9e71201e0c`.
    - p6-t2 `cmd/server/main.go`: `-management-base-url` flag + `resolveManagementBaseURL` (flag > config > `http://127.0.0.1:{port}`).
    - p6-t3 `internal/tui/{app,client}.go`: `RunWithBaseURL`/client base-URL plumbing; `internal/watcher/diff/config_diff.go` covers the new key.
  - surfaces: `internal/tui/`, `cmd/server/main.go`, `internal/config/config.go`, `internal/watcher/diff/config_diff.go`, `config.example.yaml`
  - verify: `go test ./internal/tui/... ./internal/watcher/... ./internal/config/... -count=1` green; `make build` links.
  - stop condition: local TUI client hardcodes loopback dial → halt, map client construction.

### Wave 2 — parallel phases (executors + conductor, file-disjoint)
- phase: `p7-codex-exec` — story_id: `story-20260928-p7-codex-exec` — status: `checked` — deps: `p0-devin-catalog`
  - goal: routing hint + service_tier + compat metadata strip (R7, R8, R9).
  - tasks:
    - p7-t1 `internal/runtime/executor/codex_executor.go` + `codex_websockets_executor.go`: `applyCodexRoutingHint` (resolved `baseModel` + tier from final body; skip API-key auths; operator `header:` rule wins; `override_header` last; replace forwarded hint) at HTTP/compact/WS call sites per `b97f71da0cf0`+`dd3b657b5d92`.
    - p7-t2 `internal/translator/codex/openai/chat-completions/codex_openai_request.go`: `normalizeCodexServiceTier` + `service_tier` write per `be4b28f16568`.
    - p7-t3 `internal/client/codex/optimize-multi-agent-v2/optimize_multi_agent_v2.go` + `internal/runtime/executor/helps/codex_multi_agent_v2.go`: `isCompat` variadic param, compat-mode metadata strip (`author`/`recipient`/`internal_chat_message_metadata_passthrough`) even when optimize disabled per `3b2882b72eee`; update call sites to pass compat flag.
    - p7-t4 port upstream `codex_executor_routing_hint_test.go` / `codex_websockets_routing_hint_test.go` / `codex_multi_agent_v2_summary_test.go` shapes.
  - surfaces: `internal/runtime/executor/codex_*.go`, `internal/translator/codex/openai/chat-completions/`, `internal/client/codex/optimize-multi-agent-v2/`, `internal/runtime/executor/helps/codex_multi_agent_v2*.go`
  - verify: `go test ./internal/runtime/executor/... ./internal/translator/codex/... ./internal/client/codex/... -count=1` green; header asserted on a recorded request.
  - stop condition: local auth-header-rule mechanism (`codexOperatorHeaderValue` equivalent) absent → halt, trace operator header override path.

- phase: `p8-claude-display` — story_id: `story-20260928-p8-claude-display` — status: `checked` — deps: `p0-devin-catalog`
  - goal: 2.1.280 thinking-display residual (R10).
  - tasks:
    - p8-t1 `internal/runtime/executor/claude_executor.go`: port `claudeModelUsesProgressDisplay` + `applyClaudeCloakThinkingDisplay` semantics — default `thinking.display=updates` for adaptive thinking on progress-display models when caller didn't set it; extend `claudeThinkingBindingBeta` trigger to that case (:1486) per `f7c738dad38b` net of `4332048dda19`.
    - p8-t2 verify `fable-5-1` path (:3648-3657) composes with the generalized default; port relevant upstream test expectations.
  - surfaces: `internal/runtime/executor/claude_executor*.go`
  - verify: `go test ./internal/runtime/executor/... -run 'Claude|Thinking|Display|Beta' -count=1` green.
  - stop condition: upstream gate references model-capability field absent in local registry → halt, name the field.

- phase: `p9-xai-effort` — story_id: `story-20260928-p9-xai-effort` — status: `checked` — deps: `p0-devin-catalog`
  - goal: stop stripping `reasoning.effort` (R11).
  - tasks:
    - p9-t1 `internal/runtime/executor/xai_executor.go`: reduce `sanitizeXAIResponsesBody` (:998) to `sjson.DeleteBytes(body, "stop")` — reasoning.effort left to the thinking pipeline — per `6d1d74b30d77` net of `984aee8df05e`; drop now-unused `xaiSupportsReasoningEffort` if unreferenced (:3300).
  - surfaces: `internal/runtime/executor/xai_executor*.go`
  - verify: `go test ./internal/runtime/executor/... -run 'XAI|Xai' -count=1` green; request with `reasoning.effort` reaches upstream unchanged for capable models.
  - stop condition: local thinking pipeline runs *after* the sanitizer in xAI path → halt, verify ordering before deleting the strip.

- phase: `p10-conductor` — story_id: `story-20260928-p10-conductor` — status: `checked` — deps: `p0-devin-catalog`
  - goal: credential-scope quota isolation + invalid_grant scheduling (R12, R13).
  - tasks:
    - p10-t1 `sdk/cliproxy/auth/conductor.go` `MarkResult` 429 path (:3864-3904): introduce `credentialNext`/`quotaForFailure` — credential cooldown derives from credential quota state, sibling propagation uses `credentialNext`, `auth.Quota` retains `credential_quota` reason + sets `BackoffLevel` per `aafa4e9549f7`.
    - p10-t2 same file + `auto_refresh_loop.go`: `hasDisabledInvalidGrantFailure` check in `nextRefreshCheckAt` (:365) + `dirty` cleanup in `remove`; relaxed `isInvalidGrant` detection (status 0/400/401); exponential backoff 1m→30m for enabled invalid_grant per `58461c8f476a`; find local `refreshErrorFromError`/`isInvalidGrantError` equivalents (:4703-4766).
    - p10-t3 port upstream `conductor_refresh_disabled_test.go` shapes; extend `catalog_credential_quota_regression_test.go` for credentialNext isolation.
  - surfaces: `sdk/cliproxy/auth/conductor*.go`, `sdk/cliproxy/auth/auto_refresh_loop.go`, `sdk/cliproxy/auth/cooldown_*.go`
  - verify: `go test ./sdk/cliproxy/auth/... -count=1` green; focused tests: credential 429 doesn't inherit longer model deadline; disabled invalid_grant never reschedules.
  - stop condition: local cooldown state fields differ (`QuotaState` shape) → halt, map fields before porting math.

### Wave 3 — parallel phases (infra, file-disjoint)
- phase: `p11-server-stop` — story_id: `story-20260928-p11-server-stop` — status: `checked` — deps: `p0-devin-catalog`
  - goal: immediate close on stop (R14).
  - tasks:
    - p11-t1 `internal/api/server.go`: `listenerMu`/`cfgMu` + `getConfig()` accessor; `Stop` closes listeners + `server.Close()` instead of `Shutdown` (:1529-1558) per `759c57c2612d`; `server_reload.go`/`cmd/server` exit path uses fresh timeout context.
    - p11-t2 audit all `s.cfg`/`s.muxBaseListener` readers; route through guards; port `server_stop_test.go` shape.
  - surfaces: `internal/api/server*.go`, `sdk/cliproxy/service_lifecycle*.go` equivalents
  - verify: `go test ./internal/api/... ./sdk/cliproxy/... -count=1` green; manual check: `Stop` returns promptly with an open SSE client.
  - stop condition: graceful drain is load-bearing for a local caller → halt, identify who relies on drain.

- phase: `p12-request-log` — story_id: `story-20260928-p12-request-log` — status: `checked` — deps: `p0-devin-catalog`
  - goal: unique request IDs + latest-log selection (R15).
  - tasks:
    - p12-t1 `internal/logging/requestid.go`: atomic incrementing 8-hex counter per `e6fcfa34bad1` (keep `GenerateRequestID` signature).
    - p12-t2 `internal/logging/request_logger_{streaming,writer}.go` equivalents: unique filename with numeric sequence fallback.
    - p12-t3 `internal/api/handlers/management/logs.go`: `parseLogMetadata`/`logFileIsNewer` — newest match by mtime+sequence (:258-265).
    - p12-t4 port `request_logger_collision_test.go`/`requestid_test.go`/`logs_test.go` shapes.
  - surfaces: `internal/logging/`, `internal/api/handlers/management/logs*.go`
  - verify: `go test ./internal/logging/... ./internal/api/handlers/management/... -count=1` green; two same-ID requests produce two log files, API returns newest.
  - stop condition: local request-ID format consumed downstream (e.g. redisqueue joins) → halt, keep hex shape 8 chars.

- phase: `p13-session-perf` — story_id: `story-20260928-p13-session-perf` — status: `checked` — deps: `p0-devin-catalog`
  - goal: indexed session extraction (R16).
  - tasks:
    - p13-t1 `sdk/cliproxy/session/info.go`: `sessionObject` (fields map + duplicates set) + `newSessionObject` per `1dffddf4ac8c`.
    - p13-t2 `sdk/cliproxy/session/identity.go`: `hasExplicitSession` (:343-388) uses `sessionObject.Get` for root and nested `request`.
    - p13-t3 port `info_performance_test.go`/`info_duplicate_test.go` shapes.
  - surfaces: `sdk/cliproxy/session/{info,identity,lcp}.go`
  - verify: `go test ./sdk/cliproxy/session/... -count=1` green; duplicate top-level key case covered.
  - stop condition: session lookups also scan nested paths beyond top level → halt, keep gjson fallback for nested paths.

- phase: `p14-hook-clone` — story_id: `story-20260928-p14-hook-clone` — status: `checked` — deps: `p0-devin-catalog`
  - goal: skip payload clones for read-only lifecycle hooks (R17).
  - tasks:
    - p14-t1 `sdk/api/handlers/handlers.go`: lifecycle request construction (:635-637, :789-791, :820) shares `payload` until a hook mutates body; mirror upstream lazy-clone pattern from `c9a06ba6f297`.
  - surfaces: `sdk/api/handlers/handlers*.go`, `sdk/cliproxy/session` identity enrichment path
  - verify: `go test ./sdk/api/handlers/... -count=1` green; hook that mutates body still sees isolated copy.
  - stop condition: a local hook mutates payload in place without signaling → halt, add explicit mutation signal or keep clone for that hook.

- phase: `p15-meta` — story_id: `story-20260928-p15-meta` — status: `checked` — deps: `p0-devin-catalog`
  - goal: subscription metadata + well-formed client-id + 404 retry (R19).
  - tasks:
    - p15-t1 `internal/runtime/executor/meta_executor.go`: fix `"X-Client-Id:"` → `"X-Client-Id"` (:57); Meta 404 honors dynamic retry-after with shortened default cooldown per `9bdde54b59d1` (`meta_executor_execute.go:283` `parseMetaRetryAfter` handles 404).
    - p15-t2 `sdk/auth/meta.go`: persist `subs_tier_name`/`subs_tier_id`/`is_subs_active`/`has_payment_method` from `MintedKey` into metadata (:114-129) per `7d888bf3cac8`+`9bdde54b59d1`; preserve on refresh.
    - p15-t3 `internal/api/handlers/management/auth_files_meta_oauth.go` + `internal/auth/meta/meta.go`: same persistence in the OAuth token path; port `auth_files_provider_meta_oauth_test.go` shape.
  - surfaces: `internal/runtime/executor/meta_executor*.go`, `sdk/auth/meta.go`, `internal/auth/meta/`, `internal/api/handlers/management/auth_files_meta_oauth.go`
  - verify: `go test ./internal/runtime/executor/... ./sdk/auth/... ./internal/api/handlers/management/... -run 'Meta' -count=1` green; `req.Header` contains exactly one well-formed `X-Client-Id`.
  - stop condition: subscription fields absent in local `MetaAuthBundle`/`MintedKeyResponse` → halt, they're parsed at `internal/auth/meta/meta.go:80-83` — re-check before flagging.

- phase: `p16-usage-ids` — story_id: `story-20260928-p16-usage-ids` — status: `checked` — deps: `p0-devin-catalog`
  - goal: execution/trace IDs on usage records (R20).
  - tasks:
    - p16-t1 `sdk/cliproxy/usage/manager.go`: `Record` gains `RequestID` (per-attempt UUID) + `TraceID` (inbound request ID); reporter generates/links per `2fe9932bb9d1` (`usage_helpers.go` local emit site).
    - p16-t2 `internal/redisqueue/plugin.go`: emit `execution_id`/`trace_id`, preserve `request_id` (:55,119,144).
    - p16-t3 port upstream usage/plugin test shapes adapted to local record fields.
  - surfaces: `sdk/cliproxy/usage/`, `internal/redisqueue/plugin.go`, `internal/runtime/executor/helps/usage_helpers.go`
  - verify: `go test ./sdk/cliproxy/usage/... ./internal/redisqueue/... -count=1` green; queue payload shows distinct `execution_id` per attempt, shared `trace_id` per request.
  - stop condition: usage consumers (Home report, billing) key on `request_id` semantics → halt, keep `request_id` untouched (upstream preserves it too).

- phase: `p17-file-priority` — story_id: `story-20260928-p17-file-priority` — status: `checked` — deps: `p0-devin-catalog`
  - goal: file-sourced priority survives refresh + watcher stat retry (R21).
  - tasks:
    - p17-t1 `sdk/cliproxy/auth/priority.go` (new): `AttributeFilePriority` + `ApplyAuthPriorityMetadata` verbatim from `c404af96ebac`.
    - p17-t2 `sdk/auth/filestore.go` + `internal/watcher/synthesizer/file.go`: call `ApplyAuthPriorityMetadata`; `SetMetadata` push when inherited.
    - p17-t3 `internal/api/handlers/management/auth_files.go` `syncAuthFilePriorityAttribute` (:2102): set/clear `file_priority` attribute per source backend.
    - p17-t4 `internal/watcher/events.go`: known-auth-file stat retry (3×25ms) before delete handling (:99-100) per `5530cbd2a5b0` watcher part.
    - p17-t5 port `priority_test.go` + watcher retry test shapes.
  - surfaces: `sdk/cliproxy/auth/priority*.go`, `sdk/auth/filestore.go`, `internal/watcher/{events,synthesizer}/`, `internal/api/handlers/management/auth_files.go`
  - verify: `go test ./sdk/cliproxy/auth/... ./sdk/auth/... ./internal/watcher/... -count=1` green.
  - stop condition: local filestore stores priority in Metadata not Attributes → halt, adapt helper to local field ownership.

### Wave 4 — sequential (thinking pipeline settled)
- phase: `p18-config-update` — story_id: `story-20260928-p18-config-update` — status: `checked` — deps: `p7-codex-exec`, `p8-claude-display`, `p9-xai-effort`
  - goal: `configuration_update` support propagation (R22).
  - tasks:
    - p18-t1 `internal/registry/model_registry.go`: `ModelInfo.SupportConfigurationUpdate` + `UnmarshalJSON` aux field per `ed980be34b99`; mark eligible Codex models in `models.json`.
    - p18-t2 `internal/config/config.go`: `CodexModel.SupportConfigurationUpdate` + `config.example.yaml` entry; `model_updater.go` propagation.
    - p18-t3 `internal/thinking/configuration_update.go` (new) + `apply.go`: upstream's configuration_update handling; `ApplyThinkingWithSourceAndSummary` source=`req.Payload` + `normalizedUpdatesChanged` variadic.
    - p18-t4 `internal/runtime/executor/helps/model_capabilities.go`: `ApplyRequestThinking` signature + `ResolvedModelInfo` codex-oauth metadata key (`sdk/cliproxy/auth/api_key_model_capabilities.go`); `attachResolvedExecutionModelInfo` on restore paths; update all `ApplyRequestThinking` call sites.
  - surfaces: `internal/registry/model_*.go`, `internal/registry/models/models.json`, `internal/config/config.go`, `internal/thinking/`, `sdk/cliproxy/auth/api_key_model_capabilities.go`, `internal/runtime/executor/helps/model_capabilities.go`, executor call sites
  - verify: `go test ./internal/registry/... ./internal/thinking/... ./sdk/cliproxy/auth/... ./internal/runtime/executor/... -count=1` green; port `configuration_update_plugin_test.go`/`xai_configuration_update_test.go` shapes minus plugin parts.
  - stop condition: upstream `configuration_update` semantics require a local capability surface that doesn't exist → halt, scope R22 to flag+capability plumbing only.

### Wave 5 — sequential gate
- phase: `p19-final-gate` — story_id: `story-20260928-p19-final-gate` — status: `checked` — deps: all prior phases
  - goal: re-sync upstream, pin residual delta (R25).
  - tasks:
    - p19-t1 `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi`; if checkpoint moved past v7.3.20, regenerate gap+ledger for the new range and record follow-up slices in this plan's Progress + checkpoint `scope_policy.defer`.
    - p19-t2 full gates: `go test ./...` on touched packages, `make build`, `gofmt -l .`, `git diff --check`.
  - surfaces: `docs/upstream/**`, this plan
  - verify: checkpoint JSON reflects latest tag; gap JSON for any new range written.
  - stop condition: upstream released >3 new versions mid-work → halt, keep scope at v7.3.20 and record the delta as next-cycle initiative.

## Progress
- `2026-09-28` — wave 0, task p0-t1. task_status: `DONE`. summary: `internal/registry/models/devin_models.json` replaced with upstream v7.3.20 catalog verbatim — byte-identical to `refs/upstream-checkpoints/cliproxyapi/v7.3.20`; 56 models, valid JSON; all local-only IDs mapped to upstream renames (`claude-haiku-4-5`→`MODEL_PRIVATE_11`, `claude-sonnet-4-5`→`MODEL_PRIVATE_2/3`, `gemini-3-flash`→`gemini-3-8-flash` resolve, `gpt-4-1`→`MODEL_CHAT_GPT_4_1_2025_04_14`, `swe-1-6-slow`→builtin); grok models carry `owned_by: xai`.
- `2026-09-28` — wave 0, task p0-t2. task_status: `DONE`. summary: `internal/registry/devin_models.go` was byte-identical to upstream base → clean drop-in of upstream v7.3.20 (`splitDevinUID`, thinking-level merge/rank, display-name cleaning, builtin injection via `WithDevinBuiltins`, `devinBuiltinSWE16SlowModelInfo`); builtin registered as first element of `staticDevinModels` in `model_definitions.go`; `go build` green.
- `2026-09-28` — wave 0, task p0-t3. task_status: `DONE`. summary: `cmd/fetch_devin_models/main.go` semantic port (llmhub client-metadata bytes preserved): `vendorName(id, uid)` with grok→`xai` fallback, `splitDevinUID` with compound `-fast`/`-priority`/`-1m`/`_EFFORT` suffixes and `swe-1-6` special-casing, aggregation seeds levels from registry + skips `priority`, display-name suffix stripping, `sortLevels` rank + lexical tie-break; `helps/devin_models.go` ported new `knownDevinSuffixes`, `MODEL_GPT_5_2`/`MODEL_GOOGLE_GEMINI_3_0_FLASH`/`MODEL_CLAUDE_4_5_OPUS` normalized-underscore UIDs, `glm-5-2-1m`/`claude-*-4-6[-1m]` switch cases.
- `2026-09-28` — wave 0, task p0-t4. task_status: `DONE`. summary: `internal/registry/devin_models_updater.go` drop-in of upstream fix — request context kept until `io.ReadAll` + `Body.Close` complete; cancel on error paths only; byte-identical to upstream v7.3.20.
- `2026-09-28` — wave 0, task p0-t5. task_status: `DONE`. summary: ported upstream tests — `cmd/fetch_devin_models/main_test.go` (new, `TestSplitDevinUID`, `TestAggregateModels_MergesThinkingVariants_Issue6141`, `TestAggregateModels_PreservesCatalogThinkingLevels`); registry test file gained Issue6095 cancel-ordering, Issue6111 builtin persistence, `TestWithDevinBuiltins`, Issue6141 no-suffix + aggregation tests; helps test gained 4-6/1m/`MODEL_*` table cases; local `TestDevinModelsRemoteFetchFallback` updated to expect builtin retained post-update (matches upstream test change).
- `2026-09-28` — wave 0, phase-gate. task_status: `DONE`. summary: `go test ./internal/registry/... ./internal/runtime/executor/helps/... ./cmd/fetch_devin_models/... -count=1` all pass; `devin_models.json` valid + byte-identical to upstream tag (zero unmapped local-only IDs); `devin_models.go` + `devin_models_updater.go` byte-identical to upstream v7.3.20; `gofmt` + `go vet` clean. Phase → `checked`.
- `2026-09-28` — wave 1, task p1-t1. task_status: `DONE`. summary: `internal/translator/common/gemini.go` gained `ContentHasGeminiFunctionCall`, `SplitGeminiFunctionResponseTurns`, `ContainsJSONRef`, `SetGeminiFunctionResponseResult`, `SetGeminiFunctionResponseRaw` ported verbatim from `5af6cd7582ee`+`3de5709db14c` (`JoinRawArray` already local in `bytes.go`).
- `2026-09-28` — wave 1, task p1-t2. task_status: `DONE`. summary: `internal/util/claude_model.go` gained `IsClaudeModel` (case-insensitive "claude" substring).
- `2026-09-28` — wave 1, task p1-t3. task_status: `DONE`. summary: call sites ported — `antigravity_claude_request.go` 4 `response.result` writes → new helpers + `IsClaudeModel` branch (`SplitGeminiFunctionResponseTurns`+`MergeAdjacentGeminiUserContents` for Claude, `MergeAdjacentGeminiContents` otherwise); `gemini_claude_request.go:181` + `gemini_openai-responses_request.go:1709,1718` → helpers. Upstream `interactions/*` call sites rejected — llmhub has no interactions translator dirs.
- `2026-09-28` — wave 1, task p1-t4. task_status: `DONE`. summary: ported upstream tests — `TestSplitGeminiFunctionResponseTurns`/`TestContainsJSONRef`/`TestSetGeminiFunctionResponseResult` (common), `TestIsClaudeModel` (util), `TestConvertClaudeRequestToAntigravity_FunctionResponseJSONRef` + 5 adjacency tests (antigravity/claude), `TestConvertClaudeRequestToGemini_FunctionResponseJSONRef` (gemini/claude), `TestConvertOpenAIResponsesRequestToGemini_FunctionResponseJSONRef` (gemini/openai/responses). All pass.
- `2026-09-28` — wave 1, phase-gate p1. task_status: `DONE`. summary: `go test ./internal/translator/... ./internal/util/... -count=1` — all 26 test packages green, `gofmt`/`go build` clean; `$ref`-containing function responses verified stringified by ported tests. Phase → `checked`.
- `2026-09-28` — wave 1, task p2-t1. task_status: `DONE`. summary: `claude_openai-responses_response.go` ported `781a203b99fa` — stream path skips `finalizeAssistantMessage` on `typ=="text"`; non-stream reuses `activeMessageItem` across consecutive text blocks, nils it on non-text, registers `blockToItem[idx]` on reuse.
- `2026-09-28` — wave 1, task p2-t2. task_status: `DONE`. summary: new `claude_openai-responses_citations_test.go` dropped in from upstream verbatim (`TestClaudeResponsesAdjacentCitedTextBlocksShareMessage`, stream+nonstream) — passes; local test helpers (`translateClaudeResponsesStreamThroughRegistry`, `parseClaudeResponsesSSEEvent`) satisfied it.
- `2026-09-28` — wave 1, phase-gate p2. task_status: `DONE`. summary: `go test ./internal/translator/claude/... -count=1` green. Phase → `checked`.
- `2026-09-28` — wave 1, task p3-t1. task_status: `DONE`. summary: `internal/util/claude_tool_id.go` gained `SanitizeClaudeFunctionName` + `claudeFunctionNameSanitizer` (`^[a-zA-Z0-9_-]{1,64}$`, `_` fallback) per `75b854eb2e63`; distinct from `SanitizeClaudeToolID` (ID regex, generated fallback).
- `2026-09-28` — wave 1, task p3-t2+t3. task_status: `DONE`. summary: sanitized tool names at all upstream-touched writes — `claude_gemini_request.go` (toolUse + anthropicTool name, fallback `{"type":"object","properties":{}}` input_schema for parameterless decls), `claude_openai_request.go` (toolUse name, allowedToolNames dual-key, sanitized decl/tool_choice names, `NormalizeClaudeToolInputSchema(nil)` fallback), `claude_openai-responses_request.go` (toolUse, tool_choice, function+custom tool decl names + fallback template). Interactions file N/A. Bonus parity: ported upstream `setClaudeToolChoiceFromGeminiToolConfig` (`tool_config`/`toolConfig` + ANY-with-single-`allowedFunctionNames`→`{"type":"tool"}`) — local previously mode-only, required by ported test.
- `2026-09-28` — wave 1, phase-gate p3. task_status: `DONE`. summary: `go test ./internal/translator/claude/... ./internal/util/... -count=1` green incl. `TestSanitizeClaudeFunctionName`, `TestConvert{Gemini,OpenAI}RequestToClaude_SanitizesToolNamesAndProvidesFallbackSchema`. Phase → `checked`.
- `2026-09-28` — wave 1, task p4-t1. task_status: `DONE`. summary: `antigravity_openai_request.go` ported `580df95a946b` — `assistantToolCall` gained `rawID`; tool-call IDs sanitized via `SanitizeClaudeToolID` + `_N` dedup loop (`usedToolCallIDs`); `functionResponse` matches tool responses by `rawID`. Ported 3 upstream dedup tests (adapted `signature.`→`internalsignature.`, added `regexp` import).
- `2026-09-28` — wave 1, task p4-t2. task_status: `DONE`. summary: `antigravity_openai-responses_request.go` ported `6dea3dfa13a9` — `enableAntigravityResponsesThinkingSummary` now uses `thinking.ExtractSummaryConfig(inputRawJSON, "openai-response")` with unspecified→enabled/auto default and applies resolved config (explicit `summary`/`generate_summary`/`null`/`none` now honored). Test table updated per upstream (+none case); `helps` test addition N/A — `ApplyThinkingWithSourcePayload` does not exist locally.
- `2026-09-28` — wave 1, phase-gate p4. task_status: `DONE`. summary: `go test ./internal/translator/antigravity/... ./internal/thinking/... -count=1` green. Phase → `checked`.
- `2026-09-28` — wave 1, task p5-t1. task_status: `DONE`. summary: `gemini_openai-responses_request.go` ported `107d7f76c9b4` translator part — `signature` only computed when `rawSignature` non-empty; `hasRealSignature` gates all `thoughtSignature` writes on visibleText/thought parts; empty reasoning+text carrier returns nil; function-call fallback keeps bypass constant via `funcSig`. Audit: all `geminiResponsesThoughtSignature` sites byte-match upstream post-fix state. Ported `TestConvertOpenAIResponsesRequestToGemini_UnsignedModelTextDoesNotSynthesizeBypassSignature` — passes.
- `2026-09-28` — wave 1, phase-gate p5. task_status: `DONE`. summary: `go test ./internal/translator/gemini/... -count=1` green. Phase → `checked`.
- `2026-09-28` — wave 1, task p6-t1..t3. task_status: `DONE`. summary: ported `ef9e71201e0c` — `RemoteManagement.BaseURL` (`yaml:"base-url,omitempty" json:"base-url,omitempty"`); `-management-base-url` flag + `resolveManagementBaseURL` (flag > config > `http://127.0.0.1:{port}`, default 8317); `NewClientWithBaseURL`/`BaseURL()`/`NewAppWithBaseURL`/`RunWithBaseURL` with scheme-prefix + trailing-slash normalization; client-mode `tui.Run`→`RunWithBaseURL`; `config_diff` emits `remote-management.base-url` via local `formatProxyURL` (upstream `formatURL` equivalent). `config.example.yaml` N/A — file does not exist locally (Postgres-first config).
- `2026-09-28` — wave 1, phase-gate p6. task_status: `DONE`. summary: `go test ./internal/tui/... ./internal/watcher/... ./internal/config/... ./cmd/server/ -count=1` green incl. ported `TestNewClientWithBaseURL`, `TestClient_RemoteServerInteraction`, `TestNewAppWithBaseURL`, `TestResolveManagementBaseURL`, `TestParseConfigBytesRemoteManagementBaseURL`, `TestBuildConfigChangeDetails_RemoteManagementBaseURL`; `go build ./...` links. Phase → `checked`. **Wave 1 complete (6/6).**
- `2026-09-28` — wave 2, task p7-t1..t4. task_status: `DONE`. summary: ported `b97f71da0cf0`+`dd3b657b5d92` — `codexRoutingHintHeader`/`applyCodexRoutingHint`/`codexOperatorHeaderValue` in `codex_websockets_executor.go` (resolved `baseModel` + tier from final body; API-key auths skipped; operator `header:` rule wins via `ApplyCustomHeadersFromAttrs` resolver; forwarded client hint replaced); wired at 6 call sites (Execute/alphaSearch/compact/stream + ws Execute/ws prepareStream; alphaSearch is llmhub-only, same semantics). `be4b28f16568` — `normalizeCodexServiceTier` (fast|priority→priority, ultrafast preserved) in `codex_openai_request.go`. `3b2882b72eee` — `RewriteCodexMultiAgentV2Input` variadic `isCompat` → `compatMode`/`optimizeEnabled`; `rewriteCodexAgentMessageInput(payload, optimizeEnabled, compatMode)` strips `author`/`recipient`/`internal_chat_message_metadata_passthrough` on all items in compat mode even when optimize disabled; helps call sites pass `isCompat`/`true`. Pre-range deps absorbed (R8 test parity): `codex_claude_request.go` service_tier block + speed→priority (`893412e9`+`da087d7f`, pre-range) and `codex_gemini_request.go` `normalizeGeminiCodexServiceTier`; ws-stream `SetStringIfDifferent(body,"model",baseModel)` added to match upstream body-model guarantee. Ported tests: `codex_executor_routing_hint_test.go`, `codex_websockets_routing_hint_test.go` (DisableCodexCloaking adapted → `cliproxyexecutor.WithCodexCloakingDisabled`), `TestConvertOpenAIRequestToCodexServiceTier`, `TestRewriteCodexMultiAgentV2Input_StripsAuthorAndRecipient_Issue6136` (config field adapted → `CodexOptimizeMultiAgentV2`). `TestCodexExecutor_IsCompat_StripsAuthorAndRecipient_Issue6136` N/A — `codexSpawnAgentTestPayload`/`codexSpawnAgentTestContext` fixtures are pre-range upstream, absent locally.
- `2026-09-28` — wave 2, phase-gate p7. task_status: `DONE`. summary: `go test ./internal/runtime/executor/... ./internal/translator/codex/... ./internal/client/codex/... -count=1` green (10 pkgs). Phase → `checked`.
- `2026-09-28` — wave 2, task p8-t1..t2. task_status: `DONE`. summary: ported `f7c738dad38b` display residual (net of superseded `4332048d` passthrough) — `isClaudeOpus55Model`/`isClaudeSonnet5Model`/`claudeModelUsesProgressDisplay`/`applyClaudeCloakThinkingDisplay`; `claudeModelHasPerTurnEffort` redefined on `isClaudeOpus55Model`; `claudeThinkingBindingBeta` auto-includes for progress-display models with adaptive thinking; `claudeIncludeMidConvClearAt` auto-includes for progress-display models; fable reconcile gained fallback retarget switch (fable→opus-5, opus-5-5→opus-4-8), opus-5-5 fallback injection, thinkingActive-gated injectedDisplay removal, `applyClaudeCloakThinkingDisplay` in both branches; setup path injects opus-5-5 fallback + generalized display fill. Scope-limited: utf16 fingerprint fix, turn_origin, Messages-passthrough context keys NOT ported (superseded-local per triage). Ported `claude_executor_cloaking_display_test.go` verbatim (imports rewritten); updated 3 pre-existing local tests to upstream v7.3.20 expectations (clear-at auto-beta; sonnet-5 is now progress-display → non-progress case moved to opus-4-8).
- `2026-09-28` — wave 2, phase-gate p8. task_status: `DONE`. summary: `go test ./internal/runtime/executor/ -count=1` green (full package). Phase → `checked`.

## Decisions
- `devin_models.go`, `devin_models_updater.go` were byte-identical to upstream v7.3.15 → adopted upstream v7.3.20 files wholesale (no module-path imports present).
- `cmd/fetch_devin_models/main.go` kept local drift (inlined `buildDevinClientMetadataBytes` + local module path) — upstream delta ported semantically, not overwritten.
- Devin remote URL kept at upstream `router-for-me/models` raw URL — same as pre-change local value; intentional (canonical catalog source).
- Upstream `interactions` translator call sites (`interactions_antigravity_request.go`, `interactions_gemini_common.go`, `interactions_gemini_response.go`) rejected — llmhub has no `interactions` translator directories; no equivalent sink for the `$ref` helper.
- p3: ported upstream `setClaudeToolChoiceFromGeminiToolConfig` (ANY + single `allowedFunctionNames` → `{"type":"tool","name":…}`, plus `toolConfig`/`functionCallingConfig` camelCase read) — pre-existing upstream feature local lacked; required by the ported regression test rather than weakening assertions.
- p4: upstream `helps/thinking_test.go` addition (`TestApplyThinkingWithSourcePayload_AntigravityResponsesReasoningSummaryAuto`) rejected — `ApplyThinkingWithSourcePayload` API does not exist locally (upstream-only helps surface).
- p5: `107d7f76` `response_model_rewriter.go` debug-log-noise half deferred — belongs with deferred logging commits (`8222e328`, `b4e5cc01`, `12245ad3`); `pluginhost/host.go` half rejected (no external plugin host).
- p6: `config.example.yaml` edit skipped — file does not exist locally (llmhub is Postgres-config-first); `formatURL` mapped to local `formatProxyURL` (same redaction semantics).
- p7: pre-range upstream service_tier blocks in `codex/claude` (`893412e9` normalize + `da087d7f` speed→priority) and `codex/gemini` normalizers adopted at v7.3.20 state — required for in-range R7/R8 parity (routing-hint `;tier=` suffix and ported upstream tests); recorded as pre-range dependency absorption, not scope growth. Executor-level `...IsCompat_StripsAuthorAndRecipient` test rejected — its `codexSpawnAgentTest*` fixtures belong to a pre-range upstream commit absent locally.
- p8: `f7c738da` ported display-only — utf16 `computeFingerprint` fix, `turn_origin` billing field, `claudeInboundFormat`/`claudeDirectMessagesPassthrough` ctx keys, `claude_client_detection` changes excluded per triage (`4332048d` superseded-locally; direct-Messages passthrough does not exist in this fork). `claudeModelHasPerTurnEffort` semantics narrowed per upstream (opus-5-5 prefix → exact/`[` form).
- p9: two pre-range upstream dependencies absorbed to make the ported test semantics hold: (1) `modelFamilyMismatch` clamp logic in `internal/thinking/validate.go` (from pre-v7.3.15 `db4f1cef`, required by in-range `ed980be3` resolved-model-info path); (2) `"xai"` removed from `isOpenAIFamily` — upstream never classified xAI as openai-family; it is a protocol-reuse provider, so resolved `Type:"xai"` metadata triggers the mismatch clamp rather than same-family strict rejection. Unsupported-model strip widened to whole `reasoning` object for `codex`/`xai` per upstream net state. `grok-build-0.1` unsupported-model fixture adapted to `grok-4` — local catalog legitimately gives grok-build thinking levels (fork catalog drift preserved).
- p14: upstream's interceptor surface (`handlers_interceptors.go`, `conductor_execution.go`, `executor/types.go`, `pluginapi`, `pluginhost`) is absent locally — N/A, consistent with pluginhost rejection. Local equivalent is the in-process `RequestLifecyclePlugin` surface in `sdk/api/handlers/handlers.go` + `lifecycle.go`: applied the same share-don't-clone pattern at all four payload sites plus `Enrich` OriginalRequest sharing; documented read-only contract on `RequestLifecycleRequest.Payload`.

## Validation
- `go test ./internal/registry/... ./internal/runtime/executor/helps/... ./cmd/fetch_devin_models/... -count=1` → 3 packages ok (2026-09-28).
- `python3 -c "import json; json.load(open('internal/registry/models/devin_models.json'))"` → valid.
- `diff <(git show refs/upstream-checkpoints/cliproxyapi/v7.3.20:internal/registry/models/devin_models.json) internal/registry/models/devin_models.json` → empty (byte-identical).
- `diff` of `devin_models.go`, `devin_models_updater.go` vs upstream tag → empty.
- `go test ./internal/translator/... ./internal/util/... -count=1` → 26 packages ok, zero failures (2026-09-28, p1).
- `go test ./internal/translator/claude/... -count=1` → green incl. `TestClaudeResponsesAdjacentCitedTextBlocksShareMessage` (2026-09-28, p2).
- `go test ./internal/translator/claude/... ./internal/util/... -count=1` → green incl. `TestSanitizeClaudeFunctionName` + both `...SanitizesToolNamesAndProvidesFallbackSchema` tests (2026-09-28, p3).
- `go test ./internal/translator/antigravity/... ./internal/thinking/... -count=1` → green incl. 3 `SanitizesClaudeToolIDs*` dedup tests + updated `ReasoningSummaries` table (2026-09-28, p4).
- `go test ./internal/translator/gemini/... -count=1` → green incl. `UnsignedModelTextDoesNotSynthesizeBypassSignature` (2026-09-28, p5).
- `go test ./internal/tui/... ./internal/watcher/... ./internal/config/... ./cmd/server/ -count=1` → green; `go build ./...` links (2026-09-28, p6).
- `go test ./internal/runtime/executor/... ./internal/translator/codex/... ./internal/client/codex/... -count=1` → 10 pkgs green incl. routing-hint HTTP/WS suites + service-tier + compat-strip tests (2026-09-28, p7).
- `go test ./internal/runtime/executor/ -count=1` → full package green incl. `TestReconcileClaudeCloakDisplay*` + `TestClaudeCodeCLIBetas_21280GatedBetas` (2026-09-28, p8).
- `go test ./internal/runtime/executor/... ./internal/thinking/... ./internal/translator/... -count=1` → all green incl. `TestXAIExecutorThinkingPayloadOverride` (11 cases ×2 overrides) + unsupported-model `reasoning` strip; `go build ./...` links (2026-09-28, p9).
- `go test ./sdk/cliproxy/... -count=1` → 6 packages green incl. 8 `MarkResult_*` monotonic/credential-isolation tests + 5 `RefreshAuthForRequest_*`/`AutoRefreshLoop_*` invalid_grant tests (2026-09-28, p10).
- `go test ./internal/api/... ./sdk/cliproxy/... -count=1` → all green incl. `TestServerStop_ViolentShutdownImmediatelyClosesWithoutContextDeadlineError`, `TestServiceShutdown_ViolentShutdownImmediatelyClosesServer`, `TestServiceRun_ShutdownContextFreshAtExit` (2026-09-28, p11).
- `go test ./internal/logging/... ./internal/api/handlers/management/... -count=1` → green incl. 5 `GetRequestLogByID_SelectsLatest*` + collision/requestid tests (2026-09-28, p12).
- `go test ./sdk/cliproxy/session/... -count=1` → green incl. `DoesNotRescanLargePayload` perf tests + `DuplicateMetadataKeysPreserveSessionLookup` (2026-09-28, p13).
- `go test ./sdk/api/handlers/... ./sdk/cliproxy/session/... -count=1` → green incl. `TestEnrich_DoesNotClonePayloadWhenPopulatingOriginalRequest_Issue6101` (2026-09-28, p14).
- `go test ./internal/runtime/executor/ ./internal/api/handlers/management/ ./sdk/auth/ ./sdk/cliproxy/auth/ -count=1` → green incl. 6 `*_Issue6117` meta executor tests + 2 `BuildMetaAuthRecord_*` + `MetaLoginPreservesMintSubscriptionMetadata` (2026-09-28, p15).
- `go test ./sdk/cliproxy/usage/... ./internal/redisqueue/... ./internal/runtime/executor/helps/ -count=1` → green incl. `TestUsageQueuePlugin_ExecutionIDAndTraceID`, `TestUsageQueuePlugin_StrictLegacyRequestIDPreservation`, `TestUsageReporter_AllocatesUniqueUUIDPerAttemptWithSharedTraceID`, `TestUsageReporter_ExplicitTraceIDPrecedenceOverLogRequestID`; `go build ./...` links (2026-09-28, p16).
- `2026-09-28` — wave 3, task p16-t1..t3. task_status: `DONE`. summary: ported `2fe9932bb9d1` non-plugin half — `usage.Record` gained `RequestID`/`TraceID`; `WithExecutionRequestID`/`ExecutionRequestIDFromContext`/`WithTraceID`/`TraceIDFromContext` context helpers; `Manager.Publish` fills missing IDs (ctx execution ID → fresh uuid4; ctx trace ID → `logging.GetRequestID`). `UsageReporter` generates one uuid4 per reporter (execution/attempt) and derives traceID from usage ctx → logging request ID; `buildAdditionalModelRecord` re-mints `RequestID` per upstream; `RequestID()`/`TraceID()` accessors added. `redisqueue` payload emits `execution_id`/`trace_id` (omitempty), `request_id` strictly preserved from `GetRequestID(ctx)`; fallbacks mint uuid4/trace=requestID. N/A: `sdk/pluginapi`/`internal/pluginhost` usage adapters — subsystem absent locally (rejected plugin-host surface). 4 upstream tests ported, adapted to local `withEnabledQueue` + unaliased `logging` import.
- `go test ./internal/watcher/... ./sdk/auth/ ./sdk/cliproxy/auth/ ./internal/api/handlers/management/ -count=1` → all green incl. `TestHandleEventAtomicReplaceDelayedStatPreservesClient`, 3 `ApplyAuthPriorityMetadata_*` upstream tests + 3 local-equivalent file-priority tests; `go build ./...` links (2026-09-28, p17).
- `2026-09-28` — wave 3, task p17-t1..t5. task_status: `DONE`. summary: ported `c404af96ebac` + `5530cbd2a5b0` non-plugin halves — new `sdk/cliproxy/auth/priority.go` (`AttributeFilePriority`, `ApplyAuthPriorityMetadata`) verbatim; synthesizer manual priority block → `ApplyAuthPriorityMetadata(a, metadata)`; filestore `readAuthFile` applies priority metadata + `SetMetadata` push via existing `metadataSetter` interface; `syncAuthFilePriorityAttribute` clears/sets `file_priority` (local file-source marker = `Attributes["path"]`, upstream's `AttributeSourceBackend`/`AuthSourceFile` constants are pluginhost-only and absent). Watcher `handleEvent` gained 3×25ms stat retry for known auth files on atomic-replace delete. N/A: upstream filestore/synthesizer tests exercise `RegisterPluginAuthParser`/`pluginapi` (rejected surface) — wrote local equivalents instead. `auth_files_refresh_test.go` (150 lines) is pluginhost refresh-flow — rejected.
- `2026-09-28` — wave 4, task p18-t1..t4. task_status: `DONE`. summary: ported `ed980be34b99` non-plugin half end-to-end — `SupportConfigurationUpdate` on `ModelInfo`/`CodexModel`/`OpenAICompatibilityModel`(+`IsCompat`)/`VertexCompatModel`(+`Thinking`) + `models.json` marks; `internal/thinking/configuration_update.go` + apply.go source-payload/summary pipeline (semantic port without upstream's `summaryConfig`/`sourceBody` wrappers — local signature kept); `mapConfiguredHighIntent` xhigh→max semantic remap + `normalizeUserDefinedConfig`; `ExtractReasoningEffort`/`extractCodexUsageConfig`/`extractSourceThinkingConfig`/`ExtractTranslatedReasoningEffort`/`extractThinkingConfigForUsage`; `summary.go` replaced with upstream (summary-visibility intent) + `ApplyTranslatedSummaryToClaude` wired into the three claude translator conversion returns; translator registry gained `configurationUpdates`/`updatesChanged`/`ConfigurationUpdatesChanged` envelope flag + ported `Unregister`/`HasStream|NonStreamResponseTransformer`/`hasAnyResponseTransform` + pluginhooks response-layer wiring in `TranslateStream`/`TranslateNonStream`; `sdk/cliproxy/auth/api_key_model_capabilities.go` cfg-snapshot machinery (API-key/OAuth/Home resolved-model-info attach) + `classification.go` helpers; conductor selection-vs-execution model split (`AuthSelectionModelMetadataKey`, `executionModelForAuthSelection`, restore-path capability binding, Home metadata gating on `!restoreExecutionModel`, legacy-Home capability inheritance ordering); `service.go` `buildCodexConfigModels` empty-list Pro-defaults (update=false) + apikey short-circuit; helps `ApplyRequestThinking` update-intent variants; codex/websockets/openai-compat/xai executor call sites. ~25 upstream test shapes ported/adapted (api-key capabilities, home config-update, codex service models, xai, management, summary/registry test files replaced with upstream final forms, thinking codex-usage/configured-api-key suites). N/A: `configuration_update_plugin_test.go` + `test/summary_intent_translation_test.go` need `FormatInteractions`/pluginhost — rejected surface; kimi K2.8 clamp test is post-range.
- `2026-09-28` — wave 4, phase-gate p18. task_status: `DONE`. summary: `go test ./...` (whole repo) all green incl. 15 `home_configuration_update` tests, codex service/compat suites, summary + registry suites; `go build ./...` clean; `gofmt`/`go vet` clean; no stray debug code. Phase → `checked`.
- `2026-09-28` — wave 5, task p19-t1. task_status: `DONE`. summary: `upstream_sync.py sync --slug cliproxyapi` → checkpoint moved `v7.3.20` → `v8.0.3 acdace936fa7` (4 releases). Stop condition triggered (>3 new versions mid-work): scope stays pinned at v7.3.20. Regenerated `docs/upstream/cliproxyapi-gap-v7.3.20..v8.0.3.json` (938 changed paths; 490 semantic-review, 430 diverged-absent, 14 upstream-add) + `cliproxyapi-ledger-v7.3.20..v8.0.3.md` (14 non-merge commits) as next-cycle artifacts; recorded defer in checkpoint `scope_policy.defer`.
- `2026-09-28` — wave 5, task p19-t2 + final-gate. task_status: `DONE`. summary: `go test ./...` whole-repo green; `make build` green (web embed + `llmhub` binary 63MB, version `v0.0.48-dirty`); `git diff --check` clean; `gofmt -l` — 4 touched files formatted (`internal/config/config.go`, `vertex_compat.go`, `sdk/api/handlers/lifecycle.go`, `sdk/cliproxy/usage/manager.go`), 37 pre-existing unformatted files left as fork drift. Phase → `checked`. All 20 phases checked — plan complete.

## Current State
- active_phase: none — all phases `checked`
- lifecycle_status: complete
- run_anchor: 2026-09-28
- blockers: none
- exact_next_action: none — plan validated; deferred range `v7.3.20..v8.0.3` queued for next-cycle triage (see `docs/upstream/cliproxyapi-gap-v7.3.20..v8.0.3.json`)
