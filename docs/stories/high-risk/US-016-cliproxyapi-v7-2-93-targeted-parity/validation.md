# Validation

## Proof Strategy

Phases 1–3 used deterministic focused tests, exact changed-path evidence,
reviewed patches, and scratch forward/reverse proof. At the user's direction,
Phases 4–5 switched to one-pass implementation without additional immutable
slice freezes or independent closure review. The final combined gate therefore
provides the completion proof for those phases: full Go tests, vet, Go build,
and whitespace validation.

## Phase 1 — WebSocket Message Too Big

Status: approved on 2026-07-23.

Evidence revision:
`.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P1-CLOSE/r01/`

- Baseline: `2960b690bf89232b8a23c5b8823fbe0ca831347f`.
- Patch SHA-256: `081c6029e14044651368ed6cc1a212148d791d0694fd7836228138c025d14427`.
- Forward scratch application matches the exact current six-file fingerprint.
- Reverse scratch application matches the exact baseline six-file fingerprint.
- Post-test fingerprint equals the pre-test fingerprint.
- Independent high-risk review verdict: `APPROVED`, zero verified Critical or
  Major findings.

Commands and observed results:

```text
go test ./internal/runtime/executor -run 'Test.*(MessageTooBig|1009)' -count=1
ok github.com/therealtinhtute/llmhub/internal/runtime/executor

go test ./sdk/cliproxy/auth -run 'Test.*(RequestScoped|Fallback)' -count=1
ok github.com/therealtinhtute/llmhub/sdk/cliproxy/auth

go test ./sdk/api/handlers/openai -run 'Test.*WebSocket.*(1009|MessageTooBig)' -count=1
ok github.com/therealtinhtute/llmhub/sdk/api/handlers/openai

go test ./internal/runtime/executor ./sdk/cliproxy/auth ./sdk/api/handlers/openai -count=1
ok github.com/therealtinhtute/llmhub/internal/runtime/executor
ok github.com/therealtinhtute/llmhub/sdk/cliproxy/auth
ok github.com/therealtinhtute/llmhub/sdk/api/handlers/openai
```

Acceptance covered:

- Close 1009 maps to structured request-scoped 413 `message_too_big`.
- Request-scoped classification prevents credential mutation, refresh, and
  fallback in immediate and streaming paths.
- Non-1009 and unobserved writer errors retain ordinary retry behavior.
- Handler rollback preserves transcript, output history, replay state, and tool
  repair/call caches.

## Phase 2 — Auth and CountTokens Reliability

Status: approved on 2026-07-23.

Reviewed slice evidence:

- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P2-S1-cooldown-jitter/r01/`
  - patch SHA-256: `7e17ef3ab124b1fa713285fb984ac45722926121e4a2489fd4572beb068af875`;
  - independent verdict: `APPROVED`.
- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P2-S2-error-classification/r04/`
  - patch SHA-256: `0f0b7c79e4627bc956a333ce64604dc4ef6a8357324be7915655cc660e0325ec`;
  - immutable result tree: `98d750898d239533918d092276faf9520a64adb1`;
  - independent verdict: `APPROVED`.

Rejected and superseded P2-S2 revisions remain preserved:

- `r01`: unbounded `invalid_grant` matching and missed wrapped nested
  `model_not_found`;
- `r02`: frozen patch omitted the final `strconv` import and did not match its
  tested result tree;
- `r03`: frozen patch retained structured-JSON whole-message fallback despite an
  inconsistent approval artifact; it was reversed from main before applying
  reviewed `r04`.

Commands and observed results:

```text
go test ./sdk/cliproxy/auth ./internal/api/modules/amp -count=1
ok github.com/therealtinhtute/llmhub/sdk/cliproxy/auth
ok github.com/therealtinhtute/llmhub/internal/api/modules/amp

go test ./...
all packages passed

go vet ./...
pass

go build ./...
pass

git diff --check
pass
```

Acceptance covered:

- repeated and concurrent quota failures do not advance backoff during an active
  recovery window;
- clamp-before-jitter never exceeds the configured maximum and preserves
  `disableCooling` behavior;
- only HTTP 400/401 exact or bounded `invalid_grant` failures enter the 30-minute
  suspension and credential-fallback path;
- longer identifiers and unrelated HTTP 400 errors retain request-invalid
  behavior;
- generic CountTokens endpoint 404 failures retain counters, persistence, and
  hooks without changing registry or scheduler availability;
- explicit top-level, nested, wrapped, and joined `model_not_found` failures
  remain availability-changing;
- Amp inherits the shared conductor policy without production route changes.

Decision record:
`docs/decisions/0010-auth-cooldown-and-error-classification.md`.

## Phase 3 — Translator Content Fidelity

Status: approved on 2026-07-23.

Reviewed slice evidence:

- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P3-S1-file-data-normalization/r03/`
  - patch SHA-256: `1e0dae5ab9290d2a16ce81727fb0040fb5a6f6eca591d02f77e9a2761dbaca7d`;
  - immutable result tree: `8848e25e9a5cc7e5c25dee0f80585d5bdd785384`;
  - independent verdict: `APPROVED`.
- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P3-S2-claude-tool-result-content/r03/`
  - patch SHA-256: `c7c0dff343dc6af9e00ec9858258286e3c19612b25c31bd92ddfabcd0b27c2f4`;
  - immutable result tree: `7f22b738faa2d875f37234644b51e4ceb2404f09`;
  - independent verdict: `APPROVED`.
- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P3-S3-codex-claude-output-indices/r04/`
  - patch SHA-256: `9e1e4ccd0ef3a03b83b1726448f11cb53819eb56e874765eef46ec14faec62cc`;
  - immutable result tree: `5039052c3c4c4d67808b2f9a6754f00234824231`;
  - independent verdict: `APPROVED`.

Rejected revisions remain preserved:

- P3-S1 `r02` silently discarded invalid `file_data` when another file reference
  was present;
- P3-S2 `r01` accepted schema-invalid media and rejected valid Claude variants,
  while `r02` rejected valid string-backed document content;
- P3-S3 `r01` changed text identity and overlapped signature-only reasoning,
  `r02` still overwrote reasoning state and overlapped function lifecycles, and
  `r03` reopened retired reasoning items on delayed completion.

Commands and observed results:

```text
go test ./internal/translator/common ./internal/translator/claude/openai/responses ./internal/translator/gemini/openai/chat-completions ./internal/translator/gemini-cli/openai/chat-completions ./internal/translator/codex/openai/chat-completions ./internal/translator/codex/claude -count=1
all six packages passed

go test ./...
all packages passed

go vet ./...
pass

go build ./...
pass

git diff --check
pass
```

Acceptance covered:

- raw base64 and data-URL file content share one MIME-normalization policy across
  Gemini, Gemini CLI, and Codex;
- invalid `file_data` preserves the entire structured item through deterministic
  fallback, including when `file_id` or `file_url` is also present;
- Claude function-call outputs preserve valid text, image, document,
  search-result, and tool-reference arrays only when every member is valid;
- arbitrary objects, scalars, and invalid or mixed arrays become compact JSON
  text, while valid empty arrays and document string content remain structured;
- Codex provider `output_index`, including zero, remains stable across later
  events that omit it, with sequential indexing used only as fallback;
- reasoning, text, and function blocks do not overlap, and delayed stale events
  cannot overwrite signatures, misindex stops, or reopen retired items.

Decision record:
`docs/decisions/0011-translator-structured-output-and-index-fidelity.md`.

## Phase 4 — Tool Protocol Parity

Status: implemented and covered by the final combined gate on 2026-07-23.

Evidence retained from the reviewed xAI slice:

- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r03/`
  - patch SHA-256: `869464e623c79acc03a143c3a1bbcca460c5fe127bc9a1654bbe61908861ba93`.

The final Codex declaration implementation was completed as P4-S1 `r06` after
rejected revisions exposed WebSocket bypass, stale event families, unsafe input
wrapper decoding, whitespace and `null` handling, and fragmented-name
classification defects. Per the accelerated workflow, no new immutable `r06`
evidence directory or independent closure review was created.

Acceptance covered:

- request-scoped declarations preserve original type, namespace, and name while
  sending effective names to Codex and xAI;
- existing `mcp__` names remain byte-stable;
- effective-name collisions return HTTP 400 `invalid_request_error` with code
  `tool_name_collision` before HTTP send or WebSocket dial;
- custom calls use `custom_tool_call`, `ctc_*`, and custom input delta/done event
  families in streaming and non-streaming responses;
- fragmented Chat function names defer classification, buffer arguments, and
  flush unresolved calls as ordinary functions at terminal time;
- custom input is unwrapped only for a complete one-string-member `input` object;
- provider-internal namespace-free xAI search lifecycles are removed and later
  output indexes are compacted, while exact declared namespaced identities remain
  visible.

Decision record:
`docs/decisions/0012-tool-namespace-and-x-search-lifecycle.md`.

## Phase 5 — Model and Display Compatibility

Status: implemented and covered by the final combined gate on 2026-07-23.

Acceptance covered:

- Grok 4.5 metadata is present with the existing xAI capability schema;
- selected Gemini 3 and 3.1 production IDs coexist with their preview IDs;
- optional configured display names survive YAML decoding, sanitization,
  registration, prefix clones, Codex built-in replacement, and OAuth alias forks;
- OpenAI, Codex client, Claude, Gemini, and Gemini CLI catalogs expose the
  protocol-appropriate display-name field without changing model IDs, upstream
  names, routing keys, provider selection, or auth selection;
- GPT-5.6, Kimi K3, Google Interactions, plugin systems, xAI API-key management,
  encrypted replay, and Kiro formatting remain excluded.

Decision record:
`docs/decisions/0013-model-display-name-presentation-contract.md`.

## Final Combined Gate

Status: passed on 2026-07-23.

The first full-suite run found three xAI declaration/filter regressions: an
already-qualified namespace child was qualified twice, namespaced declaration
identity was unavailable to response restoration, and an explicitly namespaced
`xs_call` lifecycle was filtered as provider-internal. Root cause was that xAI
built its declaration table after the shared OpenAI Responses translation had
already flattened namespaces, while the shared qualifier was not idempotent.
The fix captures original OpenAI Responses declarations, merges unique translated
fallback declarations, performs effective-name tool-choice matching, and makes
shared qualification idempotent.

Commands and observed results:

```text
go test ./internal/runtime/executor -run 'TestXAIExecutor(AdditionalToolsNamespaceCustomToolDeclarationRoundTrip|XSearchLifecycleUsesExactDeclarationIdentity|SameNameXSearchCallIdentityFiltering)$' -count=1
ok github.com/therealtinhtute/llmhub/internal/runtime/executor

go test ./...
all packages passed

go vet ./...
pass

go build ./...
pass

git diff --check
pass
```

No web build or live-provider smoke test was run. No commit, push, PR, merge,
release, or publication was performed.
