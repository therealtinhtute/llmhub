# Upstream ledger — cliproxyapi v7.2.113..v7.2.135

- generated: 2026-08-17T06:44:26Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `f08bec353156`
- non-merge commits: 203

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.114 | `448893290d2a` | 2026-08-01 | fix(main): ensure safe closure of homeClient to prevent idle connections | `cmd/server` |  |  |
| v7.2.114 | `41fc5e134631` | 2026-08-03 | fix(auth): add bounded timeouts to Codex and Claude token refresh paths | `internal/auth` |  |  |
| v7.2.115 | `872070259ef6` | 2026-08-03 | fix(translator): emit `response.completed` on stream end when `finish_reason` is missing | `internal/translator` |  |  |
| v7.2.115 | `0879929821cb` | 2026-08-03 | fix(translator): preserve non-object Claude tool inputs when building function call args | `internal/translator` |  |  |
| v7.2.115 | `aea71c0070fa` | 2026-08-03 | fix(translator): preserve content block order when building non-stream Claude OpenAI responses | `internal/translator` |  |  |
| v7.2.115 | `1cf9a45ca250` | 2026-08-03 | Handle encrypted collaboration message schemas | `internal/client` |  |  |
| v7.2.115 | `0cc24b2e0006` | 2026-08-03 | fix(translator): emit function_call_arguments.done and make function call stream events idempotent | `internal/translator` |  |  |
| v7.2.115 | `abbb1c7534f3` | 2026-08-03 | fix(translator): accept snake_case Gemini usage metadata in interactions conversion | `internal/translator` |  |  |
| v7.2.115 | `13435c93d226` | 2026-08-03 | feat(executor): normalize OpenAI tool results for text-only compatibility models | `config.example.yaml`, `internal/runtime` |  |  |
| v7.2.115 | `a303fd869b03` | 2026-08-03 | feat(codex): support `max-context-length` overrides for configured models | `config.example.yaml`, `internal/client`, `internal/config` +2 |  |  |
| v7.2.115 | `8d675e690ddf` | 2026-08-03 | feature(codex): Normalize invalid sub2api message item IDs during Codex input sanitization | `internal/runtime` |  |  |
| v7.2.116 | `8037ed8a0dd0` | 2026-07-18 | docs: add Alex to related projects | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.116 | `0eddcff50a14` | 2026-07-20 | Update README_CN.md | `README_CN.md` |  |  |
| v7.2.116 | `6e3512f6639d` | 2026-07-23 | docs: move Alex to end of project list | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.116 | `1e38a3a544ec` | 2026-07-31 | fix: retry Home OAuth requests after unauthorized | `internal/api`, `internal/client`, `internal/home` +3 |  |  |
| v7.2.116 | `b437d87170ef` | 2026-08-01 | fix(auth): validate stream retry results | `sdk/cliproxy` |  |  |
| v7.2.116 | `1df21b14bf4f` | 2026-08-01 | fix(usage): update token fingerprint after refresh | `internal/runtime` |  |  |
| v7.2.116 | `063f9d341285` | 2026-08-01 | fix(auth): reprepare Home credentials after refresh | `sdk/cliproxy` |  |  |
| v7.2.116 | `d952cb429706` | 2026-08-01 | fix(home): reject disabled refreshed credentials | `internal/runtime` |  |  |
| v7.2.116 | `0fc028613b1f` | 2026-08-02 | chore: exclude test changes from Home fixes | `internal/api`, `internal/client`, `internal/runtime` +1 |  |  |
| v7.2.116 | `a81b9e9cedab` | 2026-08-02 | fix(home): report every unauthorized attempt | `internal/api`, `internal/client`, `internal/runtime` +1 |  |  |
| v7.2.116 | `707934917a74` | 2026-08-02 | feat(claude): enable TLS session resumption | `internal/auth`, `internal/runtime` |  |  |
| v7.2.116 | `3c58d18579a8` | 2026-08-02 | test(claude): preserve adaptive thinking signatures | `internal/runtime`, `test/thinking_conversion_test.go` |  |  |
| v7.2.116 | `2959f9efb39f` | 2026-08-02 | fix(claude): align OAuth exchange and account inspection | `internal/auth` |  |  |
| v7.2.116 | `497cf491aab4` | 2026-08-02 | fix(claude): pass Fast errors through without retry | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.116 | `ce7fcd920f20` | 2026-08-02 | fix(claude): keep client cancellation availability-neutral | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.116 | `2228847e638b` | 2026-08-02 | fix(claude): close the count_tokens cloaking gap | `internal/runtime` |  |  |
| v7.2.116 | `05e7232d8fb7` | 2026-08-02 | fix(httpwire): retain request state after partial writes | `internal/httpwire` |  |  |
| v7.2.116 | `b3ed702e3acd` | 2026-08-02 | fix(claude): harden OAuth identity and native routing | `internal/auth`, `internal/runtime` |  |  |
| v7.2.116 | `fa6bc77f28e0` | 2026-08-02 | fix(claude): decode stacked response encodings | `internal/auth`, `internal/runtime` |  |  |
| v7.2.116 | `f63a925d15f6` | 2026-08-02 | fix(claude): replay measured OAuth wire, Fast and diagnostic profiles | `config.example.yaml`, `internal/auth`, `internal/config` +2 |  |  |
| v7.2.116 | `842dbe63853f` | 2026-08-02 | perf(claude): batch OAuth tool name rewrites | `internal/runtime` |  |  |
| v7.2.116 | `a20626f1eee9` | 2026-08-02 | fix(claude): send Anthropic header names with the real client's casing | `internal/runtime` |  |  |
| v7.2.116 | `6912a74064b9` | 2026-08-02 | fix(claude): keep a fast-mode refusal request-scoped | `internal/runtime` |  |  |
| v7.2.116 | `4fdf59c436d3` | 2026-08-02 | fix(claude): synchronize credential metadata and device pools | `internal/auth`, `internal/runtime` |  |  |
| v7.2.116 | `a2933c7737a4` | 2026-08-02 | fix(claude): scope Anthropic beta and count_tokens policies | `config.example.yaml`, `internal/runtime` |  |  |
| v7.2.116 | `afdd251ccaed` | 2026-08-02 | fix(claude): preserve semantics in MCP tool aliases | `internal/runtime` |  |  |
| v7.2.116 | `ef89c6a69d0f` | 2026-08-02 | fix(claude): reconstruct cloaked system prompts like the real client | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v7.2.116 | `f3e25ab2bae6` | 2026-08-02 | feat(claude): align OAuth wire identity and TLS with Claude Code 2.1.220 | `config.example.yaml`, `internal/api`, `internal/auth` +5 |  |  |
| v7.2.116 | `9b1142399c29` | 2026-08-03 | fix(claude): rebuild the Responses reasoning chain | `internal/runtime`, `internal/translator` |  |  |
| v7.2.116 | `6f8f11a3249e` | 2026-08-03 | fix(claude): carry caller system inputs into Claude system blocks | `internal/runtime`, `internal/translator` |  |  |
| v7.2.116 | `3fac4a09d80d` | 2026-08-03 | fix(claude): preserve cloak system block boundaries | `internal/runtime` |  |  |
| v7.2.116 | `3904c40d655e` | 2026-08-03 | fix(antigravity): strip propertyNames inside a property named "properties" | `internal/runtime`, `internal/translator`, `internal/util` |  |  |
| v7.2.116 | `56e533fb96d9` | 2026-08-03 | fix(claude): pair diagnostics with cache beta | `internal/runtime` |  |  |
| v7.2.116 | `903e41b6dcd0` | 2026-08-03 | docs(claude): clarify exact fingerprint baseline | `config.example.yaml`, `internal/config`, `internal/runtime` |  |  |
| v7.2.116 | `3e70208d435e` | 2026-08-03 | fix(claude): keep custom token counts local | `internal/runtime` |  |  |
| v7.2.116 | `1214326bd727` | 2026-08-03 | fix(claude): align OAuth betas with native accounts | `internal/runtime` |  |  |
| v7.2.116 | `a5f63909a564` | 2026-08-03 | fix(claude): harden request lifecycle | `internal/auth`, `internal/cache`, `internal/runtime` +2 |  |  |
| v7.2.116 | `134a66738c35` | 2026-08-03 | fix(codex): hydrate missing `response.completed` output item IDs | `internal/runtime` |  |  |
| v7.2.117 | `8cf1d46f065c` | 2026-08-03 | fix(usage): account for Claude thinking tokens | `internal/runtime` |  |  |
| v7.2.117 | `7fe847376667` | 2026-08-03 | feat(codex): prepare multi-agent v2 tool definitions at the Responses boundary for Codex clients | `config.example.yaml`, `internal/client`, `sdk/api` |  |  |
| v7.2.117 | `82d6242098a7` | 2026-08-04 | fix(responses): always emit `response.usage.output_tokens_details.reasoning_tokens` | `internal/translator` |  |  |
| v7.2.117 | `9b8d97441e86` | 2026-08-04 | fix(responses): preserve original request model on response.created/response.in_progress payloads | `internal/runtime`, `internal/translator`, `sdk/api` |  |  |
| v7.2.117 | `b782d4374f82` | 2026-08-04 | feat(api): add Grok Shell-aware `/v1/models` handling with dedicated model response formatting | `internal/api`, `internal/client`, `internal/registry` |  |  |
| v7.2.117 | `50e2a99fc631` | 2026-08-04 | chore: add .pi to .gitignore | `.gitignore` |  |  |
| v7.2.117 | `44d5e0bebcbc` | 2026-08-04 | fix(codex): preserve base64 PDF document blocks when converting Claude requests to Codex | `internal/translator` |  |  |
| v7.2.117 | `4f5ec105b11c` | 2026-08-04 | fix(codex): set `multi_agent_version` to v2 when optimizeMultiAgentV2 is enabled | `internal/client` |  |  |
| v7.2.118 | `84232747e20e` | 2026-07-29 | fix(xai): register video preview alias | `internal/registry`, `sdk/api` |  |  |
| v7.2.118 | `61d9a30d126e` | 2026-07-29 | fix(xai): preserve preview alias auth routing | `sdk/api` |  |  |
| v7.2.118 | `abaeb55bb20e` | 2026-07-29 | fix(xai): support Grok Imagine Video 1.5 GA | `internal/api`, `internal/client`, `internal/registry` +1 |  |  |
| v7.2.118 | `41fc0ba859b6` | 2026-08-04 | docs: change Kimi open platform links in READMEs | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.118 | `7ea5e3ae6634` | 2026-08-04 | fix: honor router skips for prepared stream routes | `sdk/api` |  |  |
| v7.2.118 | `29bdd3c1492c` | 2026-08-05 | docs: remove AICodeMirror sponsor section from all READMEs and delete its logo asset | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.2.118 | `42eef103d6d2` | 2026-08-05 | feat(antigravity): obfuscate sensitive words in system instructions | `config.example.yaml`, `internal/config`, `internal/runtime` +1 |  |  |
| v7.2.119 | `6e92e3e60e64` | 2026-08-05 | docs: add EasyCLIProxyAPI desktop client recommendation to README translations | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.119 | `690f93dc1487` | 2026-08-05 | fix(openai): drop OpenAI stream chunks after `[DONE]` | `internal/runtime`, `internal/translator` |  |  |
| v7.2.120 | `533b69e3e09e` | 2026-08-04 | fix(claude): skip context management when thinking is disabled | `internal/runtime` |  |  |
| v7.2.120 | `076ec64c16e0` | 2026-08-05 | fix(home): preserve requested usage metadata before auth preparation | `sdk/cliproxy` |  |  |
| v7.2.120 | `e400d7191d45` | 2026-08-05 | feat(xai): bump client version to 0.2.120 and include Grok Shell auth headers on XAI requests | `internal/runtime` |  |  |
| v7.2.120 | `ea37d13a9ece` | 2026-08-06 | fix(responses): support custom tool calls and namespace collisions in request/response conversion | `internal/translator` |  |  |
| v7.2.121 | `8392b180ce37` | 2026-08-06 | fix(cliproxy): avoid credential cooldown for connection-lifecycle disconnect errors | `sdk/cliproxy` |  |  |
| v7.2.121 | `1674aaf43b52` | 2026-08-06 | fix(cliproxy): canonicalize thinking-suffix model states for cooldown and scheduler sharing | `sdk/cliproxy` |  |  |
| v7.2.121 | `b148af80bd77` | 2026-08-06 | fix(api): escape the model name in the unroutable model error | `sdk/api` |  |  |
| v7.2.121 | `579f5e30fbd6` | 2026-08-06 | fix(auth): rotate credentials for unknown upstream failures | `internal/clienterror`, `sdk/api`, `sdk/cliproxy` |  |  |
| v7.2.121 | `c1d69e7b4778` | 2026-08-06 | fix(auth): avoid penalizing credentials for client faults | `internal/clienterror`, `sdk/api`, `sdk/cliproxy` |  |  |
| v7.2.121 | `fe28d582f43b` | 2026-08-06 | fix(openai): expose only client-fault streaming errors | `internal/runtime`, `sdk/api`, `sdk/cliproxy` |  |  |
| v7.2.121 | `71e87111e9d8` | 2026-08-06 | fix(claude): accumulate consecutive role turns during request conversion | `internal/translator` |  |  |
| v7.2.121 | `f1a21a9b95bf` | 2026-08-06 | fix(responses): merge consecutive OpenAI response turns and align tool call/result blocks for Claude conversio | `internal/translator` |  |  |
| v7.2.121 | `dcee14dd3c53` | 2026-08-06 | feat(compat): preserve compat-mode thinking/signature blocks for API-key models | `config.example.yaml`, `internal/config`, `internal/modelconfig` +6 |  |  |
| v7.2.121 | `65fc536e6e32` | 2026-08-06 | fix(auth): preserve session affinity across priorities | `config.example.yaml`, `sdk/cliproxy` |  |  |
| v7.2.121 | `2e91e99e0339` | 2026-08-06 | fix(responses): translate `text.format` into `response_format` in request conversion | `internal/translator` |  |  |
| v7.2.121 | `94808f1087d2` | 2026-08-06 | fix(responses): correct request wrapper detection in antigravity response conversion | `internal/translator` |  |  |
| v7.2.121 | `7fef08ea2bf4` | 2026-08-06 | feat(codex): add model-level `is-compat` flag to rewrite MultiAgentV2 `agent_message` for Responses-compatible | `internal/config` |  |  |
| v7.2.121 | `e5ea945ed93d` | 2026-08-06 | feat(codex): add model-level `is-compat` flag to rewrite MultiAgentV2 `agent_message` for Responses-compatible | `config.example.yaml`, `internal/config`, `internal/modelconfig` +3 |  |  |
| v7.2.122 | `0a95fa62a106` | 2026-08-07 | feat(compat): preserve Claude thinking/tool-call content for is-compat OpenAI compatibility models | `config.example.yaml`, `internal/config`, `internal/modelconfig` +4 |  |  |
| v7.2.122 | `5e25566c240e` | 2026-08-07 | fix(codex): normalize reasoning and function_call item IDs during input sanitization | `internal/runtime` |  |  |
| v7.2.123 | `31a4e9b4870f` | 2026-08-07 | fix(cliproxy): preserve route model on built-in selector cooldown errors | `sdk/cliproxy` |  |  |
| v7.2.123 | `035f5489e503` | 2026-08-07 | Revert "docs: remove AICodeMirror sponsor section from all READMEs and delete its logo asset" | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.2.123 | `c30e60a11b49` | 2026-08-07 | Reduce Codex request amplification for large payloads | `internal/runtime`, `internal/translator` |  |  |
| v7.2.123 | `34364fff469e` | 2026-08-07 | Prevent large Gemini payloads from being duplicated during normalization | `internal/translator` |  |  |
| v7.2.123 | `d40bd7701578` | 2026-08-07 | fix(.gitignore): update .llm-wiki entry to exclude trailing slash | `.gitignore` |  |  |
| v7.2.123 | `a4a87735a296` | 2026-08-07 | chore(.gitignore): add missing entries for AGENTS.override.md, .gocache, and .gitnexus | `.gitignore` |  |  |
| v7.2.123 | `dd67f56f265c` | 2026-08-07 | fix(xai): remap bad-credentials 403s to unauthorized | `internal/runtime` |  |  |
| v7.2.123 | `148eff206e44` | 2026-08-07 | feat(docs): add Infistar.ai sponsorship information and logo to documentation | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.2.124 | `4abb0e66026c` | 2026-08-07 | feat(signature): add Grok as a target-only provider family | `internal/signature` |  |  |
| v7.2.124 | `8558f44329af` | 2026-08-07 | feat(signature): identify Kimi thinking signatures by fixed size | `internal/signature` |  |  |
| v7.2.124 | `197f52042637` | 2026-08-08 | fix(codex): normalize `custom_tool_call` IDs with `ctc_` prefix during Codex input sanitization | `internal/runtime` |  |  |
| v7.2.124 | `aff2095abd3b` | 2026-08-08 | fix(codex): deduplicate Responses tools when converting to OpenAI Chat Completions | `internal/translator` |  |  |
| v7.2.124 | `9829bd9d3e51` | 2026-08-08 | fix(cliproxy): stop non-streaming keep-alive after OpenAI handler execution | `sdk/api` |  |  |
| v7.2.124 | `01a21b77f4dc` | 2026-08-08 | fix(cliproxy): delegate OpenAI-compatible OAuth refresh to plugin auth providers | `internal/pluginhost`, `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.124 | `e64cdbf5591d` | 2026-08-08 | fix(codex): resolve credential-aware model before forwarding API-key alpha search requests | `internal/api`, `sdk/cliproxy` |  |  |
| v7.2.124 | `36936340a33a` | 2026-08-08 | fix(kimi): canonicalize K2.7 Code model aliases to official Kimi-For-Coding IDs | `internal/runtime` |  |  |
| v7.2.124 | `4b3cc55cdc93` | 2026-08-08 | fix(cliproxy): centralize client error status mapping and apply context cancellation/deadline HTTP codes | `internal/api`, `internal/client`, `internal/clienterror` +3 |  |  |
| v7.2.124 | `37609fa17993` | 2026-08-08 | fix(claude): synthesize belated tool_use starts for empty-name calls | `internal/translator` |  |  |
| v7.2.125 | `b921b5d03264` | 2026-08-08 | Review: tighten the in-place byte write guard | `internal/util` |  |  |
| v7.2.125 | `1737596e020a` | 2026-08-08 | Make the payload-reuse guards able to fail | `internal/translator`, `internal/util` |  |  |
| v7.2.125 | `99929209840b` | 2026-08-08 | Avoid copying large payloads in Antigravity reads | `internal/runtime`, `internal/signature`, `sdk/cliproxy` |  |  |
| v7.2.125 | `b7a441522a90` | 2026-08-08 | Reduce Antigravity request amplification for large payloads | `internal/translator` |  |  |
| v7.2.125 | `ea8882bf26b2` | 2026-08-08 | Add a no-copy GJSON parse helper | `internal/util` |  |  |
| v7.2.125 | `2e6b1d83f6c3` | 2026-08-09 | fix(claude): add Claude-compatible thinking replay persistence for multi-turn sessions | `internal/cache`, `internal/config`, `internal/runtime` |  |  |
| v7.2.125 | `3522e481aa7b` | 2026-08-09 | fix(openai): emit `response.failed` stream errors for Codex requests | `internal/client`, `sdk/api` |  |  |
| v7.2.126 | `a6825fe9922b` | 2026-08-09 | fix(xai): normalize forced web_search `tool_choice` during Responses request prep | `internal/runtime` |  |  |
| v7.2.126 | `673bac5fc606` | 2026-08-09 | fix(codex): normalize `custom_tool_call_output` IDs with `ctco_` prefix during Codex input sanitization | `internal/runtime` |  |  |
| v7.2.126 | `5314b29da963` | 2026-08-09 | fix(codex): forward sequential cutoff reasoning summaries | `internal/runtime` |  |  |
| v7.2.126 | `37411842e859` | 2026-08-09 | fix(openai,claude): normalize Responses custom tool-call identity with shared namespace handling | `internal/translator` |  |  |
| v7.2.126 | `4906ead34fa5` | 2026-08-09 | fix(openai): finalize open response items on `[DONE]` when final chunk lacks `finish_reason` | `internal/translator` |  |  |
| v7.2.127 | `ecc9aa72b32f` | 2026-08-10 | fix(openai): preserve assistant content when converting Responses tool-call turns | `internal/runtime`, `internal/translator` |  |  |
| v7.2.127 | `93c378b791f7` | 2026-08-10 | fix(claude): recover malformed OAuth MCP aliases when reverse-remapping Claude tool names | `internal/runtime` |  |  |
| v7.2.128 | `bd34ceca0420` | 2026-08-10 | feat(codex): add realtime hangup forwarding and local client-secret support | `examples/realtime-openai-go`, `internal/api`, `internal/client` |  |  |
| v7.2.128 | `9c8e4a07e638` | 2026-08-10 | fix(auth): prioritize rate-limit status over error body | `internal/clienterror`, `sdk/cliproxy` |  |  |
| v7.2.128 | `d0d77182ee8e` | 2026-08-10 | fix(auth): rotate DeepSeek authentication failures | `internal/clienterror`, `sdk/cliproxy` |  |  |
| v7.2.128 | `45ffd115fdf0` | 2026-08-10 | fix(auth): rotate keys after DeepSeek insufficient balance | `internal/clienterror`, `sdk/cliproxy` |  |  |
| v7.2.129 | `177c7619b681` | 2026-08-10 | test(antigravity): clear credits state in place to fix a cleanup data race | `internal/runtime` |  |  |
| v7.2.129 | `bb7278a1af58` | 2026-08-10 | perf(antigravity): skip replay index rebuilds no item can observe | `internal/runtime` |  |  |
| v7.2.129 | `984836ba379d` | 2026-08-10 | refactor(antigravity): route replay merge through the request index | `internal/runtime` |  |  |
| v7.2.129 | `0e4c0dab7394` | 2026-08-10 | refactor(antigravity): document replay index lifecycle and align empty-contents semantics | `internal/runtime` |  |  |
| v7.2.129 | `9eedbc27bd4b` | 2026-08-10 | perf(antigravity): index reasoning replay requests | `internal/runtime` |  |  |
| v7.2.129 | `189776aab1fc` | 2026-08-11 | fix(claude): validate legacy-model system turns before sending | `internal/runtime` |  |  |
| v7.2.129 | `8638f28db556` | 2026-08-11 | fix(claude): drop auto context_management without eligible thinking | `internal/runtime` |  |  |
| v7.2.129 | `a8bbbea2b9b5` | 2026-08-11 | fix(claude): recognize and pass through native Haiku helpers | `internal/runtime` |  |  |
| v7.2.129 | `f0034ca66376` | 2026-08-11 | fix(claude): restore cloaked prompt-cache ownership and native shape | `internal/runtime` |  |  |
| v7.2.129 | `516ec3a0006b` | 2026-08-11 | fix(antigravity): harden per-credential transport pooling | `internal/runtime` |  |  |
| v7.2.129 | `c33a33e14a6d` | 2026-08-11 | perf(antigravity): reuse native upstream connections | `internal/runtime` |  |  |
| v7.2.129 | `5fa66293db18` | 2026-08-11 | fix(antigravity): preserve request plugin hook semantics | `internal/runtime`, `sdk/translator` |  |  |
| v7.2.129 | `cf8c27fe90ab` | 2026-08-11 | perf(antigravity): translate each upstream request once | `internal/runtime` |  |  |
| v7.2.129 | `d5f68856f6da` | 2026-08-11 | perf(antigravity): batch pre-upstream JSON rewrites | `internal/runtime`, `internal/signature` |  |  |
| v7.2.129 | `ba5ab795a2de` | 2026-08-11 | feat(plugin): add schema-v3 stream chunk contract to omit payload request bodies | `internal/pluginhost`, `sdk/api`, `sdk/pluginabi` +1 |  |  |
| v7.2.129 | `5d9b62996236` | 2026-08-11 | fix(runtime): close per-request uTLS HTTP/2 connections with request context | `internal/runtime` |  |  |
| v7.2.129 | `934da2379d62` | 2026-08-12 | fix(openai): preserve structured and stringified custom tool outputs during Responses conversion | `internal/translator` |  |  |
| v7.2.129 | `e9c44ae256c5` | 2026-08-12 | fix(codex): fall back to current tool-call state when item IDs don’t match | `internal/translator` |  |  |
| v7.2.129 | `a59caebc68cf` | 2026-08-12 | fix(kimi): treat `[reasoning unavailable]` as unusable reasoning in message normalization | `internal/runtime` |  |  |
| v7.2.129 | `db143aebac93` | 2026-08-12 | fix(codex): make input ID sanitization collision-resistant and deterministic | `internal/runtime` |  |  |
| v7.2.130 | `f43aad7637ad` | 2026-08-12 | fix(codex): normalize request session header to `Session-Id` and preload codex headers | `internal/runtime` |  |  |
| v7.2.130 | `133047de66bf` | 2026-08-12 | fix(codex): clear multi-agent-v2 optimization state on namespace conflicts | `internal/runtime` |  |  |
| v7.2.130 | `5b5f428ad9a6` | 2026-08-12 | fix(claude): recover OAuth tool names with duplicated server alias prefixes | `internal/runtime` |  |  |
| v7.2.130 | `2ab25eae9676` | 2026-08-12 | fix(codex): strip unsupported prompt_cache_options from OpenAI Responses requests | `internal/translator` |  |  |
| v7.2.130 | `c845ce15c9de` | 2026-08-12 | fix(openai): refactor websocket responses request merge/repair path | `sdk/api` |  |  |
| v7.2.130 | `a2337beb2414` | 2026-08-12 | fix(kimi): select upstream request format from source format (Claude/OpenAI) | `internal/runtime` |  |  |
| v7.2.130 | `17a479a8db98` | 2026-08-12 | feat(management): add request-scoped proxy override for APICall | `internal/api` |  |  |
| v7.2.130 | `b08fe3b49264` | 2026-08-12 | fix(codex): preserve multi-agent-v2 namespace handling across incremental websocket turns | `internal/client`, `internal/runtime` |  |  |
| v7.2.130 | `522b4de54a8d` | 2026-08-12 | fix(openai): handle premature SSE stream termination with terminal error emission | `internal/runtime`, `internal/translator`, `sdk/api` |  |  |
| v7.2.131 | `75d2c4a4b4e2` | 2026-08-12 | fix(openai): avoid JSON copies in websocket responses tool-call repair path | `sdk/api` |  |  |
| v7.2.131 | `323b7276bc5b` | 2026-08-13 | feat(registry): add model modality metadata to registry definitions | `internal/registry` |  |  |
| v7.2.131 | `db35b91e2a46` | 2026-08-13 | feat(openai): add xAI Grok Imagine Image 2.0 image model support | `internal/api`, `internal/client`, `internal/registry` +1 |  |  |
| v7.2.131 | `6f2cea948451` | 2026-08-13 | feat(config): add per-credential request-retry override support | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
| v7.2.131 | `8b54db36ae7a` | 2026-08-13 | fix(translator): drop Gemini hidden thought parts during request conversion | `internal/translator` |  |  |
| v7.2.132 | `d757063c9674` | 2026-08-13 | fix(docs): update RunAPI registration links in README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.132 | `a40d0e6e9de3` | 2026-08-13 | fix(docs): update FennoAI sponsorship details and links in README files | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.132 | `8d670b98ffac` | 2026-08-14 | fix(translator): make Gemini tool call IDs deterministic and robustly match responses | `internal/translator` |  |  |
| v7.2.132 | `bdde638c27a3` | 2026-08-14 | fix(claude): pass through caller MCP tools on alias server collision | `internal/runtime` |  |  |
| v7.2.132 | `f6f03e4de99c` | 2026-08-14 | fix(claude): use BIP-39 words for OAuth MCP tool aliases | `internal/runtime` |  |  |
| v7.2.132 | `7cf92793c1b0` | 2026-08-14 | fix(gemini): skip zero-token usage placeholders and always finalize stream reporters | `internal/runtime` |  |  |
| v7.2.132 | `f2d272da817d` | 2026-08-14 | fix(claude): recover OAuth tool aliases for repeated prefixes and malformed IDs | `internal/runtime` |  |  |
| v7.2.132 | `7ea9c670ea0d` | 2026-08-14 | feat(registry): add Gemini 3.7 Flash High model definition | `internal/registry` |  |  |
| v7.2.132 | `78f0c4079e3e` | 2026-08-15 | fix(openai): mark truncated/filtered responses as incomplete and avoid finalizing partial tool calls | `internal/translator` |  |  |
| v7.2.132 | `98c98d66be1a` | 2026-08-15 | fix(codex): cache multi-agent spawn-agent model data and invalidate on updates | `internal/client`, `internal/registry` |  |  |
| v7.2.132 | `fab077a04b4a` | 2026-08-15 | fix(claude): map refusal and sensitive stop reasons to `content_filter` | `internal/translator` |  |  |
| v7.2.132 | `6edf9c482199` | 2026-08-15 | fix(translator): strip nested `prompt_cache_breakpoint` from Codex Responses payloads | `internal/translator` |  |  |
| v7.2.132 | `b90d8ee9eb71` | 2026-08-15 | fix(auth): preserve custom auth-file metadata during token refresh and relogin | `internal/api`, `internal/auth`, `internal/misc` +2 |  |  |
| v7.2.133 | `970529b6eefa` | 2026-07-16 | perf(api): skip inactive request interceptors | `sdk/api` |  |  |
| v7.2.133 | `baa11ed6dd75` | 2026-08-12 | fix(openai): match websocket item metadata case-insensitively | `sdk/api` |  |  |
| v7.2.133 | `49b2f891ac32` | 2026-08-12 | fix(openai): preserve duplicate websocket input semantics | `sdk/api` |  |  |
| v7.2.133 | `f8bcd1cc5fec` | 2026-08-12 | test(openai): harden websocket transcript allocation coverage | `sdk/api` |  |  |
| v7.2.133 | `e7c3fb19834d` | 2026-08-12 | test: make race detector suite deterministic | `internal/watcher`, `sdk/api` |  |  |
| v7.2.133 | `f9bd9def2b6a` | 2026-08-12 | perf(openai): reduce websocket transcript merge allocations | `sdk/api` |  |  |
| v7.2.133 | `297139cc8d05` | 2026-08-14 | fix(claude): keep tool_result blocks first when injecting currentDate | `internal/runtime` |  |  |
| v7.2.133 | `124dab6cc198` | 2026-08-15 | perf(translator): batch assemble response arrays and merge multi-choice parts | `internal/translator` |  |  |
| v7.2.133 | `616d1b11e5a0` | 2026-08-15 | fix(claude,gemini,antigravity): centralize tool-call ID generation and harden signature sanitization | `internal/translator` |  |  |
| v7.2.133 | `1ecb7df228cf` | 2026-08-15 | fix(claude): deduplicate duplicate tool outputs in OpenAI-to-Claude request conversion | `internal/translator` |  |  |
| v7.2.133 | `c1ff55fc2f1e` | 2026-08-15 | chore(models): remove GPT-5.6 Sol Work Mode registrations from model registry config | `internal/registry` |  |  |
| v7.2.133 | `10afcc8c7751` | 2026-08-15 | fix(openai): propagate environment context and sanitize antigravity generation config in interaction adapters | `internal/translator` |  |  |
| v7.2.133 | `f53a2b6e804b` | 2026-08-15 | fix(gemini): merge conditional schema branches during JSON schema cleanup | `internal/util` |  |  |
| v7.2.133 | `9169ad56e18b` | 2026-08-15 | fix(auth): make session-affinity updates safe for rebound sessions | `sdk/cliproxy` |  |  |
| v7.2.133 | `046b59ecc519` | 2026-08-15 | feat(models): add max_completion_tokens to model definitions and responses | `internal/api`, `internal/client` |  |  |
| v7.2.133 | `dd214445ef7a` | 2026-08-15 | feat(models): add GPT-5.6 Sol Work Mode model registrations and Codex client config updates | `internal/registry` |  |  |
| v7.2.133 | `e0b4956242e5` | 2026-08-15 | fix(openai): ensure Responses usage includes token detail fields | `internal/pluginhost`, `internal/runtime`, `sdk/api` |  |  |
| v7.2.133 | `ac82bedfaf5b` | 2026-08-15 | fix(claude): add Anthropic unified rate-limit parsing helpers | `internal/runtime`, `sdk/cliproxy` |  |  |
| v7.2.133 | `203f5b1a1d84` | 2026-08-15 | fix(auth): avoid cooldown for request-scoped 401 request-faults | `sdk/cliproxy` |  |  |
| v7.2.133 | `61c4fd87f5ba` | 2026-08-15 | fix(gemini): preserve `additionalProperties:false` for Antigravity response schemas | `internal/util` |  |  |
| v7.2.133 | `7efe0a7c11ad` | 2026-08-16 | pref(claude): keep raw Claude tool IDs for deduplication in request translation | `internal/translator` |  |  |
| v7.2.133 | `8b3b304952ed` | 2026-08-16 | perf(translator): switch response translators to batched raw-array insertion via `SetRawArrayItems` | `internal/translator` |  |  |
| v7.2.133 | `0c58c3c83d3c` | 2026-08-16 | perf(translator): batch assemble translation arrays before writing JSON | `internal/translator` |  |  |
| v7.2.133 | `a581838082d8` | 2026-08-16 | fix(gemini): simplify OpenAI chat response content and choices serialization | `internal/translator` |  |  |
| v7.2.134 | `b8fbe70b37a6` | 2026-08-16 | fix(auth): stop old selectors when replacing manager selector and harden cache stop concurrency | `sdk/cliproxy` |  |  |
| v7.2.134 | `75e2454e7272` | 2026-08-16 | docs(readme): remove obsolete ecosystem links for PPAP and Alex | `README.md`, `README_CN.md`, `README_JA.md` |  |  |
| v7.2.134 | `00c4377a21b8` | 2026-08-16 | fix(auth): share session affinity across model variant suffixes | `sdk/cliproxy` |  |  |
| v7.2.134 | `aa10847e1271` | 2026-08-16 | feat(auth): add request-scoped error action handling in conductor | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
| v7.2.134 | `7d55d0da0dd2` | 2026-08-16 | fix(antigravity): preserve schema semantics | `internal/runtime`, `internal/util` |  |  |
| v7.2.134 | `e23395a92b45` | 2026-08-16 | test(antigravity): cover schema sanitizer semantics | `internal/runtime`, `internal/util` |  |  |
| v7.2.134 | `7eefab98b8e5` | 2026-08-16 | feat(claude): map OpenAI `service_tier` to Claude `speed` in responses requests | `internal/translator` |  |  |
| v7.2.134 | `92f03e68e3f1` | 2026-08-16 | fix(claude,gemini,openai): preserve upstream stream errors when no data payload is emitted | `sdk/api` |  |  |
| v7.2.135 | `745fb38dbbe6` | 2026-08-17 | chore(models): update GPT-5.6 context limits in registry metadata | `internal/registry` |  |  |
| v7.2.135 | `5bffd1514fba` | 2026-08-17 | Refactor cooling management across configuration and handlers | `config.example.yaml`, `internal/api`, `internal/config` +2 |  |  |
