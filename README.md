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

### VPS binary install

For production, install the latest GitHub Release binary on a Linux VPS:

```bash
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="aarch64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

TAG="$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/therealtinhtute/llmhub/releases/latest)"
TAG="${TAG##*/}"
VERSION="${TAG#v}"

curl -fL \
  "https://github.com/therealtinhtute/llmhub/releases/download/${TAG}/llmhub_${VERSION}_linux_${ARCH}.tar.gz" \
  -o /tmp/llmhub.tar.gz

tar -xzf /tmp/llmhub.tar.gz -C /tmp llmhub config.example.yaml
sudo install -m 0755 /tmp/llmhub /usr/local/bin/llmhub
llmhub -h
```

You can also install from this repository with:

```bash
make install-latest
```

Use `sudo make install-latest` when your user cannot write to `/usr/local/bin`.

### VPS service setup

Create a dedicated user and runtime directories:

```bash
sudo useradd --system --home /var/lib/llmhub --shell /usr/sbin/nologin llmhub
sudo mkdir -p /etc/llmhub /var/lib/llmhub/auths /var/log/llmhub
sudo install -m 0640 -o root -g llmhub /tmp/config.example.yaml /etc/llmhub/config.yaml
sudo chown -R llmhub:llmhub /var/lib/llmhub /var/log/llmhub
sudo chmod 750 /var/lib/llmhub /var/lib/llmhub/auths
```

Edit `/etc/llmhub/config.yaml` for providers, accounts, API keys, and storage paths. For a file-store deployment, set the auth directory explicitly:

```yaml
auth-dir: "/var/lib/llmhub/auths"
```

Install a systemd unit:

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
journalctl -u llmhub -f
```

Point your AI coding tool at `http://SERVER_IP:8317/v1`.

The management panel is served at `http://SERVER_IP:8317/management.html`. Before exposing it beyond localhost, deliberately configure `remote-management.secret-key` and `remote-management.allow-remote` in `/etc/llmhub/config.yaml`.

### Build from source

For local development, build from source with Go 1.21+ and Bun:

```bash
make build
./llmhub -config config.yaml
```

See `config.example.yaml` for the full configuration reference.

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
