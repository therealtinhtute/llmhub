package helps

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	ClaudeCodeSessionHeader = "X-Claude-Code-Session-Id"
	ClaudeCodeAgentHeader   = "X-Claude-Code-Agent-Id"
	ClaudeCodeMainAgentID   = "main"
)

var claudeCodeSessionSuffixPattern = regexp.MustCompile(`_session_([a-f0-9-]+)$`)

// ExtractClaudeCodeSessionID resolves a Claude Code session ID, preferring X-Claude-Code-Session-Id over payload metadata.
// Ported from upstream CLIProxyAPI claude_code_session.go (parity: 086ad91bd970).
func ExtractClaudeCodeSessionID(ctx context.Context, payload []byte, headers http.Header) string {
	if sessionID := claudeCodeHeader(ctx, headers, ClaudeCodeSessionHeader); sessionID != "" {
		return sessionID
	}
	return extractClaudeCodeSessionIDFromPayload(payload)
}

// ExtractClaudeCodeAgentID resolves the Claude Code agent ID and uses a stable sentinel for the root agent.
func ExtractClaudeCodeAgentID(ctx context.Context, headers http.Header) string {
	if agentID := claudeCodeHeader(ctx, headers, ClaudeCodeAgentHeader); agentID != "" {
		return agentID
	}
	return ClaudeCodeMainAgentID
}

func claudeCodeHeader(ctx context.Context, headers http.Header, name string) string {
	if value := headerValueCaseInsensitive(headers, name); value != "" {
		return value
	}
	if ctx != nil {
		if ginCtx, ok := ctx.Value("gin").(*gin.Context); ok && ginCtx != nil && ginCtx.Request != nil {
			return headerValueCaseInsensitive(ginCtx.Request.Header, name)
		}
	}
	return ""
}

// HeaderValueCaseInsensitive returns the first non-empty header value matching name case-insensitively.
func HeaderValueCaseInsensitive(headers http.Header, name string) string {
	return headerValueCaseInsensitive(headers, name)
}

// HeaderValuesCaseInsensitive returns all non-empty header values matching name case-insensitively.
func HeaderValuesCaseInsensitive(headers http.Header, name string) []string {
	if headers == nil {
		return nil
	}
	var result []string
	for key, values := range headers {
		if strings.EqualFold(key, name) {
			for _, value := range values {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	}
	return result
}

func headerValueCaseInsensitive(headers http.Header, name string) string {
	if headers == nil {
		return ""
	}
	if value := strings.TrimSpace(headers.Get(name)); value != "" {
		return value
	}
	for key, values := range headers {
		if !strings.EqualFold(key, name) {
			continue
		}
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

func extractClaudeCodeSessionIDFromPayload(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	userID := gjson.GetBytes(payload, "metadata.user_id").String()
	if userID == "" {
		return ""
	}
	if matches := claudeCodeSessionSuffixPattern.FindStringSubmatch(userID); len(matches) >= 2 {
		return matches[1]
	}
	if len(userID) > 0 && userID[0] == '{' {
		return strings.TrimSpace(gjson.Get(userID, "session_id").String())
	}
	return ""
}

// ClaudeAgentSessionUUID resolves the stable Claude session UUID used for
// upstream request continuity (upstream 086ad91bd970).
func ClaudeAgentSessionUUID(headers http.Header, originalPayload, translatedPayload []byte, metadataSets ...map[string]any) string {
	return claudeAgentSessionUUID(headers, originalPayload, translatedPayload, metadataSets...)
}

// ClaudeAgentSessionUUIDForRequest preserves Claude-specific session signals only
// for a confirmed native caller. Other callers use protocol session fields,
// execution metadata, or the stable derived conversation root.
func ClaudeAgentSessionUUIDForRequest(headers http.Header, originalPayload, translatedPayload []byte, confirmedClaudeCode bool, metadataSets ...map[string]any) string {
	if !confirmedClaudeCode {
		if headers != nil {
			headers = headers.Clone()
			for key := range headers {
				if strings.EqualFold(key, "X-Claude-Code-Session-Id") {
					delete(headers, key)
				}
			}
		}
		originalPayload = withoutClaudeMetadataUserID(originalPayload)
		translatedPayload = withoutClaudeMetadataUserID(translatedPayload)
	}
	return claudeAgentSessionUUID(headers, originalPayload, translatedPayload, metadataSets...)
}

type claudeExecutionMetadataKey struct{}

// WithClaudeExecutionMetadata attaches whether the request had an explicit execution session metadata to ctx.
func WithClaudeExecutionMetadata(ctx context.Context, present bool) context.Context {
	return context.WithValue(ctx, claudeExecutionMetadataKey{}, present)
}

// ClaudeExecutionMetadataFromContext retrieves whether execution metadata was present from ctx.
func ClaudeExecutionMetadataFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(claudeExecutionMetadataKey{}).(bool); ok {
		return v
	}
	return false
}

// ClaudeRequestHasExecutionMetadata reports whether the request explicitly carried
// an internal execution session metadata key.
func ClaudeRequestHasExecutionMetadata(metadataSets ...map[string]any) bool {
	for _, metadata := range metadataSets {
		if val, ok := metadata[cliproxyexecutor.ExecutionSessionMetadataKey].(string); ok && strings.TrimSpace(val) != "" {
			return true
		}
	}
	return false
}

func claudeAgentSessionUUID(headers http.Header, originalPayload, translatedPayload []byte, metadataSets ...map[string]any) string {
	metadata := mergeClaudeSessionMetadata(metadataSets...)
	identity := cliproxyauth.ExtractSessionID(headers, originalPayload, metadata)
	if identity == "" && len(translatedPayload) > 0 {
		identity = cliproxyauth.ExtractSessionID(headers, translatedPayload, metadata)
	}
	if identity == "" {
		return uuid.NewString()
	}
	if strings.HasPrefix(identity, "claude:") {
		if parsed, errParse := uuid.Parse(strings.TrimPrefix(identity, "claude:")); errParse == nil {
			return parsed.String()
		}
	}
	if parsed, errParse := uuid.Parse(identity); errParse == nil {
		return parsed.String()
	}
	stableInput := "cli-proxy-api\x00claude\x00agent-conversation\x00" + identity
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(stableInput)).String()
}

func withoutClaudeMetadataUserID(payload []byte) []byte {
	if len(payload) == 0 {
		return payload
	}
	updated, errDelete := sjson.DeleteBytes(payload, "metadata.user_id")
	if errDelete != nil {
		return payload
	}
	return updated
}

func mergeClaudeSessionMetadata(metadataSets ...map[string]any) map[string]any {
	var merged map[string]any
	for _, metadata := range metadataSets {
		if len(metadata) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]any)
		}
		for key, value := range metadata {
			if _, exists := merged[key]; !exists {
				merged[key] = value
			}
		}
	}
	return merged
}
