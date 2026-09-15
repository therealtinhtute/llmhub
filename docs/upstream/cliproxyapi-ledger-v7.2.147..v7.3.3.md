# Upstream ledger — cliproxyapi v7.2.147..v7.3.3

- generated: 2026-09-15T13:10:46Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `f382bb6b69af`
- non-merge commits: 247

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.148 | `8deeb4ac3159` | 2026-09-01 | fix(antigravity): preserve unsigned gemini thinking blocks with trailing carriers | `internal/translator` |  |  |
| v7.2.148 | `c2834b68e600` | 2026-09-01 | fix(antigravity): retry model fetching per endpoint and prioritize daily base url | `cmd/fetch_antigravity_models` |  |  |
| v7.2.148 | `15231e9fdc93` | 2026-09-01 | fix(antigravity): support native thinking signatures without prefixes in claude translator | `internal/translator` |  |  |
| v7.2.148 | `893abbabc2a5` | 2026-09-01 | feat(translator): support cache write tokens in claude responses | `internal/translator` |  |  |
| v7.2.148 | `bdcccfb8e063` | 2026-09-02 | chore(registry): remove "minimal" level from dynamic_allowed definitions in models | `internal/registry` |  |  |
| v7.2.148 | `dacae5822842` | 2026-09-02 | feat(registry): add claude fable 5.1 and gemini 3.8 flash models | `internal/registry` |  |  |
| v7.2.148 | `272c1cff4e4c` | 2026-09-02 | fix(antigravity): bypass quota cooldowns and credit hints when cooling is disabled | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.148 | `d0fb44ca95e8` | 2026-09-02 | fix(antigravity): strip tool config, labels, and session id in token counting | `internal/runtime` |  |  |
| v7.2.148 | `02c02cda50b7` | 2026-09-02 | fix: harden concurrent session handling, listener lifecycle, and token accounting | `internal/api`, `internal/logging`, `internal/pluginhost` +2 |  |  |
| v7.2.148 | `d577e630b18b` | 2026-09-03 | feat(routing): make subagent session affinity configurable via session-affinity-subagents | `config.example.yaml`, `internal/config`, `sdk/cliproxy` |  |  |
| v7.2.148 | `cdda333cd287` | 2026-09-03 | fix(codex): clear unsupported reasoning levels | `internal/client` |  |  |
| v7.2.148 | `c6dd82144baf` | 2026-09-03 | refactor(kimi): use request thinking helper | `internal/runtime` |  |  |
| v7.2.148 | `6ff680e90ab5` | 2026-09-03 | feat(auth): use home model capabilities for thinking | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.148 | `1ecf0cb60204` | 2026-09-03 | fix(models): preserve home model capability metadata | `internal/api`, `internal/client` |  |  |
| v7.2.148 | `df7e04ea2850` | 2026-09-03 | fix(claude): upgrade default Claude Code baseline and fingerprint to 2.1.258 | `config.example.yaml`, `internal/runtime` |  |  |
| v7.2.148 | `18e01a76ac72` | 2026-09-03 | fix(auth): enforce minimum cooldown floor and track attempted credentials on 429 | `sdk/cliproxy` |  |  |
| v7.2.148 | `f416175fcd29` | 2026-09-03 | fix(home): map user_credits_insufficient to 402 and user_period_limit_exceeded to 429 | `sdk/cliproxy` |  |  |
| v7.2.148 | `9812b1e76872` | 2026-09-03 | fix(auth): validate access token expiration and retain valid credentials on refresh failure | `sdk/auth`, `sdk/cliproxy` |  |  |
| v7.2.149 | `2a6b87aca083` | 2026-09-03 | feat(openai): send periodic ping control frames during responses websocket streaming | `internal/config`, `sdk/api` |  |  |
| v7.2.149 | `291cfb87efac` | 2026-09-03 | feat(codex): support orphan delegation compatibility via orphan-delegation-compatibility | `config.example.yaml`, `internal/api`, `internal/client` +4 |  |  |
| v7.2.149 | `93f5266b7b00` | 2026-09-03 | feat: add Swiftproxy Sponser | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.2.149 | `699b06594b5e` | 2026-09-03 | feat: 添加 AxisNow 赞助信息及相关图像到 README 文件 | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.2.149 | `ebbce50e08b4` | 2026-09-03 | chore: remove sponsorship images for Claude API and Code0 from README files | `README.md`, `README_CN.md`, `README_JA.md` +2 |  |  |
| v7.2.149 | `e899f0e53985` | 2026-09-03 | feat(session): derive distinct branch session ID, parent lineage on Merkle LCP forks, and enhance Codex fork/s | `sdk/cliproxy` |  |  |
| v7.2.149 | `09471dd9daba` | 2026-09-03 | fix(auth): prevent individual model quota cooldowns from blocking credential | `sdk/cliproxy` |  |  |
| v7.2.150 | `e44432ab85fd` | 2026-09-03 | perf(antigravity): batch replay degradation rewrites (#5461) | `internal/runtime` |  |  |
| v7.2.150 | `f804fb5f3077` | 2026-09-03 | fix(translator/claude): defer message_delta and cache streaming usage | `internal/translator` |  |  |
| v7.2.150 | `728ea8b8557c` | 2026-09-03 | fix(translator/gemini): nest image parts inside functionResponse | `internal/translator` |  |  |
| v7.2.150 | `7899e3bcaad8` | 2026-09-03 | docs: add Infinitus to the projects based on CLIProxyAPI | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.150 | `6f16121554df` | 2026-09-03 | fix(openai-compat): honor bounded rate-limit waits | `internal/runtime` |  |  |
| v7.2.150 | `acf919ce50fb` | 2026-09-04 | perf(antigravity): batch reasoning replay mutations | `internal/runtime` |  |  |
| v7.2.150 | `f6d19a329c68` | 2026-09-04 | fix(gemini): ensure functionResponse normalizes to user role in Gemini request normalizer | `internal/translator` |  |  |
| v7.2.150 | `f2041a2c787b` | 2026-09-04 | fix(antigravity): use ContentHasGeminiFunctionResponse instead of gjson projection | `internal/translator` |  |  |
| v7.2.150 | `e56fae88c0ac` | 2026-09-04 | fix(translator): flush pending developer notice before intervening user turn | `internal/translator` |  |  |
| v7.2.150 | `0fe19ede90a4` | 2026-09-04 | fix(translator): preserve Gemini prompt cache by demoting mid-session developer messages (#5490) | `internal/signature`, `internal/translator` |  |  |
| v7.2.150 | `4a5ab534f827` | 2026-09-04 | feat(claude): harden probe and helper request classification, diagnostics isolation, and late cloaking | `internal/runtime` |  |  |
| v7.2.150 | `de4aa600280e` | 2026-09-04 | feat(claude): add Fable 5.1 reporting outcomes block and post-payload reconciliation | `internal/runtime` |  |  |
| v7.2.150 | `d7052c96af78` | 2026-09-04 | feat(claude): add 2.1.258 dynamic beta headers, model fallbacks, and paired cache TTL | `internal/runtime` |  |  |
| v7.2.150 | `086ad91bd970` | 2026-09-04 | feat(claude): implement 2.1.258 billing header fingerprint chain and upstream request continuity | `internal/runtime` |  |  |
| v7.2.150 | `1c45093d10f6` | 2026-09-04 | fix(antigravity): tighten replacement offset guard in tool provenance degradation | `internal/runtime` |  |  |
| v7.2.150 | `4c1bebe837a6` | 2026-09-04 | fix(auth): preserve concurrent modifications during auth refresh and preparation | `sdk/cliproxy` |  |  |
| v7.2.150 | `aa3652775225` | 2026-09-04 | fix(translator/claude): downgrade strict mode when schema misses required properties | `internal/translator` |  |  |
| v7.2.150 | `6a26e92a8c7e` | 2026-09-04 | fix(translator/interactions): avoid tool name collisions with Antigravity intrinsic tools | `internal/translator` |  |  |
| v7.2.150 | `649a8bdb6f34` | 2026-09-04 | feat(plugin): omit stream chunk history on payload chunks for schema v5 | `internal/pluginhost`, `sdk/api`, `sdk/pluginabi` +1 |  |  |
| v7.2.150 | `ba2cdea3b919` | 2026-09-04 | fix(translator/claude): handle incomplete status and terminal state on max_tokens | `config.example.yaml`, `internal/runtime`, `internal/translator` |  |  |
| v7.2.150 | `c77b13694318` | 2026-09-05 | feat(models): add gpt-6-astra model and update codex client configurations | `cmd/fetch_codex_models`, `internal/registry` |  |  |
| v7.2.151 | `5208aec703b5` | 2026-09-05 | fix(codex): preserve empty supported_reasoning_levels array | `internal/client` |  |  |
| v7.2.152 | `580df36423e4` | 2026-09-04 | feat(usage): propagate session and parent session hierarchy to usage reporting queue | `internal/api`, `internal/client`, `internal/logging` +6 |  |  |
| v7.2.152 | `c76dfd4e0eda` | 2026-09-06 | chore(codex): update codex user-agent to 0.153.3 | `internal/client`, `internal/registry`, `internal/runtime` |  |  |
| v7.2.152 | `5ab0bca040cd` | 2026-09-06 | fix(codex): scope usage limit errors to credentials and parse flexible quota resets | `internal/runtime`, `sdk/cliproxy`, `test/codex_quota_failover_test.go` |  |  |
| v7.2.152 | `7c2f6ce0d1a0` | 2026-09-06 | fix(claude): avoid mid-conversation system splicing for advisor calls or results | `internal/runtime` |  |  |
| v7.2.152 | `084f25c79803` | 2026-09-06 | fix(watcher): preserve concurrent file updates during auth snapshot rescans | `internal/watcher` |  |  |
| v7.2.152 | `9dfddd613d3f` | 2026-09-06 | fix(aistudio): normalize thinking level to uppercase | `internal/runtime` |  |  |
| v7.2.152 | `31ec43621b7c` | 2026-09-06 | chore(models): remove gpt-5.4 and gpt-5.4-mini models | `internal/registry` |  |  |
| v7.2.153 | `5dc428f39270` | 2026-09-06 | fix(gemini): append trailing user turn for requests ending with model content | `internal/runtime` |  |  |
| v7.2.153 | `0e85eb46f32b` | 2026-09-06 | fix(xai): support compaction fallback from payload input or previous response id | `internal/runtime` |  |  |
| v7.2.153 | `70f456045222` | 2026-09-06 | feat(antigravity): add conversation compaction support and capsule encryption | `internal/runtime` |  |  |
| v7.2.153 | `00c63a5669c5` | 2026-09-06 | feat(pluginhost): expose outbound HTTP wire profile to plugin requests | `internal/httpwire`, `internal/pluginhost`, `sdk/pluginapi` |  |  |
| v7.2.153 | `fa01468e95ae` | 2026-09-06 | perf(auth): optimize scheduler result updates with targeted model shards | `sdk/cliproxy` |  |  |
| v7.2.153 | `934fb7928c42` | 2026-09-07 | feat: add link to Aiberm in sponsorship section of README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.153 | `4cd8ee7f4ca4` | 2026-09-07 | feat: add Aiberm sponsorship information to README_JA.md | `README_JA.md` |  |  |
| v7.2.153 | `578ac8fdcf51` | 2026-09-07 | fix Aiberm sponser logo | `assets/aiberm.png` |  |  |
| v7.2.153 | `63fd25509000` | 2026-09-07 | feat: add Aiberm sponser | `README.md`, `README_CN.md`, `assets/aiberm.png` |  |  |
| v7.2.153 | `d2f7122067b9` | 2026-09-07 | fix(claude): normalize codex agent messages in responses conversion | `internal/translator` |  |  |
| v7.2.153 | `e92f6cf55700` | 2026-09-07 | fix(antigravity): enable thinking summary when responses reasoning effort is set | `internal/translator`, `test/summary_intent_translation_test.go`, `test/thinking_conversion_test.go` |  |  |
| v7.2.153 | `1c22598d0b5a` | 2026-09-07 | fix(auth): preserve active cooldown deadlines on subsequent failures | `sdk/cliproxy` |  |  |
| v7.2.153 | `a76da7115486` | 2026-09-07 | fix(antigravity): omit tools when tool_choice is none | `internal/translator` |  |  |
| v7.2.153 | `8564142fb0ec` | 2026-09-07 | fix(claude): preserve non-result block positions during tool result alignment | `internal/translator` |  |  |
| v7.2.154 | `d01516c120fa` | 2026-09-07 | fix(codex): explicitly default function tool strict to false | `internal/translator` |  |  |
| v7.2.154 | `8696585cea41` | 2026-09-07 | fix(codex): preserve service tier and cache write tokens in openai responses | `internal/translator` |  |  |
| v7.2.154 | `7871a5a978d5` | 2026-09-07 | fix(wsrelay): preserve queued frames during terminal message delivery | `internal/wsrelay` |  |  |
| v7.2.154 | `4f03809901dd` | 2026-09-07 | feat(claude): support structured output via system prompt instructions | `internal/translator` |  |  |
| v7.2.154 | `bf20b999decb` | 2026-09-07 | fix(codex): simplify complex tool schema unions and detect empty incomplete responses | `internal/runtime` |  |  |
| v7.2.154 | `d5397905f09e` | 2026-09-07 | fix(antigravity): default to short connections and harden connection pool lifecycle (fixes #5494) | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
| v7.2.154 | `82f4f370df45` | 2026-09-07 | fix(codex): restore dotted collaboration tool names in multi-agent v2 | `internal/client`, `internal/runtime` |  |  |
| v7.2.154 | `510c9c8fa529` | 2026-09-07 | fix(gemini): hide trailing text thought signatures from responses timeline | `internal/translator` |  |  |
| v7.2.154 | `ba7e55836dee` | 2026-09-08 | fix(codex): recognize retryable server errors for bootstrap failover | `internal/runtime` |  |  |
| v7.2.154 | `bee20b994025` | 2026-09-08 | fix(codex): sanitize invalid tool name characters for upstream compatibility | `internal/translator` |  |  |
| v7.2.154 | `03a054e32fc1` | 2026-09-08 | fix(codex): recover omitted tool namespaces in openai responses | `internal/translator` |  |  |
| v7.2.155 | `280b96acead0` | 2026-09-04 | fix(claude): align beta assembly and Haiku helper transport with the measured 2.1.258 | `config.example.yaml`, `internal/runtime` |  |  |
| v7.2.155 | `d8f2dceef789` | 2026-09-07 | perf(antigravity): batch functionResponse name repairs | `internal/runtime` |  |  |
| v7.2.155 | `35a4723872a4` | 2026-09-07 | fix(claude): attach helper request IDs according to the upstream base | `internal/runtime` |  |  |
| v7.2.155 | `a163c5e7edec` | 2026-09-08 | fix(logging): silence only successful health probes | `internal/api`, `internal/logging` |  |  |
| v7.2.155 | `1119ef142466` | 2026-09-08 | fix(session): harden canonical UUIDv8 normalization for empty prefixes and context roots | `sdk/cliproxy` |  |  |
| v7.2.155 | `e026cbf4e9ce` | 2026-09-08 | fix(api): skip access logging for health probes | `internal/api` |  |  |
| v7.2.155 | `3639d9242128` | 2026-09-08 | fix(plugin-store): add network scope handling for shared GitHub API cooldowns | `internal/homeplugins`, `sdk/pluginstore` |  |  |
| v7.2.155 | `20e3f731ea0b` | 2026-09-08 | feat(auth): export helper functions for headless antigravity oauth | `sdk/auth` |  |  |
| v7.2.155 | `8c0ad8ccf288` | 2026-09-08 | fix(plugin-store): honor shared GitHub API rate-limit cooldowns | `internal/api`, `internal/pluginstore` |  |  |
| v7.2.155 | `54b17ce8f743` | 2026-09-08 | fix(plugin-store): coalesce and throttle cached release checks | `internal/api`, `internal/pluginstore` |  |  |
| v7.2.155 | `1d5f7b2ac36f` | 2026-09-08 | fix(auth): strip monotonic clock reading from quota cooldown deadlines | `sdk/cliproxy` |  |  |
| v7.2.155 | `1c9d7194e09a` | 2026-09-08 | fix(plugin-store): only check releases for installed update sources | `internal/api` |  |  |
| v7.2.155 | `6b187e778ceb` | 2026-09-08 | feat(usage): normalize reported session hierarchy to canonical UUIDv8 | `internal/redisqueue`, `sdk/cliproxy` |  |  |
| v7.2.155 | `390589159eff` | 2026-09-08 | feat(session): enhance harness hierarchy recognition and deduplicate selector extraction | `internal/runtime`, `sdk/api`, `sdk/cliproxy` |  |  |
| v7.2.155 | `e365ab0cc988` | 2026-09-08 | fix(claude): map codex reasoning tokens to thinking tokens | `internal/translator` |  |  |
| v7.2.155 | `48e5e9e03d21` | 2026-09-08 | fix(auth): cap refresh loop timer wait duration | `sdk/cliproxy` |  |  |
| v7.2.155 | `ef99119e57f4` | 2026-09-08 | fix(claude): enforce sequential content blocks for interleaved streaming tool calls | `internal/translator` |  |  |
| v7.2.155 | `68dd99d56f68` | 2026-09-08 | fix(antigravity): include resolved pool settings in transport cache key | `internal/runtime` |  |  |
| v7.2.155 | `d4146bde1248` | 2026-09-08 | feat(kimi): support openai responses api | `internal/runtime`, `internal/thinking` |  |  |
| v7.2.155 | `dc21a4263912` | 2026-09-08 | fix(antigravity): normalize gemini responseJsonSchema to responseSchema | `internal/translator` |  |  |
| v7.2.155 | `6e1f9ec4e0b6` | 2026-09-08 | fix(claude): default function parameters for tools without input schema | `internal/translator` |  |  |
| v7.2.155 | `d198db54d4c4` | 2026-09-08 | fix(docs): remove Infistar.ai sponsorship details from README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.155 | `7fac6b15bcfe` | 2026-09-09 | fix(codex): strip schema dialect keywords from tool parameters | `internal/translator` |  |  |
| v7.2.155 | `4fde97f4144a` | 2026-09-09 | fix(gemini): reorder trailing text before function responses in user turns | `internal/translator` |  |  |
| v7.2.155 | `c6327a86c9f3` | 2026-09-09 | feat(plugin): preserve raw json in management responses on schema version 6 | `internal/pluginhost`, `sdk/pluginabi`, `sdk/pluginapi` |  |  |
| v7.2.155 | `d0bb908ce4f1` | 2026-09-09 | fix(gemini): normalize responses tool call outputs and resolve alternate call ids | `internal/translator` |  |  |
| v7.2.156 | `60e5b8bd432e` | 2026-09-09 | feat(management): add endpoint to refresh auth files | `internal/api`, `internal/registry`, `internal/tui` +3 |  |  |
| v7.2.156 | `37ce368c5002` | 2026-09-09 | fix(schema): inspect patternProperties keys and avoid Unicode escape fast-path bypass | `internal/runtime`, `internal/translator` |  |  |
| v7.2.156 | `e56abd56f142` | 2026-09-09 | fix(translator): strip unsupported unicode property escape patterns from tool schemas | `internal/runtime`, `internal/translator`, `internal/util` |  |  |
| v7.2.156 | `b064b832e242` | 2026-09-09 | feat(codex): support model-level quota cooling | `config.example.yaml`, `internal/config`, `internal/runtime` +1 |  |  |
| v7.2.156 | `a59b1764e781` | 2026-09-09 | fix(claude): emit trailing usage chunk and aggregate stream usage | `internal/runtime`, `internal/translator` |  |  |
| v7.2.156 | `0796d6d13317` | 2026-09-09 | feat(plugin): add host session affinity lookup callback | `internal/pluginhost`, `sdk/cliproxy`, `sdk/pluginabi` +1 |  |  |
| v7.2.156 | `d1a024e9400b` | 2026-09-10 | feat(codex): add support for gpt-image-2.5 models | `internal/api`, `internal/client`, `internal/registry` +3 |  |  |
| v7.2.156 | `aedc9e6a3987` | 2026-09-10 | fix(auth): classify terminal upstream auth failures as non-retryable errors | `sdk/api`, `sdk/cliproxy` |  |  |
| v7.2.156 | `3bf787fc1d7b` | 2026-09-10 | feat(auth): propagate canonical session id for custom header expansion | `internal/runtime`, `internal/util`, `sdk/api` +1 |  |  |
| v7.2.157 | `09a29bd345bc` | 2026-09-10 | docs(readme): remove RunAPI sponsor | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.157 | `638ed7e1cc5d` | 2026-09-10 | fix(stream): handle split CRLF across chunk boundaries in SSE validation | `sdk/api` |  |  |
| v7.2.157 | `4dce5f3a2b9a` | 2026-09-10 | fix(translator): ignore null and empty finish reasons in openai to gemini response | `internal/translator` |  |  |
| v7.2.157 | `dde250f1c386` | 2026-09-10 | fix(auth): enable model cooldown and rotation for model not found errors | `internal/clienterror`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.157 | `3ae9093da837` | 2026-09-10 | fix(codex): treat model capacity errors as bootstrap overload failures | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.157 | `259130863d4b` | 2026-09-10 | fix(openai): preserve nested error details and sequence numbers in responses stream | `internal/runtime`, `sdk/api` |  |  |
| v7.2.157 | `bd03aabcf157` | 2026-09-10 | fix(openai): preserve prewarm input and allow named tool outputs in responses websocket | `sdk/api` |  |  |
| v7.2.157 | `6a73f3962735` | 2026-09-10 | fix(claude): preserve 1h cache ttl and beta header for subagent requests | `internal/runtime` |  |  |
| v7.2.158 | `5a07045e6d21` | 2026-09-10 | fix(translator): ignore fco output item IDs when extracting responses call ID | `internal/translator` |  |  |
| v7.2.158 | `6dce78673fbc` | 2026-09-10 | fix(auth): bound force refresh all concurrency using worker pool | `config.example.yaml`, `internal/config`, `sdk/cliproxy` |  |  |
| v7.2.158 | `b8e6ec0ae770` | 2026-09-10 | fix(gemini): preserve function call pairing for interrupted calls and replay failures | `internal/runtime`, `internal/translator` |  |  |
| v7.2.158 | `9fad50550517` | 2026-09-10 | fix(auth): treat cloudflare 520-526 origin errors as transient upstream failures | `config.example.yaml`, `internal/config`, `sdk/cliproxy` |  |  |
| v7.2.158 | `fd3e6623761d` | 2026-09-10 | fix(antigravity): keep active block open on empty text parts in claude stream | `internal/translator` |  |  |
| v7.2.158 | `c8ecb4f3c972` | 2026-09-10 | fix(openai): process reasoning deltas before content in responses stream | `internal/translator` |  |  |
| v7.2.158 | `d9b8fdb77fb6` | 2026-09-11 | fix(claude): forward unmanaged caller betas on direct anthropic requests | `internal/runtime` |  |  |
| v7.2.158 | `456d4c371b51` | 2026-09-11 | fix(auth): clear unauthorized cooldowns on credential changes and sync codex plan type | `internal/api`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v7.2.158 | `b5ba02c2e362` | 2026-09-11 | fix(websockets): prevent keepalive pong starvation during large payload writes | `internal/runtime` |  |  |
| v7.2.158 | `c2562d8a5ee5` | 2026-09-11 | fix(gemini): include thought tokens in OpenAI responses usage | `internal/translator` |  |  |
| v7.2.158 | `cd1e8e10c032` | 2026-09-11 | fix(gemini): include thought tokens in OpenAI completion usage | `internal/translator` |  |  |
| v7.2.158 | `ae8f1f8be5a6` | 2026-09-11 | feat(readme): add RapidProxy sponsorship details and logo in Chinese and Japanese | `README_CN.md`, `README_JA.md` |  |  |
| v7.2.158 | `e88cd94767e6` | 2026-09-11 | feat(readme): add RapidProxy sponsorship details and logo | `README.md`, `assets/rapidproxy.png` |  |  |
| v7.2.158 | `8bd67f3338f8` | 2026-09-11 | feat(kimi): add Kimi K2.8 model definitions, normalization, and temperature guard | `internal/registry`, `internal/runtime`, `internal/thinking` |  |  |
| v7.2.158 | `377c315fd71f` | 2026-09-11 | fix(claude): anchor billing fingerprint to initial turn for cloaked cache stability | `internal/runtime` |  |  |
| v7.2.158 | `75ce6352939f` | 2026-09-11 | feat(signature): validate Claude CAQS reasoning signatures | `internal/signature` |  |  |
| v7.2.158 | `4edf9d1dd6e3` | 2026-09-11 | fix(thinking): extract configuration_update reasoning effort in codex usage reporting | `internal/runtime`, `internal/thinking` |  |  |
| v7.2.158 | `54776fb3a019` | 2026-09-11 | fix(signature): preserve Gemini 3 server-side tool thought signatures (#5652) | `internal/signature` |  |  |
| v7.2.158 | `d1702fdffd2b` | 2026-09-11 | fix(auth): drop stale auth updates using monotonic watcher revisions | `internal/api`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v7.2.158 | `5c80a01ca970` | 2026-09-11 | fix(translator): support audio transcription parts in gemini response translation | `internal/translator` |  |  |
| v7.2.158 | `942bda99792b` | 2026-09-11 | fix(translator): preserve unknown thinking signatures in claude to codex compat translation | `internal/translator` |  |  |
| v7.2.158 | `fc96a87fa66c` | 2026-09-11 | feat(auth): support organization-hashed claude credentials and legacy migration | `internal/api`, `internal/auth`, `internal/store` +2 |  |  |
| v7.2.158 | `4cd17293a3fd` | 2026-09-11 | fix(translator): relay claude tool result images as user messages for openai | `internal/translator` |  |  |
| v7.2.158 | `2912516cea7b` | 2026-09-11 | feat(auth): support execution result policy before quota and cooldown mutations | `sdk/cliproxy` |  |  |
| v7.2.158 | `4efcac7966f5` | 2026-09-11 | fix(translator): support incomplete status and response.incomplete event in responses translation | `internal/translator` |  |  |
| v7.2.158 | `8f23ad029144` | 2026-09-11 | fix(codex): inherit template metadata for model aliases and restrict provider capabilities | `internal/client`, `internal/registry`, `sdk/api` +1 |  |  |
| v7.2.158 | `8461b4e91d6f` | 2026-09-11 | chore(codex): update codex user-agent to 0.154.0 | `cmd/fetch_codex_models`, `internal/client`, `internal/registry` +1 |  |  |
| v7.2.158 | `c8f723e0fbc2` | 2026-09-11 | feat(usage): propagate upstream base_url across usage records and plugin auth | `internal/pluginhost`, `internal/runtime`, `sdk/cliproxy` +1 |  |  |
| v7.2.158 | `5b2785617d1e` | 2026-09-12 | feat(plugins): expose model list responses to plugin interceptors | `internal/api`, `sdk/api` |  |  |
| v7.2.159 | `3c3938feb1e0` | 2026-09-12 | feat(plugins): forward query parameters as metadata in auth provider start login | `internal/api`, `internal/pluginhost`, `sdk/pluginhost` |  |  |
| v7.2.159 | `b192f6550c09` | 2026-09-12 | feat(management): add plugin quota and declarative probe endpoints | `internal/api`, `internal/pluginhost`, `sdk/pluginabi` +1 |  |  |
| v7.2.159 | `e30de3d56115` | 2026-09-12 | perf(antigravity): cache and deduplicate model capability probe requests | `sdk/cliproxy` |  |  |
| v7.2.159 | `ac02da6c05e1` | 2026-09-13 | docs(readme): update APIKEY.FUN sponsor link | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.3.0 | `308e5ad3b18d` | 2026-09-11 | feat(devin): add deepseek-v4-flash and deepseek-v4-1-flash models | `internal/registry`, `internal/runtime` |  |  |
| v7.3.0 | `8a3770710e29` | 2026-09-11 | docs(devin): document cloud-side system instructions baseline in DevinExecutor | `internal/runtime` |  |  |
| v7.3.0 | `2caab7dbf997` | 2026-09-11 | feat(devin): enhance request-log with intermediate interactions and decoded upstream body | `internal/runtime` |  |  |
| v7.3.0 | `ea2f29feecb1` | 2026-09-11 | feat(devin): add devin/gemini-3-8-flash and devin/grok-4-6 model definitions and signature recognition | `internal/registry`, `internal/runtime` |  |  |
| v7.3.0 | `5b8e3821b1fe` | 2026-09-11 | fix(devin): strip system prompt lines matching configured sensitive words to evade unicode normalization bypas | `internal/runtime` |  |  |
| v7.3.0 | `7b5741c639c9` | 2026-09-11 | feat(devin): support credential quota and seat status query via GetUserStatus | `internal/auth`, `internal/runtime`, `sdk/auth` +1 |  |  |
| v7.3.0 | `c0b76c2d0991` | 2026-09-11 | refactor(devin): keep sensitive words strictly external in config.yaml without hardcoding | `internal/runtime` |  |  |
| v7.3.0 | `c0b86059c4b3` | 2026-09-11 | fix(devin): sanitize claude subagent identity and emoji directives to prevent content policy 403 | `internal/runtime` |  |  |
| v7.3.0 | `d115fe2c450f` | 2026-09-11 | refactor(devin): align sensitive-words with antigravity to config.yaml only | `internal/runtime` |  |  |
| v7.3.0 | `1b6948513d37` | 2026-09-11 | feat(devin): support sensitive-words in auth json metadata and attributes | `internal/runtime` |  |  |
| v7.3.0 | `f5247e496f92` | 2026-09-11 | fix(devin): restrict sensitive word obfuscation strictly to system prompt only | `internal/runtime` |  |  |
| v7.3.0 | `02fd1bde78c3` | 2026-09-11 | feat(devin): restrict glm-5-2 to free tier, remove static swe-1-7-lightning, and harden cloak | `internal/registry`, `internal/runtime` |  |  |
| v7.3.0 | `eed249072d57` | 2026-09-11 | feat(devin): prefix all Devin model IDs with devin/ namespace | `internal/registry`, `internal/runtime`, `internal/util` |  |  |
| v7.3.0 | `cbe800aa28a8` | 2026-09-11 | feat(devin): bind upstream session_id and cascade_id to CPA canonical session | `internal/runtime` |  |  |
| v7.3.0 | `f94752762bb9` | 2026-09-11 | feat(devin): add Devin/Cognition provider integration and CLI OAuth | `cmd/server`, `config.example.yaml`, `internal/api` +10 |  |  |
| v7.3.0 | `ca159b303d79` | 2026-09-12 | fix(util): enforce strict devin/ prefix requirement for devin provider | `internal/util` |  |  |
| v7.3.0 | `a0ccc3a414cf` | 2026-09-12 | Revert "feat(registry): transparently resolve un-prefixed devin models in auth selection" | `internal/registry` |  |  |
| v7.3.0 | `fbabc2740fe7` | 2026-09-12 | feat(registry): transparently resolve un-prefixed devin models in auth selection | `internal/registry` |  |  |
| v7.3.0 | `85ddf3aeb5d4` | 2026-09-12 | fix(devin): calculate total_input_tokens and total_tokens correctly | `internal/runtime` |  |  |
| v7.3.0 | `0aedd05d31c8` | 2026-09-12 | feat(registry): auto-populate Gemini token limits and generation methods for Devin models | `internal/registry` |  |  |
| v7.3.0 | `bf06746d42d2` | 2026-09-12 | feat(devin): map model aliases and swe-1-7/haiku/sonnet/gpt-4-1 UIDs | `internal/runtime` |  |  |
| v7.3.0 | `ca664c6ede12` | 2026-09-12 | feat(devin): clamp maxTokens to model MaxCompletionTokens | `internal/runtime` |  |  |
| v7.3.0 | `2683ec201dde` | 2026-09-12 | feat(cmd): add fetch_devin_models CLI tool for dynamic model catalog extraction | `cmd/fetch_devin_models`, `internal/runtime` |  |  |
| v7.3.0 | `469aa3678fc6` | 2026-09-12 | feat(devin): parse protobuf timestamp and harden partial failure logging | `internal/runtime` |  |  |
| v7.3.0 | `16cb6c0b02fb` | 2026-09-12 | feat(devin): add symmetric decoded upstream response in request log | `internal/runtime` |  |  |
| v7.3.0 | `61741744889d` | 2026-09-12 | feat(devin): auto-namespace model IDs from clean devin_models.json | `internal/registry`, `internal/util` |  |  |
| v7.3.0 | `982cd124f9fe` | 2026-09-12 | feat(devin): add standalone devin_models.json catalog and remote updater | `cmd/server`, `internal/registry` |  |  |
| v7.3.0 | `59df75d20c70` | 2026-09-12 | docs(devin): update DevinExecutor comment with verbatim reconstructed cloud system prompt | `internal/runtime` |  |  |
| v7.3.0 | `0c2351bb896d` | 2026-09-12 | feat(devin): support none thinking level for glm-5-2 | `internal/registry`, `internal/runtime` |  |  |
| v7.3.0 | `30b2ac8996ea` | 2026-09-13 | refactor(devin): deduplicate auth credentials extraction, filter sparse tool calls, and optimize model lookup | `internal/registry`, `internal/runtime` |  |  |
| v7.3.0 | `9d190f309cab` | 2026-09-13 | feat(devin): integrate global signature detector for cross-provider signature detection | `internal/runtime` |  |  |
| v7.3.0 | `fb2c1c1afa8e` | 2026-09-13 | fix(devin): transport reuse, interleaved stream steps, strings.Builder panic, and updater URL | `internal/registry`, `internal/runtime`, `sdk/auth` |  |  |
| v7.3.0 | `926450e87b5c` | 2026-09-13 | fix(devin): dynamic catalog-driven chat_model_uid resolution and effort clamping | `internal/registry`, `internal/runtime` |  |  |
| v7.3.0 | `1604cb03347b` | 2026-09-13 | fix(devin): normalize upstream internal errors to 502 Bad Gateway | `internal/runtime` |  |  |
| v7.3.0 | `98b106f0e8fc` | 2026-09-13 | fix(devin): store transient quota metrics strictly in Quota.Signals and keep Metadata static | `internal/registry`, `internal/runtime`, `sdk/auth` |  |  |
| v7.3.0 | `6a239f57151a` | 2026-09-13 | fix(devin): propagate stream chunk errors, fix truncated protobuf infinite loop, respect ctx roundtripper, and | `internal/auth`, `internal/runtime` |  |  |
| v7.3.0 | `86de823daa50` | 2026-09-13 | perf(devin): cache Devin HTTP transports per proxy URL to reuse connection pools | `internal/runtime` |  |  |
| v7.3.0 | `6df8f3227815` | 2026-09-13 | perf(devin): eliminate O(K^2) tool call argument string allocations using strings.Builder | `internal/runtime` |  |  |
| v7.3.0 | `6c7d2d57f711` | 2026-09-13 | perf(devin): cache sensitive word regex matcher, preallocate request bytes and frame decompression buffer | `internal/runtime` |  |  |
| v7.3.0 | `5d0c77cf3fa7` | 2026-09-13 | fix(devin): enforce Connect-RPC EOS trailer invariant, validate frame flag, and bind OAuth callback to ctx | `internal/auth`, `internal/runtime`, `sdk/auth` |  |  |
| v7.3.0 | `50dd582641fd` | 2026-09-13 | fix(devin): update streaming tool call step metadata if arriving in subsequent frames | `internal/runtime` |  |  |
| v7.3.0 | `d754298a8b65` | 2026-09-13 | fix(devin): bound tool call indices against OOM, set Refresh timeout, and check manual callback state | `internal/runtime`, `sdk/auth` |  |  |
| v7.3.0 | `2f2f9b438112` | 2026-09-13 | fix(devin): guard against reopening thought step on late-arriving signatures | `internal/runtime` |  |  |
| v7.3.0 | `a5ea971f358f` | 2026-09-13 | fix(devin): sort streaming tool call stop events and normalize prompt CRLF | `internal/runtime` |  |  |
| v7.3.0 | `f1f5506c0b49` | 2026-09-13 | fix(devin): align wire protocol, harden streaming, and resolve multi-turn tool/signature parity | `internal/auth`, `internal/cache`, `internal/registry` +2 |  |  |
| v7.3.0 | `94d6eb535eb9` | 2026-09-13 | docs(config): document payload filter examples for codex tools | `config.example.yaml`, `internal/api`, `internal/runtime` |  |  |
| v7.3.0 | `1ca975dfc011` | 2026-09-13 | feat(cooldowns): add cooldown snapshot feature for management auth files | `internal/api`, `sdk/cliproxy` |  |  |
| v7.3.0 | `f702bc1ac263` | 2026-09-13 | feat(codex): preserve native fidelity for responses-lite requests | `internal/runtime`, `internal/util`, `sdk/api` |  |  |
| v7.3.0 | `e696ea47c5ee` | 2026-09-13 | feat(codex): forward X-Codex-Turn-State header in executor requests | `internal/runtime` |  |  |
| v7.3.1 | `294b7f5b191b` | 2026-09-13 | feat(models): expose CPA web search capability | `internal/api`, `internal/client` |  |  |
| v7.3.1 | `44e62bc8acc2` | 2026-09-14 | feat(devin): implement Devin OAuth flow with callback handling and session management | `docs/management-devin-oauth.md`, `internal/api`, `internal/auth` +1 |  |  |
| v7.3.1 | `678da56193fb` | 2026-09-14 | chore(models): sync verified native search capability metadata | `internal/registry` |  |  |
| v7.3.1 | `4311ae874774` | 2026-09-14 | feat(models): require explicit per-model native search support | `internal/api`, `internal/client`, `internal/registry` +1 |  |  |
| v7.3.2 | `db0b957c4831` | 2026-09-13 | fix(devin): reject case-insensitive duplicate model IDs | `internal/registry` |  |  |
| v7.3.2 | `014833107299` | 2026-09-13 | test(devin): add unit test for devin oauth-model-alias channel routing | `sdk/cliproxy` |  |  |
| v7.3.2 | `b4749cb204b4` | 2026-09-13 | fix(devin): parse field 8 header submessages, accumulate field 4 prompt tokens, and add field 28 usage fallbac | `internal/runtime` |  |  |
| v7.3.2 | `cca35aee9302` | 2026-09-14 | fix(devin): use loopback callback endpoint for oauth redirect uri | `docs/management-devin-oauth.md`, `internal/api` |  |  |
| v7.3.2 | `5f56ce928ecd` | 2026-09-14 | fix(devin): trigger dimension group fallback if any usage metric is zero | `internal/runtime` |  |  |
| v7.3.2 | `4c331bb9532f` | 2026-09-14 | fix(devin): unwrap repeated field 28 groups, merge partial field 7 usage, and harden APICall escaping | `internal/api`, `internal/registry`, `internal/runtime` |  |  |
| v7.3.2 | `0719520f2aed` | 2026-09-14 | feat(api-call): support $TOKEN$ replacement in request body data | `internal/api`, `internal/pluginhost`, `internal/runtime` |  |  |
| v7.3.2 | `8c5f6e185f4d` | 2026-09-14 | fix(openai): convert responses tool choice to chat completions format | `internal/translator` |  |  |
| v7.3.3 | `cb73cd993611` | 2026-09-12 | fix(codex): buffer keepalive and empty item announcements during codex bootstrap | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v7.3.3 | `748d5767310a` | 2026-09-14 | feat(pluginhost): propagate forced provider and auth ID in host model execution | `internal/pluginhost`, `sdk/api`, `sdk/pluginapi` |  |  |
| v7.3.3 | `ca929459f987` | 2026-09-14 | test(openai): add tests for responses websocket compaction replay and routing | `sdk/api` |  |  |
| v7.3.3 | `6ba444557c1e` | 2026-09-14 | feat(pluginabi): support HTTP status code propagation in plugin error envelopes | `examples/plugin`, `internal/pluginhost`, `sdk/pluginabi` |  |  |
| v7.3.3 | `b8477c7181c7` | 2026-09-14 | fix(pluginhost): keep request and method buffers alive during native call | `internal/pluginhost` |  |  |
| v7.3.3 | `13af6c002bd0` | 2026-09-14 | fix(discovery): tighten scan timeout, flags, and TCP-only types | `cmd/server`, `internal/cmd`, `internal/discovery` +1 |  |  |
| v7.3.3 | `5f74accd0e83` | 2026-09-14 | fix(devin): share manual paste parsing and surface oauth callback errors | `internal/auth`, `sdk/auth` |  |  |
| v7.3.3 | `9e847e596e3b` | 2026-09-14 | fix(discovery): uniquify instance names without a startup browse | `config.example.yaml`, `internal/discovery` |  |  |
| v7.3.3 | `2dd2fd6d05ad` | 2026-09-14 | fix(discovery): load default scan config and sanitize hostnames | `internal/cmd` |  |  |
| v7.3.3 | `7d687054329d` | 2026-09-14 | fix(discovery): keep JSON scans clean and honor interface filters | `cmd/server`, `internal/cmd` |  |  |
| v7.3.3 | `20ec9b83a120` | 2026-09-14 | fix(discovery): close lifecycle and routing gaps | `cmd/server`, `internal/cmd`, `internal/discovery` +1 |  |  |
| v7.3.3 | `c3a5e8c0060a` | 2026-09-14 | test(discovery): report temp cleanup failures | `internal/discovery` |  |  |
| v7.3.3 | `6e307553f43f` | 2026-09-14 | feat(codex): add optional time ceiling for stream bootstrap buffering | `config.example.yaml`, `internal/config`, `internal/runtime` +1 |  |  |
| v7.3.3 | `d48590a47d78` | 2026-09-14 | feat(models): add gemini-3.5-flash-lite model definition | `internal/registry` |  |  |
| v7.3.3 | `f341e07b949a` | 2026-09-14 | fix(cmd/fetch_antigravity_models): support symlinked auth dir and multi-auth fallback | `cmd/fetch_antigravity_models` |  |  |
| v7.3.3 | `f5c19d25bb4e` | 2026-09-14 | fix(discovery): bound browse resource usage | `internal/discovery` |  |  |
| v7.3.3 | `c1b7c91f2f8a` | 2026-09-14 | fix(discovery): honor caller cancellation and reload endpoint | `config.example.yaml`, `internal/cmd`, `internal/config` +2 |  |  |
| v7.3.3 | `b9005770e65a` | 2026-09-14 | fix(discovery): harden advertiser lifecycle and browse limits | `cmd/server`, `internal/discovery`, `sdk/cliproxy` |  |  |
| v7.3.3 | `2bcebaa89c98` | 2026-09-14 | fix(claude): repair tool call pairing and handle standalone tool outputs | `internal/translator` |  |  |
| v7.3.3 | `f465ebdddef6` | 2026-09-14 | test(devin): route status calls to mock server in headless auth tests | `sdk/auth` |  |  |
| v7.3.3 | `fd3cd1516173` | 2026-09-14 | docs(readme): remove FastAIToken sponsor | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.3.3 | `c1cb0c5de1cf` | 2026-09-14 | fix(translator): preserve html characters in tool arguments and fix devin sequential tool calls | `internal/runtime`, `internal/translator` |  |  |
| v7.3.3 | `3428110d49be` | 2026-09-14 | feat(discovery): add LAN gateway discovery | `cmd/server`, `config.example.yaml`, `go.mod` +6 |  |  |
| v7.3.3 | `c35db127b059` | 2026-09-14 | fix(devin): mock AuthService in headless token test and validate prompt before printing URL | `sdk/auth` |  |  |
| v7.3.3 | `09807c57ea8e` | 2026-09-14 | fix(devin): align OAuth authorization URL parameter order with official CLI binary | `internal/auth` |  |  |
| v7.3.3 | `fe2fdde8a8ee` | 2026-09-14 | feat(devin): support port-free manual code login in no-browser mode | `internal/auth`, `sdk/auth` |  |  |
| v7.3.3 | `7bbfeaf8a7ac` | 2026-09-15 | feat(executor): promote reasoning content as summary and sanitize inputs | `internal/runtime` |  |  |
| v7.3.3 | `8c984672a66a` | 2026-09-15 | feat(translator): handle orphan function outputs as user text in OpenAI and Gemini translators | `internal/translator` |  |  |
| v7.3.3 | `bef1f65c6c1d` | 2026-09-15 | feat(auth): add retry logic for pre-HTTP transport failures and related tests | `sdk/cliproxy` |  |  |
| v7.3.3 | `1fac8cc0c8ce` | 2026-09-15 | feat(executor): strip unsupported `id` fields from Gemini interactions payload | `internal/runtime`, `internal/translator` |  |  |
| v7.3.3 | `e3cbe437d00b` | 2026-09-15 | feat(auth): add tests to ensure async operations don’t block unrelated actions | `internal/api`, `sdk/cliproxy` |  |  |
