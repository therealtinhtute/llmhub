package gemini

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertGeminiRequestToClaude_DeterministicToolIDs(t *testing.T) {
	raw := []byte(`{
		"contents": [
			{
				"role": "model",
				"parts": [
					{"functionCall": {"name": "first_tool", "args": {"q": "one"}}}
				]
			},
			{
				"role": "user",
				"parts": [
					{"functionResponse": {"name": "first_tool", "response": {"result": "ok1"}}}
				]
			},
			{
				"role": "model",
				"parts": [
					{"functionCall": {"name": "second_tool", "args": {"q": "two"}}}
				]
			},
			{
				"role": "user",
				"parts": [
					{"functionResponse": {"name": "second_tool", "response": {"result": "ok2"}}}
				]
			}
		]
	}`)

	out1 := ConvertGeminiRequestToClaude("claude-sonnet-4", raw, false)
	out2 := ConvertGeminiRequestToClaude("claude-sonnet-4", raw, false)

	if string(out1) != string(out2) {
		t.Fatalf("expected deterministic output across multiple conversions, got different outputs:\nout1=%s\nout2=%s", string(out1), string(out2))
	}

	wantID1 := "toolu_gemini_0000000000000001"
	wantID2 := "toolu_gemini_0000000000000002"

	gotCall1 := gjson.GetBytes(out1, "messages.0.content.0.id").String()
	gotResp1 := gjson.GetBytes(out1, "messages.1.content.0.tool_use_id").String()
	gotCall2 := gjson.GetBytes(out1, "messages.2.content.0.id").String()
	gotResp2 := gjson.GetBytes(out1, "messages.3.content.0.tool_use_id").String()

	if gotCall1 != wantID1 || gotResp1 != wantID1 {
		t.Fatalf("expected first tool pair to have id %q, got call=%q, resp=%q", wantID1, gotCall1, gotResp1)
	}
	if gotCall2 != wantID2 || gotResp2 != wantID2 {
		t.Fatalf("expected second tool pair to have id %q, got call=%q, resp=%q", wantID2, gotCall2, gotResp2)
	}
}
