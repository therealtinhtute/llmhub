package claude

import (
	"context"
	"testing"

	"github.com/tidwall/gjson"
)

func claudeP4TerminalResponse(usageJSON string) []byte {
	return []byte(`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5","status":"completed","output":[],"usage":` + usageJSON + `}}`)
}

func TestClaudeP4_ReasoningTokensMappedToThinkingTokens(t *testing.T) {
	original := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	line := claudeP4TerminalResponse(`{"input_tokens":420,"output_tokens":518,"output_tokens_details":{"reasoning_tokens":163},"total_tokens":938}`)
	var param any
	chunks := ConvertCodexResponseToClaude(context.Background(), "m", original, nil, line, &param)
	found := false
	for _, chunk := range chunks {
		if gjson.GetBytes(chunk, "type").String() != "message_delta" {
			continue
		}
		if got := gjson.GetBytes(chunk, "usage.output_tokens_details.thinking_tokens").Int(); got != 163 {
			t.Fatalf("thinking_tokens = %d, want 163: %s", got, chunk)
		}
		found = true
	}
	if !found {
		t.Fatal("message_delta not emitted")
	}
}

func TestClaudeP4_ReasoningTokensClampedAndRejected(t *testing.T) {
	original := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	cases := []struct {
		name      string
		usage     string
		wantExist bool
		want      int64
	}{
		{"clamps oversized to output", `{"input_tokens":100,"output_tokens":50,"output_tokens_details":{"reasoning_tokens":999}}`, true, 50},
		{"explicit zero preserved", `{"input_tokens":100,"output_tokens":50,"output_tokens_details":{"reasoning_tokens":0}}`, true, 0},
		{"absent omitted", `{"input_tokens":100,"output_tokens":50}`, false, 0},
		{"negative rejected", `{"input_tokens":100,"output_tokens":50,"output_tokens_details":{"reasoning_tokens":-5}}`, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var param any
			chunks := ConvertCodexResponseToClaude(context.Background(), "m", original, nil, claudeP4TerminalResponse(tc.usage), &param)
			seen := false
			for _, chunk := range chunks {
				if gjson.GetBytes(chunk, "type").String() != "message_delta" {
					continue
				}
				seen = true
				v := gjson.GetBytes(chunk, "usage.output_tokens_details.thinking_tokens")
				if tc.wantExist && (!v.Exists() || v.Int() != tc.want) {
					t.Fatalf("thinking_tokens = %v, want %d: %s", v.Raw, tc.want, chunk)
				}
				if !tc.wantExist && v.Exists() {
					t.Fatalf("thinking_tokens must be absent, got %v: %s", v.Raw, chunk)
				}
			}
			if !seen {
				t.Fatal("message_delta not emitted")
			}
		})
	}
}

func TestClaudeP4_ReasoningTokensNonStream(t *testing.T) {
	original := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	raw := []byte(`{"type":"response.completed","response":{"id":"resp_1","model":"gpt-5","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":50,"output_tokens_details":{"reasoning_tokens":20}}}}`)
	out := ConvertCodexResponseToClaudeNonStream(context.Background(), "m", original, nil, raw, nil)
	if got := gjson.GetBytes(out, "usage.output_tokens_details.thinking_tokens").Int(); got != 20 {
		t.Fatalf("non-stream thinking_tokens = %d, want 20: %s", got, out)
	}
}
