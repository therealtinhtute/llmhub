# Implementation Notes

Phase: module-rebrand
Started: 2026-05-27

## Wave 1 — T2 extended
- Found 2 non-import `router-for-me/CLIProxyAPI` string references beyond bulk sed scope: release URL in `config_basic.go` and comment URL in `openai_responses_websocket.go`. Updated both to `therealtinhtute/llmhub`.
- Also caught and renamed `latestReleaseUserAgent` from `CLIProxyAPI` to `LLMHub` in `config_basic.go`.

## Wave 2 — T3/T4 extended
- `PanelGitHubRepository` had references in 5 additional files beyond the plan's list: `parse.go`, `config_diff.go`, `config_diff_test.go`, `server.go`, `sdk/config/config.go`. All cleaned up.
- Identity renames extended beyond plan's 3 files: also updated TUI i18n strings, gitstore git author, home/certificate.go auth dir path, and package doc comments.
- **Deliberately kept as-is** (7 references): auth header values (`xai.go` referrer, `kimi.go` X-Msh-Platform), cache key prefix (`codex_executor.go`), device identifiers (`kimi_executor.go`). These are sent to external APIs — changing them risks breaking authentication flows or invalidating caches.
- `sdk/config/config.go` re-export of `DefaultPanelGitHubRepository` removed.
