package responses

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	sigcompat "github.com/therealtinhtute/llmhub/internal/signature"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protowire"
)

// claudeAssistantBlockTypes returns the content block types of the last Claude
// assistant message produced by the Responses -> Claude request translator.
func claudeAssistantBlockTypes(t *testing.T, claudeReq []byte) []string {
	t.Helper()
	var kinds []string
	gjson.GetBytes(claudeReq, "messages").ForEach(func(_, m gjson.Result) bool {
		if m.Get("role").String() != "assistant" {
			return true
		}
		kinds = kinds[:0]
		m.Get("content").ForEach(func(_, b gjson.Result) bool {
			kinds = append(kinds, b.Get("type").String())
			return true
		})
		return true
	})
	return kinds
}

func responsesRequestFromItems(items ...string) []byte {
	raw := `{"model":"claude-test","input":[`
	for i, item := range items {
		if i > 0 {
			raw += ","
		}
		raw += item
	}
	return []byte(raw + `]}`)
}
func translateClaudeResponsesStreamThroughRegistry(chunks [][]byte) [][]byte {
	var param any
	var outputs [][]byte
	for _, chunk := range chunks {
		outputs = append(outputs, ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-test", nil, nil, chunk, &param)...)
	}
	return outputs
}

func testClaudeResponsesThinkingSignature(t *testing.T) (string, string) {
	t.Helper()
	return testClaudeResponsesThinkingSignatureForModel(t, "claude-sonnet-4-6")
}

func testClaudeResponsesThinkingSignatureForModel(t *testing.T, model string) (string, string) {
	t.Helper()
	channelBlock := []byte{}
	channelBlock = protowire.AppendTag(channelBlock, 1, protowire.VarintType)
	channelBlock = protowire.AppendVarint(channelBlock, 12)
	channelBlock = protowire.AppendTag(channelBlock, 2, protowire.VarintType)
	channelBlock = protowire.AppendVarint(channelBlock, 2)
	channelBlock = protowire.AppendTag(channelBlock, 6, protowire.BytesType)
	channelBlock = protowire.AppendString(channelBlock, model)

	container := []byte{}
	container = protowire.AppendTag(container, 1, protowire.BytesType)
	container = protowire.AppendBytes(container, channelBlock)

	payload := []byte{}
	payload = protowire.AppendTag(payload, 2, protowire.BytesType)
	payload = protowire.AppendBytes(payload, container)
	payload = protowire.AppendTag(payload, 3, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 1)

	rawSignature := base64.StdEncoding.EncodeToString(payload)
	normalized, ok := sigcompat.CompatibleSignatureForProvider(sigcompat.SignatureProviderClaude, rawSignature)
	if !ok {
		t.Fatal("test Claude signature should be compatible")
	}
	return rawSignature, normalized
}

func mustTestSignature(t *testing.T) string {
	t.Helper()
	raw, _ := testClaudeResponsesThinkingSignature(t)
	return raw
}

func responsesReasoningItem(signature, text string) string {
	return fmt.Sprintf(`{"type":"reasoning","encrypted_content":%q,"summary":[{"type":"summary_text","text":%q}]}`, signature, text)
}

func responsesFunctionCallItem(callID, name string) string {
	return fmt.Sprintf(`{"type":"function_call","call_id":%q,"name":%q,"arguments":"{}"}`, callID, name)
}

func responsesFunctionCallOutputItem(callID, output string) string {
	return fmt.Sprintf(`{"type":"function_call_output","call_id":%q,"output":%q}`, callID, output)
}

func responsesWebSearchCallItem(id, query string) string {
	return fmt.Sprintf(`{"type":"web_search_call","id":%q,"status":"completed","action":{"type":"search","query":%q}}`, id, query)
}

func TestConvertOpenAIResponsesRequestToClaude_WebSearchSeparatesToolUseWithThinking(t *testing.T) {
	rawSig, _ := testClaudeResponsesThinkingSignatureForModel(t, "claude-opus-5-test")

	t.Run("function_call then web_search_call", func(t *testing.T) {
		raw := responsesRequestFromItems(
			`{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}`,
			responsesReasoningItem(rawSig, "I should search and then run a command."),
			`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Searching, then running."}]}`,
			responsesFunctionCallItem("call_00_abc", "exec_command"),
			responsesWebSearchCallItem("ws_srvtoolu_12_x", "hello world"),
			responsesFunctionCallOutputItem("call_00_abc", "hi"),
		)

		out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
		content := gjson.GetBytes(out, "messages.1.content").Array()
		wantTypes := []string{"thinking", "text", "server_tool_use", "web_search_tool_result", "thinking", "tool_use"}
		if len(content) != len(wantTypes) {
			t.Fatalf("assistant content count = %d, want %d. Output: %s", len(content), len(wantTypes), out)
		}
		for index, wantType := range wantTypes {
			if got := content[index].Get("type").String(); got != wantType {
				t.Fatalf("assistant content[%d].type = %q, want %q. Output: %s", index, got, wantType, out)
			}
		}
		firstSignature := content[0].Get("signature").String()
		if firstSignature == "" {
			t.Fatalf("first thinking signature missing. Output: %s", out)
		}
		if got := content[4].Get("signature").String(); got != firstSignature {
			t.Fatalf("second thinking signature = %q, want replayed %q", got, firstSignature)
		}
		if got := content[4].Get("thinking").String(); got != "I should search and then run a command." {
			t.Fatalf("second thinking text = %q, want original thinking text", got)
		}
		if got := content[5].Get("id").String(); got != "call_00_abc" {
			t.Fatalf("tool_use id = %q, want call_00_abc", got)
		}
	})

	t.Run("web_search_call then function_call", func(t *testing.T) {
		raw := responsesRequestFromItems(
			`{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}`,
			responsesReasoningItem(rawSig, "I should search and then run a command."),
			`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Searching, then running."}]}`,
			responsesWebSearchCallItem("ws_srvtoolu_12_x", "hello world"),
			responsesFunctionCallItem("call_00_abc", "exec_command"),
			responsesFunctionCallOutputItem("call_00_abc", "hi"),
		)

		out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
		content := gjson.GetBytes(out, "messages.1.content").Array()
		wantTypes := []string{"thinking", "text", "server_tool_use", "web_search_tool_result", "thinking", "tool_use"}
		if len(content) != len(wantTypes) {
			t.Fatalf("assistant content count = %d, want %d. Output: %s", len(content), len(wantTypes), out)
		}
		for index, wantType := range wantTypes {
			if got := content[index].Get("type").String(); got != wantType {
				t.Fatalf("assistant content[%d].type = %q, want %q. Output: %s", index, got, wantType, out)
			}
		}
		firstSignature := content[0].Get("signature").String()
		if got := content[4].Get("signature").String(); got != firstSignature {
			t.Fatalf("second thinking signature = %q, want replayed %q", got, firstSignature)
		}
		if got := content[5].Get("id").String(); got != "call_00_abc" {
			t.Fatalf("tool_use id = %q, want call_00_abc", got)
		}
	})

	t.Run("no thinking item present", func(t *testing.T) {
		raw := responsesRequestFromItems(
			`{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}`,
			`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Searching, then running."}]}`,
			responsesWebSearchCallItem("ws_srvtoolu_12_x", "hello world"),
			responsesFunctionCallItem("call_00_abc", "exec_command"),
			responsesFunctionCallOutputItem("call_00_abc", "hi"),
		)

		out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
		content := gjson.GetBytes(out, "messages.1.content").Array()
		wantTypes := []string{"text", "server_tool_use", "web_search_tool_result", "tool_use"}
		if len(content) != len(wantTypes) {
			t.Fatalf("assistant content count = %d, want %d. Output: %s", len(content), len(wantTypes), out)
		}
		for index, wantType := range wantTypes {
			if got := content[index].Get("type").String(); got != wantType {
				t.Fatalf("assistant content[%d].type = %q, want %q. Output: %s", index, got, wantType, out)
			}
		}
	})

	t.Run("selects latest thinking block when multiple exist", func(t *testing.T) {
		firstRaw, _ := testClaudeResponsesThinkingSignatureForModel(t, "claude-opus-5-first")
		secondRaw, _ := testClaudeResponsesThinkingSignatureForModel(t, "claude-opus-5-second")

		raw := responsesRequestFromItems(
			`{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}`,
			responsesReasoningItem(firstRaw, "first reasoning"),
			`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"thought once"}]}`,
			responsesReasoningItem(secondRaw, "second reasoning"),
			responsesWebSearchCallItem("ws_srvtoolu_12_x", "hello world"),
			responsesFunctionCallItem("call_00_abc", "exec_command"),
			responsesFunctionCallOutputItem("call_00_abc", "hi"),
		)

		out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
		content := gjson.GetBytes(out, "messages.1.content").Array()
		wantTypes := []string{"thinking", "text", "thinking", "server_tool_use", "web_search_tool_result", "thinking", "tool_use"}
		if len(content) != len(wantTypes) {
			t.Fatalf("assistant content count = %d, want %d. Output: %s", len(content), len(wantTypes), out)
		}
		for index, wantType := range wantTypes {
			if got := content[index].Get("type").String(); got != wantType {
				t.Fatalf("assistant content[%d].type = %q, want %q. Output: %s", index, got, wantType, out)
			}
		}
		if got := content[5].Get("signature").String(); got != content[2].Get("signature").String() {
			t.Fatalf("separated thinking signature = %q, want latest block %q", got, content[2].Get("signature").String())
		}
		if got := content[5].Get("thinking").String(); got != "second reasoning" {
			t.Fatalf("separated thinking text = %q, want latest 'second reasoning'", got)
		}
	})
}
