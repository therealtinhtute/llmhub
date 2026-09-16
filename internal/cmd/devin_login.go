package cmd

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/config"
	sdkAuth "github.com/therealtinhtute/llmhub/sdk/auth"
)

// DoDevinLogin triggers the OAuth PKCE or headless token flow for Devin / Cognition and saves credentials.
// Ported from upstream CLIProxyAPI internal/cmd/devin_login.go (f94752762bb9).
func DoDevinLogin(cfg *config.Config, options *LoginOptions) {
	if options == nil {
		options = &LoginOptions{}
	}

	promptFn := options.Prompt
	if promptFn == nil {
		promptFn = defaultProjectPrompt()
	}

	manager := newAuthManager()
	authOpts := &sdkAuth.LoginOptions{
		NoBrowser:    options.NoBrowser,
		CallbackPort: options.CallbackPort,
		Metadata:     map[string]string{},
		Prompt:       promptFn,
	}

	record, savedPath, err := manager.Login(context.Background(), "devin", cfg, authOpts)
	if err != nil {
		log.Errorf("Devin authentication failed: %v", err)
		return
	}

	if savedPath != "" {
		fmt.Printf("Authentication saved to %s\n", savedPath)
	}
	if record != nil && record.Label != "" {
		fmt.Printf("Authenticated as %s\n", record.Label)
	}
	fmt.Println("Devin authentication successful!")
}
