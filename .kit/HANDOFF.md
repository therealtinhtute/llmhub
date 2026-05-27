---
session-date: 2026-05-27
branch: master
status: in-progress
continuity-mode: full-harness
active-phase: module-rebrand
last-updated: 2026-05-27 21:15
---

# Session Handoff — master

## Current State

**Branch**: `master` (in sync with `origin/master`, HEAD at `41420d3`)  
**Status**: planning-complete, implementation not started  
**Continuity Mode**: full-harness  
**Active Phase**: `module-rebrand`  
**Last Commit**: `41420d3` — ✨ feat(all): add new feature.

**Working Tree**:
- 0 staged files
- 0 modified files
- 3 untracked (new planning artifacts — not committed yet)

## What We're Building

Phase 1 of the LLMHub rebrand: turning the `CLIProxyAPI` fork into `LLMHub` as a monorepo. The fork currently has no in-repo frontend; the management UI (`/management.html`) is served from a runtime-downloaded file from GitHub. Phase 1 imports the CPAMC React panel into `web/`, builds it into a single HTML file, embeds it in the Go binary, and renames all product identity from upstream to LLMHub.

This session completed all planning. Zero implementation code has been written yet.

## Continuity Anchors

**Spec**: `.kit/planning/SPEC.md` — locked, status: draft/approved  
**Roadmap**: `.kit/planning/ROADMAP.md` — written this session  
**Latest Cook Run**: none  
**Latest Check Verdict**: none  
**Proof / Drift Notes**: planning artifacts are untracked (not committed) — commit before starting work or the next session will see a dirty tree

## Progress This Session

### Completed ✓
- Read and analyzed `SPEC.md` (LLMHub rebrand + embedded web UI)
- Explored repo: confirmed module path `github.com/router-for-me/CLIProxyAPI/v7`, 964 Go import lines, current `managementasset/updater.go` download logic, no existing `web/` dir
- Fetched upstream CPAMC stack: React 19 + Vite 8 + `vite-plugin-singlefile` + Bun → single HTML output
- `/think` session: produced decision-complete design plan (embed via `static/` copy, Bun build, goreleaser hooks)
- `/plan full`: wrote ROADMAP.md + 3 phases × (CONTEXT.md + PLAN.md) + workflow-state.yml

### In Progress ⏳
- Nothing — ready to start `module-rebrand` implementation

### Not Started
- `module-rebrand` — Go module path rename, product identity, config cleanup
- `embed-panel` — CPAMC import, Bun build, go:embed integration, server handler rewrite
- `doc-cleanup` — README rebrand, example config update, sponsor block removal

## Key Decisions

1. **Module path drops `/v7`**: `github.com/therealtinhtute/llmhub` — fork starts fresh versioning
2. **Single HTML embed**: CPAMC uses `vite-plugin-singlefile`, output is one `index.html` — perfect for `go:embed`
3. **Copy step for embed**: `go:embed` can't traverse `../`, so build copies `web/dist/index.html` → `internal/managementasset/static/management.html` before Go compile
4. **Remove `PanelGitHubRepository` entirely**: no deprecation — once panel is embedded the key is meaningless
5. **Gut `updater.go` fully**: all GitHub download logic removed, not kept as fallback
6. **Bun as package manager**: matches CPAMC natively, confirmed available in dev env

## Blockers & Issues

None currently. Planning is unblocked. Two preconditions for `embed-panel` phase:
- Bun installed (confirmed by user)
- CPAMC repo accessible at `https://github.com/router-for-me/Cli-Proxy-API-Management-Center`

## Technical Context

**Key call sites to change (embed-panel phase)**:
- `internal/api/server.go:731-758` — `serveManagementControlPanel` handler (reads from disk → reads from `[]byte`)
- `cmd/server/main.go:566,644` — `StartAutoUpdater` calls to remove
- `internal/api/server.go:747` — `EnsureLatestManagementHTML` call to remove

**Key constants to change (module-rebrand phase)**:
- `internal/config/config.go:25` — `DefaultAuthDir = "~/.cli-proxy-api"` → `"~/.llmhub"`
- `internal/config/config.go:23` — `DefaultPanelGitHubRepository` — remove
- `internal/config/config.go:206` — `PanelGitHubRepository` field — remove
- `cmd/server/main.go:58` — banner `CLIProxyAPI Version` → `LLMHub Version`
- `.goreleaser.yml:4,16` — binary `cli-proxy-api` → `llmhub`

**Phase guard**: after `module-rebrand`, `PanelGitHubRepository` is removed from config but `server.go:747` still calls `EnsureLatestManagementHTML(... cfg.RemoteManagement.PanelGitHubRepository)` — this call site will break and must be patched in the same phase (pass empty string as temporary bridge until `embed-panel` removes the download path entirely).

**CPAMC tech stack**:
- React 19.2.1 + TypeScript
- Vite 8.0.10 + `vite-plugin-singlefile` → single `dist/index.html`
- Bun 1.3.14
- Build: `bun install && bun run build`

## Next Steps

1. **→ START HERE: Commit planning artifacts** — `git add .kit/` and commit before starting implementation. Prevents dirty tree during work runs. Message: `docs(planning): add LLMHub Phase 1 roadmap and phase plans`
2. **Implement module-rebrand** — follow `.kit/planning/phases/module-rebrand/module-rebrand-PLAN.md` Wave 1→4. Entry: `go.mod` line 1, then bulk sed on 964 import lines. Verify with `go build ./...`.
3. **Run `/check` after module-rebrand** — gate before starting embed-panel. Update `workflow-state.yml` `current_phase` to `embed-panel`.
4. **Implement embed-panel** — follow `.kit/planning/phases/embed-panel/embed-panel-PLAN.md`. Entry: clone CPAMC → `bun run build` → copy to `static/` → write `embed.go` → rewrite handler.
5. **Implement doc-cleanup last** — follow `.kit/planning/phases/doc-cleanup/doc-cleanup-PLAN.md`. Lowest risk, no build dependency.

---

*Generated by /handoff on 2026-05-27 21:15*
