package helps

import (
	"github.com/therealtinhtute/llmhub/internal/thinking"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// APIKeyModelIsCompat reports whether the selected API-key model enables
// compatibility handling for Claude thinking blocks.
func APIKeyModelIsCompat(req cliproxyexecutor.Request) bool {
	modelInfo, ok := cliproxyauth.ResolvedAPIKeyModelInfo(req)
	return ok && modelInfo != nil && modelInfo.IsCompat
}

// ApplyRequestThinking preserves the registry lookup path unless the auth
// manager bound authoritative model capabilities to this execution attempt
// (e.g., a home-dispatched model_info carrying the real Thinking levels).
// Without it, home-dispatched models unknown to the local registry fall back to
// the user-defined path, which can rewrite a level-based config into a budget.
//
// Ported from upstream CLIProxyAPI (internal/runtime/executor/helps/
// model_capabilities.go): preserves summary visibility resolved across source
// translation and request normalization.
func ApplyRequestThinking(body []byte, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, fromFormat, toFormat, provider string, normalizedUpdatesChanged ...bool) ([]byte, error) {
	originalSource := opts.OriginalRequest
	if len(originalSource) == 0 {
		originalSource = req.Payload
	}
	source := req.Payload
	if len(source) == 0 {
		source = opts.OriginalRequest
	}
	summaryConfig := translatedRequestSummaryConfig(body, req.Payload, originalSource, req.Model, fromFormat, toFormat)
	if modelInfo, ok := cliproxyauth.ResolvedModelInfo(req); ok {
		return thinking.ApplyThinkingWithModelInfoAndSummary(body, source, req.Model, fromFormat, toFormat, provider, modelInfo, summaryConfig, normalizedUpdatesChanged...)
	}
	return thinking.ApplyThinkingWithSourceAndSummary(body, source, req.Model, fromFormat, toFormat, provider, summaryConfig, normalizedUpdatesChanged...)
}
