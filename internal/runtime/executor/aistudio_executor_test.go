package executor

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestAIStudioTranslateRequestPrependsLeadingUser(t *testing.T) {
	executor := NewAIStudioExecutor(&config.Config{}, "aistudio", nil)

	tests := []struct {
		name   string
		stream bool
		action string
	}{
		{name: "normal", action: "generateContent"},
		{name: "stream", stream: true, action: "streamGenerateContent"},
		{name: "count tokens", action: "countTokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := cliproxyexecutor.Request{
				Model:   "gemini-3.7-flash",
				Payload: []byte(`{"contents":[{"role":"model","parts":[{"text":"prior output"}]},{"role":"user","parts":[{"text":"continue"}]}]}`),
			}
			if tt.action == "countTokens" {
				req.Metadata = map[string]any{"action": "countTokens"}
			}
			_, body, err := executor.translateRequest(req, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatGemini}, tt.stream)
			if err != nil {
				t.Fatalf("translateRequest() error = %v", err)
			}
			contents := gjson.GetBytes(body.payload, "contents").Array()
			if len(contents) != 3 || contents[0].Get("role").String() != "user" || contents[1].Get("role").String() != "model" {
				t.Fatalf("leading contents malformed: %s", body.payload)
			}
			if text := contents[0].Get("parts.0.text"); !text.Exists() || text.String() != "" {
				t.Fatalf("synthetic leading user missing: %s", body.payload)
			}
			if body.action != tt.action {
				t.Fatalf("action = %q, want %q", body.action, tt.action)
			}
		})
	}
}
