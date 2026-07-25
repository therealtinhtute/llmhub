# websocket-message-too-big — PLAN

**Inputs:** SPEC.md, upstream v7.2.96 diffs (credential concurrency + 1009 handling).

**Waves:**
1. Backend auth/credential concurrency (Go files: auth, executor)
2. WebSocket error mapping and tests
3. Verification: go test + make build + parity report

**Verification commands:**
`go test ./internal/runtime/executor/... -run TestWebSocketCloseCode`
`make build-web`
`git diff --check`
Update `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`

**Touched surfaces:** auth, executor, translator, provider
**Avoid surfaces:** frontend UI, new providers, db changes

**Stop condition:** All tests pass and parity evidence updated.
