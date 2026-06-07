# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build (embeds web UI first, then compiles Go binary)
make build

# Run in dev mode (embed + go run)
make dev

# Build only the React web panel
make build-web

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/runtime/executor/...

# Run a specific test
go test ./internal/runtime/executor/ -run TestCodexExecutorExecute
```

`make build-web` requires [Bun](https://bun.sh). Without running `make embed` first, the management panel will not be included in the binary.

## Architecture

LLMHub is an OpenAI/Gemini/Claude-compatible proxy server. It accepts requests from AI coding tools (Claude Code, Codex CLI, etc.) and forwards them to the real providers using OAuth sessions, avoiding the need for API keys.

### Request pipeline

```
Client → Gin HTTP server (internal/api)
       → Route handler selects executor from registry
       → Executor translates request format (sdk/translator)
       → Executor authenticates and calls provider HTTP/WS
       → Response translated back to client format
       → SSE stream returned to client
```

### Key packages

| Path | Purpose |
|------|---------|
| `cmd/server/` | Entry point; CLI flags for login modes, TUI, config path, token stores |
| `sdk/cliproxy/` | Embeddable `Service` struct; the public SDK entry point |
| `internal/api/` | Gin router, middleware, route registration |
| `internal/runtime/executor/` | Per-provider executors: `claude_executor.go`, `codex_executor.go`, `gemini_executor.go`, `xai_executor.go`, etc. |
| `sdk/translator/` + `internal/translator/` | `RequestTransform` / `ResponseTransform` function types; per-provider format converters |
| `internal/auth/` | Per-provider OAuth token refresh logic (claude, codex, gemini, xai, antigravity, kimi) |
| `internal/registry/` | Model registry with remote update polling and local cache |
| `internal/config/` | Config struct and YAML loading |
| `internal/tui/` | Bubbletea terminal management UI (`--tui` flag) |
| `web/` | React + Vite management panel, compiled and embedded into `internal/managementasset/static/` |
| `internal/store/` | Token store backends: file (default), Postgres, S3-compatible object store, Git repo |

### Translator pattern

Every provider pair has three transform functions keyed in the `sdk/translator` registry:
- `RequestTransform` — converts OpenAI-format JSON → provider format
- `ResponseStreamTransform` — converts provider SSE chunks → OpenAI SSE
- `ResponseNonStreamTransform` — converts provider JSON response → OpenAI JSON

Translators live under `internal/translator/<provider>/` and are registered via `init()` (blank-imported in `cmd/server/main.go` via `_ "github.com/therealtinhtute/llmhub/internal/translator"`).

### Config and runtime storage

Normal server startup is Postgres-only. Bootstrap the first runtime snapshot
with `llmhub init-db-from-env` using `PGSTORE_DSN` plus
`LLMHUB_INIT_CONFIG_YAML` or `LLMHUB_INIT_CONFIG_B64`, then run `llmhub`
without a local `config.yaml`.

At runtime, `llmhub` loads config from Postgres and only applies
`LLMHUB_HOST` / `LLMHUB_PORT` as process-level overrides. The management API
still exposes `/v0/management/config.yaml`, but that YAML is stored in
Postgres rather than the working directory.

### Provider routing

Requests are routed to executors via model name or prefix. The registry (`internal/registry/`) maps client-visible model names to executor+auth combos. For provider-specific protocol shapes, use the explicit paths (`/api/provider/{provider}/v1/...`) instead of the merged `/v1/...` endpoints.

### Web panel

The management panel is a React/Vite SPA (`web/`). It is compiled and copied to `internal/managementasset/static/management.html` by `make embed`. The binary serves it from that embedded path — the panel is absent if `make embed` was not run.
