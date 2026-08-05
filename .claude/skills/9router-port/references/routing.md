# 9router routing & fallback engine — mechanisms worth porting

Source files cited relative to the 9router repo root.

## 1. Combo — cross-provider model fallback (THE gap)

`open-sse/services/combo.js:246` `handleComboChat()`

A **combo** is a named virtual model. The client asks for `"my-combo"`; the router expands it to an
ordered list of `provider/model` strings and tries them in order.

```
combo "daily" = [ "claude/claude-opus-5", "oc/grok-code", "openrouter/deepseek-v4:free" ]
                  ^ subscription           ^ free           ^ free
```

Loop semantics (`combo.js:266-327`):

1. Try candidate `i`. `result.ok` → return immediately.
2. Not ok → parse the error body, extract `error.message` and `retryAfter`.
3. `checkFallbackError(status, errorText)` decides `{shouldFallback, cooldownMs}`.
4. `shouldFallback === false` → return the error to the client as-is (do not mask a 400).
5. Transient `502/503/504` with `cooldownMs <= 5000` → **sleep that cooldown, then move on**.
   Rationale in-source: a briefly-overloaded provider deserves a beat, not instant skip.
6. All candidates exhausted → `503` (explicitly *not* 406 — 406 implies the request is invalid,
   but here the providers are merely unavailable), carrying the **earliest** `retryAfter` across
   all candidates plus a human string like `reset after 2m 30s`.

### Strategies

`getRotatedModels(models, comboName, strategy, stickyLimit)` — `combo.js:174`

- `fallback` — always start at index 0. Preferred model gets used until it breaks.
- `round-robin` — rotate the start index per combo, with a **sticky limit**: stay on the same
  candidate for N consecutive requests before advancing. State is in-memory `Map` keyed by combo
  name (`combo.js:88`), reset on config change.

Sticky limit matters for prompt caching — rotating every single request destroys the upstream
cache. LLMHub has the analogous concern in session affinity (`RoutingConfig.SessionAffinity`).

### LLMHub insertion point

`Manager.ExecuteStream(ctx, providers []string, req, opts)` — `sdk/cliproxy/auth/conductor.go:1673`
— already accepts a provider list and already distinguishes `streamBootstrapError`
(`conductor.go:1705`), i.e. a failure that happened **before** any byte was flushed downstream.

That is the whole safety condition for combo fallback:

> Falling forward to the next candidate is only legal while the response has not started
> streaming. After the first flushed chunk, the only honest option is to end the stream with an
> error event.

A combo layer therefore sits *above* `ExecuteStream`, not inside it: resolve combo name →
`[]{provider, model}` → loop → per-iteration call the existing manager → advance only on
`streamBootstrapError`. Existing credential-level retry, cooldown, and backoff stay untouched
inside each iteration.

## 2. Error classification — config-driven, not switch-statement

`open-sse/config/errorConfig.js:59` `ERROR_RULES`

Rules are evaluated **top-to-bottom, text rules before status rules**:

```
text "no credentials"            → cooldown 2m
text "request not allowed"       → cooldown 5s
text "improperly formed request" → cooldown 2m
text "rate limit" | "too many requests" | "quota exceeded" | "capacity" | "overloaded"
                                 → exponential backoff
status 401 | 402 | 403 | 404     → cooldown 2m
status 429                       → exponential backoff
(unmatched)                      → cooldown 30s
```

Backoff: `base 2s × 2^(level-1)`, capped at `5m`, `maxLevel 15` (`errorConfig.js:32`).
Separate hard cap `MAX_RATE_LIMIT_COOLDOWN_MS = 30m` for provider-reported resets — Codex can
report `resets_at` 5-6 hours out, and honoring that verbatim would park an account for the day.

**Text-before-status is the transferable insight.** Upstreams return `200`-with-error-body and
`500`-that-is-really-a-quota-error often enough that status alone misclassifies. LLMHub's
`sdk/cliproxy/auth/errors.go` should be checked against this rule ordering.

## 3. Capability auto-switch

`open-sse/services/combo.js:105` `detectRequiredCapabilities(body)`
`open-sse/services/combo.js:63` `reorderByCapabilities(models, required)`

Scans the request for required input modalities and floats capable candidates to the front.

- Capabilities: `vision`, `pdf`, `audioInput`, `videoInput` (hard) vs `search` (soft).
  Hard = missing it silently *drops request data*. Soft = only degrades a feature.
- **Only the trailing user turn is scanned** (`trailingUserItems`, `combo.js:94`). An image from
  five turns ago must not pin the whole conversation to a vision model.
- Detection spans four request shapes: OpenAI `messages`, Responses `input`, Gemini `contents`,
  and Claude content blocks — plus mime sniffing on `data:` URIs and `source.media_type`.
- Reordering is a **stable 3-tier sort** and never drops a candidate: tier 0 = all hard + all soft,
  tier 1 = all hard, tier 2 = rest. Fallback chain stays intact.

## 4. Capacity adapter — global per-capability rescue pool

`open-sse/services/capacityAdapter.js`

If *none* of the combo's candidates satisfy a hard capability, prepend models from a global pool
configured per capability (`settings.capacityAdapter.vision.models`, …). Two extras worth noting:

- `stripHistoryForContext(body, contextWindow)` (`capacityAdapter.js:117`) — when rescuing onto a
  smaller-context model, drop the **middle** of the conversation: keep all system/developer
  messages, keep the first 6 turns, keep the trailing user turn (the one carrying the media).
  Budget = `contextWindow × 0.8 × 4 chars/token`.
- `withCapacityAdapterStripping` wraps the handler so stripping only applies to pool models.

## 5. Fusion — panel + judge

`open-sse/services/combo.js:513` `handleFusionChat()`

Fan the prompt to every panel model in parallel, then one judge model synthesizes a single answer.

- Panel calls forced `stream: false`, **tools stripped**, tool history flattened to prose
  (`flattenToolHistory`, `combo.js:21`) so panel models cannot start a tool loop.
- **Quorum-grace collection** (`collectPanel`, `combo.js:461`): once `minPanel` (default 2)
  answers land, start an 8s grace timer for stragglers, then proceed. Hard cap 90s. This is the
  fix for "the slowest model dominates wall time".
- Judge prompt (`buildJudgePrompt`, `combo.js:417`) anonymizes sources as `Source N` so the judge
  weighs substance over brand, and instructs analysis along consensus / contradictions / partial
  coverage / unique insights / blind spots **before** writing.
- Degrades: 0 answers → 503; exactly 1 → return it directly, no fusion.
- Judge call keeps the client's original `stream` flag and tools, so downstream tool use survives.

## 6. Account fallback — already covered by LLMHub

`open-sse/services/accountFallback.js`. Per-account `rateLimitedUntil`, `backoffLevel`, reset on
success, `filterAvailableAccounts`, and per-model locks via flat `modelLock_${model}` fields.

LLMHub equivalents already exist and are more thorough (persisted across restarts):
`sdk/cliproxy/auth/cooldown_state.go`, `cooldown_backoff_test.go`, `credential_weight.go`,
`selector.go` (`RoundRobinSelector`, `FillFirstSelector`), `scheduler.go`.

**Do not port this.** The only detail worth cross-checking is `formatRetryAfter` — surfacing a
human "reset after 2m 30s" to the client in the error body is a small UX win.
