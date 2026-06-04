# LLMHub


A proxy server that provides OpenAI/Gemini/Claude/Codex/Grok compatible API interfaces for CLI.

It now also supports OpenAI Codex (GPT models) and Claude Code via OAuth.

So you can use local or multi-account CLI access with OpenAI(include Responses)/Gemini/Claude-compatible clients and SDKs.

## Overview

- OpenAI/Gemini/Claude/Grok compatible API endpoints for CLI models
- OpenAI Codex support (GPT models) via OAuth login
- Claude Code support via OAuth login
- Grok Build support via OAuth login
- Amp CLI and IDE extensions support with provider routing
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
- OpenAI-compatible upstream providers via config (e.g., OpenRouter)
- Reusable Go SDK for embedding the proxy

## Installation

### One-line VPS install (recommended)

For production, install the latest GitHub Release binary and set up a systemd service on a Linux VPS with one command:

```bash
curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/scripts/install.sh | sudo sh
```

This downloads the binary, creates the `llmhub` system user, seeds `/etc/llmhub/config.yaml`, and starts the service. Edit the config to add your provider accounts, then restart: `sudo systemctl restart llmhub`.

Point your AI coding tool at `http://SERVER_IP:8317/v1`. The management panel is at `http://SERVER_IP:8317/management.html` (configure `remote-management.secret-key` and `remote-management.allow-remote` before exposing beyond localhost).

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

If you provide a domain, the installer configures Caddy as a reverse proxy to
the local `llmhub` port and prints HTTPS endpoints. Point the domain's DNS A
record at the VPS first or immediately after install; TLS becomes live once DNS
propagates.

For Postgres-backed runtime storage, put `PGSTORE_DSN` in the same `.env` file.
The installer keeps runtime metadata under `/var/lib/llmhub/pgstore` by setting
`WRITABLE_PATH=/var/lib/llmhub`.

Recommended same-directory layout:

```text
llmhub-install/
  install-local.sh
  llmhub
  config.example.yaml
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
sudo mkdir -p /etc/llmhub /var/lib/llmhub/auths /var/log/llmhub
sudo chown -R llmhub:llmhub /var/lib/llmhub /var/log/llmhub
sudo chmod 750 /var/lib/llmhub /var/lib/llmhub/auths
```

Download and install the config:

```bash
curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/config.example.yaml \
  -o /tmp/config.example.yaml
sudo install -m 0640 -o root -g llmhub /tmp/config.example.yaml /etc/llmhub/config.yaml
```

Edit `/etc/llmhub/config.yaml` to set `auth-dir: "/var/lib/llmhub/auths"` and configure providers.

Install the systemd unit:

```bash
sudo tee /etc/systemd/system/llmhub.service >/dev/null <<'EOF'
[Unit]
Description=LLMHub proxy server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=llmhub
Group=llmhub
WorkingDirectory=/etc/llmhub
ExecStart=/usr/local/bin/llmhub -config /etc/llmhub/config.yaml
Restart=on-failure
RestartSec=5s

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
./llmhub -config config.yaml
```

Use `make dev` for local file-backed development. It intentionally skips `.env`
storage settings such as `PGSTORE_DSN`. Use `make dev-pg` when you want the dev
server to load `.env` and run against Postgres.

See `config.example.yaml` for the full configuration reference.

## Postgres Durable Runtime

By default, llmhub keeps using the local `config.yaml` and auth JSON files under
`auth-dir`. Set `PGSTORE_DSN` to make Postgres the runtime source of truth for
cliproxy config, OAuth/auth records, and the recent management usage queue.
This contract applies only to normal server runtime. `HOME_JWT` mode stays on
its separate home-controlled path.

Example Supabase-style configuration:

```bash
export PGSTORE_DSN='postgres://postgres.xxx:password@aws-0-region.pooler.supabase.com:6543/postgres?sslmode=require'
export PGSTORE_SCHEMA='llmhub'
export PGSTORE_USAGE_RETENTION_SECONDS='60'
llmhub -config /etc/llmhub/config.yaml
```

Boot behavior:

- Without `PGSTORE_DSN`, local file mode is unchanged.
- With `PGSTORE_DSN`, startup requires a working DB connection and does not
  silently fall back to local durable runtime stores.
- If the DB has config, llmhub loads config from Postgres.
- If the DB has no config, llmhub imports the local `-config` path, or
  `config.yaml`, and saves it to Postgres.
- On that same first boot, if the auth table is empty, existing local auth JSON
  files from the configured `auth-dir` are imported once.
- After first import, DB rows win. Local files are bootstrap-only and are not
  watched as the Postgres runtime source.
- In Postgres mode, llmhub does not keep durable local server logs or request
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

## Amp CLI Support

LLMHub includes integrated support for [Amp CLI](https://ampcode.com) and Amp IDE extensions, enabling you to use your Google/ChatGPT/Claude OAuth subscriptions with Amp's coding tools:

- Provider route aliases for Amp's API patterns (`/api/provider/{provider}/v1...`)
- Management proxy for OAuth authentication and account features
- Smart model fallback with automatic routing
- **Model mapping** to route unavailable models to alternatives (e.g., `claude-opus-4.5` → `claude-sonnet-4`)
- Security-first design with localhost-only management endpoints

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
