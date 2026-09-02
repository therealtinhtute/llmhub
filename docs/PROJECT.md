# PROJECT — identity

## What is this project?
- LLMHub is an OpenAI/Gemini/Claude/Codex/Grok-compatible proxy that forwards CLI clients through OAuth sessions instead of provider API keys.

## Who is it for?
- Operators running multi-account CLI proxies, and developers embedding the Go SDK (`sdk/cliproxy`).

## Non-goals
- Upstream branding, README/sponsors, or management-asset GitHub updater.
- File/YAML as runtime source of truth; pluginhost gRPC platform.
- Replacing llmhub management UI layout, tokens, or i18n.

## How do we run the tests?
- `go test ./...`
- `make build`
- `git diff --check`
- Frontend: type check, lint, production build, browser runtime — no new files under `web/`

## Architecture in one breath
- runtime shape: Go Gin proxy plus embedded React management panel
- where state lives: Postgres (config, credentials, usage); `LLMHUB_HOST`/`LLMHUB_PORT` are process overrides only
- entrypoints: `cmd/server`, `sdk/cliproxy.Service`

## What are we working on right now?
- plan: docs/plans/active/cliproxyapi-v7.2.147-parity.md (active)
