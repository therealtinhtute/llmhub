package cmd

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/config"
	sdkAuth "github.com/therealtinhtute/llmhub/sdk/auth"
)

// DoMetaLogin triggers the OAuth device-code flow for the Meta provider and saves tokens.
// Ported from upstream CLIProxyAPI internal/cmd/meta_login.go (54d4f4c0193c).
func DoMetaLogin(cfg *config.Config, options *LoginOptions) {
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

	record, savedPath, err := manager.Login(context.Background(), "meta", cfg, authOpts)
	if err != nil {
		log.Errorf("Meta authentication failed: %v", err)
		return
	}

	if savedPath != "" {
		fmt.Printf("Authentication saved to %s\n", savedPath)
	}
	if record != nil && record.Label != "" {
		fmt.Printf("Authenticated as %s\n", record.Label)
	}
	fmt.Println("Meta authentication successful!")
}
