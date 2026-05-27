# Context: Import + Build + Embed Management Panel

Phase: embed-panel
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: build, integration (runtime verify panel loads at /management.html)

## Goal
Import the upstream CPAMC React panel source into `web/`, build it with Bun/Vite into a single HTML file, embed that file into the Go binary via `go:embed`, and replace the runtime download serving path with embedded asset delivery. After this phase, the Go binary serves the management UI from embedded bytes with zero external dependencies.

## Scope Boundary
### Allowed Surfaces
- `web/` — new directory, entire CPAMC source import
- `internal/managementasset/` — rewrite: replace `updater.go` download logic with `embed.go` + `static/` embed
- `internal/api/server.go` — `serveManagementControlPanel` handler (lines 731-758), import of managementasset
- `cmd/server/main.go` — remove `StartAutoUpdater` calls (lines 566, 644), remove managementasset import if unused
- `.goreleaser.yml` — add `before.hooks` to build frontend before Go compile
- `.gitignore` — add `web/dist/`, `web/node_modules/`, `internal/managementasset/static/`
- `Makefile` or `scripts/build.sh` — optional build helper for dev workflow

### Forbidden Surfaces
- Management API route handlers (`/v0/management/*`) — no endpoint changes
- Provider logic, auth flows, routing, TUI, SDK
- `internal/config/config.go` — no further config changes this phase
- `README.md` — belongs to doc-cleanup phase
- `web/` source code beyond minimal changes required for in-repo build
- React component logic or styling — Phase 2 work

## Spec Hooks
- Requirement 1: `web/` is the single source of truth for frontend
- Requirement 3: import with minimal restructuring
- Requirement 4: production build uses `go:embed`
- Requirement 5: stop serving runtime-downloaded file
- Requirement 6: `/management.html` remains the entrypoint
- Requirement 10: management API routes preserved
- Requirement 12: `disable-control-panel` remains meaningful

## Locked Decisions
- CPAMC source is copied into `web/` without git history (clean monorepo ownership)
- Package manager: Bun (matches upstream, fastest)
- Build output: single `web/dist/index.html` via `vite-plugin-singlefile` (already configured upstream)
- Embed strategy: copy `web/dist/index.html` → `internal/managementasset/static/management.html`, then `//go:embed static/management.html` in `internal/managementasset/embed.go`
- This copy step is required because `go:embed` cannot traverse `../` paths
- `updater.go` is gutted: all GitHub download functions removed, `SetCurrentConfig` stays if needed for `DisableControlPanel` checks, `StartAutoUpdater` removed
- `serveManagementControlPanel` changes from `c.File(filePath)` to writing embedded bytes with correct Content-Type
- `DisableControlPanel` check stays in the handler (lines 733-735) — returns 404 when true
- `.goreleaser.yml` gets a `before.hooks` entry to build frontend + copy to embed path
- Dev workflow: a `Makefile` target or script does the same build+copy for local development

## Assumptions
- CPAMC builds successfully with `bun install && bun run build` inside `web/` without modifications
- CPAMC's API calls target the same management routes (`/v0/management/*`) that the Go server already serves
- `vite-plugin-singlefile` produces a single `index.html` with all JS/CSS/assets inlined
- Bun is available on the dev machine and CI runners
- The single HTML file will be under the `go:embed` practical size limit (~50MB, CPAMC is likely <5MB)

## Canonical Refs
- `.kit/planning/SPEC.md` — requirements 1, 3, 4, 5, 6, 12
- `internal/managementasset/updater.go` — entire file, current download logic
- `internal/api/server.go:731-758` — current panel handler
- `cmd/server/main.go:566,644` — `StartAutoUpdater` calls
- CPAMC repo: `https://github.com/router-for-me/Cli-Proxy-API-Management-Center`
- CPAMC tech stack: React 19 + TypeScript + Vite 8 + vite-plugin-singlefile + Bun

## Rejected Options
- Put `//go:embed` in root-level package — rejected: keeps embed coupled to the package that uses it
- Use `io/fs` sub-filesystem for serving — rejected: single file doesn't need filesystem abstraction
- Keep `updater.go` as a fallback download path — rejected: embedded asset makes download dead code, keeping it increases maintenance surface
- Vendor build output instead of building from source — rejected: breaks one-repo source-of-truth requirement
- Use npm/pnpm instead of bun — rejected: CPAMC uses bun natively, switching adds lockfile friction

## Deferred Ideas
- CPAMC source code modifications for LLMHub branding — Phase 2
- React component restructuring — Phase 2
- CSS/style system changes — Phase 2
- Hashed asset filenames for cache busting — Phase 2

## Escalate If
- CPAMC `bun run build` fails due to missing dependencies or Node.js version requirements
- CPAMC API calls target routes that don't exist on the Go server
- Embedded HTML file is >10MB (investigate why vite-plugin-singlefile produced outsized output)
- `go:embed` fails to include the file (path resolution issue)
