# Plan D — Model combos (cross-provider fallback)

Status: **awaiting approval** · Created 2026-08-05
Depends on: **Plan A** (`error-classification.md`) — soft dependency, see below
Source mechanism: `decolua/9router` `open-sse/services/combo.js:246` (`handleComboChat`)
Reference skill: `.claude/skills/9router-port/references/routing.md` §1

## Building

A **combo**: a named virtual model that expands to an ordered list of `provider/model` candidates
spanning different providers. The client asks for one name; the router walks the chain.

```yaml
combos:
  - name: daily
    strategy: fallback            # fallback | round-robin
    sticky-limit: 8               # round-robin only
    models:
      - claude/claude-opus-5
      - openrouter/deepseek-v4:free
      - opencode/grok-code
```

This is the one real capability gap. LLMHub falls back across **credentials of one provider**
(`sdk/cliproxy/auth/selector.go`, `scheduler.go`); it has nothing that falls back across
**models of different providers**.

## Not building

- No change to credential selection, cooldown persistence, or backoff inside a single provider —
  that layer stays exactly as-is and runs unmodified within each candidate attempt.
- No capability-aware reordering (that is Plan E, deliberately separate).
- No fusion / panel+judge.
- No new executor or translator. The combo layer sits **above** `Manager.ExecuteStream`.

## Load-bearing assumption

> **A failed candidate can only be replaced while nothing has been flushed to the client.**

Evidence it holds: `Manager.ExecuteStream` already takes `providers []string` and already
distinguishes `streamBootstrapError` — a failure raised before the first downstream byte
(`sdk/cliproxy/auth/conductor.go:1673`, `:1705`).

**If it does not hold** — if `streamBootstrapError` can be returned after bytes reached the client
— then cross-provider fallback is only safe for non-streaming requests, and streaming clients must
get an in-stream error event instead of a silent switch. That halves the feature's value and
changes the design.

**Therefore task 1 is the test that pins this, written before any routing code.** If it fails,
stop and re-scope rather than building on top of it.

## Soft dependency on Plan A

Combo needs to answer "is this failure worth advancing for?". Without Plan A the only signal is
`shouldRetryAfterError`, which returns false for every non-429 status
(`conductor.go:2621`) — so a candidate failing with `500 "rate limit exceeded"` would end the whole
chain instead of advancing. Combo *works* without A, it just advances far less often than it
should. Ship A first.

## Changes

### 1. Test first — pin the streaming boundary

`sdk/cliproxy/auth/conductor_stream_bootstrap_test.go` (new): assert `streamBootstrapError` is
returned only when zero chunks have been written downstream, and that a mid-stream upstream failure
produces a different error type. **Gate: if this cannot be made to pass, stop.**

### 2. Config

`internal/config/config.go`:

```go
type ComboConfig struct {
    Name        string   `yaml:"name" json:"name"`
    Strategy    string   `yaml:"strategy,omitempty" json:"strategy,omitempty"`       // fallback | round-robin
    StickyLimit int      `yaml:"sticky-limit,omitempty" json:"sticky-limit,omitempty"`
    Models      []string `yaml:"models" json:"models"`
}
```

Added to `Config` as `Combos []ComboConfig`. Validation at load (`internal/config/parse.go`):
name non-empty and unique; name must not collide with any registered model id; at least one
candidate; each candidate parses as `provider/model`; `strategy` in `{"", "fallback",
"round-robin"}` (empty = `fallback`); `sticky-limit < 1` normalized to `1`.

Stored in Postgres like the rest of the config and round-tripped through
`/v0/management/config.yaml` — no new local file (project `CLAUDE.md`).

### 3. Registry

`internal/registry/` — register each combo name as a client-visible model so `/v1/models` lists it
and clients can select it. Marked so it is never itself resolvable as a candidate (no recursion).

### 4. Resolution and rotation

`sdk/cliproxy/combo.go` (new), ported from `open-sse/services/combo.js:157-212`:

```go
type Candidate struct{ Provider, Model string }

func (r *Resolver) Resolve(name string) ([]Candidate, bool)
func (r *Resolver) Rotate(name, strategy string, stickyLimit int) []Candidate
```

- `fallback` — always start at index 0.
- `round-robin` — per-combo cursor plus a **sticky limit**: stay on the same candidate for N
  consecutive requests before advancing (`combo.js:174`). Sticky matters because rotating every
  single request destroys upstream prompt caching.
- Cursor state is an in-memory `map[string]state` guarded by a mutex, cleared on config reload via
  `internal/watcher/config_reload.go`.

### 5. The loop

In the request path that maps a model name to `providers []string` before calling
`Manager.Execute` / `Manager.ExecuteStream`, add a combo branch:

```
for each candidate:
    result := Manager.Execute[Stream](ctx, []string{cand.Provider}, req{Model: cand.Model}, opts)
    if err == nil                       → return
    if streaming && !streamBootstrapErr → return err   (bytes already flushed; cannot switch)
    if !retryable(err)                  → return err   (400-class: surface unmasked)
    if status in {502,503,504} && cooldown <= 5s → sleep(cooldown)
    continue
```

Two details ported verbatim in intent:

- **Transient pause** (`combo.js:311`): on `502/503/504` with a computed cooldown ≤ 5s, sleep it
  before the next candidate. A briefly-overloaded provider deserves a beat, not an instant skip.
- **Do not mask client errors** (`combo.js:303`): if the failure is not retryable, return it as-is.
  A `400` must not become a 503 after walking the whole chain.

Existing credential-level retry, cooldown, and backoff run **inside** each iteration, untouched.

### 6. Exhaustion response

Matching `combo.js:329-347`: all candidates failed → HTTP **503**, not 406. 406 implies the request
is invalid; here the providers are merely unavailable and the client should retry. Body carries the
**earliest** reset across all candidates (read from `sdk/cliproxy/auth/cooldown_state.go`) plus a
human string, e.g. `reset after 2m 30s` (`accountFallback.js:90` `formatRetryAfter`).

### 7. Panel

`web/` — a Combos page: list, create, reorder candidates, pick strategy and sticky limit. No new
test files under `web/` (project rule).

## Verify

- **Gate test from step 1 passes.** Nothing else starts until it does.
- `go test ./sdk/cliproxy/... ./internal/config/... ./internal/registry/... ./internal/watcher/...`
- New tests:
  - candidate 1 → `429`, candidate 2 → `200`: client gets candidate 2's response
  - candidate 1 → `400`: returned unmasked, candidate 2 never attempted
  - all candidates fail: `503` carrying the earliest reset and the human string
  - streaming, candidate 1 fails at bootstrap: switches silently
  - streaming, candidate 1 fails after the first chunk: does **not** switch, error reaches the
    client in-stream
  - `round-robin` with `sticky-limit: 3` advances the cursor on the 4th request, not the 2nd
  - `502` with a 3s cooldown pauses before the next candidate; `502` with a 40s cooldown does not
  - combo name colliding with a registered model id is rejected at config load
  - config reload clears the rotation cursor
- Live: define a 2-candidate combo, revoke the first provider's credential, confirm requests keep
  succeeding and the log shows the switch.

## Rollback

Remove the `combos:` block. With no combos configured the routing branch is never entered and the
request path is byte-identical to today. The registry entries disappear with the config.

## Risk

- The gate test is the real risk; everything downstream assumes it passes.
- Rotation state is per-process and in-memory. In a multi-replica deployment each replica rotates
  independently. Acceptable — 9router has the same property (`combo.js:88`) — but it must be
  documented, not discovered.
