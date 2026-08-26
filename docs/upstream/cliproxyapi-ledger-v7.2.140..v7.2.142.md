# Upstream ledger — cliproxyapi v7.2.140..v7.2.142

- generated: 2026-08-26T05:06:23Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `555ad5a5291d`
- non-merge commits: 15

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.2.141 | `9d0a60bfc361` | 2026-08-23 | fix(gemini,antigravity): preserve function/tool results as raw strings | `internal/translator` |  |  |
| v7.2.141 | `6f4b6dc5f53d` | 2026-08-23 | fix(xai): keep forced image_generation from other tools | `internal/runtime` |  |  |
| v7.2.141 | `fba1ff24ac60` | 2026-08-23 | fix(xai): preserve auto when rewriting image-only allowed_tools | `internal/runtime` |  |  |
| v7.2.141 | `d2742c5f37d8` | 2026-08-23 | fix(xai): map image_generation tool_choice to required | `internal/runtime` |  |  |
| v7.2.141 | `85749db9bcd3` | 2026-08-24 | docs(readme): add APIMart sponsor to Japanese README | `README.md`, `README_CN.md`, `README_JA.md` +2 |  |  |
| v7.2.141 | `5fead38f0d5a` | 2026-08-24 | docs: remove VisionCoder sponsor section from all READMEs | `README.md`, `README_CN.md`, `README_JA.md` +1 |  |  |
| v7.2.141 | `cef351a4644b` | 2026-08-24 | docs(readme): add APIMart sponsor | `README.md`, `README_CN.md`, `assets/Apimart-en.png` +1 |  |  |
| v7.2.142 | `ca601db05d85` | 2026-08-24 | feat: observe upstream provider quota signals (#5211) | `internal/api`, `internal/logging`, `internal/runtime` +1 |  |  |
| v7.2.142 | `1f53b2eb03b9` | 2026-08-25 | fix(auth): keep credential rotation fair when candidates are filtered | `sdk/cliproxy` |  |  |
| v7.2.142 | `998dcfeba2f1` | 2026-08-25 | fix(antigravity): safely synthesize terminal finish reasons (#5230) | `internal/runtime`, `internal/translator` |  |  |
| v7.2.142 | `f2b1996b3f95` | 2026-08-25 | fix(gemini,antigravity): align parallel tool results with preceding tool calls | `internal/runtime`, `internal/translator` |  |  |
| v7.2.142 | `80de9015502e` | 2026-08-25 | fix: preserve multi-reference video durations | `sdk/api` |  |  |
| v7.2.142 | `adf052984f8b` | 2026-08-25 | fix(antigravity): remove cross-endpoint fallback (#5209) (#5228) | `internal/runtime` |  |  |
| v7.2.142 | `e1bf89395687` | 2026-08-25 | feat(github): resolve GitHub token for release checks and asset updates | `internal/api`, `internal/managementasset`, `internal/util` |  |  |
| v7.2.142 | `ba510f85a21c` | 2026-08-25 | fix(pluginhost): add plugin quiesce handling with safe rollback during hot reload | `internal/pluginhost`, `sdk/pluginabi` |  |  |
