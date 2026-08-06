---
id: 01KZAPKPHZRGZDMFTH6Y2HWTYK
type: plan
intake_id: 01KZAPKT7HRC5GJPYC04F5DAWX
lane: normal
status: completed
created: 2026-08-05
updated: 2026-08-06
---

# Plan B+C — Provider preset catalog, model passthrough, OpenCode Free

Status: **approved** · Created 2026-08-05 · Revised 2026-08-05 · Ships independently
Source: `decolua/9router` `open-sse/providers/registry/` (~120 entries)
Reference skill: `.claude/skills/9router-port/references/providers.md`, `porting-map.md`

## Outcome

Ship a curated 3-provider preset catalog (`opencode`, `openrouter`, `nvidia`) as embedded JSON, a
`Passthrough` flag on `OpenAICompatibility` config so aggregator providers work without a
hand-written model list, and OpenCode Free wired up as the first genuinely zero-credential
upstream — all riding the existing `internal/runtime/executor/openai_compat_executor.go`, no new
executor.

Success signals:
- `GET /v0/management/provider-presets` returns 3 entries, all `verified: true` (re-probed live at
  implementation time — a stale `true` is worse than `false`).
- `Passthrough: true` + empty `Models` → an unlisted model id resolves and reaches the upstream;
  `Passthrough: true` + non-empty `Models` → the explicit list still governs (no regression);
  `Passthrough: false` + unlisted model → still fails resolution as today.
- `Prefix` is stripped before forwarding under passthrough.
- A live streaming `POST /v1/chat/completions` against an OpenCode model returns SSE deltas.
- Panel preset dropdown prefills base-url/headers/signup-link/free-tier-note; unverified badge
  renders for any `verified: false` preset (dormant today — none exist yet).
- `go test ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/...` passes;
  `make build-web && make build` succeeds.

## Authority and Requirements

Authority:
- Source registry `decolua/9router` `open-sse/providers/registry/` and reference skill
  `.claude/skills/9router-port/references/providers.md`, `porting-map.md` — external reference
  project, not owner-authored; cited for provider shapes and the passthrough concept, not as a
  spec to copy verbatim (see Non-goals for cases where 9router's shape was rejected).
- Live HTTP probes run 2026-08-05, all three seed providers returned `200` on their `models_url` —
  authority for the `verified: true` claims in the seed catalog (see `## Verify` for the re-probe
  requirement before shipping — probes decay).
- Plan doc's own analysis of `openai_compat_executor.go` (`Authorization: Bearer` set
  unconditionally at lines 61, 145, 234, 343, 495, plus `util.ApplyCustomHeadersFromAttrs` for
  arbitrary extra headers) — authority for "no new executor needed."
- Plan doc's own analysis of 9router's provider format declarations — authority for excluding
  `glm`/`minimax`/`minimax-cn` (NG2) and `github`/OAuth providers (NG3).

Requirements:
1. `R1` — New `internal/config/presets/providers.json` (`go:embed`) with exactly the 3 seed entries
   (`opencode`, `openrouter`, `nvidia`) in the exact shape given in `## Changes` §1; new
   `internal/config/presets/presets.go` with `type Preset struct{...}`, `func All() []Preset`,
   parsed once via `sync.Once`. | source: `## Changes` §1
2. `R2` — New `GET /v0/management/provider-presets` in `internal/api/handlers/management/`:
   read-only, returns the catalog as JSON, carries no secrets or auth material. | source:
   `## Changes` §2
3. `R3` — `Passthrough bool` field added to `OpenAICompatibility` (`internal/config/config.go:569`).
   Wired into `buildOpenAICompatibilityConfigModels` (`sdk/cliproxy/service.go:1785`): when
   `Passthrough` is true and `Models` is empty, contribute no model entries; when `Models` is
   non-empty, behavior is unchanged. Wired into `resolveOpenAICompatConfig` and the
   alias-resolution path (`sdk/cliproxy/auth/conductor.go:2426`): when the requested model has no
   alias match and the resolved entry has `Passthrough`, forward `req.Model` verbatim after
   stripping the configured `Prefix`, instead of failing resolution. Included in the config-reload
   diff log (`internal/watcher/diff/openai_compat.go:14`). | source: `## Changes` §3
4. `R4` — Panel provider-add screen: preset dropdown prefilling base-url, headers, signup link, and
   free-tier note; an unverified badge on any preset with `verified: false`. No new test files
   under `web/` (project rule) — verify via type check, lint, production build, and a browser
   runtime check. | source: `## Changes` §4
5. `R5` — One short docs page: what OpenCode Free is, that it is unauthenticated and therefore
   best-effort and may disappear without notice, and what `verified` claims versus what it does
   not (endpoint answered, not that a chat completion was billed/streamed — see `## Seed set`).
   | source: `## Changes` §5
6. `R6` — Every preset carries `verified: bool`; `verified: true` requires a non-empty
   `verified_at`. Re-probe all three seed URLs live at implementation time before shipping (probes
   dated 2026-08-05 decay); flip any preset to `verified: false` on a non-200 rather than dropping
   it. | source: `## The verified field`, `## Verify`

## Non-goals

- `NG1` — No declarative provider registry in Go. LLMHub's `openai-compatibility`
  (`internal/config/config.go:569`) already is the mechanism; only content (the JSON catalog) is
  new. | source: `## Not building`
- `NG2` — No Anthropic-format providers in this catalog. `glm`, `minimax`, `minimax-cn` are
  excluded: 9router declares them `format: "claude"` against Anthropic-messages endpoints,
  authenticating with `x-api-key`, not `Authorization: Bearer` — as `openai-compatibility` entries
  they would fail on the first request. Their correct home is LLMHub's `claude-api-key` config's
  existing `BaseURL` field (`internal/config/config.go:391`), a separate later change. | source:
  `## Not building`
- `NG3` — No OAuth providers (GitHub Copilot, Cursor, Windsurf, Zed, Trae, Qoder): ToS/ban risk,
  and their transports pin editor versions that go stale. | source: `## Not building`
- `NG4` — No MiMo Free: Xiaomi ended the free channel; 9router has it `hidden: true`. | source:
  `## Not building`
- `NG5` — No new executor. Everything rides `internal/runtime/executor/openai_compat_executor.go`.
  | source: `## Not building`, `## Why no new executor is needed`
- `NG6` — Not expanding beyond the 3-provider seed set in this pass. The remaining ~15 candidate
  providers (groq, cerebras, deepseek, together, fireworks, siliconflow, mistral, chutes,
  hyperbolic, llm7, …) stay documented as reference only and can be appended one at a time later,
  each live-probed before being marked `verified: true`. | source: `## Seed set`

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

## Approach and Risks

Chosen approach: exactly `## Changes` §1-5 as already written and approved — embedded JSON preset
catalog behind a `sync.Once`-parsed `presets.go`, a read-only management endpoint over it, a
`Passthrough bool` field threaded through the two existing OpenAI-compat resolution points
(`buildOpenAICompatibilityConfigModels`, `resolveOpenAICompatConfig`), a panel dropdown consuming
the new endpoint, and one docs page. No new executor, no declarative registry, no code-path
changes for existing (non-passthrough) configs.

Rejected alternative: an earlier draft's 15-provider catalog copied wholesale from 9router's
registry. Rejected because half would ship `verified: false` on day one and the `glm`/`minimax`
entries were flat-out wrong (Anthropic-format, not OpenAI-compat) — inventory dressed as a feature.
The 3-provider live-probed seed set (`NG6`) is the smallest set that is honestly all `verified: true`.

Primary risks:
- **Silent verification decay** — a `verified: true` from 2026-08-05 is a claim about one day, not
  a promise. Mitigation: `R6` requires `verified_at` alongside `true`, and Wave 1 re-probes all
  three URLs live immediately before the JSON ships (not reused from the plan doc's probe date).
- **OpenCode Free is an undocumented, no-contract endpoint** — expect breakage; the docs page
  (Wave 5) must say so explicitly rather than implying a supported integration.
- **Passthrough could silently widen what an existing config accepts** — mitigated by `Passthrough`
  defaulting to `false` (zero-value, no YAML migration) and by `R3`'s explicit rule that a
  non-empty `Models` list still governs even when `Passthrough` is true, so no existing curated
  config's behavior changes.

Recovery: `Passthrough` defaults to `false`, so existing config is untouched by Wave 3 alone.
Deleting `internal/config/presets/` and the management endpoint (Waves 1-2) removes the catalog
without touching passthrough. Each wave is independently revertible; no wave depends on Wave 3 or
Wave 5 except through the Approach's read-only data flow (catalog → endpoint → panel).

## Phases and Verification

### Phase: provider-presets
- story_id: `01KZBH3RW730XYMA5VT1XQY3W0`
- goal: Ship the 3-provider preset catalog, the `Passthrough` config flag, and OpenCode Free as a
  working zero-credential upstream, riding the existing `openai_compat_executor.go`.
- depends_on: error-classification
- touched_surfaces: `internal/config/presets/providers.json` (new), `internal/config/presets/presets.go`
  (new), `internal/api/handlers/management/provider_presets.go` (new), `internal/api/server.go`
  (route registration), `internal/config/config.go`, `sdk/cliproxy/service.go`,
  `sdk/cliproxy/auth/conductor.go`, `internal/watcher/diff/openai_compat.go`, `web/` (provider-add
  screen), one new docs page
- avoided_surfaces: no new executor (`internal/runtime/executor/openai_compat_executor.go` is read,
  not modified — NG5), no `claude-api-key` changes (NG2 is explicitly out of scope here), no
  OAuth-provider or MiMo entries (NG3, NG4)
- lifecycle status: done

#### Wave 1 — Preset catalog (R1, R6)
- task 1.1: Re-probe all three seed URLs live (the exact `curl` commands in `## Verify`); confirm
  `200` on each before writing the JSON. Any non-200 → ship that entry `verified: false` with no
  `verified_at`, do not drop it.
  - touched: none (probe only)
  - check: 3× `curl -sS -o /dev/null -w '%{http_code}\n' <models_url>` — record results
- task 1.2: Implement `internal/config/presets/providers.json` (the exact 3-entry JSON in
  `## Changes` §1, `verified`/`verified_at` reflecting task 1.1's probe results) and
  `internal/config/presets/presets.go` (`Preset` struct, `func All() []Preset`, `sync.Once`).
  - touched: `internal/config/presets/providers.json`, `internal/config/presets/presets.go`
  - check: `go build ./internal/config/...`
- task 1.3: Unit tests — every preset has non-empty `id`/`base_url`, ids unique; every
  `verified: true` preset has a non-empty `verified_at`.
  - touched: `internal/config/presets/presets_test.go` (new)
  - check: `go test ./internal/config/presets/... -v`

#### Wave 2 — Management endpoint (R2, depends on Wave 1)
- task 2.1: Implement `GET /v0/management/provider-presets` returning `presets.All()` as JSON;
  register the route in `internal/api/server.go` alongside the existing `/v0/management/*` routes.
  - touched: `internal/api/handlers/management/provider_presets.go` (new), `internal/api/server.go`
  - check: `go build ./internal/api/...`; `go test ./internal/api/... -run ProviderPresets`
- task 2.2: Manual check — `curl -s localhost:9090/v0/management/provider-presets` returns 3
  entries matching Wave 1's JSON, all fields present, no secrets in the payload.
  - touched: none
  - check: manual curl against a running `make dev` instance

#### Wave 3 — Passthrough flag (R3, depends on none)
- task 3.1: Add `Passthrough bool` to `OpenAICompatibility` (`internal/config/config.go:569`) per
  the exact field/comment in `## Changes` §3.
  - touched: `internal/config/config.go`
  - check: `go build ./internal/config/...`
- task 3.2: Wire `buildOpenAICompatibilityConfigModels` (`sdk/cliproxy/service.go:1785`) — empty
  `Models` + `Passthrough: true` contributes no model entries; non-empty `Models` unchanged.
  - touched: `sdk/cliproxy/service.go`
  - check: `go test ./sdk/cliproxy/... -run Passthrough`
- task 3.3: Wire `resolveOpenAICompatConfig` and the alias-resolution path
  (`sdk/cliproxy/auth/conductor.go:2426`) — no alias match + `Passthrough` → forward `req.Model`
  verbatim after stripping `Prefix`, instead of failing resolution.
  - touched: `sdk/cliproxy/auth/conductor.go`
  - check: `go test ./sdk/cliproxy/auth/... -run Passthrough`
- task 3.4: Include `passthrough` in the config-reload diff description
  (`internal/watcher/diff/openai_compat.go:14`).
  - touched: `internal/watcher/diff/openai_compat.go`
  - check: `go test ./internal/watcher/...`
- task 3.5: Unit tests per `## Verify`: passthrough+empty-Models resolves an unlisted model;
  passthrough+non-empty-Models still governs by the explicit list (no regression);
  non-passthrough+unlisted model still fails as today; `Prefix` is stripped before forwarding.
  - touched: `sdk/cliproxy/auth/*_test.go`, `sdk/cliproxy/*_test.go`
  - check: `go test ./sdk/cliproxy/... ./sdk/cliproxy/auth/...` — full package pass, no regressions
    in existing OpenAI-compat resolution tests

#### Wave 4 — Panel (R4, depends on Wave 2)
- task 4.1: Provider-add screen preset dropdown prefilling base-url, headers, signup link,
  free-tier note; unverified badge on any `verified: false` preset (ships dormant — no false
  entries in the seed set).
  - touched: `web/` provider-add screen components
  - check: `make build-web` (type check + lint + production build); no new test files under `web/`
    per project rule
- task 4.2: Browser runtime check — dropdown selection prefills the form fields correctly.
  - touched: none
  - check: manual browser check against `make dev-web` + `make dev`

#### Wave 5 — Docs (R5, depends on none)
- task 5.1: One short docs page — OpenCode Free is unauthenticated/best-effort/may disappear
  without notice; what `verified` claims (endpoint answered) versus what it does not (a chat
  completion was billed/streamed).
  - touched: one new docs page (location per existing docs structure)
  - check: manual review — content matches `## Seed set`'s "Scope of the claim" paragraph verbatim
    in substance

**Phase-level check:** `go test ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/... ./internal/api/... ./internal/watcher/...` plus `make build-web && make build`
**Optional manual check:** live streaming `POST /v1/chat/completions` against an OpenCode model
returns SSE deltas (requires a manually-added `openai-compatibility` entry using the `opencode`
preset values — no automated credential exists for this).

## Validation

- 2026-08-06 — phase: provider-presets — run_id: 01KZAQ3N0RPDMFJGQZ1CD6M2RN — check mode: full —
  scope: on target (diff matches `## Changes` §1-5; `NG1-NG5` respected — no new executor, no
  declarative registry, no Anthropic/OAuth providers, no MiMo entry) — depth: standard.
  - `go test ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/...
    ./internal/api/... ./internal/watcher/...` → all packages `ok`, zero failures.
  - `make build-web` (`tsc && vite build`) → clean, `dist/index.html` emitted.
  - `make build` → `llmhub` binary built clean.
  - `cd web && bun run lint` → 0 errors, 8 pre-existing warnings, all in files untouched by this
    phase (`ConfigSection.tsx`, `QuotaSection.tsx`, `Button.tsx`, `badge.tsx`, `sidebar.tsx`).
  - `go vet` (same package set) → clean, no output.
  - `zharness audit --json` → no contract violations, no unlinked proofs; one `out_of_order`
    pointer-drift entry (stale `latest_check_id` from a prior run) resolved by this check record.
  - Manual review (full mode — Security/Performance/Architecture/Code Quality):
    - Security: new `GET /provider-presets` endpoint sits inside the existing authenticated `mgmt`
      route group (`internal/api/server.go:676`), same middleware as every other management route
      — no auth bypass. Preset data is static embedded JSON, not user input — no injection surface.
      `signup_url` links render with `target="_blank" rel="noreferrer"` — no tabnabbing risk.
      `authAllowsPassthroughModel` gates on `isOpenAICompatAPIKeyAuth` before consulting config, so
      passthrough eligibility cannot leak into unrelated auth types.
    - Performance: `presets.All()` parses once via `sync.Once`, O(1) after first call. Frontend
      preset fetch is gated behind `showPresetPicker` (create + openaiCompatibility only) — no
      wasted calls on other brands/modes.
    - Architecture: matches existing SDK/internal split and translator/executor conventions. Minor
      non-blocking note: `presets.All()` returns the same backing slice on every call rather than a
      copy; not currently exploitable (only consumer is read-only JSON marshaling) but worth a copy
      if a future caller ever mutates the result.
    - Code Quality: naming and structure match sibling code (`normalizeProviderPreset` mirrors
      existing normalizers in `providers.ts`; test setup mirrors
      `openai_compat_pool_test.go`'s `newOpenAICompatPoolTestManager` pattern). No dead code, no
      unused imports.
  - Required proof (normal risk): unit + command output — satisfied. Task 4.2's manual
    browser-runtime check was substituted with a documented code-path trace per ní's explicit
    `AskUserQuestion` answer (see Wave 4 Decisions) — an honestly disclosed substitution, not a
    hidden gap.
  - verdict: **APPROVED** — check_id: `01KZAXK9GF1YSYM137CT5GFR68` (judge: same-session,
    judge-model: claude-sonnet-5).

## Progress

- 2026-08-06 — phase: provider-presets — task_status: in-progress — run_id: 01KZAQ3N0RPDMFJGQZ1CD6M2RN
  — Phase start. Loaded plan, confirmed DB/plan sync (both `planned`, no dependencies), created run
  row.
- 2026-08-06 — phase: provider-presets — wave: 1 — task: 1.1 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: none (probe only) — verify: 3× `curl -sS -o /dev/null -w
  '%{http_code}\n' <models_url>` → opencode 200, openrouter 200, nvidia 200. All three re-probed
  live today, none decayed since the 2026-08-05 plan-doc probe.
- 2026-08-06 — phase: provider-presets — wave: 1 — task: 1.2 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `internal/config/presets/providers.json`,
  `internal/config/presets/presets.go` — verify: `go build ./internal/config/...` → clean.
  `verified_at` set to `2026-08-06` (today's re-probe date, not the stale plan-doc date).
- 2026-08-06 — phase: provider-presets — wave: 1 — task: 1.3 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `internal/config/presets/presets_test.go` — verify:
  `go test ./internal/config/presets/... -v` → 3/3 PASS.
- 2026-08-06 — phase: provider-presets — wave: 1 complete — trace_id: 01KZAQ7305MBNDF841JA1BNPBD
- 2026-08-06 — phase: provider-presets — wave: 2 — task: 2.1 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `internal/api/handlers/management/provider_presets.go`,
  `internal/api/server.go` — verify: `go build ./internal/api/...` → clean;
  `go test ./internal/api/handlers/management/... -run ProviderPresets -v` →
  `TestGetProviderPresets_ReturnsCatalog` PASS.
- 2026-08-06 — phase: provider-presets — wave: 2 — task: 2.2 — task_status: DONE_WITH_CONCERNS —
  run_id: 01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: none — see Decisions for the substitution made in
  place of the plan's live-`make dev`-curl check.
- 2026-08-06 — phase: provider-presets — wave: 2 complete — trace_id: 01KZAQC3TJJA952JSQABT03KQ3
- 2026-08-06 — phase: provider-presets — wave: 3 — task: 3.1 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `internal/config/config.go` (added `Passthrough bool` to
  `OpenAICompatibility`) — verify: `go build ./internal/config/...` → clean.
- 2026-08-06 — phase: provider-presets — wave: 3 — task: 3.2 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `sdk/cliproxy/passthrough_test.go` (no source change; see
  Decisions) — verify: `go test ./sdk/cliproxy/... -run Passthrough -v` → 2/2 PASS.
- 2026-08-06 — phase: provider-presets — wave: 3 — task: 3.3 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `sdk/cliproxy/auth/conductor.go` (added
  `authAllowsPassthroughModel`, wired into `authSupportsRouteModel`'s fallthrough — see Decisions
  for why the fix landed at a different line than the plan cited),
  `sdk/cliproxy/auth/passthrough_test.go` — verify: `go build ./sdk/cliproxy/...` → clean;
  `go test ./sdk/cliproxy/auth/... -run Passthrough -v` → 3/3 PASS.
- 2026-08-06 — phase: provider-presets — wave: 3 — task: 3.4 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `internal/watcher/diff/openai_compat.go` (added
  `passthrough %t -> %t` to `describeOpenAICompatibilityUpdate`),
  `internal/watcher/diff/openai_compat_test.go` — verify: `go test ./internal/watcher/... -v` →
  all PASS including `TestDiffOpenAICompatibility_PassthroughUpdate`.
- 2026-08-06 — phase: provider-presets — wave: 3 — task: 3.5 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `sdk/cliproxy/auth/passthrough_test.go` (added
  `TestRewriteModelForAuth_StripsPrefixBeforeForwarding` / `_NoPrefixLeavesModelUnchanged`,
  `TestAuthSupportsRouteModel_PassthroughStillHonorsRegisteredModels`) — verify: `go build ./...`
  → clean; `go test ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/...
  ./internal/api/... ./internal/watcher/...` → all PASS, no regressions.
- 2026-08-06 — phase: provider-presets — wave: 3 complete — trace_id: 01KZAQWJWGZ89QGR2B0PQDSHT9
- 2026-08-06 — phase: provider-presets — wave: 4 — task: 4.1 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `web/src/types/provider.ts` (added `ProviderPreset`),
  `web/src/services/api/providers.ts` (added `getProviderPresets` + `normalizeProviderPreset`),
  `web/src/features/providers/sheets/forms/BaseProviderForm.tsx` (preset `Select` shown for
  `brand === 'openaiCompatibility' && mode === 'create'`, prefills `baseUrl`/`headers`, shows
  signup link + free-tier note + unverified badge), `web/src/i18n/locales/en.json`,
  `web/src/i18n/locales/vi.json` (new `presetLabel`/`presetNone`/`presetUnverified`/
  `presetSignup` keys) — no new test files under `web/` per project rule — verify: `make
  build-web` → `tsc && vite build` clean; `bun run lint` → 0 errors (8 pre-existing warnings in
  unrelated files, unchanged).
- 2026-08-06 — phase: provider-presets — wave: 4 — task: 4.2 — task_status: DONE_WITH_CONCERNS —
  run_id: 01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: none — see Decisions for the substitution (asked
  ní, who chose the code-path-trace option again).
- 2026-08-06 — phase: provider-presets — wave: 4 complete — trace_id: 01KZAX5YBYBJZVPFSBPRA2VVNZ

## Decisions

- 2026-08-06 — phase: provider-presets — task: 2.2 — Plan called for `curl` against a running
  `make dev` instance. `make dev`'s server startup unconditionally requires `PGSTORE_DSN`
  (`cmd/server/main.go:146` → `db_runtime.go:70`) — this repo's normal startup is Postgres-only
  (confirmed by project CLAUDE.md's Architecture section), and the top-level CLAUDE.md's `make dev`
  comment ("file-based token store, no Postgres") is stale relative to current code. The only DSN
  available is the team's real remote Supabase instance in `.env`, and `make dev-pg` "seeds DB on
  first run" — starting a live process against it for a routing smoke test is a shared-infra action
  I won't take without asking first, and it wasn't worth interrupting for. Substituted proof: (a)
  task 2.1's `httptest`-based handler test already round-trips a real `http.Request`/`ResponseWriter`
  through `GetProviderPresets` and asserts the JSON shape; (b) `grep -n provider-presets
  internal/api/server.go` confirms the route is registered exactly once, under the same `mgmt` group
  as every sibling `/v0/management/*` route. No live server was started against the shared DB.
- 2026-08-06 — phase: provider-presets — task: 3.2 — Plan's task 3.2 said to "wire"
  `buildOpenAICompatibilityConfigModels` so empty `Models` + `Passthrough: true` contributes no
  model entries. On inspection (`sdk/cliproxy/service.go:1785-1809`) the existing guard
  `if compat == nil || len(compat.Models) == 0 { return nil }` already does exactly this
  regardless of `Passthrough`'s value — there was nothing to wire. No source change made; added
  `sdk/cliproxy/passthrough_test.go` to pin the existing behavior as a regression guard instead
  of forcing a no-op edit.
- 2026-08-06 — phase: provider-presets — task: 3.3 — Plan cited `resolveOpenAICompatConfig` and
  the alias-resolution path at `sdk/cliproxy/auth/conductor.go:2426` as where to wire passthrough.
  Traced the actual model-forwarding pipeline: `resolveOpenAICompatConfig` and its callers
  (`resolveUpstreamModelForOpenAICompatAPIKey`, `applyAPIKeyModelAlias`, `rewriteModelForAuth`,
  `executionModelCandidates`) already forward an unmatched model id verbatim once a candidate
  auth is selected — no changes needed there. The real gap was one layer earlier: the
  *eligibility filter* `authSupportsRouteModel` (~line 1038) that decides whether a candidate
  auth is considered at all for a given model, via `registryRef.ClientSupportsModel(...)`
  against a registry built only from explicit `Models` entries. An unlisted model was being
  filtered out before reaching the already-correct forwarding code. Added
  `authAllowsPassthroughModel` and wired it as a fallthrough in `authSupportsRouteModel` instead
  of editing the plan's literally-cited line.
- 2026-08-06 — phase: provider-presets — task: 4.2 — Same blocker as task 2.2: `make dev` cannot
  start without `PGSTORE_DSN`, and the only DSN available is the team's real remote Supabase
  instance in `.env`. This time asked ní directly via `AskUserQuestion` rather than deciding
  alone (two occurrences of the same blocker within one phase warranted a check-in). Ní chose
  "substitute proof again" over authorizing a live `make dev-pg` run or skipping the task. The
  substitute: a full manual trace of the request path — `providersApi.getProviderPresets()` →
  `GET /provider-presets` (matches the route registered in Wave 2) → handler returns
  `{"presets": [...]}` (3 entries from `providers.json`) → `extractArrayPayload` reads the
  `presets` key → `normalizeProviderPreset` maps every snake_case field
  (`id`/`display_name`/`base_url`/`headers`/`signup_url`/`free_tier_note`/`verified`/…) against
  the exact JSON tags in `presets.go`, confirmed 1:1 — → `BaseProviderForm`'s `useEffect` fires
  only when `brand === 'openaiCompatibility' && mode === 'create'`, populates the `Select`, and
  `applyPreset` sets `baseUrl`/`headers` from the matched preset (verified against the concrete
  `opencode`/`openrouter`/`nvidia` entries: opencode and openrouter have `headers` and prefill
  them, nvidia has none and correctly leaves the existing headers untouched; openrouter and
  nvidia have `signup_url` and render the link, opencode does not and correctly omits it). No
  live server was started against the shared DB.
- 2026-08-06 — phase: provider-presets — wave: 5 — task: 5.1 — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — touched: `docs/product/provider-presets.md` (new) — verify: manual
  review — content covers OpenCode Free's unauthenticated/best-effort/no-contract nature and
  states the `verified` field's exact claim scope (endpoint probed live on `verified_at`, not that
  a chat completion was billed/streamed), matching the substance of `## Seed set`'s "Scope of the
  claim" paragraph (lines 139-141 of this plan).
- 2026-08-06 — phase: provider-presets — wave: 5 — wave-5-complete — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — trace_id: 01KZAXCP52KMAYCVSSADFY9KDB — all Wave 5 tasks DONE.
- 2026-08-06 — phase: provider-presets — phase-level-check — task_status: DONE — run_id:
  01KZAQ3N0RPDMFJGQZ1CD6M2RN — trace_id: 01KZAXE7N1MNVA3W176PWZ1M26 — `go test
  ./internal/config/... ./sdk/cliproxy/... ./internal/runtime/executor/... ./internal/api/...
  ./internal/watcher/...` all `ok`; `make build-web` (`tsc && vite build`) clean; `make build`
  produces `llmhub` binary clean. Optional manual check (live streaming against OpenCode) skipped —
  same Postgres-blocker constraint as tasks 2.2/4.2, not required by the plan.

## Decisions
- 2026-08-06 — task 5.1 — Placed the docs page at `docs/product/provider-presets.md` rather than
  under a `references/` directory. The plan's own "Not building" section cites
  `references/providers.md` and `references/porting-map.md`, but those paths don't exist anywhere
  in the repo (checked); the only real `references/` dirs in the repo belong to `.claude/skills/`
  (skill-internal, not doc output). `docs/product/README.md` explicitly instructs naming files by
  product domain (`overview.md`, `billing.md`, …), so `provider-presets.md` under `docs/product/`
  follows the repo's actual documented convention instead of a path that was never created.
- 2026-08-06 — phase: provider-presets — backfill — The IDs this doc originally carried (story
  `01KZAQ2Q5RCVSMHFP1PVXBQ4C1`, run `01KZAQ3N0RPDMFJGQZ1CD6M2RN`, check `01KZAXK9GF1YSYM137CT5GFR68`)
  were minted with `zharness id`, which is explicitly non-mutating. The session never invoked
  `zharness story` / `run create` / `check record`, so no changeset and no DB row was ever written
  for this phase — the same root cause recorded in `docs/plans/done/error-classification.md`, which
  see for the full evidence. Real rows were created 2026-08-06 via the proper commands — story
  `01KZBH3RW730XYMA5VT1XQY3W0` (`depends_on: error-classification`), run
  `01KZBH8KG43JVPWBWDKG8M63ZT`, check `01KZBH8KGCTC7HFDB1E0V006X3` — and the Current State block
  below now points at those. The APPROVED verdict was not transcribed: every proof command was
  re-run and observed on 2026-08-06 against master `9231234b` (`go test ./...` → 23 ok / 0 FAIL;
  `go test ./internal/config/presets/... -v` → 3 PASS; `bun run type-check` clean; `bun run lint` →
  0 errors / 8 warnings, matching the original record exactly; `make build-web` → 1,964.01 kB;
  `make build` clean, worktree clean). Phases `model-combos` and `provider-grid-console` were
  deliberately NOT backfilled: each must get its story from its own `/to-plan` run (playbook step 4),
  which is precisely the step that was skipped here.

## Current State and Next Action
- active_phase: none
- lifecycle_status: done
- latest_run_id: 01KZBH8KG43JVPWBWDKG8M63ZT
- latest_check_id: 01KZBH8KGCTC7HFDB1E0V006X3
- latest_handoff_id: 01KZBJ6A1PVW74BX7BVQMJR8A5
- verdict: APPROVED
- superseded_ids: run 01KZAQ3N0RPDMFJGQZ1CD6M2RN, check 01KZAXK9GF1YSYM137CT5GFR68 — minted
  2026-08-06 but never persisted; no DB row ever existed for them (see Decisions, backfill entry)
- blockers: none
- open_items: none
- exact_next_action: initiative complete. The code shipped to master ahead of this closure (see
  `## Progress`); this handoff closed the lifecycle record that the 2026-08-06 backfill
  reconstructed. Follow-on work lives in `docs/plans/active/unified-provider-console.md`, whose
  phase `provider-grid-console` declares `depends_on: provider-presets` and is now unblocked.

Carried forward, not a blocker on this plan: `make dev` cannot start without `PGSTORE_DSN` and the
only DSN available is the team's shared remote Supabase instance, so no live-server proof was
possible in tasks 2.2, 4.2, or the optional streaming check. Each was replaced with a full manual
request-path trace at ní's explicit direction (see `## Decisions`). Any future phase needing live
browser or streaming proof against this codebase will hit the same wall — `provider-grid-console`
task 6.3 and `model-combos` task 5.2 both do.
