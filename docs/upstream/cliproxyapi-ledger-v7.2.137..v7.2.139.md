# Upstream ledger — cliproxyapi v7.2.137..v7.2.139

- generated: 2026-08-22T04:33:17Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `3e8cb015b428`
- non-merge commits: 18

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.138 | `9d6b5cdd163b` | 2026-08-20 | fix(xai): treat response.incomplete as terminal success (#5113) | `internal/runtime` |  |  |
| v7.2.138 | `4b9d404fb04f` | 2026-08-20 | feat(codex): add opt-in stream bootstrap buffering and overload failover (#5115) | `config.example.yaml`, `internal/config`, `internal/runtime` +2 |  |  |
| v7.2.138 | `92d96e0b729d` | 2026-08-20 | fix(cliproxy): support base_URL-only config credentials and skip stale auth headers | `internal/api`, `internal/config`, `internal/runtime` +2 |  |  |
| v7.2.138 | `556328c12253` | 2026-08-20 | feat(gemini): add namespace-aware OpenAI Responses tool resolution and custom tool call conversion | `internal/translator`, `internal/util` |  |  |
| v7.2.138 | `48749717645e` | 2026-08-20 | fix(cliproxy): add warn-level diagnostics for auth cooldown and upstream execution failures | `sdk/cliproxy` |  |  |
| v7.2.138 | `9dc51b1f8777` | 2026-08-20 | feat(cliproxy): add OAuth request-scoped error rules support | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
| v7.2.138 | `1d5b7612c6ba` | 2026-08-21 | fix(cliproxy): add protocol-aware plugin executor usage parsing for response and streaming payloads | `sdk/api` |  |  |
| v7.2.138 | `4053c026e79c` | 2026-08-21 | fix(executor): sanitize thought signatures in Gemini and Gemini Vertex executors (#5110) | `internal/runtime` |  |  |
| v7.2.138 | `b1c000590b47` | 2026-08-21 | fix(gemini): include empty annotations/logprobs in `response.output_item.done` message content | `internal/translator` |  |  |
| v7.2.138 | `68e96c27165e` | 2026-08-21 | fix(openai): map `max_completion_tokens` to Antigravity `maxOutputTokens` | `internal/translator` |  |  |
| v7.2.138 | `5b232e3e981b` | 2026-08-21 | fix(gemini): generate deterministic sequential IDs for Gemini tool call pairing in Codex/Claude conversions | `internal/translator` |  |  |
| v7.2.138 | `3db591eecd6a` | 2026-08-21 | fix(gemini): preserve Gemini thought signatures in non-stream Claude conversion | `internal/translator` |  |  |
| v7.2.138 | `8eb3ac2e036b` | 2026-08-21 | `fix(openai): fallback to `reasoning` when `reasoning_content` is missing in response conversion` | `internal/translator` |  |  |
| v7.2.138 | `aa5dccc23688` | 2026-08-21 | fix(claude): use `message.reasoning_content` for converted thinking output | `internal/translator` |  |  |
| v7.2.138 | `42d8e746e540` | 2026-08-21 | fix(claude): avoid long global cooldowns for Fable-only 7d_oi rate limits | `internal/runtime` |  |  |
| v7.2.139 | `0a14eb70ce19` | 2026-08-21 | feat(auth): add retry round credential filtering | `config.example.yaml`, `internal/home`, `sdk/cliproxy` |  |  |
| v7.2.139 | `601ca43090f7` | 2026-08-21 | feat(auth): add credential retry round contract | `config.example.yaml`, `internal/config`, `internal/home` +2 |  |  |
| v7.2.139 | `85e7add6adf3` | 2026-08-21 | feat(models): add Gemini 3.7 Flash model registrations to model registry | `internal/registry` |  |  |
