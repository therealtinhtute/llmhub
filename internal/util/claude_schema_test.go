package util

import (
	"encoding/json"
	"testing"
)

func TestNormalizeClaudeToolInputSchemaFlattensObjectUnion(t *testing.T) {
	input := []byte(`{
		"anyOf": [
			{"type":"null"},
			{"type":"object","properties":{"query":{"type":"string"}}}
		],
		"description":"lookup input"
	}`)

	got := NormalizeClaudeToolInputSchema(input)
	var out map[string]any
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("NormalizeClaudeToolInputSchema() returned invalid JSON: %v", err)
	}
	if out["type"] != "object" {
		t.Fatalf("type = %v, want object; raw=%s", out["type"], got)
	}
	if _, exists := out["anyOf"]; exists {
		t.Fatalf("anyOf was not removed: %s", got)
	}
	properties, ok := out["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong type: %s", got)
	}
	if _, exists := properties["query"]; !exists {
		t.Fatalf("query property missing after normalization: %s", got)
	}
}

func TestNormalizeClaudeToolInputSchemaDefaultsInvalidInput(t *testing.T) {
	got := string(NormalizeClaudeToolInputSchema([]byte(`not-json`)))
	if got != emptyClaudeToolInputSchema {
		t.Fatalf("invalid schema normalized to %s, want %s", got, emptyClaudeToolInputSchema)
	}
}

func TestHasUnsupportedUnicodePropertyEscape(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{name: "Plain regex", pattern: `^[0-9a-f]{32}$`, want: false},
		{name: "Unicode property lowercase", pattern: `^\p{L}+$`, want: true},
		{name: "Unicode property uppercase", pattern: `^\P{L}+$`, want: true},
		{name: "Escaped backslash before p is literal", pattern: `^\\p{L}$`, want: false},
		{name: "Backreference", pattern: `^(a)\1$`, want: false},
		{name: "Octal NUL escape rejected by strict validators", pattern: `^[^\0]*$`, want: true},
		{name: "Hex NUL escape is the accepted spelling", pattern: `^[^\x00]*$`, want: false},
		{name: "Escaped backslash before zero is literal and safe", pattern: `^\\0$`, want: false},
		{name: "Trailing backslash", pattern: `foo\`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasUnsupportedUnicodePropertyEscape(tt.pattern); got != tt.want {
				t.Errorf("HasUnsupportedUnicodePropertyEscape(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}
