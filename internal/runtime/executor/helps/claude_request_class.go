package helps

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// IsClaudeProbeOrHelperRequest reports whether a Claude messages payload is a
// lightweight probe or title-helper turn rather than a main interaction query.
// Adapted from upstream d7052c96af78 / 4a5ab534f827 at a reduced scope: only the
// signals needed for beta/TTL/billing decisions are implemented (single short
// text probe, Haiku title helper). Normal multi-turn traffic returns false.
//
// Probe: exactly one user message carrying exactly one non-reminder text block
// whose trimmed content matches a known probe token (quota/test/probe/./Hi).
// Helper: Haiku-family model with a title-generation prompt (session title
// patterns). Both suppress effort/thinking-display betas and 1h cache TTL.
func IsClaudeProbeOrHelperRequest(payload []byte) bool {
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return false
	}
	if isClaudeProbePayload(payload) {
		return true
	}
	return isClaudeTitleHelperPayload(payload)
}

func isClaudeProbePayload(payload []byte) bool {
	messages := gjson.GetBytes(payload, "messages")
	if !messages.IsArray() || len(messages.Array()) != 1 {
		return false
	}
	msg := messages.Array()[0]
	if !strings.EqualFold(strings.TrimSpace(msg.Get("role").String()), "user") {
		return false
	}
	content := msg.Get("content")
	var texts []string
	if content.Type == gjson.String {
		texts = []string{content.String()}
	} else if content.IsArray() {
		for _, part := range content.Array() {
			partType := strings.TrimSpace(part.Get("type").String())
			if partType == "system-reminder" || strings.HasSuffix(partType, "reminder") {
				continue
			}
			if partType != "text" {
				return false
			}
			texts = append(texts, part.Get("text").String())
		}
	} else {
		return false
	}
	if len(texts) != 1 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(texts[0])) {
	case "hi", ".", "probe", "test", "quota":
		return true
	default:
		return false
	}
}

func isClaudeTitleHelperPayload(payload []byte) bool {
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "model").String()))
	if !strings.Contains(model, "haiku") {
		return false
	}
	haystack := strings.ToLower(collectClaudePayloadText(payload))
	if haystack == "" {
		return false
	}
	for _, marker := range []string{
		"generate a title",
		"generate title",
		"session title",
		"summarize this conversation in a title",
		"conversation title",
	} {
		if strings.Contains(haystack, marker) {
			return true
		}
	}
	return false
}

func collectClaudePayloadText(payload []byte) string {
	var sb strings.Builder
	if system := gjson.GetBytes(payload, "system"); system.Exists() {
		if system.Type == gjson.String {
			sb.WriteString(system.String())
			sb.WriteByte('\n')
		} else if system.IsArray() {
			for _, part := range system.Array() {
				if strings.EqualFold(part.Get("type").String(), "text") {
					sb.WriteString(part.Get("text").String())
					sb.WriteByte('\n')
				}
			}
		}
	}
	if messages := gjson.GetBytes(payload, "messages"); messages.IsArray() {
		for _, msg := range messages.Array() {
			content := msg.Get("content")
			if content.Type == gjson.String {
				sb.WriteString(content.String())
				sb.WriteByte('\n')
			} else if content.IsArray() {
				for _, part := range content.Array() {
					if strings.EqualFold(part.Get("type").String(), "text") {
						sb.WriteString(part.Get("text").String())
						sb.WriteByte('\n')
					}
				}
			}
		}
	}
	return sb.String()
}

// IsClaudeSubagentRequest reports whether the request is a subagent/side query
// rather than the main interaction thread. Adapted from upstream d7052c96af78:
// agent identity headers or an explicit subagent marker in metadata/session.
// Subagents omit extended-cache-ttl and 1h TTL and carry cc_is_subagent=true.
func IsClaudeSubagentRequest(headers http.Header, payload []byte) bool {
	if headers != nil {
		for _, name := range []string{
			"X-Claude-Code-Agent-Id",
			"X-Claude-Code-Parent-Agent-Id",
			"X-Claude-Remote-Container-Id",
			"X-Claude-Remote-Session-Id",
		} {
			if strings.TrimSpace(headers.Get(name)) != "" {
				return true
			}
		}
		if strings.EqualFold(strings.TrimSpace(headers.Get("X-Client-App")), "subagent") {
			return true
		}
	}
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return false
	}
	if v := gjson.GetBytes(payload, "metadata.is_subagent"); v.Exists() && v.Bool() {
		return true
	}
	if v := gjson.GetBytes(payload, "session.subagent"); v.Exists() && v.Bool() {
		return true
	}
	return false
}

// ClaudePayloadHas1hTTL reports whether any cache_control block in the payload
// carries ttl=1h. Used to decide extended-cache-ttl beta pairing.
func ClaudePayloadHas1hTTL(payload []byte) bool {
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return false
	}
	found := false
	visit := func(block gjson.Result) {
		if found {
			return
		}
		if cc := block.Get("cache_control"); cc.IsObject() {
			if ttl := cc.Get("ttl"); ttl.Type == gjson.String && strings.EqualFold(strings.TrimSpace(ttl.String()), "1h") {
				found = true
			}
		}
	}
	if tools := gjson.GetBytes(payload, "tools"); tools.IsArray() {
		for _, tool := range tools.Array() {
			visit(tool)
		}
	}
	if !found {
		if system := gjson.GetBytes(payload, "system"); system.IsArray() {
			for _, block := range system.Array() {
				visit(block)
			}
		}
	}
	if !found {
		if messages := gjson.GetBytes(payload, "messages"); messages.IsArray() {
			for _, msg := range messages.Array() {
				if content := msg.Get("content"); content.IsArray() {
					for _, block := range content.Array() {
						visit(block)
					}
				}
			}
		}
	}
	return found
}
