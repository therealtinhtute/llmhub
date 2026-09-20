# Upstream ledger — cliproxyapi v7.3.5..v7.3.6

- generated: 2026-09-18T07:19:01Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `ee727c649fed`
- non-merge commits: 10
- triaged: 2026-09-18 (dispositions proposed, pending owner approval)

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.3.6 | `923a8c30dcb3` | 2026-09-04 | fix(codex): recognize codex_exec user agent for multi-agent v2 optimization | `internal/client` | adapt | upstream `optimize_multi_agent_v2.go` `IsCodexClientUserAgent` adds `codex_exec/` prefix; local `internal/client/codex/optimize-multi-agent-v2/optimize_multi_agent_v2.go:193-198` lacks it |
| v7.3.6 | `7c961b34fd55` | 2026-09-09 | docs: add PiCloud to related projects | `README.md`, `README_CN.md`, `README_JA.md` | reject | branding-docs group (scope_policy exclude); no local README_CN/JA |
| v7.3.6 | `cfeeeb341884` | 2026-09-10 | docs(readme): update links after GitHub username change | `README.md`, `README_CN.md`, `README_JA.md` | reject | branding-docs group |
| v7.3.6 | `ad088a879507` | 2026-09-17 | fix(devin): filter automation update tools and sanitize tool descriptions | `internal/runtime`, `internal/translator`, `internal/util` | adapt | new `internal/translator/common/devin_tools.go`; executor parse loop maps onto local `devin_executor.go:1404-1416` `parseInteractionsPayload`; wire changes map onto `helps/devin_wire.go:347` `BuildDevinGetChatMessageRequest`, `:1169` `BuildDevinUpstreamLogBody`, `:676` `SanitizeDevinSystemPrompt`. `interactions_openai_responses_*` + `util/responses_tools.go` hunks are n/a — local has no interactions translator package (executor inlines) |
| v7.3.6 | `a9e92b81453f` | 2026-09-17 | feat(translator): preserve model metadata in requests | `internal/client`, `internal/runtime`, `internal/translator` +2 | adapt (split) | (a) `homeDispatchModelInfo.NativeCapabilities` → local `sdk/cliproxy/auth/conductor.go:5827-5852`; (b) envelope API (`RequestEnvelopeTransform`/`TranslateRequestEnvelope`/`ModelInfo`) + codex-multi-agent envelope variants + antigravity consumer — local `antigravity_openai-responses_request.go` is a 12-line pass-through shim; the only envelope consumer (responses web-search routing, `ef63d2e7`/`7fcbdf88` lineage) was never ported → see open decision |
| v7.3.6 | `46f6cb01976c` | 2026-09-17 | docs: document CLIProxyAPIHome integration areas | `AGENTS.md` | reject | upstream-specific doc; branding-docs group |
| v7.3.6 | `45160378200d` | 2026-09-17 | docs: add cc-status-line to community projects | `README.md`, `README_CN.md`, `README_JA.md` | reject | branding-docs group |
| v7.3.6 | `f668ac417d97` | 2026-09-17 | fix(schema): strip unsupported schema identifier keywords | `internal/util` | adapt | upstream `gemini_schema.go` `removeUnsupportedKeywords` adds `id`,`$anchor`,`$vocabulary`,`$dynamicRef`,`$dynamicAnchor`; local `internal/util/gemini_schema.go:909-914` list lacks all five |
| v7.3.6 | `311efcb3a29e` | 2026-09-17 | test(executor): use metaUserAgent constant in meta executor test | `internal/runtime` | already-present | local `internal/runtime/executor/meta_executor_test.go:149-150` already asserts `metaUserAgent` (comment notes upstream's stale literal) |
| v7.3.6 | `c4982e846e16` | 2026-09-17 | fix(executor): strip relayed tool result images for text-only models | `internal/runtime` | adapt | whole feature absent locally — base `13435c93` (`NormalizeOpenAIToolResultsTextOnly`/`ShouldNormalizeOpenAIToolResultsForModel`, `helps/openai_compat_tool_results.go`) never ported; local `internal/config/config.go:780` `OpenAICompatibilityModel` lacks `InputModalities`; call sites `openai_compat_executor.go:130,345` upstream have no local counterpart |
