# websocket-message-too-big — CONTEXT

**Goal:** Propagate upstream WebSocket close code 1009 as structured request-scoped error without credential fallback + start credential concurrency support.

**Blast radius:** Low (affects only WebSocket handling and auth credential logic).
**Expected proof class:** integration + unit tests + `git diff --check` + parity report update.
**Allowed/forbidden surfaces:** Only upstream parity behavior + credential changes; no new providers or UI features.
**Escalation:** If credential changes break existing auth flows, revert to previous parity slice.

**Deferred Ideas:** Full token estimation coverage; frontend UI for new models.
