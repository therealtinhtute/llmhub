---
id: plan-20260920-v739
type: plan
intake_id: intake-20260920-v739
lane: high-risk
status: active
created: 2026-09-20
updated: 2026-09-20
---

# Plan: CLIProxyAPI v7.3.6..v7.3.9 full parity + Devin web-panel entry

## Outcome
- result: llmhub ports the owner-approved v7.3.6..v7.3.9 capability slices (43 commits, 166 paths) as independently verifiable semantic ports behind existing executor, translator, auth, registry, config, and util interfaces — plus the llmhub-specific Devin OAuth entry in the web panel — without wholesale merge or branding/pluginhost churn.
- success_signals:
  - Each accepted slice lands with focused tests citing its upstream commit(s) and local symbol.
  - Response-model observability: `helps/response_model.go` + `helps/stream_response_model_observer.go` record authentic upstream response models; silent model substitution warns across all providers; devin records expected upstream model; observer memory bounded.
  - Devin wave-2: images in tool results; orphaned tool results handled + `function_result` payloads normalized; tool calls aggregated by id; cache-write tokens tracked; post-tool text buffered so tool calls order before assistant response.
  - Claude cooling/overage: `ClaudeKey` gains model-level cooling + scope-overage rate-limit fields; overage-only rejections skip `retry-after`; advisor tool check restricted to server tool use; hybrid passthrough restores mcp tool names.
  - Translator correctness: tool responses scoped per turn (repeated call IDs); tool-call messages aligned + ordering preserved on ambiguous outputs; gemini `parametersJsonSchema` keeps constraints/`additionalProperties`; strict tool mode maps for gemini/claude; tool choice fail-closed; `cache_write_tokens` deducted from `input_tokens` in openai/codex→claude; `true` boolean subschemas normalized; `common/openai_tools.go` exists.
  - Codex: ultrafast service tier normalized; item-level/tool-output prompt cache breakpoints stripped; compat models preserve reasoning content/IDs; stream request timeout classified as server error; duplex websocket streaming behind config flag.
  - xAI: forced hosted tool choice normalized; aliased client web-search tool name restored; grok imagine accepts 20:9 and 9:20.
  - Management: api-call rejects unresolved token placeholders; auth-file listing paginates; claude key patch accepts `priority`; expired cooldowns reconcile to active; Devin models display `(Devin)` suffix.
  - OpenAI-compat: `use-max-completion-tokens` honored via new `helps/openai_compat_max_tokens.go`.
  - Web panel: `devin` OAuth provider entry visible — start-auth hits `/v0/management/devin-auth-url`, callback/status flow completes, account appears under provider sheet.
  - `go test` on touched packages, `make build`, `git diff --check`, `gofmt` clean at gates; `bun run build`/typecheck green for web changes.
  - Newer-than-v7.3.9 upstream delta pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.9` at `61fdfc34` / `refs/upstream-checkpoints/cliproxyapi/v7.3.9`; `scope_policy` is the scope authority (15 include / 3 exclude / 6 defer carried).
  - `docs/upstream/cliproxyapi-gap-v7.3.6..v7.3.9.json` — 166 paths (baseline 5, upstream-add-absent 35, diverged-absent 59, semantic-review 67). [gitignored artifact — regenerate via `upstream_gap.py`]
  - `docs/upstream/cliproxyapi-ledger-v7.3.6..v7.3.9.md` — 43 non-merge commits across v7.3.7/v7.3.8/v7.3.9; per-commit dispositions + citations (38 adapt / 5 reject).
  - `docs/plans/completed/cliproxyapi-v7.3.6-parity.md` — prior cycle; its slices must not regress.
  - `CLAUDE.md` / `docs/WORKFLOW.md` — Postgres-authoritative store, SDK additive-only, pluginhost non-goal, no `web/**/*_test.go`.
- rejected_alternatives:
  - Wholesale merge of 166 paths — diverged-absent dominates; upstream split executors/config into multi-file packages while local stays monolithic → semantic ports only.
  - Porting upstream file splits (`claude_executor_{execute,stream,request,tokens,cloaking}.go`, `codex_websockets_{execute,stream}.go`, `xai_executor_{execute,request,response,stream,media}.go`, `conductor_{cooldown,selection}.go`, `config_{types,normalization,yaml}.go`, `internal/modelconfig/`) — map onto local `claude_executor.go`, `codex_executor.go`, `codex_websockets_executor.go`, `xai_executor.go`, `conductor.go`, `config.go`, `internal/watcher/diff/model_hash.go`.
  - pluginhost commits `b715526a`, `afba07ba`, `1c878749` — local `internal/pluginhost/` is `lifecycle.go` only; platform stays excluded.
  - Branding commits `0b9a91fb` (FluxA), `784285a4` (PatewayAI) + assets; CI commit `b773607e` (upstream go-cross workflow vs local goreleaser).
- requirements:
  - R1 [accepted]: **Response-model observability** — new `helps/response_model.go` + `helps/stream_response_model_observer.go` (+tests) ported from upstream end-state; call sites across `aistudio`, `antigravity`, `claude`, `devin`, `gemini`, `kimi`, `meta`, `xai`, `openai_compat`, `codex` (terminal + images) executors; `redisqueue/plugin.go` records upstream model; bounded observer memory. | source: `e84e248c` `cde7d57e` `f8467f07` `e9463ff5` `25f40d8c`
  - R2 [accepted]: **Devin executor wave-2** — `64c9433f` image tool results (executor part only; `internal/translator/interactions/claude/` absent locally); `b6fe4f20` orphaned tool results + `function_result` payload normalization; `9e10db53` tool-call aggregation by id + cache-write tokens (`devin_executor.go` + `devin_wire.go`); `b6d1f050` post-tool text buffering for correct ordering; `e84e248c` devin expected-model wiring (consumes R1 helpers). | source: `64c9433f` `b6fe4f20` `9e10db53` `b6d1f050` `e84e248c` `f8467f07`(devin hunks)
  - R3 [accepted]: **Claude cooling + overage** — `44eaef00` model-level cooling fields on `ClaudeKey` + new `helps/claude_ratelimit.go`; `1cce9325` skip `retry-after` on overage-only rejections; `f86a33f7` advisor tool check → server tool use only; `22392c53` hybrid passthrough mcp tool names; `f4852170` cloak persistence test. | source: `44eaef00` `1cce9325` `f86a33f7` `22392c53` `f4852170`
  - R4 [accepted]: **Config schema additions** (wave-0 shared, single owner of `config.go`/`sdk_config.go`) — `ClaudeKey` cooling fields (`44eaef00`), `OpenAICompatibilityModel.use-max-completion-tokens` (`690f4f31`), codex websocket duplex flag (`42c9680e`), `config.example.yaml` updates. | source: `44eaef00` `690f4f31` `42c9680e` (config hunks)
  - R5 [accepted]: **Translator tool correctness** — new `internal/translator/common/openai_tools.go` (`cc545cbf`); per-turn tool-response scoping (`c2ea2684`); strict tool mode (`f247e2b0`); tool-choice fail-closed (`49eec664`); `parametersJsonSchema` constraints/`additionalProperties` (`b532db9c`); `cache_write_tokens` deduction (`883660fb`); boolean subschemas + keywords (`c93978c4`); codex ultrafast tier (`859c4865`) + cache-breakpoint strip (`3662d153`); `sdk/translator/registry.go` +21 hunks; `helps/codex_multi_agent_v2.go` hunk. | source: `c2ea2684` `cc545cbf` `b532db9c` `f247e2b0` `49eec664` `883660fb` `c93978c4` `859c4865` `3662d153`
  - R6 [accepted]: **Responses→Gemini multimodal** — `61fdfc34`: audio/video/file input parts in `gemini_openai-responses_request.go` (+769-line rewrite) + antigravity responses test + `b532db9c`'s responses-test hunk. | source: `61fdfc34` `b532db9c`(responses test)
  - R7 [accepted]: **Codex executor misc** — `cb62a674` stream timeout→server error (`sdk/api/handlers/openai_responses_stream_error.go` + `test/codex_incomplete_stream_error_type_test.go`); `81d6ba77` compat reasoning content/ID preservation incl. new `openai_responses_signature.go` (executor hunks; websocket hunks go to R8). | source: `cb62a674` `81d6ba77`
  - R8 [accepted]: **Codex duplex websockets** — `42c9680e`: new `codex_websockets_duplex.go` (~601 LOC) + 7 test files + `sdk/cliproxy/executor/websocket_input.go` + `sdk/api/handlers/openai/openai_responses_websocket_input.go` + steering tests + `server_options.go` wiring; `81d6ba77` ws hunks. Consumes R4 flag. | source: `42c9680e` `81d6ba77`(ws hunks)
  - R9 [accepted]: **xAI fixes** — `c616193a` forced hosted tool choice normalization; `660a5800` aliased web-search tool name restore (+request/response/stream wiring onto monolith); `f049e00b`+`75bd6a60` grok imagine aspect ratios in `openai_images_handlers.go`. | source: `c616193a` `660a5800` `f049e00b` `75bd6a60`
  - R10 [accepted]: **Management API** — `0b550539` api-call placeholder rejection (+838-line test); `76ac75e6` auth-file pagination; `93b94d22` claude-key `priority` patch field; `b4ff581d` expired-cooldown reconcile (`auth_files.go` + conductor hunk); `28743473` `(Devin)` display suffix (`client/codex/models/models.go` + `server_routes.go`). | source: `0b550539` `76ac75e6` `93b94d22` `b4ff581d` `28743473`
  - R11 [accepted]: **OpenAI-compat max-completion-tokens** — `690f4f31`: `helps/openai_compat_max_tokens.go` + call sites in `openai_compat_executor.go`; `internal/watcher/diff/model_hash.go` hash update (upstream `internal/modelconfig/` absent). Consumes R4 field. | source: `690f4f31`
  - R12 [accepted]: **Devin web-panel entry** (llmhub-specific, no upstream source) — `OAuthProvider` unions (`web/src/types/oauth.ts`, `web/src/services/api/oauth.ts`) + `WEBUI_SUPPORTED` gain `'devin'`; `entries.ts` provider entry; `devin_oauth_*` i18n keys in `en.json`+`vi.json`; `web/src/assets/icons/devin{,-dark}.svg`. Backend `/v0/management/devin-auth-url` + `/devin/callback` + `get-auth-status` contract already shipped in v0.0.41.
  - R13 [accepted]: Invariants — Postgres authoritative runtime store; SDK additive-only; semantic ports behind local interfaces; no `web/**/*_test.go`; no commits from subagents.
  - R14 [accepted]: Final gate — re-resolve latest upstream release, refresh checkpoint, pin newer delta as explicit follow-up.

## Plan

### Wave 0 — infra (sequential gate before wave 1)
Two parallel-safe slices; both must land before wave 1 fans out.

- **W0-A config schema** (R4): `internal/config/config.go`, `internal/config/sdk_config.go`, `config.example.yaml`, new config test files. Owns `config.go` for the cycle.
- **W0-B response-model observability** (R1): new `internal/runtime/executor/helps/{response_model,stream_response_model_observer}{,_test}.go`, `helps/usage_helpers.go`, call-site lines in every executor file, `internal/redisqueue/plugin.go`, `codex_response_model_test.go`, `response_model_multiprovider_test.go`, `internal/runtime/executor/openai_responses_signature.go`.

### Wave 1 — parallel phases (file-disjoint)
| Phase | Requirement | Owns | Depends |
|---|---|---|---|
| W1-devin | R2 | `devin_executor*`, `helps/devin_wire*` | W0-B |
| W1-claude | R3 | `claude_executor*`, `helps/claude_ratelimit*`, `config/claude_code_test.go`, `config/cloak_save_test.go` | W0-A |
| W1-translator | R5 | `internal/translator/**` except `gemini/openai/responses/gemini_openai-responses_request{,_test}.go` + `antigravity/openai/responses/*_test.go`; `util/gemini_schema*`; `helps/codex_multi_agent_v2*`; `sdk/translator/registry.go` | — |
| W1-multimodal | R6 | `gemini/openai/responses/gemini_openai-responses_request{,_test}.go`, `antigravity/openai/responses/antigravity_openai-responses_request_test.go` | W1-translator ordering note: applies b532db9c's responses-test hunk itself |
| W1-codex | R7 | `codex_executor*`, `openai_responses_signature*` (shared w/ W0-B — W0-B creates, W1 extends), `sdk/api/handlers/openai_responses_stream_error*`, `test/codex_incomplete_stream_error_type_test.go` | W0-B |
| W1-duplex | R8 | `codex_websockets_*`, `sdk/cliproxy/executor/websocket_input.go`, `sdk/api/handlers/openai/openai_responses_websocket{,_input}.go` + steering tests, `internal/api/server_options.go` | W0-A |
| W1-xai | R9 | `xai_executor*`, `xai_websockets_executor*`, `sdk/api/handlers/openai/openai_images_handlers*` | — |
| W1-mgmt | R10 | `internal/api/handlers/management/{api_tools,auth_files,config_lists}*` + tests, `sdk/cliproxy/auth/conductor.go` (cooldown hunk), `internal/client/codex/models/*`, `internal/api/server_routes.go`, `server_test.go`, `sdk/api/handlers/openai/codex_client_models_test.go` | — |
| W1-compat | R11 | `helps/openai_compat_max_tokens*`, `openai_compat_executor*` (call sites only), `watcher/diff/model_hash*` | W0-A |
| W1-web | R12 | `web/src/types/oauth.ts`, `web/src/services/api/oauth.ts`, `web/src/features/providers/entries.ts`, `web/src/i18n/locales/{en,vi}.json`, `web/src/assets/icons/devin*` | — |

Conflict notes:
- `config.go`: only W0-A edits it; consumers in W1 read the landed fields.
- `devin_executor.go`: W0-B adds call-site lines first; W1-devin owns the file afterward — sequential, no overlap.
- `gemini_openai-responses_request_test.go`: W1-multimodal owns wholesale (incl. b532db9c hunk); W1-translator must not touch it.
- `openai_responses_signature.go`: W0-B creates if needed by e9463ff5; W1-codex extends for 81d6ba77.

### Final gate
- Re-resolve latest upstream release (`upstream_sync.py sync`); pin newer-than-v7.3.9 delta as follow-up.
- `go test ./...`, `make build`, `git diff --check`, `gofmt -l` on changed files; `bun run build` for web.
- Update ledgers/checkpoint; move plan to `completed/` after validation.

## Progress
- 2026-09-20 — checkpoint synced v7.3.6→v7.3.9 (`61fdfc34`, 3 releases, baseline `f5c0ff0d`); gap 166 paths (baseline 5 / add-absent 35 / diverged-absent 59 / semantic-review 67); ledger 43 commits dispositioned (38 adapt / 5 reject); scope_policy recorded (15 include / 3 exclude). FE Devin-entry gap verified: backend contract complete in v0.0.41; web lacks `OAuthProvider`/entries/i18n/icon wiring. Next action: fan out Wave 0.
- 2026-09-20 — implementation complete via 12 fan-out subagents (W0-A config, W0-B observability, W1 devin/claude/translator/multimodal/codex/duplex/xai/mgmt/compat/web). Gate: `go build ./...` clean; `make build` clean (web embed incl. new devin provider entry); `go test ./...` green except 2 pre-existing env failures (`internal/updater TestRollbackFailure`, `cmd/server TestUpdateCommandRollback` — uid 0 bypasses chmod assertions, untouched files); `bun run build` + `tsc` + `eslint` clean; `git diff --check` clean; gofmt clean on 129 changed Go files. Upstream re-resolved: v7.3.9 still latest — no new delta. **Carry-over items**: `translateCodexRequestPair` isCompat path unported (needs `helps.TranslateRequestWithAPIKeyModelCompatibility` + `codexclaude.ConvertClaudeRequestToCodexWithCompat` — translator machinery absent); `8f23ad02` codexClientMetadataModelID template-alias (outside range); upstream `959067ed` mass `NewUsageReporter`→`NewExecutorUsageReporter` migration (≤v7.2.147, long-deferred); xai websockets executor absent locally (ws hunks of 660a5800 skipped). Plan ready to move to `completed/` after commit.
