# ROADMAP: LLMHub Phase 1 — Rebrand + Embedded Web UI

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- entry phase: `module-rebrand`
- execution mode: sequential (each phase depends on prior)

## Phase 1: module-rebrand
**Goal:** Migrate the Go module path to `github.com/therealtinhtute/llmhub`, rename product identity strings to LLMHub, change auth dir default to `~/.llmhub`, remove the `panel-github-repository` config key, and rename build artifacts.

**Deliverables:**
- `go.mod` module line changed to `github.com/therealtinhtute/llmhub`
- All ~964 Go import paths updated
- `DefaultAuthDir` changed to `~/.llmhub`
- `DefaultPanelGitHubRepository` constant and `PanelGitHubRepository` config field removed
- Startup banner prints `LLMHub`
- `.goreleaser.yml` binary renamed to `llmhub`
- `docker-compose.yml` auth mount path updated
- `go build ./...` passes clean

**Dependencies:**
- None (first phase)

**Risks / Watch-fors:**
- Bulk import replacement can break if `v7` version suffix isn't handled correctly (new path drops `/v7`)
- Config removal may leave dead references in management API handlers or TUI
- Docker volume path change may surprise existing deployments (spec-accepted)

## Phase 2: embed-panel
**Goal:** Import the upstream CPAMC React panel source into `web/`, build it with Bun + Vite (produces single HTML via `vite-plugin-singlefile`), embed the build output into the Go binary via `go:embed`, and replace the runtime download serving path with embedded asset delivery.

**Deliverables:**
- `web/` directory with CPAMC source (copied, no git history)
- `bun install && bun run build` produces `web/dist/index.html`
- `internal/managementasset/static/management.html` copied from build output
- `internal/managementasset/embed.go` with `//go:embed` serving the panel
- `internal/managementasset/updater.go` GitHub download logic removed
- `internal/api/server.go` `serveManagementControlPanel` serves from embedded bytes
- `cmd/server/main.go` `StartAutoUpdater` calls removed
- `disable-control-panel` flag still works (returns 404 when true)
- `.goreleaser.yml` `before.hooks` builds frontend before Go compile
- `web/dist/` added to `.gitignore`

**Dependencies:**
- Phase 1 complete (module path must be settled before adding new packages)
- Bun available in dev environment
- CPAMC source cloneable from `https://github.com/router-for-me/Cli-Proxy-API-Management-Center`

**Risks / Watch-fors:**
- CPAMC API calls may not match the Go server's management API routes (requires runtime verification)
- `go:embed` cannot traverse `../` — requires a copy step from `web/dist/` to `internal/managementasset/static/`
- Build pipeline must include frontend build before Go compile in CI/goreleaser
- CPAMC's `bun.lockb` or `bun.lock` must be committed for reproducible builds

## Phase 3: doc-cleanup
**Goal:** Update README files, example config, and other docs to reflect LLMHub branding. Remove sponsor blocks, promotional content, and ecosystem listing noise while preserving operational documentation.

**Deliverables:**
- `README.md` rebranded to LLMHub, sponsor/promo sections removed
- `README_CN.md` and `README_JA.md` rebranded
- `config.example.yaml` updated: `panel-github-repository` removed, `auth-dir` default shows `~/.llmhub`
- `assets/` sponsor images evaluated for removal
- No `router-for-me` or `CLI Proxy API` branding in primary docs
- Operational docs (install, config reference, build instructions) preserved

**Dependencies:**
- Phase 1 complete (product identity settled)
- Phase 2 ideally complete (config example should reflect embedded panel, not download)

**Risks / Watch-fors:**
- Over-cleanup risk: some upstream doc content may still be operationally useful
- Non-English READMEs may have different structure than English version
