package config

import "testing"

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
