# Validation

## Proof Strategy

Use focused executor regression tests first, then build and run the real binary
against a local mocked Kiro upstream and hit the public API repeatedly with
multiple prompt and history shapes.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | schema normalization keeps `required: []`; adjacent Kiro user history turns are merged; tools are synthesized from historical tool calls when request `tools` are absent |
| Integration | `go test` passes for the Kiro executor, cliproxy routing, and registry packages |
| Binary smoke | built `llmhub` serves local API requests that previously mapped to malformed Kiro tool-use payloads |
| Platform | local mock server confirms Kiro upstream accepts repeated Claude-compatible and OpenAI-compatible request bodies without `Improperly formed request` |

## Commands

```text
env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/runtime/executor
env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/runtime/executor ./sdk/cliproxy ./internal/registry
env GOCACHE=/private/tmp/llmhub-gocache go test ./...
make build
env GOCACHE=/private/tmp/llmhub-gocache go run /private/tmp/llmhub-kiro-mock/main.go
./llmhub init-db-from-env -env-file /private/tmp/llmhub-kiro.env
set -a; . /private/tmp/llmhub-kiro.env; set +a; ./llmhub
bash /private/tmp/llmhub-verify-many.sh
```

## Acceptance Evidence

- `env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/runtime/executor -run 'TestBuildKiroPayloadFromOpenAI_(StripsHistoricalStructuredToolTurns|FlattensOrphanCurrentToolResults|MergesAdjacentUserHistoryTurns|SynthesizesToolsFromHistory)|TestBuildKiroRequest_ClaudeSourceSynthesizesToolsFromHistory'` passed on 2026-06-07.
- `env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/runtime/executor` passed on 2026-06-07 after hardening the Kiro history sanitizer for completed and orphaned tool-result turns.
- `env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/runtime/executor` passed on 2026-06-04.
- `env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/runtime/executor ./sdk/cliproxy ./internal/registry` passed on 2026-06-04.
- `env GOCACHE=/private/tmp/llmhub-gocache go test ./...` passed on 2026-06-04.
- `env GOCACHE=/private/tmp/llmhub-gocache make build` passed on 2026-06-04. The final `go build` emitted a non-fatal Go module stat-cache permission warning after producing the binary.
- Built `./llmhub` ran successfully against a local mock Kiro upstream with `LLMHUB_SKIP_DOTENV=1`, temp file-backed config/auth, and live localhost API traffic on 2026-06-04.
- Repeated localhost verification passed on 2026-06-04 with `bash /private/tmp/llmhub-verify-many.sh`: `TOTAL_PASS=24`, `0` failures.
- Prompt and history formats covered in the 24 successful runtime requests:
  - `/v1/messages` with explicit Claude tools and minimal `input_schema.properties`
  - `/v1/messages` with Claude `system` prompt blocks plus tools
  - `/v1/messages` with Claude `thinking` enabled plus tools
  - `/v1/messages` with historical `tool_use` / `tool_result` follow-up turns and omitted `tools`
  - `/v1/messages` with mixed text around `tool_result`
  - `/v1/messages` with multimodal text + image input plus tools
  - `/v1/chat/completions` with OpenAI function tools using minimal `parameters.properties`
  - `/v1/chat/completions` with `developer` role plus tools
  - `/v1/chat/completions` with historical assistant `tool_calls` and omitted `tools`
  - `/v1/chat/completions` with multimodal text + `image_url` plus tools
  - `/v1/chat/completions` with `reasoning_effort` on the `-thinking-agentic` Kiro model
  - `/v1/chat/completions` with a trailing assistant turn and no tools
- The live `llmhub` server log recorded `12` successful `POST /v1/messages` and `12` successful `POST /v1/chat/completions` requests during the batch run.
