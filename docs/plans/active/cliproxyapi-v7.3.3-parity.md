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

### Phase `translator-hardening` (story-20260915-translator-hardening) — status: in-progress
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

### Phase `model-registry-adds` (story-20260915-model-registry-adds) — status: in-progress
- goal: R9 — new model definitions and explicit per-model native-search capability metadata.
- dependencies: none.
- allowed surfaces: `internal/registry/**` incl. `models/models.json`, `internal/api`/`internal/client` only where capability metadata is read.
- avoided surfaces: translator, executor.
- waves:
  - W1:
    - T1 add claude-fable-5.1, gemini-3.8-flash, gpt-6-astra, gemini-3.5-flash-lite definitions (`dacae5822842`, `c77b13694318`, `d48590a47d78`). check: `go test ./internal/registry/...`
    - T2 explicit per-model native search capability + web-search exposure (`4311ae874774`, `294b7f5b191b`, `678da56193fb`). check: `go test ./internal/registry/... ./internal/api/... ./internal/client/...`

### Phase `auth-cooldown-fairness` (story-20260915-auth-cooldown-fairness) — status: in-progress
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

## Decisions
- none

## Validation
- none

## Current State and Next Action
- active_phase: translator-hardening, model-registry-adds, auth-cooldown-fairness — all `in-progress`, none gated
- lifecycle_status: in-progress
- latest_anchors: run anchor 2026-09-15 (phase-start Progress lines); handoff 2026-09-15 (interrupted parallel fanout); branch `docs/cliproxyapi-v7.3.3-parity`, PR #20, commits 9a925e06+63812d91 pushed
- blockers: none structural — execution interrupted, not blocked
- open_items:
  - model-registry-adds: agent reported T1+T2 DONE — models.json (+397 lines: fable-5.1, gemini-3.8-flash/-high, gemini-3.5-flash-lite, gpt-6-astra, gpt-image-2.5 defs), `codex_client_models.json` gpt-6-astra template, `NativeCapabilities{WebSearch}` tri-state + `cpa_capabilities` exposure, native-capability routes/resolution, server_test.go +128. Scoped checks green (`go test ./internal/registry/... ./internal/api/... ./internal/client/...`). UNGATED. Follow-up owed: sdk/cliproxy-side capability propagation for config-declared models (`4311ae874774` second half — out of that phase's surfaces).
  - translator-hardening: partial WIP — signature_validation.go +115 (antigravity/claude), claude/openai/responses request +366/response +345, codex/claude +20, openai/claude +59, plus test additions. T1–T6 coverage unverified; no evidence report received.
  - auth-cooldown-fairness: `sdk/cliproxy/auth/conductor.go` +167 landed; T1–T4 coverage unverified; no evidence report received.
  - 210 semantic-review paths still need per-commit disposition across remaining phases.
- exact_next_action: verify WIP — `go build ./... && go test ./internal/translator/... ./internal/registry/... ./internal/api/... ./internal/client/... ./sdk/cliproxy/...` — then resume the three phases (in-session or fresh agents): audit per-task coverage against plan task lists, flush Progress entries, then gate each phase in-session per `work-full.md` step 11
