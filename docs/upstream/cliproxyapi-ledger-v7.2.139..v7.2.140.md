# Upstream ledger — cliproxyapi v7.2.139..v7.2.140

- generated: 2026-08-23T09:00:59Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `ba6738dabe3b`
- non-merge commits: 13

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.140 | `a7e3596b7e35` | 2026-08-22 | fix(gemini): normalize malformed schema nodes before cleanup | `internal/util` |  |  |
| v7.2.140 | `4d68ca8a6349` | 2026-08-22 | fix(codex): convert Grok client keepalive SSE frames to comments | `internal/client`, `internal/runtime` |  |  |
| v7.2.140 | `d5b57a2d8ac0` | 2026-08-22 | fix(openai): validate and filter thought signatures in responses conversion | `internal/translator` |  |  |
| v7.2.140 | `ebda7509114d` | 2026-08-22 | feat(auth): enhance key deletion and patching with base URL validation | `internal/api` |  |  |
| v7.2.140 | `ab8f00dbd91f` | 2026-08-22 | fix(claude): derive stable request-scoped `metadata.user_id` for converters | `internal/translator` |  |  |
| v7.2.140 | `b3f72cef6565` | 2026-08-22 | fix(gemini): normalize Claude thinking signatures in Gemini request and response conversion | `internal/translator` |  |  |
| v7.2.140 | `a834917e871d` | 2026-08-22 | fix(antigravity): raise the fallback client version to 2.9.1 (#5175) | `internal/misc` |  |  |
| v7.2.140 | `65071f7c47d4` | 2026-08-22 | test(claude): expand OpenAI conversion tests for `reasoning_content` and stream/non-stream parity | `internal/translator` |  |  |
| v7.2.140 | `d9869ed9085d` | 2026-08-22 | test(openai): expand reasoning fallback coverage in OpenAI responses non-stream conversion tests | `internal/translator` |  |  |
| v7.2.140 | `87fb01b23788` | 2026-08-22 | fix(xai): drop orphaned tool_choice after compact strips tools | `internal/runtime` |  |  |
| v7.2.140 | `dfdf183fcfb6` | 2026-08-22 | fix(xai): keep image_generation on grok-4.6+ conversation requests | `internal/runtime` |  |  |
| v7.2.140 | `71c3c144a078` | 2026-08-22 | fix(registry): detect gemini interactions changes | `internal/registry` |  |  |
| v7.2.140 | `e04d620cc1a5` | 2026-08-22 | feat(auth): normalize credential metadata keys | `config.example.yaml`, `internal/api`, `internal/pluginhost` +5 |  |  |
