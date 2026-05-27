# Plan: Module Path Migration + Product Rebrand

Phase: module-rebrand
Status: ready
Wave Count: 4
Execution Owner: work
Updated At: 2026-05-27

## Goal
Migrate Go module path, rename product identity to LLMHub, change auth dir default, remove `PanelGitHubRepository` config, and rename build artifacts. Binary must compile clean.

## Inputs
- `go.mod` current module path: `github.com/router-for-me/CLIProxyAPI/v7`
- `internal/config/config.go` constants at lines 23-26
- `internal/config/config.go` RemoteManagement struct at lines 194-207
- `cmd/server/main.go` banner at line 58
- `.goreleaser.yml` build config
- `docker-compose.yml` volume mount

## Wave 1
### T1 — Migrate go.mod module line
- type: migration
- inputs:
  - `go.mod:1`
- touches:
  - `go.mod`
- avoid:
  - dependency lines in go.mod — only change the `module` line
- steps:
  1. Change `module github.com/router-for-me/CLIProxyAPI/v7` to `module github.com/therealtinhtute/llmhub`
- expected outputs:
  - `go.mod` line 1 reads `module github.com/therealtinhtute/llmhub`
- verification:
  - `head -1 go.mod` shows `module github.com/therealtinhtute/llmhub`
- stop if:
  - go.mod has unexpected structure or multiple module declarations
- escalate to:
  - user clarification

### T2 — Bulk-replace import paths in all Go files
- type: migration
- inputs:
  - All `*.go` files referencing `github.com/router-for-me/CLIProxyAPI/v7`
- touches:
  - All `*.go` files (~964 import lines)
- avoid:
  - `go.sum`, `vendor/`, `.git/`
  - changing anything other than the import string literal
- steps:
  1. Run: `find . -name "*.go" -not -path "./.git/*" -exec sed -i '' 's|github.com/router-for-me/CLIProxyAPI/v7|github.com/therealtinhtute/llmhub|g' {} +`
  2. Verify no partial replacements or missed files: `grep -rn "router-for-me/CLIProxyAPI" --include="*.go" | wc -l` should be 0
- expected outputs:
  - Zero remaining references to old module path in Go source
- verification:
  - `grep -rn "router-for-me/CLIProxyAPI" --include="*.go"` returns empty
- stop if:
  - grep finds remaining references after sed
- escalate to:
  - plan phase (inspect missed patterns)

## Wave 2
### T3 — Rename product identity constants and strings
- type: refactor
- inputs:
  - `internal/config/config.go:25` — `DefaultAuthDir`
  - `cmd/server/main.go:58` — banner string
  - `internal/managementasset/updater.go:31` — `httpUserAgent`
- touches:
  - `internal/config/config.go`
  - `cmd/server/main.go`
  - `internal/managementasset/updater.go`
- avoid:
  - any config struct field names or YAML tag changes beyond PanelGitHubRepository removal
  - any management API handler logic
- steps:
  1. Change `DefaultAuthDir = "~/.cli-proxy-api"` to `DefaultAuthDir = "~/.llmhub"` in `internal/config/config.go:25`
  2. Change banner `fmt.Printf("CLIProxyAPI Version: %s, Commit: %s, BuiltAt: %s\n", ...)` to `fmt.Printf("LLMHub Version: %s, Commit: %s, BuiltAt: %s\n", ...)` in `cmd/server/main.go:58`
  3. Change `httpUserAgent = "CLIProxyAPI-management-updater"` to `httpUserAgent = "LLMHub-management-updater"` in `internal/managementasset/updater.go:31`
- expected outputs:
  - `DefaultAuthDir` reads `~/.llmhub`
  - Banner prints `LLMHub Version: ...`
  - HTTP user-agent reads `LLMHub-management-updater`
- verification:
  - `grep -n "cli-proxy-api" internal/config/config.go` returns no matches for DefaultAuthDir
  - `grep -n "CLIProxyAPI" cmd/server/main.go` returns no matches
  - `grep -n "CLIProxyAPI" internal/managementasset/updater.go` returns no matches
- stop if:
  - other files reference `DefaultAuthDir` by value string instead of constant
- escalate to:
  - plan phase

### T4 — Remove PanelGitHubRepository config field
- type: refactor
- inputs:
  - `internal/config/config.go` — constant, struct field, LoadConfig defaults, sanitizer, isKnownDefaultValue
- touches:
  - `internal/config/config.go`
- avoid:
  - other RemoteManagement fields (DisableControlPanel, DisableAutoUpdatePanel, AllowRemote, SecretKey)
  - management API handler code that reads the panel repo value (will be handled in embed-panel)
- steps:
  1. Remove `DefaultPanelGitHubRepository` constant (line 23)
  2. Remove `PanelGitHubRepository` field from `RemoteManagement` struct (line 206)
  3. Remove `cfg.RemoteManagement.PanelGitHubRepository = DefaultPanelGitHubRepository` default assignment in `LoadConfigOptional` (line 648)
  4. Remove the sanitizer block: `cfg.RemoteManagement.PanelGitHubRepository = strings.TrimSpace(...)` and the empty-check fallback (lines 687-689)
  5. Remove the `"remote-management.panel-github-repository"` case from `isKnownDefaultValue` (lines 1380-1383)
  6. Fix any compile errors from references to the removed field (update `EnsureLatestManagementHTML` call signatures if they pass PanelGitHubRepository — pass empty string or hardcode the URL in updater.go since it will be removed in embed-panel phase)
- expected outputs:
  - `RemoteManagement` struct has no `PanelGitHubRepository` field
  - No `DefaultPanelGitHubRepository` constant
  - Code compiles with the field removed
- verification:
  - `grep -rn "PanelGitHubRepository\|panel-github-repository" --include="*.go"` — only `updater.go` internal references remain (its own `resolveReleaseURL` and `defaultManagementReleaseURL`, which will be removed in embed-panel)
  - `go build ./...` passes
- stop if:
  - more than 5 call sites reference PanelGitHubRepository outside config.go and server.go
- escalate to:
  - plan phase

## Wave 3
### T5 — Rename build and deployment artifacts
- type: refactor
- inputs:
  - `.goreleaser.yml`
  - `docker-compose.yml`
  - `docker-compose.cluster.yml` (if present and references old names)
  - `Dockerfile`
- touches:
  - `.goreleaser.yml`
  - `docker-compose.yml`
  - `docker-compose.cluster.yml`
  - `Dockerfile`
- avoid:
  - changing build behavior or CI logic
  - changing Docker base images
- steps:
  1. In `.goreleaser.yml`: change `id: "cli-proxy-api"` to `id: "llmhub"`, change `binary: cli-proxy-api` to `binary: llmhub`, change archive `id: "cli-proxy-api"` to `id: "llmhub"`
  2. In `docker-compose.yml`: change `${CLI_PROXY_AUTH_PATH:-./auths}:/root/.cli-proxy-api` to `${LLMHUB_AUTH_PATH:-./auths}:/root/.llmhub`
  3. In `docker-compose.cluster.yml`: apply same auth path rename if referenced
  4. In `Dockerfile`: rename binary references from `cli-proxy-api` to `llmhub` if present
- expected outputs:
  - Goreleaser builds `llmhub` binary
  - Docker compose mounts to `~/.llmhub`
- verification:
  - `grep -n "cli-proxy-api" .goreleaser.yml docker-compose.yml docker-compose.cluster.yml Dockerfile 2>/dev/null` returns empty
- stop if:
  - Dockerfile structure is complex and binary name isn't a simple replace
- escalate to:
  - user clarification

## Wave 4
### T6 — Run go mod tidy and full build verification
- type: test
- inputs:
  - All changes from Waves 1-3
- touches:
  - `go.sum` (regenerated)
- avoid:
  - manual edits to go.sum
- steps:
  1. Run `go mod tidy`
  2. Run `go build ./...`
  3. Run `go vet ./...`
  4. Run `go test ./...` (if test suite exists and is meaningful)
  5. Verify no remaining old-path references: `grep -rn "router-for-me/CLIProxyAPI" --include="*.go" | wc -l` = 0
  6. Verify no remaining old auth-dir constant references: `grep -rn '\.cli-proxy-api' --include="*.go"` — should only appear in migration comments if any
- expected outputs:
  - Clean build with zero errors
  - Zero old module path references in Go source
- verification:
  - `go build ./...` exits 0
  - `go vet ./...` exits 0
  - grep counts are 0
- stop if:
  - build fails — investigate the specific errors before proceeding
  - tests fail on something unrelated to the rename (pre-existing failure)
- escalate to:
  - check (run `/check` for code review before merging)

## Risks / Watch-fors
- The `v7` suffix removal means any code using `import/v7/some/package` patterns will break if not caught by the bulk sed — verify with grep
- `EnsureLatestManagementHTML` and `serveManagementControlPanel` in `server.go` pass `cfg.RemoteManagement.PanelGitHubRepository` — those call sites need updating when the field is removed. Use `""` (empty string) as the argument, which will fall through to `defaultManagementReleaseURL` in `resolveReleaseURL`. This is a temporary bridge until embed-panel removes the download path entirely.
- `docker-compose.cluster.yml` may have different volume structure — inspect before blindly renaming
