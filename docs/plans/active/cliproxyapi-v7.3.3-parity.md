---
id: plan-20260915-v733
type: plan
intake_id: intake-20260915-v733
lane: high-risk
status: active
created: 2026-09-15
updated: 2026-09-15
---

# Plan: CLIProxyAPI v7.3.3 targeted parity

## Outcome
- result: llmhub ports the owner-approved v7.2.147..v7.3.3 capability slices as independently verifiable semantic ports behind existing translator, executor, auth, registry, and SDK interfaces — including the new Devin/Cognition provider and LAN gateway discovery — without wholesale merge, pluginhost, Home/gitstore work, or branding churn.
- success_signals:
  - Each accepted include slice lands with focused tests citing its upstream commit(s) and local symbol.
  - Devin provider authenticates via OAuth (browser + port-free manual code), executes through the local `Executor` interface, and resolves `devin/`-prefixed models from a catalog.
  - LAN discovery advertises and browses local gateways behind existing config/cmd wiring.
  - Postgres remains the authoritative runtime store; no new file/YAML source of truth.
  - `go test` on touched packages, `make build`, and `git diff --check` pass at gates.
  - Newer-than-v7.3.3 upstream delta is pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.3` at `7bbfeaf8a7ac` / `refs/upstream-checkpoints/cliproxyapi/v7.3.3`; `scope_policy` strategy `targeted-semantic-ports` (26 include, 6 exclude, 6 defer) is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.2.147..v7.3.3.json` — 538 paths (match 3, baseline 8, upstream-add-absent 129, diverged-absent 188, upstream-delete-present-local 0, semantic-review 210).
  - `docs/upstream/cliproxyapi-ledger-v7.2.147..v7.3.3.md` — 247 non-merge commits across 16 releases.
  - `CLAUDE.md` / `docs/PROJECT.md` — Postgres-authoritative store, llmhub management UI, SDK compatibility, branding, pluginhost non-goal.
  - `docs/plans/completed/cliproxyapi-v7.2.147-parity.md` — prior cycle; do not regress R1–R20 slices.
- rejected_alternatives:
  - Wholesale file-for-file merge of 538 paths — local divergence (210 semantic-review, 188 diverged-absent) makes it unsafe.
  - Translators-only slice — owner approved all core slices plus devin and discovery.
  - Copying the upstream Devin executor file-for-file — lands behind local `cliproxyexecutor.Executor`, `internal/auth/<provider>`, and registry interfaces instead.
- requirements:
  - R1 [accepted]: Claude translator hardening — unsigned/prefixed thinking-signature handling, cache write tokens in responses, deferred message_delta and streaming usage, tool-call pairing across interrupted streams and standalone tool outputs, strict-mode downgrade on missing required properties, max_tokens terminal state, orphan function outputs as user text, trailing usage chunk. | source: `8deeb4ac3159` `15231e9fdc93` `893abbabc2a5` `f804fb5f3077` `2bcebaa89c98` `aa3652775225` `ba2cdea3b919` `8c984672a66a`
  - R2 [accepted]: Gemini/Antigravity translator normalization — functionResponse user-role nesting and image parts, responseJsonSchema → responseSchema, empty/null finish reasons, tool-name collision avoidance vs Antigravity intrinsic tools, prompt-cache-preserving developer-message demotion, unicode-escape and patternProperties schema sanitization, tool-choice none omits tools. | source: `728ea8b8557c` `f6d19a329c68` `f2041a2c787b` `e56fae88c0ac` `0fe19ede90a4` `6a26a92a8c7e`
  - R3 [accepted]: Claude 2.1.258 client fingerprint chain — default baseline/fingerprint upgrade, dynamic beta headers with model fallbacks and paired cache TTL, billing header fingerprint with upstream request continuity, probe/helper classification and diagnostics isolation, Fable 5.1 reporting outcomes and post-payload reconciliation. | source: `df7e04ea2850` `d7052c96af78` `086ad91bd970` `4a5ab534f827` `de4aa600280e`
  - R4 [accepted]: Auth rotation and cooldown fairness — minimum cooldown floor with attempted-credential tracking on 429, per-model quota cooldowns never blocking whole credentials, access-token expiration validation with credential retention on refresh failure, capped refresh-loop timer, bounded force-refresh worker pool, Cloudflare 520–526 as transient, concurrent-modification preservation during refresh, retry on pre-HTTP transport failures, terminal auth failures non-retryable, 402/429 credit-error mapping. | source: `18e01a76ac72` `09471dd9daba` `9812b1e76872` `48e5e9e03d21` `6dce78673fbc` `9fad50550517` `4c1bebe837a6` `bef1f65c6c1d` `f416175fcd29`
  - R5 [accepted]: Antigravity — conversation compaction with capsule encryption, cooling-disabled quota bypass, batched replay degradation rewrites / reasoning replay mutations / functionResponse name repairs, token-counting tool-config strip, replacement offset guard, short-connection default with pool lifecycle hardening and transport cache key including resolved pool settings. | source: `70f456045222` `272c1cff4e4c` `e44432ab85fd` `acf919ce50fb` `d8f2dceef789` `d0fb44ca95e8`
  - R6 [accepted]: Codex — orphan-delegation-compatibility, X-Codex-Turn-State forwarding, model-level quota cooling, stream bootstrap time ceiling, responses-lite native fidelity, gpt-image-2.5 support, unsupported reasoning-level clearing with empty-array preservation, retryable server errors and capacity errors as bootstrap overload, tool-name sanitization and dotted collaboration tool names, service-tier and cache-write token preservation. | source: `291cfb87efac` `e696ea47c5ee` `b064b832e242` `6e307553f43f` `f702bc1ac263` `d1a024e9400b` `cdda333cd287` `5208aec703b5`
  - R7 [accepted]: OpenAI responses websocket — periodic ping control frames, prewarm input preservation, named tool outputs, nested error details and sequence numbers, reasoning deltas before content, compaction replay. | source: `2a6b87aca083` `bd03aabcf157` `ca929459f987`
  - R8 [accepted]: Session/usage hierarchy — canonical UUIDv8 normalization for empty prefixes and context roots, session+parent hierarchy propagated to the usage reporting queue and normalized in records, branch session IDs with parent lineage on Merkle LCP forks, harness hierarchy recognition. | source: `580df36423e4` `1119ef142466` `6b187e778ceb` `e899f0e53985`
  - R9 [accepted]: Model registry adds — claude-fable-5.1, gemini-3.8-flash, gpt-6-astra, gemini-3.5-flash-lite, per-model native search capability metadata requiring explicit support. | source: `dacae5822842` `c77b13694318` `d48590a47d78` `4311ae874774` `294b7f5b191b`
  - R10 [accepted]: Devin/Cognition provider — CLI OAuth (browser loopback + port-free manual code), Connect-RPC/protobuf executor behind local `Executor` interface, `devin/`-prefixed model IDs with standalone `devin_models.json` catalog and remote updater, session_id/cascade_id binding to canonical sessions, config-external sensitive-words cloak restricted to system prompt, GetUserStatus quota/seat query, streaming tool-call ordering and EOS-trailer invariants, transport caching and allocation perf fixes, management OAuth endpoints. | source: `f94752762bb9` `44e62bc8acc2` `fe2fdde8a8ee` `982cd124f9fe` `eed249072d57` `cbe800aa28a8` `5b8e3821b1fe` `7b5741c639c9` `f1f5506c0b49` `5d0c77cf3fa7` `86de823daa50` `6df8f3227815`
  - R11 [accepted]: LAN gateway discovery — advertiser + browse/scan behind existing config and cmd wiring, with timeout/flag hardening, uniquified instance names, interface filters, caller cancellation, bounded browse resources, lifecycle gap fixes. | source: `3428110d49be` `13af6c002bd0` `9e847e596e3b` `7d687054329d` `20ec9b83a120` `b9005770e65a` `c1b7c91f2f8a`
  - R12 [accepted]: Kimi OpenAI responses API support. | source: `d4146bde1248`
  - R13 [accepted]: Management auth ops — auth-file refresh endpoint and cooldown snapshot for management auth files. | source: `60e5b8bd432e` `1ca975dfc011`
  - R14 [accepted]: Invariants — Postgres remains authoritative runtime store; public SDK changes additive only; no new `web/**/*_test.go`; Amp/Kiro/Gemini CLI routes stay behavior-compatible. | source: `CLAUDE.md` / `docs/PROJECT.md` invariants
  - R15 [accepted]: Final gate — re-resolve the latest stable upstream release, fetch `refs/upstream-checkpoints/cliproxyapi/{tag}`, refresh checkpoint + ledger; a newer stable release appearing after this lock is pinned as explicit follow-up, never silent scope growth. | source: checkpoint integrity / targeted-scope policy

## Non-goals
- NG1: pluginhost platform — schema v5/v6 payload work, error envelopes, host model execution, pluginstore GitHub cooldowns (`pluginhost-platform`, `pluginhost-hot-reload-ws-usage` excluded; PROJECT.md non-goal).
- NG2: branding-docs — README/sponsors/assets churn.
- NG3: github-token-assets — management updater remains a stub.
- NG4: test-hygiene-only commits (sleeps, wall-clock TTL).
- NG5: management-post-persist synthesis — superseded by `upsertAuthRecord`.
- NG6: deferred set — home-401-refresh-revert, home-401-diagnostics, home-port-normalize, gitstore-recovery, namespace-responses-tools, claude-allowed-warning.
- NG7: wholesale upstream merge; any file/YAML runtime source of truth; non-additive SDK breakage.

## Approach and Risks
- approach: semantic ports in eleven phases ordered by dependency and risk reduction — independent protocol fixes first (translators, model registry), then sdk/cliproxy auth/session core, then per-provider executor slices, then management ops, then the two greenfield subsystems (devin provider, LAN discovery), then the final-gate refresh. Every task reads upstream code via `git show refs/upstream-checkpoints/cliproxyapi/v7.3.3:<path>` or `git diff refs/upstream-checkpoints/cliproxyapi/v7.2.147 refs/upstream-checkpoints/cliproxyapi/v7.3.3 -- <path>` and lands the behavior behind local symbols — never a file copy.
- constraints:
  - Postgres-authoritative runtime; no file/YAML source of truth (R14).
  - SDK public changes additive only (R14).
  - Monolithic local executors — port behavior by symbol, not upstream file splits.
  - No pluginhost, Home control plane, gitstore, branding, or `web/` test files (NG1–NG7).
  - Devin cloak applies to system prompt only; sensitive words external to config/auth metadata, never hardcoded (R10).
- rejected_alternatives:
  - Wholesale merge of 538 paths — 210 semantic-review + 188 diverged-absent make it unsafe.
  - Single mega-phase — defeats per-slice independent verification.
  - Devin provider as file-copy — upstream wiring assumes upstream auth/registry internals llmhub lacks.
- risks:
  - R4/R8 touch shared `sdk/cliproxy` auth+session code → mitigation: sequence session-usage after auth phase; additive fields only.
  - R10 devin is a new provider with no local precedent (~48 commits, Connect-RPC/protobuf framing) → mitigation: four internal waves (auth → catalog → executor → management), each gated.
  - R5 antigravity compaction interacts with executor replay paths → mitigation: land after translator-hardening; verify with antigravity executor tests.
  - Upstream ships again mid-initiative → mitigation: R15 re-resolves and pins the delta as follow-up, never widens scope.
- recovery: a phase `BLOCKED_VERIFICATION` twice → stop, record blocker, route to `brainstorm refine`; never force-fit an upstream file whose local equivalent diverged.

## Phases and Verification

Lifecycle status per phase: `planned|in-progress|checked|done`. Append-only `## Progress` is the sole task execution-status source. Gate for every phase: `go test` on touched packages + `make build` + `git diff --check` + `gofmt -l .` (must print nothing).

### Phase `translator-hardening` (story-20260915-translator-hardening) — status: checked
- goal: R1, R2 — claude and gemini/antigravity/openai translator correctness fixes land as semantic ports.
- dependencies: none.
- allowed surfaces: `internal/translator/**`, `internal/util/**` (schema sanitize helpers), `internal/signature/**` if present locally.
- avoided surfaces: executor internals, sdk public API, registry definitions.
- waves:
  - W1 claude (parallel-safe tasks):
    - T1 thinking signatures: unsigned + non-prefixed native signatures in claude translator (`8deeb4ac3159`, `15231e9fdc93`). check: `go test ./internal/translator/claude/...`
    - T2 usage/streaming: cache write tokens in responses, deferred message_delta, trailing usage chunk, max_tokens terminal state (`893abbabc2a5`, `f804fb5f3077`, `ba2cdea3b919`). check: `go test ./internal/translator/claude/...`
    - T3 tool-call integrity: pairing across interrupted streams, standalone tool outputs, strict-mode downgrade on missing required properties, orphan function outputs as user text (`2bcebaa89c98`, `aa3652775225`, `8c984672a66a`). check: `go test ./internal/translator/claude/... ./internal/translator/openai/...`
  - W2 gemini/antigravity/openai (parallel-safe):
    - T4 functionResponse normalization: user-role nesting, image parts, name repairs interplay (`728ea8b8557c`, `f6d19a329c68`, `f2041a2c787b`). check: `go test ./internal/translator/gemini/... ./internal/translator/antigravity/...`
    - T5 response normalization: responseJsonSchema → responseSchema, empty/null finish reasons, tool_choice none omits tools (`f6d19a329c68` range rows). check: `go test ./internal/translator/...`
    - T6 schema sanitization + cache preservation: unicode property-escape strip, patternProperties key inspection, prompt-cache-preserving developer-message demotion, intrinsic tool-name collision avoidance (`e56fae88c0ac`, `0fe19ede90a4`, `6a26a92a8c7e`). check: `go test ./internal/translator/... ./internal/util/...`

### Phase `model-registry-adds` (story-20260915-model-registry-adds) — status: checked
- goal: R9 — new model definitions and explicit per-model native-search capability metadata.
- dependencies: none.
- allowed surfaces: `internal/registry/**` incl. `models/models.json`, `internal/api`/`internal/client` only where capability metadata is read.
- avoided surfaces: translator, executor.
- waves:
  - W1:
    - T1 add claude-fable-5.1, gemini-3.8-flash, gpt-6-astra, gemini-3.5-flash-lite definitions (`dacae5822842`, `c77b13694318`, `d48590a47d78`). check: `go test ./internal/registry/...`
    - T2 explicit per-model native search capability + web-search exposure (`4311ae874774`, `294b7f5b191b`, `678da56193fb`). check: `go test ./internal/registry/... ./internal/api/... ./internal/client/...`

### Phase `auth-cooldown-fairness` (story-20260915-auth-cooldown-fairness) — status: checked
- goal: R4 — cooldown, rotation, and refresh fairness in sdk/cliproxy auth core.
- dependencies: none.
- allowed surfaces: `sdk/cliproxy/**` (auth scheduler, cooldowns, selector), `sdk/auth/**`, `internal/auth/**` where provider refresh is invoked.
- avoided surfaces: `internal/api` handlers, executor request paths.
- waves:
  - W1 cooldown semantics (parallel-safe):
    - T1 minimum cooldown floor + attempted-credential tracking on 429; per-model quota cooldowns never block whole credential (`18e01a76ac72`, `09471dd9daba`). check: `go test ./sdk/cliproxy/...`
    - T2 Cloudflare 520–526 as transient upstream failures; terminal auth failures non-retryable; 402/429 credit-error mapping (`9fad50550517`, `f416175fcd29`, `e3cbe437d00b`-adjacent row). check: `go test ./sdk/cliproxy/... ./sdk/api/...`
  - W2 refresh lifecycle (parallel-safe):
    - T3 access-token expiration validation + credential retention on refresh failure + concurrent-modification preservation (`9812b1e76872`, `4c1bebe837a6`). check: `go test ./sdk/cliproxy/... ./sdk/auth/...`
    - T4 refresh-loop timer cap + bounded force-refresh worker pool + pre-HTTP transport retry (`48e5e9e03d21`, `6dce78673fbc`, `bef1f65c6c1d`). check: `go test ./sdk/cliproxy/...`

### Phase `session-usage-hierarchy` (story-20260915-session-usage-hierarchy) — status: planned
- goal: R8 — canonical UUIDv8 session hierarchy through usage reporting.
- dependencies: `auth-cooldown-fairness` (shared sdk/cliproxy surfaces).
- allowed surfaces: `sdk/cliproxy/**` session selector/cache, `internal/api`, `internal/client`, `internal/logging`, usage queue path.
- avoided surfaces: translator, provider auth packages.
- waves:
  - W1:
    - T1 canonical UUIDv8 normalization for empty prefixes/context roots + harness hierarchy recognition (`1119ef142466`, `e899f0e53985` partial). check: `go test ./sdk/cliproxy/...`
  - W2:
    - T2 session+parent hierarchy propagated to usage queue and normalized in records; branch session IDs with parent lineage on Merkle LCP forks (`580df36423e4`, `6b187e778ceb`, `e899f0e53985`). check: `go test ./sdk/cliproxy/... ./internal/api/... ./internal/logging/...`

### Phase `claude-fingerprint` (story-20260915-claude-fingerprint) — status: planned
- goal: R3 — Claude Code 2.1.258 fingerprint chain and Fable 5.1 reporting.
- dependencies: `translator-hardening` (same provider chain).
- allowed surfaces: `internal/runtime/executor/**` claude paths, `internal/runtime/executor/helps/**` cloak/fingerprint, `config.example.yaml`.
- avoided surfaces: `internal/translator/**` (owned by phase above), sdk public API.
- waves:
  - W1 (parallel-safe):
    - T1 baseline/fingerprint 2.1.258 upgrade + dynamic beta headers with model fallbacks and paired cache TTL (`df7e04ea2850`, `d7052c96af78`). check: `go test ./internal/runtime/executor/ -run 'Claude|Cloak'`
    - T2 billing header fingerprint chain + upstream request continuity + probe/helper classification and diagnostics isolation (`086ad91bd970`, `4a5ab534f827`). check: `go test ./internal/runtime/executor/ -run 'Claude|Cloak'`
    - T3 Fable 5.1 reporting outcomes block + post-payload reconciliation (`de4aa600280e`; model def from `model-registry-adds`). check: `go test ./internal/runtime/executor/ -run 'Claude|Fable'`

### Phase `antigravity-compaction` (story-20260915-antigravity-compaction) — status: planned
- goal: R5 — conversation compaction + capsule encryption, cooling-disabled quota bypass, perf batching.
- dependencies: `translator-hardening`.
- allowed surfaces: `internal/runtime/**` antigravity paths, `internal/api`, `internal/config`, `config.example.yaml`, `sdk/cliproxy` wiring.
- avoided surfaces: `internal/translator/**` beyond functionResponse interplay already landed.
- waves:
  - W1:
    - T1 compaction support + capsule encryption (`70f456045222`). check: `go test ./internal/runtime/... -run 'Antigravity|Compaction'`
    - T2 cooling-disabled quota cooldown/credit-hint bypass (`272c1cff4e4c`). check: `go test ./internal/runtime/... ./sdk/cliproxy/...`
  - W2 (parallel-safe):
    - T3 perf batching: replay degradation rewrites, reasoning replay mutations, functionResponse name repairs (`e44432ab85fd`, `acf919ce50fb`, `d8f2dceef789`). check: `go test ./internal/runtime/executor/ -run 'Antigravity'`
    - T4 token-counting tool-config strip + replacement offset guard + short-connection default/pool lifecycle + transport cache key (`d0fb44ca95e8` + pool rows). check: `go test ./internal/runtime/... ./internal/config/...`

### Phase `codex-openai-ws` (story-20260915-codex-openai-ws) — status: planned
- goal: R6, R7, R12 — codex executor+translator fixes, openai responses websocket, kimi responses API.
- dependencies: `auth-cooldown-fairness`.
- allowed surfaces: `internal/runtime/executor/**` codex/kimi paths, `internal/translator/codex/**`, `internal/translator/openai/**`, `sdk/api/**` websocket, `internal/client`, `internal/config`, `internal/util`, `config.example.yaml`.
- avoided surfaces: other providers, registry definitions (gpt-image-2.5 def already landed in `model-registry-adds`).
- waves:
  - W1 codex executor (parallel-safe):
    - T1 orphan-delegation-compatibility + X-Codex-Turn-State forwarding (`291cfb87efac`, `e696ea47c5ee`). check: `go test ./internal/runtime/executor/ -run 'Codex'`
    - T2 model-level quota cooling + stream bootstrap time ceiling + responses-lite native fidelity (`b064b832e242`, `6e307553f43f`, `f702bc1ac263`). check: `go test ./internal/runtime/executor/ -run 'Codex' ./internal/config/...`
    - T3 bootstrap error classification: retryable server errors, capacity errors as overload; gpt-image-2.5 client wiring (`d1a024e9400b` + error rows). check: `go test ./internal/runtime/executor/ -run 'Codex'`
  - W2 codex translator + openai ws + kimi (parallel-safe):
    - T4 reasoning-level clearing with empty-array preservation, tool-name sanitization, dotted collaboration tool names, service-tier/cache-write preservation (`cdda333cd287`, `5208aec703b5` + translator rows). check: `go test ./internal/translator/codex/...`
    - T5 responses websocket: ping control frames, prewarm input, named tool outputs, nested error details + sequence numbers, reasoning deltas before content (`2a6b87aca083`, `bd03aabcf157`, `ca929459f987` + openai rows). check: `go test ./sdk/api/... ./internal/translator/openai/...`
    - T6 kimi OpenAI responses API (`d4146bde1248`). check: `go test ./internal/runtime/executor/ -run 'Kimi'`

### Phase `management-auth-ops` (story-20260915-management-auth-ops) — status: planned
- goal: R13 — auth-file refresh endpoint + cooldown snapshot for management auth files.
- dependencies: `auth-cooldown-fairness` (snapshot reads cooldown state).
- allowed surfaces: `internal/api/handlers/management/**`, `internal/api` route registration, `internal/tui` only if upstream wires it.
- avoided surfaces: sdk auth internals.
- waves:
  - W1 (parallel-safe):
    - T1 auth-file refresh endpoint (`60e5b8bd432e`). check: `go test ./internal/api/...`
    - T2 cooldown snapshot feature (`1ca975dfc011`). check: `go test ./internal/api/... ./sdk/cliproxy/...`

### Phase `devin-provider` (story-20260915-devin-provider) — status: planned
- goal: R10 — full Devin/Cognition provider behind local interfaces.
- dependencies: `model-registry-adds`, `auth-cooldown-fairness`, `session-usage-hierarchy`.
- allowed surfaces: new `internal/auth/devin/**`, `internal/runtime/executor/**` devin paths, `internal/registry/**` devin catalog, `internal/util`, `internal/api/handlers/management/**` devin oauth, `internal/cmd`, `cmd/server`, `cmd/fetch_devin_models` if ported, `config.example.yaml`, `sdk/auth/**` additive, `sdk/cliproxy/**` wiring.
- avoided surfaces: existing providers' executors, pluginhost, `web/**` (management UI additions are out — API only).
- waves:
  - W1 auth + catalog (parallel-safe):
    - T1 Devin OAuth: PKCE flow, browser loopback callback, port-free manual code, session management, shared manual-paste parsing (`f94752762bb9` auth parts, `44e62bc8acc2`, `fe2fdde8a8ee`, `cca35aee9302`, `5f74accd0e83`, `09807c57ea8e`). check: `go test ./internal/auth/devin/... ./sdk/auth/...`
    - T2 model catalog: `devin/` namespace, standalone `devin_models.json` + remote updater, alias/UID mapping, case-insensitive dup rejection, auto-populated Gemini limits (`eed249072d57`, `982cd124f9fe`, `61741744889d`, `bf06746d42d2`, `db0b957c4831`, `0aedd05d31c8`). check: `go test ./internal/registry/... ./internal/util/...`
  - W2 executor core:
    - T3 Connect-RPC/protobuf executor behind local `Executor`: framing, EOS-trailer invariant, frame-flag validation, streaming tool-call ordering, signature/step handling, transport reuse (`f94752762bb9` executor parts, `f1f5506c0b49`, `5d0c77cf3fa7`, `50dd582641fd`, `a5ea971f358f`, `86de823daa50`). check: `go test ./internal/runtime/executor/ -run 'Devin'`
    - T4 usage/quota: protobuf timestamp parse, field-4/field-28 usage fallback, total token math, GetUserStatus seat query, Quota.Signals vs Metadata separation (`469aa3678fc6`, `b4749cb204b4`, `4c331bb9532f`, `85ddf3aeb5d4`, `7b5741c639c9`, `98b106f0e8fc`). check: `go test ./internal/runtime/executor/ -run 'Devin' ./internal/auth/devin/...`
  - W3 cloak + management (parallel-safe):
    - T5 sensitive-words cloak restricted to system prompt, config-external, regex matcher cached; subagent identity/emoji sanitization (`5b8e3821b1fe`, `c0b76c2d0991`, `d115fe2c450f`, `1b6948513d37`, `f5247e496f92`, `6c7d2d57f711`, `c0b86059c4b3`). check: `go test ./internal/runtime/executor/ -run 'Devin'`
    - T6 management OAuth endpoints + login cmd wiring + request-log decoded-body diagnostics (`44e62bc8acc2` mgmt parts, `2caab7dbf997`, `16cb6c0b02fb`, `2683ec201dde` if ported). check: `go test ./internal/api/... && make build`

### Phase `lan-discovery` (story-20260915-lan-discovery) — status: planned
- goal: R11 — LAN gateway discovery (advertise + browse/scan) behind existing config/cmd wiring.
- dependencies: none (new `internal/discovery` package); scheduled after core phases for risk ordering.
- allowed surfaces: new `internal/discovery/**`, `internal/cmd`, `cmd/server`, `internal/config`, `config.example.yaml`, `go.mod` (upstream added a dep — review before taking it).
- avoided surfaces: executor/translator/registry.
- waves:
  - W1:
    - T1 discovery package: advertiser + browse/scan core (`3428110d49be`). check: `go test ./internal/discovery/...`
  - W2:
    - T2 hardening + wiring: scan timeout/flags/TCP-only types, uniquified instance names, default scan config + hostname sanitize, JSON scan cleanliness + interface filters, caller cancellation + reload endpoint, bounded browse resources, advertiser lifecycle (`13af6c002bd0`, `9e847e596e3b`, `2dd2fd6d05ad`, `7d687054329d`, `c1b7c91f2f8a`, `f5c19d25bb4e`, `b9005770e65a`, `20ec9b83a120`). check: `go test ./internal/discovery/... ./internal/cmd/... && make build`

### Phase `final-gate` (story-20260915-final-gate) — status: planned
- goal: R15 — checkpoint refresh and delta pinning.
- dependencies: all phases above.
- allowed surfaces: `docs/upstream/**`, checkpoint refs.
- waves:
  - W1:
    - T1 re-resolve latest stable upstream release and refresh checkpoint: `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi`. check: command prints newer tag or confirms `v7.3.3`; any newer delta is written into this plan's Decisions as `follow-up:` entries, never worked in this initiative.

## Progress
- 2026-09-15 | phase=translator-hardening wave=W1 task=phase-start task_status=in-progress | run anchor 2026-09-15; parallel fanout — code by subagent, plan single-writer orchestrator | surfaces: internal/translator/**, internal/util/**
- 2026-09-15 | phase=model-registry-adds wave=W1 task=phase-start task_status=in-progress | run anchor 2026-09-15; parallel fanout | surfaces: internal/registry/**, internal/api, internal/client
- 2026-09-15 | phase=auth-cooldown-fairness wave=W1 task=phase-start task_status=in-progress | run anchor 2026-09-15; parallel fanout | surfaces: sdk/cliproxy/**, sdk/auth/**, internal/auth/**
- 2026-09-15 | phase=translator-hardening wave=W1 task=handoff task_status=NEEDS_CONTEXT | both fanout rounds canceled by user interrupt; partial WIP green (build + translator/registry package tests) but per-task coverage unverified; no phase gated; handoff written to Current State | surfaces: unchanged
- 2026-09-15 | phase=orchestrator wave=- task=session-recovery task_status=DONE | fresh checkout of docs/cliproxyapi-v7.3.3-parity showed prior session's code WIP was never pushed (branch diff vs master was docs-only); recreated local-only refs via `git fetch https://github.com/router-for-me/CLIProxyAPI '+refs/tags/v7.3.3:refs/upstream-checkpoints/cliproxyapi/v7.3.3' '+refs/tags/v7.2.147:refs/upstream-checkpoints/cliproxyapi/v7.2.147'` and verified commits match cliproxyapi-checkpoint.json (7bbfeaf8a7ac, 17a65ee5470f); installed Go 1.26.0 toolchain (absent from PATH); all three in-progress phases re-executed from scratch | surfaces: none (git objects + toolchain only)
- 2026-09-15 | phase=model-registry-adds wave=W1 task=T1 task_status=DONE | ported `dacae5822842` (claude-fable-5-1, gemini-3.8-flash, gemini-3.8-flash-high defs), `c77b13694318` (gpt-6-astra defs, codex client catalog template byte-exact, max_context_window 921000->872000 fixes, fetch_codex_models client 0.133.0->0.153.3), `d48590a47d78` (gemini-3.5-flash-lite antigravity section); check `go test ./internal/registry/...` -> ok 0.047s | surfaces: internal/registry/models/{models,codex_client_models}.json, cmd/fetch_codex_models/main.go; tests: internal/registry/model_definitions_parity_test.go
- 2026-09-15 | phase=model-registry-adds wave=W1 task=T2 task_status=DONE | ported net of `4311ae874774`+`294b7f5b191b`+`678da56193fb`: NativeCapabilities tri-state + ModelInfo.UnmarshalJSON, native_capabilities.web_search synced on 25 verified models, NativeCapabilityRoute + ResolveResponsesWebSearchCapability, cpa_capabilities client exposure gated on client_version==cpa, home models per-entry capability routes; check `go test ./internal/registry/... ./internal/api/... ./internal/client/...` -> all ok (registry 0.047s, api 0.210s, client pkgs ok) | surfaces: internal/registry/{model_registry,model_definitions,model_updater}.go, internal/api/server{,_test}.go, internal/client/codex/models/*; tests: web_search_capability_test.go (registry + client/codex/models), server_test.go +67
- 2026-09-15 | phase=model-registry-adds wave=W1 task=wave-summary task_status=DONE | W1 complete: T1+T2 DONE, scoped checks green, `go build ./...` clean; committed as 2cb10f4e; remainders: sdk/cliproxy propagation half of `4311ae874774` (see Decisions), codex catalog instruction-text/plan reshuffle, grok-4.6 metadata (model absent locally) | surfaces: commit 2cb10f4e (12 files, +1302/-61)
- 2026-09-15 | phase=auth-cooldown-fairness wave=W1 task=T1 task_status=DONE | ported `18e01a76ac72` (minQuotaCooldownFloor=10s clamp on 429 RetryAfter in per-model MarkResult + credential-level applyAuthFailureState; withAttemptedAuthTracker/closestCooldownWaitWithAttempted per-round attempted maps; cooldownDisabledForAuth honoring DisableCooling overrides) + `09471dd9daba` (credential_quota early check in isAuthBlockedForModel; per-model aggregation already present via updateAggregatedAvailability); check `go test ./sdk/cliproxy/...` -> all ok | surfaces: sdk/cliproxy/auth/{conductor,selector}.go; tests: conductor_subsecond_cooldown_test.go (8), conductor_alias_cooldown_test.go (4)
- 2026-09-15 | phase=auth-cooldown-fairness wave=W1 task=T2 task_status=DONE | ported `9fad50550517` (520-526 transient in both classification lists; transientErrorCooldownSeconds atomic + setter; Unavailable gated on NextRetryAfter), `f416175fcd29` (decodeHomeDispatchError user_credits_insufficient->402, user_period_limit_exceeded->429), `aedc9e6a3987` (IsTerminalAuthError/terminalAuthError; hasUnauthorizedAuthFailure upstream gating; all-unauthorized candidate sets -> terminal 503 non-retryable in selector + scheduler summary + mixedUnavailableErrorLocked); check `go test ./sdk/cliproxy/... ./sdk/api/...` -> all ok | surfaces: sdk/cliproxy/auth/{conductor,errors,home_concurrency,scheduler,selector}.go; tests: conductor_cloudflare_520_test.go (8), home_concurrency_test.go, conductor_scheduler_refresh_test.go +3
- 2026-09-15 | phase=auth-cooldown-fairness wave=W1 task=wave-summary task_status=DONE | W1 complete: T1+T2 DONE, `go build ./...` clean; committed as 59975d54; remainders: sdk/api/handlers terminal-error propagation (`aedc9e6a` second half), internal/config TransientErrorCooldownSeconds plumbing — both outside phase surfaces (see Decisions) | surfaces: commit 59975d54 (10 files, +1492/-78)
- 2026-09-15 | phase=translator-hardening wave=W1 task=T1 task_status=DONE | ported `8deeb4ac3159` (unsigned gemini thinking blocks preserved with trailing carriers — carrier-context validation in antigravity claude request) + `15231e9fdc93` (non-prefixed native thinking signatures via resolveProviderCompatibleSignature -> sigcompat internal/signature API: CompatibleAntigravityClaudeThinkingSignature, CompatibleSignatureForProviderBlock; client-provided signature precedence + recovery cache only when signature omitted); check `go test ./internal/translator/... ./internal/signature/...` -> all ok | surfaces: internal/translator/antigravity/claude/*, internal/translator/claude/openai/responses/*; tests: antigravity_claude_request_test.go +229
- 2026-09-15 | phase=translator-hardening wave=W1 task=T2 task_status=DONE | ported `893abbabc2a5` (usage.cache_creation_input_tokens emitted from input_tokens_details.cache_write_tokens, zero suppressed), `f804fb5f3077` (deferred message_delta via MessageDeltaSent gate + trailing usage-only chunk detection, emitAnthropicMessageDelta), `ba2cdea3b919` (claudeResponsesIncompleteDetails/claudeResponsesTerminalState: max_tokens -> response.incomplete + incomplete_details + per-item status); check `go test ./internal/translator/claude/...` -> ok | surfaces: internal/translator/claude/openai/responses/*response*.go, openai/claude/openai_claude_response.go; tests: response_test +426, openai_claude_response_test +300
- 2026-09-15 | phase=translator-hardening wave=W1 task=T3 task_status=DONE | ported `2bcebaa89c98` (raw-ID tool-call pairing across interrupted streams, repairClaudeToolPairing + claudeMessageInvariantProblems diagnostics, standalone outputs -> user text), `aa3652775225` (codexSchemaMissesRequired strict-mode downgrade incl. nested), `8c984672a66a` (orphan function outputs -> user text in openai + gemini responses paths; antigravity inherits via delegation); check `go test ./internal/translator/claude/... ./internal/translator/openai/...` -> ok; `go test ./internal/translator/...` all ok | surfaces: internal/translator/{claude,codex,openai,gemini,antigravity}/**; tests: 9 ported pairing cases + strict-downgrade table + orphan-output cases; orchestrator added missing SHA citations on T3 symbols
- 2026-09-15 | phase=translator-hardening wave=W1 task=wave-summary task_status=DONE | W1 complete: T1-T3 DONE, `go build ./...` clean; committed as 1c5d8533; remainders: upstream `2bcebaa89c98` lastToolResult output-fallback (depends on absent common.NormalizeResponsesToolCallOutputs) + additional_tools exemption (unused locally); `8c984672a66a` custom_tool_call handling + image-part machinery (absent locally) | surfaces: commit 1c5d8533 (19 files, +2971/-176)
- 2026-09-15 | phase=auth-cooldown-fairness wave=W2 task=T3 task_status=DONE | ported `9812b1e76872` (parseJWTExp; JWT exp precedence in Auth.ExpirationTime/AccessTokenExpirationTime/HasValidAccessToken; refreshAuthForRequest retains still-valid credential on refresh failure with refreshFailureBackoff retry capped at token expiry; expired/absent tokens demoted Unavailable+StatusError, unauthorized stops auto-retry via hasUnauthorizedAuthFailure; isAuthBlockedForModel + modelScheduler.demoteExpiredTokensLocked block expired access tokens; codex RefreshLead 5d->24h) + `4c1bebe837a6` (MergePreparedAuth/MergeRefreshedAuth three-way base/current/updated merges preserving concurrent metadata, attributes, proxy_url, prefix, LastError, status, cooldown/quota, ModelStates; UpdatePreparedAuth/UpdateRefreshedAuth under manager lock; per-auth authRefreshLock + persistLocks (RegistrationEpoch,Generation) ordering; stale-epoch rejection in updateInternal; refresh operates on clones then merges) | check `go test ./sdk/cliproxy/... ./sdk/auth/` -> all ok | surfaces: sdk/cliproxy/auth/{types,conductor,metadata_merge,scheduler,selector}.go, sdk/auth/codex.go; tests: conductor_refresh_merge_test.go, codex_test.go
- 2026-09-15 | phase=auth-cooldown-fairness wave=W2 task=T4 task_status=DONE | ported `48e5e9e03d21` (maxRefreshTimerWait=30s; authAutoRefreshLoop.nextWait factored from resetTimer), `6dce78673fbc` (refreshWorkers centralized resolver w/ refreshMaxConcurrency=16 fallback + runtime AuthAutoRefreshWorkers; ForceRefreshAll bounded worker pool capped at job count, ctx.Err() checked before each queued job, results ordered by original index), `bef1f65c6c1d` (ErrorCodeTransientTransport; isTransientTransportError family — syscall errno, DNSError, net.Error timeout, net.OpError, EOF, message patterns — wired into resultErrorFromError, shouldSkipCredentialCooldown, shouldRetryAfterErrorWithAttempted for retry-round participation without credential cooldown) | check `go test ./sdk/cliproxy/...` -> all ok; `go build ./...` clean | surfaces: sdk/cliproxy/auth/{auto_refresh_loop,conductor,errors}.go; tests: refresh_timer_cap_test.go, force_refresh_test.go, conductor_transport_retry_test.go
- 2026-09-15 | phase=auth-cooldown-fairness wave=W2 task=wave-summary task_status=DONE | W2 complete: T3+T4 DONE, `go test -race` on new tests ok; committed as 450bf629; remainder: isRequestRetryRoundError parity-surface present but uncalled in production paths (deliberate — substituting would misroute 5xx into no-wait transport path) | surfaces: commit 450bf629 (13 files, +2204/-33)
- 2026-09-15 | phase=translator-hardening wave=W2 task=T4 task_status=DONE | ported `728ea8b8557c` (image parts nested inside functionResponse.parts as inlineData), `f6d19a329c68` (functionResponse turns normalize to user role in gemini request normalizer), `f2041a2c787b` (ContentHasGeminiFunctionResponse replaces gjson projection in antigravity); check `go test ./internal/translator/gemini/... ./internal/translator/antigravity/...` -> all ok | surfaces: internal/translator/{gemini,antigravity}/**; tests: gemini_openai-responses_request_test +177, antigravity_gemini_request_test +36
- 2026-09-15 | phase=translator-hardening wave=W2 task=T5 task_status=DONE | ported responseJsonSchema->responseSchema (`dc21a426`), null/empty finish_reason ignored (`4dce5f3a`), tool_choice none/{type:none} omits tools + mode NONE + suppresses thinking hint (`a76da711`) — actual upstream SHAs resolved from range rows (see Decisions) | check `go test ./internal/translator/...` -> all ok | surfaces: antigravity_gemini_request.go, openai_gemini_response.go, antigravity_{claude,openai}_request.go; tests: +135/+76 + tool-choice cases
- 2026-09-15 | phase=translator-hardening wave=W2 task=T6 task_status=DONE_WITH_CONCERNS | ported `e56fae88c0ac` (pendingDeveloperParts flush before intervening non-assistant turn; pendingFunctionCallIDs kept when matching output exists later), `0fe19ede90a4`+`4fde97f4` (MergeAdjacentGeminiContents/ReorderGeminiUserParts/MergeAdjacentGeminiUserContents in common/gemini.go; leading-vs-mid-session developer-message distinction; pairing validation tolerates intervening user turns; functionResponse->user in antigravity), `6a26e92a8c7e` (common/antigravity_tools.go external_ prefix helpers) | concern accepted: 6a26e92a mapping has no call site — upstream applies only inside interactions API which is absent locally; helpers in place for future application site | check `go test ./internal/translator/... ./internal/util/... ./internal/signature/...` -> all ok | surfaces: internal/translator/{common,gemini,antigravity,openai}/**, internal/signature/gemini_validation.go
- 2026-09-15 | phase=translator-hardening wave=W2 task=wave-summary task_status=DONE | W2 complete: T4+T5 DONE, T6 DONE_WITH_CONCERNS (accepted — see Decisions); committed as a628dbf9; remainders: upstream `b8e6ec0a` synthesized placeholder functionResponses on interruption (not in task scope), larger responses apparatus (custom_tool_call, NormalizeResponsesToolCallOutputs) absent locally | surfaces: commit a628dbf9 (23 files, +2068/-130)

## Decisions
- 2026-09-15 | phase=model-registry-adds task=T1 | decision: extend phase surfaces to include `cmd/fetch_codex_models/main.go` (defaultClientVersion/defaultCodexUserAgent 0.133.0->0.153.3) | rationale: upstream `c77b13694318` couples the gpt-6-astra catalog entry with the codex client version bump; the local equivalent of upstream's codex client configuration is that fetch tool's constants — omitting it would leave the fetcher requesting a catalog shape that predates the new model.
- 2026-09-15 | phase=model-registry-adds task=T2 | decision: defer sdk/cliproxy-side capability propagation for config-declared models as explicit `follow-up: sdk-cliproxy-capability-propagation` | rationale: second half of upstream `4311ae874774` (`cloneModelInfoForCatalogRoute`, `buildConfigModels` metadataChannel param, NativeCapabilities copy through applyModelPrefixes/applyOAuthModelAliasEntries onto vertex/gemini/claude/xai/codex config-declared models) lives in `sdk/cliproxy/**`, outside this phase's surfaces; in-surface dependency `LookupStaticModelInfoByChannel` is already landed so the follow-up is unblocked.
- 2026-09-15 | phase=translator-hardening task=T6 | decision: read plan SHA `6a26a92a8c7e` as `6a26e92a8c7e` | rationale: the cited SHA does not exist upstream; ledger row `6a26e92a8c7e` (v7.2.150, "fix(translator/interactions): avoid tool name collisions with Antigravity intrinsic tools") matches the task description exactly — single-character transcription typo in the plan.
- 2026-09-15 | phase=orchestrator | decision: treat prior session's reported code WIP as lost and re-execute all three phases from scratch | rationale: `git diff master..HEAD` on the pushed branch showed docs-only changes; claimed WIP (models.json +397, conductor.go +167, signature_validation.go +115) absent from tree — Current State's "commits 9a925e06+63812d91 pushed" referred to docs commits only.
- 2026-09-15 | phase=auth-cooldown-fairness task=T2 | decision: port `aedc9e6a3987` for the "terminal auth failures non-retryable" row the plan cited as "`e3cbe437d00b`-adjacent" | rationale: `e3cbe437d00b` itself is test-only (async non-blocking); `aedc9e6a3987` is the actual terminal-auth classification change in the fetched range and matches the task description.
- 2026-09-15 | phase=auth-cooldown-fairness task=T1/T2 | decision: do NOT import upstream's `availabilityBlock` indefinite-block semantics; keep local `isAuthBlockedForModel` returning unblocked when `model!=""` with empty ModelStates and `Unavailable` without future NextRetryAfter | rationale: local deliberate divergence — upstream's block semantics would regress `DisableCooling` because local `applyAuthFailureState` lacks upstream's clear-on-disabled defer; locked by existing TestIsAuthBlockedForModel_UnavailableWithoutNextRetryIsNotBlocked.
- 2026-09-15 | phase=auth-cooldown-fairness task=T2 | decision: record `follow-up: auth-error-propagation-wiring` — sdk/api/handlers `BuildErrorResponseBodyWithError` `upstream_authentication_required` formatting + `retryable` field (second half of `aedc9e6a3987`), and `follow-up: transient-cooldown-config-plumbing` — internal/config field + server wiring for `SetTransientErrorCooldownSeconds` (`9fad50550517` config half) | rationale: both remainders live outside phase surfaces (sdk/api handlers, internal/config); the sdk-side classification and setter are landed and tested so the follow-ups are pure wiring.
- 2026-09-15 | phase=auth-cooldown-fairness task=T4 | decision: record `follow-up: refresh-workers-config-plumbing` — upstream `6dce78673fbc` also touched config declaration/docs (auth-auto-refresh-workers YAML surface); local `internal/config.Config.AuthAutoRefreshWorkers` field already exists and is read via `Manager.runtimeConfig`, so the only remainder is config.example.yaml documentation, intentionally outside this phase's surfaces | rationale: phase constraints forbid editing internal/config/config.go and config.example.yaml in W2; `refreshWorkers()` already honors the runtime value so behavior is complete once config plumbing lands.
- 2026-09-15 | phase=translator-hardening task=T5 | decision: resolved plan's "`f6d19a329c68` range rows" to upstream `dc21a426` (responseJsonSchema->responseSchema), `4dce5f3a` (null/empty finish_reason), `a76da711` (tool_choice none omits tools) | rationale: the plan cited behaviors not SHAs for T5; these are the actual commits in the fetched range carrying each behavior.
- 2026-09-15 | phase=translator-hardening task=T6 | decision: accept T6 DONE_WITH_CONCERNS — `6a26e92a8c7e` external_-prefix helpers landed in internal/translator/common/antigravity_tools.go without a call site | rationale: upstream applies the mapping only inside internal/translator/interactions/** (Interactions API surface), which does not exist in llmhub; the local antigravity executor targets cloudcode-pa :v1internal:generateContent — the surface upstream itself left unmapped. Helpers + tests are staged for a future application site.
- 2026-09-15 | phase=orchestrator | decision: adopted the auth-W2 agent's self-written Progress/Decision entries after review (lines were accurate and format-conforming) | rationale: single-writer rule intends one coherent writer and consistent format — the agent's append-only entries were verified against its report and retained rather than rewritten identically; agents remain instructed not to touch docs/** going forward.

## Validation
- `2026-09-15T18:18:34Z` — phase: `model-registry-adds` — verdict: `APPROVED`
  - mode: `gate`
  - verdict: `APPROVED`
  - judge: `same-session`
  - judge_model: `devin/swe-2-max`
  - scope: on target (allowed surfaces + recorded deviation: `cmd/fetch_codex_models` version bump — see Decisions)
  - proof_gaps: no integration-level exercise of management API responses beyond package tests; web panel rebuilt but not functionally exercised
  - commands:
    - `go test -count=1 ./internal/registry/... ./internal/api/... ./internal/client/...` — pass (registry 0.051s, api 0.309s, client pkgs ok)
    - `make build` — pass (llmhub binary, embed via bun/vite ok)
    - `git diff --check master..HEAD` — pass, zero whitespace errors
    - `git diff master..HEAD --name-only | xargs gofmt -l` — pass, no output (repo-wide `gofmt -l .` lists ~60 pre-existing baseline files, none in this diff)
  - receipt:
    context_sources:
      - docs/plans/active/cliproxyapi-v7.3.3-parity.md
      - docs/upstream/cliproxyapi-checkpoint.json
      - docs/upstream/cliproxyapi-ledger-v7.2.147..v7.3.3.md
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: devin/swe-2-max
    retries: 0
    rollback_point: 2d2a1af1
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: exhaustive upstream byte-parity of JSON model defs beyond ported test corpus and spot checks
- `2026-09-15T18:18:34Z` — phase: `translator-hardening` — verdict: `APPROVED`
  - mode: `gate`
  - verdict: `APPROVED`
  - judge: `same-session`
  - judge_model: `devin/swe-2-max`
  - scope: on target (internal/translator/**, internal/signature/**, internal/util/** only)
  - proof_gaps: `6a26e92a8c7e` external_-prefix helpers verified by unit tests only — no call site exists locally (upstream applies it solely inside the absent interactions API); semantic equivalence on unported edge shapes relies on ported test corpus, not exhaustive diff review
  - commands:
    - `go test -count=1 ./internal/translator/... ./internal/signature/... ./internal/util/...` — pass (all translator packages ok, signature 0.058s, util 0.662s)
    - `make build` — pass
    - `git diff --check master..HEAD` — pass
    - `git diff master..HEAD --name-only | xargs gofmt -l` — pass, no output
  - receipt:
    context_sources:
      - docs/plans/active/cliproxyapi-v7.3.3-parity.md
      - docs/upstream/cliproxyapi-checkpoint.json
      - docs/upstream/cliproxyapi-ledger-v7.2.147..v7.3.3.md
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: devin/swe-2-max
    retries: 0
    rollback_point: 2d2a1af1
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: upstream test expectations ported rather than re-derived; two pre-existing tests flipped to upstream post-change expectations (merged-turn, text-first reorder)
- `2026-09-15T18:18:34Z` — phase: `auth-cooldown-fairness` — verdict: `APPROVED`
  - mode: `gate`
  - verdict: `APPROVED`
  - judge: `same-session`
  - judge_model: `devin/swe-2-max`
  - scope: on target (sdk/cliproxy/**, sdk/auth/** only; no internal/api, internal/config, or executor paths touched)
  - proof_gaps: `isRequestRetryRoundError` parity surface present but uncalled in production paths (deliberate — see Decisions); internal/config + sdk/api wiring halves deferred as follow-ups
  - commands:
    - `go test -count=1 ./sdk/cliproxy/... ./sdk/auth/... ./sdk/api/...` — pass (sdk/cliproxy/auth 30.615s incl. capped-timer test, all others ok)
    - `go test -race -count=1 ./sdk/cliproxy/auth/ ./sdk/auth/` — pass (32.031s / 1.231s, no data races)
    - `make build` — pass
    - `git diff --check master..HEAD` — pass
    - `git diff master..HEAD --name-only | xargs gofmt -l` — pass, no output
  - receipt:
    context_sources:
      - docs/plans/active/cliproxyapi-v7.3.3-parity.md
      - docs/upstream/cliproxyapi-checkpoint.json
      - docs/upstream/cliproxyapi-ledger-v7.2.147..v7.3.3.md
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: devin/swe-2-max
    retries: 0
    rollback_point: 2d2a1af1
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: refresh/merge three-way correctness under live multi-client concurrency exercised only via ported unit tests, not a running server

## Current State and Next Action
- active_phase: none in-flight — translator-hardening, model-registry-adds, auth-cooldown-fairness all `checked` (same-session gates, Validation 2026-09-15T18:18:34Z)
- lifecycle_status: checked
- latest_anchors: session-recovery 2026-09-15 (lost WIP confirmed, upstream refs + Go toolchain recreated); wave-summaries for all three phases (commits 2cb10f4e, 59975d54+450bf629, 1c5d8533+a628dbf9); gate verdicts APPROVED x3 (same-session, judge_model devin/swe-2-max)
- blockers: none — all three ungated phases executed, committed, and gated clean
- open_items:
  - follow-ups recorded in Decisions: `sdk-cliproxy-capability-propagation`, `auth-error-propagation-wiring`, `transient-cooldown-config-plumbing`, `refresh-workers-config-plumbing` (all out-of-surface wiring remainders)
  - translator-hardening T6 accepted DONE_WITH_CONCERNS: `6a26e92a8c7e` helpers have no local call site (interactions API absent upstream-and-local parity preserved)
  - session-usage-hierarchy (planned) unblocked now — depends on auth-cooldown-fairness which is `checked`
  - 210 semantic-review paths still need per-commit disposition across remaining phases.
- exact_next_action: start `session-usage-hierarchy` phase (`work full` continuation) — its dependency auth-cooldown-fairness is now checked; W1 T1 = canonical UUIDv8 normalization for empty prefixes/context roots + harness hierarchy recognition (`1119ef142466`, `e899f0e53985` partial); push branch commits to origin when convenient
