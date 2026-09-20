# Upstream ledger — cliproxyapi v7.3.6..v7.3.8

- generated: 2026-09-19T06:28:48Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `ee727c649fed`
- non-merge commits: 34

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| unassigned | `0b55053944aa` | 2026-09-17 | fix(management): reject unresolved token placeholders in api-call | `internal/api` |  |  |
| unassigned | `b715526add0c` | 2026-09-17 | feat(plugin): support scheduling across priorities | `internal/pluginhost`, `sdk/cliproxy`, `sdk/pluginapi` +1 |  |  |
| unassigned | `afba07ba265a` | 2026-09-17 | fix(config): preserve plugin configurations when saving yaml | `internal/config` |  |  |
| unassigned | `64c9433fd2c2` | 2026-09-17 | fix(devin): support images in tool results | `internal/runtime`, `internal/translator` |  |  |
| unassigned | `76ac75e68ae6` | 2026-09-17 | feat(management): paginate auth file listings | `internal/api` |  |  |
| unassigned | `f86a33f72175` | 2026-09-18 | fix(executor): restrict claude advisor tool check to server tool use | `internal/runtime` |  |  |
| unassigned | `c2ea2684099f` | 2026-09-18 | fix(translator): scope tool responses per turn to handle repeated tool call IDs | `internal/translator` |  |  |
| unassigned | `cb62a6748b99` | 2026-09-18 | fix(openai): classify stream request timeout as server error | `sdk/api`, `test/codex_incomplete_stream_error_type_test.go` |  |  |
| unassigned | `81d6ba774621` | 2026-09-18 | fix(codex): preserve reasoning content and IDs for compat models in responses | `internal/runtime` |  |  |
| unassigned | `e84e248c51e5` | 2026-09-18 | fix(executor): record expected Devin upstream model and bound stream observer memory | `internal/runtime` |  |  |
| unassigned | `cde7d57e44e6` | 2026-09-18 | fix(executor): robust response model observability across meta, kimi, and openai-compat streams | `internal/runtime` |  |  |
| unassigned | `f8467f07dca5` | 2026-09-18 | fix(executor): record authentic Devin and Gemini Interactions response models | `internal/runtime` |  |  |
| unassigned | `e9463ff5a795` | 2026-09-18 | feat(executor): extend response model recording and substitution warnings to all providers | `internal/runtime` |  |  |
| unassigned | `0b9a91fb7871` | 2026-09-18 | feat: add FluxA | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| unassigned | `784285a4854d` | 2026-09-18 | feat: add PatewayAI | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| unassigned | `cc545cbf906b` | 2026-09-18 | fix(openai): align tool call messages and preserve ordering on ambiguous outputs | `internal/runtime`, `internal/translator`, `sdk/translator` |  |  |
| unassigned | `25f40d8cf8df` | 2026-09-18 | feat(codex): record upstream response model and warn on silent model substitution | `internal/redisqueue`, `internal/runtime`, `sdk/cliproxy` |  |  |
| unassigned | `859c486512b7` | 2026-09-18 | fix(codex): normalize and support ultrafast service tier | `internal/translator` |  |  |
| unassigned | `28743473c11a` | 2026-09-18 | feat(codex): append (Devin) suffix to Devin model display names | `internal/api`, `internal/client`, `sdk/api` |  |  |
| unassigned | `660a5800e777` | 2026-09-18 | fix(xai): restore aliased client web search tool name in responses | `internal/runtime` |  |  |
| unassigned | `3662d1535a8b` | 2026-09-18 | fix(codex): strip item-level and tool output prompt cache breakpoints | `internal/translator` |  |  |
| unassigned | `1cce9325738f` | 2026-09-18 | fix(claude): skip retry-after header on overage-only rejections | `internal/runtime` |  |  |
| unassigned | `f049e00b76ae` | 2026-09-18 | feat(xai): also allow grok imagine aspect_ratio 20:9 | `sdk/api` |  |  |
| unassigned | `75bd6a60eb1b` | 2026-09-18 | feat(xai): allow grok imagine aspect_ratio 9:20 | `sdk/api` |  |  |
| unassigned | `b773607e3e77` | 2026-09-18 | fix(ci): replace go-cross/cgo-actions with direct FreeBSD sysroot cross-compilation | `.github/workflows` |  |  |
| unassigned | `c616193a6cf6` | 2026-09-18 | fix(xai): unify forced hosted tool choice normalization | `internal/runtime` |  |  |
| unassigned | `44eaef0009f8` | 2026-09-18 | feat(claude): support model-level cooling and scope overage rate limits | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| unassigned | `b6fe4f20c4ea` | 2026-09-18 | fix(devin): handle orphaned tool results and normalize function result payloads | `internal/runtime` |  |  |
| unassigned | `9e10db53ad89` | 2026-09-18 | fix(devin): aggregate tool calls by id and track cache write tokens | `internal/runtime` |  |  |
| unassigned | `c93978c4ea2e` | 2026-09-19 | fix(schema): normalize true boolean subschemas and strip unsupported keywords | `internal/runtime`, `internal/util` |  |  |
| unassigned | `b6d1f050af28` | 2026-09-19 | fix(executor): buffer post-tool text to order devin tool calls before assistant response | `internal/runtime` |  |  |
| unassigned | `22392c537d95` | 2026-09-19 | fix(executor): restore hybrid passthrough mcp tool names in claude oauth | `internal/runtime` |  |  |
| unassigned | `690f4f3116b6` | 2026-09-19 | feat(executor): support use-max-completion-tokens for openai compatibility models | `config.example.yaml`, `internal/config`, `internal/modelconfig` +3 |  |  |
| unassigned | `f4852170ee59` | 2026-09-19 | test(config): verify claude cloak persistence and update behavior | `internal/api`, `internal/config` |  |  |

> 34 commit(s) are not reachable from any recorded release commit.
