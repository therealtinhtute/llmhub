# Context: Module Path Migration + Product Rebrand

Phase: module-rebrand
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: build

## Goal
Migrate Go module path from `github.com/router-for-me/CLIProxyAPI/v7` to `github.com/therealtinhtute/llmhub`, rename product identity to LLMHub, change default auth directory, remove dead config key, and rename build artifacts. The binary must compile and the test suite must pass after this phase.

## Scope Boundary
### Allowed Surfaces
- `go.mod` — module line only (leave dependency lines untouched)
- All `*.go` files — import paths and the specific constants/strings listed below
- `internal/config/config.go` — `DefaultAuthDir`, `DefaultPanelGitHubRepository`, `RemoteManagement.PanelGitHubRepository` field
- `cmd/server/main.go` — startup banner string (line 58)
- `.goreleaser.yml` — binary name, build ID
- `docker-compose.yml` — volume mount path default
- `Dockerfile` — binary name reference if present
- `internal/managementasset/updater.go` — `httpUserAgent` constant only (rename user-agent string)

### Forbidden Surfaces
- `web/` — does not exist yet, belongs to embed-panel phase
- `README.md`, `README_CN.md`, `README_JA.md` — belongs to doc-cleanup phase
- `config.example.yaml` — belongs to doc-cleanup phase
- Provider logic, auth flows, routing, TUI, SDK — no behavioral changes
- `go.sum` — regenerated automatically by `go mod tidy`, not manually edited
- Management API route handlers — no endpoint changes

## Spec Hooks
- Requirement 7: module path change to `github.com/therealtinhtute/llmhub`
- Requirement 8: product-visible identity changed to LLMHub
- Requirement 9: auth storage path `~/.cli-proxy-api` → `~/.llmhub`
- Requirement 11: `panel-github-repository` removed from target architecture
- Requirement 16: no deep runtime redesign

## Locked Decisions
- New module path drops `/v7` version suffix — the fork starts fresh versioning
- `PanelGitHubRepository` field and constant are removed entirely, not deprecated
- The `DefaultPanelGitHubRepository` constant and its `isKnownDefaultValue` case are removed
- `LoadConfig` default assignment `cfg.RemoteManagement.PanelGitHubRepository = DefaultPanelGitHubRepository` is removed
- Config sanitizer line `cfg.RemoteManagement.PanelGitHubRepository = ...` is removed
- `resolveReleaseURL` in `updater.go` still uses `defaultManagementReleaseURL` as a hardcoded fallback — this is acceptable because the embed-panel phase will replace it entirely
- Auth dir rename is a constant change only — no runtime migration of existing `~/.cli-proxy-api` directories
- Goreleaser binary name becomes `llmhub`
- Docker volume mount default changes from `.cli-proxy-api` to `.llmhub`

## Assumptions
- The new module path `github.com/therealtinhtute/llmhub` does not conflict with any existing Go module
- No external consumers depend on the current module path (fork-owned)
- Tests that reference the old auth dir path (if any) will fail and need updating
- `go mod tidy` after import path replacement will produce a clean `go.sum`

## Canonical Refs
- `.kit/planning/SPEC.md` — requirements 7, 8, 9, 11
- `go.mod:1` — current module line
- `internal/config/config.go:23-26` — constants to change/remove
- `internal/config/config.go:194-207` — `RemoteManagement` struct
- `internal/config/config.go:648` — default assignment to remove
- `internal/config/config.go:687-689` — sanitizer line to remove
- `internal/config/config.go:1380-1383` — `isKnownDefaultValue` case to remove
- `cmd/server/main.go:58` — startup banner
- `.goreleaser.yml:4,16` — build ID and binary name
- `internal/managementasset/updater.go:31` — `httpUserAgent` constant

## Rejected Options
- Keep `/v7` in module path — rejected because the fork is a new product, not a v7 continuation
- Deprecate `PanelGitHubRepository` with a warning instead of removing — rejected because embedded panel makes it meaningless (spec requirement 11)
- Rename the entire `managementasset` package now — rejected, belongs to embed-panel phase

## Deferred Ideas
- Runtime migration helper for `~/.cli-proxy-api` → `~/.llmhub` (could be a Phase 3 feature)
- SDK package path renaming for downstream consumers (post-phase-1)

## Escalate If
- `go build ./...` fails after import replacement due to circular or missing dependencies
- Tests reference hardcoded upstream URLs that need updating but touch provider logic
- More than 5 files outside the allowed surfaces need changes to compile
