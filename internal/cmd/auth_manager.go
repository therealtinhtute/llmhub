package cmd

import (
	sdkAuth "github.com/therealtinhtute/llmhub/sdk/auth"
)

// newAuthManager creates a new authentication manager instance with all supported
// authenticators and a file-based token store. It initializes authenticators for
// Gemini, Codex, Claude, Antigravity, Kimi, xAI, Devin, and Meta providers.
//
// Returns:
//   - *sdkAuth.Manager: A configured authentication manager instance
//
// Meta registration ported from upstream CLIProxyAPI commit 54d4f4c0193c.
func newAuthManager() *sdkAuth.Manager {
	store := sdkAuth.GetTokenStore()
	manager := sdkAuth.NewManager(store,
		sdkAuth.NewGeminiAuthenticator(),
		sdkAuth.NewCodexAuthenticator(),
		sdkAuth.NewClaudeAuthenticator(),
		sdkAuth.NewAntigravityAuthenticator(),
		sdkAuth.NewKimiAuthenticator(),
		sdkAuth.NewXAIAuthenticator(),
		sdkAuth.NewDevinAuthenticator(),
		sdkAuth.NewMetaAuthenticator(),
	)
	return manager
}
