package responses

import (
	"testing"

	"github.com/tidwall/gjson"
)

// Ports upstream e92f6cf55 as refined by 6dea3dfa1: Responses reasoning effort
// without an explicit summary enables Antigravity includeThoughts; an explicit
// summary (including null/none) is honored and never overridden.
func TestConvertOpenAIResponsesRequestToAntigravity_ReasoningEffortEnablesSummary(t *testing.T) {
	t.Run("effort alone enables includeThoughts", func(t *testing.T) {
		in := []byte(`{"model":"gemini-3-flash","reasoning":{"effort":"high"},"input":"hello"}`)
		out := ConvertOpenAIResponsesRequestToAntigravity("gemini-3-flash", in, false)
		got := gjson.GetBytes(out, "request.generationConfig.thinkingConfig.includeThoughts")
		if !got.Exists() || !got.Bool() {
			t.Fatalf("includeThoughts = %q exists=%v, want true: %s", got.Raw, got.Exists(), out)
		}
	})

	t.Run("explicit null summary disables includeThoughts", func(t *testing.T) {
		in := []byte(`{"model":"gemini-3-flash","reasoning":{"effort":"high","summary":null},"input":"hello"}`)
		out := ConvertOpenAIResponsesRequestToAntigravity("gemini-3-flash", in, false)
		got := gjson.GetBytes(out, "request.generationConfig.thinkingConfig.includeThoughts")
		if !got.Exists() || got.Bool() {
			t.Fatalf("includeThoughts = %q exists=%v, want false: %s", got.Raw, got.Exists(), out)
		}
	})

	t.Run("explicit auto summary enables includeThoughts", func(t *testing.T) {
		in := []byte(`{"model":"gemini-3-flash","reasoning":{"effort":"high","summary":"auto"},"input":"hello"}`)
		out := ConvertOpenAIResponsesRequestToAntigravity("gemini-3-flash", in, false)
		got := gjson.GetBytes(out, "request.generationConfig.thinkingConfig.includeThoughts")
		if !got.Exists() || !got.Bool() {
			t.Fatalf("includeThoughts = %q exists=%v, want true: %s", got.Raw, got.Exists(), out)
		}
	})
}
