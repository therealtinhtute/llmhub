package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

// Ports upstream a76da7115 (OpenAI chat-completions side, NONE-only scope):
// tool_choice none omits request.tools and pins mode NONE.
func TestConvertOpenAIRequestToAntigravity_ToolChoiceNoneOmitsTools(t *testing.T) {
	newInput := func(toolChoice string) []byte {
		return []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{}}}}],"tool_choice":` + toolChoice + `}`)
	}
	for _, tc := range []struct {
		name       string
		toolChoice string
	}{
		{"string none", `"none"`},
		{"object none", `{"type":"none"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := ConvertOpenAIRequestToAntigravity("gemini-3-flash", newInput(tc.toolChoice), false)
			if gjson.GetBytes(out, "request.tools").Exists() {
				t.Fatalf("request.tools should be omitted for tool_choice %s: %s", tc.toolChoice, out)
			}
			if got := gjson.GetBytes(out, "request.toolConfig.functionCallingConfig.mode").String(); got != "NONE" {
				t.Fatalf("mode = %q, want NONE: %s", got, out)
			}
		})
	}

	t.Run("absent tool_choice keeps tools without mode", func(t *testing.T) {
		in := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{}}}}]}`)
		out := ConvertOpenAIRequestToAntigravity("gemini-3-flash", in, false)
		if !gjson.GetBytes(out, "request.tools").Exists() {
			t.Fatalf("request.tools should be kept without tool_choice: %s", out)
		}
		if gjson.GetBytes(out, "request.toolConfig.functionCallingConfig.mode").Exists() {
			t.Fatalf("mode should be absent without tool_choice: %s", out)
		}
	})
}
