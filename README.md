# LLMHub


A proxy server that provides OpenAI/Gemini/Claude/Codex/Grok compatible API interfaces for CLI.

It now also supports OpenAI Codex (GPT models) and Claude Code via OAuth.

So you can use local or multi-account CLI access with OpenAI(include Responses)/Gemini/Claude-compatible clients and SDKs.

## Overview

- OpenAI/Gemini/Claude/Grok compatible API endpoints for CLI models
- OpenAI Codex support (GPT models) via OAuth login
- Claude Code support via OAuth login
- Grok Build support via OAuth login
- Streaming, non-streaming, and WebSocket responses where supported
- Function calling/tools support
- Multimodal input support (text and images)
- Multiple accounts with round-robin load balancing (Gemini, OpenAI, Claude, Grok)
- Simple CLI authentication flows (Gemini, OpenAI, Claude, Grok)
- Generative Language API Key support
- AI Studio Build multi-account load balancing
- Gemini CLI multi-account load balancing
- Claude Code multi-account load balancing
- OpenAI Codex multi-account load balancing
- Grok Build multi-account load balancing
- OpenAI-compatible upstream providers via config
- Reusable Go SDK for embedding the proxy

## Installation

### One-line VPS install (recommended)

For production, install the latest GitHub Release binary and set up a systemd service on a Linux VPS with one command:

```bash
curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/install.sh | sudo bash
```

This installs LLMHub cleanly into `/opt/llmhub` (all-in-one directory containing binary, `.env`, `data/`, and auto-generated quota encryption keys), configures a systemd service, and starts it automatically. To customize the directory (e.g. `$HOME/llmhub`), specify `LLMHUB_DIR`:

```bash
curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/install.sh | LLMHUB_DIR=/opt/llmhub sudo -E bash
```

Point your AI coding tool at `http://SERVER_IP:9090/v1`. The management panel is at `http://SERVER_IP:9090/management.html` (configure `remote-management.secret-key` and `remote-management.allow-remote` before exposing beyond localhost).

### Local binary VPS install

When you already have a Linux `llmhub` binary, copy it beside the local
installer and run it from that directory:

```bash
sudo ./install-local.sh
```

The local installer defaults to port `9090`. It prompts for an optional domain
name for Caddy HTTPS; leave the prompt blank to keep the direct IP/port setup.
Override the port or preseed domain mode when needed:

```bash
sudo DEFAULT_PORT=8080 ./install-local.sh
sudo CADDY_DOMAIN=llm.example.com ./install-local.sh
sudo ./install-local.sh /path/to/llmhub-linux-amd64
```

The generated `systemd` unit is VPS-oriented by default: it uses
`Restart=always`, waits `3s` between restart attempts, and disables the default
start-rate lockout with `StartLimitIntervalSec=0` so `llmhub` keeps coming back
after repeated crashes until you fix the underlying issue. Override these at
install time with `SERVICE_RESTART=...`, `SERVICE_RESTART_SEC=...`, or
`SERVICE_START_LIMIT_INTERVAL=...` if you want a different policy.

If you provide a domain, the installer configures Caddy as a reverse proxy to
the local `llmhub` port and prints HTTPS endpoints. Point the domain's DNS A
record at the VPS first or immediately after install; TLS becomes live once DNS
propagates.

For Postgres-backed runtime storage, the installer can prompt for
`PGSTORE_DSN`, `PGSTORE_SCHEMA`, and
`PGSTORE_USAGE_RETENTION_SECONDS` during the run, or you can put them in the
same `.env` file ahead of time. The installer keeps runtime metadata under
`/var/lib/llmhub/pgstore` by setting `WRITABLE_PATH=/var/lib/llmhub`.
The installer never prompts for YAML: it auto-derives the first-boot config
seed from `LLMHUB_HOST`/`LLMHUB_PORT`. Set `LLMHUB_INIT_CONFIG_B64` or
`LLMHUB_INIT_CONFIG_YAML` in the `.env` file ahead of time only if you need a
richer initial config than host/port.

Example `.env`:

```bash
PGSTORE_DSN='postgres://postgres.xxx:password@aws-0-region.pooler.supabase.com:6543/postgres?sslmode=require'
PGSTORE_SCHEMA='llmhub'
PGSTORE_USAGE_RETENTION_SECONDS='60'
```

Recommended same-directory layout:

```text
llmhub-install/
  install-local.sh
  llmhub
  .env
```

### Manual binary install

If you prefer manual installation:

```bash
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

TAG="$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/therealtinhtute/llmhub/releases/latest)"
TAG="${TAG##*/}"

curl -fL \
  "https://github.com/therealtinhtute/llmhub/releases/download/${TAG}/llmhub-linux-${ARCH}" \
  -o llmhub

chmod +x llmhub
sudo install -m 0755 llmhub /usr/local/bin/llmhub
llmhub -h
```

You can also use:

```bash
make install-latest
```

Use `sudo make install-latest` when your user cannot write to `/usr/local/bin`.

### Manual service setup

The one-line installer does this automatically. If installing manually, create the system user and directories:

```bash
sudo useradd --system --home /var/lib/llmhub --shell /usr/sbin/nologin llmhub
sudo mkdir -p /etc/llmhub /var/lib/llmhub /var/log/llmhub
sudo chown -R llmhub:llmhub /var/lib/llmhub /var/log/llmhub
sudo chmod 750 /var/lib/llmhub
```

Create the bootstrap env file:

```bash
sudo tee /etc/llmhub/llmhub.env >/dev/null <<'EOF'
LLMHUB_HOST=0.0.0.0
LLMHUB_PORT=8317
PGSTORE_DSN='postgres://postgres.xxx:password@aws-0-region.pooler.supabase.com:6543/postgres?sslmode=require'
PGSTORE_SCHEMA='llmhub'
PGSTORE_USAGE_RETENTION_SECONDS='60'
LLMHUB_INIT_CONFIG_B64='<base64 of initial YAML config>'
EOF
```

Seed the database once:

```bash
sudo -u llmhub env $(grep -v '^#' /etc/llmhub/llmhub.env | xargs) \
  /usr/local/bin/llmhub init-db-from-env -env-file /etc/llmhub/llmhub.env
```

Install the systemd unit:

```bash
sudo tee /etc/systemd/system/llmhub.service >/dev/null <<'EOF'
[Unit]
Description=LLMHub proxy server
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
User=llmhub
Group=llmhub
WorkingDirectory=/var/lib/llmhub
Environment=HOME=/var/lib/llmhub
EnvironmentFile=-/etc/llmhub/llmhub.env
ExecStartPre=/usr/local/bin/llmhub init-db-from-env -env-file /etc/llmhub/llmhub.env
ExecStart=/usr/local/bin/llmhub
Restart=always
RestartSec=3s

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now llmhub
sudo systemctl status llmhub
```

Logs: `journalctl -u llmhub -f`

### Build from source

For local development, build from source with Go 1.21+ and Bun:

```bash
make build
./llmhub init-db-from-env -env-file .env
./llmhub
```

Use `make dev-pg` when you want the dev server to load `.env`, seed Postgres,
and run against the DB-backed runtime.

The initial runtime config is supplied through `LLMHUB_INIT_CONFIG_YAML` or
`LLMHUB_INIT_CONFIG_B64`, then stored in Postgres. Runtime reads and writes
config only from the database after that.

## Updating LLMHub

On the systemd install (`scripts/install-local.sh`), `apply-staged-update` runs
as root before every service start. The operator flow:

1. **Check** — `llmhub update --check` compares the running version with the
   latest stable release and reports `update available` / `up to date` / newer
   than stable. No network data is written anywhere.
2. **Stage** — `llmhub update` downloads the release for your platform,
   verifies it against the published checksums, probes it, and writes it to
   `${DATA_DIR}/update/`. The installed binary is never touched by staging.
3. **Apply** — `sudo systemctl restart llmhub` (or reboot). Before starting,
   the root apply step re-fetches `checksums.txt` over HTTPS for the staged
   version, re-hashes `llmhub.staged`, and only then swaps it over the
   installed binary, retaining the previous one as `<binary>.previous`.
4. **Automatic revert** — if the new binary never reaches a healthy start, the
   next start restores `<binary>.previous` automatically and leaves the failed
   candidate staged for inspection.
5. **Manual rollback** — `sudo llmhub update rollback` restores
   `<binary>.previous` immediately and clears the pending apply.

Every `llmhub update` log line (check, stage, rollback) is prefixed with an
RFC 3339 timestamp showing the current version, the target version, and each
verification result, so update actions are traceable in shell history and
journal output.

On the systemd install, the management panel can drive the same flow from the
**System** page's version card: **Check for updates** reports the latest
release, **Update** stages it (same verify-and-probe path as `llmhub update`),
and **Restart to apply** restarts the service, which applies the staged binary
on the way back up. The panel connection drops briefly during the restart and
returns once the new version is running. This requires the management key, a
systemd install, and sudo (the installer writes a sudoers drop-in allowing the
service user to restart only its own unit, and validates it with `visudo`
before activating it).

Supported platforms are Linux and macOS on amd64 and arm64; Windows and
FreeBSD builds exist but refuse self-update explicitly.

Updates replace the binary only: configuration, tokens, and database state are
never written by the updater, so an update cannot lose runtime state and
rollback never restores database data. Recovery always needs a binary that can
start — if both the current and previous binaries are broken, reinstall with
`scripts/install-local.sh`.

## Postgres Durable Runtime

LLMHub runtime is now Postgres-only. Provide `PGSTORE_DSN` plus one of
`LLMHUB_INIT_CONFIG_YAML` or `LLMHUB_INIT_CONFIG_B64` for the initial seed, and
Postgres becomes the source of truth for cliproxy config, OAuth/auth records,
and the recent management usage queue.

Example Supabase-style configuration:

```bash
export PGSTORE_DSN='postgres://postgres.xxx:password@aws-0-region.pooler.supabase.com:6543/postgres?sslmode=require'
export PGSTORE_SCHEMA='llmhub'
export PGSTORE_USAGE_RETENTION_SECONDS='60'
export LLMHUB_INIT_CONFIG_B64="$(printf '%s\n' 'host: 0.0.0.0' 'port: 8317' | base64 | tr -d '\n')"
llmhub init-db-from-env
llmhub
```

Boot behavior:

- Startup requires a working Postgres connection and does not fall back to any
  local durable runtime store.
- `init-db-from-env` creates schema/tables if needed and seeds config only when
  the config row is still empty.
- Runtime fails fast when the DB has no config yet; run `llmhub init-db-from-env`
  or `llmhub migrate-local-to-db` first.
- Management edits and OAuth/auth writes persist directly to Postgres.
- Runtime does not keep durable local auth files, config files, or request
  archive files. Operational logs stay on stdout/stderr.

Operational notes:

- Protect `PGSTORE_DSN` as a secret.
- Use SSL for remote Postgres, for example `sslmode=require`.
- Use a Supabase transaction/session pooler URL suitable for a long-running app.
- Keep the usage retention short; it is a recent management queue, not a
  long-term analytics store.
- If you previously relied on `logging-to-file`, request log files, or forced
  error request archives, those local files are intentionally disabled when
  `PGSTORE_DSN` is active.

## Management Panel

The management panel is served at `http://localhost:PORT/management.html` once the service is running. It provides visual config editing, OAuth flows, auth file management, quota tracking, and log tailing.

## Provider-specific protocol paths

When you need the request/response shape of a specific backend family, use the provider-specific paths instead of the merged `/v1/...` endpoints:

- Use `/api/provider/{provider}/v1/messages` for messages-style backends.
- Use `/api/provider/{provider}/v1beta/models/...` for model-scoped generate endpoints.
- Use `/api/provider/{provider}/v1/chat/completions` for chat-completions backends.

These routes help you select the protocol surface, but they do not by themselves guarantee a unique inference executor when the same client-visible model name is reused across multiple backends. Inference routing is still resolved from the request model/alias. For strict backend pinning, use unique aliases, prefixes, or otherwise avoid overlapping client-visible model names.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
