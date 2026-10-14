package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestClaudeP4_OmittedMaxTokensUsesRegistryLimit(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		rawJSON   string
		wantLimit int64
	}{
		{
			name:      "opus 5 omitted field uses registry limit",
			model:     "claude-opus-5",
			rawJSON:   `{"model":"claude-opus-5","messages":[{"role":"user","content":"hi"}]}`,
			wantLimit: 128000,
		},
		{
			name:      "opus 5 null max_tokens uses registry limit",
			model:     "claude-opus-5",
			rawJSON:   `{"model":"claude-opus-5","max_tokens":null,"messages":[{"role":"user","content":"hi"}]}`,
			wantLimit: 128000,
		},
		{
			name:      "null max_tokens falls back to explicit max_completion_tokens",
			model:     "claude-opus-5",
			rawJSON:   `{"model":"claude-opus-5","max_tokens":null,"max_completion_tokens":4096,"messages":[{"role":"user","content":"hi"}]}`,
			wantLimit: 4096,
		},
		{
			name:      "sonnet 4.6 omitted field uses registry limit",
			model:     "claude-sonnet-4-6",
			rawJSON:   `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}`,
			wantLimit: 64000,
		},
		{
			name:      "opus 4.1 omitted field keeps 32000 default",
			model:     "claude-opus-4-1-20250805",
			rawJSON:   `{"model":"claude-opus-4-1-20250805","messages":[{"role":"user","content":"hi"}]}`,
			wantLimit: 32000,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := ConvertOpenAIRequestToClaude(tc.model, []byte(tc.rawJSON), false)
			if got := gjson.GetBytes(out, "max_tokens").Int(); got != tc.wantLimit {
				t.Fatalf("max_tokens = %d, want %d. Output: %s", got, tc.wantLimit, string(out))
			}
		})
	}
}
