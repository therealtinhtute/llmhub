# Upstream ledger — cliproxyapi v7.2.147..v8.0.28

- generated: 2026-10-14T08:45:43Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `b923e639f075`
- non-merge commits: 654

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
| v7.3.10 | `ddc3f731f45a` | 2026-09-20 | feat(steering): add documentation for Codex response steering and duplex transport | `docs/STEERING.md` |  |  |
| v7.3.10 | `e56547f39d57` | 2026-09-20 | fix(auth): avoid unnecessary scheduler rebuilds by checking sync needs from auth map | `internal/registry`, `sdk/cliproxy` |  |  |
| v7.3.10 | `33ae35d53a23` | 2026-09-20 | docs: add international Kimi Code link alongside domestic subscription link | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.3.10 | `c52ca7bd4e0b` | 2026-09-20 | docs(claude): clarify patch version floor rationale for native passthrough | `internal/runtime` |  |  |
| v7.3.10 | `83a4913aa49c` | 2026-09-20 | fix(claude): accept newer patch releases as native clients (#5820) | `config.example.yaml`, `internal/runtime` |  |  |
| v7.3.10 | `cdfb79ef843f` | 2026-09-20 | feat(translator): support builtin tools in gemini and antigravity interactions | `internal/translator` |  |  |
| v7.3.10 | `a5ab69521f7b` | 2026-09-21 | feat: add full support for kimi.ai oauth and runtime execution | `cmd/server`, `internal/api`, `internal/auth` +8 |  |  |
| v7.3.10 | `28100e54b9b5` | 2026-09-21 | feat(auth): propagate context compaction and node kind session metadata | `internal/home`, `internal/logging`, `internal/redisqueue` +1 |  |  |
| v7.3.10 | `563865e77adb` | 2026-09-21 | test: fix timing synchronization and channel races in streaming tests | `internal/runtime`, `sdk/api` |  |  |
| v7.3.10 | `40cc6489879a` | 2026-09-21 | fix(translator): preserve reasoning content across tool turns in responses conversion | `internal/translator` |  |  |
| v7.3.11 | `ffe6ad3c5fcf` | 2026-09-21 | fix(gemini): enforce array type for schema nodes declaring items | `internal/util` |  |  |
| v7.3.11 | `ac3849e5d981` | 2026-09-21 | feat(pluginapi): propagate response model, service tier, and stream flag to usage plugins | `internal/pluginhost`, `sdk/pluginapi` |  |  |
| v7.3.11 | `dd013f9e2993` | 2026-09-21 | fix(responses): filter upstream private and telemetry events in SSE streams | `sdk/api` |  |  |
| v7.3.11 | `7b6fafce1b32` | 2026-09-21 | fix(responses): support mid-connection prewarm in websocket handler | `sdk/api` |  |  |
| v7.3.11 | `50585bf208a8` | 2026-09-21 | refactor(translator): encapsulate tool-result cache-control hoisting in common | `internal/translator` |  |  |
| v7.3.11 | `bcd13ca91c87` | 2026-09-21 | fix(translator): hoist tool-result content part cache_control to the block (#5432) | `internal/translator` |  |  |
| v7.3.11 | `ed751ea08af8` | 2026-09-21 | fix(claude): gate fallback-credit beta on fallback tokens or explicit fallbacks | `internal/runtime` |  |  |
| v7.3.11 | `fd5cd228a804` | 2026-09-21 | test(executor): allow non-negative TTFT in xAI usage record assertions | `internal/runtime` |  |  |
| v7.3.12 | `56518489ce92` | 2026-09-09 | perf(codex): batch tool schema rewrites | `internal/runtime` |  |  |
| v7.3.12 | `cbf8318315a1` | 2026-09-09 | perf(codex): avoid discarded copies of SSE data lines | `internal/runtime` |  |  |
| v7.3.12 | `8a6a39684d11` | 2026-09-20 | perf: index Responses tools and cache stream request validation | `internal/runtime`, `internal/translator` |  |  |
| v7.3.12 | `2eb8dd11d248` | 2026-09-22 | fix(gemini): preserve items for uppercase ARRAY type in schema sanitizer | `internal/util` |  |  |
| v7.3.12 | `a26cf2a8c2e5` | 2026-09-22 | perf(executor): avoid discarded copies of Gemini and Kimi stream lines | `internal/runtime` |  |  |
| v7.3.12 | `639b7f1126b4` | 2026-09-22 | perf(executor): reuse identical Claude and Gemini request translations | `internal/runtime`, `sdk/translator` |  |  |
| v7.3.12 | `d582067c066f` | 2026-09-22 | feat(executor): support execution-scoped request proxy overrides | `internal/pluginhost`, `internal/runtime`, `sdk/api` +3 |  |  |
| v7.3.12 | `b9b50a83cb9d` | 2026-09-22 | fix(models): set grok-4.7 max completion to the context window | `internal/registry` |  |  |
| v7.3.12 | `bf44a7f89206` | 2026-09-22 | style: gofmt codex bootstrap and cache-control tests | `internal/runtime`, `internal/translator` |  |  |
| v7.3.12 | `94b7cc2ee0f0` | 2026-09-22 | chore(models): drop gpt-5.3-codex-spark from the embedded catalog | `internal/registry` |  |  |
| v7.3.12 | `130c879206ba` | 2026-09-22 | feat(models): add grok-4.7 model definition | `internal/registry` |  |  |
| v7.3.13 | `cc77410866c2` | 2026-09-22 | feat(management): support updating priority in credential patch endpoints | `internal/api` |  |  |
| v7.3.13 | `f351924f42cb` | 2026-09-22 | feat(codex): add support for disable codex cloaking per credential | `config.example.yaml`, `internal/api`, `internal/config` +3 |  |  |
| v7.3.13 | `4b5adbbe9a05` | 2026-09-22 | feat(openai): support video input in responses request translation | `internal/runtime`, `internal/translator` |  |  |
| v7.3.13 | `ed70aeaa1627` | 2026-09-22 | fix(registry): avoid double-counting clients with active quota and suspension | `internal/registry`, `sdk/cliproxy` |  |  |
| v7.3.13 | `b989e34881c7` | 2026-09-22 | fix(claude): support string input in responses request translation | `internal/translator` |  |  |
| v7.3.13 | `320100ecf767` | 2026-09-22 | fix(codex): strip octal NUL pattern escapes from tool schemas | `internal/runtime`, `internal/util` |  |  |
| v7.3.13 | `555662940411` | 2026-09-22 | docs(README_CN): fix FluxA AgenticPlan | `README_CN.md` |  |  |
| v7.3.13 | `2430354330af` | 2026-09-23 | feat(registry): update model definitions and bump codex client version to 0.155.0 | `cmd/fetch_codex_models`, `internal/registry` |  |  |
| v7.3.14 | `937ebb8f11f7` | 2026-09-23 | feat(registry): add gpt-6-luna model to codex-free | `internal/registry` |  |  |
| v7.3.14 | `a962b77d4383` | 2026-09-23 | feat(server): support trusted proxies configuration for client IP resolution | `config.example.yaml`, `internal/api`, `internal/config` +3 |  |  |
| v7.3.15 | `6ed58a7c5548` | 2026-09-22 | docs: add CLIProxy Quota Tray to the "Who is with us?" list | `README.md` |  |  |
| v7.3.15 | `673131f57484` | 2026-09-23 | fix(codex): compact client model catalog and preserve required fields | `internal/api`, `internal/client`, `sdk/api` |  |  |
| v7.3.15 | `bd584a752329` | 2026-09-23 | feat(claude): support Claude Code 2.1.280 feature-gated betas | `internal/runtime` |  |  |
| v7.3.15 | `779bf317e030` | 2026-09-23 | feat(claude): bump claude cli baseline to 2.1.280 and support tool changes beta | `config.example.yaml`, `internal/runtime` |  |  |
| v7.3.15 | `fc914b9debb9` | 2026-09-23 | feat(registry): add grok-4.7-build-fast model and update grok-4.7 | `internal/registry` |  |  |
| v7.3.15 | `e01806f971b1` | 2026-09-23 | docs(readme): add CLIProxy Quota Tray to community projects | `README_CN.md`, `README_JA.md` |  |  |
| v7.3.16 | `5af6cd7582ee` | 2026-09-23 | fix(gemini): stringify function responses containing JSON schema $ref | `internal/translator` |  |  |
| v7.3.16 | `2fe9932bb9d1` | 2026-09-23 | feat(usage): track execution request ID and trace ID across usage records | `internal/pluginhost`, `internal/redisqueue`, `internal/runtime` +2 |  |  |
| v7.3.16 | `e6fcfa34bad1` | 2026-09-23 | fix(logging): prevent request log collisions and select latest log by ID | `internal/api`, `internal/logging` |  |  |
| v7.3.16 | `65459cace01f` | 2026-09-23 | feat(registry): update devin model definitions | `internal/registry` |  |  |
| v7.3.16 | `580df95a946b` | 2026-09-23 | fix(antigravity): sanitize Claude tool IDs and resolve collisions | `internal/translator` |  |  |
| v7.3.16 | `c404af96ebac` | 2026-09-24 | fix(auth): preserve file priority across plugin auth refreshes | `internal/api`, `internal/pluginhost`, `internal/watcher` +2 |  |  |
| v7.3.16 | `781a203b99fa` | 2026-09-24 | fix(claude): merge adjacent text blocks into single assistant message | `internal/translator` |  |  |
| v7.3.16 | `7d888bf3cac8` | 2026-09-24 | feat(auth): persist meta mint subscription metadata in auth record | `sdk/auth` |  |  |
| v7.3.16 | `855a722349ff` | 2026-09-24 | feat(pluginhost): add host http operation bridge and lifecycle scoping | `examples/plugin`, `internal/pluginhost`, `sdk/pluginabi` |  |  |
| v7.3.16 | `aafa4e9549f7` | 2026-09-24 | fix(auth): isolate credential-scoped quota cooldown from model quota state | `sdk/cliproxy` |  |  |
| v7.3.17 | `b97f71da0cf0` | 2026-09-23 | fix(codex): send the ChatGPT routing hint native Codex sends | `internal/runtime` |  |  |
| v7.3.17 | `74c1ebd37f47` | 2026-09-24 | fix(registry): keep request context active until response body is read | `internal/registry` |  |  |
| v7.3.17 | `759c57c2612d` | 2026-09-24 | fix(api): immediately close connections on stop and guard state access | `internal/api`, `sdk/cliproxy` |  |  |
| v7.3.17 | `377747b8b661` | 2026-09-24 | fix(registry,cliproxy): set xai owned_by for Devin grok models and align Antigravity web search tests | `cmd/fetch_devin_models`, `internal/registry`, `internal/runtime` +1 |  |  |
| v7.3.17 | `c7f358d4d46f` | 2026-09-24 | fix(executor,auth): support Session_id header in Antigravity replay and remove redundant stream chunk logging | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.3.17 | `107d7f76c9b4` | 2026-09-24 | fix(translator,auth): avoid synthetic bypass on unsigned text parts and reduce stream rewrite log noise | `internal/pluginhost`, `internal/translator`, `sdk/cliproxy` |  |  |
| v7.3.17 | `8222e328c02e` | 2026-09-24 | Enhance logging and improve thought signature handling (#6106) | `internal/runtime`, `internal/signature`, `internal/translator` |  |  |
| v7.3.17 | `75b854eb2e63` | 2026-09-24 | fix(translator): sanitize tool names for Claude and inject fallback schema for parameterless tools | `internal/translator`, `internal/util` |  |  |
| v7.3.17 | `f7c738dad38b` | 2026-09-24 | fix(claude): align 2.1.280 fingerprint and thinking visibility (#6096) | `internal/runtime`, `internal/thinking`, `internal/translator` +2 |  |  |
| v7.3.17 | `dd3b657b5d92` | 2026-09-24 | fix(codex): use resolved base model for routing hint | `internal/runtime` |  |  |
| v7.3.17 | `6d1d74b30d77` | 2026-09-24 | refactor(executor): remove unsupported reasoning effort handling and update related tests | `internal/runtime` |  |  |
| v7.3.17 | `984aee8df05e` | 2026-09-24 | feat(executor): enhance sanitizeXAIResponsesBody to utilize resolved thinking support | `internal/runtime` |  |  |
| v7.3.17 | `9bdde54b59d1` | 2026-09-25 | fix(meta): fix client id header and preserve subscription metadata | `internal/api`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.3.17 | `1dffddf4ac8c` | 2026-09-25 | fix(session): avoid rescanning large payloads during session extraction | `sdk/cliproxy` |  |  |
| v7.3.17 | `c9a06ba6f297` | 2026-09-25 | fix(interceptors): avoid cloning request body for read-only interceptors | `internal/pluginhost`, `sdk/api`, `sdk/cliproxy` +1 |  |  |
| v7.3.18 | `58461c8f476a` | 2026-09-24 | fix(auth): unschedule disabled invalid_grant credentials and apply exponential backoff for enabled invalid_gra | `sdk/cliproxy` |  |  |
| v7.3.18 | `4332048dda19` | 2026-09-25 | fix(claude): apply cloaking by default for direct Messages OAuth requests | `internal/runtime` |  |  |
| v7.3.18 | `5530cbd2a5b0` | 2026-09-25 | fix(auth): preserve attributes and path across plugin refreshes | `internal/api`, `internal/pluginhost`, `internal/watcher` |  |  |
| v7.3.18 | `60325f472a2e` | 2026-09-25 | fix(devin): ensure swe-1-6-slow is preserved as a builtin model | `internal/registry`, `sdk/cliproxy` |  |  |
| v7.3.18 | `ed980be34b99` | 2026-09-26 | fix(auth): preserve resolved model capabilities and configuration update support | `config.example.yaml`, `internal/api`, `internal/config` +5 |  |  |
| v7.3.19 | `be4b28f16568` | 2026-09-26 | feat(codex): support service tier in OpenAI request translation | `internal/translator` |  |  |
| v7.3.19 | `3b2882b72eee` | 2026-09-26 | fix(codex): strip non-standard multi-agent metadata for compat models | `internal/client`, `internal/runtime` |  |  |
| v7.3.19 | `6cd86fb272b4` | 2026-09-26 | feat(auth): support terminal rejection in plugin scheduler | `internal/pluginhost`, `sdk/cliproxy`, `sdk/pluginapi` |  |  |
| v7.3.19 | `3de5709db14c` | 2026-09-26 | fix(claude): preserve tool result adjacency for Antigravity requests | `internal/translator`, `internal/util` |  |  |
| v7.3.19 | `b4e5cc019d07` | 2026-09-26 | fix(thinking): log effective effort and baseline for native Responses requests | `internal/thinking` |  |  |
| v7.3.19 | `12245ad38884` | 2026-09-26 | fix(thinking): log baseline configuration for native Responses requests | `internal/thinking` |  |  |
| v7.3.19 | `6dea3dfa13a9` | 2026-09-27 | fix(antigravity): extract and apply reasoning summary config for OpenAI responses | `internal/runtime`, `internal/translator` |  |  |
| v7.3.19 | `fc2c98d1ca2a` | 2026-09-27 | feat(runtime): enhance Devin model aggregation and UID normalization | `cmd/fetch_devin_models`, `internal/registry`, `internal/runtime` |  |  |
| v7.3.2 | `db0b957c4831` | 2026-09-13 | fix(devin): reject case-insensitive duplicate model IDs | `internal/registry` |  |  |
| v7.3.2 | `014833107299` | 2026-09-13 | test(devin): add unit test for devin oauth-model-alias channel routing | `sdk/cliproxy` |  |  |
| v7.3.2 | `b4749cb204b4` | 2026-09-13 | fix(devin): parse field 8 header submessages, accumulate field 4 prompt tokens, and add field 28 usage fallbac | `internal/runtime` |  |  |
| v7.3.2 | `cca35aee9302` | 2026-09-14 | fix(devin): use loopback callback endpoint for oauth redirect uri | `docs/management-devin-oauth.md`, `internal/api` |  |  |
| v7.3.2 | `5f56ce928ecd` | 2026-09-14 | fix(devin): trigger dimension group fallback if any usage metric is zero | `internal/runtime` |  |  |
| v7.3.2 | `4c331bb9532f` | 2026-09-14 | fix(devin): unwrap repeated field 28 groups, merge partial field 7 usage, and harden APICall escaping | `internal/api`, `internal/registry`, `internal/runtime` |  |  |
| v7.3.2 | `0719520f2aed` | 2026-09-14 | feat(api-call): support $TOKEN$ replacement in request body data | `internal/api`, `internal/pluginhost`, `internal/runtime` |  |  |
| v7.3.2 | `8c5f6e185f4d` | 2026-09-14 | fix(openai): convert responses tool choice to chat completions format | `internal/translator` |  |  |
| v7.3.20 | `ef9e71201e0c` | 2026-09-27 | feat(tui): support remote management base URL for client mode | `cmd/server`, `config.example.yaml`, `internal/config` +2 |  |  |
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
| v7.3.4 | `e475807a96c9` | 2026-09-03 | feat(meta): add Meta API key management handlers, routes, logo, and tests | `assets/logo`, `internal/api`, `sdk/auth` |  |  |
| v7.3.4 | `18385de00193` | 2026-09-03 | fix(tui): rename Meta provider label from Meta (Muse) to Meta | `internal/tui` |  |  |
| v7.3.4 | `78c14b805a58` | 2026-09-03 | fix(management): disable caching for management.html to prevent stale SPA views | `internal/api` |  |  |
| v7.3.4 | `23c16e2985bb` | 2026-09-03 | feat(meta): add Meta OAuth management endpoint and WebUI / TUI support | `internal/api`, `internal/tui` |  |  |
| v7.3.4 | `30191c3ad6f9` | 2026-09-03 | feat(meta): mint LLM API key from device access token via /muse-code/key | `internal/auth`, `internal/runtime`, `sdk/auth` |  |  |
| v7.3.4 | `fbf746459b06` | 2026-09-03 | fix(cli): make config optional in commandMode so login works without existing config.yaml | `cmd/server` |  |  |
| v7.3.4 | `54d4f4c0193c` | 2026-09-03 | feat(meta): add native Meta (Muse Code) provider and OAuth device flow | `cmd/server`, `config.example.yaml`, `internal/api` +9 |  |  |
| v7.3.4 | `cee799f61af3` | 2026-09-04 | fix(meta): address review feedback on credential lifecycle, models, and config loading | `internal/api`, `internal/auth`, `internal/config` +4 |  |  |
| v7.3.4 | `4a0131c062cd` | 2026-09-05 | fix(meta): unify credential minting and preserve auth lifecycle | `internal/api`, `internal/auth`, `internal/runtime` +1 |  |  |
| v7.3.4 | `06660dd6f47b` | 2026-09-05 | fix(meta): validate catalog section, singleflight management mints, and synchronize singleflight test | `internal/api`, `internal/registry`, `internal/runtime` |  |  |
| v7.3.4 | `1144ae707bff` | 2026-09-05 | fix(meta): require OAuth storage for DCA tokens and reject them in config meta-api-key | `internal/api`, `internal/config`, `internal/runtime` |  |  |
| v7.3.4 | `d09042a54810` | 2026-09-05 | fix(meta): implement request preparer, disable unconditional refresh, scope 429 quota, and stabilize auth inde | `internal/runtime`, `sdk/auth`, `sdk/cliproxy` |  |  |
| v7.3.4 | `be7323f3bf66` | 2026-09-05 | fix(meta): recover and persist minted credentials correctly | `internal/api`, `internal/auth`, `internal/config` +2 |  |  |
| v7.3.4 | `21aa46b60347` | 2026-09-05 | chore(meta): keep provider PR scoped to Meta support | `cmd/server`, `config.example.yaml`, `internal/api` +2 |  |  |
| v7.3.4 | `47cc31aed833` | 2026-09-05 | fix(meta): exclude cleared expired timestamp on persist, add runtime alias resolution, and fix test race | `internal/auth`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.3.4 | `07e85de1a474` | 2026-09-14 | fix(plugin-quota): preserve valid mapped summaries | `go.mod`, `internal/api` |  |  |
| v7.3.4 | `6aea72e2948b` | 2026-09-14 | fix(plugin-quota): preserve mapped summaries | `internal/api` |  |  |
| v7.3.4 | `617b9fd17db9` | 2026-09-14 | fix(plugin-quota): validate summary metadata | `internal/api` |  |  |
| v7.3.4 | `d3cb685944a6` | 2026-09-14 | fix(plugin-quota): validate summary identifiers | `internal/api` |  |  |
| v7.3.4 | `782a5cd16991` | 2026-09-14 | fix(plugin-quota): tolerate malformed summaries | `internal/api` |  |  |
| v7.3.4 | `7b60be35164a` | 2026-09-14 | fix(plugin-quota): require summary metric values | `internal/api` |  |  |
| v7.3.4 | `5cb41e233f60` | 2026-09-14 | fix(plugin-quota): accept summary-only probes | `internal/api` |  |  |
| v7.3.4 | `acb0eae2eba9` | 2026-09-14 | feat(plugin-quota): add typed summary metrics | `internal/api`, `sdk/pluginapi` |  |  |
| v7.3.4 | `8335eac73194` | 2026-09-15 | feat(config): add Meta OAuth aliasing and error rules support | `config.example.yaml`, `internal/config`, `internal/watcher` +1 |  |  |
| v7.3.4 | `65348b959425` | 2026-09-15 | feat(executor): add native Meta (Muse Code) provider integration | `internal/registry`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.3.4 | `772c63c8e2e2` | 2026-09-15 | feat(translator): refine tool call argument validation for finish reason logic | `internal/translator` |  |  |
| v7.3.5 | `f51d3ae93b87` | 2026-09-14 | fix(interactions): strip invalid id from function_result and redundant call_id from function_call | `internal/runtime`, `internal/translator` |  |  |
| v7.3.5 | `7c32971b91c8` | 2026-09-16 | fix(executor): prevent stream failure on client disconnect after claude completion | `internal/runtime` |  |  |
| v7.3.5 | `d44901f91c3b` | 2026-09-16 | fix(devin): classify high-demand errors as rate limits | `internal/runtime` |  |  |
| v7.3.5 | `7def842554bc` | 2026-09-16 | feat(translator): preserve tool names in tool result messages | `internal/translator` |  |  |
| v7.3.5 | `ef63d2e7fa8a` | 2026-09-16 | feat(antigravity): support web search translation and URL resolution in OpenAI Responses | `internal/registry`, `internal/runtime`, `internal/translator` |  |  |
| v7.3.5 | `dea4ce8aeb1d` | 2026-09-16 | fix(devin): support swe-1-6-slow model variant | `internal/registry`, `internal/runtime` |  |  |
| v7.3.5 | `e3e97ad9be0e` | 2026-09-16 | feat(translator): introduce namespace-aware tool name capping and collision handling | `internal/translator` |  |  |
| v7.3.5 | `e54a8e97fff8` | 2026-09-16 | feat(translator): enhance cache usage handling and input token computation logic | `internal/translator` |  |  |
| v7.3.5 | `6f908cbcff28` | 2026-09-16 | feat(translator): enhance finish reason handling and tool call translations | `internal/runtime`, `internal/translator` |  |  |
| v7.3.5 | `6724a95851f9` | 2026-09-16 | feat(auth): enhance retry logic with enriched error handling and tests | `sdk/api`, `sdk/cliproxy` |  |  |
| v7.3.5 | `b681a1e0f7b8` | 2026-09-17 | feat(translator): wrap demoted mid-session system messages in system-reminder envelope | `internal/translator` |  |  |
| v7.3.5 | `c2bb91d2cb9c` | 2026-09-17 | fix(devin): buffer content deltas to handle late thinking signatures | `internal/runtime` |  |  |
| v7.3.5 | `77820cb2f46d` | 2026-09-17 | feat(translator): support alternative reasoning fields in OpenAI Claude translation | `config.example.yaml`, `internal/translator` |  |  |
| v7.3.5 | `7fcbdf889675` | 2026-09-17 | feat(translator): enhance web search streaming and citation mapping in OpenAI Responses | `internal/runtime`, `internal/translator` |  |  |
| v7.3.6 | `923a8c30dcb3` | 2026-09-04 | fix(codex): recognize codex_exec user agent for multi-agent v2 optimization | `internal/client` |  |  |
| v7.3.6 | `7c961b34fd55` | 2026-09-09 | docs: add PiCloud to related projects | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.3.6 | `cfeeeb341884` | 2026-09-10 | docs(readme): update links after GitHub username change | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.3.6 | `ad088a879507` | 2026-09-17 | fix(devin): filter automation update tools and sanitize tool descriptions | `internal/runtime`, `internal/translator`, `internal/util` |  |  |
| v7.3.6 | `a9e92b81453f` | 2026-09-17 | feat(translator): preserve model metadata in requests | `internal/client`, `internal/runtime`, `internal/translator` +2 |  |  |
| v7.3.6 | `46f6cb01976c` | 2026-09-17 | docs: document CLIProxyAPIHome integration areas | `AGENTS.md` |  |  |
| v7.3.6 | `45160378200d` | 2026-09-17 | docs: add cc-status-line to community projects | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.3.6 | `f668ac417d97` | 2026-09-17 | fix(schema): strip unsupported schema identifier keywords | `internal/util` |  |  |
| v7.3.6 | `311efcb3a29e` | 2026-09-17 | test(executor): use metaUserAgent constant in meta executor test | `internal/runtime` |  |  |
| v7.3.6 | `c4982e846e16` | 2026-09-17 | fix(executor): strip relayed tool result images for text-only models | `internal/runtime` |  |  |
| v7.3.7 | `0b55053944aa` | 2026-09-17 | fix(management): reject unresolved token placeholders in api-call | `internal/api` |  |  |
| v7.3.7 | `b715526add0c` | 2026-09-17 | feat(plugin): support scheduling across priorities | `internal/pluginhost`, `sdk/cliproxy`, `sdk/pluginapi` +1 |  |  |
| v7.3.7 | `afba07ba265a` | 2026-09-17 | fix(config): preserve plugin configurations when saving yaml | `internal/config` |  |  |
| v7.3.7 | `64c9433fd2c2` | 2026-09-17 | fix(devin): support images in tool results | `internal/runtime`, `internal/translator` |  |  |
| v7.3.7 | `76ac75e68ae6` | 2026-09-17 | feat(management): paginate auth file listings | `internal/api` |  |  |
| v7.3.7 | `b773607e3e77` | 2026-09-18 | fix(ci): replace go-cross/cgo-actions with direct FreeBSD sysroot cross-compilation | `.github/workflows` |  |  |
| v7.3.7 | `c616193a6cf6` | 2026-09-18 | fix(xai): unify forced hosted tool choice normalization | `internal/runtime` |  |  |
| v7.3.7 | `44eaef0009f8` | 2026-09-18 | feat(claude): support model-level cooling and scope overage rate limits | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v7.3.7 | `b6fe4f20c4ea` | 2026-09-18 | fix(devin): handle orphaned tool results and normalize function result payloads | `internal/runtime` |  |  |
| v7.3.7 | `9e10db53ad89` | 2026-09-18 | fix(devin): aggregate tool calls by id and track cache write tokens | `internal/runtime` |  |  |
| v7.3.8 | `f86a33f72175` | 2026-09-18 | fix(executor): restrict claude advisor tool check to server tool use | `internal/runtime` |  |  |
| v7.3.8 | `c2ea2684099f` | 2026-09-18 | fix(translator): scope tool responses per turn to handle repeated tool call IDs | `internal/translator` |  |  |
| v7.3.8 | `cb62a6748b99` | 2026-09-18 | fix(openai): classify stream request timeout as server error | `sdk/api`, `test/codex_incomplete_stream_error_type_test.go` |  |  |
| v7.3.8 | `81d6ba774621` | 2026-09-18 | fix(codex): preserve reasoning content and IDs for compat models in responses | `internal/runtime` |  |  |
| v7.3.8 | `e84e248c51e5` | 2026-09-18 | fix(executor): record expected Devin upstream model and bound stream observer memory | `internal/runtime` |  |  |
| v7.3.8 | `cde7d57e44e6` | 2026-09-18 | fix(executor): robust response model observability across meta, kimi, and openai-compat streams | `internal/runtime` |  |  |
| v7.3.8 | `f8467f07dca5` | 2026-09-18 | fix(executor): record authentic Devin and Gemini Interactions response models | `internal/runtime` |  |  |
| v7.3.8 | `e9463ff5a795` | 2026-09-18 | feat(executor): extend response model recording and substitution warnings to all providers | `internal/runtime` |  |  |
| v7.3.8 | `0b9a91fb7871` | 2026-09-18 | feat: add FluxA | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.3.8 | `784285a4854d` | 2026-09-18 | feat: add PatewayAI | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.3.8 | `cc545cbf906b` | 2026-09-18 | fix(openai): align tool call messages and preserve ordering on ambiguous outputs | `internal/runtime`, `internal/translator`, `sdk/translator` |  |  |
| v7.3.8 | `25f40d8cf8df` | 2026-09-18 | feat(codex): record upstream response model and warn on silent model substitution | `internal/redisqueue`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.3.8 | `859c486512b7` | 2026-09-18 | fix(codex): normalize and support ultrafast service tier | `internal/translator` |  |  |
| v7.3.8 | `28743473c11a` | 2026-09-18 | feat(codex): append (Devin) suffix to Devin model display names | `internal/api`, `internal/client`, `sdk/api` |  |  |
| v7.3.8 | `660a5800e777` | 2026-09-18 | fix(xai): restore aliased client web search tool name in responses | `internal/runtime` |  |  |
| v7.3.8 | `3662d1535a8b` | 2026-09-18 | fix(codex): strip item-level and tool output prompt cache breakpoints | `internal/translator` |  |  |
| v7.3.8 | `1cce9325738f` | 2026-09-18 | fix(claude): skip retry-after header on overage-only rejections | `internal/runtime` |  |  |
| v7.3.8 | `f049e00b76ae` | 2026-09-18 | feat(xai): also allow grok imagine aspect_ratio 20:9 | `sdk/api` |  |  |
| v7.3.8 | `75bd6a60eb1b` | 2026-09-18 | feat(xai): allow grok imagine aspect_ratio 9:20 | `sdk/api` |  |  |
| v7.3.8 | `c93978c4ea2e` | 2026-09-19 | fix(schema): normalize true boolean subschemas and strip unsupported keywords | `internal/runtime`, `internal/util` |  |  |
| v7.3.8 | `b6d1f050af28` | 2026-09-19 | fix(executor): buffer post-tool text to order devin tool calls before assistant response | `internal/runtime` |  |  |
| v7.3.8 | `22392c537d95` | 2026-09-19 | fix(executor): restore hybrid passthrough mcp tool names in claude oauth | `internal/runtime` |  |  |
| v7.3.8 | `690f4f3116b6` | 2026-09-19 | feat(executor): support use-max-completion-tokens for openai compatibility models | `config.example.yaml`, `internal/config`, `internal/modelconfig` +3 |  |  |
| v7.3.8 | `f4852170ee59` | 2026-09-19 | test(config): verify claude cloak persistence and update behavior | `internal/api`, `internal/config` |  |  |
| v7.3.9 | `93b94d22a9bd` | 2026-09-19 | feat(management): support priority field in claude key patch | `internal/api` |  |  |
| v7.3.9 | `b4ff581dafa4` | 2026-09-19 | fix(management): reconcile expired cooldowns to active status in auth files | `internal/api`, `sdk/cliproxy` |  |  |
| v7.3.9 | `b532db9c4b14` | 2026-09-19 | fix(translator): preserve constraints and additionalProperties in gemini parametersJsonSchema | `internal/translator`, `internal/util` |  |  |
| v7.3.9 | `f247e2b017a8` | 2026-09-19 | feat(translator): support strict tool mode mapping for gemini and claude | `internal/translator` |  |  |
| v7.3.9 | `49eec664f419` | 2026-09-19 | fix(translator): align tool choice mapping and enforce fail-closed handling | `internal/translator` |  |  |
| v7.3.9 | `883660fb8153` | 2026-09-19 | fix(translator): deduct cache_write_tokens from input_tokens in openai/codex to claude (#5956) | `internal/translator` |  |  |
| v7.3.9 | `61fdfc341b96` | 2026-09-20 | feat(translator): support multimodal audio, video, and file inputs in responses to gemini | `internal/translator` |  |  |
| v7.3.9 | `1c87874966a0` | 2026-09-20 | fix(pluginhost): preserve http status codes in host callback and execution errors | `internal/pluginhost` |  |  |
| v7.3.9 | `42c9680eee55` | 2026-09-20 | feat(executor): support duplex streaming for codex websockets | `config.example.yaml`, `internal/api`, `internal/config` +3 |  |  |
| v8.0.0 | `31f4cfab3fc1` | 2026-09-27 | feat: migrate CLIProxyAPI to version 8 | `cmd/fetch_antigravity_models`, `cmd/fetch_codex_models`, `cmd/fetch_devin_models` +48 |  |  |
| v8.0.0 | `b1a33b80dc4c` | 2026-09-27 | docs: prune obsolete management notes | `.gitignore`, `config.example.yaml`, `docs/STEERING.md` +2 |  |  |
| v8.0.0 | `9a0502c87ec3` | 2026-09-27 | feat(api): use query params for v8 oauth routes | `config.example.yaml`, `internal/api` |  |  |
| v8.0.0 | `d9922f058902` | 2026-09-27 | feat(config): add v8 configuration API | `config.example.yaml`, `internal/api`, `internal/client` +5 |  |  |
| v8.0.1 | `6c385e606957` | 2026-09-27 | fix(config): drop obsolete legacy keys during v8 migration | `internal/api`, `internal/config` |  |  |
| v8.0.10 | `9f35c1dcdadc` | 2026-10-01 | refactor(signature): improve logging and sanitize logic for Gemini signatures | `internal/runtime`, `internal/signature` |  |  |
| v8.0.10 | `f3fd2f232db6` | 2026-10-01 | fix(signature): suppress routine Gemini bypass logs | `internal/runtime`, `internal/signature` |  |  |
| v8.0.10 | `028f6a191aa9` | 2026-10-01 | docs(readme): refresh provider models and add Muse Code and Devin (#6118) | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.10 | `2783e10c5962` | 2026-10-02 | fix(executor): publish antigravity stream usage first | `internal/runtime` |  |  |
| v8.0.10 | `cf3102cc03a1` | 2026-10-02 | refactor(executor): optimize translation logic and improve plugin invocation handling | `internal/runtime` |  |  |
| v8.0.11 | `3be5fa44f93d` | 2026-10-02 | fix(config): support historical v8 aliases | `docs/management-api-v8.md`, `internal/api`, `internal/config` +1 |  |  |
| v8.0.11 | `52d5507dd38c` | 2026-10-02 | feat(config): add shared upstream provider settings | `config.example.yaml`, `docs/management-api-v8.md`, `internal/api` +3 |  |  |
| v8.0.11 | `d7c3c0aaaaf4` | 2026-10-02 | feat(antigravity): trace upstream connection reuse in debug logs | `internal/logging`, `internal/runtime` |  |  |
| v8.0.11 | `8fbf152b5b49` | 2026-10-02 | fix(executor): preserve top-level responses token usage when service tier is present | `internal/pluginhost`, `internal/runtime`, `sdk/api` |  |  |
| v8.0.11 | `30aa1f120026` | 2026-10-02 | test(executor): relax connection reuse bounds in transport tests | `internal/runtime` |  |  |
| v8.0.11 | `5ec31442991a` | 2026-10-02 | fix(gemini): handle split usage metadata and max tokens in responses translation | `internal/runtime`, `internal/translator` |  |  |
| v8.0.12 | `8348923a1782` | 2026-10-02 | feat(pluginhost): add host routing reset cooldown callback | `internal/pluginhost`, `sdk/pluginabi`, `sdk/pluginapi` |  |  |
| v8.0.12 | `2044a01f4229` | 2026-10-03 | fix(codex): preserve web search sources in responses request include | `internal/translator` |  |  |
| v8.0.12 | `e6f6f26ad556` | 2026-10-03 | fix(claude): handle pause_turn stop reason in responses translation | `internal/translator` |  |  |
| v8.0.12 | `c163bae4bcc1` | 2026-10-03 | fix(auth): preserve terminal unauthorized state and prevent unsafe refresh retry | `internal/auth`, `sdk/cliproxy` |  |  |
| v8.0.12 | `0fb50a18c3eb` | 2026-10-03 | feat(management): inject auth indexes into v8 api-keys config | `internal/api`, `internal/config` |  |  |
| v8.0.12 | `6d57ac908762` | 2026-10-03 | fix(claude): pin session date in cloaked reminder to prevent prompt cache invalidation | `internal/runtime` |  |  |
| v8.0.12 | `d306f2c56dd7` | 2026-10-03 | fix(codex): retain template freeform apply_patch capability by default | `config.example.yaml`, `internal/api`, `internal/client` +1 |  |  |
| v8.0.13 | `d7914afdedca` | 2026-10-03 | fix(codex): expand integer field normalization mappings for tools | `internal/client` |  |  |
| v8.0.13 | `2c1dcc7c8b88` | 2026-10-03 | fix(claude): include web search sources in codex request when search tool is present | `internal/translator` |  |  |
| v8.0.13 | `a3b77566adf4` | 2026-10-03 | fix(codex): publish main usage before image tool usage in execute | `internal/runtime` |  |  |
| v8.0.13 | `ed4d972ef0bf` | 2026-10-03 | fix(responses): complete reasoning summary stream lifecycle events | `internal/translator` |  |  |
| v8.0.13 | `ea8ffd5f92c2` | 2026-10-03 | fix(devin): stream content early and defer thought stop for late signatures | `internal/runtime` |  |  |
| v8.0.13 | `d4692663be0b` | 2026-10-03 | fix(auth): snapshot auth before unlocking during registration and update | `sdk/cliproxy` |  |  |
| v8.0.13 | `42d484f9a2ff` | 2026-10-03 | fix(claude): map request timeout status to timeout_error | `sdk/api` |  |  |
| v8.0.13 | `6b037f614cc3` | 2026-10-03 | fix(antigravity): use actual Claude 5.5 high model IDs | `internal/registry` |  |  |
| v8.0.13 | `c575291ad31b` | 2026-10-03 | fix(antigravity): align Claude 5.5 token limits with upstream catalog | `internal/registry` |  |  |
| v8.0.13 | `e3abd9ae8e3d` | 2026-10-03 | fix(antigravity): replace retired Claude 4.6 models with 5.5 | `internal/registry` |  |  |
| v8.0.13 | `5dbce4f3fbfd` | 2026-10-03 | perf(claude): prefilter JSON payloads before diagnostic walks | `internal/runtime` |  |  |
| v8.0.14 | `ad381435a15d` | 2026-10-03 | fix(antigravity): coalesce probes and respect plugin model ownership | `internal/pluginhost`, `sdk/cliproxy` |  |  |
| v8.0.14 | `9521fe29b281` | 2026-10-03 | fix(antigravity): preserve model lifecycle and cooldown state | `sdk/cliproxy` |  |  |
| v8.0.14 | `7e857c375a1a` | 2026-10-03 | feat(antigravity): refresh per-account model entitlements safely | `internal/registry`, `sdk/cliproxy` |  |  |
| v8.0.14 | `61a70dc68701` | 2026-10-03 | feat(claude): adapt native messages JSON to SSE for non-stream translation | `internal/translator` |  |  |
| v8.0.14 | `91c934f68baf` | 2026-10-03 | feat(responses): support local shell tool translation | `internal/translator` |  |  |
| v8.0.14 | `174248ca7162` | 2026-10-03 | fix(antigravity): reject ambiguous CAQS signature wrappers | `internal/signature` |  |  |
| v8.0.14 | `9b9c2bdf54b0` | 2026-10-03 | feat(antigravity): support Claude 5.5 CAQS thinking signature replay | `internal/runtime`, `internal/signature` |  |  |
| v8.0.14 | `8a945b3fb8d1` | 2026-10-03 | fix(antigravity): restore Claude 4.6 model definitions | `internal/registry` |  |  |
| v8.0.14 | `678a92578ca3` | 2026-10-04 | feat(claude): preserve OAuth tool aliases across thread continuations | `internal/runtime` |  |  |
| v8.0.14 | `5c959cb032ef` | 2026-10-04 | fix(claude): mark missing thread state error for client replay | `internal/clienterror`, `sdk/api` |  |  |
| v8.0.14 | `26e5efbbdf67` | 2026-10-04 | fix(claude): reserve cache breakpoint for thread continuation | `internal/runtime` |  |  |
| v8.0.14 | `5d890405b59c` | 2026-10-04 | fix(claude): keep system messages top level for terminal user turns | `internal/runtime` |  |  |
| v8.0.14 | `c997cbb58afe` | 2026-10-04 | feat(httpwire): introduce error-aware response flushing helper | `internal/api`, `internal/cache`, `internal/httpwire` +5 |  |  |
| v8.0.14 | `8a01bc022bf5` | 2026-10-04 | feat(auth): introduce mutation locking and persistence delta merging | `sdk/cliproxy` |  |  |
| v8.0.14 | `1672be5d25bd` | 2026-10-04 | feat(registry): support custom model catalog sources and dynamic reload | `cmd/server`, `config.example.yaml`, `internal/config` +3 |  |  |
| v8.0.14 | `8ef43e4df3b2` | 2026-10-04 | fix(readme): remove AxisNow sponsorship details from README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.14 | `33ec5502c38a` | 2026-10-04 | fix(claude): refine model version parsing for assistant prefill rejection | `internal/translator` |  |  |
| v8.0.14 | `cbc4cd55329f` | 2026-10-04 | feat(interactions): support antigravity session continuation and model detail lookup | `internal/api`, `internal/cache`, `internal/runtime` +2 |  |  |
| v8.0.14 | `868e137f6c1d` | 2026-10-04 | feat(payload): introduce payload finalizer for protobuf and multipart requests | `AGENTS.md`, `internal/runtime` |  |  |
| v8.0.14 | `16d98881d4bb` | 2026-10-05 | feat(codex): preserve URL citations in OpenAI response translation | `internal/translator` |  |  |
| v8.0.14 | `4e4dede4811a` | 2026-10-05 | fix(antigravity): surface backend errors and support multi-line stream payloads | `internal/runtime` |  |  |
| v8.0.14 | `d31b61cb87c2` | 2026-10-05 | fix(websockets): propagate upstream disconnect errors during session activation | `internal/runtime` |  |  |
| v8.0.15 | `a4acc9f752bd` | 2026-10-05 | fix(antigravity): treat client disconnect after terminal stream chunk as success | `internal/runtime` |  |  |
| v8.0.15 | `97dd9eec628c` | 2026-10-05 | fix(openai): report error on truncated stream missing finish_reason | `sdk/api` |  |  |
| v8.0.16 | `980c3bc32d60` | 2026-09-24 | docs(readme): add panel4cliproxyapi to community projects | `README.md`, `README_CN.md` |  |  |
| v8.0.16 | `dfd7c638d68d` | 2026-10-04 | docs(readme): add Oh-My-CPA to usage statistics | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.16 | `6d06098dbebe` | 2026-10-04 | docs(readme): add cliproxy-rs to More choices | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.16 | `e1e3a811b84e` | 2026-10-05 | fix(gemini): enforce object type on schema nodes with properties | `internal/util` |  |  |
| v8.0.16 | `ba1bbd75d881` | 2026-10-05 | fix(gemini): support document blocks in Claude request translation | `internal/translator` |  |  |
| v8.0.16 | `c7b4d573f9f8` | 2026-10-05 | fix(codex): record reasoning effort after applying payload overrides | `internal/runtime` |  |  |
| v8.0.16 | `01e2856704d8` | 2026-10-05 | fix(registry): disable Devin catalog for home builds | `internal/registry` |  |  |
| v8.0.16 | `0ea4e1dcb4f0` | 2026-10-05 | fix(openai): complete responses stream on clean EOF after finish reason | `internal/runtime`, `internal/translator` |  |  |
| v8.0.16 | `eb6a768d103d` | 2026-10-05 | feat(claude): support explicit prompt cache options | `internal/runtime` |  |  |
| v8.0.16 | `40f4df07b7c8` | 2026-10-05 | feat(antigravity): enhance context hash rejection logging and batch retry logic | `internal/runtime` |  |  |
| v8.0.16 | `17dc1b81c248` | 2026-10-05 | feat(antigravity): enhance context hash rejection logging and introduce replay phases | `internal/runtime` |  |  |
| v8.0.16 | `611ebba41590` | 2026-10-05 | refactor(executor): unify model reporting and replay logs | `internal/runtime` |  |  |
| v8.0.16 | `68afd0d1b47b` | 2026-10-05 | feat(codex): support action sources in Claude web search translation | `internal/translator` |  |  |
| v8.0.16 | `a2976eb8a303` | 2026-10-06 | fix(claude): share OAuth tool alias store across scoped executor copies | `internal/runtime` |  |  |
| v8.0.16 | `d2e3a77ef16d` | 2026-10-06 | docs(readme): add japanese translation for panel4cliproxyapi | `README_JA.md` |  |  |
| v8.0.16 | `d2597a24e4b1` | 2026-10-06 | fix(openai): preserve unknown encrypted content in responses compat mode | `internal/runtime` |  |  |
| v8.0.17 | `da5f109f34b6` | 2026-10-06 | test(cliproxy): fix model catalog restore configuration in tests | `sdk/cliproxy` |  |  |
| v8.0.17 | `68ea171da276` | 2026-10-06 | fix(gemini): handle carrier thinking blocks for signed visible text and tool calls in Claude response translat | `internal/translator` |  |  |
| v8.0.17 | `a0d31924e05c` | 2026-10-06 | fix(gemini): map refusal finish reasons and handle partless stream completion in Claude response translation | `internal/translator` |  |  |
| v8.0.17 | `d510dccf1b0b` | 2026-10-06 | fix(gemini): support camelCase systemInstruction in request translation | `internal/translator` |  |  |
| v8.0.17 | `57bde35179ec` | 2026-10-07 | fix(executor): preserve upstream usage on apply patch failures | `internal/runtime` |  |  |
| v8.0.17 | `39632e52a6e0` | 2026-10-07 | fix(claude): keep system prompt at top level for leading user message runs | `internal/runtime` |  |  |
| v8.0.17 | `eca1d113859e` | 2026-10-07 | fix(claude): keep system prompt at top level for single terminal user turn | `internal/runtime` |  |  |
| v8.0.17 | `dbc4aad9e921` | 2026-10-07 | fix(gemini): ignore context cancellation after terminal chunk in Vertex stream | `internal/runtime` |  |  |
| v8.0.17 | `935aa6e3eac5` | 2026-10-07 | fix(cliproxy): prevent in-flight token refresh from overwriting concurrently updated credentials | `sdk/cliproxy` |  |  |
| v8.0.17 | `6920ed5dfc35` | 2026-10-07 | fix(cliproxy): propagate credential version and registration epoch in execution results | `sdk/cliproxy` |  |  |
| v8.0.17 | `50d5fe1831fe` | 2026-10-07 | fix(cliproxy): avoid selecting rejected credentials during refresh and preserve active quota cooldowns | `sdk/cliproxy` |  |  |
| v8.0.18 | `6a853dce05f5` | 2026-10-06 | feat(xai): expose Grok TTS on /v1/audio/speech | `internal/api`, `internal/client`, `internal/registry` +3 |  |  |
| v8.0.18 | `cd9e5acec79a` | 2026-10-07 | fix(api): compact json error response bodies for SSE compatibility | `sdk/api` |  |  |
| v8.0.18 | `14dc4e05a0ca` | 2026-10-07 | fix(claude): relocate caller system prompt to messages for OAuth requests | `internal/runtime` |  |  |
| v8.0.18 | `306c9bd824ed` | 2026-10-07 | fix(routing): restrict speech-only models to speech endpoints and refine xAI error scoping | `internal/runtime`, `sdk/api` |  |  |
| v8.0.19 | `e7b22260ddce` | 2026-10-07 | fix(claude): support plain-text and structured response bodies for missing thread errors | `internal/clienterror`, `internal/runtime`, `sdk/api` |  |  |
| v8.0.19 | `7c41cd05e433` | 2026-10-08 | feat: add global server GitHub token configuration | `cmd/server`, `config.example.yaml`, `internal/api` +5 |  |  |
| v8.0.2 | `4a2c81864f31` | 2026-09-27 | fix(config): preserve unknown legacy sections as comments during v8 migration | `internal/api`, `internal/config` |  |  |
| v8.0.20 | `3b69068e6c8c` | 2026-10-07 | fix(translator): stop dropping attachments and empty turns | `internal/pluginhost`, `internal/runtime`, `internal/translator` +2 |  |  |
| v8.0.20 | `0f96f568e4db` | 2026-10-08 | refactor(translator): update openai and interactions request converters to return errors | `examples/custom-provider`, `internal/runtime`, `internal/thinking` +3 |  |  |
| v8.0.20 | `34e73c75a3a4` | 2026-10-08 | fix(xai): add proxy support and semver validation to Grok CLI updater | `cmd/server`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.20 | `f51624aa35c1` | 2026-10-08 | fix(translator): report unsendable user turn attachments in openai responses to interactions | `internal/runtime`, `internal/translator` |  |  |
| v8.0.20 | `90654da57320` | 2026-10-08 | feat(xai): dynamically fetch latest Grok CLI version from npm registry with fallback | `cmd/server`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.21 | `54946fa3dfa2` | 2026-10-08 | docs(readme): remove full QQ group in README_CN | `README_CN.md` |  |  |
| v8.0.21 | `7c2133e201d9` | 2026-10-08 | fix(cliproxy): reconcile concurrent auth updates during batch model registration | `sdk/cliproxy` |  |  |
| v8.0.21 | `0a23627d229f` | 2026-10-08 | fix(config): normalize collection styles and unquote mapping keys in v8 api | `internal/api`, `internal/config` |  |  |
| v8.0.21 | `249e9ef97611` | 2026-10-08 | fix(meta): preserve foreign reasoning encrypted content in responses requests | `internal/runtime` |  |  |
| v8.0.21 | `2d5351b3e582` | 2026-10-08 | feat(vertex): add support for native gemini interactions api | `config.example.yaml`, `internal/config`, `internal/runtime` +2 |  |  |
| v8.0.22 | `80809679914f` | 2026-10-08 | fix(claude): ignore overage billing resets in unified rate limit parsing | `internal/runtime` |  |  |
| v8.0.22 | `ab151f5d04dc` | 2026-10-08 | fix(antigravity): include thoughts token count in openai completion tokens | `internal/translator` |  |  |
| v8.0.22 | `6331b38bba2c` | 2026-10-08 | feat(openai): support response.interrupt in responses websocket handling | `internal/runtime`, `sdk/api`, `sdk/cliproxy` |  |  |
| v8.0.22 | `67465884ca17` | 2026-10-09 | fix(pluginhost): detach unselected plugins on config apply | `internal/pluginhost` |  |  |
| v8.0.22 | `a881c0361c2e` | 2026-10-09 | fix(claude): respect registry completion limit for default responses max tokens | `internal/translator` |  |  |
| v8.0.22 | `2bc236d2fcc1` | 2026-10-09 | feat(claude): support responses compaction in claude executor | `internal/runtime` |  |  |
| v8.0.23 | `d318bcc3afb9` | 2026-10-09 | fix(codex): handle late response interrupts on websocket sessions | `internal/runtime`, `sdk/api` |  |  |
| v8.0.23 | `76c87ce91dcd` | 2026-10-09 | fix(xai): reset terminal error on websocket connection configuration | `internal/runtime` |  |  |
| v8.0.23 | `e9c8769a7a98` | 2026-10-09 | fix(executor): normalize and restore non-portable tool call IDs | `internal/runtime` |  |  |
| v8.0.23 | `4a8ecf5ceb39` | 2026-10-09 | fix(auth): skip local filtering for home dispatch | `sdk/cliproxy` |  |  |
| v8.0.23 | `f8e4a84074c0` | 2026-10-09 | fix(openai-compat): measure streaming TTFT on effective token frames | `internal/runtime` |  |  |
| v8.0.23 | `2d8ec73b3370` | 2026-10-09 | fix(claude): preserve mid-conversation effort directives during system insertion | `internal/runtime` |  |  |
| v8.0.23 | `9a7db8e36dae` | 2026-10-09 | fix(claude): defer stream message delta emission to trailing usage chunk | `internal/translator` |  |  |
| v8.0.23 | `51bde0ff9609` | 2026-10-09 | fix(codex): preserve resolved upstream model in direct openai image requests | `internal/runtime` |  |  |
| v8.0.24 | `a68ad1344fa4` | 2026-10-10 | fix(claude): address tool_search review findings | `internal/api`, `internal/client`, `internal/translator` |  |  |
| v8.0.24 | `58c0f6a2b247` | 2026-10-10 | feat(claude): support Codex client tool_search on the Responses API | `internal/client`, `internal/translator` |  |  |
| v8.0.24 | `dfe0a0effe43` | 2026-10-10 | fix(antigravity): omit tool_calls from default streaming delta template | `internal/translator` |  |  |
| v8.0.24 | `37fe2608ee42` | 2026-10-10 | feat(usage): propagate access provider and native key metadata to usage records | `internal/api`, `internal/pluginhost`, `internal/redisqueue` +3 |  |  |
| v8.0.24 | `cad1b50347f2` | 2026-10-10 | fix(devin): do not seed input token estimates for claude streams | `internal/runtime` |  |  |
| v8.0.24 | `5dbd92a0e0df` | 2026-10-10 | fix(auth): preserve upstream error cooldown on canceled stream contexts | `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.24 | `b0debb353796` | 2026-10-10 | fix(codex): shorten over-length tool call IDs in chat completions | `internal/translator` |  |  |
| v8.0.24 | `5cff76dd2a52` | 2026-10-10 | docs: refresh CPA-Manager-Plus descriptions in multilingual READMEs | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.24 | `be8c35f294a3` | 2026-10-10 | fix(claude): ignore invalid and null cache control entries in executor | `internal/runtime` |  |  |
| v8.0.24 | `a27183143e6b` | 2026-10-11 | fix(antigravity): drop trailing thought-only model turns | `internal/runtime` |  |  |
| v8.0.24 | `15d2148bc24b` | 2026-10-11 | fix(gemini): ensure trailing user turn for function response with media | `internal/runtime` |  |  |
| v8.0.24 | `f1d4a94ef439` | 2026-10-11 | fix(assets): update Modelflare logo image | `assets/modelflare.png` |  |  |
| v8.0.24 | `0cb6ecfd3117` | 2026-10-11 | feat(sponsorship): add Modelflare sponsorship details to README files | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v8.0.24 | `ce518c9bcfb9` | 2026-10-11 | test(claude): cover a reload between tool_search fallback and translation | `internal/runtime` |  |  |
| v8.0.24 | `29bd0415b973` | 2026-10-11 | fix(claude): pin upstream.claude.tool-search per request | `internal/runtime`, `internal/translator` |  |  |
| v8.0.24 | `03a3cfecb540` | 2026-10-11 | feat(claude): honor upstream.claude.tool-search in executor, catalog and server | `internal/api`, `internal/runtime`, `sdk/api` |  |  |
| v8.0.24 | `956c3da1a3fa` | 2026-10-11 | feat(claude): gate Responses tool_search translation behind upstream.claude.tool-search | `internal/translator` |  |  |
| v8.0.24 | `77e3d0d336a6` | 2026-10-11 | feat(config): add upstream.claude.tool-search switch | `config.example.yaml`, `internal/config`, `internal/watcher` |  |  |
| v8.0.24 | `8eed08efc854` | 2026-10-11 | fix(claude): drop cache_control from deferred tool_search tools | `internal/runtime`, `internal/translator` |  |  |
| v8.0.24 | `11f04c207738` | 2026-10-11 | feat(codex): advertise tool search only for natively supported Claude routes | `internal/api`, `internal/client`, `internal/registry` +3 |  |  |
| v8.0.24 | `52992659ecdf` | 2026-10-11 | feat(claude): fall back to pre-tool-search translation on unsupported upstreams | `internal/runtime` |  |  |
| v8.0.24 | `ae33711cee39` | 2026-10-11 | feat(config): add native-tool-search switch for Claude API keys | `config.example.yaml`, `internal/config`, `internal/watcher` |  |  |
| v8.0.24 | `aeec5e9761a0` | 2026-10-11 | feat(claude): emit native deferred tools and tool_reference for tool_search | `internal/translator` |  |  |
| v8.0.24 | `48ad3487b290` | 2026-10-11 | feat(executor): add Responses tool_search strip and legacy model helpers | `internal/runtime` |  |  |
| v8.0.24 | `18b6cc8aef4b` | 2026-10-11 | refactor(registry): use model getters for Devin and Meta | `internal/registry` |  |  |
| v8.0.24 | `3de4e248f6bb` | 2026-10-11 | docs(readme): remove aiberm sponsorship | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v8.0.24 | `091ff27f3128` | 2026-10-12 | feat(antigravity): persist model capability hints in auth metadata | `internal/store`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.25 | `6e4c54e5ea89` | 2026-10-12 | feat(pluginhost): expose quota provider plugin IDs | `sdk/pluginhost` |  |  |
| v8.0.25 | `bbcd5734f87d` | 2026-10-12 | fix(registry): treat credential_quota as quota cooldown in model availability | `internal/registry` |  |  |
| v8.0.25 | `9126772dfddb` | 2026-10-12 | fix(gemini): strip unsupported dependent schema keywords | `internal/util` |  |  |
| v8.0.25 | `3d61c9542c64` | 2026-10-12 | fix(management): conditionally expose quota_plugin_id for v8 auth files | `internal/api` |  |  |
| v8.0.25 | `0c7084019b43` | 2026-10-12 | fix(management): resolve v8 plugin quota contract gaps | `internal/api`, `internal/pluginhost` |  |  |
| v8.0.25 | `7a687f424c6b` | 2026-10-12 | feat(routing): support session-affinity-rebind-on option | `config.example.yaml`, `internal/config`, `sdk/cliproxy` |  |  |
| v8.0.25 | `e240bd84321c` | 2026-10-12 | fix(codex): fallback default model template when gpt-5.5 is missing | `cmd/fetch_codex_models`, `internal/client`, `internal/registry` |  |  |
| v8.0.25 | `7b0319fb1c19` | 2026-10-12 | feat(codex): support Sign in with ChatGPT | `cmd/server`, `config.example.yaml`, `internal/auth` +7 |  |  |
| v8.0.26 | `fdbc61030327` | 2026-10-12 | feat(auth): enhance plugin auth handling with multi-auth support and improved state management | `internal/pluginhost` |  |  |
| v8.0.26 | `dd7b4334920b` | 2026-10-12 | feat(server): support max-request-body-bytes request body limit | `config.example.yaml`, `internal/api`, `internal/client` +5 |  |  |
| v8.0.26 | `1e37b52b2a25` | 2026-10-12 | fix(executor): preserve websocket sessions across reload and auth removal | `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.26 | `77cf88b0883d` | 2026-10-12 | test(claude): pin executor clock in tool_search disabled golden | `internal/runtime` |  |  |
| v8.0.27 | `67870b67a550` | 2026-10-12 | docs: add sub2api to "Who is with us?" | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.27 | `519d590e831b` | 2026-10-12 | feat(streaming): support send-keepalive-before-ft option | `config.example.yaml`, `internal/config`, `sdk/api` |  |  |
| v8.0.28 | `a4a545c09d72` | 2026-10-03 | docs(readme): clarify Antigravity Claude 5.5 availability | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.28 | `02efbf13bef7` | 2026-10-11 | docs(readme): add Haiku 5.5 and refresh Antigravity availability | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.28 | `97ba1e896181` | 2026-10-12 | fix(claude): keep prompt cache when tool_search discovers the first deferred tool | `internal/runtime`, `internal/translator` |  |  |
| v8.0.28 | `50bfc50fc50a` | 2026-10-12 | docs(readme): list Muse Wonder models first | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.28 | `333f62b39653` | 2026-10-12 | docs(readme): note Meta Muse Wonder availability | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.28 | `02fc76b149a9` | 2026-10-12 | fix(claude): restore passthrough MCP tools wrapped with their full mcp__ name | `internal/runtime` |  |  |
| v8.0.28 | `c96d9c920a85` | 2026-10-13 | docs(readme): keep Gemini model list consistent | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.28 | `39ace7e919ca` | 2026-10-13 | docs(readme): note Gemini 4 Argon availability | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.28 | `51952a7b2f30` | 2026-10-13 | feat(httptransport): add request-scoped HTTP transport decorator | `internal/runtime`, `sdk/api`, `sdk/httptransport` |  |  |
| v8.0.28 | `f62d4d28ceb4` | 2026-10-13 | feat(registry): add claude-haiku-5-5 and remove legacy gemini models | `internal/registry` |  |  |
| v8.0.28 | `77fd91e8f463` | 2026-10-13 | feat(gemini): support client-executed tool_search in OpenAI Responses translator | `internal/translator`, `internal/util` |  |  |
| v8.0.28 | `a0fd8448f575` | 2026-10-13 | fix(schema): preserve multi-branch anyOf/oneOf unions during schema sanitization | `internal/runtime`, `internal/util` |  |  |
| v8.0.28 | `e235fb3d64a5` | 2026-10-13 | fix(server): prevent data races during concurrent config access and reloads | `internal/api`, `internal/watcher`, `sdk/cliproxy` |  |  |
| v8.0.28 | `4f9630df230b` | 2026-10-13 | feat(registry): add web_search native capability | `internal/registry` |  |  |
| v8.0.28 | `4c159069e201` | 2026-10-13 | feat(registry): add muse-wonder-1.0 models | `internal/registry` |  |  |
| v8.0.28 | `0d18d3b365be` | 2026-10-14 | feat(claude): support model-specific default max tokens from registry in OpenAI chat completions | `internal/runtime`, `internal/translator` |  |  |
| v8.0.28 | `81573f042982` | 2026-10-14 | fix(server): prevent data races during concurrent config access and reloads | `internal/api`, `sdk/api` |  |  |
| v8.0.28 | `b0db5448e8f8` | 2026-10-14 | feat(translator): use unique prefixes for fallback item IDs in OpenAI responses stream | `internal/translator` |  |  |
| v8.0.28 | `e230a04b01f8` | 2026-10-14 | fix(server): prevent data races during concurrent config access and reloads | `sdk/cliproxy` |  |  |
| v8.0.28 | `56eee6a4c2f1` | 2026-10-14 | fix(cliproxy): treat HTTP/2 closed connection errors as transient lifecycle drops | `sdk/cliproxy` |  |  |
| v8.0.28 | `57d005fe6c20` | 2026-10-14 | fix(codex): strip out-of-range int64 numeric bounds from tool schemas | `internal/runtime` |  |  |
| v8.0.28 | `40ce0e0121cd` | 2026-10-14 | feat(kimi): support schema normalization for namespaced tools and input tool lists | `internal/runtime` |  |  |
| v8.0.28 | `6143cc11c9e1` | 2026-10-14 | feat(registry): add gemini-4-argon model | `internal/registry` |  |  |
| v8.0.28 | `7795de19ca80` | 2026-10-14 | feat(sse): implement CodexSSEScanner for improved event data handling | `internal/runtime` |  |  |
| v8.0.28 | `866d84d08674` | 2026-10-14 | feat(registry): add gemini-4-argon-high model | `internal/registry` |  |  |
| v8.0.3 | `2c27e61fe791` | 2026-09-27 | fix(meta): strip unsupported search_content_types from web search tools | `internal/runtime` |  |  |
| v8.0.3 | `01274e226ed8` | 2026-09-27 | feat(config): warn on unrecognized sections during v8 migration | `internal/config`, `internal/logging` |  |  |
| v8.0.3 | `5756d76ef3da` | 2026-09-27 | fix(pluginhost): preserve disabled state in host auth callbacks | `internal/pluginhost` |  |  |
| v8.0.3 | `acdace936fa7` | 2026-09-28 | fix(discovery): support bonded interfaces and deduplicate build warnings | `internal/discovery`, `sdk/cliproxy` |  |  |
| v8.0.3 | `04f88169ada8` | 2026-09-28 | fix(gemini): preserve thought signatures in interactions translation | `internal/translator` |  |  |
| v8.0.3 | `0a45f2533440` | 2026-09-28 | fix(translator): append trailing newlines to SSEEventData frames | `internal/translator` |  |  |
| v8.0.3 | `4ad5bba756f9` | 2026-09-28 | fix(openai): prevent false positive namespace prefix matches in responses tools | `internal/translator` |  |  |
| v8.0.3 | `4b99d1d6f886` | 2026-09-28 | feat(openai): support quality parameter for xAI image requests | `sdk/api` |  |  |
| v8.0.4 | `ac42accc885d` | 2026-09-28 | fix(openai): use routed home model compatibility options | `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.4 | `d3cd466ddbbd` | 2026-09-28 | fix(codex): use resolved compat for multi-agent v2 | `internal/runtime` |  |  |
| v8.0.4 | `b9971ee3dfcc` | 2026-09-28 | feat(sdk): add Home credential capabilities and quota APIs | `internal/runtime`, `sdk/cliproxy`, `sdk/pluginhost` |  |  |
| v8.0.4 | `59baf7314af5` | 2026-09-28 | feat(codex): extract and persist subscription plan type | `cmd/fetch_codex_models`, `internal/api`, `internal/auth` +4 |  |  |
| v8.0.4 | `958eeca719c8` | 2026-09-28 | fix(claude): remap MCP aliases in tool_addition/tool_removal blocks | `internal/runtime` |  |  |
| v8.0.4 | `5d8bb3d9b774` | 2026-09-28 | fix(codex): normalize empty function call arguments in responses requests | `internal/translator` |  |  |
| v8.0.4 | `48686ccc8fbe` | 2026-09-28 | feat(config): preserve unknown nested fields as comments during v8 migration | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
| v8.0.4 | `d33f63f8e3d9` | 2026-09-29 | feat(logging): switch request ID generation to UUIDv7 | `internal/api`, `internal/logging` |  |  |
| v8.0.5 | `b87ac8371c57` | 2026-09-28 | feat(readme): add OpenLux sponsorship information and logo | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v8.0.5 | `b7f321f2a1a8` | 2026-09-29 | feat(api): support request path overrides and image protocols in model execution | `internal/pluginhost`, `internal/runtime`, `sdk/api` +2 |  |  |
| v8.0.5 | `fbc8e443f036` | 2026-09-29 | fix(claude): replay thinking block before tool use after web search result | `internal/translator` |  |  |
| v8.0.5 | `a0cc065d4d31` | 2026-09-29 | fix(claude): distinguish claude-sonnet-5-5 from sonnet-5 in model checks | `internal/runtime` |  |  |
| v8.0.5 | `cb9a0b5ce88e` | 2026-09-29 | feat(thinking): support output_config.effort for enabled Claude thinking | `internal/thinking`, `internal/translator` |  |  |
| v8.0.5 | `07c5dc0cda96` | 2026-09-29 | feat(readme): update OpenLux sponsorship links and promotional offer | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.5 | `743bbdfc1a8b` | 2026-09-29 | feat(readme): update OpenLux sponsorship details and promotional offer | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v8.0.5 | `775a308f0d2d` | 2026-09-29 | test(config): add test for DELETE operation preserving document presence | `internal/api` |  |  |
| v8.0.5 | `93b683cc53fa` | 2026-09-29 | docs(readme): remove CyberPay sponsor | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v8.0.5 | `dace3bb99fe1` | 2026-09-29 | docs(agents): mark /v0/management endpoints as deprecated | `AGENTS.md` |  |  |
| v8.0.5 | `9a3b869f6092` | 2026-09-29 | feat(models): add claude-sonnet-5-5 model | `internal/registry` |  |  |
| v8.0.5 | `df2774ca1867` | 2026-09-29 | feat(oauth): support per-channel model settings overrides | `config.example.yaml`, `internal/api`, `internal/config` +3 |  |  |
| v8.0.5 | `e5b5a1cfca35` | 2026-09-30 | fix(kimi): reorder interleaved tool outputs in responses input | `internal/runtime` |  |  |
| v8.0.5 | `4b4a444880f3` | 2026-09-30 | fix(claude): normalize boolean subschemas in tool parameter schemas | `internal/translator` |  |  |
| v8.0.5 | `bfa5aed4ea25` | 2026-09-30 | fix(home): defer plugin load marking and preempt failing config on retry | `sdk/cliproxy` |  |  |
| v8.0.5 | `097511b81fff` | 2026-09-30 | fix(management): remove deprecated credential quota endpoints from v8 API | `docs/management-api-v8.md`, `internal/api` |  |  |
| v8.0.5 | `71a5f1f6c5b3` | 2026-09-30 | fix(codex): normalize tool parameter number types to integer for codex clients | `internal/client`, `internal/runtime` |  |  |
| v8.0.5 | `f33f04d6c0c9` | 2026-09-30 | fix(codex): convert agent_message in compat mode without multi-agent optimization | `internal/client`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.5 | `369df975eac7` | 2026-09-30 | fix(claude): map and disambiguate tool names in responses translation | `internal/translator` |  |  |
| v8.0.5 | `6004ebdfdb0f` | 2026-09-30 | fix(claude): fail open on unmapped oauth mcp tool aliases | `internal/runtime` |  |  |
| v8.0.5 | `f1697119f6b0` | 2026-09-30 | fix(interactions): handle failure events in response translators and stream executor | `internal/runtime`, `internal/translator` |  |  |
| v8.0.5 | `81756a57e3a4` | 2026-09-30 | fix(auth): track auto-refresh job epochs and bound credential acquisition | `internal/runtime`, `sdk/cliproxy` |  |  |
| v8.0.5 | `c8a2bfe8f5c6` | 2026-09-30 | feat(models): add gpt-6.1-sol model | `cmd/fetch_codex_models`, `internal/registry` |  |  |
| v8.0.5 | `02548d6727a0` | 2026-09-30 | fix(auth): log warnings on auth persistence failures | `internal/logging`, `sdk/cliproxy` |  |  |
| v8.0.6 | `a8ffd5a84328` | 2026-10-01 | fix(codex): skip tool parameter integer normalization for codex executor targets | `internal/runtime` |  |  |
| v8.0.6 | `82f8e92b21e5` | 2026-10-01 | fix(executor): support negotiated ALPN protocols in uTLS client | `internal/runtime` |  |  |
| v8.0.7 | `97f244b8ddb9` | 2026-10-01 | fix(codex): pass target executor to compatibility translation and token counting | `internal/runtime` |  |  |
| v8.0.7 | `9e71c20d0169` | 2026-10-01 | fix(codex): handle upstream stream disconnect before first payload as bad gateway | `internal/runtime`, `sdk/api`, `test/codex_stream_disconnect_failover_test.go` |  |  |
| v8.0.7 | `67cb32b6d3a0` | 2026-10-01 | fix(codex): determine tool integer normalization by target executor identity | `internal/runtime` |  |  |
| v8.0.8 | `b467a83c0fe5` | 2026-10-01 | fix(xai): bump pinned grok client version to 1.0.44 for chat-proxy | `internal/runtime` |  |  |
| v8.0.9 | `63c04b4bd473` | 2026-10-02 | feat(config): introduce client configuration section for codex settings | `config.example.yaml`, `docs/management-api-v8.md`, `internal/api` +6 |  |  |
| v8.0.9 | `3ebee065161b` | 2026-10-02 | fix(interactions): bridge apply_patch tool calls in responses translation | `internal/api`, `internal/client`, `internal/pluginhost` +6 |  |  |
