package helps

import (
	"net/http"
	"testing"
)

func TestIsClaudeProbeOrHelperRequest_ProbeTokens(t *testing.T) {
	for _, token := range []string{"hi", ".", "probe", "test", "quota", "  Hi  "} {
		body := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":` + quoteForTest(token) + `}]}]}`)
		if !IsClaudeProbeOrHelperRequest(body) {
			t.Fatalf("probe token %q not detected", token)
		}
	}
}

func TestIsClaudeProbeOrHelperRequest_NormalTraffic(t *testing.T) {
	bodies := []string{
		`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"hello world, help me code"}]}]}`,
		`{"model":"claude-opus-5","messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"hello"}]}`,
		`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"hi"},{"type":"text","text":"there"}]}]}`,
		`{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"x","content":"hi"}]}]}`,
	}
	for _, raw := range bodies {
		if IsClaudeProbeOrHelperRequest([]byte(raw)) {
			t.Fatalf("normal traffic misclassified as probe: %s", raw)
		}
	}
}

func TestIsClaudeProbeOrHelperRequest_TitleHelper(t *testing.T) {
	body := []byte(`{"model":"claude-haiku-4-5-20251001","messages":[{"role":"user","content":[{"type":"text","text":"Generate a title for this session please"}]}]}`)
	if !IsClaudeProbeOrHelperRequest(body) {
		t.Fatal("haiku title helper not detected")
	}
	nonHaiku := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"Generate a title for this session please"}]}]}`)
	if IsClaudeProbeOrHelperRequest(nonHaiku) {
		t.Fatal("non-haiku title prompt must not be a helper")
	}
}

func TestIsClaudeSubagentRequest_Headers(t *testing.T) {
	h := http.Header{"X-Claude-Code-Agent-Id": []string{"agent-1"}}
	if !IsClaudeSubagentRequest(h, []byte(`{"model":"claude-opus-5"}`)) {
		t.Fatal("agent header not detected as subagent")
	}
	if IsClaudeSubagentRequest(http.Header{}, []byte(`{"model":"claude-opus-5"}`)) {
		t.Fatal("empty headers misclassified as subagent")
	}
	if !IsClaudeSubagentRequest(nil, []byte(`{"model":"claude-opus-5","metadata":{"is_subagent":true}}`)) {
		t.Fatal("metadata subagent marker not detected")
	}
}

func TestClaudePayloadHas1hTTL(t *testing.T) {
	withTTL := []byte(`{"tools":[{"name":"a","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`)
	if !ClaudePayloadHas1hTTL(withTTL) {
		t.Fatal("1h TTL not detected")
	}
	withoutTTL := []byte(`{"tools":[{"name":"a","cache_control":{"type":"ephemeral"}}]}`)
	if ClaudePayloadHas1hTTL(withoutTTL) {
		t.Fatal("ephemeral without ttl misreported as 1h")
	}
	if ClaudePayloadHas1hTTL([]byte(`{"model":"x"}`)) {
		t.Fatal("payload without cache blocks misreported as 1h")
	}
}

func quoteForTest(s string) string {
	out := `"`
	for _, r := range s {
		switch r {
		case '"':
			out += `\"`
		case '\\':
			out += `\\`
		default:
			out += string(r)
		}
	}
	return out + `"`
}
