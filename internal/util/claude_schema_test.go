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
