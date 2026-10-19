# Upstream ledger — cliproxyapi v8.0.28..v8.0.33

- generated: 2026-10-19T05:23:21Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `38a3e7388d28`
- non-merge commits: 54

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v8.0.29 | `50eaed0d1686` | 2026-10-13 | feat(pluginapi): expose config field defaults | `internal/api`, `internal/pluginhost`, `sdk/pluginapi` |  |  |
| v8.0.29 | `08e5762a9a3c` | 2026-10-14 | fix(cliproxy): add model registration fencings and epoch-checked publications to prevent superseded updates | `internal/registry`, `sdk/cliproxy` |  |  |
| v8.0.29 | `5efe97582c05` | 2026-10-14 | feat(antigravity): flatten multi-branch tool unions for Claude model requests | `internal/runtime`, `internal/util` |  |  |
| v8.0.29 | `b3dc6c3fe52b` | 2026-10-14 | refactor(config): move tls-handshake-timeout-seconds to root request configuration | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v8.0.29 | `345cc9d5a895` | 2026-10-14 | feat(claude): add timeout and retry logic for silent TLS handshakes | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
| v8.0.29 | `8fb633c9fa21` | 2026-10-14 | fix(antigravity): raise fallback client version to 2.23.0 | `internal/misc` |  |  |
| v8.0.29 | `f9ac397be288` | 2026-10-14 | feat(registry): add muse-wonder-full-1.0 model variants and update context limits | `internal/registry` |  |  |
| v8.0.29 | `912a038b8b39` | 2026-10-14 | fix(antigravity): correct token usage calculations for cached tokens in Gemini responses | `internal/translator` |  |  |
| v8.0.29 | `7f2f17d76fe1` | 2026-10-14 | feat(gemini): treat MALFORMED_FUNCTION_CALL as retryable API error in Claude translation | `internal/runtime`, `internal/translator`, `sdk/api` |  |  |
| v8.0.29 | `582570076282` | 2026-10-14 | feat(claude): ensure unique tool use IDs across turns in Gemini response translator | `internal/translator` |  |  |
| v8.0.29 | `a19a8ecde3eb` | 2026-10-14 | feat(registry): add gemini-3.8 model variants and update test cases | `internal/runtime` |  |  |
| v8.0.29 | `5fc580b1d72d` | 2026-10-15 | feat(watcher): add file removal tracking and single-file auth snapshotting for robust deletion reconciliation | `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.30 | `2d7f5377e4b7` | 2026-10-15 | feat(vertex): honor full /publishers/ prefix in API-key base-url | `internal/config`, `internal/runtime` |  |  |
| v8.0.30 | `7459255675f7` | 2026-10-15 | feat(registry): add gemini-4-argon model definition | `internal/registry` |  |  |
| v8.0.30 | `35af74f92721` | 2026-10-15 | fix(auth): protect active model and credential cooldowns from being cleared by stale in-flight execution succe | `sdk/cliproxy` |  |  |
| v8.0.30 | `59930ba36a82` | 2026-10-15 | feat(claude): map incomplete OpenAI Responses tool status to tool_result is_error | `internal/translator` |  |  |
| v8.0.30 | `bbfd7886ee13` | 2026-10-15 | fix(xai): drop web_search options the Responses API rejects | `internal/runtime` |  |  |
| v8.0.30 | `e9ba5c507f46` | 2026-10-15 | feat(util): strip unsupported discriminator keyword from JSON schemas for Gemini and Antigravity | `internal/util` |  |  |
| v8.0.30 | `1631875b9ac3` | 2026-10-15 | fix(watcher): guard management saves against stale config reloads and serialize file updates | `internal/api`, `internal/config`, `internal/watcher` +1 |  |  |
| v8.0.30 | `ceb4397f1e83` | 2026-10-15 | feat(executor): drop trailing thought-only model turns in Gemini Vertex requests | `internal/runtime` |  |  |
| v8.0.30 | `f94377007581` | 2026-10-15 | feat(config): add PreapareConfigFileForUpload and seed remote storage with hashed secrets | `cmd/server`, `internal/config`, `internal/store` |  |  |
| v8.0.30 | `4e8a4125ace0` | 2026-10-15 | feat(config): add operator overrides for Claude header defaults and client request ID modes | `internal/runtime` |  |  |
| v8.0.30 | `9db0729f5fef` | 2026-10-15 | feat(config): add operator overrides for Claude header defaults and client request ID modes | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v8.0.31 | `355fd10e76b8` | 2026-10-16 | feat(pluginhost): add content-addressed shadow copies for dynamic library loading | `internal/pluginhost`, `internal/runtime`, `internal/thinking` +2 |  |  |
| v8.0.31 | `fa68946ad572` | 2026-10-16 | feat(management): add optimistic concurrency control with ETag and If-Match to v8 config endpoints | `internal/api`, `internal/util` |  |  |
| v8.0.31 | `7a511e4db78a` | 2026-10-16 | feat(claude): pin validated request body per-stream in OpenAI Responses translation | `internal/translator` |  |  |
| v8.0.32 | `088e445715e2` | 2026-10-16 | test(meta): handle client disconnect gracefully after terminal stream events | `internal/runtime` |  |  |
| v8.0.32 | `7604d3df3e62` | 2026-10-16 | feat(claude): add session date pin retention to preserve continuity across credential switches | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v8.0.32 | `202b8b8c80bd` | 2026-10-16 | feat(claude): clean and normalize Claude tool arguments against original schemas | `internal/translator`, `internal/util` |  |  |
| v8.0.32 | `e7003d2a3e2c` | 2026-10-16 | feat(docs): add kedaya.ai as a new AI model API relay service provider in README files | `README.md`, `README_CN.md`, `assets/kedaya.jpg` |  |  |
| v8.0.32 | `911f49cbd49f` | 2026-10-17 | feat(claude): add automatic thinking block binding and beta policy for preserved-thinking models | `internal/runtime`, `internal/translator` |  |  |
| v8.0.32 | `b946d54e4be6` | 2026-10-17 | feat(devin): add account-wide quota exhaustion tracking and credential cooldown refresh | `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.33 | `b015f208d28c` | 2026-10-16 | fix(oauth): restore credential exclusion priority | `internal/api`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.33 | `8071ef599395` | 2026-10-16 | docs(readme): add Hermes CLIProxyAPI Provider to "Who is with us?" | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.33 | `94d1a9a04646` | 2026-10-17 | Refactor authentication exclusion handling and improve model registration | `internal/api`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.33 | `16b53545e006` | 2026-10-17 | test(cliproxy): stabilize antigravity dynamic registration lifecycle tests | `sdk/cliproxy` |  |  |
| v8.0.33 | `c4c1d43eaf8c` | 2026-10-17 | feat(auth): prevent stale file snapshots from overwriting later persisted state | `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.33 | `4f2e52709dcd` | 2026-10-17 | feat(runtime): detect upstream stream truncation across executors and proxy handlers | `internal/runtime`, `sdk/api` |  |  |
| v8.0.33 | `a43026d75196` | 2026-10-17 | feat(gemini): strip unsupported contentMediaType and contentEncoding JSON schema keywords | `internal/util` |  |  |
| v8.0.33 | `ad9e456d4a07` | 2026-10-17 | feat(claude): support `between_tools` thinking disable type and adaptive fallbacks | `internal/modelconfig`, `internal/pluginhost`, `internal/registry` +5 |  |  |
| v8.0.33 | `e13bc9a1f0ca` | 2026-10-17 | feat(claude): synthesize adaptive thinking block binding for preserved-thinking models | `internal/runtime` |  |  |
| v8.0.33 | `294ed09797a0` | 2026-10-18 | feat(discovery): add robust mDNS host conflict resolution and IP binding fallback | `internal/discovery` |  |  |
| v8.0.33 | `acaab599988f` | 2026-10-18 | feat(sdk): expose antigravity user agent helpers | `sdk/auth` |  |  |
| v8.0.33 | `66c2758159ab` | 2026-10-18 | feat(pluginhost): remove max line buffer size limit in stream executor to preserve usage telemetry | `internal/pluginhost` |  |  |
| v8.0.33 | `a7608a8a2573` | 2026-10-18 | feat(pluginhost): preserve auth identity and model registration for relative auth paths | `internal/pluginhost` |  |  |
| v8.0.33 | `adc436ff87eb` | 2026-10-18 | fix(pluginhost): bind catalog-only plugin models to credentials | `internal/pluginhost`, `sdk/cliproxy`, `sdk/pluginapi` |  |  |
| v8.0.33 | `d7005b63c7f4` | 2026-10-18 | fix(cliproxy): apply excluded models to plugin models | `sdk/cliproxy` |  |  |
| v8.0.33 | `3e309fb47471` | 2026-10-18 | fix(auth): preserve plugin excluded models on synthesis | `internal/api`, `internal/watcher` |  |  |
| v8.0.33 | `bb4282471a88` | 2026-10-18 | docs(config): clarify oauth excluded-models priority | `config.example.yaml` |  |  |
| v8.0.33 | `1a28c61e5632` | 2026-10-18 | fix(auth): tolerate invalid excluded models on load | `internal/api`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.33 | `db1bab011d98` | 2026-10-18 | feat(runtime): narrow Claude per-minute rate-limit cooldown when short Retry-After is present | `internal/runtime` |  |  |
| v8.0.33 | `f75ea0733657` | 2026-10-18 | feat(cliproxy): reconcile latest auth state when model registration is superseded mid-flight | `sdk/cliproxy` |  |  |
| v8.0.33 | `3a0722984ab7` | 2026-10-18 | feat(meta): filter subscription usage telemetry from SSE streams and responses | `internal/runtime`, `sdk/api` |  |  |
| v8.0.33 | `b8d4e14a8e65` | 2026-10-18 | feat(runtime): optimize antigravity schema sanitization to scale linearly with declarations | `internal/runtime` |  |  |
