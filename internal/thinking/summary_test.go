package thinking

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestExtractSummaryConfig(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		body       string
		wantMode   SummaryMode
		wantDetail string
	}{
		{
			name:       "openai effort implies auto summary",
			format:     "openai",
			body:       `{"reasoning_effort":"high"}`,
			wantMode:   SummaryEnabled,
			wantDetail: "auto",
		},
		{
			name:     "responses effort alone is not summary",
			format:   "openai-response",
			body:     `{"reasoning":{"effort":"high"}}`,
			wantMode: SummaryUnspecified,
		},
		{
			name:       "responses summary preserves detail",
			format:     "openai-response",
			body:       `{"reasoning":{"effort":"high","summary":"detailed"}}`,
			wantMode:   SummaryEnabled,
			wantDetail: "detailed",
		},
		{
			name:     "responses null disables summary",
			format:   "codex",
			body:     `{"reasoning":{"summary":null}}`,
			wantMode: SummaryDisabled,
		},
		{
			name:       "gemini includeThoughts enables summary",
			format:     "gemini",
			body:       `{"generationConfig":{"thinkingConfig":{"includeThoughts":true}}}`,
			wantMode:   SummaryEnabled,
			wantDetail: "auto",
		},
		{
			name:     "claude display ignored without active thinking",
			format:   "claude",
			body:     `{"thinking":{"display":"summarized"}}`,
			wantMode: SummaryUnspecified,
		},
		{
			name:       "claude active display enables summary",
			format:     "claude",
			body:       `{"thinking":{"type":"adaptive","display":"summarized"}}`,
			wantMode:   SummaryEnabled,
			wantDetail: "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSummaryConfig([]byte(tt.body), tt.format)
			if got.Mode != tt.wantMode || got.Detail != tt.wantDetail {
				t.Fatalf("ExtractSummaryConfig() = {%v %q}, want {%v %q}", got.Mode, got.Detail, tt.wantMode, tt.wantDetail)
			}
		})
	}
}

func TestApplySummaryConfigForModel(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		model      string
		body       string
		config     SummaryConfig
		path       string
		want       string
		wantExists bool
	}{
		{
			name:       "enabled responses summary activates Claude display",
			format:     "claude",
			model:      "claude-opus-5",
			body:       `{"model":"claude-opus-5","max_tokens":32000}`,
			config:     SummaryConfig{Mode: SummaryEnabled, Detail: "auto"},
			path:       "thinking.display",
			want:       "summarized",
			wantExists: true,
		},
		{
			name:       "disabled Claude summary does not activate thinking",
			format:     "claude",
			model:      "claude-opus-5",
			body:       `{"model":"claude-opus-5","max_tokens":32000}`,
			config:     SummaryConfig{Mode: SummaryDisabled},
			path:       "thinking",
			wantExists: false,
		},
		{
			name:       "disabled Gemini summary writes includeThoughts false",
			format:     "gemini",
			body:       `{"generationConfig":{"thinkingConfig":{"thinkingLevel":"high"}}}`,
			config:     SummaryConfig{Mode: SummaryDisabled},
			path:       "generationConfig.thinkingConfig.includeThoughts",
			want:       "false",
			wantExists: true,
		},
		{
			name:       "enabled Responses summary normalizes detail",
			format:     "openai-response",
			body:       `{"reasoning":{"effort":"high","generate_summary":"concise"}}`,
			config:     SummaryConfig{Mode: SummaryEnabled, Detail: "detailed"},
			path:       "reasoning.summary",
			want:       "detailed",
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := ApplySummaryConfigForModel([]byte(tt.body), tt.format, tt.model, tt.config)
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
