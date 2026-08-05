# Plan B+C — Provider preset catalog, model passthrough, OpenCode Free

Status: **awaiting approval** · Created 2026-08-05 · Revised 2026-08-05 · Ships independently
Source: `decolua/9router` `open-sse/providers/registry/` (~120 entries)
Reference skill: `.claude/skills/9router-port/references/providers.md`, `porting-map.md`

## Building

Three small things that share one code path and are not worth separating:

1. A curated catalog of known OpenAI-compatible providers, shipped as embedded data.
2. A `passthrough` flag so aggregator providers stop requiring a hand-written model list.
3. OpenCode Free wired up as the first genuinely zero-credential upstream.

## Seed set — three providers, all free-tier, all live-probed

| id | Why it is in | Credential | Probe (2026-08-05) |
|---|---|---|---|
| `opencode` | Zero-credential. The only upstream here that works with no signup at all. | none (literal `public`) | `GET /zen/v1/models` → **200** |
| `openrouter` | Aggregator. One key unlocks 27+ free models plus everything paid. | free key | `GET /api/v1/models` → **200** |
| `nvidia` | Free for NVIDIA Developer Program members. Hosts DeepSeek/GLM/Kimi/MiniMax/Nemotron. | free key | `GET /v1/models` → **200** |

Deliberately **three, not fifteen**. An earlier draft listed 15 providers lifted from 9router's
registry; that catalog is inventory, not a feature, and half of it would ship unverified. The
remaining candidates (groq, cerebras, deepseek, together, fireworks, siliconflow, mistral, chutes,
hyperbolic, llm7, …) stay documented in `references/providers.md` and can be appended one at a time
as each is live-probed — appending a preset is a JSON edit, not a code change.

### The `verified` field

Every preset carries `verified: true|false`.

- `true` — the `base_url` shape and `models_url` were probed live and returned `200` on the date in
  `verified_at`. All three seed entries are `true`.
- `false` — transcribed from 9router's registry but never probed by us. Any preset added without a
  live probe **must** ship `false`.

The panel renders an **unverified** badge next to any `false` preset. This exists so the catalog can
grow cheaply without the growth quietly implying we tested anything.

Scope of the claim, stated so it is not overread: a `true` means *the endpoint answered*. It does
not mean a chat completion was billed and streamed — for `openrouter` and `nvidia` that needs a key
we do not hold. Only `opencode` is verifiable end-to-end without credentials.

## Not building

- **No declarative provider registry in Go.** LLMHub's `openai-compatibility`
  (`internal/config/config.go:569`) already is 9router's `executor: "default"` path. What is
  missing is content, not mechanism — see `references/porting-map.md`.
- **No Anthropic-format providers.** `glm`, `minimax`, and `minimax-cn` were in the earlier draft's
  seed set and were **wrong there**: 9router declares them `format: "claude"` against
  Anthropic-messages endpoints (`https://api.z.ai/api/anthropic/v1/messages`,
  `https://api.minimax.io/anthropic/v1/messages`), authenticating with `x-api-key`, not
  `Authorization: Bearer`. As `openai-compatibility` entries they would fail on the first request.
  Their correct home is LLMHub's `claude-api-key` config, which already has a `BaseURL` field
  (`internal/config/config.go:391`) — a separate, later change. Recorded here so the finding is not
  rediscovered.
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
  `passthrough`, `default_api_key`, `verified`, `verified_at`.
- `internal/config/presets/presets.go` (new): `type Preset struct{...}`, `func All() []Preset`,
  parsed once via `sync.Once`.

All three entries in full — this is the whole seed file, not a sample:

```json
[
  {
    "id": "opencode",
    "display_name": "OpenCode Free",
    "base_url": "https://opencode.ai/zen/v1",
    "headers": { "x-opencode-client": "desktop" },
    "models_url": "https://opencode.ai/zen/v1/models",
    "free_tier_note": "No credentials required. Best-effort, undocumented endpoint.",
    "passthrough": true,
    "default_api_key": "public",
    "verified": true,
    "verified_at": "2026-08-05"
  },
  {
    "id": "openrouter",
    "display_name": "OpenRouter",
    "base_url": "https://openrouter.ai/api/v1",
    "headers": { "HTTP-Referer": "https://llmhub.local", "X-Title": "LLMHub" },
    "models_url": "https://openrouter.ai/api/v1/models",
    "signup_url": "https://openrouter.ai/settings/keys",
    "free_tier_note": "27+ free models, no card, 200 req/day (1000 after any credit purchase).",
    "passthrough": true,
    "verified": true,
    "verified_at": "2026-08-05"
  },
  {
    "id": "nvidia",
    "display_name": "NVIDIA NIM",
    "base_url": "https://integrate.api.nvidia.com/v1",
    "models_url": "https://integrate.api.nvidia.com/v1/models",
    "signup_url": "https://build.nvidia.com/settings/api-keys",
    "free_tier_note": "Free for NVIDIA Developer Program members (prototyping & testing).",
    "passthrough": true,
    "verified": true,
    "verified_at": "2026-08-05"
  }
]
```

Notes on the values:

- `base_url` is the **API root**, not the completions path. 9router stores the full
  `.../chat/completions` URL (`nvidia.js` `transport.baseUrl`); LLMHub's `OpenAICompatibility.BaseURL`
  is a root and the executor appends the path. Every entry above is converted, not copied.
- `nvidia` gets `passthrough: true` even though 9router ships a curated 11-model list, because that
  list mixes `kind: "tts"` / `"stt"` / `"embedding"` entries that are not chat models and goes stale
  fast. NVIDIA's `models_url` is public, so the panel can list ids without us maintaining them.
- `default_api_key` appears only on `opencode`. It is a public literal, not a secret.

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
  the free-tier note; an **unverified** badge on any preset with `verified: false` (none today, so
  the badge ships dormant and is exercised by the first unverified addition).
- No new test files under `web/` (project rule) — verify with type check, lint, production build,
  and a browser runtime check.

### 5. Docs

- One short page: what OpenCode Free is, that it is unauthenticated and therefore best-effort, and
  that it may disappear without notice. Plus what `verified` claims and what it does not.

## Verify

- `go test ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/...`
- New tests:
  - every preset has non-empty `id` and `base_url`; ids unique
  - every preset with `verified: true` also has a non-empty `verified_at`
  - `Passthrough: true` + empty `Models` → an unlisted model id resolves and reaches the upstream
  - `Passthrough: true` + non-empty `Models` → the explicit list still governs (no regression)
  - `Passthrough: false` + unlisted model → still fails resolution as today
  - `Prefix` is stripped before forwarding under passthrough
- `curl -s localhost:9090/v0/management/provider-presets` → **3** entries, all `verified: true`
- Re-probe at implementation time (endpoints move; a stale `verified: true` is worse than `false`):
  ```
  curl -sS -o /dev/null -w '%{http_code}\n' https://opencode.ai/zen/v1/models \
    -H "Authorization: Bearer public" -H "x-opencode-client: desktop"
  curl -sS -o /dev/null -w '%{http_code}\n' https://openrouter.ai/api/v1/models
  curl -sS -o /dev/null -w '%{http_code}\n' https://integrate.api.nvidia.com/v1/models
  ```
  Any non-200 → flip that preset to `verified: false` rather than dropping it.
- Live smoke: a streaming `POST /v1/chat/completions` against an OpenCode model returns SSE deltas.
- `make build-web && make build` — panel loads, preset dropdown prefills base-url.

## Rollback

`Passthrough` defaults to `false`, so existing config is untouched. Deleting the preset JSON and
the endpoint removes the rest. Nothing here changes an existing code path unless the new flag is
explicitly set.

## Risk

- OpenCode Free is an undocumented public endpoint with no contract. Treat breakage as expected, not
  as a regression — the docs page must say so.
- `verified: true` decays silently. It is a timestamped claim about one day, which is why
  `verified_at` is required and why re-probing is a verification step rather than a one-off.
- Everything else here is inert config.
