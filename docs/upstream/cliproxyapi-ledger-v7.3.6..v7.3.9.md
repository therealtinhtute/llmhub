# Upstream ledger — cliproxyapi v7.3.6..v7.3.9

- generated: 2026-09-20T15:10:39Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `f5c0ff0dfe41`
- non-merge commits: 43

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.3.7 | `0b55053944aa` | 2026-09-17 | fix(management): reject unresolved token placeholders in api-call | `internal/api` | adapt | `internal/api/handlers/management/api_tools.go` exists; placeholder rejection ports onto local api-call handler |
| v7.3.7 | `b715526add0c` | 2026-09-17 | feat(plugin): support scheduling across priorities | `internal/pluginhost`, `sdk/cliproxy`, `sdk/pluginapi` +1 | reject | pluginhost-platform (scope_policy exclude); local `internal/pluginhost/` has only `lifecycle.go` |
| v7.3.7 | `afba07ba265a` | 2026-09-17 | fix(config): preserve plugin configurations when saving yaml | `internal/config` | reject | pluginhost-platform; plugin config persistence presumes upstream rpc layer absent locally |
| v7.3.7 | `64c9433fd2c2` | 2026-09-17 | fix(devin): support images in tool results | `internal/runtime`, `internal/translator` | adapt | executor part maps onto `devin_executor.go`; `internal/translator/interactions/claude/` absent locally — translator hunks n/a |
| v7.3.7 | `76ac75e68ae6` | 2026-09-17 | feat(management): paginate auth file listings | `internal/api` | adapt | `internal/api/handlers/management/auth_files.go` exists; pagination extends local listing |
| v7.3.7 | `b773607e3e77` | 2026-09-18 | fix(ci): replace go-cross/cgo-actions with direct FreeBSD sysroot cross-compilation | `.github/workflows` | reject | ci-infra; upstream `release.yaml` uses go-cross — local uses goreleaser `release.yml` |
| v7.3.7 | `c616193a6cf6` | 2026-09-18 | fix(xai): unify forced hosted tool choice normalization | `internal/runtime` | adapt | upstream `xai_executor_request.go` maps onto local monolith `xai_executor.go` |
| v7.3.7 | `44eaef0009f8` | 2026-09-18 | feat(claude): support model-level cooling and scope overage rate limits | `config.example.yaml`, `internal/config`, `internal/runtime` | adapt | `ClaudeKey` at `internal/config/config.go:475`; new `helps/claude_ratelimit.go`; monolith executor |
| v7.3.7 | `b6fe4f20c4ea` | 2026-09-18 | fix(devin): handle orphaned tool results and normalize function result payloads | `internal/runtime` | adapt | `devin_executor.go` + `helps/devin_wire.go`; orphaned-result handling onto local parse/stream |
| v7.3.7 | `9e10db53ad89` | 2026-09-18 | fix(devin): aggregate tool calls by id and track cache write tokens | `internal/runtime` | adapt | `devin_executor.go` + `devin_wire.go`; aggregate-by-id + cache-write tokens |
| v7.3.8 | `f86a33f72175` | 2026-09-18 | fix(executor): restrict claude advisor tool check to server tool use | `internal/runtime` | adapt | upstream `claude_executor_cloaking.go` maps onto local `claude_executor.go` advisor check |
| v7.3.8 | `c2ea2684099f` | 2026-09-18 | fix(translator): scope tool responses per turn to handle repeated tool call IDs | `internal/translator` | adapt | `antigravity_openai_request.go` + `gemini_openai_request.go` (chat-completions) exist |
| v7.3.8 | `cb62a6748b99` | 2026-09-18 | fix(openai): classify stream request timeout as server error | `sdk/api`, `test/codex_incomplete_stream_error_type_test.go` | adapt | `sdk/api/handlers/openai_responses_stream_error.go` exists |
| v7.3.8 | `81d6ba774621` | 2026-09-18 | fix(codex): preserve reasoning content and IDs for compat models in responses | `internal/runtime` | adapt | upstream split codex files map onto `codex_executor.go` + `codex_websockets_executor.go`; new `openai_responses_signature.go` |
| v7.3.8 | `e84e248c51e5` | 2026-09-18 | fix(executor): record expected Devin upstream model and bound stream observer memory | `internal/runtime` | adapt | `devin_executor.go` + new `helps/stream_response_model_observer.go` |
| v7.3.8 | `cde7d57e44e6` | 2026-09-18 | fix(executor): robust response model observability across meta, kimi, and openai-compat streams | `internal/runtime` | adapt | new observer + call sites in kimi/meta/openai_compat/claude executors + `usage_helpers.go` |
| v7.3.8 | `f8467f07dca5` | 2026-09-18 | fix(executor): record authentic Devin and Gemini Interactions response models | `internal/runtime` | adapt | new `helps/response_model.go`; devin + gemini executor call sites |
| v7.3.8 | `e9463ff5a795` | 2026-09-18 | feat(executor): extend response model recording and substitution warnings to all providers | `internal/runtime` | adapt | one-line call sites across aistudio/antigravity/claude/devin/gemini/kimi/meta/xai/compat/codex executors |
| v7.3.8 | `0b9a91fb7871` | 2026-09-18 | feat: add FluxA | `README.md`, `README_CN.md`, `README_JA.md` +1 | reject | branding-docs group (scope_policy exclude) |
| v7.3.8 | `784285a4854d` | 2026-09-18 | feat: add PatewayAI | `README.md`, `README_CN.md`, `README_JA.md` +1 | reject | branding-docs group |
| v7.3.8 | `cc545cbf906b` | 2026-09-18 | fix(openai): align tool call messages and preserve ordering on ambiguous outputs | `internal/runtime`, `internal/translator`, `sdk/translator` | adapt | new `translator/common/openai_tools.go`; touches `openai_openai-responses_request.go`, `openai_claude_request.go`, `helps/codex_multi_agent_v2.go` |
| v7.3.8 | `25f40d8cf8df` | 2026-09-18 | feat(codex): record upstream response model and warn on silent model substitution | `internal/redisqueue`, `internal/runtime`, `sdk/cliproxy` | adapt | `internal/redisqueue/plugin.go` + codex terminal + response_model helper |
| v7.3.8 | `859c486512b7` | 2026-09-18 | fix(codex): normalize and support ultrafast service tier | `internal/translator` | adapt | `codex_openai-responses_request.go` exists |
| v7.3.8 | `28743473c11a` | 2026-09-18 | feat(codex): append (Devin) suffix to Devin model display names | `internal/api`, `internal/client`, `sdk/api` | adapt | `internal/client/codex/models/models.go` + `server_routes.go` |
| v7.3.8 | `660a5800e777` | 2026-09-18 | fix(xai): restore aliased client web search tool name in responses | `internal/runtime` | adapt | upstream split xai files map onto `xai_executor.go` + `xai_websockets_executor.go` |
| v7.3.8 | `3662d1535a8b` | 2026-09-18 | fix(codex): strip item-level and tool output prompt cache breakpoints | `internal/translator` | adapt | `codex_openai-responses_request.go` exists |
| v7.3.8 | `1cce9325738f` | 2026-09-18 | fix(claude): skip retry-after header on overage-only rejections | `internal/runtime` | adapt | needs `helps/claude_ratelimit.go` (44eaef00 port) |
| v7.3.8 | `f049e00b76ae` | 2026-09-18 | feat(xai): also allow grok imagine aspect_ratio 20:9 | `sdk/api` | adapt | `sdk/api/handlers/openai/openai_images_handlers.go` — add 20:9 |
| v7.3.8 | `75bd6a60eb1b` | 2026-09-18 | feat(xai): allow grok imagine aspect_ratio 9:20 | `sdk/api` | adapt | same file — add 9:20 |
| v7.3.8 | `c93978c4ea2e` | 2026-09-19 | fix(schema): normalize true boolean subschemas and strip unsupported keywords | `internal/runtime`, `internal/util` | adapt | `util/gemini_schema.go` — extends `removeUnsupportedKeywords` with boolean subschemas |
| v7.3.8 | `b6d1f050af28` | 2026-09-19 | fix(executor): buffer post-tool text to order devin tool calls before assistant response | `internal/runtime` | adapt | `devin_executor.go` post-tool text buffering |
| v7.3.8 | `22392c537d95` | 2026-09-19 | fix(executor): restore hybrid passthrough mcp tool names in claude oauth | `internal/runtime` | adapt | upstream `claude_executor_request.go` maps onto local `claude_executor.go` |
| v7.3.8 | `690f4f3116b6` | 2026-09-19 | feat(executor): support use-max-completion-tokens for openai compatibility models | `config.example.yaml`, `internal/config`, `internal/modelconfig` +3 | adapt | fold field into `OpenAICompatibilityModel` (config.go; upstream config_types.go absent) + new `helps/openai_compat_max_tokens.go` |
| v7.3.8 | `f4852170ee59` | 2026-09-19 | test(config): verify claude cloak persistence and update behavior | `internal/api`, `internal/config` | adapt | cloak persistence test — folds into claude-cooling slice |
| v7.3.9 | `93b94d22a9bd` | 2026-09-19 | feat(management): support priority field in claude key patch | `internal/api` | adapt | `config_lists.go` — priority field on claude key patch |
| v7.3.9 | `b4ff581dafa4` | 2026-09-19 | fix(management): reconcile expired cooldowns to active status in auth files | `internal/api`, `sdk/cliproxy` | adapt | `auth_files.go` + upstream `conductor_cooldown.go` maps onto `sdk/cliproxy/auth/conductor.go` |
| v7.3.9 | `b532db9c4b14` | 2026-09-19 | fix(translator): preserve constraints and additionalProperties in gemini parametersJsonSchema | `internal/translator`, `internal/util` | adapt | `gemini_claude_request.go`, `gemini_openai_request.go`, `util/gemini_schema.go` |
| v7.3.9 | `f247e2b017a8` | 2026-09-19 | feat(translator): support strict tool mode mapping for gemini and claude | `internal/translator` | adapt | `claude_openai_request.go`, `gemini_claude_request.go`, `gemini_openai_request.go` |
| v7.3.9 | `49eec664f419` | 2026-09-19 | fix(translator): align tool choice mapping and enforce fail-closed handling | `internal/translator` | adapt | `claude_openai_request.go`, `gemini_openai_request.go`, `openai_claude_request.go` |
| v7.3.9 | `883660fb8153` | 2026-09-19 | fix(translator): deduct cache_write_tokens from input_tokens in openai/codex to claude (#5956) | `internal/translator` | adapt | `codex_claude_response.go`, `openai_claude_response.go` |
| v7.3.9 | `61fdfc341b96` | 2026-09-20 | feat(translator): support multimodal audio, video, and file inputs in responses to gemini | `internal/translator` | adapt | `gemini_openai-responses_request.go` (P4 surface) +769-line multimodal rewrite |
| v7.3.9 | `1c87874966a0` | 2026-09-20 | fix(pluginhost): preserve http status codes in host callback and execution errors | `internal/pluginhost` | reject | pluginhost-platform group |
| v7.3.9 | `42c9680eee55` | 2026-09-20 | feat(executor): support duplex streaming for codex websockets | `config.example.yaml`, `internal/api`, `internal/config` +3 | adapt | `codex_websockets_executor.go` + new duplex/input files; config flag folds into config.go |
