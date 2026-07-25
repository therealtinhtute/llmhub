# ROADMAP: All post-v7.2.93 updates (full sync)

Entry Phase: `websocket-message-too-big`
Execution Mode: full

## Phases
- **auth-credential-concurrency** — Home + general credential concurrency
- **token-estimation** — Token state handling + improved counting + perf in executor
- **model-routing** — Codex Alpha Search + new model support + error handling
- **websocket-pluginhost** — WebSocket 1009 + race fixes + pluginhost
- **docs-frontend** — AIUsage showcase + management panel UI + changelog + Docker improvements

Dependencies: credential/auth changes first, then executor/translator, then frontend/docs.
