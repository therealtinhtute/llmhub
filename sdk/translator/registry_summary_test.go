package translator

import (
	"bytes"
	"testing"

	"github.com/tidwall/gjson"
)

func TestRegistryTranslateRequestAppliesSummaryIntent(t *testing.T) {
	tests := []struct {
		name       string
		from       Format
		to         Format
		input      string
		translated string
		path       string
		want       string
		wantExists bool
	}{
		{
			name:       "chat effort enables Claude summary",
			from:       FormatOpenAI,
			to:         FormatClaude,
			input:      `{"reasoning_effort":"high"}`,
			translated: `{"thinking":{"type":"adaptive"}}`,
			path:       "thinking.display",
			want:       "summarized",
			wantExists: true,
		},
		{
			name:       "responses effort alone leaves Claude display absent",
			from:       FormatOpenAIResponse,
			to:         FormatClaude,
			input:      `{"reasoning":{"effort":"high"}}`,
			translated: `{"thinking":{"type":"adaptive"}}`,
			path:       "thinking.display",
		},
		{
			name:       "responses summary enables Claude summary",
			from:       FormatOpenAIResponse,
			to:         FormatClaude,
			input:      `{"reasoning":{"effort":"high","summary":"auto"}}`,
			translated: `{"thinking":{"type":"adaptive"}}`,
			path:       "thinking.display",
			want:       "summarized",
			wantExists: true,
		},
		{
			name:       "responses null summary disables Gemini summaries",
			from:       FormatOpenAIResponse,
			to:         FormatGemini,
			input:      `{"reasoning":{"effort":"high","summary":null}}`,
			translated: `{"generationConfig":{"thinkingConfig":{"thinkingLevel":"high"}}}`,
			path:       "generationConfig.thinkingConfig.includeThoughts",
			want:       "false",
			wantExists: true,
		},
		{
			name:       "Google Chat extension overrides effort",
			from:       FormatOpenAI,
			to:         FormatGemini,
			input:      `{"reasoning_effort":"high","extra_body":{"google":{"thinking_config":{"include_thoughts":false}}}}`,
			translated: `{"generationConfig":{"thinkingConfig":{"thinkingLevel":"high","includeThoughts":true}}}`,
			path:       "generationConfig.thinkingConfig.includeThoughts",
			want:       "false",
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(tt.from, tt.to, func(_ string, _ []byte, _ bool) []byte {
				return []byte(tt.translated)
			}, ResponseTransform{})
			out := registry.TranslateRequest(tt.from, tt.to, "model", []byte(tt.input), false)
			got := gjson.GetBytes(out, tt.path)
			if got.Exists() != tt.wantExists {
				t.Fatalf("%s exists = %v, want %v; body=%s", tt.path, got.Exists(), tt.wantExists, out)
			}
			if tt.wantExists && got.String() != tt.want {
				t.Fatalf("%s = %q, want %q; body=%s", tt.path, got.String(), tt.want, out)
			}
		})
	}
}

func TestRegistryTranslateRequestActivatesClaudeForEnabledSummary(t *testing.T) {
	registry := NewRegistry()
	registry.Register(FormatOpenAIResponse, FormatClaude, func(_ string, _ []byte, _ bool) []byte {
		return []byte(`{"model":"claude-opus-5","max_tokens":32000}`)
	}, ResponseTransform{})
	out := registry.TranslateRequest(
		FormatOpenAIResponse,
		FormatClaude,
		"claude-opus-5",
		[]byte(`{"reasoning":{"summary":"auto"},"input":"hi"}`),
		false,
	)
	if got := gjson.GetBytes(out, "thinking.type").String(); got != "adaptive" {
		t.Fatalf("thinking.type = %q, want adaptive; body=%s", got, out)
	}
	if got := gjson.GetBytes(out, "thinking.display").String(); got != "summarized" {
		t.Fatalf("thinking.display = %q, want summarized; body=%s", got, out)
	}
}

func TestRegistryTranslateRequestDoesNotActivateClaudeForDisabledSummary(t *testing.T) {
	registry := NewRegistry()
	registry.Register(FormatOpenAIResponse, FormatClaude, func(_ string, _ []byte, _ bool) []byte {
		return []byte(`{"model":"claude-opus-5","max_tokens":32000}`)
	}, ResponseTransform{})
	out := registry.TranslateRequest(
		FormatOpenAIResponse,
		FormatClaude,
		"claude-opus-5",
		[]byte(`{"reasoning":{"summary":null},"input":"hi"}`),
		false,
	)
	if gjson.GetBytes(out, "thinking").Exists() {
		t.Fatalf("disabled summary activated Claude thinking: %s", out)
	}
}

func TestRegistryTranslateRequestPreservesNativeClaudeMissingDisplay(t *testing.T) {
	registry := NewRegistry()
	body := []byte(`{"model":"claude-opus-5","thinking":{"type":"adaptive"}}`)
	out := registry.TranslateRequest(FormatClaude, FormatClaude, "claude-opus-5", body, true)
	if gjson.GetBytes(out, "thinking.display").Exists() {
		t.Fatalf("native Claude request without display gained one: %s", out)
	}
}

func TestRegistryTranslateRequestDoesNotMixSummaryIntoFallback(t *testing.T) {
	registry := NewRegistry()
	body := []byte(`{"model":"gemini-3.6-flash","reasoning":{"summary":"auto"},"input":"hi"}`)
	out := registry.TranslateRequest(FormatOpenAIResponse, FormatGemini, "gemini-3.6-flash", body, false)
	if !bytes.Equal(out, body) {
		t.Fatalf("missing translator changed fallback body: got %s, want %s", out, body)
	}
	if gjson.GetBytes(out, "generationConfig").Exists() {
		t.Fatalf("missing translator mixed Gemini fields into Responses body: %s", out)
	}
}
