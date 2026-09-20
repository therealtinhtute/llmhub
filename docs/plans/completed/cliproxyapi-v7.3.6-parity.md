---
id: plan-20260918-v736
type: plan
intake_id: intake-20260918-v736
lane: high-risk
status: active
created: 2026-09-18
updated: 2026-09-18
---

# Plan: CLIProxyAPI v7.3.4..v7.3.6 full parity

## Outcome
- result: llmhub ports the owner-approved v7.3.4..v7.3.6 capability slices as independently verifiable semantic ports behind existing executor, translator, auth, registry, config, and util interfaces — including the full antigravity/gemini OpenAI-Responses web-search feature — without wholesale merge or branding churn.
- success_signals:
  - Each accepted slice lands with focused tests citing its upstream commit(s) and local symbol.
  - Devin executor: `permission_denied`+"high demand"→429; `swe-1-6-slow` resolves; function_result/function_call id rules match upstream; stream buffers deltas until thinking signatures; `interaction.status`/`finish_reason` emitted from `lastStopReason`; `automation_update` filtered, exec/write_stdin descriptions obfuscated, Codex directives stripped, namespace `children`/`function_declarations` expanded.
  - Claude executor: stream skips scanner-error handling when `upstreamCompleted` (no spurious failure on client disconnect after completion).
  - Translators: tool messages carry `"name"`; `reasoning`/`reasoning_details` collected besides `reasoning_content`; demoted system/developer messages wrapped in `<system-reminder>`; namespaced tool names capped to 64 chars with collision disambiguation; Gemini schema strips `id`/`$anchor`/`$vocabulary`/`$dynamicRef`/`$dynamicAnchor`.
  - Auth: `authUnavailableError` carries retry-deadline headers + cause enrichment; home dispatch `model_info` round-trips `native_capabilities`.
  - Web search: OpenAI-Responses `web_search` tools translate to `googleSearch` for antigravity + gemini paths with included-domains, streaming citation mapping, URL resolution; `antigravity` in the supported provider bucket; `sdk/translator` envelope API preserves `ModelInfo` end-to-end.
  - OpenAI-compat: `input-modalities: [text]` models get normalized tool results + stripped relayed images.
  - `codex_exec/<ver>` UA recognized as Codex client.
  - `go test` on touched packages, `make build`, `git diff --check` pass at gates.
  - Newer-than-v7.3.6 upstream delta is pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.6` at `8c664b2f` / `refs/upstream-checkpoints/cliproxyapi/v7.3.6`; `scope_policy` is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.3.4..v7.3.5.json` — 60 paths (baseline 1, upstream-add-absent 5, diverged-absent 20, semantic-review 34). [gitignored artifact — regenerate via `upstream_gap.py`]
  - `docs/upstream/cliproxyapi-gap-v7.3.5..v7.3.6.json` — 37 paths (baseline 2, upstream-add-absent 3, diverged-absent 15, semantic-review 17).
  - `docs/upstream/cliproxyapi-ledger-v7.3.4..v7.3.5.md` + `cliproxyapi-ledger-v7.3.5..v7.3.6.md` — 24 non-merge commits; per-commit dispositions + citations.
  - `docs/plans/completed/cliproxyapi-v7.3.4-parity.md` — prior cycle; R1–R9 slices must not regress.
  - `CLAUDE.md` / `docs/WORKFLOW.md` — Postgres-authoritative store, SDK additive-only, pluginhost non-goal.
- rejected_alternatives:
  - Wholesale merge of 97 paths — diverged-absent/semantic-review dominate (~66); local architecture differs.
  - Porting upstream's `internal/translator/{interactions/claude, claude/interactions, openai/interactions}/` packages — local Devin pipeline inlines interactions inside `devin_executor.go`; port semantics onto executor + `helps/usage_helpers.go` equivalents.
  - Copying upstream's split `antigravity_executor_{execute,stream,tokens}.go`, `claude_executor_stream.go`, `conductor_home.go`, `handlers_errors.go` — local equivalents are consolidated (`antigravity_executor.go`, `claude_executor.go`, `conductor.go`, `handlers.go`); port semantics into local structure.
  - Porting v7.3.5 web-search state then re-working for envelope — port the v7.3.6 end-state directly; envelope API (P3) merges first.
  - Deferring web-search/envelope/compat slices — owner approved full parity; only docs/branding and absent-architecture hunks stay out.
- requirements:
  - R1 [accepted]: **Devin executor slice** — `d44901f9` high-demand→429 in `ParseDevinTrailerError` (`devin_wire.go:1047`); `dea4ce8a` `swe-1-6-slow` in `devin_models.go:143-147` + `devin_models.json` (baseline, clean apply); `f51d3ae9` `call_id`/`id` precedence swap (`devin_executor.go:1314,1328`); `c2bb91d2` delta buffering for late thinking signatures in `streamDevinFrames` (`devin_executor.go:463`); `6f908cbc` `lastStopReason`→`interaction.status`/`finish_reason` in `streamDevinFrames` + `consumeDevinFramesToInteractions` (`:903`); `e54a8e97` cache-usage semantics onto local `helps/usage_helpers.go:parseInteractionsUsageDetail` + executor usage emission (total−cache input math, cache_read/cache_creation split); `ad088a87` tool filtering/sanitize (`internal/translator/common/devin_tools.go` new; `parseInteractionsPayload` `appendDevinTool` rework at `:1404`; `BuildDevinGetChatMessageRequest`/`BuildDevinUpstreamLogBody`/`SanitizeDevinSystemPrompt` in `devin_wire.go`). | source: `d44901f9` `dea4ce8a` `f51d3ae9` `c2bb91d2` `6f908cbc` `e54a8e97` `ad088a87`
  - R2 [accepted]: **Auth retry-enriched errors + native_capabilities** — `6724a958` `authUnavailableError` type in `sdk/cliproxy/auth/errors.go` (`Headers()`, `WithAuthError`, `Unwrap`, `As`, `Is`, `StatusCode`, `IsRequestScoped`, `MarkRequestScoped`, `MarshalJSON` preserving `Error` layout); call sites: `conductor.go:1209-1214` unavailable path, `scheduler.go` unavailable-error constructors, `selector.go` `getAvailableAuthsWithPriorityMode`, `home_concurrency.go:271` `SafeResponseHeaders`, `sdk/api/handlers/handlers.go` `enrichAuthSelectionError` carrier enrichment; `a9e92b81` hunk: `homeDispatchModelInfo.NativeCapabilities` + `registryModelInfo()` mapping (`conductor.go:5827-5852`). | source: `6724a958` + `a9e92b81` (conductor hunk)
  - R3 [accepted]: **Translator envelope metadata API + codex_exec UA** — `ModelInfo *registry.ModelInfo` on `RequestEnvelope`; `RequestEnvelopeTransform` type; `RegisterRequestEnvelope`; `TranslateRequestEnvelope` on `Registry` + default + `Pipeline` terminal; envelope variants `TranslateRequestEnvelopeWithCodexMultiAgentV2`/`TranslateRequestEnvelopePairWithCodexMultiAgentV2` in `internal/client/codex/optimize-multi-agent-v2` + `helps/codex_multi_agent_v2.go`; `923a8c30` `codex_exec/` prefix in `IsCodexClientUserAgent` (`optimize_multi_agent_v2.go:193-198`). Local registry is hook-free → additive-only port. | source: `a9e92b81` (sdk/client/helps hunks) + `923a8c30`
  - R4 [accepted]: **OpenAI-Responses web-search feature (v7.3.6 end-state)** — `ef63d2e7`+`7fcbdf88`+`a9e92b81` consumer: move `antigravity` to supported bucket in `responsesWebSearchProviderPathSupport` (`model_registry.go:267`); NEW `internal/translator/gemini/openai/responses/gemini_openai-responses_web_search.go` (~900 LOC end-state: `HasResponsesWebSearchTool`, `ExtractResponsesWebSearchAllowedDomains`, `AllowsResponsesWebSearchToolChoice`, `ModelSupportsWebSearch`, URL/citation resolution, streaming citation mapping); `googleSearch` tool block in `gemini_openai-responses_request.go`; citation mapping in `gemini_openai-responses_response.go`; `hasAntigravity{ResponsesWebSearch,ClaudeTypedWebSearch,GoogleSearch}Tool` in `antigravity_executor.go`; antigravity responses translator v7.3.6 end-state (`ConvertOpenAIResponsesRequestEnvelopeToAntigravity` with `antigravitySupportsNativeResponsesWebSearch(model, modelInfo)`, `buildAntigravityResponsesWebSearchRequest`, `ensureAntigravityResponsesWebSearch{Tool,SystemInstruction}`, `stripAntigravityResponsesGoogleSearch`, `rewriteOpenAIResponsesReasoningForAntigravityClaude`, `enableAntigravityResponsesThinkingSummary`) + `RegisterRequestEnvelope` in `init.go`; `ResolvedModelInfo`→envelope call sites in `antigravity_executor.go`. Depends on R3 merge. | source: `ef63d2e7` `7fcbdf88` `a9e92b81` + `b681a1e0` (responses hunk only)
  - R5 [accepted]: **Misc translator/executor batch** — `7c32971b` `!upstreamCompleted` guards + breaks in `claude_executor.go:728,783`; `7def8425` `toolNameByID`+`"name"` on tool messages (`openai_claude_request.go:181`); `77820cb2` `collectOpenAIObjectReasoningTexts` (`reasoning_content`/`reasoning`/`reasoning_details`) at `openai_claude_response.go:227,412,777`; `b681a1e0` export `SystemReminderText` (`common/claude_system.go`) + `geminiDemotedSystemText` wraps in `gemini_openai_request.go:157-173` + `antigravity_openai_request.go` equivalents; `e3e97ad9` `disambiguateResponsesChatToolNames` (64-char cap, `_N` suffixes, local-name burn) in `openai_openai-responses_tools.go`; `f51d3ae9` gemini sanitizer (`sanitizeGeminiInteractionsUnsupportedInputIDs` — absent locally) in `gemini_executor.go`; `f668ac41` five schema keywords in `removeUnsupportedKeywords` (`util/gemini_schema.go:909-914`). | source: `7c32971b` `7def8425` `77820cb2` `b681a1e0` `e3e97ad9` `f51d3ae9` `f668ac41`
  - R6 [accepted]: **OpenAI-compat text-only tool results** — `13435c93`+`c4982e84`: `helps/openai_compat_tool_results.go` (`ShouldNormalizeOpenAIToolResultsForModel`, `NormalizeOpenAIToolResultsTextOnly`, `flattenOpenAIToolResultContent`, `openAICompatibilityModelExcludesImages`, relay-notice/placeholder stripping, `[image omitted]` markers); `InputModalities []string` (`yaml:"input-modalities,omitempty"`) on `OpenAICompatibilityModel` (`config.go:780`); two call sites in `openai_compat_executor.go` (upstream `:130`/`:345`). | source: `c4982e84` + `13435c93`
  - R7 [accepted]: Invariants — Postgres authoritative runtime store; public SDK additive-only; semantic ports behind local interfaces, never text-applied; no `web/**/*_test.go`.
  - R8 [accepted]: Final gate — re-resolve latest stable upstream release, refresh checkpoint + ledger; newer release pinned as follow-up.

## Non-goals
- NG1: README/README_CN/README_JA/AGENTS.md/`config.example.yaml` doc churn (`7c961b34`, `cfeeeb34`, `46f6cb01`, `45160378`, `77820cb2` yaml hunk) — branding-docs; no local README_CN/JA; `config.example.yaml` deleted locally.
- NG2: meta executor test UA constant (`311efcb3`) — already-present (`meta_executor_test.go:149`).
- NG3: upstream `internal/translator/{interactions/claude, claude/interactions, openai/interactions}/` package contents (12+ diverged-absent paths) — no local equivalents; behavior mapped onto executor-inlined surfaces per-slice instead.
- NG4: upstream test-file reorganization for absent files (`claude_executor_stream_terminal_test.go` maps into local `claude_executor*_test.go` convention instead).
- NG5: wholesale upstream merge; non-additive SDK breakage; branding churn; `web/` and `internal/tui/` (zero paths in range).

## Approach and Risks
- approach: six work phases organized by **file ownership** so they fan out without write conflicts, plus a final gate. Merge order constraint: P3 (envelope API) before P4 (web-search consumes `RegisterRequestEnvelope` + envelope pair helpers). All others independent.
- constraints: ports re-express upstream semantics behind local interfaces — monolithic `conductor.go`/`claude_executor.go`/`antigravity_executor.go`, inlined devin interactions, hook-free `sdk/translator` registry. Public SDK additive-only.
- dependencies: P4 needs P3 (envelope API + envelope pair helpers + ResolvedModelInfo call pattern). R1's devin tasks share `devin_executor.go`/`devin_wire.go` — same phase, sequential waves. R5 tasks are all disjoint files within one phase.
- risks + mitigations:
  - `streamDevinFrames` 250-line rework (`c2bb91d2`) + `ad088a87` parse rework + `6f908cbc` finish reason all land in `devin_executor.go` — highest-risk phase; mitigate by porting upstream's `devin_executor_test.go` additions per-commit and gating on them.
  - Web-search feature ~1500 LOC prod + ~5300 LOC tests — largest slice; mitigate by porting upstream test files alongside (web_search_test.go end-state) and requiring them green.
  - `authUnavailableError` must preserve `Error`'s four-field layout (unkeyed literals downstream) — port `MarshalJSON`/layout verbatim; run full `sdk/cliproxy/auth` + `sdk/api/handlers` suites.
  - `disambiguateResponsesChatToolNames` semantics subtle (identity vs collision, local-name burn) — port upstream `openai_openai-responses_request_test.go` +545 additions.
  - `id` schema keyword could strip a real property named `id` — keep upstream `isPropertyDefinition` guard semantics.
  - Envelope API must not change existing `RequestTransform` behavior — port `TestRequestEnvelopePreservesRegisteredTransformDispatch` semantics (Register wraps, envelope route, precedence).
  - Recovery: phases merge independently; each phase's files revert without touching others.
- stop_conditions: upstream diff can't be re-expressed without breaking a locked SDK signature → escalate; same `go test` package fails twice → escalate.

## Phases and Verification

### Phase P1 — devin executor slice (story_id: p1-devin-20260918) — status: planned
- goal: land R1 — all devin commits from both releases.
- depends_on: none.
- surfaces_allowed: `internal/runtime/executor/devin_executor{,_test}.go`, `internal/runtime/executor/helps/devin_wire{,_test}.go`, `internal/runtime/executor/helps/devin_models{,_test}.go`, `internal/registry/models/devin_models.json`, `internal/runtime/executor/helps/usage_helpers{,_test}.go`, new `internal/translator/common/devin_tools{,_test}.go`.
- surfaces_avoided: `internal/translator/openai/`, `internal/util/`, all other executors.
- wave 1 (independent):
  - T1.1 `d44901f9` — `permission_denied`+"high demand"→429 in `ParseDevinTrailerError` (`devin_wire.go:1047`) + 4 test cases. check: `go test ./internal/runtime/executor/helps/ -run 'TestParseDevinTrailerError'`.
  - T1.2 `dea4ce8a` — `swe-1-6-slow` in `devin_models.go:143` + `devin_models.json` upstream entries (baseline apply). check: `go test ./internal/runtime/executor/helps/ -run 'TestDevinModel|TestChatModelUID'`; `go test ./internal/registry/ -run 'TestDevin|TestModels'`.
  - T1.3 `ad088a87` part A — new `internal/translator/common/devin_tools.go` + test. check: `go test ./internal/translator/common/ -run 'TestDevin|TestSanitize|TestObfuscate|TestIsDevinCodex'`.
- wave 2:
  - T1.4 `c2bb91d2` — `streamDevinFrames` (`devin_executor.go:463`) buffer content deltas until thinking signature resolves; port upstream `devin_executor_test.go` additions (+551). check: `go test ./internal/runtime/executor/ -run 'TestStreamDevinFrames|TestDevinStream|TestDevinThinking'`.
  - T1.5 `ad088a87` part B — `parseInteractionsPayload` `appendDevinTool` rework (`:1404`: namespace `tools`→`children`, `function_declarations`/`functionDeclarations`, `parametersJsonSchema`, automation_update, sanitize) + `f51d3ae9` devin hunk (`call_id` before `id` at `:1314,1328`); `devin_wire.go` filter+sanitize in `BuildDevinGetChatMessageRequest`/`BuildDevinUpstreamLogBody` + two Codex directive strips in `SanitizeDevinSystemPrompt`. check: `go test ./internal/runtime/executor/ ./internal/runtime/executor/helps/ -run 'TestParseInteractions|TestBuildDevin|TestSanitizeDevin|TestDevinUpstreamLog'`.
  - T1.6 `6f908cbc` + `e54a8e97` — `lastStopReason`→`interaction.status`/`finish_reason` in both stream + non-stream paths; cache-usage semantics (total−cache input, cache_read/cache_creation split) onto `parseInteractionsUsageDetail` + executor emission sites (`:841`,`:1104`). check: `go test ./internal/runtime/executor/ ./internal/runtime/executor/helps/ -run 'TestConsumeDevin|TestParseInteractionsUsage|TestDevinUsage|TestDevinFinish'`.
- phase check: `go test ./internal/runtime/executor/ ./internal/runtime/executor/helps/ ./internal/translator/common/ ./internal/registry/` all pass; `git diff --check` clean.

### Phase P2 — auth retry errors + capabilities (story_id: p2-auth-20260918) — status: planned
- goal: land R2 — `authUnavailableError` + `native_capabilities` dispatch.
- depends_on: none.
- surfaces_allowed: `sdk/cliproxy/auth/{errors,scheduler,selector,conductor,home_concurrency}.go` + tests (incl. new `retry_deadline_test.go`, `home_web_search_capability_test.go` equivalents), `sdk/api/handlers/handlers.go` + tests.
- surfaces_avoided: other sdk/cliproxy/auth files, internal/.
- wave 1:
  - T2.1 — `authUnavailableError` type in `errors.go` (upstream end-state verbatim semantics: 4-field `Error` layout, `Headers()` retry hints, `WithAuthError`, `Unwrap`/`As`/`Is`, `StatusCode`, `MarkRequestScoped`, `MarshalJSON`); port `retry_deadline_test.go` cases. check: `go test ./sdk/cliproxy/auth/ -run 'TestRetryDeadline|TestAuthUnavailable'`.
- wave 2 (depends T2.1):
  - T2.2 — call sites: `conductor.go:1209-1214` → `newAuthUnavailableErrorWithCause(earliest, now, lastCandidateErr)` (track `earliest` like upstream `availableAuthsForRouteModelWithPriorityMode`); `scheduler.go` unavailable constructors; `selector.go:538`; `home_concurrency.go:271` `SafeResponseHeaders` `errors.As` branch; `handlers.go` `enrichAuthSelectionError` `WithAuthError` carrier. check: `go test ./sdk/cliproxy/auth/ ./sdk/api/handlers/ -run 'TestUnavailable|TestSelection|TestRetry|TestEnrichAuth|TestSafeResponseHeaders|TestSchedule'`.
  - T2.3 `a9e92b81` hunk — `homeDispatchModelInfo.NativeCapabilities` + `registryModelInfo()` mapping (`conductor.go:5827-5852`) + `home_web_search_capability_test.go` equivalent. check: `go test ./sdk/cliproxy/auth/ -run 'TestHomeWebSearch|TestHomeDispatch|TestModelInfo'`.
- phase check: `go test ./sdk/cliproxy/auth/ ./sdk/api/handlers/` all pass; `git diff --check` clean.

### Phase P3 — envelope API + codex_exec UA (story_id: p3-envelope-20260918) — status: planned
- goal: land R3 — `RequestEnvelope.ModelInfo` transform API.
- depends_on: none (merge before P4).
- surfaces_allowed: `sdk/translator/{types,pipeline,registry,registry_test}.go`, `internal/client/codex/optimize-multi-agent-v2/optimize_multi_agent_v2{,_test}.go`, `internal/runtime/executor/helps/codex_multi_agent_v2{,_test}.go`.
- surfaces_avoided: `antigravity_executor.go` (P4 owns call sites), `internal/translator/`.
- wave 1:
  - T3.1 — `sdk/translator`: `RequestEnvelopeTransform` (types.go), `ModelInfo *registry.ModelInfo` field (pipeline.go), `requests` map → `RequestEnvelopeTransform` with `Register` wrapping legacy transforms, `RegisterRequestEnvelope`, `TranslateRequestEnvelope` (Registry + default + package), `Pipeline` terminal → envelope. Preserve hook-free flow (summaryConfig → fn → ApplySummaryConfigForModel); `TranslateRequest` stays as shim. Port envelope tests. check: `go test ./sdk/translator/`.
- wave 2 (depends T3.1):
  - T3.2 — envelope variants in `optimize_multi_agent_v2.go` + `helps/codex_multi_agent_v2.go` (incl. `sameByteSlice` fast path preserved); `923a8c30` `codex_exec/` prefix. check: `go test ./internal/client/codex/optimize-multi-agent-v2/ ./internal/runtime/executor/helps/ -run 'TestTranslate|TestEnvelope|TestCodexMultiAgent|TestIsCodexClient'`.
- phase check: `go test ./sdk/translator/ ./internal/client/codex/optimize-multi-agent-v2/ ./internal/runtime/executor/helps/` all pass; `go build ./...` clean.

### Phase P4 — responses web-search feature (story_id: p4-websearch-20260918) — status: planned
- goal: land R4 — antigravity + gemini OpenAI-Responses web-search at v7.3.6 end-state.
- depends_on: P3 merged (needs `RegisterRequestEnvelope`, envelope pair helpers).
- surfaces_allowed: `internal/translator/gemini/openai/responses/{gemini_openai-responses_request.go, gemini_openai-responses_response.go, gemini_openai-responses_web_search.go, *_test.go}`, `internal/translator/antigravity/openai/responses/*`, `internal/runtime/executor/antigravity_executor*.go`, `internal/registry/model_registry.go` + `web_search_capability_test.go`.
- surfaces_avoided: `sdk/translator/` (P3), `gemini/openai/chat-completions/` (P5).
- wave 1:
  - T4.1 — `model_registry.go:267` antigravity → supported bucket + test update; `hasAntigravity{ResponsesWebSearch,ClaudeTypedWebSearch,GoogleSearch}Tool` helpers + `ResolvedModelInfo`→envelope call sites in `antigravity_executor.go`. check: `go test ./internal/registry/ -run 'TestResponsesWebSearch|TestWebSearch'`; `go test ./internal/runtime/executor/ -run 'TestAntigravity'`.
  - T4.2 — NEW `gemini_openai-responses_web_search.go` (v7.3.6 end-state ~900 LOC) + its test file; `googleSearch` tool block in `gemini_openai-responses_request.go`; citation/streaming mapping in `gemini_openai-responses_response.go`. check: `go test ./internal/translator/gemini/openai/responses/`.
- wave 2 (depends T4.1+T4.2):
  - T4.3 — antigravity responses translator end-state: `ConvertOpenAIResponsesRequestEnvelopeToAntigravity` (+ legacy shim delegating), `antigravitySupportsNativeResponsesWebSearch(model, modelInfo)`, `buildAntigravityResponsesWebSearchRequest`, `ensureAntigravityResponsesWebSearch{Tool,SystemInstruction}`, `stripAntigravityResponsesGoogleSearch`, `rewriteOpenAIResponsesReasoningForAntigravityClaude`, `enableAntigravityResponsesThinkingSummary`; `RegisterRequestEnvelope(OpenAIResponse→Antigravity)` in `init.go`; port request/response test files. check: `go test ./internal/translator/antigravity/... ./sdk/translator/`.
- phase check: `go test ./internal/translator/gemini/openai/responses/ ./internal/translator/antigravity/... ./internal/runtime/executor/ ./internal/registry/` all pass; `go build ./...` clean.

### Phase P5 — misc translators + executors (story_id: p5-misc-20260918) — status: planned
- goal: land R5 — seven small independent fixes.
- depends_on: none.
- surfaces_allowed: `internal/runtime/executor/claude_executor.go` + tests, `internal/runtime/executor/gemini_executor.go` + tests, `internal/translator/openai/claude/openai_claude_{request,response}.go` + tests, `internal/translator/common/claude_system{,_test}.go`, `internal/translator/gemini/openai/chat-completions/gemini_openai_request{,_test}.go`, `internal/translator/antigravity/openai/chat-completions/antigravity_openai_request{,_test}.go`, `internal/translator/openai/openai/responses/openai_openai-responses_tools{,_test}.go`, `internal/util/gemini_schema{,_test}.go`.
- surfaces_avoided: `internal/translator/gemini/openai/responses/` (P4), `internal/translator/antigravity/openai/responses/` (P4), `internal/runtime/executor/devin_*` (P1).
- wave 1 (all independent):
  - T5.1 `7c32971b` — `!upstreamCompleted` guard on `scanner.Err()`/emit paths + break at `claude_executor.go:728,783`; port terminal test semantics into local `claude_executor*_test.go`. check: `go test ./internal/runtime/executor/ -run 'TestClaude.*Stream|TestClaudeExecutor'`.
  - T5.2 `7def8425` — `toolNameByID` map + `"name"` on tool messages (`openai_claude_request.go:181`). check: `go test ./internal/translator/openai/claude/ -run 'TestConvert|TestToolName'`.
  - T5.3 `77820cb2` — `collectOpenAIObjectReasoningTexts` at `:227,412,777` + helper. check: `go test ./internal/translator/openai/claude/ -run 'TestReasoning|TestStream'`.
  - T5.4 `b681a1e0` (minus responses hunk) — export `SystemReminderText`; `geminiDemotedSystemText` wraps in `gemini_openai_request.go:157-173` + `antigravity_openai_request.go` equivalent. check: `go test ./internal/translator/common/ ./internal/translator/gemini/openai/chat-completions/ ./internal/translator/antigravity/openai/chat-completions/ -run 'TestSystem|TestReminder|TestConvert|TestDemoted'`.
  - T5.5 `e3e97ad9` — `disambiguateResponsesChatToolNames` + cap wiring in `openai_openai-responses_tools.go`; port +545 test additions. check: `go test ./internal/translator/openai/openai/responses/ -run 'TestTool|TestNamespace|TestDisambiguate|TestCap'`.
  - T5.6 `f51d3ae9` gemini hunk — `sanitizeGeminiInteractionsUnsupportedInputIDs` (step-type id/call_id rules) in `gemini_executor.go`. check: `go test ./internal/runtime/executor/ -run 'TestGemini'`.
  - T5.7 `f668ac41` — five keywords in `removeUnsupportedKeywords` (`gemini_schema.go:909`). check: `go test ./internal/util/ -run 'TestGeminiSchema|TestRemoveUnsupported|TestClean'`.
- phase check: `go test ./internal/runtime/executor/ ./internal/translator/... ./internal/util/` all pass; `git diff --check` clean.

### Phase P6 — compat text-only tool results (story_id: p6-compat-20260918) — status: planned
- goal: land R6.
- depends_on: none.
- surfaces_allowed: `internal/config/config.go`, new `internal/runtime/executor/helps/openai_compat_tool_results{,_test}.go`, `internal/runtime/executor/openai_compat_executor.go` + tests.
- surfaces_avoided: other config structs, other executors.
- wave 1:
  - T6.1 — `OpenAICompatibilityModel.InputModalities` field. check: `go test ./internal/config/`; yaml round-trip `input-modalities: [text]`.
  - T6.2 — `helps/openai_compat_tool_results.go` v7.3.6 end-state + both upstream test files. check: `go test ./internal/runtime/executor/helps/ -run 'TestNormalize|TestShouldNormalize|TestFlatten'`.
- wave 2 (depends T6.1+T6.2):
  - T6.3 — two call sites in `openai_compat_executor.go` (resolve compat → `ShouldNormalize…` → `NormalizeOpenAIToolResultsTextOnly`) + executor test additions. check: `go test ./internal/runtime/executor/ -run 'TestOpenAICompat|TestCompat'`.
- phase check: `go test ./internal/config/ ./internal/runtime/executor/helps/ ./internal/runtime/executor/` all pass; `git diff --check` clean.

### Phase P7 — final gate (story_id: p7-gate-20260918) — status: planned
- goal: R8.
- depends_on: P1–P6 checked.
- wave 1:
  - T7.1 — `upstream_sync.py sync --slug cliproxyapi`; if latest > v7.3.6, regenerate gap+ledger for `v7.3.6..{latest}` as follow-up; refresh `scope_policy`. check: `gh release list --repo router-for-me/CLIProxyAPI --limit 1` agrees with checkpoint.
- phase check: `go test ./...` green on touched packages; `make build` succeeds; `git diff --check` clean.

## Progress
- 2026-09-18 — triage v7.3.5..v7.3.6 (37 paths / 10 commits) + v7.3.4..v7.3.5 (60 paths / 14 commits) complete; ledgers written. Owner approved FULL parity scope (incl. web-search feature, envelope API, compat slice). Plan full-planned (to-plan full): 6 work phases file-disjoint for fan-out, P3→P4 merge order.
- 2026-09-19 — implementation complete via 6 fan-out subagents (P1 devin, P2 auth, P3 envelope, P4 web-search, P5 misc, P6 compat). Gate: `go build ./...` clean; `make build` clean; `go test ./...` green except pre-existing env failures (`internal/updater TestRollbackFailure`, `cmd/server TestUpdateCommandRollback` — uid 0 bypasses chmod-permission assertions, files untouched); `git diff --check` clean; gofmt clean; zero `web/` changes. **Follow-up pinned**: upstream now at v7.3.8 (`c93978c4`); v7.3.6..v7.3.8 = 34 commits / 127 paths — gap+ledger generated (`cliproxyapi-gap-v7.3.6..v7.3.8.json`, `cliproxyapi-ledger-v7.3.6..v7.3.8.md`) for next triage cycle. Plan ready to move to `completed/` after review.
