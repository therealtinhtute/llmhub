---
id: 01KY1T7GGY3DBEW2NAAV620VCK
type: spec
phase: websocket-message-too-big
lane: high-risk
intake_id: 01KY1T9KV4F9NKFP1APTK2M7ZG
created: 2026-07-23
updated: 2026-07-25
---

# SPEC: Implement all post-v7.2.93 updates from CLIProxyAPI

Status: locked
Input Type: new-initiative
Lane: high-risk
Risk Flags: external-systems, public-contract, existing-behavior, multi-domain
Affected Surfaces: api, provider, db, docs, frontend

## Source Mode
files

## Source Inputs
- All changes between CLIProxyAPI v7.2.93 and v7.2.96 (credential concurrency, token handling, model routing, docs, etc.)

## Goal
Improve and implement **all** updates that happened in upstream after v7.2.93 into llmhub, covering both Go backend and React frontend, while preserving existing behavior.

## In Scope
- Credential concurrency (Home/general)
- Token estimation/state handling and perf improvements
- Model routing updates (Codex Alpha Search + new models)
- Translator/executor behavior fixes (empty responses, output indexing, MIME types, WebSocket 1009)
- Docs and management panel UI updates
- Maintain existing provider behavior (no breaking changes)

## Out of Scope
- Unrelated new features
- Full upstream merge beyond listed gaps
- Frontend-only changes without backend parity

## Key Decisions
- Full post-v7.2.93 coverage (not targeted slice)
- Phased implementation (auth/credential first, then executor/translator, then frontend)
- YAGNI: only implement what is in the upstream diff list

## Deferred Ideas
- End-to-end integration tests for new token logic
- Backend migration script for credential changes

## Validation Expectations
- `go test ./...` and `make build-web` + `make build` pass
- Parity report updated with new evidence
- Frontend UI matches new model display/name changes
- No regression in existing providers

## Next Step
Run `/to-plan` after approval.
