# Plan: docs-consistency

Phase: docs-consistency
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-05-29

## Goal
README and Makefile reflect raw-binary asset names; no dead tar.gz references remain.

## Inputs
- Final asset names from Phase 1 (`.goreleaser.yml`)
- `scripts/install.sh` path/one-liner from Phase 2
- `README.md`, `Makefile`

## Wave 1
### T1 — Rewrite README Installation section
- type: docs
- inputs:
  - one-liner command, asset names
- touches:
  - `README.md` Installation section (lines ~31–125)
- avoid:
  - other README sections; `.goreleaser.yml`; `scripts/install.sh`
- steps:
  1. Make the one-liner the primary path: `curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/scripts/install.sh | sudo sh`, describing it as "downloads binary + sets up systemd service".
  2. Replace the tar.gz download block with manual raw-binary steps: detect arch, `curl -fL .../llmhub-linux-${ARCH}` → `chmod +x` → `sudo install -m 0755 ... /usr/local/bin/llmhub`. Use `amd64`/`arm64` (not `aarch64`).
  3. Reframe the systemd/user/dir block as "Manual service setup (the installer does this automatically)".
  4. Keep "Build from source", management-panel, and config-reference notes intact.
- expected outputs:
  - README Installation section with working one-liner + raw-binary manual steps
- verification:
  - grep README for `tar.gz` and `aarch64` → zero matches in Installation
  - asset names in README exactly match `.goreleaser.yml` `name_template` output
- stop if:
  - asset names differ from Phase 1 (re-sync, don't guess)
- escalate to:
  - plan phase

## Wave 2
### T2 — Fix Makefile download/install targets
- type: refactor
- inputs:
  - asset naming contract
- touches:
  - `Makefile` `download-latest`, `install-latest` targets
- avoid:
  - `build`, `embed`, `dev`, `release*`, `clean` targets
- steps:
  1. In `download-latest`: build asset name `llmhub-$${os}-$${asset_arch}` with `.exe` for windows; drop `aarch64` mapping (use `arm64`) and the `ext`/tar.gz logic; download the raw binary into `dist/downloads/`.
  2. In `install-latest`: download raw binary (reuse download-latest), `chmod +x`, `install -m 0755` to `$(PREFIX)`; remove `tar -xzf` extraction.
  3. Update `help` text only if the target descriptions become inaccurate.
- expected outputs:
  - `make download-latest` / `make install-latest` referencing raw `llmhub-{os}-{arch}` assets
- verification:
  - `make -n download-latest` / `make -n install-latest` show raw-binary URLs, no `tar` extraction
  - against a real published release: `make install-latest` then `llmhub -h` works (or document if no release exists yet)
- stop if:
  - targets depend on archive-only behavior not reproducible with raw binaries
- escalate to:
  - plan phase | user clarification

## Risks / Watch-fors
- README one-liner correctness is the highest-visibility risk — verify char-for-char against the published asset names.
- Makefile uses `$$` escaping inside recipes; preserve it.
