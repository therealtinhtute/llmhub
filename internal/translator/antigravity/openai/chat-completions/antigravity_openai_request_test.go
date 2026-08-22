package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIRequestToAntigravityTranslatesVideoURL(t *testing.T) {
	input := []byte(`{
		"model": "gemini-3.7-flash-high",
		"messages": [{
			"role": "user",
			"content": [
				{"type": "text", "text": "Name the colours in order"},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,AAAA"}},
				{"type": "video_url", "video_url": {"url": "data:video/mp4;base64,AAAAIGZ0eXBtcDQy"}}
			]
		}]
	}`)

	output := ConvertOpenAIRequestToAntigravity("gemini-3.7-flash-high", input, false)
	parts := gjson.GetBytes(output, "request.contents.0.parts").Array()
	if len(parts) != 3 {
		t.Fatalf("parts length = %d, want 3. Output: %s", len(parts), output)
	}

	if got := parts[0].Get("text").String(); got != "Name the colours in order" {
		t.Fatalf("parts[0].text = %q, want text content", got)
	}
	if got := parts[1].Get("inlineData.mimeType").String(); got != "image/png" {
		t.Fatalf("parts[1].inlineData.mimeType = %q, want image/png", got)
	}
	if got := parts[2].Get("inlineData.mimeType").String(); got != "video/mp4" {
		t.Fatalf("parts[2].inlineData.mimeType = %q, want video/mp4", got)
	}
	if got := parts[2].Get("inlineData.data").String(); got != "AAAAIGZ0eXBtcDQy" {
		t.Fatalf("parts[2].inlineData.data = %q, want video payload", got)
	}
	if parts[2].Get("thoughtSignature").Exists() {
		t.Fatal("video part should not receive an image thought signature")
	}
}

func TestConvertOpenAIRequestToAntigravity_MaxCompletionTokens(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected float64
	}{
		{
			name:     "only max_tokens",
			body:     `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"max_tokens":100}`,
			expected: 100,
		},
		{
			name:     "only max_completion_tokens",
			body:     `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"max_completion_tokens":200}`,
			expected: 200,
		},
		{
			name:     "max_tokens preferred over max_completion_tokens",
			body:     `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"max_tokens":100,"max_completion_tokens":200}`,
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := ConvertOpenAIRequestToAntigravity("gemini-2.5-flash", []byte(tt.body), false)
			got := gjson.GetBytes(out, "request.generationConfig.maxOutputTokens")
			if !got.Exists() {
				t.Fatalf("request.generationConfig.maxOutputTokens missing. Output: %s", out)
			}
			if got.Float() != tt.expected {
				t.Fatalf("maxOutputTokens = %v, want %v. Output: %s", got.Float(), tt.expected, out)
			}
		})
	}
}
