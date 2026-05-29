# Plan: vps-installer

Phase: vps-installer
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-05-29

## Goal
`curl -fsSL …/master/scripts/install.sh | sudo sh` → running, enabled `llmhub` service on fresh Ubuntu.

## Inputs
- Phase 1 asset naming (`llmhub-linux-{arch}`)
- `README.md` existing systemd unit + user/dir commands (lines ~66–114)
- `config.example.yaml` at repo root

## Wave 1
### T1 — Write `scripts/install.sh` core (download + install binary)
- type: implementation
- inputs:
  - asset name contract `llmhub-linux-{amd64|arm64}`
- touches:
  - `scripts/install.sh` (new)
- avoid:
  - `.goreleaser.yml`, README, Makefile
- steps:
  1. Shebang `#!/bin/sh`, `set -eu`.
  2. Require root: if `id -u` != 0, error "run with sudo" and exit 1.
  3. Detect arch from `uname -m`: `x86_64`→`amd64`, `aarch64`|`arm64`→`arm64`, else error + exit 1.
  4. Resolve tag: use `${LLMHUB_VERSION:-${VERSION:-}}`; if empty, derive from `releases/latest` redirect via `curl -fsSLI -o /dev/null -w '%{url_effective}'`, strip to tag.
  5. Download `https://github.com/therealtinhtute/llmhub/releases/download/<tag>/llmhub-linux-<arch>` to a temp file; `install -m 0755` to `/usr/local/bin/llmhub`.
  6. Print installed version (`llmhub -h` or version line).
- expected outputs:
  - `scripts/install.sh` that installs the binary to `/usr/local/bin/llmhub`
- verification:
  - `sh -n scripts/install.sh` parses clean; `shellcheck scripts/install.sh` (if available) clean
  - dry run in Ubuntu container with a known tag → `/usr/local/bin/llmhub -h` works
- stop if:
  - asset name uncertainty (must match Phase 1 exactly)
- escalate to:
  - plan phase (re-sync naming) | user clarification

## Wave 2
### T2 — Add full-deploy steps (user, dirs, config, systemd)
- type: implementation
- inputs:
  - core script from T1
  - README user/dir/systemd commands
  - `config.example.yaml`
- touches:
  - `scripts/install.sh` (same file)
- avoid:
  - overwriting an existing `/etc/llmhub/config.yaml`
- steps:
  1. Create user idempotently: `id llmhub >/dev/null 2>&1 || useradd --system --home /var/lib/llmhub --shell /usr/sbin/nologin llmhub`.
  2. `mkdir -p /etc/llmhub /var/lib/llmhub/auths /var/log/llmhub`; chown/chmod per README (750 on lib dirs, llmhub:llmhub ownership).
  3. If `/etc/llmhub/config.yaml` absent: fetch `https://raw.githubusercontent.com/therealtinhtute/llmhub/master/config.example.yaml` → install to `/etc/llmhub/config.yaml` (mode 0640, group llmhub); ensure `auth-dir: "/var/lib/llmhub/auths"` is set (append if missing). Else: leave untouched, print "keeping existing config".
  4. Write `/etc/systemd/system/llmhub.service` heredoc (mirror README unit: User/Group llmhub, ExecStart `/usr/local/bin/llmhub -config /etc/llmhub/config.yaml`, Restart on-failure).
  5. `systemctl daemon-reload`, `systemctl enable --now llmhub`.
  6. Print `systemctl status llmhub --no-pager` and the management panel URL (`http://SERVER_IP:8317/management.html`).
- expected outputs:
  - re-runnable installer that leaves `llmhub.service` active and enabled
- verification:
  - container/VPS dry run: `systemctl is-enabled llmhub` = enabled, `systemctl is-active llmhub` = active (or documented manual VPS check if systemd unavailable in CI container)
  - re-run the script → no errors, existing config preserved
- stop if:
  - README setup commands are insufficient for idempotency, or config lacks `auth-dir`
- escalate to:
  - user clarification | brainstorm refine

## Risks / Watch-fors
- Idempotency is the main risk — guard every create with existence checks.
- Never clobber `/etc/llmhub/config.yaml` (R11).
- Keep everything POSIX `sh` (no bash-isms, no `jq`).
