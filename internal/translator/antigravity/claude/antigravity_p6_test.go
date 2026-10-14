package claude

import (
	"testing"

	"github.com/tidwall/gjson"
)

// Ports upstream a76da7115 (Claude side): tool_choice none omits request.tools.
func TestConvertClaudeRequestToAntigravity_ToolChoiceNoneOmitsTools(t *testing.T) {
	newInput := func(toolChoice string) []byte {
		return []byte(`{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],"tools":[{"name":"json","input_schema":{"type":"object","properties":{}}}],"tool_choice":` + toolChoice + `}`)
	}
	for _, tc := range []struct {
		name       string
		toolChoice string
	}{
		{"object none", `{"type":"none"}`},
		{"string none", `"none"`},
		{"string NONE case-insensitive", `"NONE"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := ConvertClaudeRequestToAntigravity("gemini-3-flash", newInput(tc.toolChoice), false)
			if gjson.GetBytes(out, "request.tools").Exists() {
				t.Fatalf("request.tools should be omitted for tool_choice %s: %s", tc.toolChoice, out)
			}
			if got := gjson.GetBytes(out, "request.toolConfig.functionCallingConfig.mode").String(); got != "NONE" {
				t.Fatalf("mode = %q, want NONE: %s", got, out)
			}
		})
	}

	t.Run("specific tool keeps tools", func(t *testing.T) {
		in := []byte(`{"model":"m","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],"tools":[{"name":"json","input_schema":{"type":"object","properties":{}}}],"tool_choice":{"type":"tool","name":"json"}}`)
		out := ConvertClaudeRequestToAntigravity("gemini-3-flash", in, false)
		if !gjson.GetBytes(out, "request.tools").Exists() {
			t.Fatalf("request.tools should be kept for specific tool: %s", out)
		}
	})
}
