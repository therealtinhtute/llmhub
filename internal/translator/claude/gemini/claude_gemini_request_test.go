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

func TestConvertGeminiRequestToClaude_PreservesCallerSuppliedMetadataUserID(t *testing.T) {
	testCases := []struct {
		name     string
		rawJSON  string
		expected string
	}{
		{
			name:     "plain string",
			rawJSON:  `{"model":"claude-test","metadata":{"user_id":"custom-gemini-user-123"},"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
			expected: "custom-gemini-user-123",
		},
		{
			name:     "special characters and json string",
			rawJSON:  `{"model":"claude-test","metadata":{"user_id":"foo\"bar\nbaz\\qux"},"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
			expected: "foo\"bar\nbaz\\qux",
		},
		{
			name:     "claude code json format",
			rawJSON:  `{"model":"claude-test","metadata":{"user_id":"{\"device_id\":\"0000000000000000000000000000000000000000000000000000000000000000\",\"session_id\":\"11111111-2222-4333-8444-555555555555\"}"},"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
			expected: `{"device_id":"0000000000000000000000000000000000000000000000000000000000000000","session_id":"11111111-2222-4333-8444-555555555555"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := ConvertGeminiRequestToClaude("claude-test", []byte(tc.rawJSON), false)
			if !gjson.ValidBytes(out) {
				t.Fatalf("output is invalid json: %s", string(out))
			}
			got := gjson.GetBytes(out, "metadata.user_id").String()
			if got != tc.expected {
				t.Fatalf("metadata.user_id = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestConvertGeminiRequestToClaude_DifferentSessionsProduceDifferentUserIDs(t *testing.T) {
	a := []byte(`{"model":"claude-test","prompt_cache_key":"gemini-session-a","contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
	b := []byte(`{"model":"claude-test","prompt_cache_key":"gemini-session-b","contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
	outA := ConvertGeminiRequestToClaude("claude-test", a, false)
	outB := ConvertGeminiRequestToClaude("claude-test", b, false)
	idA := gjson.GetBytes(outA, "metadata.user_id").String()
	idB := gjson.GetBytes(outB, "metadata.user_id").String()
	if idA == idB {
		t.Fatalf("different prompt_cache_key produced identical metadata.user_id: %q", idA)
	}
}

func TestConvertGeminiRequestToClaude_DefaultRoleDifferentContentProducesDifferentUserIDs(t *testing.T) {
	a := []byte(`{"contents":[{"parts":[{"text":"first prompt"}]}]}`)
	b := []byte(`{"contents":[{"parts":[{"text":"second prompt"}]}]}`)
	outA := ConvertGeminiRequestToClaude("claude-test", a, false)
	outB := ConvertGeminiRequestToClaude("claude-test", b, false)
	idA := gjson.GetBytes(outA, "metadata.user_id").String()
	idB := gjson.GetBytes(outB, "metadata.user_id").String()
	if idA == "" || idB == "" || idA == "unknown" || idB == "unknown" {
		t.Fatalf("expected valid derived user_id without role, got idA=%q idB=%q", idA, idB)
	}
	if idA == idB {
		t.Fatalf("different prompt texts without role produced identical metadata.user_id: %q", idA)
	}
}
