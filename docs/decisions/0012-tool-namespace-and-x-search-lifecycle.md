# 0012 Tool Namespace and x_search Lifecycle

Date: 2026-07-23

## Status

Accepted

## Context

OpenAI Responses tools may carry a type, namespace, and name that cannot be
reconstructed safely from a flattened outbound name alone. Codex requires an
effective name such as `<namespace>__<name>`, while clients must receive the
original declaration identity. Existing `mcp__` names must remain unchanged.
Two declarations can also flatten to the same effective name, which would make
response restoration ambiguous.

Streaming Chat Completions can fragment `function.name` across chunks and can
interleave name and argument fragments. Custom-tool inputs may be raw text or a
compatibility wrapper shaped as `{"input":"..."}`. Delimiter-based inference or
eager wrapper decoding can therefore change valid client content.

xAI additionally emits provider-internal `x_search` lifecycle items. Returning
those items to clients can make clients attempt to execute a provider-owned tool,
but a client-declared namespaced tool with the same short name must remain
visible.

## Decision

Build a request-scoped declaration table before any Codex HTTP send or WebSocket
dial. Record each tool's original type, namespace, and name together with its
effective outbound name. Preserve existing `mcp__` names byte-for-byte. Reject
effective-name collisions with HTTP 400, type `invalid_request_error`, and code
`tool_name_collision` before network activity.

Restore response items only through the request declaration table. Do not infer a
namespace by splitting an undeclared name on `__`. Translate declared custom-tool
lifecycles from function events to custom-tool events, use `ctc_*` identifiers,
and restore the exact declared namespace, name, and type in streaming and
non-streaming responses.

Accumulate fragmented Chat `function.name` values. While the accumulated name is
a prefix of a declared effective name, defer classification and buffer argument
fragments. Once identity is unambiguous, replay the buffered arguments through
the matching function or custom-tool lifecycle. At terminal time, flush an
unresolved name as an ordinary function call.

Unwrap custom input only when the complete value is a valid JSON object with
exactly one member named `input` whose value is a string. Preserve all other
values as raw text, including brace-prefixed text, incomplete wrapper-like input,
duplicate or extra keys, `null`, arrays, objects, booleans, and numbers.

For xAI, suppress namespace-free provider-internal `x_search` lifecycle items,
including internal `xs_call` identities, and compact the remaining output
indexes. Exact request declarations outrank prefix heuristics, so an explicitly
declared namespaced `x_search` remains visible.

## Alternatives Considered

1. Infer namespaces by splitting every name containing `__`. Rejected because
   ordinary tool names may contain that delimiter and undeclared response names
   have no trustworthy original identity.
2. Let the provider resolve flattened-name collisions. Rejected because response
   restoration would remain ambiguous and network activity would occur before a
   deterministic client error.
3. Classify each Chat name fragment independently. Rejected because partial names
   can be emitted through the wrong lifecycle and arguments can be lost or
   misrouted.
4. Decode any JSON object containing `input`. Rejected because it destructively
   changes legitimate raw JSON and mishandles non-string or additional members.
5. Drop every `x_search`-named item. Rejected because clients may legitimately
   declare a namespaced tool with that short name.

## Consequences

Positive:

- Tool identity round-trips exactly across HTTP, WebSocket, streaming, and
  non-streaming paths.
- Ambiguous declarations fail deterministically before provider activity.
- Fragmented names and inputs retain one consistent lifecycle.
- Provider-internal xAI search traces cannot be re-executed by clients.
- Existing `mcp__` names and explicitly declared namespaced tools remain stable.

Tradeoffs:

- Each request retains a small declaration and streaming-state table until it
  completes.
- New tool event families must be added explicitly rather than inferred from
  delimiters.
- Compatibility-wrapper recognition intentionally waits for sufficient input or
  terminal flush before emitting some custom input.
