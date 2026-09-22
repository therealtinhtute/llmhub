# Upstream ledger — cliproxyapi v7.3.10..v7.3.11

- generated: 2026-09-21T16:54:04Z
- upstream: https://github.com/router-for-me/CLIProxyAPI
- local baseline: `1ccfcecdb773`
- non-merge commits: 8
- status: **pinned-follow-up** — range appeared during v7.3.10 parity execution (p7 re-resolve); dispositions intentionally blank, NOT triaged, NOT absorbed. Next `triage upstream` cycle starts here.

Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.
Every non-empty disposition needs a citation on both sides.

| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| v7.3.11 | `ffe6ad3c5fcf` | 2026-09-21 | fix(gemini): enforce array type for schema nodes declaring items | `internal/util` |  |  |
| v7.3.11 | `ac3849e5d981` | 2026-09-21 | feat(pluginapi): propagate response model, service tier, and stream flag to usage plugins | `internal/pluginhost`, `sdk/pluginapi` |  |  |
| v7.3.11 | `dd013f9e2993` | 2026-09-21 | fix(responses): filter upstream private and telemetry events in SSE streams | `sdk/api` |  |  |
| v7.3.11 | `7b6fafce1b32` | 2026-09-21 | fix(responses): support mid-connection prewarm in websocket handler | `sdk/api` |  |  |
| v7.3.11 | `50585bf208a8` | 2026-09-21 | refactor(translator): encapsulate tool-result cache-control hoisting in common | `internal/translator` |  |  |
| v7.3.11 | `bcd13ca91c87` | 2026-09-21 | fix(translator): hoist tool-result content part cache_control to the block (#5432) | `internal/translator` |  |  |
| v7.3.11 | `ed751ea08af8` | 2026-09-21 | fix(claude): gate fallback-credit beta on fallback tokens or explicit fallbacks | `internal/runtime` |  |  |
| v7.3.11 | `fd5cd228a804` | 2026-09-21 | test(executor): allow non-negative TTFT in xAI usage record assertions | `internal/runtime` |  |  |
