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

Generation requests use Kiro-compatible endpoint fallback for reliability:
Kiro Q, CodeWhisperer, then Amazon Q. Retryable upstream failures such as 429
and 5xx can fall through to the next endpoint, while auth/payment failures do
not fan out. Generation URLs are regionalized from `profile_arn` or `region`
when the account is outside `us-east-1`.

## Interface Contract

Kiro is available as provider key `kiro`. Existing model alias and excluded
model configuration accepts the `kiro` channel.

Kiro upstream also imposes a hard 64-character limit on tool names inside
`toolSpecification.name` and `toolUses[].name`. LLMHub shortens long
MCP/Codex-style names before sending Kiro requests and restores the original
names in client-visible tool calls.

## Data Model

No schema migration is required. Kiro auth state uses the existing auth metadata
file or auth store record shape.

## UI / Platform Impact

Operators can upload existing auth import files through the current auth-file
route. The management auth-file and quota pages show Kiro runtime quota and
cooldown state from llmhub auth records.

Kiro provider quota is fetched per auth account through `getUsageLimits` and
stored as normalized runtime metadata under `kiro_quota`. The management
auth-file list exposes this normalized snapshot, and the quota UI renders
provider quota, trial, overage, and runtime cooldown state together.

The quota fetch path follows the current 9router endpoint fallback sequence:
CodeWhisperer `GET /getUsageLimits`, CodeWhisperer JSON-RPC-style `POST` with
`AmazonCodeWhispererService.GetUsageLimits`, then Q `GET /getUsageLimits`.
The normalized state keeps scalar `current/limit/remaining/percent` fields for
routing compatibility and a `quotas` collection for all provider quota rows.

When persisted Kiro provider quota is exhausted, routing skips that auth until
the provider reset time is reached. This is a transient runtime block, not a
permanent disable. The existing 429/402/403 cooldown path remains the fallback
when provider quota is missing or stale.

Kiro stream `metricsEvent`, `contextUsageEvent`, and `meteringEvent` are parsed
for runtime usage accounting. Per-auth request/token stats are stored in auth
metadata as `kiro_usage_stats` and shown below provider quota in the quota UI.
When metrics are absent, context-usage estimates use the live model context
length when available, otherwise 1M tokens for Claude 4.6+ and 200K for older
Kiro models.

## Observability

Executor requests and responses use the existing upstream request/response
recording helpers. Token refresh failures surface as provider errors. Kiro
quota refresh errors are shown as quota metadata/status text and do not mark the
auth permanently disabled.

## Alternatives Considered

1. Store 9router exports verbatim. Rejected because executors need a stable
   llmhub metadata shape and disabled state must be preserved.
