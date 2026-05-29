# Context: docs-consistency

Phase: docs-consistency
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low
Expected Proof: inspection (asset-name cross-check), make target dry run

## Goal
Make every documented install path match the raw-binary asset names; remove all references to the now-dead tar.gz assets.

## Scope Boundary
### Allowed Surfaces
- `README.md` (Installation section)
- `Makefile` (`download-latest`, `install-latest` targets only)

### Forbidden Surfaces
- `.goreleaser.yml`, `.github/workflows/` (Phase 1)
- `scripts/install.sh` (Phase 2 — reference it, don't edit it)
- other Makefile targets (`build`, `embed`, `release*`, etc.)
- README sections unrelated to Installation

## Spec Hooks
- R14 (README), R15 (Makefile).

## Locked Decisions
- README one-liner `curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/scripts/install.sh | sudo sh` is the primary install path.
- Replace the tar.gz download snippet with manual raw-binary steps (download `llmhub-linux-{arch}`, `chmod +x`, move to `/usr/local/bin`).
- systemd/user/dir content stays but reframed as "the installer does this for you / manual alternative."
- `Makefile` `download-latest`/`install-latest` use `llmhub-{os}-{arch}` naming, drop `tar -xzf` extraction and the `aarch64` mapping.

## Assumptions
- Asset names from Phase 1 are final and authoritative; this phase mirrors them, never redefines them.
- `make install-latest` should still install a runnable binary (raw file → chmod → move), no archive extraction.

## Canonical Refs
- `.kit/planning/SPEC.md`
- `README.md` current Installation section (lines ~31–125)
- `Makefile` current `download-latest`/`install-latest` targets
- `.goreleaser.yml` final naming (after Phase 1) — source of truth for asset names

## Rejected Options
- Leave Makefile targets as-is — they'd point at nonexistent tar.gz assets and silently break.
- Keep both tar.gz and raw instructions — confusing, contradicts the single asset format.

## Deferred Ideas
- Document npm/docker install once those channels exist.
- Add a `make uninstall` target.

## Escalate If
- Phase 1 asset naming differs from what this phase assumes (→ re-sync, do not guess).
