# Upstream ledger — cliproxyapi v7.2.112..v7.2.113

- generated: 2026-08-02T09:50:21Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `6984b74e3151`
- non-merge commits: 7

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.113 | `8c37dd986289` | 2026-07-31 | feat(executor): introduce Kimi Thinking Replay cache and continuity mechanisms | `internal/cache`, `internal/runtime` | `adapt` | adapted as bounded Kimi Claude-path replay via `internal/cache/kimi_thinking_replay_cache.go`, `internal/runtime/executor/kimi_thinking_replay.go`, and `internal/runtime/executor/kimi_executor.go`; verified by `go test ./internal/cache ./internal/runtime/executor` |
| v7.2.113 | `d9460a8df6c1` | 2026-07-31 | feat: add sponser LMU | `README.md`, `README_CN.md`, `README_JA.md` +1 | `reject` | sponsor/docs-only churn remains excluded by active-plan non-goal NG3; no product surface changed |
| v7.2.113 | `91561df0b16b` | 2026-08-01 | fix(codex): align websocket cloaking headers | `config.example.yaml`, `internal/config`, `internal/runtime` | `already-present` | local Codex websocket executor already applies shared cloaking/header defaults and prompt-cache continuity in `internal/runtime/executor/codex_websockets_executor.go`; focused executor tests passed |
| v7.2.113 | `08eb05ae8730` | 2026-08-01 | feat(config): add `support-prompt-cache-key` option for OpenAI compatibility | `config.example.yaml`, `internal/api`, `internal/config` +2 | `adapt` | added `support-prompt-cache-key` config/management/diff handling and gated OpenAI-compatible executor propagation in `internal/config/config.go`, `internal/api/handlers/management/`, `internal/watcher/diff/openai_compat.go`, and `internal/runtime/executor/openai_compat_executor.go`; focused tests passed |
| v7.2.113 | `3d4e1604959d` | 2026-08-01 | feat(executor): add conditional support for Claude fast mode beta | `internal/runtime` | `already-present` | local Claude executor already carries `fast-mode-2026-02-01` in `Anthropic-Beta` handling and preserves request-provided beta merging in `internal/runtime/executor/claude_executor.go` |
| v7.2.113 | `4d498ec7f715` | 2026-08-01 | feat(translator): enhance OpenAI response handling for structured tool outputs | `internal/translator` | `already-present` | OpenAI Responses structured tool-output repair is already implemented in local response framing/translation paths and covered by existing focused handler/translator tests; no duplicate implementation added |
| v7.2.113 | `198a26737c09` | 2026-08-01 | feat(codex): add Alpha Search API key support with configurable endpoint | `config.example.yaml`, `internal/api`, `internal/config` +3 | `adapt` | added Codex `alpha-search` config synthesis, management/diff support, auth selection filtering, `/backend-api/codex/alpha/search`, and raw executor POST to `/alpha/search`; verified by focused config/management/watcher/executor/auth/API tests |
