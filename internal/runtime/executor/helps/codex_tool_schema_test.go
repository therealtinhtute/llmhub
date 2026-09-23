package helps

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNormalizeCodexToolSchemas_StripsUnsupportedUnicodePropertyEscapePattern(t *testing.T) {
	// Python's built-in re module (and validators relying on it) fail on \p{...}
	// with "bad escape \p", so the pattern must come off before upstream submission.
	input := []byte(`{
		"model": "gpt-5.6",
		"tools": [{
			"type": "function",
			"name": "Artifact",
			"parameters": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"pattern": "^\\p{L}+$",
						"minLength": 1
					}
				}
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	params := gjson.GetBytes(out, "tools.0.parameters")

	if params.Get("properties.name.pattern").Exists() {
		t.Errorf("expected properties.name.pattern removed, got: %s",
			params.Get("properties.name.pattern").Raw)
	}
	if got := params.Get("properties.name.minLength").String(); got != "1" {
		t.Errorf("expected minLength preserved, got %q", got)
	}
}

func TestNormalizeCodexToolSchemas_StripsOctalNULPatternEscape(t *testing.T) {
	// Claude Code's built-in Artifact tool guards file paths with the octal NUL
	// escape, e.g. "^[^\\0]*$" on the upload_asset file_paths item. Strict upstream
	// validators reject that spelling ("is not a 'regex'") while accepting the
	// equivalent \\x00, so the pattern has to come off before the request goes out.
	// (upstream CLIProxyAPI commit 320100ecf767, issue #6025)
	input := []byte(`{
		"model": "gpt-5.6",
		"tools": [{
			"type": "function",
			"name": "Artifact",
			"parameters": {
				"type": "object",
				"properties": {
					"file_paths": {
						"type": "array",
						"minItems": 1,
						"items": {
							"type": "string",
							"minLength": 1,
							"maxLength": 1024,
							"pattern": "^[^\\0]*$"
						}
					},
					"asset_id": {
						"type": "string",
						"pattern": "^[0-9a-f]{32}$"
					},
					"hex_nul": {
						"type": "string",
						"pattern": "^[^\\x00]*$"
					}
				},
				"required": ["file_paths"]
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)

	params := gjson.GetBytes(out, "tools.0.parameters")

	// The octal NUL pattern must be removed, while the rest of the item schema stays.
	if params.Get("properties.file_paths.items.pattern").Exists() {
		t.Errorf("expected properties.file_paths.items.pattern to be removed, got: %s",
			params.Get("properties.file_paths.items.pattern").Raw)
	}
	if got := params.Get("properties.file_paths.items.type").String(); got != "string" {
		t.Errorf("expected properties.file_paths.items.type == 'string', got %q", got)
	}
	for path, want := range map[string]string{
		"properties.file_paths.items.minLength": "1",
		"properties.file_paths.items.maxLength": "1024",
		"properties.file_paths.minItems":        "1",
	} {
		if got := params.Get(path).String(); got != want {
			t.Errorf("expected %s == %q, got %q", path, want, got)
		}
	}

	// A plain pattern is valid and must survive.
	if got := params.Get("properties.asset_id.pattern").String(); got != "^[0-9a-f]{32}$" {
		t.Errorf("expected properties.asset_id.pattern preserved, got %q", got)
	}

	// The hex NUL spelling is the one strict validators accept, so it must survive.
	if got := params.Get("properties.hex_nul.pattern").String(); got != `^[^\x00]*$` {
		t.Errorf("expected properties.hex_nul.pattern preserved, got %q", got)
	}

	// Idempotence test
	outAgain := NormalizeCodexToolSchemas(out)
	if string(outAgain) != string(out) {
		t.Errorf("expected NormalizeCodexToolSchemas to be idempotent")
	}
}

func TestNormalizeCodexToolSchemas_StripsPatternInPatternPropertiesKey(t *testing.T) {
	// Regex keys under patternProperties are compiled by strict validators too,
	// so a key carrying an unsupported escape must be dropped with its subschema.
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "Search",
			"parameters": {
				"type": "object",
				"patternProperties": {
					"^\\p{L}+$": {"type": "string"},
					"^[a-z]+$":  {"type": "number"}
				}
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	pp := gjson.GetBytes(out, "tools.0.parameters.patternProperties")

	if pp.Get(`^\p{L}+$`).Exists() {
		t.Errorf("expected invalid patternProperties key removed, got: %s", pp.Raw)
	}
	if !pp.Get(`^[a-z]+$`).Exists() {
		t.Errorf("expected valid patternProperties key preserved, got: %s", pp.Raw)
	}
}

func TestNormalizeCodexToolSchemas_PreservesNonSchemaPatternKeys(t *testing.T) {
	// A 'pattern' key inside user data (default, enum, description-adjacent
	// structures) is not a schema attribute and must NOT be touched — the walk
	// only descends through known JSON Schema keywords.
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "Widget",
			"parameters": {
				"type": "object",
				"properties": {
					"cfg": {
						"type": "object",
						"default": {"pattern": "^[^\\0]*$"},
						"description": "user data, not schema"
					}
				}
			}
		}],
		"metadata": {"pattern": "^[^\\0]*$"}
	}`)

	out := NormalizeCodexToolSchemas(input)

	if got := gjson.GetBytes(out, `tools.0.parameters.properties.cfg.default.pattern`).String(); got != `^[^\0]*$` {
		t.Errorf("expected default.pattern user data preserved, got %q", got)
	}
	if got := gjson.GetBytes(out, `metadata.pattern`).String(); got != `^[^\0]*$` {
		t.Errorf("expected metadata.pattern outside tools preserved, got %q", got)
	}
}

func TestNormalizeCodexToolSchemas_StripsNestedAndNamespacePatterns(t *testing.T) {
	// Nested combinators ($defs + allOf) and namespace tools' nested tool lists
	// must both be reached by the recursive walk.
	input := []byte(`{
		"tools": [{
			"type": "namespace",
			"name": "multi",
			"tools": [{
				"type": "function",
				"name": "Inner",
				"parameters": {
					"type": "object",
					"$defs": {
						"leaf": {"type": "string", "pattern": "^\\0+$"}
					},
					"allOf": [{"$ref": "#/$defs/leaf"}]
				}
			}]
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	leaf := gjson.GetBytes(out, "tools.0.tools.0.parameters.$defs.leaf")

	if leaf.Get("pattern").Exists() {
		t.Errorf("expected nested $defs pattern removed, got: %s", leaf.Raw)
	}
	if got := leaf.Get("type").String(); got != "string" {
		t.Errorf("expected leaf type preserved, got %q", got)
	}
}

func TestNormalizeCodexToolSchemas_PassthroughWhenNoCandidateEscape(t *testing.T) {
	input := []byte(`{"tools":[{"type":"function","name":"f","parameters":{"type":"object","properties":{"a":{"type":"string","pattern":"^ok$"}}}}]}`)
	out := NormalizeCodexToolSchemas(input)
	if string(out) != string(input) {
		t.Errorf("expected byte-identical passthrough, got: %s", out)
	}
}
