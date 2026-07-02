# CLIProxyAPI Upstream Parity Audit

Created: 2026-07-02
Upstream: `router-for-me/CLIProxyAPI`
Local repo: `therealtinhtute/llmhub`

## Verdict

Do not merge upstream wholesale. The upstream range is large and structurally
divergent: `v7.1.23..v7.2.49` changes 497 backend-relevant files with about
79k insertions and 16k deletions under `internal/`, `sdk/`, `cmd/server`, and
config. The largest areas are plugin host/store, auth scheduling, runtime
executors, translators, OpenAI handlers, and management APIs.

Use targeted backport slices. Preserve llmhub's custom Postgres-only runtime,
embedded web, Kiro provider, release/install flow, and Amp support.

## Baseline

- llmhub starts at `41420d3` on `2026-05-27T17:34:07+07:00`.
- Primary upstream baseline: `v7.1.23`, published `2026-05-26T16:54:40Z`.
- Same-day boundary check: `v7.1.24`, published `2026-05-27T19:06:41Z`.
- Latest upstream checked: `v7.2.49`, published `2026-07-02T01:40:12Z`.
- Upstream default branch at audit time: `main`, `cde9336b`.

Commands used:

```bash
gh repo view router-for-me/CLIProxyAPI --json nameWithOwner,defaultBranchRef,latestRelease,pushedAt
git -C /tmp/cliproxyapi-upstream-plan diff --shortstat v7.1.23..v7.2.49 -- internal sdk cmd/server config.example.yaml go.mod go.sum
git -C /tmp/cliproxyapi-upstream-plan log --date=short --format='%h %ad %s' v7.1.23..v7.2.49 --no-merges -- internal sdk cmd/server config.example.yaml
rg -n -F '<symbol>' internal sdk cmd README.md
```

## Already Present Or Mostly Present

| Area | Upstream evidence | Local status | Recommendation |
| --- | --- | --- | --- |
| OpenAI Responses full-transcript replacement | `8f686345 fix(responses): full transcript replay on WS-to-SSE Codex paths` | Present in `sdk/api/handlers/openai/openai_responses_websocket.go` and tests mention "full transcript". | Keep. Only compare upstream test expansion before Slice 1. |
| OAuth model alias | `f66376f0 feat(auth): add per-auth OAuth model alias support`; later plugin-provider alias tests | Present locally in `internal/config`, `sdk/cliproxy/auth/oauth_model_alias.go`, management routes, watcher diff, and tests. | Keep local shape. Backport only per-auth/plugin gaps if needed. |
| `disable-cooling` metadata/config | `21d8164c feat(handlers): add disable-cooling support in OpenAI compatibility configuration` | Present in config structs, metadata parsing, watcher synthesizer, and built web asset. | Keep. Verify management API parity if Slice 3 runs. |
| OpenAI image/video baseline | `7de9757c feat: add OpenAI video support`; later `ae6c5eae` adds `gpt-image-1.5` | Local has `openai_images_handlers.go`, `openai_videos_handlers.go`, routes for `/v1/videos`, `gpt-image-2`, `grok-imagine-image`, and `grok-imagine-video`; local does not have `gpt-image-1.5`. | Partial. Backport only the newer model/version deltas in Slice 4. |
| Codex websocket/full transcript basics | `56988aea feat(websockets): add Codex websocket passthrough`; `f33bc56b feat(websockets): add transcript state tracking` | Local has `internal/wsrelay` and OpenAI websocket handler tests. | Keep. Compare upstream tests before touching. |
| API key usage grouping basics | `369e560f feat(api): refactor provider key logic for API key usage` | Local has API key management page/story and management API. | Keep local UI/API unless a failing use case appears. |
| Kiro parity | Not upstream; llmhub custom | Local has `internal/auth/kiro`, Kiro executor/quota/overage stories. | Preserve. Treat upstream auth/runtime changes as possible support code only. |

## Missing Useful Backports

### Slice 1 - Translator, Responses, WebSocket Bug Fixes

Priority: P1. These are low-to-medium conflict if handled as targeted tests and
small code ports.

| Item | Upstream commits | Affected local packages | Notes | Smallest verification |
| --- | --- | --- | --- | --- |
| Responses SSE framing and force-map fixes | `150e7f0d`, `a7250275`, `fd93ee03` | `sdk/api/handlers/openai`, `sdk/cliproxy/auth` | Local has force/OAuth alias logic, but these commits specifically repair Responses SSE and upstream response model rewrite. Backport as tests first. | `go test ./sdk/api/handlers/openai ./sdk/cliproxy/auth` |
| Response ID rewrite for repeated responses | `2fa4dabe`, `ed52c614` | `sdk/api/handlers/openai` | Local has full transcript handling, but upstream added repeated-response and pending-tool-call tests later. Port tests first to expose gap. | `go test ./sdk/api/handlers/openai` |
| Reasoning content merge/signature handling | `611d65ea`, `3648bc15`, `4b51f85c`, `ec672446`, `4b681031`, `2cbb8c7b` | `internal/translator/openai`, `internal/translator/gemini`, `internal/translator/antigravity`, `internal/translator/claude` | High user impact for Claude/Codex/Gemini compatibility. Avoid broad package replacement because llmhub has Kiro and preserved Gemini CLI paths. | `go test ./internal/translator/...` |
| Tool-call and schema cleanup fixes | `57e1bf97`, `011ffe1d`, `b4bec344`, `c44d4fcc`, `34639c3c`, `75fa6265`, `30dc2e7f` | `internal/translator/...`, `sdk/api/handlers/openai` | Good backport candidates. Port tests for FIFO tool IDs, `parametersJsonSchema`, `$comment`, `enumDescriptions`, delayed Codex function starts, and Claude server tool blocks. | `go test ./internal/translator/... ./sdk/api/handlers/openai` |
| Multimodal field preservation | `51aa5ba9`, `aa2ad995`, `35c3d80a` | `internal/translator/gemini`, `internal/translator/openai`, `internal/translator/antigravity` | Local already has some `input_audio`/`input_image`; upstream adds more coverage and `video_url` request handling. | `go test ./internal/translator/gemini/... ./internal/translator/openai/...` |
| Antigravity reasoning replay/signature repair | `365e8fc2`, `292456a8`, `b17d29ad`, `ef19f5fc`, `c55157dc`, `48dcadd9`, `ca7478a1` | `internal/runtime/executor`, `internal/translator/antigravity`, `internal/cache`, `internal/misc` | Worth auditing if Antigravity users hit bad signatures, web search, or UA failures. Local does not have upstream `internal/signature` package, so port carefully. | `go test ./internal/runtime/executor ./internal/translator/antigravity/...` |
| XAI reasoning replay for Claude messages | `05d1792d`, `53a21dfb`, `303685c2` | `internal/runtime/executor`, `internal/translator/*`, `internal/cache` | Useful for Grok/Claude compatibility. Keep separate from XAI websocket transport. | `go test ./internal/runtime/executor ./internal/translator/...` |

### Slice 2 - Model Registry And API Metadata

Priority: P1/P2. Low conflict but visible.

| Item | Upstream commits | Affected local packages | Notes | Smallest verification |
| --- | --- | --- | --- | --- |
| Claude Sonnet 5 metadata | `956ce7cf` | `internal/registry/models/models.json`, registry tests | Missing locally by `rg "Claude Sonnet 5"`. Add only if the model is desired in llmhub registry. | `go test ./internal/registry ./internal/api` |
| Gemini 3.5 Flash variants/Medium tier | `65f2288a`, `09179a70` | `internal/registry/models/models.json`, API model listing | Local has `gemini-3.5-flash` and `gemini-3.5-flash-low`; audit exact upstream model list before editing. | `go test ./internal/registry ./internal/api` |
| Codex client model service tiers and max reasoning depth | `bb414de3`, `71c185f6`, `893412e9` | `internal/registry/models/codex_client_models.json`, `sdk/api/handlers/openai/codex_client_models.go` | Local already has `service_tiers`; compare upstream model JSON and tests, then port deltas only. | `go test ./sdk/api/handlers/openai ./internal/api` |
| New xAI video model | `ac1360f4`, `fb4f39d3` | `internal/registry`, OpenAI videos handler | Local supports `grok-imagine-video`; upstream adds `grok-imagine-video-1.5-preview`. | `go test ./sdk/api/handlers/openai ./internal/registry` |
| Kimi K2.7 Code | `82235202` | `internal/registry/models/models.json` | Add only if Kimi remains part of llmhub supported matrix. | `go test ./internal/registry` |

### Slice 3 - Auth, Cooldown, Quota, Management

Priority: P2. Medium conflict because llmhub owns Postgres durable runtime.

| Item | Upstream commits | Affected local packages | Notes | Smallest verification |
| --- | --- | --- | --- | --- |
| Management quota reset endpoint | `5771abbc` | `internal/api/handlers/management/quota.go`, `sdk/cliproxy/auth/conductor.go`, web if exposed | Missing locally by `rg ResetQuota`. Useful operationally, but must clear Postgres-persisted quota/runtime state consistently. | `go test ./internal/api/handlers/management ./sdk/cliproxy/auth ./internal/store` |
| Persistent cooldown state | `07c297a5`, `d33ac5e1`, `052f1934` | `sdk/cliproxy/auth`, `internal/store`, Postgres mapping | Local has cooldown scheduler and auth metadata, but upstream file-backed store must be adapted to Postgres instead of copied. | `go test ./sdk/cliproxy/auth ./internal/store` |
| Auth removal and unscheduling | `55440f0a` | `sdk/cliproxy/auth`, management auth files | Useful for stale auth removal. Must preserve local auth-manager persistence error behavior. | `go test ./sdk/cliproxy/auth ./internal/api/handlers/management` |
| Refresh dedupe and retry/backoff | `8e52c403`, `45f58d4f`, `77061aad`, `c9dc6bd6` | `internal/auth/*`, executor helpers | Good low-level reliability work. Port provider-by-provider and keep Kiro separate. | `go test ./internal/auth/... ./internal/runtime/executor` |
| Error events and Redis queue integration | `fd309448`, `f353979e`, `959067ed` | `sdk/cliproxy/auth`, `internal/redisqueue`, usage manager | Local has Postgres usage queue and Redis pieces. Treat as optional unless llmhub needs upstream-style external event consumers. | `go test ./internal/redisqueue ./sdk/cliproxy/auth ./internal/store` |
| API key alias rebuild optimization | `079ec51f`, `eb2e1e33` | `sdk/cliproxy/auth`, management API key usage | Worth porting if model alias rebuilds are slow or response model aliases are wrong. | `go test ./sdk/cliproxy/auth ./internal/api/handlers/management` |

### Slice 4 - Media, Image, Video

Priority: P2/P3. Medium conflict, user-visible API surface.

| Item | Upstream commits | Affected local packages | Notes | Smallest verification |
| --- | --- | --- | --- | --- |
| `gpt-image-1.5` support | `ae6c5eae` | `sdk/api/handlers/openai/openai_images_handlers.go`, `internal/runtime/executor/codex_openai_images.go`, registry | Missing locally by `rg "gpt-image-1.5"`. Local defaults to `gpt-image-2`. Add as explicit supported model without changing `gpt-image-2` default unless intended. | `go test ./sdk/api/handlers/openai ./internal/runtime/executor ./internal/registry` |
| Video auth binding/model propagation | `bbef8da4`, `644ba74b`, `87e6d9cf` | `sdk/api/handlers/openai/openai_videos_handlers.go`, config | Local has video endpoints but lacks upstream `video-result-auth-cache-ttl` config and newer auth binding behavior. Useful if video retrieve/download loses selected auth. | `go test ./sdk/api/handlers/openai ./internal/api` |
| `/openai/v1/videos` and video content download parity | Later upstream video handler changes around `openai_videos_handlers.go` | `internal/api/server.go`, `sdk/api/handlers/openai` | Local routes `/v1/videos`; upstream also treats `/openai/v1/videos` and content download paths as AI API paths. Decide whether llmhub wants those aliases. | `go test ./sdk/api/handlers/openai ./internal/api ./internal/logging` |
| `grok-imagine-video-1.5-preview` | `ac1360f4`, later handler/model updates | registry, video handler | Missing locally. Add with tests only if xAI upstream supports it for target auths. | `go test ./sdk/api/handlers/openai ./internal/registry` |

### Slice 5 - Plugin System

Priority: P3/high-risk initiative. Do not default backport.

Upstream adds a large plugin subsystem:

- `d625cadd`, `0ed85bb8`: pluginhost command/execution/thinking capabilities.
- `e38ba28d`, `303c0f2f`, `40f4b8b8`, `1f16e87e`: plugin store, third-party sources, latest release installs, direct installs.
- `8e39db2e`, `538e3416`, `87132e54`: host model callbacks, recursion prevention, model router.
- `44ea9abc`, `60eae92b`, `e1302645`: management resources, plugin deletion, auth provider methods and metadata.
- New SDK packages: `sdk/pluginabi`, `sdk/pluginapi`, `sdk/pluginhost`, `sdk/pluginstore`.

Local status: all plugin packages are absent. This is not a small parity fix.
It also overlaps with auth, model routing, management API, CGO/platform loading,
and external code execution. Treat it as a separate high-risk product decision
with its own threat model and UI story.

Smallest safe first step if desired:

```bash
go test ./internal/pluginhost ./internal/pluginstore ./sdk/pluginapi ./sdk/pluginabi ./sdk/pluginhost ./sdk/pluginstore
```

That command will only be valid after adding the packages.

## Conflicts With llmhub Custom

| Upstream change | Why it conflicts | Local decision |
| --- | --- | --- |
| `8122b9fe feat!: remove amp integration support` | llmhub README still advertises Amp CLI/IDE support and local tree has `internal/api/modules/amp`. | Keep Amp by default. Do not port upstream deletion. |
| File-backed cooldown/auth persistence | llmhub's runtime source of truth is Postgres. File-backed upstream state can reintroduce `/etc` or local filesystem runtime assumptions. | Adapt concepts to Postgres only. |
| Upstream pluginhost/pluginstore | Executes external plugins, adds management APIs, platform loaders, auth provider hooks, and routing callbacks. | Separate high-risk initiative. |
| Upstream Gemini CLI package removals | Local still has Gemini CLI paths and translator packages. | Do not remove without explicit product decision. |
| Upstream release/Docker/sponsor/docs churn | llmhub owns release binaries, embedded web build, VPS installer, and branding. | Skip except dependency requirements for selected code. |
| Upstream management web assumptions | llmhub embeds and customizes web management panel. | Backend API changes need local web compatibility checks before exposing. |

## Skip

- Sponsor asset/readme churn.
- Docker/release workflow rewrites.
- Upstream automatic plugin release/install documentation until Slice 5 is approved.
- Upstream Amp removal.
- Broad package replacement of translators, executors, auth manager, or config.
- Any local Kiro behavior removal or rewrite while backporting common auth/runtime helpers.

## Backport Order

1. Slice 1a: add upstream tests for Responses/SSE/force-map/response ID issues, then port minimal fixes.
2. Slice 1b: translator schema/tool/reasoning fixes with tests.
3. Slice 2: registry/model metadata deltas.
4. Slice 3a: quota reset design for Postgres, then backend endpoint.
5. Slice 3b: cooldown/auth retry reliability adapted to Postgres.
6. Slice 4: media model deltas and video binding if needed.
7. Slice 5: plugin system only after explicit approval.

## Verification Matrix

Run focused checks per slice first:

```bash
go test ./sdk/api/handlers/openai ./internal/translator/...
go test ./internal/runtime/executor ./sdk/cliproxy/auth
go test ./internal/api/handlers/management ./internal/store
go test ./internal/registry ./internal/api
```

Before any release or broad merge:

```bash
go test ./...
cd web && bun run type-check && bun run build
git diff --check
```

For management API or web-visible changes, add an API/browser smoke for the
changed endpoint or page. Static Go tests are not enough for those surfaces.

## Concrete Next Work Packet

Recommended first implementation packet:

Title: Backport low-conflict Responses and translator fixes from CLIProxyAPI

Scope:

- Port upstream tests and fixes for `150e7f0d`, `a7250275`, `fd93ee03`,
  `2fa4dabe`, `611d65ea`, `3648bc15`, and `4b51f85c`.
- Touch only `sdk/api/handlers/openai`, `sdk/cliproxy/auth`, and the minimum
  translator packages needed by failing tests.
- Preserve Amp, Kiro, Postgres runtime, and embedded web.

Acceptance:

```bash
go test ./sdk/api/handlers/openai ./sdk/cliproxy/auth ./internal/translator/...
git diff --check
```

Stop conditions:

- A fix requires pluginhost/pluginstore.
- A fix requires removing Amp or Gemini CLI routes.
- A fix changes Postgres runtime persistence behavior outside tests.

## Implementation Status - 2026-07-02

Completed in `US-015-upstream-parity-backport`:

- Added Codex `gpt-image-1.5` model support while preserving `gpt-image-2`
  as the existing default.
- Added xAI `grok-imagine-video-1.5-preview` registry/API support.
- Added `/openai/v1/videos` aliases for video create/retrieve/content.
- Added video result auth binding with `video-result-auth-cache-ttl` and
  proxy-aware content download.
- Added management `POST /v0/management/reset-quota`, keyed only by stable
  `auth_index`.
- Added `Manager.ResetQuota` to clear auth/model quota state, persist through
  the existing manager store, resume registry routing, and update scheduler
  state.
- Kept Amp, Kiro, Gemini CLI, Postgres runtime, embedded web, installer/release
  flow, and branding untouched.

Verification run:

```bash
go test ./sdk/api/handlers/openai ./internal/registry ./internal/api ./internal/logging
go test ./internal/api/handlers/management ./sdk/cliproxy/auth ./internal/store ./internal/redisqueue
go test ./sdk/api/handlers/openai ./sdk/cliproxy/auth ./internal/translator/...
go test ./internal/runtime/executor ./internal/translator/antigravity/... ./internal/auth/...
```

Deferred:

- Upstream `pluginhost/pluginstore`: still a separate high-risk initiative.
- XAI websocket executor and broad upstream `internal/signature` subsystem:
  not imported wholesale because local runtime/executor tests already pass and
  the upstream change touches many provider paths. Backport only with a separate
  focused story if live XAI websocket mode is required.
