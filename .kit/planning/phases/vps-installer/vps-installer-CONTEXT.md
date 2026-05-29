# Context: vps-installer

Phase: vps-installer
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: integration (container/VPS dry run), inspection (sh -n, shellcheck)

## Goal
Provide `scripts/install.sh` so one `curl -fsSL …/master/scripts/install.sh | sudo sh` brings up a running, enabled `llmhub` service on a fresh Ubuntu VPS.

## Scope Boundary
### Allowed Surfaces
- `scripts/install.sh` (new file)

### Forbidden Surfaces
- `.goreleaser.yml`, `.github/workflows/` (Phase 1)
- `README.md`, `Makefile` (Phase 3)
- application source code

## Spec Hooks
- R7–R13 (installer).
- Out of scope: macOS/Windows install, TLS/hardening, uninstall flag.

## Locked Decisions
- POSIX `sh`, `set -eu`, idempotent (safe to re-run).
- Linux-only; arch from `uname -m`: `x86_64`→`amd64`, `aarch64`/`arm64`→`arm64`; anything else = clear error + non-zero exit.
- Resolve latest tag via `releases/latest` redirect (`curl -fsSLI -o /dev/null -w '%{url_effective}'`), overridable by `LLMHUB_VERSION`/`VERSION` env.
- Download `llmhub-linux-{arch}` → `/usr/local/bin/llmhub`, mode 0755 (matches Phase 1 naming exactly).
- Create system user `llmhub` (nologin) + dirs `/etc/llmhub`, `/var/lib/llmhub/auths`, `/var/log/llmhub` with README-documented ownership/perms.
- Seed `/etc/llmhub/config.yaml` from `config.example.yaml` fetched off repo raw `master` ONLY if absent; set `auth-dir: /var/lib/llmhub/auths`. Never overwrite existing config.
- Write `/etc/systemd/system/llmhub.service`, `daemon-reload`, `enable --now`, print status + management panel URL.

## Assumptions
- Target VPS has `curl`, `systemd`, `install`, `useradd` (standard Ubuntu).
- `config.example.yaml` remains present at repo root on `master`.
- Script runs as root (via `sudo sh`); it may check effective UID and error if not root.

## Canonical Refs
- `.kit/planning/SPEC.md`
- `README.md` existing systemd unit + user/dir setup (lines ~66–114) — reuse exact commands
- `config.example.yaml`
- Phase 1 naming contract (`llmhub-linux-{arch}`)

## Rejected Options
- Embed config as a heredoc in the script — drifts from `config.example.yaml`; rejected.
- Use GitHub API + `jq` to resolve latest — adds a dependency; redirect trick is POSIX-only.
- Bundle config in a tar.gz — contradicts raw-binary decision.

## Deferred Ideas
- `--uninstall` flag.
- Optional checksum verification against `checksums.txt`.
- `--skip-systemd` / binary-only mode flag.

## Escalate If
- README systemd/user setup commands are ambiguous or insufficient for idempotency (→ user clarification).
- `config.example.yaml` lacks an `auth-dir` key to set, requiring config-schema decisions (→ brainstorm refine).
