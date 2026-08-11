# 9router feature port — index

Umbrella doc. Shared context and exclusions live here; each feature has its own plan file.
Reference skill: `.claude/skills/9router-port/` · Source: `decolua/9router` @ 2026-08-05

## The gap, in one line

LLMHub falls back across **credentials of one provider**. 9router additionally falls back across
**models of different providers**. Everything else is either already covered by LLMHub or is
provider inventory.

## Plans

| Plan | File | Effort | Ships alone | Status |
|---|---|---|---|---|
| **A** — Unified error classification | [`error-classification.md`](../done/error-classification.md) | S | yes | done |
| **B+C** — Preset catalog (opencode, openrouter, nvidia), passthrough | [`provider-presets.md`](../done/provider-presets.md) | S | yes | done (shipped to master) |
| **D** — Model combos (cross-provider fallback) | [`model-combos.md`](../done/model-combos.md) | L | yes | done (merged to master, 2026-08-11) |
| **F** — Unified provider console (category grid + sheet) | [`unified-provider-console.md`](../done/unified-provider-console.md) | M | yes | done (merged to master, 2026-08-11) |

Recommended order: **A → B+C → D**. **F** borrows only 9router's category taxonomy and card-grid
presentation; it depends on B+C's preset catalog but not on D, and can run in parallel with D —
they share no surface.

A first because it is the smallest and fixes a standalone bug — `shouldRetryAfterError` returns
false for every non-429 status (`sdk/cliproxy/auth/conductor.go:2621`), so a `500` carrying
`"rate limit exceeded"` currently gets no retry and no cooldown. D also needs A's classifier to
know when advancing to the next candidate is warranted.

B+C before D so D has real free candidates to fall back onto.

## Deferred, not planned

| Feature | Why deferred |
|---|---|
| **E** — Capability-aware reordering (`combo.js:63,105`) | Needs D first. Also needs new capability flags in `internal/registry/models/models.json`, which today carries only `context_length` (87 entries, no vision/pdf/audio). |
| **F** — Fusion panel + judge (`combo.js:513`) | Self-contained and low risk, but lowest value of the set. |

## Not building at all

| Excluded | Why |
|---|---|
| RTK / Headroom / Caveman token compression | Out of the agreed analysis scope |
| GLM / MiniMax as `openai-compatibility` | They are Anthropic-messages format (`x-api-key`, `/anthropic/v1/messages`). Possible later as `claude-api-key` entries; see `provider-presets.md` §Not building |
| Format-translation rework | LLMHub's translator registry already covers it |
| Declarative provider registry in Go | `openai-compatibility` already is this mechanism; see `references/porting-map.md` |
| Account round-robin, per-model cooldown, backoff, weighted credentials | Already exist and are more thorough than 9router's (`sdk/cliproxy/auth/`) |
| GitHub Copilot, Cursor, Windsurf, Zed, Trae, Qoder upstreams | ToS/ban risk plus pinned-editor-version maintenance. 9router marks `github` itself `deprecated` + `RISK_NOTICE` |
| MiMo Free | Upstream ended the free channel; 9router has it `hidden: true` |
| MITM proxy, Tailscale/Cloudflare tunnel, cloud sync | Not asked for |

## Constraints binding every plan here

- **Postgres-only config** (`CLAUDE.md`): new config blocks live in the Postgres-stored YAML and
  round-trip through `/v0/management/config.yaml`. No new working-directory file.
- **No frontend test files under `web/`** (`CLAUDE.md`): panel work is verified by type check,
  lint, production build, and a browser runtime check.
- **Streaming boundary**: cross-provider fallback is only legal before the first byte reaches the
  client (`conductor.go:1705` `streamBootstrapError`). Any design that retries after a flush is
  wrong.
