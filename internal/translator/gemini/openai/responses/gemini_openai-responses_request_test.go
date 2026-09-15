package responses

import (
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/signature"
	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestToGemini_OrphanFunctionCallOutputBecomesUserText(t *testing.T) {
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

	out := ConvertOpenAIResponsesRequestToGemini("gemini-3.7-flash-high", []byte(inputJSON), false)
	if errPair := signature.ValidateGeminiFunctionCallPairing(out); errPair != nil {
		t.Fatalf("ValidateGeminiFunctionCallPairing failed on orphan output request: %v; output=%s", errPair, out)
	}

	delegationFound := false
	bashCallID := ""
	bashResponseID := ""
	for _, content := range gjson.GetBytes(out, "contents").Array() {
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

func TestConvertOpenAIResponsesRequestToGemini_UnpairedExplicitCallIDBecomesUserText(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.7-flash-high",
		"input": [
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Task initialization"}]},
			{"type":"function_call_output","call_id":"call_missing","name":"send_message_to_thread","output":"<codex_delegation>Execute sub-task</codex_delegation>"},
			{"type":"function_call","call_id":"call_1789387253098037589_85","name":"Bash","arguments":"{\"command\":\"pwd\"}"},
			{"type":"function_call_output","call_id":"call_1789387253098037589_85","output":"/Users/developer"}
		]
	}`

	out := ConvertOpenAIResponsesRequestToGemini("gemini-3.7-flash-high", []byte(inputJSON), false)
	if errPair := signature.ValidateGeminiFunctionCallPairing(out); errPair != nil {
		t.Fatalf("ValidateGeminiFunctionCallPairing failed on unpaired output request: %v; output=%s", errPair, out)
	}

	delegationFound := false
	bashResponseFound := false
	for _, content := range gjson.GetBytes(out, "contents").Array() {
		for _, part := range content.Get("parts").Array() {
			if fr := part.Get("functionResponse"); fr.Exists() {
				if fr.Get("id").String() == "call_missing" {
					t.Fatalf("unpaired output emitted as functionResponse: %s", string(out))
				}
				if fr.Get("id").String() == "call_1789387253098037589_85" {
					bashResponseFound = true
				}
			}
			if content.Get("role").String() == "user" && strings.Contains(part.Get("text").String(), "<codex_delegation>") {
				delegationFound = true
			}
		}
	}
	if !delegationFound {
		t.Fatalf("expected unpaired send_message_to_thread output as user text; output=%s", string(out))
	}
	if !bashResponseFound {
		t.Fatalf("expected paired Bash functionResponse; output=%s", string(out))
	}
}
