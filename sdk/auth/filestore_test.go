package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAccessToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]any
		expected string
	}{
		{
			"antigravity top-level access_token",
			map[string]any{"access_token": "tok-abc"},
			"tok-abc",
		},
		{
			"gemini nested token.access_token",
			map[string]any{
				"token": map[string]any{"access_token": "tok-nested"},
			},
			"tok-nested",
		},
		{
			"top-level takes precedence over nested",
			map[string]any{
				"access_token": "tok-top",
				"token":        map[string]any{"access_token": "tok-nested"},
			},
			"tok-top",
		},
		{
			"empty metadata",
			map[string]any{},
			"",
		},
		{
			"whitespace-only access_token",
			map[string]any{"access_token": "   "},
			"",
		},
		{
			"wrong type access_token",
			map[string]any{"access_token": 12345},
			"",
		},
		{
			"token is not a map",
			map[string]any{"token": "not-a-map"},
			"",
		},
		{
			"nested whitespace-only",
			map[string]any{
				"token": map[string]any{"access_token": "  "},
			},
			"",
		},
		{
			"fallback to nested when top-level empty",
			map[string]any{
				"access_token": "",
				"token":        map[string]any{"access_token": "tok-fallback"},
			},
			"tok-fallback",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractAccessToken(tt.metadata)
			if got != tt.expected {
				t.Errorf("extractAccessToken() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFileTokenStore_ReadAuthFile_NormalizesKiro9RouterJSON(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "kiro-account.json")
	raw := []byte(`{
		"provider":"kiro",
		"authType":"oauth",
		"accessToken":"access-1",
		"refreshToken":"refresh-1",
		"expiresAt":"2026-05-29T07:43:18.341Z",
		"isActive":false,
		"providerSpecificData":{
			"clientId":"client-1",
			"clientSecret":"secret-1",
			"region":"us-east-1",
			"authMethod":"builder-id"
		}
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}

	store := NewFileTokenStore()
	auth, err := store.readAuthFile(path, baseDir)
	if err != nil {
		t.Fatalf("readAuthFile() error = %v", err)
	}
	if auth.Provider != "kiro" {
		t.Fatalf("provider = %q, want kiro", auth.Provider)
	}
	if !auth.Disabled {
		t.Fatal("expected inactive 9router connection to become disabled auth")
	}
	if auth.Metadata["access_token"] != "access-1" || auth.Metadata["refresh_token"] != "refresh-1" {
		t.Fatalf("metadata not normalized: %#v", auth.Metadata)
	}

	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted auth file: %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(persisted, &metadata); err != nil {
		t.Fatalf("persisted file is not JSON: %v", err)
	}
	if metadata["type"] != "kiro" || metadata["provider"] != nil {
		t.Fatalf("persisted metadata = %#v, want llmhub kiro shape", metadata)
	}
}
