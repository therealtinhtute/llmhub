package thinking

import (
	"bytes"
	"testing"

	"github.com/tidwall/gjson"
)

func TestHasThinkingSuffix(t *testing.T) {
	for _, tc := range []struct {
		model string
		want  bool
	}{
		{model: "model", want: false},
		{model: "model()", want: false},
		{model: "model(invalid)", want: false},
		{model: "model(high)", want: true},
		{model: "model(xhigh)", want: true},
		{model: "model(none)", want: true},
		{model: "model(auto)", want: true},
		{model: "model(16384)", want: true},
	} {
		t.Run(tc.model, func(t *testing.T) {
			if got := HasThinkingSuffix(tc.model); got != tc.want {
				t.Fatalf("HasThinkingSuffix(%q) = %t, want %t", tc.model, got, tc.want)
			}
		})
	}
}

func TestStripConfigurationUpdatesExported(t *testing.T) {
	t.Run("items removed without changing top level effort", func(t *testing.T) {
		body := []byte(`{"reasoning":{"effort":"medium"},"input":[{"type":"configuration_update","reasoning":{"effort":"high"}},{"role":"user","content":"hi"}]}`)
		got := StripConfigurationUpdates(body)
		if input := gjson.GetBytes(got, "input").Raw; input != `[{"role":"user","content":"hi"}]` {
			t.Fatalf("input = %s, want user message; body=%s", input, got)
		}
		if effort := gjson.GetBytes(got, "reasoning.effort").String(); effort != "medium" {
			t.Fatalf("reasoning.effort = %q, want medium; body=%s", effort, got)
		}
	})

	t.Run("input order and unrelated fields are preserved", func(t *testing.T) {
		body := []byte(`{"model":"gpt-6-astra","input":[{"role":"user","content":"first"},{"type":"configuration_update","reasoning":{"effort":"high"}},{"role":"assistant","content":"reply"}]}`)
		got := StripConfigurationUpdates(body)
		if input := gjson.GetBytes(got, "input").Raw; input != `[{"role":"user","content":"first"},{"role":"assistant","content":"reply"}]` {
			t.Fatalf("input = %s, want order preserved; body=%s", input, got)
		}
		if model := gjson.GetBytes(got, "model").String(); model != "gpt-6-astra" {
			t.Fatalf("model = %q, want gpt-6-astra; body=%s", model, got)
		}
	})

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "empty input", body: `{"input":[]}`},
		{name: "invalid JSON", body: `{"input":[`},
		{name: "nonarray input", body: `{"input":{"type":"configuration_update","reasoning":{"effort":"high"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(tc.body)
			if got := StripConfigurationUpdates(body); !bytes.Equal(got, body) {
				t.Fatalf("body changed: got %s, want %s", got, body)
			}
		})
	}
}
