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

// Ported from upstream CLIProxyAPI commit 728ea8b8557c ("nest image parts inside
// functionResponse"): image blocks in a function_call_output must land under
// functionResponse.parts as inlineData, not as sibling parts.
func TestConvertOpenAIResponsesRequestToGemini_FunctionCallOutputWithImages(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.7-flash-high",
		"input": [
			{
				"role": "user",
				"content": [{"type": "input_text", "text": "read the screenshot"}]
			},
			{
				"type": "function_call",
				"id": "fc_1",
				"call_id": "call_img",
				"name": "read_file",
				"arguments": "{}"
			},
			{
				"type": "function_call_output",
				"call_id": "call_img",
				"output": [
					{"type": "input_text", "text": "Read image file [image/png]"},
					{"type": "input_image", "image_url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg=="}
				]
			}
		]
	}`

	output := ConvertOpenAIResponsesRequestToGemini("gemini-3.7-flash-high", []byte(inputJSON), false)
	userContent := gjson.GetBytes(output, "contents.2")
	if userContent.Get("role").String() != "user" {
		t.Fatalf("expected role user in tool response content, got %s", userContent.Raw)
	}

	parts := userContent.Get("parts").Array()
	if len(parts) != 1 {
		t.Fatalf("expected 1 part (functionResponse with nested inlineData), got %d; raw: %s", len(parts), userContent.Raw)
	}

	fr := parts[0].Get("functionResponse")
	if !fr.Exists() {
		t.Fatalf("expected functionResponse part, got %s", parts[0].Raw)
	}
	if got := fr.Get("name").String(); got != "read_file" {
		t.Fatalf("expected functionResponse.name = %q, got %q", "read_file", got)
	}
	if got := fr.Get("response.result").String(); got != "Read image file [image/png]" {
		t.Fatalf("expected functionResponse.response.result = %q, got %q", "Read image file [image/png]", got)
	}

	img := fr.Get("parts.0.inlineData")
	if !img.Exists() {
		t.Fatalf("expected functionResponse.parts.0 to have inlineData, got %s", fr.Raw)
	}
	if got := img.Get("mimeType").String(); got != "image/png" {
		t.Fatalf("expected mimeType = %q, got %q", "image/png", got)
	}
	if got := img.Get("data").String(); got != "iVBORw0KGgoAAAANSUhEUg==" {
		t.Fatalf("expected data = %q, got %q", "iVBORw0KGgoAAAANSUhEUg==", got)
	}
}

// Ported from upstream CLIProxyAPI commit 728ea8b8557c: multiple images in one
// tool output each become a nested functionResponse.parts entry.
func TestConvertOpenAIResponsesRequestToGemini_FunctionCallOutputWithMultipleImages(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.7-flash-high",
		"input": [
			{
				"role": "user",
				"content": [{"type": "input_text", "text": "show two screenshots"}]
			},
			{
				"type": "function_call",
				"id": "fc_multi",
				"call_id": "call_multi",
				"name": "take_screenshots",
				"arguments": "{}"
			},
			{
				"type": "function_call_output",
				"call_id": "call_multi",
				"output": [
					{"type": "input_text", "text": "captured 2 images"},
					{"type": "input_image", "image_url": "data:image/png;base64,QUJD"},
					{"type": "input_image", "image_url": "data:image/jpeg;base64,REVm"}
				]
			}
		]
	}`

	output := ConvertOpenAIResponsesRequestToGemini("gemini-3.7-flash-high", []byte(inputJSON), false)
	userContent := gjson.GetBytes(output, "contents.2")
	parts := userContent.Get("parts").Array()
	if len(parts) != 1 {
		t.Fatalf("expected 1 functionResponse part, got %d; raw: %s", len(parts), userContent.Raw)
	}

	fr := parts[0].Get("functionResponse")
	if !fr.Exists() {
		t.Fatalf("expected functionResponse, got %s", parts[0].Raw)
	}
	if got := fr.Get("id").String(); got != "call_multi" {
		t.Fatalf("expected id call_multi, got %q", got)
	}
	if got := fr.Get("response.result").String(); got != "captured 2 images" {
		t.Fatalf("expected result 'captured 2 images', got %q", got)
	}

	nested := fr.Get("parts").Array()
	if len(nested) != 2 {
		t.Fatalf("expected 2 nested inlineData parts, got %d; raw: %s", len(nested), fr.Raw)
	}
	if got := nested[0].Get("inlineData.mimeType").String(); got != "image/png" {
		t.Fatalf("expected first nested mimeType image/png, got %q", got)
	}
	if got := nested[1].Get("inlineData.mimeType").String(); got != "image/jpeg" {
		t.Fatalf("expected second nested mimeType image/jpeg, got %q", got)
	}
}

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4 ("preserve Gemini prompt
// cache by demoting mid-session developer messages").
func TestConvertOpenAIResponsesRequestToGemini_MidSessionDeveloperMessageDoesNotMutateSystemInstruction(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.5-flash",
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

	output := ConvertOpenAIResponsesRequestToGemini("gemini-3.5-flash", []byte(inputJSON), false)
	result := gjson.ParseBytes(output)

	// systemInstruction must remain strictly unchanged (only original instructions, not developer notice)
	systemInstruction := result.Get("systemInstruction")
	if !systemInstruction.Exists() {
		t.Fatalf("systemInstruction missing; output=%s", output)
	}
	parts := systemInstruction.Get("parts").Array()
	if len(parts) != 1 {
		t.Fatalf("systemInstruction parts count = %d, want 1; output=%s", len(parts), output)
	}
	if got := parts[0].Get("text").String(); got != "Be a helpful assistant" {
		t.Fatalf("systemInstruction part = %q, want %q; output=%s", got, "Be a helpful assistant", output)
	}

	// contents should contain user, model, user (with merged developer notice + turn 2 user text)
	contents := result.Get("contents").Array()
	if len(contents) != 3 {
		t.Fatalf("contents count = %d, want 3; output=%s", len(contents), output)
	}
	if contents[0].Get("role").String() != "user" || contents[0].Get("parts.0.text").String() != "Turn 1 user" {
		t.Fatalf("turn 1 user content malformed; output=%s", output)
	}
	if contents[1].Get("role").String() != "model" || contents[1].Get("parts.0.text").String() != "Turn 1 assistant" {
		t.Fatalf("turn 1 model content malformed; output=%s", output)
	}
	if contents[2].Get("role").String() != "user" {
		t.Fatalf("turn 2 user content role = %q, want user; output=%s", contents[2].Get("role").String(), output)
	}
	turn2Parts := contents[2].Get("parts").Array()
	if len(turn2Parts) != 2 {
		t.Fatalf("turn 2 parts count = %d, want 2; output=%s", len(turn2Parts), output)
	}
	if got := turn2Parts[0].Get("text").String(); got != "<image_resize_notice>Image 1 was resized to 800x600</image_resize_notice>" {
		t.Fatalf("turn 2 part 0 = %q, want image_resize_notice; output=%s", got, output)
	}
	if got := turn2Parts[1].Get("text").String(); got != "Turn 2 user" {
		t.Fatalf("turn 2 part 1 = %q, want Turn 2 user; output=%s", got, output)
	}
}

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
func TestConvertOpenAIResponsesRequestToGemini_MultipleMidSessionDeveloperMessagesArrayContent(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.5-flash",
		"instructions": "Be a helpful assistant",
		"input": [
			{
				"type": "message",
				"role": "user",
				"content": [
					{"type": "input_text", "text": "Turn 1"}
				]
			},
			{
				"type": "message",
				"role": "assistant",
				"content": [
					{"type": "output_text", "text": "Reply 1"}
				]
			},
			{
				"type": "message",
				"role": "developer",
				"content": [
					{"type": "input_text", "text": "<permissions instructions>\nApproved: git\n</permissions instructions>"}
				]
			},
			{
				"type": "message",
				"role": "developer",
				"content": [
					{"type": "input_text", "text": "<collaboration_mode>\nPlan\n</collaboration_mode>"}
				]
			},
			{
				"type": "message",
				"role": "user",
				"content": [
					{"type": "input_text", "text": "Proceed"}
				]
			}
		]
	}`

	output := ConvertOpenAIResponsesRequestToGemini("gemini-3.5-flash", []byte(inputJSON), false)
	result := gjson.ParseBytes(output)

	// systemInstruction only contains original instructions
	parts := result.Get("systemInstruction.parts").Array()
	if len(parts) != 1 || parts[0].Get("text").String() != "Be a helpful assistant" {
		t.Fatalf("systemInstruction corrupted: %s", output)
	}

	// All mid-session developer messages coalesced into the final user turn
	contents := result.Get("contents").Array()
	if len(contents) != 3 {
		t.Fatalf("contents count = %d, want 3; output=%s", len(contents), output)
	}
	turn2Parts := contents[2].Get("parts").Array()
	if len(turn2Parts) != 3 {
		t.Fatalf("turn 2 parts count = %d, want 3; output=%s", len(turn2Parts), output)
	}
	if !strings.Contains(turn2Parts[0].Get("text").String(), "permissions instructions") {
		t.Fatalf("part 0 mismatch; output=%s", output)
	}
	if !strings.Contains(turn2Parts[1].Get("text").String(), "collaboration_mode") {
		t.Fatalf("part 1 mismatch; output=%s", output)
	}
	if turn2Parts[2].Get("text").String() != "Proceed" {
		t.Fatalf("part 2 mismatch; output=%s", output)
	}
}

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
func TestConvertOpenAIResponsesRequestToGemini_InterveningDeveloperMessagePreservesToolPairing(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.5-flash",
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

	output := ConvertOpenAIResponsesRequestToGemini("gemini-3.5-flash", []byte(inputJSON), false)
	result := gjson.ParseBytes(output)

	// Validate function call pairing passes strictly (no content turn before pending functionResponse)
	if errPair := signature.ValidateGeminiFunctionCallPairing(output); errPair != nil {
		t.Fatalf("ValidateGeminiFunctionCallPairing failed: %v; output=%s", errPair, output)
	}

	// systemInstruction only contains original instructions
	parts := result.Get("systemInstruction.parts").Array()
	if len(parts) != 1 || parts[0].Get("text").String() != "Be a helpful assistant" {
		t.Fatalf("systemInstruction corrupted: %s", output)
	}

	// Function response should have matching call id and name
	foundFR := false
	for _, content := range result.Get("contents").Array() {
		for _, part := range content.Get("parts").Array() {
			if part.Get("functionResponse.name").String() == "run_command" {
				foundFR = true
			}
		}
	}
	if !foundFR {
		t.Fatalf("functionResponse run_command not found or lost pairing: %s", output)
	}
}

// Ported from upstream CLIProxyAPI commit e56fae88c0ac ("flush pending developer
// notice before intervening user turn").
func TestConvertOpenAIResponsesRequestToGemini_InterveningDeveloperAndUserMessageFlushesInOrder(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3.5-flash",
		"instructions": "Be a helpful assistant",
		"input": [
			{
				"type": "function_call",
				"call_id": "call-1",
				"name": "run_command",
				"arguments": "{\"command\":\"test\"}"
			},
			{
				"type": "message",
				"role": "developer",
				"content": "<permissions instructions>\nApproved: test\n</permissions instructions>"
			},
			{
				"type": "message",
				"role": "user",
				"content": [
					{"type": "input_text", "text": "Wait, also check this"}
				]
			},
			{
				"type": "function_call_output",
				"call_id": "call-1",
				"output": "done"
			}
		]
	}`

	output := ConvertOpenAIResponsesRequestToGemini("gemini-3.5-flash", []byte(inputJSON), false)
	result := gjson.ParseBytes(output)

	// Pairing should be valid
	if errPair := signature.ValidateGeminiFunctionCallPairing(output); errPair != nil {
		t.Fatalf("ValidateGeminiFunctionCallPairing failed: %v; output=%s", errPair, output)
	}

	contents := result.Get("contents").Array()
	if len(contents) != 3 {
		t.Fatalf("contents count = %d, want 3; output=%s", len(contents), output)
	}
	if contents[0].Get("role").String() != "model" {
		t.Fatalf("turn 0 role = %q, want model", contents[0].Get("role").String())
	}
	midParts := contents[1].Get("parts").Array()
	if len(midParts) != 2 {
		t.Fatalf("turn 1 parts count = %d, want 2; output=%s", len(midParts), output)
	}
	if !strings.Contains(midParts[0].Get("text").String(), "permissions instructions") {
		t.Fatalf("turn 1 part 0 should be developer notice; got %s", midParts[0].Raw)
	}
	if midParts[1].Get("text").String() != "Wait, also check this" {
		t.Fatalf("turn 1 part 1 should be user text; got %s", midParts[1].Raw)
	}
	if !contents[2].Get("parts.0.functionResponse").Exists() {
		t.Fatalf("turn 2 should be functionResponse; got %s", contents[2].Raw)
	}
}
