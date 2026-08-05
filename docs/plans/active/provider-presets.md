# Plan B+C — Provider preset catalog, model passthrough, OpenCode Free

Status: **awaiting approval** · Created 2026-08-05 · Ships independently
Source: `decolua/9router` `open-sse/providers/registry/` (~120 entries)
Reference skill: `.claude/skills/9router-port/references/providers.md`, `porting-map.md`

## Building

Three small things that share one code path and are not worth separating:

1. A curated catalog of known OpenAI-compatible providers, shipped as embedded data.
2. A `passthrough` flag so aggregator providers stop requiring a hand-written model list.
3. OpenCode Free wired up as the first genuinely zero-credential upstream.

## Not building

- **No declarative provider registry in Go.** LLMHub's `openai-compatibility`
  (`internal/config/config.go:569`) already is 9router's `executor: "default"` path. What is
  missing is content, not mechanism — see `references/porting-map.md`.
- **No OAuth providers.** GitHub Copilot, Cursor, Windsurf, Zed, Trae, Qoder are excluded: ToS/ban
  risk, and their transports pin editor versions that go stale (9router marks `github` itself
  `deprecated: true, deprecationNotice: "RISK_NOTICE"`, `open-sse/providers/registry/github.js:16`).
- **No MiMo Free.** Xiaomi ended the free channel; 9router has it `hidden: true`.
- **No new executor.** Everything here rides `internal/runtime/executor/openai_compat_executor.go`.

## Why no new executor is needed

`openai_compat_executor.go` sets `Authorization: Bearer "+apiKey` unconditionally
(lines 61, 145, 234, 343, 495) and applies arbitrary extra headers through
`util.ApplyCustomHeadersFromAttrs`. OpenCode Free's contract is exactly
`Authorization: Bearer public` plus `x-opencode-client: desktop` — an api-key entry whose key is
the literal string `public`, plus the existing `Headers` map. No special-casing.

## Changes

### 1. Preset catalog

- `internal/config/presets/providers.json` (new, `go:embed`). One entry per provider:
  `id`, `display_name`, `base_url`, `headers`, `models_url`, `signup_url`, `free_tier_note`,
  `passthrough`, `default_api_key` (only OpenCode uses it).
- `internal/config/presets/presets.go` (new): `type Preset struct{...}`, `func All() []Preset`,
  parsed once via `sync.Once`.
- Seed set (15), values lifted from `open-sse/providers/registry/{id}.js`:
  openrouter, groq, cerebras, deepseek, together, fireworks, nvidia, siliconflow, mistral, glm,
  minimax, chutes, hyperbolic, llm7, **opencode**.

OpenRouter's entry, as the shape reference (`open-sse/providers/registry/openrouter.js`):

```json
{
  "id": "openrouter",
  "display_name": "OpenRouter",
  "base_url": "https://openrouter.ai/api/v1",
  "headers": { "HTTP-Referer": "https://llmhub.local", "X-Title": "LLMHub" },
  "models_url": "https://openrouter.ai/api/v1/models",
  "signup_url": "https://openrouter.ai/settings/keys",
  "free_tier_note": "27+ free models, no card, 200 req/day (1000 after any credit purchase)",
  "passthrough": true
}
```

OpenCode Free (`open-sse/providers/registry/opencode.js`, `open-sse/executors/opencode.js`):

```json
{
  "id": "opencode",
  "display_name": "OpenCode Free",
  "base_url": "https://opencode.ai/zen/v1",
  "headers": { "x-opencode-client": "desktop" },
  "models_url": "https://opencode.ai/zen/v1/models",
  "free_tier_note": "No credentials required. Best-effort, undocumented endpoint.",
  "passthrough": true,
  "default_api_key": "public"
}
```

### 2. Management endpoint

- `GET /v0/management/provider-presets` in `internal/api/handlers/management/` → the catalog as
  JSON. Read-only, carries no secrets, no auth material.

### 3. `passthrough` flag

- `internal/config/config.go:569` — add to `OpenAICompatibility`:
  ```go
  // Passthrough forwards the client's model id to the upstream unchanged instead of
  // requiring an explicit Models entry. Intended for aggregators (OpenRouter, OpenCode).
  Passthrough bool `yaml:"passthrough,omitempty" json:"passthrough,omitempty"`
  ```
- `sdk/cliproxy/service.go:1785` `buildOpenAICompatibilityConfigModels` — when `Passthrough` is
  true and `Models` is empty, contribute no model entries (the provider stays selectable but
  advertises nothing). When `Models` is non-empty, behavior is unchanged — an explicit list still
  wins, so passthrough never hides a curated catalog.
- `sdk/cliproxy/auth/conductor.go:2426` `resolveOpenAICompatConfig` and the alias-resolution path —
  when the requested model has no alias match and the resolved compat entry has `Passthrough`,
  forward `req.Model` verbatim after stripping the configured `Prefix`, instead of failing
  resolution.
- `internal/watcher/diff/openai_compat.go:14` — include `passthrough` in the change description so
  a config reload logs it.

### 4. Panel

- `web/` provider-add screen: a preset dropdown that prefills base-url, headers, signup link, and
  the free-tier note. No new test files under `web/` (project rule) — verify with type check,
  lint, production build, and a browser runtime check.

### 5. Docs

- One short page: what OpenCode Free is, that it is unauthenticated and therefore best-effort, and
  that it may disappear without notice.

## Verify

- **Precondition, run first.** If this fails, drop OpenCode from the seed set and ship the rest:
  ```
  curl -sS https://opencode.ai/zen/v1/models \
    -H "Authorization: Bearer public" -H "x-opencode-client: desktop"
  ```
- `go test ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/...`
- New tests:
  - every preset has non-empty `id` and `base_url`; ids unique
  - `Passthrough: true` + empty `Models` → an unlisted model id resolves and reaches the upstream
  - `Passthrough: true` + non-empty `Models` → the explicit list still governs (no regression)
  - `Passthrough: false` + unlisted model → still fails resolution as today
  - `Prefix` is stripped before forwarding under passthrough
- `curl -s localhost:9090/v0/management/provider-presets` → 15 entries
- Live smoke: a streaming `POST /v1/chat/completions` against an OpenCode model returns SSE deltas.
- `make build-web && make build` — panel loads, preset dropdown prefills base-url.

## Rollback

`Passthrough` defaults to `false`, so existing config is untouched. Deleting the preset JSON and
the endpoint removes the rest. Nothing here changes an existing code path unless the new flag is
explicitly set.

## Risk

OpenCode Free is an undocumented public endpoint with no contract. Treat breakage as expected, not
as a regression — the docs page must say so. Everything else in this plan is inert config.
