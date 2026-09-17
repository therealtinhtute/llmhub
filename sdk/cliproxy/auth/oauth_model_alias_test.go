package auth

import (
	"testing"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
)

func TestResolveOAuthUpstreamModel_SuffixPreservation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		aliases map[string][]internalconfig.OAuthModelAlias
		channel string
		input   string
		want    string
	}{
		{
			name: "numeric suffix preserved",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro(8192)",
			want:    "gemini-2.5-pro-exp-03-25(8192)",
		},
		{
			name: "level suffix preserved",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"claude": {{Name: "claude-sonnet-4-5-20250514", Alias: "claude-sonnet-4-5"}},
			},
			channel: "claude",
			input:   "claude-sonnet-4-5(high)",
			want:    "claude-sonnet-4-5-20250514(high)",
		},
		{
			name: "no suffix unchanged",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro",
			want:    "gemini-2.5-pro-exp-03-25",
		},
		{
			name: "config suffix takes priority",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"claude": {{Name: "claude-sonnet-4-5-20250514(low)", Alias: "claude-sonnet-4-5"}},
			},
			channel: "claude",
			input:   "claude-sonnet-4-5(high)",
			want:    "claude-sonnet-4-5-20250514(low)",
		},
		{
			name: "auto suffix preserved",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro(auto)",
			want:    "gemini-2.5-pro-exp-03-25(auto)",
		},
		{
			name: "none suffix preserved",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro(none)",
			want:    "gemini-2.5-pro-exp-03-25(none)",
		},
		{
			name: "kimi suffix preserved",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"kimi": {{Name: "kimi-k2.5", Alias: "k2.5"}},
			},
			channel: "kimi",
			input:   "k2.5(high)",
			want:    "kimi-k2.5(high)",
		},
		{
			// Ported from upstream CLIProxyAPI commit 8335eac7.
			name: "meta suffix preserved",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"meta": {{Name: "muse-spark-1.3", Alias: "muse-latest"}},
			},
			channel: "meta",
			input:   "muse-latest(high)",
			want:    "muse-spark-1.3(high)",
		},
		{
			name: "case insensitive alias lookup with suffix",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "Gemini-2.5-Pro"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro(high)",
			want:    "gemini-2.5-pro-exp-03-25(high)",
		},
		{
			name: "no alias returns empty",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "gemini-cli",
			input:   "unknown-model(high)",
			want:    "",
		},
		{
			name: "wrong channel returns empty",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "claude",
			input:   "gemini-2.5-pro(high)",
			want:    "",
		},
		{
			name: "empty suffix filtered out",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro()",
			want:    "gemini-2.5-pro-exp-03-25",
		},
		{
			name: "incomplete suffix treated as no suffix",
			aliases: map[string][]internalconfig.OAuthModelAlias{
				"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro(high"}},
			},
			channel: "gemini-cli",
			input:   "gemini-2.5-pro(high",
			want:    "gemini-2.5-pro-exp-03-25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager(nil, nil, nil)
			mgr.SetConfig(&internalconfig.Config{})
			mgr.SetOAuthModelAlias(tt.aliases)

			auth := createAuthForChannel(tt.channel)
			got := mgr.resolveOAuthUpstreamModel(auth, tt.input)
			if got != tt.want {
				t.Errorf("resolveOAuthUpstreamModel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func createAuthForChannel(channel string) *Auth {
	switch channel {
	case "gemini-cli":
		return &Auth{Provider: "gemini-cli"}
	case "claude":
		return &Auth{Provider: "claude", Attributes: map[string]string{"auth_kind": "oauth"}}
	case "vertex":
		return &Auth{Provider: "vertex", Attributes: map[string]string{"auth_kind": "oauth"}}
	case "codex":
		return &Auth{Provider: "codex", Attributes: map[string]string{"auth_kind": "oauth"}}
	case "aistudio":
		return &Auth{Provider: "aistudio"}
	case "antigravity":
		return &Auth{Provider: "antigravity"}
	case "kimi":
		return &Auth{Provider: "kimi"}
	case "meta":
		// Ported from upstream CLIProxyAPI commit 8335eac7: meta OAuth
		// credentials mint an LLM API key but remain OAuth-backed.
		return &Auth{Provider: "meta", Attributes: map[string]string{"auth_kind": "oauth"}}
	default:
		return &Auth{Provider: channel}
	}
}

func TestOAuthModelAliasChannel_Kimi(t *testing.T) {
	t.Parallel()

	if got := OAuthModelAliasChannel("kimi", "oauth"); got != "kimi" {
		t.Fatalf("OAuthModelAliasChannel() = %q, want %q", got, "kimi")
	}
}

// Ported from upstream CLIProxyAPI commit 8335eac7.
func TestOAuthModelAliasChannel_Meta(t *testing.T) {
	t.Parallel()

	if got := OAuthModelAliasChannel("meta", "oauth"); got != "meta" {
		t.Fatalf("OAuthModelAliasChannel() = %q, want %q", got, "meta")
	}
	if got := OAuthModelAliasChannel("meta", "api_key"); got != "" {
		t.Fatalf("OAuthModelAliasChannel() = %q, want empty channel for meta-api-key", got)
	}
}

func TestApplyOAuthModelAlias_SuffixPreservation(t *testing.T) {
	t.Parallel()

	aliases := map[string][]internalconfig.OAuthModelAlias{
		"gemini-cli": {{Name: "gemini-2.5-pro-exp-03-25", Alias: "gemini-2.5-pro"}},
	}

	mgr := NewManager(nil, nil, nil)
	mgr.SetConfig(&internalconfig.Config{})
	mgr.SetOAuthModelAlias(aliases)

	auth := &Auth{ID: "test-auth-id", Provider: "gemini-cli"}

	resolvedModel := mgr.applyOAuthModelAlias(auth, "gemini-2.5-pro(8192)")
	if resolvedModel != "gemini-2.5-pro-exp-03-25(8192)" {
		t.Errorf("applyOAuthModelAlias() model = %q, want %q", resolvedModel, "gemini-2.5-pro-exp-03-25(8192)")
	}
}

// Ported from upstream CLIProxyAPI commit 8335eac7
// (TestApplyOAuthModelAlias_Meta). The upstream ForceMapping field/assertion is
// omitted because OAuthModelAlias.ForceMapping is not part of the local struct.
func TestApplyOAuthModelAlias_Meta(t *testing.T) {
	t.Parallel()

	aliases := map[string][]internalconfig.OAuthModelAlias{
		"meta": {
			{
				Name:  "muse-spark-1.3",
				Alias: "muse-latest",
				Fork:  true,
			},
		},
	}

	mgr := NewManager(nil, nil, nil)
	mgr.SetConfig(&internalconfig.Config{})
	mgr.SetOAuthModelAlias(aliases)

	// Meta OAuth credentials mint an LLM API key; aliases must still apply.
	auth := &Auth{
		ID:       "meta-auth",
		Provider: "meta",
		Attributes: map[string]string{
			"auth_kind": "oauth",
		},
	}

	resolved := mgr.applyOAuthModelAlias(auth, "muse-latest")
	if resolved != "muse-spark-1.3" {
		t.Fatalf("applyOAuthModelAlias() = %q, want muse-spark-1.3", resolved)
	}

	// meta-api-key credentials use per-credential models, not oauth-model-alias.
	apiKeyAuth := &Auth{
		ID:       "meta-apikey",
		Provider: "meta",
		Attributes: map[string]string{
			"auth_kind": "apikey",
			"api_key":   "meta-key",
		},
	}
	if got := mgr.applyOAuthModelAlias(apiKeyAuth, "muse-latest"); got != "muse-latest" {
		t.Fatalf("applyOAuthModelAlias(apikey) = %q, want unchanged muse-latest", got)
	}
}
