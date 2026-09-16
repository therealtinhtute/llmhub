package helps

import (
	"strings"

	kimiauth "github.com/therealtinhtute/llmhub/internal/auth/kimi"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// ResolveKimiResponsesURL resolves the upstream URL for Kimi Responses API requests.
// Ported from upstream CLIProxyAPI commit d4146bde1248 ("feat(kimi): support
// openai responses api").
func ResolveKimiResponsesURL(auth *cliproxyauth.Auth) string {
	if auth != nil && auth.Attributes != nil {
		if raw := strings.TrimRight(strings.TrimSpace(auth.Attributes["base_url"]), "/"); raw != "" {
			if strings.HasSuffix(raw, "/v1") {
				return raw + "/responses"
			}
			return raw + "/v1/responses"
		}
	}
	return kimiauth.KimiAPIBaseURL + "/v1/responses"
}
