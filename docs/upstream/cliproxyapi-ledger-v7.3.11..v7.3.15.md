# Upstream ledger — cliproxyapi v7.3.11..v7.3.15

- generated: 2026-09-23T03:22:50Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `1e16cf32d9f2`
- non-merge commits: 27

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
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
