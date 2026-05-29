# ROADMAP: GitHub binary release pipeline + one-line VPS installer

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- recommended entry phase: `release-pipeline`
- execution mode: sequential (each phase establishes a contract the next consumes)

## Phase 1: release-pipeline
**Goal:** Publish raw per-arch binaries on tag push, establishing the canonical asset-naming contract (`llmhub-{os}-{arch}`) that all downstream phases consume.

**Deliverables:**
- `.goreleaser.yml` archives block emits raw binaries (`formats: [binary]`), named `llmhub-{os}-{arch}` (`.exe` on Windows), full platform matrix preserved, web-embed `before` hooks intact, `checksums.txt` kept (R3, R4, R5).
- `.github/workflows/release.yml` — tag `v*` trigger, ubuntu-latest, checkout + Go + Bun setup, GoReleaser publish via `GITHUB_TOKEN`, build+release only (R1, R2, R6).

**Dependencies:**
- existing `.goreleaser.yml`, `go.mod`, `web/` build hooks.

**Risks / Watch-fors:**
- GoReleaser v2 raw-binary syntax (`formats: [binary]` in `archives`, not deprecated `format:`).
- arm64 naming must be literal `arm64` (not `aarch64` as the old tar.gz template used) — this is the contract phases 2 & 3 depend on.
- Bun setup must precede GoReleaser so embed hooks succeed in CI.

## Phase 2: vps-installer
**Goal:** A single `curl … | sudo sh` command on a fresh Ubuntu VPS yields a running, enabled `llmhub` systemd service.

**Deliverables:**
- `scripts/install.sh` — POSIX, `set -eu`, idempotent: arch detect → resolve latest tag → download `llmhub-linux-{arch}` → create user/dirs → seed config (no overwrite) → write+enable systemd unit → start + status (R7–R13).

**Dependencies:**
- Phase 1 asset naming (`llmhub-linux-{arch}`).
- `config.example.yaml` fetched from repo raw `master`.
- systemd unit content from existing README section.

**Risks / Watch-fors:**
- Idempotency: re-running must not fail on existing user/dirs/config.
- Must never overwrite an existing `/etc/llmhub/config.yaml` (R11).
- `releases/latest` redirect resolution must work without `jq` (POSIX-only).

## Phase 3: docs-consistency
**Goal:** Every documented install path (README, Makefile) matches the new raw-binary asset names; no references to dead tar.gz assets remain.

**Deliverables:**
- `README.md` Installation rewrite: one-liner as primary path; manual raw-binary steps replace the tar.gz snippet; systemd content reframed as "what the installer does / manual alternative" (R14).
- `Makefile` `download-latest` / `install-latest` updated to `llmhub-{os}-{arch}` naming (R15).

**Dependencies:**
- Phase 1 asset naming.
- Phase 2 installer path/command (`scripts/install.sh` one-liner).

**Risks / Watch-fors:**
- README one-liner must match published asset names exactly (cross-check against `.goreleaser.yml`).
- Makefile targets reference tar.gz extraction logic that must be simplified for raw binaries (no `tar -xzf`).
