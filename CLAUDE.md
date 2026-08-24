# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build (embeds web UI first, then compiles Go binary)
make build

# Run in dev mode (file-based token store, no Postgres)
make dev

# Run with Postgres (reads .env, seeds DB on first run)
make dev-pg

# Build only the React web panel
make build-web

# Run the React panel with Vite hot reload (proxies API to localhost:9090)
make dev-web

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/runtime/executor/...

# Run a specific test
go test ./internal/runtime/executor/ -run TestCodexExecutorExecute
```

`make build-web` requires [Bun](https://bun.sh). Without running `make embed` first, the management panel will not be included in the binary.

Integration tests that require external services are gated by env vars and skip automatically when unset (e.g. `LLMHUB_POSTGRES_TEST_DSN` for Postgres store tests).

## Architecture

LLMHub is an OpenAI/Gemini/Claude-compatible proxy server. It accepts requests from AI coding tools (Claude Code, Codex CLI, and others) and forwards them to the real providers using OAuth sessions, avoiding the need for API keys.

### Request pipeline

```
Client → Gin HTTP server (internal/api)
       → Route handler selects executor from registry
       → Executor translates request format (sdk/translator)
       → Executor authenticates and calls provider HTTP/WS
       → Response translated back to client format
       → SSE stream returned to client
```

### SDK vs internal split

`sdk/` is the embeddable public surface; `internal/` is the server-only implementation.

| Layer | SDK (`sdk/`) | Internal (`internal/`) |
|-------|-------------|------------------------|
| Translator types & registry | `sdk/translator/` | `internal/translator/<provider>/` |
| Executor interface | `sdk/cliproxy/executor/` | `internal/runtime/executor/` |
| Auth types | `sdk/cliproxy/auth/`, `sdk/auth/` | `internal/auth/<provider>/` |
| Service entry point | `sdk/cliproxy/` | — |
| HTTP handlers | `sdk/api/handlers/` | `internal/api/handlers/`, `internal/api/modules/` |

### Key packages

| Path | Purpose |
|------|---------|
| `cmd/server/` | Entry point; CLI flags for login modes, TUI, config path, token stores |
| `sdk/cliproxy/` | Embeddable `Service` struct; the public SDK entry point |
| `internal/api/` | Gin router, middleware, route registration |
| `internal/runtime/executor/` | Per-provider executors: `claude_executor.go`, `codex_executor.go`, `gemini_executor.go`, `xai_executor.go`, `kiro_executor.go`, etc. |
| `internal/runtime/executor/helps/` | Shared executor helpers: caching, logging, token counting, uTLS client, tool cloak, session/user-ID caches |
| `sdk/translator/` + `internal/translator/` | `RequestTransform` / `ResponseTransform` function types; per-provider format converters |
| `internal/auth/` | Per-provider OAuth token refresh logic (claude, codex, gemini, xai, antigravity, kimi, kiro, vertex) |
| `internal/registry/` | Model registry with static definitions (`models/models.json`), remote update polling, and ref-count–based model visibility |
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

### Adding a new provider

1. Create `internal/auth/<provider>/` — implement token refresh.
2. Create `internal/runtime/executor/<provider>_executor.go` — implement the `cliproxyexecutor.Executor` interface (`Execute`, `ExecuteStream`, `Identifier`, `Refresh`, …).
3. Create `internal/translator/<provider>/` — implement and register the three transform functions.
4. Add model definitions to `internal/registry/model_definitions.go` and `internal/registry/models/models.json`.
5. Wire the executor into the registry in `sdk/cliproxy/` and blank-import the translator in `cmd/server/main.go`.

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

For frontend development, run `make dev-web` (hot reload at Vite default port, proxying API calls to `DEV_WEB_API_BASE`, default `http://localhost:9090`) alongside a running `make dev` server.

Do not create new frontend test files anywhere under `web/`. Verify frontend changes with type checking, linting, production builds, and browser runtime checks instead.

<!-- ZHARNESS:BEGIN -->
@AGENTS.md
<!-- ZHARNESS:END -->
