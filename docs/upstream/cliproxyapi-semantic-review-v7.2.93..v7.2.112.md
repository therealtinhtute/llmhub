# Semantic-review path dispositions — cliproxyapi v7.2.93..v7.2.112

- generated_from: `docs/upstream/cliproxyapi-gap-v7.2.93..v7.2.112.json` + `docs/upstream/cliproxyapi-ledger-v7.2.93..v7.2.112.md`
- rule: if any touching commit is `adapt`, path disposition is `adapt`; otherwise `defer` > `reject` > `already-present`.
- semantic-review paths: 181

| Disposition | Paths |
| --- | --- |
| `adapt` | 109 |
| `already-present` | 64 |
| `reject` | 2 |
| `defer` | 6 |

| Disposition | Upstream path | Local path | Slices | Commits |
| --- | --- | --- | --- | --- |
| `adapt` | `go.mod` | `go.mod` | codex-live, executor-runtime-fixes | `49be36aef6a6`, `bda79b21bb94`, `f3e36f19c088` |
| `adapt` | `go.sum` | `go.sum` | codex-live, executor-runtime-fixes | `bda79b21bb94`, `f3e36f19c088` |
| `adapt` | `internal/api/handlers/management/auth_files.go` | `internal/api/handlers/management/auth_files.go` | credential-weight-api, management-api, plugin-platform | `e8e39526b3c9`, `fe4ae4989c4d`, `d25b6b41e85e` |
| `adapt` | `internal/api/server.go` | `internal/api/server.go` | codex-live, codex-model-resolution, codex-multiagent-v2, home-parity, model-catalog, plugin-platform | `fe4ae4989c4d`, `bda79b21bb94`, `46172dd452b8`, `0296600be60a`, `71d591296b93`, `84bf9376e5a5`, `3ecd4afe808a`, `f71ec0eb6776` |
| `adapt` | `internal/api/server_test.go` | `internal/api/server_test.go` | claude-cloaking-toggle, codex-live, codex-model-resolution, home-parity, model-catalog, server-routing | `7b233fa31604`, `69144785622a`, `46172dd452b8`, `0296600be60a`, `3ecd4afe808a`, `f71ec0eb6776` |
| `adapt` | `internal/client/codex/live/media.go` | `internal/client/codex/live/media.go` | codex-live, logging-metadata, watcher-config-diff | `f8dffa0522c6`, `49be36aef6a6`, `ebc74469292c`, `bda79b21bb94` |
| `adapt` | `internal/client/codex/live/media_test.go` | `internal/client/codex/live/media_test.go` | codex-live, logging-metadata, watcher-config-diff | `f8dffa0522c6`, `49be36aef6a6`, `ebc74469292c`, `bda79b21bb94` |
| `adapt` | `internal/client/codex/live/tcp_proxy.go` | `internal/client/codex/live/tcp_proxy.go` | codex-live, logging-metadata | `f8dffa0522c6`, `49be36aef6a6` |
| `adapt` | `internal/client/codex/live/tcp_proxy_test.go` | `internal/client/codex/live/tcp_proxy_test.go` | codex-live, logging-metadata | `f8dffa0522c6`, `49be36aef6a6` |
| `adapt` | `internal/config/codex_websocket_header_defaults_test.go` | `internal/config/codex_websocket_header_defaults_test.go` | codex-cloaking-toggle, codex-multiagent-v2 | `a80e8082ef75`, `84bf9376e5a5` |
| `adapt` | `internal/config/config.go` | `internal/config/config.go` | codex-live, codex-multiagent-v2, executor-runtime-fixes, home-parity, plugin-platform, watcher-config-diff | `8423cce2d100`, `fe4ae4989c4d`, `ebc74469292c`, `bda79b21bb94`, `84bf9376e5a5`, `3ecd4afe808a` |
| `adapt` | `internal/config/sdk_config.go` | `internal/config/sdk_config.go` | claude-cloaking-toggle, codex-multiagent-v2 | `69144785622a`, `84bf9376e5a5` |
| `adapt` | `internal/config/vertex_compat.go` | `internal/config/vertex_compat.go` | thinking-consolidation, weighted-scheduler | `f32291436ad9`, `5dcca50fd9d9` |
| `adapt` | `internal/logging/global_logger.go` | `internal/logging/global_logger.go` | logging-metadata | `f8dffa0522c6` |
| `adapt` | `internal/redisqueue/plugin.go` | `internal/redisqueue/plugin.go` | home-401-refresh, home-parity, logging-metadata, usage-normalization | `a63da8ae76b1`, `4db8e1202942`, `c9417c8ae9b1`, `fe8a616aa303`, `416a08017447` |
| `adapt` | `internal/redisqueue/plugin_test.go` | `internal/redisqueue/plugin_test.go` | logging-metadata, usage-normalization | `c9417c8ae9b1`, `42f36b94e080`, `fe8a616aa303`, `416a08017447` |
| `adapt` | `internal/runtime/executor/aistudio_executor.go` | `internal/runtime/executor/aistudio_executor.go` | codex-multiagent-v2, executor-runtime-fixes, thinking-summary-visibility | `0c2ec7da235d`, `5d307c195dd2`, `b3046d29b985`, `84bf9376e5a5`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/antigravity_executor.go` | `internal/runtime/executor/antigravity_executor.go` | antigravity-replay, codex-multiagent-v2, executor-runtime-fixes, plugin-platform | `fe4ae4989c4d`, `f6c32ec3ffe7`, `a4d18cb04acc`, `72886356dd9b`, `84bf9376e5a5`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/antigravity_executor_buildrequest_test.go` | `internal/runtime/executor/antigravity_executor_buildrequest_test.go` | executor-runtime-fixes | `f6c32ec3ffe7` |
| `adapt` | `internal/runtime/executor/antigravity_reasoning_replay.go` | `internal/runtime/executor/antigravity_reasoning_replay.go` | antigravity-replay, executor-runtime-fixes, home-parity | `20784c67ff52`, `a06e21b43999`, `d2c0c58b7516`, `e47c43650c5a`, `f6c32ec3ffe7`, `72886356dd9b` |
| `adapt` | `internal/runtime/executor/claude_executor.go` | `internal/runtime/executor/claude_executor.go` | codex-multiagent-v2, executor-runtime-fixes, plugin-platform | `fe4ae4989c4d`, `29406436c3e8`, `95d5b2485f7d`, `84bf9376e5a5` |
| `adapt` | `internal/runtime/executor/claude_executor_test.go` | `internal/runtime/executor/claude_executor_test.go` | executor-runtime-fixes, server-routing, thinking-summary-visibility | `b3046d29b985`, `c405398a48df`, `57ef7842242a`, `08e5515703ab`, `29406436c3e8` |
| `adapt` | `internal/runtime/executor/codex_executor.go` | `internal/runtime/executor/codex_executor.go` | codex-model-resolution, codex-multiagent-v2, executor-runtime-fixes, plugin-platform, responses-websocket | `fe4ae4989c4d`, `f6c32ec3ffe7`, `6cdd51fce1d2`, `84bf9376e5a5`, `e05ae0942507`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/codex_executor_cache_test.go` | `internal/runtime/executor/codex_executor_cache_test.go` | executor-runtime-fixes | `f6c32ec3ffe7` |
| `adapt` | `internal/runtime/executor/codex_openai_images.go` | `internal/runtime/executor/codex_openai_images.go` | thinking-summary-visibility | `0c2ec7da235d`, `b3046d29b985` |
| `adapt` | `internal/runtime/executor/codex_websockets_executor.go` | `internal/runtime/executor/codex_websockets_executor.go` | codex-multiagent-v2, executor-runtime-fixes, home-parity, plugin-platform, responses-websocket | `fe4ae4989c4d`, `f6c32ec3ffe7`, `84bf9376e5a5`, `3ecd4afe808a`, `e05ae0942507`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/codex_websockets_executor_test.go` | `internal/runtime/executor/codex_websockets_executor_test.go` | codex-cloaking-toggle, executor-runtime-fixes, home-parity, responses-websocket | `a80e8082ef75`, `f6c32ec3ffe7`, `3ecd4afe808a`, `e05ae0942507` |
| `adapt` | `internal/runtime/executor/gemini_executor.go` | `internal/runtime/executor/gemini_executor.go` | codex-multiagent-v2, executor-runtime-fixes, thinking-consolidation | `f32291436ad9`, `84bf9376e5a5`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/gemini_executor_test.go` | `internal/runtime/executor/gemini_executor_test.go` | thinking-summary-visibility | `b3046d29b985` |
| `adapt` | `internal/runtime/executor/gemini_vertex_executor.go` | `internal/runtime/executor/gemini_vertex_executor.go` | codex-multiagent-v2, executor-runtime-fixes, thinking-consolidation | `f32291436ad9`, `84bf9376e5a5`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/helps/claude_input_tokens.go` | `internal/runtime/executor/helps/claude_input_tokens.go` | executor-runtime-fixes | `57ef7842242a`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/helps/claude_input_tokens_test.go` | `internal/runtime/executor/helps/claude_input_tokens_test.go` | executor-runtime-fixes | `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/helps/derived_session.go` | `internal/runtime/executor/helps/derived_session.go` | executor-runtime-fixes | `f6c32ec3ffe7` |
| `adapt` | `internal/runtime/executor/helps/derived_session_test.go` | `internal/runtime/executor/helps/derived_session_test.go` | executor-runtime-fixes | `f6c32ec3ffe7` |
| `adapt` | `internal/runtime/executor/kimi_executor.go` | `internal/runtime/executor/kimi_executor.go` | codex-multiagent-v2, executor-runtime-fixes, thinking-summary-visibility | `0c2ec7da235d`, `b3046d29b985`, `57ef7842242a`, `84bf9376e5a5`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/kimi_executor_test.go` | `internal/runtime/executor/kimi_executor_test.go` | executor-runtime-fixes | `57ef7842242a` |
| `adapt` | `internal/runtime/executor/openai_compat_executor.go` | `internal/runtime/executor/openai_compat_executor.go` | codex-multiagent-v2, executor-runtime-fixes, thinking-consolidation | `f32291436ad9`, `84bf9376e5a5`, `f3e36f19c088` |
| `adapt` | `internal/runtime/executor/xai_executor.go` | `internal/runtime/executor/xai_executor.go` | codex-multiagent-v2, executor-runtime-fixes, plugin-platform | `8423cce2d100`, `fe4ae4989c4d`, `f6c32ec3ffe7`, `41f6ea8950bb`, `84bf9376e5a5`, `f3e36f19c088`, `cb110ad4fa86` |
| `adapt` | `internal/runtime/executor/xai_executor_test.go` | `internal/runtime/executor/xai_executor_test.go` | codex-multiagent-v2, executor-runtime-fixes | `8423cce2d100`, `90c2ff90de39`, `f6c32ec3ffe7`, `41f6ea8950bb`, `84bf9376e5a5`, `cb110ad4fa86` |
| `adapt` | `internal/signature/gemini_validation.go` | `internal/signature/gemini_validation.go` | antigravity-replay, claude-cais-signatures | `5b2890b38c22`, `72886356dd9b` |
| `adapt` | `internal/signature/gemini_validation_test.go` | `internal/signature/gemini_validation_test.go` | antigravity-replay, claude-cais-signatures | `5b2890b38c22`, `72886356dd9b` |
| `adapt` | `internal/store/gitstore.go` | `internal/store/gitstore.go` | gitstore-recovery, watcher-delete-guard, weighted-scheduler | `a2ff6914bb49`, `5dcca50fd9d9`, `e49b1cc76fe6` |
| `adapt` | `internal/store/gitstore_test.go` | `internal/store/gitstore_test.go` | gitstore-recovery, watcher-delete-guard | `a2ff6914bb49`, `e49b1cc76fe6` |
| `adapt` | `internal/thinking/apply.go` | `internal/thinking/apply.go` | thinking-consolidation, thinking-summary-visibility | `c4dcd8703ad9`, `0c2ec7da235d`, `87ceaf83bb70`, `b3046d29b985`, `f32291436ad9` |
| `adapt` | `internal/thinking/provider/antigravity/apply.go` | `internal/thinking/provider/antigravity/apply.go` | thinking-summary-visibility | `87ceaf83bb70`, `b3046d29b985` |
| `adapt` | `internal/thinking/provider/claude/apply.go` | `internal/thinking/provider/claude/apply.go` | thinking-summary-visibility | `b3046d29b985` |
| `adapt` | `internal/thinking/provider/gemini/apply.go` | `internal/thinking/provider/gemini/apply.go` | thinking-summary-visibility | `87ceaf83bb70`, `b3046d29b985` |
| `adapt` | `internal/thinking/strip.go` | `internal/thinking/strip.go` | thinking-summary-visibility | `b3046d29b985` |
| `adapt` | `internal/translator/antigravity/claude/antigravity_claude_request.go` | `internal/translator/antigravity/claude/antigravity_claude_request.go` | antigravity-replay, thinking-summary-visibility | `b3046d29b985`, `72886356dd9b` |
| `adapt` | `internal/translator/antigravity/claude/antigravity_claude_request_test.go` | `internal/translator/antigravity/claude/antigravity_claude_request_test.go` | antigravity-replay, thinking-summary-visibility | `b3046d29b985`, `72886356dd9b` |
| `adapt` | `internal/translator/antigravity/openai/chat-completions/antigravity_openai_request.go` | `internal/translator/antigravity/openai/chat-completions/antigravity_openai_request.go` | antigravity-replay, claude-schema-normalization, thinking-summary-visibility, translator-structured-output | `b3046d29b985`, `1c1d8efdd5a4`, `5dedb303f1a4`, `62a2c0d374f2` |
| `adapt` | `internal/translator/antigravity/openai/chat-completions/antigravity_openai_response.go` | `internal/translator/antigravity/openai/chat-completions/antigravity_openai_response.go` | translator-fidelity | `27b466063365` |
| `adapt` | `internal/translator/antigravity/openai/chat-completions/antigravity_openai_response_test.go` | `internal/translator/antigravity/openai/chat-completions/antigravity_openai_response_test.go` | translator-fidelity | `27b466063365` |
| `adapt` | `internal/translator/claude/gemini/claude_gemini_request.go` | `internal/translator/claude/gemini/claude_gemini_request.go` | thinking-summary-visibility | `5d307c195dd2` |
| `adapt` | `internal/translator/claude/openai/chat-completions/claude_openai_request.go` | `internal/translator/claude/openai/chat-completions/claude_openai_request.go` | claude-schema-normalization, translator-fidelity | `4a2eb54dc6bf`, `59aa35a4344b` |
| `adapt` | `internal/translator/claude/openai/chat-completions/claude_openai_request_test.go` | `internal/translator/claude/openai/chat-completions/claude_openai_request_test.go` | claude-schema-normalization, translator-fidelity | `4a2eb54dc6bf`, `59aa35a4344b` |
| `adapt` | `internal/translator/codex/claude/codex_claude_request.go` | `internal/translator/codex/claude/codex_claude_request.go` | thinking-summary-visibility | `b3046d29b985` |
| `adapt` | `internal/translator/codex/claude/codex_claude_response.go` | `internal/translator/codex/claude/codex_claude_response.go` | translator-fidelity | `fecebcca5908`, `94cf4674d74f` |
| `adapt` | `internal/translator/codex/claude/codex_claude_response_test.go` | `internal/translator/codex/claude/codex_claude_response_test.go` | translator-fidelity | `fecebcca5908`, `94cf4674d74f` |
| `adapt` | `internal/translator/codex/gemini/codex_gemini_request.go` | `internal/translator/codex/gemini/codex_gemini_request.go` | thinking-summary-visibility | `b3046d29b985` |
| `adapt` | `internal/translator/codex/openai/chat-completions/codex_openai_request.go` | `internal/translator/codex/openai/chat-completions/codex_openai_request.go` | thinking-summary-visibility, translator-fidelity, translator-input-image | `b3046d29b985`, `181242b1c015`, `6491ce399132` |
| `adapt` | `internal/translator/codex/openai/chat-completions/codex_openai_request_test.go` | `internal/translator/codex/openai/chat-completions/codex_openai_request_test.go` | translator-fidelity, translator-input-image | `181242b1c015`, `6491ce399132` |
| `adapt` | `internal/translator/codex/openai/chat-completions/codex_openai_response.go` | `internal/translator/codex/openai/chat-completions/codex_openai_response.go` | translator-fidelity | `6491ce399132`, `58ede93e33c4` |
| `adapt` | `internal/translator/codex/openai/chat-completions/codex_openai_response_test.go` | `internal/translator/codex/openai/chat-completions/codex_openai_response_test.go` | translator-fidelity | `58ede93e33c4` |
| `adapt` | `internal/translator/gemini/claude/gemini_claude_request.go` | `internal/translator/gemini/claude/gemini_claude_request.go` | thinking-summary-visibility, translator-naming | `b3046d29b985`, `cade44b9cdee` |
| `adapt` | `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go` | `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go` | thinking-summary-visibility, translator-structured-output | `b3046d29b985`, `20f83cae910b` |
| `adapt` | `internal/translator/gemini/openai/chat-completions/gemini_openai_response.go` | `internal/translator/gemini/openai/chat-completions/gemini_openai_response.go` | translator-fidelity | `27b466063365` |
| `adapt` | `internal/translator/gemini/openai/responses/gemini_openai-responses_request.go` | `internal/translator/gemini/openai/responses/gemini_openai-responses_request.go` | antigravity-replay, thinking-summary-visibility | `b3046d29b985`, `72886356dd9b` |
| `adapt` | `internal/translator/gemini/openai/responses/gemini_openai-responses_response.go` | `internal/translator/gemini/openai/responses/gemini_openai-responses_response.go` | antigravity-replay, translator-fidelity | `4d9bf9160a87`, `72886356dd9b` |
| `adapt` | `internal/translator/gemini/openai/responses/gemini_openai-responses_response_test.go` | `internal/translator/gemini/openai/responses/gemini_openai-responses_response_test.go` | antigravity-replay, translator-fidelity | `4d9bf9160a87`, `72886356dd9b` |
| `adapt` | `internal/translator/openai/claude/openai_claude_response.go` | `internal/translator/openai/claude/openai_claude_response.go` | translator-fidelity | `f04605d84fcc` |
| `adapt` | `internal/translator/openai/claude/openai_claude_response_test.go` | `internal/translator/openai/claude/openai_claude_response_test.go` | translator-fidelity | `f04605d84fcc` |
| `adapt` | `internal/translator/openai/openai/responses/openai_openai-responses_request.go` | `internal/translator/openai/openai/responses/openai_openai-responses_request.go` | translator-fidelity | `3ad6dfe30e0b` |
| `adapt` | `internal/translator/openai/openai/responses/openai_openai-responses_request_test.go` | `internal/translator/openai/openai/responses/openai_openai-responses_request_test.go` | translator-fidelity | `3ad6dfe30e0b` |
| `adapt` | `internal/watcher/config_reload.go` | `internal/watcher/config_reload.go` | watcher-config-diff | `ebc74469292c` |
| `adapt` | `internal/watcher/diff/config_diff.go` | `internal/watcher/diff/config_diff.go` | claude-cloaking-toggle, codex-cloaking-toggle, codex-multiagent-v2, executor-runtime-fixes, watcher-config-diff | `a80e8082ef75`, `69144785622a`, `8423cce2d100`, `ebc74469292c`, `84bf9376e5a5` |
| `adapt` | `internal/watcher/diff/config_diff_test.go` | `internal/watcher/diff/config_diff_test.go` | claude-cloaking-toggle, codex-cloaking-toggle, executor-runtime-fixes, watcher-config-diff | `a80e8082ef75`, `69144785622a`, `8423cce2d100`, `ebc74469292c` |
| `adapt` | `internal/watcher/diff/model_hash.go` | `internal/watcher/diff/model_hash.go` | thinking-consolidation | `f32291436ad9` |
| `adapt` | `internal/watcher/diff/model_hash_test.go` | `internal/watcher/diff/model_hash_test.go` | thinking-consolidation | `f32291436ad9` |
| `adapt` | `internal/watcher/diff/models_summary.go` | `internal/watcher/diff/models_summary.go` | thinking-consolidation | `f32291436ad9` |
| `adapt` | `internal/watcher/diff/openai_compat.go` | `internal/watcher/diff/openai_compat.go` | watcher-config-diff | `ebc74469292c` |
| `adapt` | `internal/watcher/synthesizer/config.go` | `internal/watcher/synthesizer/config.go` | thinking-consolidation, weighted-scheduler | `f32291436ad9`, `5dcca50fd9d9` |
| `adapt` | `internal/watcher/synthesizer/config_test.go` | `internal/watcher/synthesizer/config_test.go` | thinking-consolidation, weighted-scheduler | `f32291436ad9`, `5dcca50fd9d9` |
| `adapt` | `sdk/api/handlers/handlers.go` | `sdk/api/handlers/handlers.go` | executor-runtime-fixes, home-parity, logging-metadata, plugin-platform, responses-websocket | `c9417c8ae9b1`, `fe4ae4989c4d`, `f6c32ec3ffe7`, `3ecd4afe808a`, `e05ae0942507` |
| `adapt` | `sdk/api/handlers/handlers_metadata_test.go` | `sdk/api/handlers/handlers_metadata_test.go` | executor-runtime-fixes, logging-metadata | `c9417c8ae9b1`, `f6c32ec3ffe7` |
| `adapt` | `sdk/api/handlers/openai/codex_client_models.go` | `sdk/api/handlers/openai/codex_client_models.go` | codex-multiagent-v2, model-catalog | `71d591296b93`, `84bf9376e5a5` |
| `adapt` | `sdk/api/handlers/openai/openai_responses_websocket.go` | `sdk/api/handlers/openai/openai_responses_websocket.go` | plugin-platform, responses-websocket | `fe4ae4989c4d`, `a661172b1fe6`, `e05ae0942507` |
| `adapt` | `sdk/api/handlers/openai/openai_responses_websocket_test.go` | `sdk/api/handlers/openai/openai_responses_websocket_test.go` | home-parity, responses-websocket | `3ecd4afe808a`, `a661172b1fe6`, `e05ae0942507` |
| `adapt` | `sdk/api/handlers/openai/openai_responses_websocket_toolcall_repair.go` | `sdk/api/handlers/openai/openai_responses_websocket_toolcall_repair.go` | responses-websocket | `a661172b1fe6` |
| `adapt` | `sdk/auth/filestore.go` | `sdk/auth/filestore.go` | auth-session-fixes, weighted-scheduler | `5dcca50fd9d9`, `22ec415975bb` |
| `adapt` | `sdk/auth/filestore_test.go` | `sdk/auth/filestore_test.go` | auth-session-fixes, weighted-scheduler | `5dcca50fd9d9`, `22ec415975bb` |
| `adapt` | `sdk/cliproxy/auth/conductor.go` | `sdk/cliproxy/auth/conductor.go` | auth-session-fixes, executor-runtime-fixes, home-parity, logging-metadata, plugin-platform, thinking-consolidation | `f32291436ad9`, `a97b1ae6226f`, `fe4ae4989c4d`, `f6c32ec3ffe7`, `3727d87cefef`, `3ecd4afe808a` |
| `adapt` | `sdk/cliproxy/auth/home_session_alias.go` | `sdk/cliproxy/auth/home_session_alias.go` | auth-session-fixes | `c702c9ac21b4`, `a97b1ae6226f` |
| `adapt` | `sdk/cliproxy/auth/home_session_alias_test.go` | `sdk/cliproxy/auth/home_session_alias_test.go` | auth-session-fixes | `c702c9ac21b4`, `a97b1ae6226f` |
| `adapt` | `sdk/cliproxy/auth/oauth_model_alias.go` | `sdk/cliproxy/auth/oauth_model_alias.go` | thinking-consolidation | `f32291436ad9` |
| `adapt` | `sdk/cliproxy/auth/oauth_model_alias_test.go` | `sdk/cliproxy/auth/oauth_model_alias_test.go` | thinking-consolidation | `f32291436ad9` |
| `adapt` | `sdk/cliproxy/auth/openai_compat_pool_test.go` | `sdk/cliproxy/auth/openai_compat_pool_test.go` | thinking-consolidation | `f32291436ad9` |
| `adapt` | `sdk/cliproxy/auth/selector.go` | `sdk/cliproxy/auth/selector.go` | auth-session-fixes, executor-runtime-fixes, weighted-scheduler | `5dcca50fd9d9`, `a97b1ae6226f`, `f6c32ec3ffe7` |
| `adapt` | `sdk/cliproxy/auth/selector_test.go` | `sdk/cliproxy/auth/selector_test.go` | auth-session-fixes, executor-runtime-fixes, weighted-scheduler | `5dcca50fd9d9`, `a97b1ae6226f`, `f6c32ec3ffe7` |
| `adapt` | `sdk/cliproxy/config_model_display_name_test.go` | `sdk/cliproxy/config_model_display_name_test.go` | codex-model-resolution | `928478e4b915`, `2c8e5ba46334` |
| `adapt` | `sdk/cliproxy/executor/types.go` | `sdk/cliproxy/executor/types.go` | executor-runtime-fixes, home-parity, plugin-platform | `30efd7c4fd16`, `f6c32ec3ffe7`, `3ecd4afe808a` |
| `adapt` | `sdk/cliproxy/service.go` | `sdk/cliproxy/service.go` | executor-runtime-fixes, home-parity, plugin-platform, postgres-cooldown | `f329b9d10c21`, `fe4ae4989c4d`, `27fc3169bb4e`, `3ecd4afe808a` |
| `adapt` | `sdk/cliproxy/service_codex_executor_binding_test.go` | `sdk/cliproxy/service_codex_executor_binding_test.go` | executor-runtime-fixes | `27fc3169bb4e` |
| `adapt` | `sdk/translator/registry.go` | `sdk/translator/registry.go` | thinking-summary-visibility | `c4dcd8703ad9`, `5d307c195dd2`, `b3046d29b985` |
| `adapt` | `test/thinking_conversion_test.go` | `test/thinking_conversion_test.go` | thinking-summary-visibility | `0c2ec7da235d`, `87ceaf83bb70`, `b3046d29b985` |
| `already-present` | `internal/api/handlers/management/auth_files_patch_fields_test.go` | `internal/api/handlers/management/auth_files_patch_fields_test.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/api/handlers/management/config_basic.go` | `internal/api/handlers/management/config_basic.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/api/handlers/management/config_lists.go` | `internal/api/handlers/management/config_lists.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/cache/antigravity_reasoning_replay_cache.go` | `internal/cache/antigravity_reasoning_replay_cache.go` | antigravity-replay | `72886356dd9b` |
| `already-present` | `internal/cache/antigravity_reasoning_replay_cache_test.go` | `internal/cache/antigravity_reasoning_replay_cache_test.go` | antigravity-replay, home-parity | `20784c67ff52`, `72886356dd9b` |
| `already-present` | `internal/config/credential_concurrency_fixture_test.go` | `internal/config/credential_concurrency_fixture_test.go` | home-parity | `7692ccdca5be`, `3ecd4afe808a` |
| `already-present` | `internal/config/credential_in_flight.go` | `internal/config/credential_in_flight.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/config/parse.go` | `internal/config/parse.go` | home-parity, weighted-scheduler | `5dcca50fd9d9`, `3ecd4afe808a` |
| `already-present` | `internal/home/client_test.go` | `internal/home/client_test.go` | home-parity | `20784c67ff52`, `7f2f4a5bb891`, `bb43b7a30fc8`, `f943926fca79`, `3b4f4cf1f896`, `8eed5f1bf8e0`, `7cb71ed6b25e`, `3ecd4afe808a` |
| `already-present` | `internal/home/concurrency_release.go` | `internal/home/concurrency_release.go` | home-parity | `8eed5f1bf8e0`, `3ecd4afe808a` |
| `already-present` | `internal/home/concurrency_release_test.go` | `internal/home/concurrency_release_test.go` | home-parity | `8eed5f1bf8e0`, `3ecd4afe808a` |
| `already-present` | `internal/home/global.go` | `internal/home/global.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/home/testdata/concurrency_dispatch_accounted.json` | `internal/home/testdata/concurrency_dispatch_accounted.json` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/home/testdata/concurrency_dispatch_busy.json` | `internal/home/testdata/concurrency_dispatch_busy.json` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/home/testdata/concurrency_release.json` | `internal/home/testdata/concurrency_release.json` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/registry/models/models.json` | `internal/registry/models/models.json` | model-catalog | `7d00936acc2e`, `a432d763058a`, `61a6f08d18f1`, `3073dab0b693`, `ace2e843cbb4` |
| `already-present` | `internal/runtime/executor/antigravity_executor_credits_test.go` | `internal/runtime/executor/antigravity_executor_credits_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/runtime/executor/antigravity_executor_signature_test.go` | `internal/runtime/executor/antigravity_executor_signature_test.go` | antigravity-replay | `72886356dd9b` |
| `already-present` | `internal/runtime/executor/codex_websockets_executor_store_test.go` | `internal/runtime/executor/codex_websockets_executor_store_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/runtime/executor/helps/usage_helpers_test.go` | `internal/runtime/executor/helps/usage_helpers_test.go` | usage-normalization | `42f36b94e080`, `fe8a616aa303`, `416a08017447` |
| `already-present` | `internal/runtime/executor/websocket_lifecycle_bind_test.go` | `internal/runtime/executor/websocket_lifecycle_bind_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `internal/store/objectstore.go` | `internal/store/objectstore.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/store/postgres_cooldown_store.go` | `internal/store/postgres_cooldown_store.go` | postgres-cooldown | `f329b9d10c21` |
| `already-present` | `internal/store/postgres_cooldown_store_test.go` | `internal/store/postgres_cooldown_store_test.go` | postgres-cooldown | `f329b9d10c21` |
| `already-present` | `internal/store/postgresstore.go` | `internal/store/postgresstore.go` | postgres-cooldown, weighted-scheduler | `5dcca50fd9d9`, `f329b9d10c21` |
| `already-present` | `internal/translator/antigravity/claude/antigravity_claude_response.go` | `internal/translator/antigravity/claude/antigravity_claude_response.go` | antigravity-replay | `72886356dd9b` |
| `already-present` | `internal/translator/antigravity/claude/antigravity_claude_response_test.go` | `internal/translator/antigravity/claude/antigravity_claude_response_test.go` | antigravity-replay | `72886356dd9b` |
| `already-present` | `internal/translator/antigravity/claude/signature_validation.go` | `internal/translator/antigravity/claude/signature_validation.go` | antigravity-replay | `72886356dd9b` |
| `already-present` | `internal/translator/antigravity/gemini/antigravity_gemini_request.go` | `internal/translator/antigravity/gemini/antigravity_gemini_request.go` | antigravity-replay | `62a2c0d374f2` |
| `already-present` | `internal/translator/antigravity/gemini/antigravity_gemini_request_test.go` | `internal/translator/antigravity/gemini/antigravity_gemini_request_test.go` | antigravity-replay | `62a2c0d374f2`, `72886356dd9b` |
| `already-present` | `internal/translator/claude/openai/chat-completions/claude_openai_response.go` | `internal/translator/claude/openai/chat-completions/claude_openai_response.go` | translator-usage-metadata | `74d38e0999bf` |
| `already-present` | `internal/translator/claude/openai/responses/claude_openai-responses_request.go` | `internal/translator/claude/openai/responses/claude_openai-responses_request.go` | claude-schema-normalization | `59aa35a4344b` |
| `already-present` | `internal/translator/claude/openai/responses/claude_openai-responses_request_test.go` | `internal/translator/claude/openai/responses/claude_openai-responses_request_test.go` | claude-schema-normalization | `59aa35a4344b` |
| `already-present` | `internal/translator/gemini/claude/gemini_claude_request_test.go` | `internal/translator/gemini/claude/gemini_claude_request_test.go` | translator-naming | `cade44b9cdee` |
| `already-present` | `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go` | `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go` | translator-structured-output | `20f83cae910b` |
| `already-present` | `internal/util/claude_schema_test.go` | `internal/util/claude_schema_test.go` | claude-schema-normalization | `59aa35a4344b` |
| `already-present` | `internal/util/gemini_schema.go` | `internal/util/gemini_schema.go` | antigravity-replay, claude-schema-normalization | `2b63d6bcda13`, `5dedb303f1a4`, `a4d18cb04acc` |
| `already-present` | `internal/util/gemini_schema_test.go` | `internal/util/gemini_schema_test.go` | claude-schema-normalization | `2b63d6bcda13`, `5dedb303f1a4` |
| `already-present` | `internal/watcher/clients.go` | `internal/watcher/clients.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/watcher/synthesizer/file.go` | `internal/watcher/synthesizer/file.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/watcher/synthesizer/file_test.go` | `internal/watcher/synthesizer/file_test.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `internal/watcher/watcher_test.go` | `internal/watcher/watcher_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/api/handlers/handlers_stream_bootstrap_test.go` | `sdk/api/handlers/handlers_stream_bootstrap_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/antigravity_credits_test.go` | `sdk/cliproxy/auth/antigravity_credits_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/conductor_overrides_test.go` | `sdk/cliproxy/auth/conductor_overrides_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/cooldown_backoff_test.go` | `sdk/cliproxy/auth/cooldown_backoff_test.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `sdk/cliproxy/auth/cooldown_state.go` | `sdk/cliproxy/auth/cooldown_state.go` | postgres-cooldown | `f329b9d10c21` |
| `already-present` | `sdk/cliproxy/auth/cooldown_state_test.go` | `sdk/cliproxy/auth/cooldown_state_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/home_concurrency.go` | `sdk/cliproxy/auth/home_concurrency.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/home_concurrency_test.go` | `sdk/cliproxy/auth/home_concurrency_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/home_in_flight_publisher.go` | `sdk/cliproxy/auth/home_in_flight_publisher.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/home_in_flight_publisher_test.go` | `sdk/cliproxy/auth/home_in_flight_publisher_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/home_selection_attempt_test.go` | `sdk/cliproxy/auth/home_selection_attempt_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/home_websocket_reuse_test.go` | `sdk/cliproxy/auth/home_websocket_reuse_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/request_auth_prepare_test.go` | `sdk/cliproxy/auth/request_auth_prepare_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/auth/scheduler.go` | `sdk/cliproxy/auth/scheduler.go` | weighted-scheduler | `5dcca50fd9d9` |
| `already-present` | `sdk/cliproxy/auth/scheduler_test.go` | `sdk/cliproxy/auth/scheduler_test.go` | home-parity, weighted-scheduler | `5dcca50fd9d9`, `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/builder.go` | `sdk/cliproxy/builder.go` | home-parity, postgres-cooldown, weighted-scheduler | `5dcca50fd9d9`, `f329b9d10c21`, `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/executionregistry/registry.go` | `sdk/cliproxy/executionregistry/registry.go` | home-parity | `8eed5f1bf8e0`, `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/executionregistry/registry_test.go` | `sdk/cliproxy/executionregistry/registry_test.go` | home-parity | `8eed5f1bf8e0`, `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/pprof_server.go` | `sdk/cliproxy/pprof_server.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/cliproxy/service_stale_state_test.go` | `sdk/cliproxy/service_stale_state_test.go` | home-parity | `3ecd4afe808a` |
| `already-present` | `sdk/config/config.go` | `sdk/config/config.go` | claude-cloaking-toggle | `69144785622a` |
| `already-present` | `sdk/proxyutil/proxy.go` | `sdk/proxyutil/proxy.go` | codex-live | `49be36aef6a6` |
| `defer` | `internal/home/client.go` | `internal/home/client.go` | home-401-refresh, home-parity | `a63da8ae76b1`, `4db8e1202942`, `20784c67ff52`, `7f2f4a5bb891`, `bb43b7a30fc8`, `f943926fca79`, `3b4f4cf1f896`, `8eed5f1bf8e0` +2 |
| `defer` | `internal/home/requests.go` | `internal/home/requests.go` | home-401-refresh, home-parity | `a63da8ae76b1`, `4db8e1202942`, `3ecd4afe808a` |
| `defer` | `internal/runtime/executor/helps/usage_helpers.go` | `internal/runtime/executor/helps/usage_helpers.go` | home-401-refresh, home-parity, usage-normalization | `a63da8ae76b1`, `4db8e1202942`, `42f36b94e080`, `fe8a616aa303`, `416a08017447` |
| `defer` | `sdk/cliproxy/auth/home_selection.go` | `sdk/cliproxy/auth/home_selection.go` | home-401-refresh, home-parity | `a63da8ae76b1`, `4db8e1202942`, `3ecd4afe808a` |
| `defer` | `sdk/cliproxy/auth/home_selection_test.go` | `sdk/cliproxy/auth/home_selection_test.go` | home-401-refresh, home-parity | `a63da8ae76b1`, `4db8e1202942`, `3ecd4afe808a` |
| `defer` | `sdk/cliproxy/usage/manager.go` | `sdk/cliproxy/usage/manager.go` | home-401-refresh, home-parity, usage-normalization | `a63da8ae76b1`, `4db8e1202942`, `416a08017447` |
| `reject` | `.gitignore` | `.gitignore` | branding-docs | `3a995bd801e4` |
| `reject` | `README.md` | `README.md` | branding-docs | `a14dfc779f43`, `9a52607f18e2`, `87750f9e2a7a`, `9ffc3baa848a`, `db82d65d1cc3`, `2457e01e8781` |
| `adapt` | `internal/logging/request_logger.go` | `internal/logging/request_logger.go` | plugin-platform | `fe4ae4989c4d` |
| `adapt` | `sdk/api/handlers/claude/code_handlers.go` | `sdk/api/handlers/claude/code_handlers.go` | claude-cloaking-toggle, model-catalog, plugin-platform | `30efd7c4fd16`, `69144785622a`, `0296600be60a` |
| `adapt` | `sdk/api/handlers/handlers_error_response_test.go` | `sdk/api/handlers/handlers_error_response_test.go` | home-parity, plugin-platform | `30efd7c4fd16`, `3ecd4afe808a` |
| `adapt` | `sdk/api/handlers/header_filter.go` | `sdk/api/handlers/header_filter.go` | plugin-platform | `30efd7c4fd16` |
