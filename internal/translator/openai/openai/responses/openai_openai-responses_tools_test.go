package responses

import (
	"errors"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func TestResponsesToolDeclarationTableNormalizesTopLevelAdditionalAndNamespaceTools(t *testing.T) {
	request := []byte(`{
		"tools":[
			{"type":"function","name":"lookup","parameters":{"type":"object","properties":{"q":{"type":"string"}}}},
			{"type":"namespace","name":"repo","tools":[
				{"type":"function","name":"read","parameters":{"type":"object"}},
				{"type":"custom","name":"patch"},
				{"type":"function","name":"mcp__stable"}
			]},
			{"type":"web_search_preview"}
		],
		"input":[
			{"type":"additional_tools","tools":[{"type":"custom","name":"shell"}]},
			{"type":"function_call","call_id":"call_1","namespace":"repo","name":"read","arguments":"{}"},
			{"type":"message","role":"user","content":"hello"}
		],
		"tool_choice":{"type":"function","namespace":"repo","name":"read"}
	}`)

	table, err := BuildResponsesToolDeclarationTable(request)
	if err != nil {
		t.Fatalf("BuildResponsesToolDeclarationTable() error = %v", err)
	}
	if len(table.declarations) != 5 {
		t.Fatalf("declaration count = %d, want 5", len(table.declarations))
	}
	if got := table.byEffective["repo__read"].Namespace; got != "repo" {
		t.Fatalf("repo__read namespace = %q, want repo", got)
	}
	if got := table.byEffective["repo__patch"].Type; got != "custom" {
		t.Fatalf("repo__patch type = %q, want custom", got)
	}
	if _, ok := table.byEffective["mcp__stable"]; !ok {
		t.Fatal("mcp__stable was qualified or omitted")
	}

	normalized := NormalizeResponsesToolsForCodex(request)
	if got := gjson.GetBytes(normalized, "tools.#").Int(); got != 6 {
		t.Fatalf("normalized tools count = %d, want 6: %s", got, normalized)
	}
	wantNames := []string{"lookup", "repo__read", "repo__patch", "mcp__stable", "", "shell"}
	for i, want := range wantNames {
		if got := gjson.GetBytes(normalized, "tools."+string(rune('0'+i))+".name").String(); got != want {
			t.Fatalf("tools[%d].name = %q, want %q: %s", i, got, want, normalized)
		}
	}
	if got := gjson.GetBytes(normalized, "input.#").Int(); got != 2 {
		t.Fatalf("normalized input count = %d, want 2: %s", got, normalized)
	}
	if got := gjson.GetBytes(normalized, "input.0.name").String(); got != "repo__read" {
		t.Fatalf("input function call name = %q, want repo__read: %s", got, normalized)
	}
	if gjson.GetBytes(normalized, "input.0.namespace").Exists() {
		t.Fatalf("outbound input call retained namespace: %s", normalized)
	}
	if got := gjson.GetBytes(normalized, "tool_choice.name").String(); got != "repo__read" {
		t.Fatalf("tool_choice.name = %q, want repo__read: %s", got, normalized)
	}
	if gjson.GetBytes(normalized, "tool_choice.namespace").Exists() {
		t.Fatalf("outbound tool_choice retained namespace: %s", normalized)
	}
	if got := gjson.GetBytes(normalized, "tools.0.parameters.properties.q.type").String(); got != "string" {
		t.Fatalf("function schema was not preserved: %s", normalized)
	}
}

func TestResponsesToolDeclarationTableRejectsDistinctIdentityCollision(t *testing.T) {
	request := []byte(`{
		"tools":[
			{"type":"function","name":"repo__read"},
			{"type":"namespace","name":"repo","tools":[{"type":"function","name":"read"}]}
		]
	}`)

	_, err := BuildResponsesToolDeclarationTable(request)
	var collision *ResponsesToolNameCollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("error = %v, want ResponsesToolNameCollisionError", err)
	}
	if collision.EffectiveName != "repo__read" {
		t.Fatalf("collision effective name = %q, want repo__read", collision.EffectiveName)
	}
}

func TestResponsesToolDeclarationTableRestoresExactIdentityWithoutDelimiterParsing(t *testing.T) {
	request := []byte(`{
		"tools":[
			{"type":"namespace","name":"repo","tools":[
				{"type":"function","name":"read"},
				{"type":"custom","name":"patch"}
			]}
		]
	}`)
	table, err := BuildResponsesToolDeclarationTable(request)
	if err != nil {
		t.Fatalf("BuildResponsesToolDeclarationTable() error = %v", err)
	}

	functionEvent := table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.done","item":{"id":"fc_1","type":"function_call","name":"repo__read","arguments":"{}"}}`))
	if got := gjson.GetBytes(functionEvent, "item.name").String(); got != "read" {
		t.Fatalf("restored function name = %q, want read: %s", got, functionEvent)
	}
	if got := gjson.GetBytes(functionEvent, "item.namespace").String(); got != "repo" {
		t.Fatalf("restored function namespace = %q, want repo: %s", got, functionEvent)
	}

	customEvent := table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.done","item":{"id":"fc_2","type":"function_call","name":"repo__patch","arguments":"{\"input\":\"diff\"}"}}`))
	if got := gjson.GetBytes(customEvent, "item.type").String(); got != "custom_tool_call" {
		t.Fatalf("restored custom type = %q, want custom_tool_call: %s", got, customEvent)
	}
	if got := gjson.GetBytes(customEvent, "item.input").String(); got != "diff" {
		t.Fatalf("restored custom input = %q, want diff: %s", got, customEvent)
	}
	if gjson.GetBytes(customEvent, "item.arguments").Exists() {
		t.Fatalf("custom call retained arguments: %s", customEvent)
	}

	unknown := table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.done","item":{"type":"function_call","name":"other__read","arguments":"{}"}}`))
	if gjson.GetBytes(unknown, "item.namespace").Exists() || gjson.GetBytes(unknown, "item.name").String() != "other__read" {
		t.Fatalf("unknown qualified name was restored by parsing: %s", unknown)
	}
}

func TestResponsesToolDeclarationTableRestoresCompleteCustomStreamingLifecycle(t *testing.T) {
	request := []byte(`{"tools":[{"type":"custom","name":"patch"}]}`)
	table, err := BuildResponsesToolDeclarationTable(request)
	if err != nil {
		t.Fatalf("BuildResponsesToolDeclarationTable() error = %v", err)
	}

	added := table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","name":"patch","arguments":""}}`))
	if got := gjson.GetBytes(added, "item.type").String(); got != "custom_tool_call" {
		t.Fatalf("added item type = %q, want custom_tool_call: %s", got, added)
	}
	if got := gjson.GetBytes(added, "item.id").String(); got != "ctc_1" {
		t.Fatalf("added item id = %q, want ctc_1: %s", got, added)
	}

	delta := table.RestoreResponsesToolCalls([]byte(`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"input\":\"diff\"}"}`))
	if got := gjson.GetBytes(delta, "type").String(); got != "response.custom_tool_call_input.delta" {
		t.Fatalf("delta type = %q, want response.custom_tool_call_input.delta: %s", got, delta)
	}
	if got := gjson.GetBytes(delta, "item_id").String(); got != "ctc_1" {
		t.Fatalf("delta item_id = %q, want ctc_1: %s", got, delta)
	}
	if got := gjson.GetBytes(delta, "delta").String(); got != "" {
		t.Fatalf("ambiguous custom delta = %q, want withheld until done: %s", got, delta)
	}

	doneEvents := table.RestoreResponsesToolCallEvents([]byte(`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{\"input\":\"diff\"}"}`))
	if len(doneEvents) != 2 {
		t.Fatalf("done event count = %d, want buffered delta plus done", len(doneEvents))
	}
	if got := gjson.GetBytes(doneEvents[0], "delta").String(); got != "diff" {
		t.Fatalf("terminal custom delta = %q, want diff: %s", got, doneEvents[0])
	}
	done := doneEvents[1]
	if got := gjson.GetBytes(done, "type").String(); got != "response.custom_tool_call_input.done" {
		t.Fatalf("done type = %q, want response.custom_tool_call_input.done: %s", got, done)
	}
	if got := gjson.GetBytes(done, "item_id").String(); got != "ctc_1" {
		t.Fatalf("done item_id = %q, want ctc_1: %s", got, done)
	}
	if got := gjson.GetBytes(done, "input").String(); got != "diff" {
		t.Fatalf("custom input = %q, want diff: %s", got, done)
	}
	if gjson.GetBytes(done, "arguments").Exists() {
		t.Fatalf("custom done retained arguments: %s", done)
	}

	itemDone := table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","name":"patch","arguments":"{\"input\":\"diff\"}"}}`))
	if got := gjson.GetBytes(itemDone, "item.id").String(); got != "ctc_1" {
		t.Fatalf("output_item.done id = %q, want ctc_1: %s", got, itemDone)
	}
	completed := table.RestoreResponsesToolCalls([]byte(`{"type":"response.completed","response":{"output":[{"id":"fc_1","type":"function_call","name":"patch","arguments":"{\"input\":\"diff\"}"}]}}`))
	if got := gjson.GetBytes(completed, "response.output.0.id").String(); got != "ctc_1" {
		t.Fatalf("completed item id = %q, want ctc_1: %s", got, completed)
	}
	if got := gjson.GetBytes(completed, "response.output.0.type").String(); got != "custom_tool_call" {
		t.Fatalf("completed item type = %q, want custom_tool_call: %s", got, completed)
	}
}

func TestResponsesToolDeclarationTableRestoresFragmentedCustomInputWithoutWrapperLeak(t *testing.T) {
	table, err := BuildResponsesToolDeclarationTable([]byte(`{"tools":[{"type":"custom","name":"patch"}]}`))
	if err != nil {
		t.Fatalf("BuildResponsesToolDeclarationTable() error = %v", err)
	}
	table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.added","item":{"id":"fc_1","type":"function_call","name":"patch","arguments":""}}`))

	fragments := []string{`{"in`, `put":"d`, `if`, `f"}`}
	var restored string
	for _, fragment := range fragments {
		event := []byte(`{"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":""}`)
		event, _ = sjson.SetBytes(event, "delta", fragment)
		event = table.RestoreResponsesToolCalls(event)
		if got := gjson.GetBytes(event, "type").String(); got != "response.custom_tool_call_input.delta" {
			t.Fatalf("delta type = %q, want custom input delta: %s", got, event)
		}
		restored += gjson.GetBytes(event, "delta").String()
	}
	for _, event := range table.RestoreResponsesToolCallEvents([]byte(`{"type":"response.function_call_arguments.done","item_id":"fc_1","arguments":"{\"input\":\"diff\"}"}`)) {
		if gjson.GetBytes(event, "type").String() == "response.custom_tool_call_input.delta" {
			restored += gjson.GetBytes(event, "delta").String()
		}
	}
	if restored != "diff" {
		t.Fatalf("restored deltas = %q, want diff", restored)
	}
	if strings.Contains(restored, `{"input"`) {
		t.Fatalf("restored deltas leaked wrapper JSON: %q", restored)
	}
}

func TestResponsesToolDeclarationTableDoesNotAuthorizeNamespacedToolByShortName(t *testing.T) {
	request := []byte(`{
		"tools":[{"type":"namespace","name":"repo","tools":[{"type":"function","name":"read"}]}],
		"tool_choice":{"type":"function","name":"read"}
	}`)
	normalized := NormalizeResponsesToolsForCodex(request)
	if got := gjson.GetBytes(normalized, "tool_choice.name").String(); got != "read" {
		t.Fatalf("short tool choice was authorized as namespaced declaration: %s", normalized)
	}
}

func TestResponsesToolDeclarationTablePreservesCustomInputFidelity(t *testing.T) {
	tests := []struct {
		name      string
		arguments string
		wantInput string
	}{
		{name: "raw brace text", arguments: `{diff --git a/file b/file`, wantInput: `{diff --git a/file b/file`},
		{name: "incomplete wrapper", arguments: `{"input":"diff`, wantInput: `{"input":"diff`},
		{name: "raw JSON with extra member", arguments: `{"input":"literal","other":1}`, wantInput: `{"input":"literal","other":1}`},
		{name: "exact compatibility wrapper", arguments: `{"input":"literal"}`, wantInput: "literal"},
		{name: "leading whitespace exact wrapper", arguments: " \n { \t\"input\" : \"snow \\u96ea\" } \r", wantInput: "snow 雪"},
		{name: "null input remains raw", arguments: `{"input":null}`, wantInput: `{"input":null}`},
		{name: "plain raw input", arguments: "diff", wantInput: "diff"},
		{name: "empty input", arguments: "", wantInput: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, err := BuildResponsesToolDeclarationTable([]byte(`{"tools":[{"type":"custom","name":"patch"}]}`))
			if err != nil {
				t.Fatalf("BuildResponsesToolDeclarationTable() error = %v", err)
			}
			table.RestoreResponsesToolCalls([]byte(`{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","name":"patch","arguments":""}}`))

			var delta strings.Builder
			for i := 0; i < len(tt.arguments); i++ {
				event := []byte(`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":""}`)
				event, _ = sjson.SetBytes(event, "delta", tt.arguments[i:i+1])
				for _, restored := range table.RestoreResponsesToolCallEvents(event) {
					if gjson.GetBytes(restored, "type").String() == "response.custom_tool_call_input.delta" {
						delta.WriteString(gjson.GetBytes(restored, "delta").String())
					}
				}
			}

			doneEvent := []byte(`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":""}`)
			doneEvent, _ = sjson.SetBytes(doneEvent, "arguments", tt.arguments)
			var doneInput string
			for _, restored := range table.RestoreResponsesToolCallEvents(doneEvent) {
				switch gjson.GetBytes(restored, "type").String() {
				case "response.custom_tool_call_input.delta":
					delta.WriteString(gjson.GetBytes(restored, "delta").String())
				case "response.custom_tool_call_input.done":
					doneInput = gjson.GetBytes(restored, "input").String()
				}
			}
			if got := delta.String(); got != tt.wantInput {
				t.Fatalf("custom input deltas = %q, want %q", got, tt.wantInput)
			}
			if doneInput != tt.wantInput {
				t.Fatalf("custom input done = %q, want %q", doneInput, tt.wantInput)
			}

			itemDone := []byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","name":"patch","arguments":""}}`)
			itemDone, _ = sjson.SetBytes(itemDone, "item.arguments", tt.arguments)
			itemDone = table.RestoreResponsesToolCalls(itemDone)
			if got := gjson.GetBytes(itemDone, "item.input").String(); got != tt.wantInput {
				t.Fatalf("output_item.done input = %q, want %q: %s", got, tt.wantInput, itemDone)
			}

			completed := []byte(`{"type":"response.completed","response":{"output":[{"id":"fc_1","type":"function_call","name":"patch","arguments":""}]}}`)
			completed, _ = sjson.SetBytes(completed, "response.output.0.arguments", tt.arguments)
			completed = table.RestoreResponsesToolCalls(completed)
			if got := gjson.GetBytes(completed, "response.output.0.input").String(); got != tt.wantInput {
				t.Fatalf("response.completed input = %q, want %q: %s", got, tt.wantInput, completed)
			}
		})
	}
}
