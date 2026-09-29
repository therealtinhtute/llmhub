# Upstream ledger — cliproxyapi v7.3.20..v8.0.3

- generated: 2026-09-28T10:52:05Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `f19668f8a00c`
- non-merge commits: 14

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v8.0.0 | `31f4cfab3fc1` | 2026-09-27 | feat: migrate CLIProxyAPI to version 8 | `cmd/fetch_antigravity_models`, `cmd/fetch_codex_models`, `cmd/fetch_devin_models` +48 |  |  |
| v8.0.0 | `b1a33b80dc4c` | 2026-09-27 | docs: prune obsolete management notes | `.gitignore`, `config.example.yaml`, `docs/STEERING.md` +2 |  |  |
| v8.0.0 | `9a0502c87ec3` | 2026-09-27 | feat(api): use query params for v8 oauth routes | `config.example.yaml`, `internal/api` |  |  |
| v8.0.0 | `d9922f058902` | 2026-09-27 | feat(config): add v8 configuration API | `config.example.yaml`, `internal/api`, `internal/client` +5 |  |  |
| v8.0.1 | `6c385e606957` | 2026-09-27 | fix(config): drop obsolete legacy keys during v8 migration | `internal/api`, `internal/config` |  |  |
| v8.0.2 | `4a2c81864f31` | 2026-09-27 | fix(config): preserve unknown legacy sections as comments during v8 migration | `internal/api`, `internal/config` |  |  |
| v8.0.3 | `2c27e61fe791` | 2026-09-27 | fix(meta): strip unsupported search_content_types from web search tools | `internal/runtime` |  |  |
| v8.0.3 | `01274e226ed8` | 2026-09-27 | feat(config): warn on unrecognized sections during v8 migration | `internal/config`, `internal/logging` |  |  |
| v8.0.3 | `5756d76ef3da` | 2026-09-27 | fix(pluginhost): preserve disabled state in host auth callbacks | `internal/pluginhost` |  |  |
| v8.0.3 | `acdace936fa7` | 2026-09-28 | fix(discovery): support bonded interfaces and deduplicate build warnings | `internal/discovery`, `sdk/cliproxy` |  |  |
| v8.0.3 | `04f88169ada8` | 2026-09-28 | fix(gemini): preserve thought signatures in interactions translation | `internal/translator` |  |  |
| v8.0.3 | `0a45f2533440` | 2026-09-28 | fix(translator): append trailing newlines to SSEEventData frames | `internal/translator` |  |  |
| v8.0.3 | `4ad5bba756f9` | 2026-09-28 | fix(openai): prevent false positive namespace prefix matches in responses tools | `internal/translator` |  |  |
| v8.0.3 | `4b99d1d6f886` | 2026-09-28 | feat(openai): support quality parameter for xAI image requests | `sdk/api` |  |  |
