# Design

## Domain Model

Kiro auth records use `type: "kiro"` with OAuth metadata:

- `access_token`
- `refresh_token`
- `expired`
- `email`
- `profile_arn`
- `auth_method`
- `disabled`

9router `isActive: false` maps to `disabled: true`.

## Application Flow

Auth import parses raw input at the upload boundary. If the input is a 9router
connection with no access token, llmhub refreshes with the supplied refresh
token before persisting.

At runtime the Kiro executor refreshes expired or unauthorized credentials,
translates OpenAI chat-completions payloads to Kiro `conversationState`, calls
`GenerateAssistantResponse`, and emits OpenAI-compatible responses.

## Interface Contract

Kiro is available as provider key `kiro`. Existing model alias and excluded
model configuration accepts the `kiro` channel.

## Data Model

No schema migration is required. Kiro auth state uses the existing auth metadata
file or auth store record shape.

## UI / Platform Impact

Operators can upload existing auth import files through the current auth-file
route. The management auth-file and quota pages show Kiro runtime quota and
cooldown state from llmhub auth records. Kiro does not expose a provider quota
endpoint in this slice, so the UI labels provider quota as unavailable instead
of showing invented limits or reset windows.

## Observability

Executor requests and responses use the existing upstream request/response
recording helpers. Token refresh failures surface as provider errors.

## Alternatives Considered

1. Store 9router exports verbatim. Rejected because executors need a stable
   llmhub metadata shape and disabled state must be preserved.
