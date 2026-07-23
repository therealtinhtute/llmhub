# 0011 Translator Structured Output and Index Fidelity

Date: 2026-07-23

## Status

Accepted

## Context

OpenAI file content can carry raw base64 or a `data:<mime>;base64,<payload>`
value, but Gemini, Gemini CLI, and Codex translators handled those forms
inconsistently. Some invalid structured file items could also lose `file_data`
when another file reference was present.

OpenAI Responses function-call outputs were flattened through string conversion
on the Claude path. Preserving every array blindly is also unsafe because Claude
accepts only specific tool-result content blocks with provider-valid nested
sources. Codex-to-Claude streaming additionally used one sequential block index
for events that already supplied stable `output_index` values, and mixed event
ordering could overwrite reasoning identity or overlap text, thinking, and tool
blocks.

## Decision

Use `NormalizeOpenAIFileData` as the shared parser for the Gemini, Gemini CLI,
and Codex chat-completions paths. Accept non-empty raw payloads only when a MIME
type is supplied or can be inferred from the filename. For data URLs, require a
non-empty media type, payload, and `base64` marker, and use the media type encoded
in the URL. If a structured file item cannot be normalized, preserve the entire
item through the translator's deterministic fallback rather than silently using
another file field and dropping invalid `file_data`.

For OpenAI Responses function-call outputs translated to Claude, keep strings as
strings. Keep an array structured only when every member converts to a valid
Claude tool-result content block; otherwise serialize the entire original value
as compact JSON text. Validate Claude image media types and base64 syntax,
document source variants, search-result required fields and text-only content,
and tool references. A document content source accepts either a string or an
array of text/image blocks. Preserve valid empty arrays. Do not partially emit a
mixed or invalid array.

For Codex-to-Claude streaming, use `output_index` whenever the field exists,
including zero. Use sequential `BlockIndex` only when the provider omits the
field. Retain text, reasoning, and function identity across later events that
omit an index; close an active content block before a different block starts;
and retain retired reasoning/function identities so delayed completion events
cannot reopen or stop the wrong lifecycle.

Do not extend this phase to Antigravity or Google Interactions.

## Alternatives Considered

1. Forward `file_data` unchanged in each translator. Rejected because data URLs
   would be sent as payload bytes and MIME fallback would remain inconsistent.
2. Prefer `file_id` or `file_url` when `file_data` is invalid. Rejected because
   it silently discards part of the structured tool output.
3. Preserve every function-output array as Claude structured content. Rejected
   because unknown, mixed, or schema-invalid blocks can make the provider reject
   the next request.
4. Flatten every non-string output to ordinary text. Rejected because it loses
   valid image, document, search-result, and tool-reference semantics.
5. Always allocate sequential Claude block indices. Rejected because it breaks
   provider item identity and can misroute later deltas or stops.
6. Track one global reasoning or function index. Rejected because interleaved or
   delayed events can overwrite signatures, overlap blocks, or close a newer
   item with an older completion.

## Consequences

Positive:

- File data and MIME extraction are consistent across the selected translators.
- Invalid structured file items retain deterministic fallback content.
- Valid Claude tool-result media and search blocks remain structured, while
  invalid or mixed values cannot leak as partial arrays.
- Provider-supplied output indices remain stable through text, thinking, and
  tool lifecycles.
- Delayed or interleaved events cannot overlap content blocks or reopen retired
  reasoning items.

Tradeoffs:

- Claude-native blocks require local schema validation that must evolve if the
  provider adds new tool-result content variants.
- Per-stream identity maps retain small amounts of state until the response
  completes.
- Unsupported structured values remain visible to Claude as compact JSON text
  rather than provider-native content blocks.
