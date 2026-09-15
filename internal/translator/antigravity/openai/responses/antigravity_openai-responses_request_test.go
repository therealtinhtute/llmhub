package responses

import (
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/signature"
	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestToAntigravity_OrphanFunctionCallOutputBecomesUserText(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.7-flash-high",
		"input": [
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Task initialization"}]},
			{"type":"function_call_output","id":"fco_01a09fca-8d33-73a1-97fd-4d83ecc02f9d","name":"send_message_to_thread","output":"<codex_delegation>\n  <source_thread_id>01a022d7-d4d0-72b2-8571-4590484ccaee</source_thread_id>\n  <input>Execute sub-task</input>\n</codex_delegation>"},
			{"type":"function_call","call_id":"call_1789387253098037589_85","name":"Bash","arguments":"{\"command\":\"pwd\"}"},
			{"type":"function_call_output","call_id":"call_1789387253098037589_85","id":"fco_01a09fca-a5f0-7b40-9943-21fbc923c537","output":"/Users/developer"}
		],
		"tools": [{"type":"function","name":"Bash","description":"Runs Bash command.","strict":false,
			"parameters":{"type":"object","properties":{"command":{"type":"string"}},
			"required":["command"],"additionalProperties":false}}],
		"tool_choice": "auto",
		"parallel_tool_calls": false,
		"store": false,
		"stream": false
	}`

	out := ConvertOpenAIResponsesRequestToAntigravity("gemini-3.7-flash-high", []byte(inputJSON), false)
	rawRequest := gjson.GetBytes(out, "request").Raw
	if errPair := signature.ValidateGeminiFunctionCallPairing([]byte(rawRequest)); errPair != nil {
		t.Fatalf("ValidateGeminiFunctionCallPairing failed on Antigravity orphan output request: %v; output=%s", errPair, out)
	}

	delegationFound := false
	bashCallID := ""
	bashResponseID := ""
	for _, content := range gjson.GetBytes(out, "request.contents").Array() {
		for _, part := range content.Get("parts").Array() {
			if fr := part.Get("functionResponse"); fr.Exists() {
				if fr.Get("id").String() == "" {
					t.Fatalf("orphan output emitted as functionResponse with empty id: %s", string(out))
				}
				if fr.Get("name").String() == "Bash" {
					bashResponseID = fr.Get("id").String()
				}
			}
			if part.Get("functionCall.name").String() == "Bash" {
				bashCallID = part.Get("functionCall.id").String()
			}
			if content.Get("role").String() == "user" && strings.Contains(part.Get("text").String(), "<codex_delegation>") {
				delegationFound = true
			}
		}
	}
	if !delegationFound {
		t.Fatalf("expected orphan send_message_to_thread output as user text; output=%s", string(out))
	}
	if bashCallID != "call_1789387253098037589_85" {
		t.Fatalf("bash functionCall.id = %q; output=%s", bashCallID, string(out))
	}
	if bashResponseID != "call_1789387253098037589_85" {
		t.Fatalf("bash functionResponse.id = %q; output=%s", bashResponseID, string(out))
	}
}

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4 ("preserve Gemini prompt
// cache by demoting mid-session developer messages").
func TestConvertOpenAIResponsesRequestToAntigravity_MidSessionDeveloperMessageDoesNotMutateSystemInstruction(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3-flash",
		"instructions": "Be a helpful assistant",
		"input": [
			{
				"type": "message",
				"role": "user",
				"content": [
					{"type": "input_text", "text": "Turn 1 user"}
				]
			},
			{
				"type": "message",
				"role": "assistant",
				"content": [
					{"type": "output_text", "text": "Turn 1 assistant"}
				]
			},
			{
				"type": "message",
				"role": "developer",
				"content": "<image_resize_notice>Image 1 was resized to 800x600</image_resize_notice>"
			},
			{
				"type": "message",
				"role": "user",
				"content": [
					{"type": "input_text", "text": "Turn 2 user"}
				]
			}
		]
	}`

	out := ConvertOpenAIResponsesRequestToAntigravity("gemini-3-flash", []byte(inputJSON), false)
	if !gjson.ValidBytes(out) {
		t.Fatalf("invalid JSON output: %s", out)
	}

	// In Antigravity envelope, systemInstruction is at request.systemInstruction
	sysParts := gjson.GetBytes(out, "request.systemInstruction.parts").Array()
	if len(sysParts) != 1 {
		t.Fatalf("request.systemInstruction parts count = %d, want 1; output=%s", len(sysParts), out)
	}
	if got := sysParts[0].Get("text").String(); got != "Be a helpful assistant" {
		t.Fatalf("systemInstruction part = %q, want %q; output=%s", got, "Be a helpful assistant", out)
	}

	contents := gjson.GetBytes(out, "request.contents").Array()
	if len(contents) != 3 {
		t.Fatalf("request.contents count = %d, want 3; output=%s", len(contents), out)
	}
	if contents[2].Get("role").String() != "user" {
		t.Fatalf("turn 2 role = %q, want user; output=%s", contents[2].Get("role").String(), out)
	}
	turn2Parts := contents[2].Get("parts").Array()
	if len(turn2Parts) != 2 {
		t.Fatalf("turn 2 parts count = %d, want 2; output=%s", len(turn2Parts), out)
	}
	if got := turn2Parts[0].Get("text").String(); got != "<image_resize_notice>Image 1 was resized to 800x600</image_resize_notice>" {
		t.Fatalf("turn 2 part 0 = %q, want image_resize_notice; output=%s", got, out)
	}
	if got := turn2Parts[1].Get("text").String(); got != "Turn 2 user" {
		t.Fatalf("turn 2 part 1 = %q, want Turn 2 user; output=%s", got, out)
	}
}

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
func TestConvertOpenAIResponsesRequestToAntigravity_InterveningDeveloperMessagePreservesToolPairing(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3-flash",
		"instructions": "Be a helpful assistant",
		"input": [
			{
				"type": "message",
				"role": "user",
				"content": [
					{"type": "input_text", "text": "Run tool"}
				]
			},
			{
				"type": "function_call",
				"call_id": "call-1",
				"name": "run_command",
				"arguments": "{\"command\":\"echo test\"}"
			},
			{
				"type": "message",
				"role": "developer",
				"content": "<permissions instructions>\nApproved: echo\n</permissions instructions>"
			},
			{
				"type": "function_call_output",
				"call_id": "call-1",
				"output": "test"
			}
		]
	}`

	out := ConvertOpenAIResponsesRequestToAntigravity("gemini-3-flash", []byte(inputJSON), false)
	if !gjson.ValidBytes(out) {
		t.Fatalf("invalid JSON output: %s", out)
	}

	rawRequest := gjson.GetBytes(out, "request").Raw
	if errPair := signature.ValidateGeminiFunctionCallPairing([]byte(rawRequest)); errPair != nil {
		t.Fatalf("ValidateGeminiFunctionCallPairing failed on Antigravity request: %v; output=%s", errPair, out)
	}
}
