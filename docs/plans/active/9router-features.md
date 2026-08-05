# Plan: port 9router upstream-provider + routing features into LLMHub

Status: **awaiting approval** · Created 2026-08-05 · Source: `decolua/9router` @ 2026-08-05
Reference skill: `.claude/skills/9router-port/`

## Building

Close the one real capability gap between LLMHub and 9router — **fallback across models and
providers, not just across credentials of one provider** — plus the two cheapest provider-inventory
wins (a preset catalog, and OpenCode Free as a genuinely no-auth upstream).

## Not building

| Excluded | Why |
|---|---|
| RTK / Headroom / Caveman token compression | Out of the agreed scope |
| Format-translation rework | LLMHub's translator registry already covers it |
| Declarative provider registry in Go | LLMHub's `openai-compatibility` already is this; see `references/porting-map.md` |
| Account round-robin, per-model cooldown, backoff | Already exists and is more thorough (`sdk/cliproxy/auth/`) |
| GitHub Copilot, Cursor, Windsurf, Zed, Trae, Qoder upstreams | ToS/ban risk + pinned-editor-version maintenance treadmill. 9router itself marks `github` as `deprecated` + `RISK_NOTICE` |
| MiMo Free | Upstream shut the free channel down; 9router has it `hidden: true` |
| MITM proxy, Tailscale/Cloudflare tunnel, cloud sync | Not asked for |

## Load-bearing assumption (premise collapse)

> **This plan assumes a combo can retry a failed candidate only while nothing has been flushed to
> the client, and that LLMHub already detects that condition.**

Evidence it holds: `Manager.ExecuteStream` accepts `providers []string` and already distinguishes
`streamBootstrapError` — a failure raised before the first downstream byte
(`sdk/cliproxy/auth/conductor.go:1673,1705`).

If it does **not** hold — i.e. `streamBootstrapError` is raised in cases where bytes already
reached the client — then Phase 3 must degrade to non-stream-only fallback, and streaming clients
get an in-stream error event instead of a silent switch. Phase 3's first task is a test that pins
this behavior, so the failure is caught before any routing code is written.

## Phases

Each phase is independently mergeable: after it ships, the system is usable even if no later phase
lands.

---

### Phase 1 — Provider preset catalog + model passthrough

**Value alone:** a user can add OpenRouter / Groq / DeepSeek / GLM / MiniMax / Cerebras from the
panel in two clicks instead of hand-writing YAML. No routing changes.

**Changes**

1. `internal/config/presets/providers.json` (new, embedded via `go:embed`) — one entry per known
   OpenAI-compatible provider: `id`, `display_name`, `base_url`, `headers`, `models_url`,
   `signup_url`, `free_tier_note`, `passthrough`. Seed set (14): openrouter, groq, cerebras,
   deepseek, together, fireworks, nvidia, siliconflow, mistral, glm, minimax, chutes, hyperbolic,
   llm7. Values taken from `open-sse/providers/registry/{id}.js`.
2. `internal/config/presets/presets.go` (new) — `Load() []Preset`, parsed once at init.
3. `internal/api/handlers/management/` — add `GET /v0/management/provider-presets` returning the
   catalog. Read-only, no secrets.
4. `internal/config/config.go` — add `Passthrough bool \`yaml:"passthrough,omitempty"\`` to
   `OpenAICompatibility`.
5. `internal/runtime/executor/openai_compat_executor.go` + the registry path that expands
   `OpenAICompatibility.Models` — when `Passthrough` is true, forward `req.Model` verbatim
   (stripping the configured `Prefix` if present) instead of requiring an alias entry, and skip
   contributing per-model entries to the visible model list beyond those explicitly configured.
6. `web/` — on the provider-add screen, a preset dropdown that prefills base-url/headers. No new
   test files under `web/` (project rule).

**Verify**
- `go test ./internal/config/... ./internal/runtime/executor/...`
- New test: `openai_compat_executor` with `Passthrough: true` forwards an unlisted model id.
- New test: every preset in `providers.json` has non-empty `id` and `base_url`, and ids are unique.
- `curl -s localhost:9090/v0/management/provider-presets` returns 14 entries.
- `make build-web && make build` — panel loads, preset dropdown prefills base-url.

**Rollback:** delete the endpoint + preset files; `Passthrough` defaults false, so existing config
is unaffected.

---

### Phase 2 — OpenCode Free upstream

**Value alone:** a working zero-credential free provider. Nothing else in LLMHub needs it.

**Mechanism** (`open-sse/providers/registry/opencode.js`, `open-sse/executors/opencode.js`):

```
POST https://opencode.ai/zen/v1/chat/completions     (OpenAI format)
GET  https://opencode.ai/zen/v1/models
Authorization: Bearer public
x-opencode-client: desktop
Accept: text/event-stream
```

**Changes**

1. Add an `opencode` preset to `internal/config/presets/providers.json` with
   `passthrough: true`, `headers: {"x-opencode-client": "desktop"}`, and a documented literal
   api-key of `public`.
2. `internal/runtime/executor/openai_compat_executor.go` — no change needed if the static bearer
   works through the existing `api-key-entries` path. If it does not (e.g. the executor rejects a
   non-secret key or omits `Accept: text/event-stream`), add the header via the existing
   `Headers` map rather than special-casing the provider.
3. `docs/` — one short page: what OpenCode Free is, that it is unauthenticated and therefore
   best-effort, and that it may disappear without notice.

**Verify**
- `curl -sS https://opencode.ai/zen/v1/models -H "Authorization: Bearer public" -H "x-opencode-client: desktop"` returns a model list (run this first — if upstream is gone, stop and report).
- Live smoke: a streaming `POST /v1/chat/completions` through LLMHub against an opencode model
  returns SSE deltas.
- `go test ./internal/config/...`

**Rollback:** remove the preset. Zero code surface if step 2 turns out to be a no-op.

**Risk:** this is an undocumented public endpoint. Treat breakage as expected, not as a regression.

---

### Phase 3 — Model combos (cross-provider fallback)

**Value alone:** the headline capability. A client asks for one model name and never sees a rate
limit as long as any candidate in the chain is alive.

**Config** (new block, stored in Postgres like the rest — no local file):

```yaml
combos:
  - name: daily
    strategy: fallback          # fallback | round-robin
    sticky-limit: 8             # round-robin only; requests before advancing
    models:
      - claude/claude-opus-5
      - openrouter/deepseek-v4:free
      - opencode/grok-code
```

**Changes**

1. `internal/config/config.go` — `Combos []ComboConfig` with `Name`, `Strategy`, `StickyLimit`,
   `Models []string`. Validation: unique non-empty name, name must not collide with a registered
   model id, at least one candidate, each candidate must parse as `provider/model`.
2. `internal/registry/` — register each combo name as a client-visible model so `/v1/models`
   lists it and clients can select it.
3. `sdk/cliproxy/combo.go` (new) — the resolution + rotation layer, ported from
   `open-sse/services/combo.js`:
   - `Resolve(name) ([]Candidate, bool)`
   - `Rotate(name, strategy, stickyLimit) []Candidate` — in-memory cursor per combo, mutex-guarded,
     cleared on config reload (hook into `internal/watcher/config_reload.go`).
4. Routing: in the request path that currently maps a model name to `providers []string` before
   calling `Manager.Execute`/`ExecuteStream`, add a combo branch that loops candidates. Advance to
   the next candidate **only** when the error is a `streamBootstrapError` (stream) or any
   retryable classification (non-stream). Otherwise return the error unmasked.
5. Error surfacing, matching `combo.js:329-347`: all candidates exhausted → HTTP **503** (not 406),
   body carries the earliest `retry-after` across candidates plus a human `reset after 2m 30s`
   string. Reuse `sdk/cliproxy/auth/cooldown_state.go` for the reset timestamps.
6. Transient handling, matching `combo.js:311`: on `502/503/504` with a computed cooldown ≤ 5s,
   sleep that cooldown before trying the next candidate.
7. Cross-check `sdk/cliproxy/auth/errors.go` against `open-sse/config/errorConfig.js:59` — adopt
   **text rules before status rules** if LLMHub currently classifies on status alone.
8. `web/` — a Combos page: list, create, reorder candidates, pick strategy.

**Verify**
- **First**, before any routing code: a test pinning that `streamBootstrapError` is only returned
  when nothing has been written downstream. If it fails, stop and re-scope per the premise-collapse
  note above.
- `go test ./sdk/cliproxy/... ./internal/config/... ./internal/registry/...`
- New tests: candidate 1 returns 429 → candidate 2 serves; candidate 1 returns 400 → 400 is
  returned unmasked, no fallback; all candidates fail → 503 with earliest retry-after;
  round-robin with `sticky-limit: 3` advances on the 4th request; combo name colliding with a real
  model id is rejected at config load.
- Live: define a 2-candidate combo, revoke the first credential, confirm requests still succeed and
  the log shows the switch.

**Rollback:** remove the `combos:` block. Absent config, the routing branch is never entered.

---

### Phase 4 — Capability-aware candidate reordering

**Value alone:** a combo stops silently dropping images/PDFs when the leading candidate is
text-only. Requires Phase 3; Phase 3 is fully useful without it.

**Changes**

1. `internal/registry/model_definitions.go` + `models/models.json` — per-model capability flags:
   `vision`, `pdf`, `audio_input`, `video_input`, plus `context_window` where known.
2. `sdk/cliproxy/capability.go` (new), ported from `combo.js:63,105`:
   - `DetectRequired(body)` → set of required capabilities. Scan **only the trailing user turn**
     (`combo.js:94` — history media must not pin the whole conversation). Handle OpenAI
     `messages`, Responses `input`, Gemini `contents`, Claude content blocks, and mime sniffing on
     `data:` URIs and `source.media_type`.
   - `Reorder(candidates, required)` → stable 3-tier sort (all-caps / hard-caps-only / rest).
     Never drops a candidate — the fallback chain stays intact.
3. Wire `Reorder` into the Phase 3 loop, after rotation, before the first attempt.

**Verify**
- `go test ./sdk/cliproxy/...`
- New tests: image in the current turn floats a vision candidate to the front; image five turns
  back does **not**; PDF and audio route independently; a combo where no candidate has vision
  keeps its original order and still tries every candidate.

**Rollback:** stop calling `Reorder`; the flags in `models.json` are inert data.

---

### Phase 5 — Fusion combos (optional)

**Value alone:** a quality mode — fan out to a panel, one judge synthesizes. Self-contained; skip
without cost if priorities shift.

**Changes** (ported from `combo.js:513`)

1. `strategy: fusion` + `judge-model:` on `ComboConfig`.
2. Panel calls forced non-streaming, tools stripped, tool history flattened to prose
   (`combo.js:21`) so panel models cannot start a tool loop.
3. Quorum-grace collection (`combo.js:461`): proceed once 2 answers land + an 8s straggler grace;
   hard cap 90s. Implement with a `context.WithTimeout` per call plus a grace timer.
4. Judge prompt from `combo.js:417` — anonymized `Source N`, analyze consensus / contradictions /
   partial coverage / unique insights / blind spots, then write one answer. Judge call keeps the
   client's original `stream` flag and tools.
5. Degrade: 0 answers → 503; exactly 1 → return it directly without fusion.

**Verify**
- `go test ./sdk/cliproxy/...`
- New tests: 3 panel models, 1 hangs → judge still runs after grace; 1 succeeds → returned
  directly; 0 succeed → 503; panel request body has no `tools` and `stream: false`.

**Rollback:** remove the `fusion` strategy value; config validation rejects it, existing combos
unaffected.

---

## Recommended cut

Ship **1 → 2 → 3**. Phase 3 is the point of the exercise; 1 and 2 are cheap and de-risk it by
giving Phase 3 real free candidates to fall back onto. Phase 4 after that. Phase 5 only if the
quality mode is actually wanted.

## Credentials and external dependencies

| Needed | For | Notes |
|---|---|---|
| OpenRouter API key | Phase 1 live check, Phase 3 fallback candidate | Free tier, no card |
| *(none)* | Phase 2 | OpenCode Free is unauthenticated |
| An existing LLMHub credential for a subscription provider | Phase 3 live check | Already configured |
| Postgres (`PGSTORE_DSN`) | all phases | Already required by LLMHub |

No new MCP servers, third-party CLIs, or paid accounts.

## Project-rule check

- **Postgres-only config** (`CLAUDE.md`): the new `combos:` and `passthrough:` fields live in the
  Postgres-stored config YAML and round-trip through `/v0/management/config.yaml`. No new local
  file. ✅
- **No frontend test files under `web/`** (`CLAUDE.md`): Phase 1 and Phase 3 panel work is verified
  by type check, lint, production build, and a browser runtime check only. ✅
- **Minimal change** (`rules/karpathy-guidelines.md`): no phase touches the executor or translator
  interfaces; the combo layer sits above `Manager.ExecuteStream` rather than inside it. ✅
