package util

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// IsCodexResponsesLiteRequest recognizes the native header and its websocket metadata mirror.
//
// Ported from upstream CLIProxyAPI internal/util/codex.go (f702bc1ac263).
func IsCodexResponsesLiteRequest(body []byte, headers http.Header) bool {
	if strings.EqualFold(strings.TrimSpace(headers.Get("X-OpenAI-Internal-Codex-Responses-Lite")), "true") {
		return true
	}
	// Codex Desktop mirrors websocket-only request headers into client_metadata.
	value := gjson.GetBytes(body, "client_metadata.ws_request_header_x_openai_internal_codex_responses_lite")
	return value.Type == gjson.True || value.Type == gjson.String && strings.EqualFold(strings.TrimSpace(value.String()), "true")
}
