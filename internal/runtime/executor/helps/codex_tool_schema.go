package helps

// Codex tool-schema pattern normalization, ported from upstream CLIProxyAPI
// commit 320100ecf767 ("fix(codex): strip octal NUL pattern escapes from tool
// schemas", issue #6025), which extends the earlier Unicode-property-escape
// strip (e56abd56f142, 37ce368c5002).
//
// Scope note: upstream's file also carries the union-simplification and
// strict-shape machinery from bf20b999decb/56518489ce92; those parts are
// deferred (see the parity plan's perf/normalization exclusions). This file
// implements only the schema-aware pattern strip.

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/util"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// NormalizeCodexToolSchemas inspects function tools in a Codex request payload
// and strips regex `pattern` attributes that strict upstream validators reject.
func NormalizeCodexToolSchemas(body []byte) []byte {
	updatedTools, changed := normalizeCodexToolList(gjson.GetBytes(body, "tools"))
	if !changed {
		return body
	}
	out, errSet := sjson.SetRawBytes(body, "tools", updatedTools)
	if errSet != nil {
		return body
	}
	log.Debugf("codex: normalized tool schemas to prevent upstream failure")
	return out
}

// normalizeCodexToolList batches changed elements so a long request (or a wide
// namespace) is copied once rather than once per tool. Copy the gaps verbatim
// to retain array formatting and unchanged declarations.
func normalizeCodexToolList(tools gjson.Result) ([]byte, bool) {
	if !tools.IsArray() {
		return nil, false
	}
	var out []byte
	offset := 0
	tools.ForEach(func(_, tool gjson.Result) bool {
		updated, changed := normalizeCodexTool(tool)
		if !changed {
			return true
		}
		if out == nil {
			out = make([]byte, 0, len(tools.Raw))
		}
		// ForEach indexes share the source of the containing array's index.
		start := tool.Index - tools.Index
		out = append(out, tools.Raw[offset:start]...)
		out = append(out, updated...)
		offset = start + len(tool.Raw)
		return true
	})
	if out == nil {
		return nil, false
	}
	return append(out, tools.Raw[offset:]...), true
}

func normalizeCodexTool(tool gjson.Result) ([]byte, bool) {
	toolType := tool.Get("type").String()
	// Handle namespace tools (e.g. multi-agent nested tools)
	if toolType == "namespace" {
		updatedTools, changed := normalizeCodexToolList(tool.Get("tools"))
		if !changed {
			return nil, false
		}
		updated, errSet := sjson.SetRawBytes([]byte(tool.Raw), "tools", updatedTools)
		return updated, errSet == nil
	}

	if toolType != "function" && toolType != "custom" {
		return nil, false
	}

	params := tool.Get("parameters")
	if !params.Exists() || !params.IsObject() {
		return nil, false
	}

	rawTool := []byte(tool.Raw)
	updatedParams, paramsChanged := normalizeCodexParameters(params)
	if !paramsChanged {
		return nil, false
	}

	updatedTool, errSet := sjson.SetRawBytes(rawTool, "parameters", updatedParams)
	if errSet != nil {
		return nil, false
	}

	log.Debugf("codex: normalized schema for tool %s to avoid upstream abort", tool.Get("name").String())
	return updatedTool, true
}

func normalizeCodexParameters(params gjson.Result) ([]byte, bool) {
	return stripIncompatiblePatternsFromJSON([]byte(params.Raw))
}

// stripIncompatiblePatternsFromJSON recursively removes pattern attributes that strict
// upstream validators reject: unsupported Unicode property escapes (\p{...} / \P{...})
// and the octal NUL escape (\0). See util.HasUnsupportedUnicodePropertyEscape for the
// predicate and the upstream errors behind each one.
// It is schema-aware: only subschemas under known JSON Schema keyword locations are visited,
// preventing accidental deletion of 'pattern' keys inside user data (e.g. description, default, enum).
func stripIncompatiblePatternsFromJSON(raw []byte) ([]byte, bool) {
	rawStr := string(raw)
	// The fast path avoids parsing when no candidate escape is present. Patterns
	// arrive JSON-escaped, so a literal backslash before '0' is `\\0` in the raw bytes.
	if !strings.Contains(rawStr, `\p{`) && !strings.Contains(rawStr, `\P{`) && !strings.Contains(rawStr, `\u`) &&
		!strings.Contains(rawStr, `\\0`) {
		return raw, false
	}
	var root any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil || root == nil {
		return raw, false
	}
	// Verify no trailing garbage
	var dummy any
	if err := dec.Decode(&dummy); err != io.EOF {
		return raw, false
	}
	if !stripIncompatiblePatterns(root) {
		return raw, false
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(root); err != nil {
		return raw, false
	}
	return bytes.TrimSpace(buf.Bytes()), true
}

func stripIncompatiblePatterns(v any) bool {
	changed := false
	switch schema := v.(type) {
	case map[string]any:
		if patternVal, ok := schema["pattern"].(string); ok && util.HasUnsupportedUnicodePropertyEscape(patternVal) {
			delete(schema, "pattern")
			changed = true
		}

		// Inspect regex keys under patternProperties
		if patternProps, ok := schema["patternProperties"].(map[string]any); ok {
			for patternKey, subSchema := range patternProps {
				if util.HasUnsupportedUnicodePropertyEscape(patternKey) {
					delete(patternProps, patternKey)
					changed = true
				} else if stripIncompatiblePatterns(subSchema) {
					changed = true
				}
			}
		}

		for _, mapKey := range util.SchemaMapKeywords {
			if mapKey == "patternProperties" {
				continue
			}
			if subMap, ok := schema[mapKey].(map[string]any); ok {
				for _, subSchema := range subMap {
					if stripIncompatiblePatterns(subSchema) {
						changed = true
					}
				}
			}
		}

		for _, valKey := range util.SchemaValueKeywords {
			if val, exists := schema[valKey]; exists {
				switch sub := val.(type) {
				case map[string]any:
					if stripIncompatiblePatterns(sub) {
						changed = true
					}
				case []any:
					for _, item := range sub {
						if stripIncompatiblePatterns(item) {
							changed = true
						}
					}
				}
			}
		}
	case []any:
		for _, item := range schema {
			if stripIncompatiblePatterns(item) {
				changed = true
			}
		}
	}
	return changed
}
