package helps

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestEnsureGeminiLeadingUserContent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		path      string
		wantRoles string
		wantEmpty bool
		wantSame  bool
	}{
		{
			name:      "user first is unchanged",
			input:     `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
			path:      "contents",
			wantRoles: "user",
			wantSame:  true,
		},
		{
			name:      "model first is prepended",
			input:     `{"contents":[{"role":"model","parts":[{"text":"answer"}]},{"role":"user","parts":[{"text":"continue"}]}]}`,
			path:      "contents",
			wantRoles: "user,model,user",
			wantEmpty: true,
		},
		{
			name:      "nested contents are normalized",
			input:     `{"request":{"contents":[{"role":"model","parts":[{"text":"answer"}]}]}}`,
			path:      "request.contents",
			wantRoles: "user,model",
			wantEmpty: true,
		},
		{
			name:     "empty contents are unchanged",
			input:    `{"contents":[]}`,
			path:     "contents",
			wantSame: true,
		},
		{
			name:     "missing contents are unchanged",
			input:    `{"model":"test"}`,
			path:     "contents",
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.input)
			output := EnsureGeminiLeadingUserContent(input, tt.path)
			contents := gjson.GetBytes(output, tt.path).Array()
			roles := make([]string, 0, len(contents))
			for _, content := range contents {
				roles = append(roles, content.Get("role").String())
			}
			if got := strings.Join(roles, ","); got != tt.wantRoles {
				t.Fatalf("roles = %q, want %q; output=%s", got, tt.wantRoles, output)
			}
			if tt.wantEmpty {
				leadingText := gjson.GetBytes(output, tt.path+".0.parts.0.text")
				if !leadingText.Exists() || leadingText.String() != "" {
					t.Fatalf("leading empty user missing; output=%s", output)
				}
			}
			if tt.wantSame && &output[0] != &input[0] {
				t.Fatal("unchanged payload should reuse the input bytes")
			}
		})
	}
}

func TestEnsureGeminiLeadingUserContentIsIdempotent(t *testing.T) {
	input := []byte(`{"contents":[{"role":"model","parts":[{"text":"answer"}]}]}`)
	first := EnsureGeminiLeadingUserContent(input, "contents")
	second := EnsureGeminiLeadingUserContent(first, "contents")

	contents := gjson.GetBytes(second, "contents").Array()
	if len(contents) != 2 {
		t.Fatalf("contents length = %d, want 2; output=%s", len(contents), second)
	}
	if contents[0].Get("role").String() != "user" || contents[1].Get("role").String() != "model" {
		t.Fatalf("roles changed on repeated normalization: %s", second)
	}
}
