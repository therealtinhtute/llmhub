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

---

Phase: doc-cleanup
Started: 2026-05-27

## Wave 1 — README rebrand approach
- Full rewrite of all three READMEs rather than surgical edits. The sponsor sections were deeply interleaved with branding, making surgical edits error-prone.
- "Getting Started" links to `help.router-for.me` replaced with local config reference note ("Documentation is being migrated"). The external doc site is tied to the old project.
- Third-party project names containing "CLIProxyAPI" (Tray, Dashboard, Quota Inspector) deliberately kept — these are external GitHub repo names we don't control.
- Telegram group URL `t.me/CLIProxyAPI` kept in CN README — it's an external resource.
- Sections renamed: "Who is with us?" → "Community Projects", "谁与我们在一起？" → "社区项目", "関連プロジェクト" → "コミュニティプロジェクト".

## Wave 2 — config.example.yaml
- Removed `panel-github-repository` and `disable-auto-update-panel` entries entirely since the panel is now embedded (no GitHub download needed).
- Updated `disable-control-panel` comment from "asset download" to "bundled management control panel".

## Wave 2 — .github/ CI files (extended scope)
- Extended T4 beyond the plan's `assets/` cleanup to also fix `.github/` CI files. These had `router-for-me/models.git` references in 3 workflow files and old branding in FUNDING.yml and docker-image.yml.
- DOCKERHUB_REPO changed from `eceasy/cli-proxy-api` to `eceasy/llmhub` — matches the docker-compose.yml change from embed-panel phase.
