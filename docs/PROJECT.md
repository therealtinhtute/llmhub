# PROJECT — identity (answer inline; this is the single forced write step at
# brainstorm lock; keep the whole file at or under 50 lines)

## What is this project?
- LLMHub is an OpenAI/Gemini/Claude/Codex/Grok-compatible proxy that forwards CLI clients through OAuth sessions instead of provider API keys.

## Who is it for?
- Operators running multi-account CLI proxies, and developers embedding the Go SDK (`sdk/cliproxy`).

## Non-goals
- Upstream branding, README/sponsors, or management-asset GitHub updater.
- File/YAML as runtime source of truth; pluginhost gRPC platform.
- Replacing llmhub management UI layout structure, navigation, or i18n translations (theme styling tokens and dark mode fixes are governed by active plans).

## What are the gate commands?
- run from: repository root
- tests: `go test ./...`
- types: n/a
- lint: n/a
- build: `make build`
- format: `gofmt -l .` and `git diff --check`

## Architecture in one breath
- runtime shape: Go Gin proxy plus embedded React management panel
- where state lives: Postgres (config, credentials, usage); `LLMHUB_HOST`/`LLMHUB_PORT` are process overrides only
- entrypoints: `cmd/server`, `sdk/cliproxy.Service`

## What are we working on right now?
- plan: docs/plans/active/cliproxyapi-v7.3.3-parity.md (active)
