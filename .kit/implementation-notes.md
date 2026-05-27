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

---

Phase: embed-panel
Started: 2026-05-27

## Wave 2 — T3 embed strategy
- `internal/managementasset/static/` added to `.gitignore` as a generated artifact (rebuilt from `web/` source).
- The `static/management.html` file must exist at `go build` time — CI and goreleaser hooks handle this.

## Wave 3 — T4 updater.go gutting
- Gutted entire `updater.go` to just 2 constant declarations (`managementAssetName`, `ManagementFileName`).
- Removed `SetCurrentConfig` from `server.go` (2 call sites) and `main.go` (1 call site) — no longer needed since handler reads `s.cfg` directly and doesn't check config for download decisions.
- Removed `managementasset` import from `main.go` entirely.

## Wave 4 — Dockerfile multi-stage
- Added `oven/bun:1` as a frontend build stage to Dockerfile. The `COPY --from=frontend` step copies the built HTML into the Go build context before `go build`.
- Also updated `.gitignore` binary name from `cli-proxy-api` to `llmhub` (missed in module-rebrand).
