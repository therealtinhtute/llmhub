package auth

import (
	"strings"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/thinking"
)

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
