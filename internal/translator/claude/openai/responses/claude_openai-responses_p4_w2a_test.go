package responses

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestClaudeP4_CollapseConsecutiveThinkingKeepsLatest(t *testing.T) {
	raw := []byte(`{
		"model": "claude-opus-5",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hi"}]},
			{"type": "reasoning", "encrypted_content": "sig-first", "summary": [{"type": "summary_text", "text": "first"}]},
			{"type": "reasoning", "encrypted_content": "sig-second", "summary": [{"type": "summary_text", "text": "second"}]},
			{"type": "function_call", "call_id": "call_1", "name": "lookup", "arguments": "{}"},
			{"type": "function_call_output", "call_id": "call_1", "output": "done"}
		]
	}`)
	out := ConvertOpenAIResponsesRequestToClaude("claude-opus-5", raw, false)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 3 {
		t.Fatalf("messages = %d, want 3 (user, assistant, tool_result): %s", len(msgs), out)
	}
	content := msgs[1].Get("content").Array()
	if len(content) != 2 {
		t.Fatalf("assistant blocks = %d, want 2 (thinking+tool_use): %s", len(content), out)
	}
	if got := content[0].Get("type").String(); got != "thinking" {
		t.Fatalf("block 0 = %q, want thinking: %s", got, out)
	}
	if got := content[0].Get("thinking").String(); got != "second" {
		t.Fatalf("thinking text = %q, want latest 'second': %s", got, out)
	}
	if got := content[1].Get("type").String(); got != "tool_use" {
		t.Fatalf("block 1 = %q, want tool_use: %s", got, out)
	}
}

func TestClaudeP4_StripTrailingThinkingDropsAssistant(t *testing.T) {
	raw := []byte(`{
		"model": "claude-haiku-4-5-20251001",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hello"}]},
			{"type": "reasoning", "encrypted_content": "sig-only", "summary": [{"type": "summary_text", "text": "thought"}]}
		]
	}`)
	out := ConvertOpenAIResponsesRequestToClaude("claude-haiku-4-5-20251001", raw, false)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1 after stripping trailing thinking: %s", len(msgs), out)
	}
	if got := msgs[0].Get("role").String(); got != "user" {
		t.Fatalf("role = %q, want user: %s", got, out)
	}
}

func TestClaudeP4_UnsupportedPrefillDroppedForFableOpus(t *testing.T) {
	for _, model := range []string{"claude-fable-5-1", "claude-opus-5", "claude-sonnet-4-6"} {
		raw := []byte(`{
			"model": "` + model + `",
			"input": [
				{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hello"}]},
				{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "prefill text"}]}
			]
		}`)
		out := ConvertOpenAIResponsesRequestToClaude(model, raw, false)
		msgs := gjson.GetBytes(out, "messages").Array()
		if len(msgs) != 1 {
			t.Fatalf("model %s: messages = %d, want 1 after prefill drop: %s", model, len(msgs), out)
		}
	}
}

func TestClaudeP4_SupportedPrefillPreservedForSonnet45(t *testing.T) {
	raw := []byte(`{
		"model": "claude-sonnet-4-5",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hello"}]},
			{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "prefill text"}]}
		]
	}`)
	out := ConvertOpenAIResponsesRequestToClaude("claude-sonnet-4-5", raw, false)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2 preserving prefill: %s", len(msgs), out)
	}
}

func TestClaudeP4_TrailingToolUsePreservedForRejectingModels(t *testing.T) {
	raw := []byte(`{
		"model": "claude-opus-5",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hi"}]},
			{"type": "function_call", "call_id": "call_9", "name": "lookup", "arguments": "{}"}
		]
	}`)
	out := ConvertOpenAIResponsesRequestToClaude("claude-opus-5", raw, false)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 3 {
		t.Fatalf("messages = %d, want 3 (tool_use + synthesized error tool_result): %s", len(msgs), out)
	}
	content := msgs[1].Get("content").Array()
	if len(content) != 1 || content[0].Get("type").String() != "tool_use" {
		t.Fatalf("trailing tool_use lost: %s", out)
	}
	stub := msgs[2].Get("content.0")
	if stub.Get("type").String() != "tool_result" || stub.Get("tool_use_id").String() != "call_9" {
		t.Fatalf("synthesized error tool_result stub missing: %s", out)
	}
}
