# SPEC — GitHub binary release pipeline + one-line VPS installer

- **Status:** Locked
- **Input Type:** change-request
- **Lane:** normal
- **Risk Flags:** external-systems, cross-platform, public-contract, existing-behavior
- **Affected Surfaces:** worker (CI), docs
- **Downstream:** plan
- **Updated At:** 2026-05-29

## Goal

Re-establish an automated GitHub release that builds and publishes **raw per-architecture binaries** on tag push (better-ccflare style), and provide a **single-command VPS installer** (`scripts/install.sh`) that downloads the binary, seeds config, and brings up a running systemd service in one shot. Update `README.md` and `Makefile` so all documented install paths match the new raw-binary asset naming.

## Context

- `.github/workflows/release.yaml` (goreleaser-on-tag) was deleted in commit `affb64b`; all other workflows were removed too. The user wants release automation back, but **minimal** — build + publish binaries only, no guard/docker/PR workflows.
- `.goreleaser.yml` still exists and currently produces **tar.gz archives** (`llmhub_{version}_{os}_{aarch64|arch}.tar.gz`) bundling `config.example.yaml`, `README.md`, `LICENSE`, plus `checksums.txt`, for linux/windows/darwin/freebsd × amd64/arm64.
- README already has an Installation section, but its snippet downloads tar.gz archives and does multi-step systemd setup manually.
- `Makefile` `download-latest` / `install-latest` targets also assume tar.gz + `aarch64` naming.
- Reference (better-ccflare v3.5.20): raw executables named `better-ccflare-{os}-{arch}` (`.exe` on Windows), no archive wrapper, no checksums file, npm publish alongside.

## Decisions (locked via interview)

1. **Release style:** Match better-ccflare — raw per-arch binaries, no archive wrapper.
2. **Installer scope:** Full deploy — binary + system user + config + systemd unit + service start.
3. **Installer location:** `scripts/install.sh` committed in repo, invoked via `curl … raw.githubusercontent.com/…/master/scripts/install.sh | sudo sh`.
4. **Platforms:** Keep current full matrix — linux, windows, darwin, freebsd × amd64, arm64.

## Requirements

### CI / Release

- **R1.** A new workflow `.github/workflows/release.yml` triggers on pushed tags matching `v*` and runs on `ubuntu-latest`.
- **R2.** The workflow checks out with full history/tags, sets up Go (matching `go.mod`), sets up Bun, and runs GoReleaser to publish a GitHub Release with assets attached. It uses the default `GITHUB_TOKEN` for publishing.
- **R3.** `.goreleaser.yml` is updated so the `archives` block emits **raw binaries** (`formats: [binary]`) instead of tar.gz, named `llmhub-{os}-{arch}` with arch values `amd64` / `arm64` and a `.exe` suffix on Windows. No file bundling (LICENSE/README/config) since raw binaries cannot bundle.
- **R4.** GoReleaser still runs the existing `before` hooks (bun install, bun run build, embed `management.html`) so the web panel is embedded in every published binary.
- **R5.** `checksums.txt` continues to be published (defensive, low cost); its presence must not be required by the installer.
- **R6.** The workflow scope is limited to build+release. It does NOT re-introduce docker, PR-guard, agents-md, or retarget workflows.

### Installer (`scripts/install.sh`)

- **R7.** Running `curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/scripts/install.sh | sudo sh` on a fresh Ubuntu VPS results in a running, enabled `llmhub` systemd service — no further manual steps.
- **R8.** The script detects architecture from `uname -m` (`x86_64`→`amd64`, `aarch64`/`arm64`→`arm64`) and exits with a clear error on anything unsupported.
- **R9.** It resolves the latest release tag via the `releases/latest` redirect (overridable with a `VERSION`/`LLMHUB_VERSION` env var) and downloads `llmhub-linux-{arch}` to `/usr/local/bin/llmhub` with mode `0755`.
- **R10.** It creates a system user `llmhub` (nologin) and directories `/etc/llmhub`, `/var/lib/llmhub/auths`, `/var/log/llmhub` with the ownership/permissions the README already documents — idempotently (safe to re-run).
- **R11.** If `/etc/llmhub/config.yaml` does not already exist, it fetches `config.example.yaml` from the repo raw URL (`master`) and installs it as the config, with `auth-dir` pointed at `/var/lib/llmhub/auths`. An existing config is never overwritten.
- **R12.** It writes `/etc/llmhub/llmhub.service` ... `/etc/systemd/system/llmhub.service`, runs `daemon-reload`, then `enable --now`, and prints service status + the management panel URL.
- **R13.** The script is POSIX `sh`-compatible, uses `set -eu`, and fails loudly (non-zero exit, readable message) on any download or privilege error.

### Docs / Makefile consistency

- **R14.** `README.md` Installation section is rewritten: the one-line installer command is the primary path; the existing manual tar.gz snippet is replaced with manual raw-binary steps consistent with R9. systemd content stays but is presented as "what the installer does / manual alternative."
- **R15.** `Makefile` `download-latest` and `install-latest` targets are updated to the raw-binary naming (`llmhub-{os}-{arch}`, `.exe` on Windows) so they don't reference now-nonexistent tar.gz assets.

## In Scope

- New `.github/workflows/release.yml`.
- `.goreleaser.yml` archive/naming change to raw binaries.
- New `scripts/install.sh`.
- `README.md` Installation rewrite.
- `Makefile` download/install target naming fix.

## Out of Scope

- npm publishing (better-ccflare has it; not requested for llmhub).
- Docker image / docker-compose deployment.
- macOS Gatekeeper/quarantine handling, Windows install script (installer targets Linux VPS only).
- Re-introducing the other deleted workflows (PR guards, agents-md, retarget, funding).
- TLS/reverse-proxy/hardening of the deployed service beyond what the README already documents.
- Code-signing or notarization of binaries.

## Validation Expectations

- `make release-check` passes against the updated `.goreleaser.yml`.
- `make release-snapshot` produces raw `llmhub-{os}-{arch}` binaries under `dist/` (verify naming + that the binary runs `llmhub -h`).
- `sh -n scripts/install.sh` parses clean; shellcheck (if available) reports no errors.
- Dry run / container test: execute installer against latest release in a throwaway Ubuntu container (or document the manual VPS verification) → `systemctl status llmhub` shows active, `curl localhost:8317` reachable.
- README one-liner copy-pastes and matches the published asset names exactly.

## Key Decisions (with rejected alternatives)

- **Raw binaries over tar.gz archives** (chosen): matches better-ccflare and the user's explicit pick. *Rejected:* keeping goreleaser tar.gz — less churn and bundles config, but doesn't match the requested style.
- **Installer fetches config separately from raw master** (chosen): raw binaries can't bundle `config.example.yaml`, so the full-deploy installer pulls it from the repo. *Rejected:* embedding a heredoc config in the script (drifts from `config.example.yaml`); *rejected:* keeping tar.gz just to bundle config (contradicts decision 1).
- **In-repo `scripts/install.sh` on master** (chosen): always current, no extra release upload step. *Rejected:* release-attached installer — version-pinned but adds a publish step and staleness risk.
- **Keep `checksums.txt`** (chosen): cheap integrity signal even though better-ccflare omits it; installer doesn't depend on it. *Rejected:* dropping it for exact parity — loses a free safety net.
- **Linux-only installer** (chosen): the deploy target is a Linux VPS. *Rejected:* cross-platform installer — unneeded scope (YAGNI).

## Deferred Ideas

- Optional checksum verification inside `install.sh`.
- `--uninstall` flag for the installer.
- npm distribution channel.
- Docker/compose deployment path.
- Pin Go toolchain version in CI via `go.mod` `toolchain` directive.
