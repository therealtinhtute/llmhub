package executor

// Per-credential Codex cloaking override coverage, ported from upstream
// CLIProxyAPI commit f351924f42cb ("feat(codex): add support for disable codex
// cloaking per credential", issue #6034).
//
// Adaptation: upstream's global toggle is the static config field
// cfg.Codex.DisableCodexCloaking; the fork's global toggle is the cloaking
// runtime control carried on the request context
// (cliproxyexecutor.WithCodexCloakingDisabled). The precedence order is
// identical: auth attribute -> credential disable-codex-cloaking -> global.
// Upstream also hard-overrides User-Agent/Originator after custom headers so
// header:* attributes cannot defeat cloaking; the fork deliberately keeps its
// layered precedence (existing header > config defaults > client > official
// identity) so those attrs still win when cloaking is enabled.

import (
	"context"
	"net/http"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

func TestCodexPerCredentialDisableCloaking_HTTP_ExplicitTrue(t *testing.T) {
	cfg := &config.Config{
		CodexKey: []config.CodexKey{
			{
				APIKey:               "custom-key",
				BaseURL:              "https://example.com/v1",
				DisableCodexCloaking: boolPtr(true),
			},
		},
	}

	auth := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"api_key":           "custom-key",
			"base_url":          "https://example.com/v1",
			"header:User-Agent": "CustomAgent/1.0",
			"header:Originator": "CustomOriginator",
			cliproxyauth.AttributeCodexDisableCloaking: "true",
		},
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	applyCodexHeaders(req, auth, "test-token", false, cfg, nil)

	if got := req.Header.Get("User-Agent"); got != "CustomAgent/1.0" {
		t.Errorf("User-Agent = %q, want %q", got, "CustomAgent/1.0")
	}
	if got := req.Header.Get("Originator"); got != "CustomOriginator" {
		t.Errorf("Originator = %q, want %q", got, "CustomOriginator")
	}
}

func TestCodexPerCredentialDisableCloaking_HTTP_ExplicitFalseOverridesGlobalTrue(t *testing.T) {
	cfg := &config.Config{
		CodexKey: []config.CodexKey{
			{
				APIKey:               "cloaked-key",
				BaseURL:              "https://example.com/v1",
				DisableCodexCloaking: boolPtr(false),
			},
		},
	}

	// OAuth-style auth so the official Codex identity is observable: API-key
	// credentials never get Originator/UA injection locally (the fork lacks
	// upstream's post-custom-headers hard override).
	auth := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"auth_kind": "oauth",
			cliproxyauth.AttributeCodexDisableCloaking: "false",
		},
		Metadata: map[string]any{
			"account_id": "acc-123",
		},
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req = req.WithContext(cliproxyexecutor.WithCodexCloakingDisabled(req.Context()))

	applyCodexHeaders(req, auth, "test-token", false, cfg, nil)

	// Since disable-codex-cloaking is explicitly false, cloaking should be enforced.
	if got := req.Header.Get("User-Agent"); got != codexUserAgent {
		t.Errorf("User-Agent = %q, want %q", got, codexUserAgent)
	}
	if got := req.Header.Get("Originator"); got != codexOriginator {
		t.Errorf("Originator = %q, want %q", got, codexOriginator)
	}
}

func TestCodexPerCredentialDisableCloaking_HTTP_FallbackToGlobal(t *testing.T) {
	cfg := &config.Config{
		CodexKey: []config.CodexKey{
			{
				APIKey:  "key-nil",
				BaseURL: "https://example.com/v1",
			},
		},
	}

	authNil := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"api_key":  "key-nil",
			"base_url": "https://example.com/v1",
		},
	}

	// 1. Global disabled (runtime control on ctx), credential nil -> cloaking
	// disabled: no official identity is injected, so User-Agent stays empty.
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req = req.WithContext(cliproxyexecutor.WithCodexCloakingDisabled(req.Context()))
	applyCodexHeaders(req, authNil, "test-token", false, cfg, nil)
	if got := req.Header.Get("User-Agent"); got != "" {
		t.Errorf("Global disabled fallback User-Agent = %q, want empty (cloaking off)", got)
	}

	// 2. Global enabled (no runtime control), credential nil -> cloaking enabled
	req2, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	applyCodexHeaders(req2, authNil, "test-token", false, cfg, nil)
	if got := req2.Header.Get("User-Agent"); got != codexUserAgent {
		t.Errorf("Global enabled fallback User-Agent = %q, want %q", got, codexUserAgent)
	}
}

func TestCodexPerCredentialDisableCloaking_WebSocket_ExplicitTrue(t *testing.T) {
	cfg := &config.Config{}

	auth := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"api_key":           "custom-key",
			"base_url":          "https://example.com/v1",
			"header:User-Agent": "CustomWSAgent/1.0",
			"header:Originator": "CustomWSOriginator",
			cliproxyauth.AttributeCodexDisableCloaking: "true",
		},
	}

	headers := applyCodexWebsocketHeaders(context.Background(), nil, auth, "test-token", cfg, true)

	if got := headers.Get("User-Agent"); got != "CustomWSAgent/1.0" {
		t.Errorf("WebSocket User-Agent = %q, want %q", got, "CustomWSAgent/1.0")
	}
	if got := headers.Get("Originator"); got != "CustomWSOriginator" {
		t.Errorf("WebSocket Originator = %q, want %q", got, "CustomWSOriginator")
	}
}

func TestCodexPerCredentialDisableCloaking_WebSocket_ExplicitFalseOverridesGlobalTrue(t *testing.T) {
	cfg := &config.Config{}

	// OAuth-style auth: API-key credentials never get identity injection locally.
	auth := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"auth_kind": "oauth",
			cliproxyauth.AttributeCodexDisableCloaking: "false",
		},
		Metadata: map[string]any{
			"account_id": "acc-123",
		},
	}

	ctx := cliproxyexecutor.WithCodexCloakingDisabled(context.Background())
	headers := applyCodexWebsocketHeaders(ctx, nil, auth, "test-token", cfg, true)

	if got := headers.Get("User-Agent"); got != codexUserAgent {
		t.Errorf("WebSocket User-Agent = %q, want %q", got, codexUserAgent)
	}
	if got := headers.Get("Originator"); got != codexOriginator {
		t.Errorf("WebSocket Originator = %q, want %q", got, codexOriginator)
	}
}

func TestCodexPerCredentialDisableCloaking_OAuthUnaffectedByAPIKeyOverride(t *testing.T) {
	cfg := &config.Config{
		CodexKey: []config.CodexKey{
			{
				APIKey:               "custom-key",
				BaseURL:              "https://example.com/v1",
				DisableCodexCloaking: boolPtr(true),
			},
		},
	}

	// OAuth credential (does not match codex-api-key)
	oauthAuth := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{
			"account_id": "acc-123",
		},
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	applyCodexHeaders(req, oauthAuth, "oauth-token", false, cfg, nil)

	// OAuth credential must remain cloaked with official Codex identity
	if got := req.Header.Get("User-Agent"); got != codexUserAgent {
		t.Errorf("OAuth User-Agent = %q, want %q", got, codexUserAgent)
	}
	if got := req.Header.Get("Originator"); got != codexOriginator {
		t.Errorf("OAuth Originator = %q, want %q", got, codexOriginator)
	}
}

func TestCodexPerCredentialDisableCloaking_ResolvedFromConfigWithoutAttribute(t *testing.T) {
	cfg := &config.Config{
		CodexKey: []config.CodexKey{
			{
				APIKey:               "lookup-key",
				BaseURL:              "https://example.com/v1",
				DisableCodexCloaking: boolPtr(true),
			},
		},
	}

	// Auth object with only APIKey/BaseURL attributes (no pre-synthesized disable_codex_cloaking attribute)
	authWithoutAttr := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"api_key":           "lookup-key",
			"base_url":          "https://example.com/v1",
			"header:User-Agent": "CustomLookupAgent/1.0",
		},
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	applyCodexHeaders(req, authWithoutAttr, "lookup-key", false, cfg, nil)

	// Since resolveCodexKeyConfig finds DisableCodexCloaking == true in cfg.CodexKey, cloaking must be disabled.
	if got := req.Header.Get("User-Agent"); got != "CustomLookupAgent/1.0" {
		t.Errorf("User-Agent = %q, want %q", got, "CustomLookupAgent/1.0")
	}
}
