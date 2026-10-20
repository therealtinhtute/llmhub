package config

import (
	"testing"
	"time"
)

// Ported from upstream CLIProxyAPI internal/config/claude_code_test.go
// (44eaef0009f8). Upstream nests the flag under claude.model-level-cooling;
// this repository's flat schema exposes it as claude-model-level-cooling.
func TestParseConfigBytesClaudeModelLevelCooling(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "defaults to false",
			yaml: "port: 8317\n",
			want: false,
		},
		{
			name: "enables model-level cooling",
			yaml: "claude-model-level-cooling: true\n",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errParse := ParseConfigBytes([]byte(tt.yaml))
			if errParse != nil {
				t.Fatalf("ParseConfigBytes() error = %v", errParse)
			}
			if got := cfg.ClaudeModelLevelCooling; got != tt.want {
				t.Fatalf("cfg.ClaudeModelLevelCooling = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestParseConfigBytesClaudeSessionDatePinDays(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want int
	}{
		{name: "defaults to 7 days", yaml: "port: 8317\n", want: 7},
		{name: "legacy layout", yaml: "claude:\n  session-date-pin-days: 14\n", want: 14},
		{name: "v8 layout", yaml: "config-version: 8\nupstream:\n  claude:\n    session-date-pin-days: 7\n", want: 7},
		{name: "v8 explicit zero", yaml: "config-version: 8\nupstream:\n  claude:\n    session-date-pin-days: 0\n", want: 0},
		{name: "legacy explicit zero", yaml: "claude:\n  session-date-pin-days: 0\n", want: 0},
		{name: "flat schema", yaml: "claude-session-date-pin-days: 21\n", want: 21},
		{name: "flat explicit zero", yaml: "claude-session-date-pin-days: 0\n", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errParse := ParseConfigBytes([]byte(tt.yaml))
			if errParse != nil {
				t.Fatalf("ParseConfigBytes() error = %v", errParse)
			}
			if got := cfg.Claude.SessionDatePinDays; got == nil || *got != tt.want {
				t.Fatalf("cfg.Claude.SessionDatePinDays = %v, want %d", got, tt.want)
			}
		})
	}
}

func TestClaudeSessionDatePinRetention(t *testing.T) {
	zero := 0
	fourteen := 14
	cfgDefault := &Config{}
	cfgDefault.NormalizeClaudeConfig()
	if got := cfgDefault.ClaudeSessionDatePinRetention(); got != 7*24*time.Hour {
		t.Fatalf("default retention = %v, want %v", got, 7*24*time.Hour)
	}

	cfgZero := &Config{Claude: ClaudeConfig{SessionDatePinDays: &zero}}
	cfgZero.NormalizeClaudeConfig()
	if got := cfgZero.ClaudeSessionDatePinRetention(); got != 0 {
		t.Fatalf("zero retention = %v, want 0", got)
	}

	cfg14 := &Config{Claude: ClaudeConfig{SessionDatePinDays: &fourteen}}
	cfg14.NormalizeClaudeConfig()
	if got := cfg14.ClaudeSessionDatePinRetention(); got != 14*24*time.Hour {
		t.Fatalf("14 days retention = %v, want %v", got, 14*24*time.Hour)
	}
}
