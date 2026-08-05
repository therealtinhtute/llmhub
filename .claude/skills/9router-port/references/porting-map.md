# Structural tradeoffs: 9router's declarative registry vs LLMHub's Go-per-provider

## What 9router does

One file per provider: `open-sse/providers/registry/{id}.js`, ~120 of them, each a plain data
object. `open-sse/providers/index.js` folds the whole directory into four lookup tables at import
time:

```js
for (const entry of REGISTRY) {
  if (entry.transport) PROVIDERS[entry.id]      = buildTransport(entry.transport, entry.oauth);
  if (entry.models)    PROVIDER_MODELS[alias]   = entry.models.map(normalizeModel);
  if (entry.oauth)     PROVIDER_OAUTH[entry.id] = entry.oauth;
  /* + PROVIDER_MEDIA from top-level media keys */
}
```

The entry schema (`open-sse/providers/schema.js`) carries: `id`, `category`, `display`,
`transport` (baseUrl, format, headers, auth, quirks, retry, timeoutMs, **executor**, regions,
modelsFetcher, validateUrl), `oauth` (authorizeUrl, tokenUrl, deviceCodeUrl, scopes, redirectUri,
callbackPath, fixedPort, codeChallengeMethod, refreshLeadMs), `models`, `features`,
`thinkingConfig`, `passthroughModels`.

Two properties make it work:

1. **Defaults are layered, not repeated.** `PROVIDER_DEFAULTS` + `ENDPOINT_DEFAULTS[format]` mean a
   provider file only states what differs. OpenRouter's entry is ~60 lines and most of it is media
   config; the LLM part is 8 lines.
2. **`transport.executor` is an escape hatch.** Default `"default"` = generic OpenAI passthrough.
   Only providers with real protocol weirdness (kiro, cursor, antigravity, codex, github) name a
   custom executor. Roughly 80% of entries never do.

`oauth.{clientId,clientSecret,tokenUrl}` are declared once in the `oauth` block and injected into
`transport` at build time (`index.js:9`) — one source of truth, no duplication.

## What LLMHub does

Adding a provider today means, per `CLAUDE.md`:

1. `internal/auth/<provider>/` — token refresh
2. `internal/runtime/executor/<provider>_executor.go` — the `Executor` interface
3. `internal/translator/<provider>/` — three transform functions, registered via `init()`
4. `internal/registry/model_definitions.go` + `models/models.json`
5. wire into `sdk/cliproxy/` and blank-import in `cmd/server/main.go`

Five files and a rebuild. That is the right cost for kiro/antigravity/codex-class protocol work,
and the wrong cost for "Groq exists now".

LLMHub *does* already have the config-driven escape hatch: `openai-compatibility`
(`internal/config/config.go:569`) gives baseUrl + headers + api-key-entries + model aliases +
priority + weight, served by `internal/runtime/executor/openai_compat_executor.go`. It is
functionally 9router's `executor: "default"` path.

## Verdict

**Do not rebuild 9router's registry in Go.** LLMHub already has the mechanism; what it lacks is
*content* — a curated catalog so users don't hand-write YAML for each of 30 known providers.

The cheap version of the same benefit:

- Ship a preset catalog (embedded JSON) of known `openai-compatibility` entries: name, base-url,
  headers, models endpoint, free-tier note, signup URL.
- Expose it read-only from the management API so the web panel can offer "add from preset".
- A preset is pure data — adding Groq becomes a data PR, not a Go PR.

What 9router's registry has that a preset catalog would *not* cover, and that stays Go-side:

| capability | why it stays in Go |
|---|---|
| OAuth flow config per provider | LLMHub's OAuth is per-package with real refresh logic, not declarative |
| `modelsFetcher` (remote model discovery) | LLMHub has `internal/registry/model_updater.go`; a per-provider fetcher type is a separate feature |
| `quirks` / protocol shims | Real code, belongs in the executor |
| `regions` / `defaultRegion` | Vertex-class concern, already handled |

## Two smaller borrowings worth flagging

- **`passthroughModels: true`** — for aggregators (OpenRouter, OpenCode), forward the client's
  model id untouched instead of maintaining an alias list. LLMHub's `OpenAICompatibilityModel`
  requires explicit `name`/`alias`, so aggregators mean hand-listing hundreds of models. A
  `passthrough: true` flag on the `openai-compatibility` block removes that entirely.
- **`display.notice`** — each entry carries the signup/API-key URL and a one-line free-tier
  description ("27+ free models, no credit card, 200 req/day"). Cheap to carry in a preset, and it
  is most of the onboarding UX.

## Constraints that bind any port here

- LLMHub loads config from **Postgres**, not a working-directory `config.yaml` (project
  `CLAUDE.md`). New config blocks must round-trip through `/v0/management/config.yaml`.
- No new frontend test files under `web/` (project `CLAUDE.md`). Verify panel changes with type
  check, lint, production build, and a browser runtime check.
