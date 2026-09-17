package helps

import (
	"github.com/therealtinhtute/llmhub/internal/thinking"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// ApplyRequestThinking preserves the registry lookup path unless the auth
// manager bound authoritative model capabilities to this execution attempt
// (e.g., a home-dispatched model_info carrying the real Thinking levels).
// Without it, home-dispatched models unknown to the local registry fall back to
// the user-defined path, which can rewrite a level-based config into a budget.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
// (internal/runtime/executor/helps/model_capabilities.go). The fork's thinking
// pipeline has no summary stage and no API-key capability snapshot producer, so
// the helper resolves attached model info and otherwise keeps the existing
// registry-lookup behavior.
func ApplyRequestThinking(body []byte, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, fromFormat, toFormat, provider string) ([]byte, error) {
	if modelInfo, ok := cliproxyauth.ResolvedModelInfo(req); ok {
		return thinking.ApplyThinkingWithModelInfo(body, req.Model, fromFormat, toFormat, provider, modelInfo)
	}
	return thinking.ApplyThinking(body, req.Model, fromFormat, toFormat, provider)
}
