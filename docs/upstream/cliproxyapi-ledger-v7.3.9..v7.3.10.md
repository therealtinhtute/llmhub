# Upstream ledger — cliproxyapi v7.3.9..v7.3.10

- generated: 2026-09-21T14:24:25Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `1ccfcecdb773`
- non-merge commits: 10

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.3.10 | `ddc3f731f45a` | 2026-09-20 | feat(steering): add documentation for Codex response steering and duplex transport | `docs/STEERING.md` | already-present | Behavior shipped earlier; local has `internal/runtime/executor/codex_websockets_duplex.go` + `CodexResponseSteering` at `internal/config/sdk_config.go:65-70`; upstream commit is docs-only |
| v7.3.10 | `e56547f39d57` | 2026-09-20 | fix(auth): avoid unnecessary scheduler rebuilds by checking sync needs from auth map | `internal/registry`, `sdk/cliproxy` | defer | Perf-only: upstream adds `registrationEpoch`/`structuralEpoch`/`syncedVersion` guards (`v7.3.10 e56547f39d57` scheduler.go); local `syncScheduler` rebuilds unconditionally (`sdk/cliproxy/auth/conductor.go:438-443`). Pull in when scheduler rebuilds show up in profiling |
| v7.3.10 | `33ae35d53a23` | 2026-09-20 | docs: add international Kimi Code link alongside domestic subscription link | `README.md`, `README_CN.md`, `README_JA.md` | reject | Branding/docs only; local README is llmhub's own (invariant 5) |
| v7.3.10 | `c52ca7bd4e0b` | 2026-09-20 | docs(claude): clarify patch version floor rationale for native passthrough | `internal/runtime` | reject | Comment-only changes in upstream `claude_client_detection.go` (file absent locally) |
| v7.3.10 | `83a4913aa49c` | 2026-09-20 | fix(claude): accept newer patch releases as native clients (#5820) | `config.example.yaml`, `internal/runtime` | superseded-locally | Local `shouldUpgradeClaudeDeviceProfile` learns ANY newer version (`internal/runtime/executor/helps/claude_device_profile.go:186` `Compare > 0`), already accepting newer patch releases; upstream fix targets `claude_client_detection.go` (absent locally). Policy note: local also accepts newer major/minor — defensible local choice |
| v7.3.10 | `cdfb79ef843f` | 2026-09-20 | feat(translator): support builtin tools in gemini and antigravity interactions | `internal/translator` | reject | Targets `internal/translator/gemini/interactions/` + `antigravity/interactions/` — subsystem with no local equivalent (`internal/translator/common/antigravity_tools.go:11-13`); local builtin-tool support lives in openai-responses paths (`internal/translator/gemini/openai/responses/gemini_openai-responses_request.go`) |
| v7.3.10 | `a5ab69521f7b` | 2026-09-21 | feat: add full support for kimi.ai oauth and runtime execution | `cmd/server`, `internal/api`, `internal/auth` +8 | adapt | kimi.ai domain support absent locally: 0 hits for `kimi-ai|kimi.ai` in local tree; local `internal/auth/kimi/kimi.go:23-40` hardcodes kimi.com hosts; `KimiTokenStorage` lacks Domain/BaseURL (`internal/auth/kimi/token.go:22-31`). Port behavior behind local kimi auth/executor/thinking |
| v7.3.10 | `28100e54b9b5` | 2026-09-21 | feat(auth): propagate context compaction and node kind session metadata | `internal/home`, `internal/logging`, `internal/redisqueue` +1 | adapt | LCP compaction engine + node-kind propagation absent: local `lcp.go` is pre-v7.3.10 API (`sdk/cliproxy/session/lcp.go:734-830` Prepare/MatchFingerprints/BindFingerprints); no NodeKind/IsCompaction anywhere except one string match `lcp.go:201`; local `syncScheduler`-era conductor has no equivalent of upstream `MatchFingerprintsWithContext` |
| v7.3.10 | `563865e77adb` | 2026-09-21 | test: fix timing synchronization and channel races in streaming tests | `internal/runtime`, `sdk/api` | reject | Test-only churn on upstream's own test suite; local tests differ (diverged) |
| v7.3.10 | `40cc6489879a` | 2026-09-21 | fix(translator): preserve reasoning content across tool turns in responses conversion | `internal/translator` | adapt | Fix absent locally: no `latestReasoningContent`/`fallbackToolReasoning`/`isUsableResponsesReasoning` in `internal/translator/openai/openai/responses/openai_openai-responses_request.go` (600 lines, old API). Self-contained single-file port |
