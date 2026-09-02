# Upstream ledger — cliproxyapi v7.2.140..v7.2.147

- generated: 2026-09-02T06:35:38Z
- dispositions filled: 2026-09-02T07:00:00Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `555ad5a5291d`
- non-merge commits: 78
- checkpoint: `refs/upstream-checkpoints/cliproxyapi/v7.2.147` (`17a65ee5470f`)
- policy: `docs/upstream/cliproxyapi-checkpoint.json` `scope_policy`

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

## Slice summary

| Slice | Disposition | Count |
| --- | --- | --- |
| xai-runtime-fixes | `adapt` | 6 |
| gemini-schema-normalize | `adapt` | 2 |
| gemini-antigravity-protocol | `adapt` | 8 |
| claude-translator-fixes | `adapt` | 10 |
| claude-allowed-warning | `defer` | 1 |
| codex-openai-compat | `adapt` | 6 |
| auth-rotation-fairness | `adapt` | 2 |
| session-subagent-affinity | `adapt` | 1 |
| session-lcp-merkle | `adapt` | 6 |
| session-cache-lru | `adapt` | 1 |
| auth-epochs-generations | `adapt` | 1 |
| auth-error-propagation | `adapt` | 3 |
| proxy-http11-alpn | `adapt` | 1 |
| video-multi-ref-duration | `adapt` | 1 |
| kimi-schema-normalize | `adapt` | 1 |
| quota-signals | `adapt` | 1 |
| ttft-measurement | `adapt` | 1 |
| usage-streaming-state | `adapt` | 1 |
| pluginhost-hot-reload-ws-usage | `reject` | 4 |
| branding-docs | `reject` | 8 |
| github-token-assets | `reject` | 1 |
| test-hygiene | `reject` | 4 |
| management-post-persist | `superseded-locally` | 1 |
| home-401-diagnostics | `defer` | 4 |
| home-port-normalize | `defer` | 3 |
| gitstore-recovery | `defer` | 1 |

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.141 | `9d0a60bfc361` | 2026-08-23 | fix(gemini,antigravity): preserve function/tool results as raw strings | `internal/translator` | `adapt` | gemini-antigravity-protocol: upstream antigravity_openai_request.go keeps function responses as raw strings; local antigravity_openai_request.go:332 still parses JSON strings |
| v7.2.141 | `6f4b6dc5f53d` | 2026-08-23 | fix(xai): keep forced image_generation from other tools | `internal/runtime` | `adapt` | xai-runtime-fixes: upstream xai_executor_request.go:699 `xaiKeepOnlyImageGenerationTools`; local xai_executor.go:1194 still rewrites to allowed_tools |
| v7.2.141 | `fba1ff24ac60` | 2026-08-23 | fix(xai): preserve auto when rewriting image-only allowed_tools | `internal/runtime` | `adapt` | xai-runtime-fixes: upstream preserves mode=auto; local always mode=required at xai_executor.go:1199 |
| v7.2.141 | `d2742c5f37d8` | 2026-08-23 | fix(xai): map image_generation tool_choice to required | `internal/runtime` | `adapt` | xai-runtime-fixes: upstream `xaiSetToolChoiceString(..., "required")` at xai_executor_request.go:699; local allowed_tools object |
| v7.2.141 | `85749db9bcd3` | 2026-08-24 | docs(readme): add APIMart sponsor to Japanese README | `README.md`, `README_CN.md`, `README_JA.md` +2 | `reject` | branding-docs: invariant 5 — llmhub branding/release contracts are not upstream's |
| v7.2.141 | `5fead38f0d5a` | 2026-08-24 | docs: remove VisionCoder sponsor section from all READMEs | `README.md`, `README_CN.md`, `README_JA.md` +1 | `reject` | branding-docs: invariant 5 |
| v7.2.141 | `cef351a4644b` | 2026-08-24 | docs(readme): add APIMart sponsor | `README.md`, `README_CN.md`, `assets/Apimart-en.png` +1 | `reject` | branding-docs: invariant 5 |
| v7.2.142 | `ca601db05d85` | 2026-08-24 | feat: observe upstream provider quota signals (#5211) | `internal/api`, `internal/logging`, `internal/runtime` +1 | `adapt` | quota-signals: upstream QuotaState.ObserveResponseHeadersForProvider at sdk/cliproxy/auth/quota_signals.go:37; local file ABSENT |
| v7.2.142 | `1f53b2eb03b9` | 2026-08-25 | fix(auth): keep credential rotation fair when candidates are filtered | `sdk/cliproxy` | `adapt` | auth-rotation-fairness: upstream successorIndex in selector.go; local RoundRobinSelector at selector.go:323 uses monotonic cursor |
| v7.2.142 | `998dcfeba2f1` | 2026-08-25 | fix(antigravity): safely synthesize terminal finish reasons (#5230) | `internal/runtime`, `internal/translator` | `adapt` | gemini-antigravity-protocol: upstream antigravity_openai_response.go synthesizes finish on [DONE]; local antigravity_openai_response.go:62 does not |
| v7.2.142 | `f2b1996b3f95` | 2026-08-25 | fix(gemini,antigravity): align parallel tool results with preceding tool calls | `internal/runtime`, `internal/translator` | `adapt` | gemini-antigravity-protocol: upstream AlignClaudeToolResults; local ABSENT |
| v7.2.142 | `80de9015502e` | 2026-08-25 | fix: preserve multi-reference video durations | `sdk/api` | `adapt` | video-multi-ref-duration: upstream deleted 10s clamp; local openai_videos_handlers.go:350 still caps duration when referenceImages>0 |
| v7.2.142 | `adf052984f8b` | 2026-08-25 | fix(antigravity): remove cross-endpoint fallback (#5209) (#5228) | `internal/runtime` | `adapt` | gemini-antigravity-protocol: local still calls antigravityBaseURLFallbackOrder at antigravity_executor.go:619 |
| v7.2.142 | `e1bf89395687` | 2026-08-25 | feat(github): resolve GitHub token for release checks and asset updates | `internal/api`, `internal/managementasset`, `internal/util` | `reject` | github-token-assets: invariant 5; local managementasset/updater.go is a filename-constant stub |
| v7.2.142 | `ba510f85a21c` | 2026-08-25 | fix(pluginhost): add plugin quiesce handling with safe rollback during hot reload | `internal/pluginhost`, `sdk/pluginabi` | `reject` | pluginhost-hot-reload-ws-usage: ní declined; local pluginhost is aliases only at lifecycle.go:5 |
| v7.2.143 | `2555cde2f1c5` | 2026-08-26 | fix(xai): inline local refs and broaden codex app tool normalization | `internal/runtime`, `internal/util` | `adapt` | xai-runtime-fixes: upstream InlineLocalRefs; local gemini_schema.go has no InlineLocalRefs |
| v7.2.143 | `1502ac826dcf` | 2026-08-26 | fix(home): allow home config to override default port | `cmd/server` | `defer` | home-port-normalize: later reverted; HomeConfig at internal/config/home.go:4 has no NormalizeHomePort |
| v7.2.143 | `ba200aefa0c6` | 2026-08-26 | fix: update Infistar registration links in README files | `README.md`, `README_CN.md`, `README_JA.md` | `reject` | branding-docs: invariant 5 |
| v7.2.143 | `4b5f1eab25fc` | 2026-08-27 | feat(plugin): support observing upstream websocket response events | `internal/pluginhost`, `internal/runtime`, `sdk/api` +3 | `reject` | pluginhost-hot-reload-ws-usage: ní declined; websocket observer helpers ABSENT locally |
| v7.2.143 | `b7f6c15f83e4` | 2026-08-27 | fix(pluginhost): detach context and log rpc failures in usage handling | `internal/pluginhost` | `reject` | pluginhost-hot-reload-ws-usage: ní declined |
| v7.2.143 | `6f6856e7849e` | 2026-08-27 | fix(gemini): map cached content tokens to claude cache read usage | `internal/translator` | `adapt` | gemini-antigravity-protocol: upstream gemini_claude_response.go maps cachedContentTokenCount; local gemini/claude translator has no cache_read_input_tokens |
| v7.2.143 | `4fa1de2f9bf5` | 2026-08-27 | feat(claude): support server-side web search translation for openai responses | `internal/translator` | `adapt` | claude-translator-fixes: upstream claude_openai-responses_web_search.go:110 convertResponsesWebSearchCallToClaudeBlocks; local request.go:684 tool-def only |
| v7.2.143 | `cb8746fb6365` | 2026-08-27 | fix(claude): use zero-based sequential index for streamed tool calls | `internal/translator` | `adapt` | claude-translator-fixes: upstream NextToolCallIndex; local claude_openai_response.go uses content block index; NextToolCallIndex ABSENT |
| v7.2.143 | `adac1e5816ee` | 2026-08-27 | fix(gemini): improve schema normalization for unions and unsupported constraints | `internal/util` | `adapt` | gemini-schema-normalize: upstream flattenAnyOfOneOf merges parent properties; local flattenAnyOfOneOf at gemini_schema.go:700 overwrites parent |
| v7.2.143 | `9b88808fc75a` | 2026-08-27 | fix(xai): fold namespace tools and restore dispatcher tool calls | `internal/runtime` | `adapt` | xai-runtime-fixes: upstream namespace fold/restorer; local xai_executor.go has namespace constants but no fold-when->200 dispatcher |
| v7.2.144 | `f8c45c30c5b3` | 2026-08-27 | feat(codex): map claude output_config format to text format | `internal/translator` | `adapt` | codex-openai-compat: upstream codex_claude_request.go maps output_config.format; local codex_claude_request.go:338 only reads output_config.effort |
| v7.2.144 | `8ee9add75daa` | 2026-08-27 | fix(claude): drop trailing assistant prefill for fable models | `internal/translator` | `adapt` | claude-translator-fixes: dropUnsupportedClaudeAssistantPrefill ABSENT locally |
| v7.2.144 | `95f83a8d96cf` | 2026-08-27 | fix(claude): treat allowed_warning as allowed in unified rate limit checks | `internal/runtime` | `defer` | claude-allowed-warning: upstream helps/claude_ratelimit.go:42 isClaudeWindowAllowed; local file ABSENT |
| v7.2.144 | `06997df44a09` | 2026-08-27 | fix(gemini): cache trailing text thought signatures instead of emitting carriers | `internal/translator` | `adapt` | gemini-antigravity-protocol: upstream antigravity_claude_response.go caches trailing signatures; local :171 still emits carriers |
| v7.2.144 | `fcea738f74ee` | 2026-08-27 | fix(codex): use codex status error for websocket handshake rejections | `internal/runtime` | `adapt` | codex-openai-compat: local CodexWebsocketsExecutor.Execute at codex_websockets_executor.go returns raw statusErr not newCodexStatusErr |
| v7.2.144 | `1cc72b9d13c4` | 2026-08-27 | fix(codex): filter extended reasoning levels for older client versions | `internal/api`, `internal/client`, `sdk/api` | `adapt` | codex-openai-compat: local models.go/codex_client_models.go ignore client_version; no max/ultra filter |
| v7.2.144 | `d36b776c790a` | 2026-08-28 | fix(test): improve cross-platform compatibility across unit tests | `internal/api`, `internal/managementasset`, `internal/runtime` +1 | `reject` | test-hygiene: no product behavior |
| v7.2.145 | `7bc16ee3dbcf` | 2026-08-27 | fix(auth): preserve round-robin successor across ready view rebuilds | `sdk/cliproxy` | `adapt` | auth-rotation-fairness: local snapshotReadyViewCursors at scheduler.go:112 restores numeric cursor only; lastPicked/scheduledSuccessorIndex ABSENT |
| v7.2.145 | `d9cea8904b14` | 2026-08-28 | docs: remove BmoPlus sponsorship | `README.md`, `README_CN.md`, `README_JA.md` +1 | `reject` | branding-docs: invariant 5 |
| v7.2.145 | `3db0a86c6015` | 2026-08-28 | fix(management): synthesize persisted auth record before invoking post-persist hook | `internal/api` | `superseded-locally` | management-post-persist: local upsertAuthRecord at auth_files.go:1394 already persists via Postgres-backed manager |
| v7.2.145 | `8dd78042e062` | 2026-08-28 | fix(proxyutil): enforce HTTP/1.1 ALPN and configure TLS dial context for HTTPS proxies | `sdk/proxyutil` | `adapt` | proxy-http11-alpn: upstream buildHTTPSProxyDialTLSContext NextProtos http/1.1 at proxy.go:158; local DialContext at proxy.go:173 uses tls.Config{ServerName} with no ALPN |
| v7.2.145 | `b5cde4ba4276` | 2026-08-28 | fix(gemini): add default items schema for tool array definitions | `internal/util` | `adapt` | gemini-schema-normalize: upstream repairSchemaNode addMissingArrayItems; local normalizeMalformedSchemaObjects at gemini_schema.go:171 has no items default |
| v7.2.145 | `3e20d662a70e` | 2026-08-28 | Revert "fix(home): allow home config to override default port" | `cmd/server` | `defer` | home-port-normalize: revert of 1502ac826dcf; stay with deferred home-port work |
| v7.2.146 | `d198817c61b5` | 2026-08-26 | docs: add WebBrain to related projects | `README.md`, `README_CN.md`, `README_JA.md` | `reject` | branding-docs: invariant 5 |
| v7.2.146 | `9a2201c36a0a` | 2026-08-28 | fix(auth): forward Home unauthorized upstream errors | `internal/api`, `internal/client`, `internal/logging` +4 | `defer` | home-401-diagnostics: prior defer home-401-refresh-revert; local home_refresh.go still RefreshAuthViaHome |
| v7.2.146 | `bc918ab27664` | 2026-08-28 | fix(runtime): log safe home refresh error types | `internal/runtime` | `defer` | home-401-diagnostics: same Home control-plane cluster |
| v7.2.146 | `e4a8f9891344` | 2026-08-28 | fix(logging): enhance error diagnostics and logging for home refresh operations | `internal/logging`, `internal/runtime`, `sdk/cliproxy` | `defer` | home-401-diagnostics: same cluster |
| v7.2.146 | `dd5f9e74e4b8` | 2026-08-28 | fix(kimi): normalize tool and function parameter schemas | `internal/runtime` | `adapt` | kimi-schema-normalize: local kimi_executor.go has normalizeKimiToolMessageLinks at :347 but no $ref inline / root type:object schema normalize |
| v7.2.146 | `4b2beb3da153` | 2026-08-28 | feat(executor): measure effective TTFT with protocol-aware token classification (#5313) | `internal/runtime` | `adapt` | ttft-measurement: upstream ObserveChatTokenEvent at chat_ttft_helpers.go:105; local helps/ has no *_ttft_helpers.go |
| v7.2.146 | `be1763e59e2b` | 2026-08-28 | fix(claude): fallback to array index when tool call index is omitted | `internal/translator` | `adapt` | claude-translator-fixes: openai_claude_response.go local lacks arrayIndex fallback |
| v7.2.146 | `677dbe1dc5a5` | 2026-08-28 | fix(claude): emit message_delta when openai streaming finish reason is omitted | `internal/translator` | `adapt` | claude-translator-fixes: local openai_claude_response.go omits message_delta when FinishReason empty |
| v7.2.146 | `5ff4a31eaf82` | 2026-08-29 | feat(claude): support advisor tool beta header | `internal/runtime`, `internal/util` | `adapt` | claude-translator-fixes: claudeAdvisorToolBeta ABSENT; local claude_executor.go has tool_choice thinking disable only |
| v7.2.146 | `6a489fa84d93` | 2026-08-29 | fix(auth): prefer errors from upstream attempts | `internal/pluginhost`, `internal/runtime`, `internal/wsrelay` +2 | `adapt` | auth-error-propagation: upstream preferredExecutionAttemptError / WithCause; local errors.go has no errorWithCause |
| v7.2.146 | `c350d3f52057` | 2026-08-29 | fix(claude): strip trailing thinking blocks from assistant messages | `internal/translator` | `adapt` | claude-translator-fixes: stripTrailingClaudeThinkingBlocks ABSENT |
| v7.2.146 | `6f25b9a1495f` | 2026-08-29 | fix(claude): drop unsupported assistant prefill for opus-5 and sonnet-4-6 | `internal/translator` | `adapt` | claude-translator-fixes: dropUnsupportedClaudeAssistantPrefill ABSENT |
| v7.2.146 | `07d8156375ca` | 2026-08-29 | fix(claude): collapse consecutive thinking blocks in openai responses translation | `internal/translator` | `adapt` | claude-translator-fixes: appendReasoning collapse ABSENT |
| v7.2.146 | `d31b15916d15` | 2026-08-30 | feat(executor): support token usage parsing for plugin executors | `internal/pluginhost`, `internal/runtime`, `sdk/api` | `reject` | pluginhost-hot-reload-ws-usage: ní declined |
| v7.2.146 | `e4119f83b448` | 2026-08-30 | fix(xai): use chat base url resolution for image and video requests | `internal/auth`, `internal/runtime` | `adapt` | xai-runtime-fixes: local executeImages at xai_executor.go:199 does not share chat-proxy base URL resolution |
| v7.2.146 | `6c6473f8998d` | 2026-08-30 | fix(openai): ignore empty tool calls array in responses translation | `internal/translator` | `adapt` | codex-openai-compat: local openai_openai-responses_response.go:599 checks tcs.IsArray() without len>0 |
| v7.2.146 | `12b88f3ad614` | 2026-08-30 | fix(auth): synchronize auth lifecycle and registry state with epochs and generations | `internal/registry`, `sdk/cliproxy` | `adapt` | auth-epochs-generations: RegistrationEpoch/Generation ABSENT on local Auth at types.go; no clientEpochs in model_registry.go |
| v7.2.146 | `7ab999a80fe8` | 2026-08-30 | fix(config): add NormalizeHomePort function and update home port handling | `cmd/server`, `internal/config` | `defer` | home-port-normalize: NormalizeHomePort ABSENT; llmhub uses LLMHUB_PORT process override |
| v7.2.146 | `02e3d33c49ab` | 2026-08-30 | chore(codex): update codex client model definitions | `internal/registry` | `adapt` | codex-openai-compat: local codex_client_models.json missing multi_agent_reasoning_effort and related fields |
| v7.2.147 | `ff811577eb5f` | 2026-08-31 | fix(auth): isolate subagent session affinity for antigravity and gemini | `sdk/cliproxy` | `adapt` | session-subagent-affinity: isSubagentSession / allowsSubagentAuthInheritance ABSENT; local SessionAffinitySelector.Pick at selector.go:804 |
| v7.2.147 | `9721d9939ed5` | 2026-08-31 | feat(usage): track streaming execution state in usage records | `internal/redisqueue`, `internal/runtime`, `sdk/api` +1 | `adapt` | usage-streaming-state: local sdk/cliproxy/usage/manager.go record struct has no Stream field |
| v7.2.147 | `5755d00b1bed` | 2026-08-31 | refactor(session): prune in-memory session tree store in favor of flat KV affinity and pure info extraction | `sdk/cliproxy` | `adapt` | session-lcp-merkle: follow-up to #5202; local has no sdk/cliproxy/session/ |
| v7.2.147 | `9fdc46058592` | 2026-08-31 | fix(claude): preserve tool pairing across intervening system messages | `internal/runtime`, `internal/translator` | `adapt` | claude-translator-fixes: AlignClaudeToolResults ABSENT in local translator/common |
| v7.2.147 | `c99ce1212370` | 2026-08-31 | fix(session): harden depth-cap index cleanup, bound lcp fingerprints, and support nested payloads (#5202) | `sdk/cliproxy` | `adapt` | session-lcp-merkle: upstream lcp.go MerklePrefixMatcher; local package ABSENT |
| v7.2.147 | `c2021cd2fe7d` | 2026-08-31 | perf(test): eliminate physical sleeps and normalize mock timeouts in xai and home (#5328) | `internal/auth`, `internal/home` | `reject` | test-hygiene: no product behavior |
| v7.2.147 | `100c564313bb` | 2026-08-31 | fix(session): harden nil fallback, normalize agent id, and support nested antigravity payloads (#5202) | `sdk/cliproxy` | `adapt` | session-lcp-merkle: local session package ABSENT |
| v7.2.147 | `8211388946e5` | 2026-08-31 | feat(session): implement Merkle LCP session affinity, session tree, and distributed Home hierarchy (#5202) | `internal/home`, `sdk/cliproxy` | `adapt` | session-lcp-merkle: upstream sdk/cliproxy/session/lcp.go:746 MatchFingerprints; local SessionAffinitySelector uses explicit IDs only at selector.go:804. Constraint: additive SDK package, keep existing ID affinity |
| v7.2.147 | `81e1b5374f99` | 2026-08-31 | docs(readme): remove LMU sponsor | `README.md`, `README_CN.md`, `README_JA.md` +1 | `reject` | branding-docs: invariant 5 |
| v7.2.147 | `b908ed5d8644` | 2026-08-31 | fix(store): close git repos during recovery | `internal/runtime`, `internal/store` | `defer` | gitstore-recovery: prior defer; Postgres is authoritative (invariant 1) |
| v7.2.147 | `17a65ee5470f` | 2026-09-01 | fix(codex): support reasoning_text in openai chat completion responses | `internal/translator` | `adapt` | codex-openai-compat: upstream codex_openai_response.go handles reasoning_text.delta/done; local ConvertCodexResponseToOpenAI only reasoning_summary_text |
| v7.2.147 | `dc0f7a594b11` | 2026-09-01 | fix(session): derive LCP session ID from rolling prefix keys and harden matcher | `sdk/cliproxy` | `adapt` | session-lcp-merkle: local package ABSENT |
| v7.2.147 | `5679bbf330aa` | 2026-09-01 | fix(auth): optimize session cache LRU capacity eviction | `sdk/cliproxy` | `adapt` | session-cache-lru: local SessionCache at session_cache.go:14 is unbounded TTL map; no evictionOrder/maxEntries |
| v7.2.147 | `acc1500f6078` | 2026-09-01 | test(executor): update antigravity interactions test to use valid model | `internal/runtime` | `adapt` | gemini-antigravity-protocol: test fixture for remaining AG protocol ports |
| v7.2.147 | `35e3d97daccd` | 2026-09-01 | fix(registry): remove defunct gemini-3-flash-agent model from antigravity provider | `internal/registry`, `sdk/cliproxy` | `adapt` | gemini-antigravity-protocol: drop defunct model from local models.json |
| v7.2.147 | `fae57a3a881c` | 2026-09-01 | fix(session): support identity and subagent extraction from nested request payloads | `sdk/cliproxy` | `adapt` | session-lcp-merkle: nested payload identity; local extractSessionIDs at selector.go:1025 is header/body hash only |
| v7.2.147 | `22a57401a145` | 2026-09-01 | fix(tests): improve test reliability by avoiding wall-clock time in TTL and cache-eviction tests | `internal/api`, `internal/cache` | `reject` | test-hygiene: no product behavior |
| v7.2.147 | `dbcc4ecd6f53` | 2026-09-01 | feat(home): enhance homeDispatchConn with untrack method and improve connection closure handling | `internal/home` | `defer` | home-401-diagnostics: Home control-plane; stay deferred with home-401 cluster |
| v7.2.147 | `8bcae8cf207f` | 2026-09-01 | docs: document timer granularity and sleep convention in AGENTS.md | `AGENTS.md` | `reject` | branding-docs / test-hygiene: upstream repo convention, not llmhub behavior |
| v7.2.147 | `707078514028` | 2026-09-01 | fix(auth): preserve upstream status codes in antigravity auth errors | `internal/auth`, `internal/runtime`, `sdk/cliproxy` | `adapt` | auth-error-propagation: upstream HTTPStatusError in antigravity/auth.go; local ExchangeCodeForTokens uses generic fmt.Errorf |
| v7.2.147 | `f2e2d713b29d` | 2026-09-01 | fix(auth): propagate upstream error causes in auth selection failures | `sdk/api`, `sdk/cliproxy` | `adapt` | auth-error-propagation: WithCause / ExtractUpstreamErrorSummary ABSENT locally |
