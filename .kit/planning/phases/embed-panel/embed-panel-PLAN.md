# Plan: Import + Build + Embed Management Panel

Phase: embed-panel
Status: ready
Wave Count: 5
Execution Owner: work
Updated At: 2026-05-27

## Goal
Import CPAMC source into `web/`, build single HTML, embed into Go binary, replace download serving with embedded delivery. `/management.html` serves the panel from memory. `disable-control-panel` still works.

## Inputs
- Phase module-rebrand completed (module path is `github.com/therealtinhtute/llmhub`)
- CPAMC source: `https://github.com/router-for-me/Cli-Proxy-API-Management-Center`
- `internal/managementasset/updater.go` — download logic to replace
- `internal/api/server.go:731-758` — panel handler to rewrite
- `cmd/server/main.go:566,644` — auto-updater calls to remove

## Wave 1
### T1 — Import CPAMC source into web/
- type: migration
- inputs:
  - CPAMC git repository
- touches:
  - `web/` (new directory, entire source)
  - `.gitignore`
- avoid:
  - modifying CPAMC source files
  - importing `.git/` directory
- steps:
  1. Clone CPAMC: `git clone https://github.com/router-for-me/Cli-Proxy-API-Management-Center /tmp/cpamc-source`
  2. Copy source (excluding .git): `rsync -a --exclude='.git' /tmp/cpamc-source/ web/`
  3. Clean up: `rm -rf /tmp/cpamc-source`
  4. Add to `.gitignore`: `web/dist/`, `web/node_modules/`
- expected outputs:
  - `web/package.json` exists
  - `web/src/` contains React source
  - `web/vite.config.ts` exists
  - `.gitignore` includes `web/dist/` and `web/node_modules/`
- verification:
  - `ls web/package.json web/vite.config.ts web/src/main.tsx 2>/dev/null | wc -l` = 3
  - `grep "web/dist" .gitignore` returns a match
- stop if:
  - CPAMC repo is inaccessible or structure differs from expected
- escalate to:
  - user clarification (provide alternative source path)

### T2 — Build frontend with Bun
- type: implementation
- inputs:
  - `web/package.json`
- touches:
  - `web/node_modules/` (installed)
  - `web/dist/` (build output)
- avoid:
  - modifying any source files in `web/src/`
- steps:
  1. `cd web && bun install`
  2. `cd web && bun run build`
  3. Verify single file output: `ls -la web/dist/index.html`
  4. Check file size: `wc -c web/dist/index.html` — expect >50KB and <10MB
- expected outputs:
  - `web/dist/index.html` — single HTML file with all JS/CSS inlined
- verification:
  - `test -f web/dist/index.html && echo "OK"` prints OK
  - `wc -c < web/dist/index.html` is between 50000 and 10000000
- stop if:
  - `bun install` fails (check bun version, node compatibility)
  - `bun run build` fails (check vite config, TypeScript errors)
  - output is not a single file (check vite-plugin-singlefile config)
- escalate to:
  - user clarification (bun version, build errors)

## Wave 2
### T3 — Create embed infrastructure
- type: implementation
- inputs:
  - `web/dist/index.html` from T2
  - `internal/managementasset/` package
- touches:
  - `internal/managementasset/static/` (new directory)
  - `internal/managementasset/static/management.html` (copied from build)
  - `internal/managementasset/embed.go` (new file)
- avoid:
  - modifying `updater.go` yet (done in T4)
- steps:
  1. Create directory: `mkdir -p internal/managementasset/static`
  2. Copy build output: `cp web/dist/index.html internal/managementasset/static/management.html`
  3. Add `internal/managementasset/static/` to `.gitignore` (generated artifact, rebuilt from web/ source)
  4. Create `internal/managementasset/embed.go`:
     ```go
     package managementasset

     import _ "embed"

     //go:embed static/management.html
     var embeddedManagementHTML []byte

     // EmbeddedManagementHTML returns the compiled management panel as raw bytes.
     func EmbeddedManagementHTML() []byte {
         return embeddedManagementHTML
     }
     ```
- expected outputs:
  - `internal/managementasset/embed.go` exists with `go:embed` directive
  - `internal/managementasset/static/management.html` exists
- verification:
  - `go build ./internal/managementasset/` compiles without errors
- stop if:
  - `go:embed` fails to find the static file (path resolution issue)
- escalate to:
  - plan phase (embed path strategy)

## Wave 3
### T4 — Gut updater.go download logic
- type: refactor
- inputs:
  - `internal/managementasset/updater.go` — entire file
- touches:
  - `internal/managementasset/updater.go`
- avoid:
  - removing `SetCurrentConfig` if it's still used for DisableControlPanel checks
  - removing `ManagementFileName` constant if referenced elsewhere
- steps:
  1. Remove all download-related functions: `EnsureLatestManagementHTML`, `StartAutoUpdater`, `runAutoUpdater`, `ensureFallbackManagementHTML`, `fetchLatestAsset`, `downloadAsset`, `resolveReleaseURL`, `newHTTPClient`, `fileSHA256`, `atomicWriteFile`, `parseDigest`
  2. Remove download-related constants: `defaultManagementReleaseURL`, `defaultManagementFallbackURL`, `managementSyncMinInterval`, `updateCheckInterval`, `maxAssetDownloadSize`
  3. Remove download-related vars: `lastUpdateCheckMu`, `lastUpdateCheckTime`, `schedulerOnce`, `schedulerConfigPath`, `sfGroup`
  4. Remove download-related structs: `releaseAsset`, `releaseResponse`
  5. Keep: `managementAssetName` constant, `ManagementFileName` constant, `SetCurrentConfig` + `currentConfigPtr` (if used for DisableControlPanel), `StaticDir` and `FilePath` (evaluate — may still be useful for logging or may be removable)
  6. Remove unused imports resulting from the deletions
  7. If `updater.go` is reduced to just constants and SetCurrentConfig, consider renaming the file or consolidating into embed.go
- expected outputs:
  - No GitHub download functions remain in `managementasset` package
  - No HTTP client creation for panel download
  - Package still exports what the server handler needs
- verification:
  - `grep -n "EnsureLatestManagementHTML\|StartAutoUpdater\|downloadAsset\|fetchLatestAsset" internal/managementasset/*.go` returns empty
  - `go build ./internal/managementasset/` compiles
- stop if:
  - other packages outside server.go reference download functions
- escalate to:
  - plan phase (map all callers)

### T5 — Rewrite server panel handler to use embedded bytes
- type: implementation
- inputs:
  - `internal/api/server.go:731-758` — current handler
  - `internal/managementasset/embed.go` — new embed function
- touches:
  - `internal/api/server.go` — `serveManagementControlPanel` method
- avoid:
  - changing the route registration at line 374
  - changing management API routes
  - removing DisableControlPanel check
- steps:
  1. Rewrite `serveManagementControlPanel` to:
     - Keep the `DisableControlPanel` check (return 404)
     - Keep the `cfg.Home.Enabled` check (return 404)
     - Replace filesystem read with `managementasset.EmbeddedManagementHTML()`
     - Write bytes with `c.Data(http.StatusOK, "text/html; charset=utf-8", data)`
  2. Remove `os.Stat` + `EnsureLatestManagementHTML` fallback logic
  3. Remove `managementasset.FilePath` and `managementasset.StaticDir` calls from this handler
- expected outputs:
  - Handler serves embedded HTML bytes at `/management.html`
  - 404 when `DisableControlPanel` is true
  - No filesystem access in the handler
- verification:
  - `go build ./internal/api/` compiles
  - grep confirms no `FilePath\|StaticDir\|EnsureLatest` in the handler function
- stop if:
  - handler references configFilePath for purposes other than panel serving
- escalate to:
  - plan phase

### T6 — Remove StartAutoUpdater calls from main.go
- type: refactor
- inputs:
  - `cmd/server/main.go:566,644` — auto-updater start calls
- touches:
  - `cmd/server/main.go`
- avoid:
  - removing other managementasset calls (SetCurrentConfig may still be needed)
- steps:
  1. Remove `managementasset.StartAutoUpdater(context.Background(), configFilePath)` at line 566
  2. Remove `managementasset.StartAutoUpdater(context.Background(), configFilePath)` at line 644
  3. Remove `managementasset` import if no longer referenced
  4. Clean up any unused variables
- expected outputs:
  - No auto-updater goroutine launched at startup
  - Binary compiles clean
- verification:
  - `grep -n "StartAutoUpdater" cmd/server/main.go` returns empty
  - `go build ./cmd/server/` compiles
- stop if:
  - StartAutoUpdater is called from other entry points
- escalate to:
  - plan phase

## Wave 4
### T7 — Add frontend build to goreleaser hooks
- type: implementation
- inputs:
  - `.goreleaser.yml`
- touches:
  - `.goreleaser.yml`
- avoid:
  - changing Go build configuration
- steps:
  1. Add `before:` section with hooks that run before Go compilation:
     ```yaml
     before:
       hooks:
         - cmd: bun install
           dir: web
         - cmd: bun run build
           dir: web
         - cmd: mkdir -p internal/managementasset/static
         - cmd: cp web/dist/index.html internal/managementasset/static/management.html
     ```
  2. Verify the YAML structure is valid
- expected outputs:
  - `.goreleaser.yml` includes frontend build in before hooks
- verification:
  - `grep -A5 "before:" .goreleaser.yml` shows the hook commands
- stop if:
  - goreleaser YAML structure doesn't support `dir` field for hooks
- escalate to:
  - user clarification (check goreleaser docs)

### T8 — Create dev build helper
- type: implementation
- inputs:
  - Build steps from T1-T3
- touches:
  - `Makefile` (new file)
- avoid:
  - complex build orchestration
- steps:
  1. Create `Makefile` with targets:
     - `build-web`: `cd web && bun install && bun run build`
     - `embed`: `mkdir -p internal/managementasset/static && cp web/dist/index.html internal/managementasset/static/management.html`
     - `build`: `build-web` + `embed` + `go build ./cmd/server/`
     - `dev`: `build-web` + `embed` + `go run ./cmd/server/`
- expected outputs:
  - `make build` produces a working binary with embedded panel
- verification:
  - `make build` exits 0
  - `ls -la` shows the compiled binary
- stop if:
  - project already has a Makefile with conflicting targets
- escalate to:
  - user clarification

## Wave 5
### T9 — Full build and runtime verification
- type: test
- inputs:
  - All changes from Waves 1-4
- touches:
  - none (verification only)
- avoid:
  - making code changes during verification
- steps:
  1. Clean build: `rm -rf web/dist web/node_modules internal/managementasset/static`
  2. Run: `make build` (or manual steps: bun install, bun build, copy, go build)
  3. Run: `go build ./...` — full project compile
  4. Run: `go vet ./...`
  5. Start the server: `./llmhub -config config.example.yaml` (or equivalent)
  6. Verify: `curl -s http://localhost:8317/management.html | head -c 200` returns HTML content
  7. Verify: the panel loads in a browser and the management UI is interactive
  8. Verify disable-control-panel: set `disable-control-panel: true` in config, restart, confirm `/management.html` returns 404
  9. Verify: `grep -rn "EnsureLatestManagementHTML\|StartAutoUpdater\|downloadAsset" --include="*.go"` — zero hits outside test files
- expected outputs:
  - Binary compiles and runs
  - `/management.html` serves the embedded React panel
  - Management API routes respond correctly
  - `disable-control-panel: true` returns 404
- verification:
  - curl returns HTML with `<html` or `<!DOCTYPE` prefix
  - Browser shows the management UI
  - 404 with disable flag set
- stop if:
  - panel loads but API calls fail (CPAMC targets routes that don't exist)
  - panel doesn't load at all (embed path wrong, Content-Type wrong)
- escalate to:
  - check (run `/check` for code review)

## Risks / Watch-fors
- CPAMC may hardcode the API base URL or use relative paths — verify the panel's network requests target the same origin
- `vite-plugin-singlefile` output may include source maps or debug artifacts in dev mode — ensure `bun run build` uses production mode
- Goreleaser `before.hooks` must run in the repo root — verify `dir` field support in the goreleaser version used
- The `static/management.html` file must exist at `go build` time — CI pipelines need the frontend build step before Go compile
- If CPAMC upgrades React or Vite in the future, the build may need bun version changes
