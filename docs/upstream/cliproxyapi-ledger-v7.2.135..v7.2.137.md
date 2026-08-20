# Upstream ledger — cliproxyapi v7.2.135..v7.2.137

- generated: 2026-08-20T11:47:47Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `6f74bae11039`
- non-merge commits: 25

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.136 | `a8f9814a6961` | 2026-08-17 | fix(claude): keep Fable-only rate limits model scoped | `internal/runtime` |  |  |
| v7.2.136 | `4ac37ed3cd53` | 2026-08-18 | fix(claude): always send `stop` as an array when converting `stop_sequences` in OpenAI requests | `internal/translator` |  |  |
| v7.2.136 | `e424bfad00a6` | 2026-08-18 | feat(executor): support `$`-based custom headers from downstream request headers | `config.example.yaml`, `internal/runtime`, `internal/util` |  |  |
| v7.2.136 | `f3e836ce6c36` | 2026-08-18 | refactor(claude): deduplicate CLI identity application and tidy helpers | `internal/runtime`, `internal/watcher` |  |  |
| v7.2.136 | `aec70dfec4ab` | 2026-08-18 | fix(claude): gate CCH signing by upstream origin and validate fingerprint-profile | `config.example.yaml`, `internal/api`, `internal/config` +1 |  |  |
| v7.2.136 | `f1b0431c7718` | 2026-08-18 | feat(claude): add fingerprint-profile=claude-code-cli for API keys and delegated providers (#5047) | `config.example.yaml`, `internal/api`, `internal/config` +3 |  |  |
| v7.2.136 | `ee2c494788f8` | 2026-08-18 | fix: update LMU registration links in README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.136 | `3230e37023db` | 2026-08-18 | fix(claude): accept both `max_tokens` and `max_completion_tokens` in OpenAI→Claude request conversion | `internal/translator` |  |  |
| v7.2.136 | `20f84e78c162` | 2026-08-18 | fix(gemini): support mixed and structured `function_call_output` handling in OpenAI→Gemini responses translati | `internal/translator` |  |  |
| v7.2.136 | `628f535aac35` | 2026-08-18 | docs(readme): link Bestproxy free trial | `README.md` |  |  |
| v7.2.136 | `45c90e8d0ed8` | 2026-08-18 | fix(websocket): drop consumed compaction triggers | `sdk/api` |  |  |
| v7.2.136 | `d3a5988fc07d` | 2026-08-18 | docs: temporarily hide Qiniu Cloud and FennoAI sponsors | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.136 | `781544961d09` | 2026-08-18 | add bestproxy sponser | `README.md`, `assets/bestproxy.png` |  |  |
| v7.2.136 | `497673bf6bda` | 2026-08-19 | feat(antigravity): add `video_url` content part support to OpenAI request conversion | `internal/translator` |  |  |
| v7.2.137 | `0f69f09a70b0` | 2026-08-16 | fix(logging): exclude HTTP 499 and client cancellations from forced error logs | `internal/api`, `internal/clienterror`, `internal/runtime` |  |  |
| v7.2.137 | `788e9b792852` | 2026-08-19 | fix(claude): restore tool_search_tool_result references and recognize advisor/agent server tools (#5044) (#508 | `internal/runtime` |  |  |
| v7.2.137 | `ec105dac9416` | 2026-08-19 | fix(cliproxy): handle `responses/compact` auth cooldowns and fallback semantics | `sdk/api`, `sdk/cliproxy` |  |  |
| v7.2.137 | `2005788fc396` | 2026-08-19 | fix(codex): filter out `prompt_cache_retention` in OpenAI→Codex responses request conversion | `internal/translator` |  |  |
| v7.2.137 | `ac0d1888c04e` | 2026-08-19 | fix(gemini): strip 'encrypted' metadata from tool parameters schemas (#5065) | `internal/runtime`, `internal/util` |  |  |
| v7.2.137 | `5fef17e2ecd1` | 2026-08-19 | fix(claude): pass incoming headers when applying custom headers in caller-owned mode | `internal/runtime` |  |  |
| v7.2.137 | `79ef3618d650` | 2026-08-19 | fix(antigravity): attach sibling tool images to the nearest functionResponse (#5075) | `internal/translator` |  |  |
| v7.2.137 | `62f5a2798c52` | 2026-08-19 | fix(executor): prepend empty user turn for model-first Gemini/Antigravity requests (#4959) (#5048) | `internal/runtime` |  |  |
| v7.2.137 | `55397bf68d01` | 2026-08-19 | fix: remove temporarily hidden sponsorship details from README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.137 | `85d2faddd17e` | 2026-08-20 | fix(claude): preserve native subagent and environment headers (#4982) (#5084) | `internal/runtime` |  |  |
| v7.2.137 | `8aa6868d0dc1` | 2026-08-20 | fix(claude): support setup-tokens and gracefully handle 403 OAuth profile errors (#4983) (#5083) | `internal/runtime` |  |  |
