package auth

import (
	"maps"
	"strings"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
// (sdk/cliproxy/auth/api_key_model_capabilities.go). The fork has no API-key
// capability snapshot producer yet; the API-key key is kept so ResolvedModelInfo
// preserves upstream's home-first precedence when one is introduced.
const (
	resolvedAPIKeyModelInfoMetadataKey = "cliproxy.resolved_api_key_model_info"
	resolvedHomeModelInfoMetadataKey   = "cliproxy.resolved_home_model_info"
)

// ResolvedAPIKeyModelInfo returns the exact configured model definition bound to
// this API-key execution attempt.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func ResolvedAPIKeyModelInfo(req cliproxyexecutor.Request) (*registry.ModelInfo, bool) {
	modelInfo, ok := req.Metadata[resolvedAPIKeyModelInfoMetadataKey].(*registry.ModelInfo)
	if !ok || modelInfo == nil {
		return nil, false
	}
	return modelInfo, true
}

// ResolvedModelInfo returns the authoritative model capabilities bound to this
// execution attempt. Home-dispatched model info wins over configured API-key
// model info.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func ResolvedModelInfo(req cliproxyexecutor.Request) (*registry.ModelInfo, bool) {
	if modelInfo, ok := req.Metadata[resolvedHomeModelInfoMetadataKey].(*registry.ModelInfo); ok && modelInfo != nil {
		return modelInfo, true
	}
	return ResolvedAPIKeyModelInfo(req)
}

// attachResolvedHomeModelInfo binds the home-dispatched model capabilities to
// the downstream executor request via request metadata.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func attachResolvedHomeModelInfo(req cliproxyexecutor.Request, modelInfo *registry.ModelInfo) cliproxyexecutor.Request {
	if modelInfo == nil {
		return req
	}
	metadata := make(map[string]any, len(req.Metadata)+1)
	maps.Copy(metadata, req.Metadata)
	metadata[resolvedHomeModelInfoMetadataKey] = modelInfo
	req.Metadata = metadata
	return req
}

// CodexAPIKeyModelIsCompat reports whether the selected codex-api-key model has
// is-compat enabled. When true and codex.optimize-multi-agent-v2 is also true,
// Codex MultiAgentV2 agent_message items are converted into portable Responses
// message/user input for third-party Responses-compatible endpoints.
func CodexAPIKeyModelIsCompat(cfg *internalconfig.Config, auth *Auth, model string) bool {
	if cfg == nil || auth == nil || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return false
	}
	entry := resolveCodexAPIKeyConfig(cfg, auth)
	if entry == nil || len(entry.Models) == 0 {
		return false
	}
	requested := strings.TrimSpace(model)
	if requested == "" {
		return false
	}
	baseModel := strings.TrimSpace(thinking.ParseSuffix(requested).ModelName)
	if baseModel == "" {
		baseModel = requested
	}
	for i := range entry.Models {
		name := strings.TrimSpace(entry.Models[i].Name)
		alias := strings.TrimSpace(entry.Models[i].Alias)
		if name == "" {
			name = alias
		}
		if alias == "" {
			alias = name
		}
		if name == "" {
			continue
		}
		if strings.EqualFold(name, requested) || strings.EqualFold(name, baseModel) ||
			strings.EqualFold(alias, requested) || strings.EqualFold(alias, baseModel) {
			return entry.Models[i].IsCompat
		}
	}
	return false
}
