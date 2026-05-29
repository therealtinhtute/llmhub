# Context: release-pipeline

Phase: release-pipeline
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: integration (goreleaser snapshot/check), inspection

## Goal
Publish raw per-arch binaries on tag push and lock the canonical asset-naming contract `llmhub-{os}-{arch}` (`.exe` on Windows, arch = `amd64`/`arm64`).

## Scope Boundary
### Allowed Surfaces
- `.goreleaser.yml` (archives + naming only)
- `.github/workflows/release.yml` (new file)

### Forbidden Surfaces
- `scripts/install.sh` (Phase 2)
- `README.md`, `Makefile` (Phase 3)
- `.goreleaser.yml` `builds`/`before` blocks — leave platform matrix and embed hooks unchanged
- any other deleted workflow (no docker/PR-guard/agents-md/retarget revival)

## Spec Hooks
- R1–R6 (CI/release).
- Constraint: build+release binaries ONLY; minimal workflow.

## Locked Decisions
- Raw binaries via `archives: [{ formats: [binary] }]` in GoReleaser v2 (NOT `format: binary` — deprecated).
- Name template yields `llmhub-{{.Os}}-{{.Arch}}` with `.exe` appended on Windows; arch stays literal `amd64`/`arm64` (drop the old `aarch64` mapping).
- Keep `checksums.txt`; installer must not depend on it.
- Keep existing `before` hooks (bun install/build + embed management.html) so the panel ships in every binary.
- Workflow triggers on tags matching `v*`, runs on ubuntu-latest, uses default `GITHUB_TOKEN`.

## Assumptions
- GoReleaser v2 (`version: 2`, pinned `GORELEASER_VERSION ?= v2.16.0` in Makefile) supports `formats: [binary]` under `archives`.
- `go.mod` Go version is the toolchain CI should set up.
- Bun is required at CI time for the web embed hook.

## Canonical Refs
- `.kit/planning/SPEC.md`
- `.goreleaser.yml` (current tar.gz config)
- `Makefile` (`release-check`, `release-snapshot` targets)
- GoReleaser v2 archives docs

## Rejected Options
- Keep tar.gz archives — contradicts locked decision 1 (raw binaries).
- Drop checksums for exact better-ccflare parity — loses a free integrity signal at no benefit.
- Build matrix via a hand-written matrix in Actions instead of GoReleaser — reinvents what `.goreleaser.yml` already does.

## Deferred Ideas
- npm publish channel.
- Docker image workflow.
- Pin Go toolchain via `go.mod` `toolchain` directive.

## Escalate If
- GoReleaser v2.16.0 does not support `formats: [binary]` syntax (→ plan phase / brainstorm refine for naming approach).
- `before` hooks fail in CI in a way that forces archive-format changes.
