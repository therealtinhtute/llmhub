package responses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ResponsesToolDeclaration preserves the client identity of one callable tool
// while recording the flat name sent to providers without namespace support.
type ResponsesToolDeclaration struct {
	Namespace     string
	Name          string
	Type          string
	EffectiveName string
}

// ResponsesToolDeclarationTable is scoped to one Responses request.
type ResponsesToolDeclarationTable struct {
	declarations         []ResponsesToolDeclaration
	byEffective          map[string]ResponsesToolDeclaration
	byCallItemID         map[string]ResponsesToolDeclaration
	customCallArgs       map[string]*strings.Builder
	customCallInputState map[string]*responsesCustomToolInputState
}

// ResponsesToolNameCollisionError reports distinct client declarations that
// flatten to the same outbound name.
type ResponsesToolNameCollisionError struct {
	EffectiveName string
	First         ResponsesToolDeclaration
	Second        ResponsesToolDeclaration
}

func (e *ResponsesToolNameCollisionError) Error() string {
	return fmt.Sprintf("tool name collision: %q and %q both map to %q", responsesDeclarationIdentity(e.First), responsesDeclarationIdentity(e.Second), e.EffectiveName)
}

// BuildResponsesToolDeclarationTable collects callable declarations from both
// top-level tools and Codex Desktop additional_tools input items.
func BuildResponsesToolDeclarationTable(requestRawJSON []byte) (*ResponsesToolDeclarationTable, error) {
	table := &ResponsesToolDeclarationTable{
		byEffective:          make(map[string]ResponsesToolDeclaration),
		byCallItemID:         make(map[string]ResponsesToolDeclaration),
		customCallArgs:       make(map[string]*strings.Builder),
		customCallInputState: make(map[string]*responsesCustomToolInputState),
	}
	root := gjson.ParseBytes(requestRawJSON)

	collect := func(tools gjson.Result) error {
		var collectTools func(gjson.Result, string) error
		collectTools = func(items gjson.Result, namespace string) error {
			if !items.Exists() || !items.IsArray() {
				return nil
			}
			var collectErr error
			items.ForEach(func(_, tool gjson.Result) bool {
				toolType := tool.Get("type").String()
				if strings.TrimSpace(toolType) == "namespace" {
					collectErr = collectTools(tool.Get("tools"), tool.Get("name").String())
					return collectErr == nil
				}
				if !responsesCallableToolType(toolType) {
					return true
				}
				name := responsesToolName(tool)
				if strings.TrimSpace(name) == "" {
					return true
				}
				declaration := ResponsesToolDeclaration{
					Namespace:     namespace,
					Name:          name,
					Type:          toolType,
					EffectiveName: qualifyResponsesNamespaceToolName(namespace, name),
				}
				if existing, ok := table.byEffective[declaration.EffectiveName]; ok {
					if !sameResponsesDeclarationIdentity(existing, declaration) {
						collectErr = &ResponsesToolNameCollisionError{EffectiveName: declaration.EffectiveName, First: existing, Second: declaration}
						return false
					}
					return true
				}
				table.byEffective[declaration.EffectiveName] = declaration
				table.declarations = append(table.declarations, declaration)
				return true
			})
			return collectErr
		}
		return collectTools(tools, "")
	}

	if err := collect(root.Get("tools")); err != nil {
		return nil, err
	}
	if input := root.Get("input"); input.Exists() && input.IsArray() {
		var collectErr error
		input.ForEach(func(_, item gjson.Result) bool {
			if item.Get("type").String() != "additional_tools" {
				return true
			}
			collectErr = collect(item.Get("tools"))
			return collectErr == nil
		})
		if collectErr != nil {
			return nil, collectErr
		}
	}
	return table, nil
}

// NormalizeResponsesToolsForCodex flattens namespace declarations, promotes
// additional_tools, and rewrites request references to exact effective names.
func NormalizeResponsesToolsForCodex(requestRawJSON []byte) []byte {
	table, err := BuildResponsesToolDeclarationTable(requestRawJSON)
	if err != nil {
		return requestRawJSON
	}

	root := gjson.ParseBytes(requestRawJSON)
	result := requestRawJSON
	toolsOut := []byte(`[]`)
	hasTools := false
	appendTools := func(tools gjson.Result) {
		if !tools.Exists() || !tools.IsArray() {
			return
		}
		tools.ForEach(func(_, tool gjson.Result) bool {
			for _, normalized := range normalizeResponsesToolForCodex(tool, "") {
				toolsOut, _ = sjson.SetRawBytes(toolsOut, "-1", normalized)
				hasTools = true
			}
			return true
		})
	}
	appendTools(root.Get("tools"))

	if input := root.Get("input"); input.Exists() && input.IsArray() {
		inputOut := []byte(`[]`)
		input.ForEach(func(_, item gjson.Result) bool {
			if item.Get("type").String() == "additional_tools" {
				appendTools(item.Get("tools"))
				return true
			}
			normalized := table.normalizeCallReference([]byte(item.Raw), "")
			inputOut, _ = sjson.SetRawBytes(inputOut, "-1", normalized)
			return true
		})
		result, _ = sjson.SetRawBytes(result, "input", inputOut)
	}

	if hasTools {
		result, _ = sjson.SetRawBytes(result, "tools", toolsOut)
	}
	result = table.normalizeCallReference(result, "tool_choice")
	if choices := gjson.GetBytes(result, "tool_choice.tools"); choices.IsArray() {
		for i := range choices.Array() {
			result = table.normalizeCallReference(result, fmt.Sprintf("tool_choice.tools.%d", i))
		}
	}
	return result
}

// RestoreResponsesToolCalls restores declarations on direct Responses events
// and complete response payloads using exact effective-name lookup only.
func (t *ResponsesToolDeclarationTable) RestoreResponsesToolCalls(rawJSON []byte) []byte {
	events := t.RestoreResponsesToolCallEvents(rawJSON)
	if len(events) == 0 {
		return rawJSON
	}
	return events[len(events)-1]
}

// RestoreResponsesToolCallEvents restores one upstream event and may prepend a
// buffered custom-input delta when the terminal event resolves wrapper ambiguity.
func (t *ResponsesToolDeclarationTable) RestoreResponsesToolCallEvents(rawJSON []byte) [][]byte {
	if t == nil || len(t.byEffective) == 0 || len(rawJSON) == 0 {
		return [][]byte{rawJSON}
	}
	result := rawJSON
	var pending []byte
	eventType := gjson.GetBytes(result, "type").String()
	if strings.HasPrefix(eventType, "response.output_item.") {
		result = t.restoreCallItem(result, "item")
	} else if strings.HasPrefix(eventType, "response.function_call_arguments.") {
		result, pending = t.restoreCustomCallInputEvent(result)
	}
	for _, path := range []string{"response.output", "output"} {
		items := gjson.GetBytes(result, path)
		if !items.IsArray() {
			continue
		}
		for i := range items.Array() {
			result = t.restoreCallItem(result, fmt.Sprintf("%s.%d", path, i))
		}
	}
	if len(pending) > 0 {
		return [][]byte{pending, result}
	}
	return [][]byte{result}
}

func (t *ResponsesToolDeclarationTable) restoreCallItem(item []byte, itemPath string) []byte {
	prefix := ""
	if itemPath != "" {
		prefix = itemPath + "."
	}
	name := gjson.GetBytes(item, prefix+"name").String()
	declaration, ok := t.byEffective[name]
	if !ok {
		return item
	}
	if id := gjson.GetBytes(item, prefix+"id").String(); id != "" {
		t.byCallItemID[id] = declaration
	}
	item, _ = sjson.SetBytes(item, prefix+"name", declaration.Name)
	if declaration.Namespace != "" {
		item, _ = sjson.SetBytes(item, prefix+"namespace", declaration.Namespace)
	} else {
		item, _ = sjson.DeleteBytes(item, prefix+"namespace")
	}
	if responsesDeclarationKind(declaration.Type) == "custom" {
		item, _ = sjson.SetBytes(item, prefix+"type", "custom_tool_call")
		arguments := gjson.GetBytes(item, prefix+"arguments")
		if arguments.Exists() && !gjson.GetBytes(item, prefix+"input").Exists() {
			item, _ = sjson.SetBytes(item, prefix+"input", unwrapCustomToolInput(arguments.String()))
		}
		item, _ = sjson.DeleteBytes(item, prefix+"arguments")
		if id := gjson.GetBytes(item, prefix+"id").String(); strings.HasPrefix(id, "fc_") {
			item, _ = sjson.SetBytes(item, prefix+"id", "ctc_"+strings.TrimPrefix(id, "fc_"))
		}
	} else {
		item, _ = sjson.SetBytes(item, prefix+"type", "function_call")
	}
	return item
}

func (t *ResponsesToolDeclarationTable) restoreCustomCallInputEvent(event []byte) ([]byte, []byte) {
	itemID := gjson.GetBytes(event, "item_id").String()
	declaration, ok := t.byCallItemID[itemID]
	if !ok || responsesDeclarationKind(declaration.Type) != "custom" {
		return event, nil
	}

	eventType := gjson.GetBytes(event, "type").String()
	suffix := strings.TrimPrefix(eventType, "response.function_call_arguments.")
	event, _ = sjson.SetBytes(event, "type", "response.custom_tool_call_input."+suffix)
	clientItemID := itemID
	if strings.HasPrefix(itemID, "fc_") {
		clientItemID = "ctc_" + strings.TrimPrefix(itemID, "fc_")
		event, _ = sjson.SetBytes(event, "item_id", clientItemID)
	}
	state := t.customCallInputState[itemID]
	if state == nil {
		state = &responsesCustomToolInputState{}
		t.customCallInputState[itemID] = state
	}
	if suffix == "done" {
		arguments := gjson.GetBytes(event, "arguments")
		terminalArguments := ""
		if arguments.Exists() {
			terminalArguments = arguments.String()
		} else if args := t.customCallArgs[itemID]; args != nil {
			terminalArguments = args.String()
		}
		inputDelta := responsesCustomToolInputDelta(terminalArguments, state, true)
		event, _ = sjson.SetBytes(event, "input", unwrapCustomToolInput(terminalArguments))
		event, _ = sjson.DeleteBytes(event, "arguments")
		if inputDelta == "" {
			return event, nil
		}
		pending := []byte(`{"type":"response.custom_tool_call_input.delta","item_id":"","output_index":0,"delta":""}`)
		pending, _ = sjson.SetBytes(pending, "item_id", clientItemID)
		if outputIndex := gjson.GetBytes(event, "output_index"); outputIndex.Exists() {
			pending, _ = sjson.SetBytes(pending, "output_index", outputIndex.Int())
		}
		pending, _ = sjson.SetBytes(pending, "delta", inputDelta)
		return event, pending
	}
	if suffix == "delta" {
		delta := gjson.GetBytes(event, "delta")
		if delta.Exists() {
			args := t.customCallArgs[itemID]
			if args == nil {
				args = &strings.Builder{}
				t.customCallArgs[itemID] = args
			}
			args.WriteString(delta.String())
			event, _ = sjson.SetBytes(event, "delta", responsesCustomToolInputDelta(args.String(), state, false))
		}
	}
	return event, nil
}

func (t *ResponsesToolDeclarationTable) normalizeCallReference(rawJSON []byte, itemPath string) []byte {
	prefix := ""
	if itemPath != "" {
		prefix = itemPath + "."
	}
	nameResult := gjson.GetBytes(rawJSON, prefix+"name")
	if !nameResult.Exists() {
		return rawJSON
	}
	name := nameResult.String()
	namespaceResult := gjson.GetBytes(rawJSON, prefix+"namespace")
	namespace := namespaceResult.String()
	kind := responsesCallReferenceKind(gjson.GetBytes(rawJSON, prefix+"type").String())
	declaration, ok := t.findOriginal(namespace, name, kind)
	if !ok {
		return rawJSON
	}
	rawJSON, _ = sjson.SetBytes(rawJSON, prefix+"name", declaration.EffectiveName)
	if namespaceResult.Exists() {
		rawJSON, _ = sjson.DeleteBytes(rawJSON, prefix+"namespace")
	}
	return rawJSON
}

func (t *ResponsesToolDeclarationTable) findOriginal(namespace, name, kind string) (ResponsesToolDeclaration, bool) {
	for _, declaration := range t.declarations {
		if declaration.Namespace != namespace || declaration.Name != name {
			continue
		}
		if kind != "" && responsesDeclarationKind(declaration.Type) != kind {
			continue
		}
		return declaration, true
	}
	return ResponsesToolDeclaration{}, false
}

func normalizeResponsesToolForCodex(tool gjson.Result, namespace string) [][]byte {
	toolType := tool.Get("type").String()
	if strings.TrimSpace(toolType) == "namespace" {
		namespaceName := tool.Get("name").String()
		children := tool.Get("tools")
		if !children.IsArray() {
			return nil
		}
		var out [][]byte
		children.ForEach(func(_, child gjson.Result) bool {
			out = append(out, normalizeResponsesToolForCodex(child, namespaceName)...)
			return true
		})
		return out
	}
	if !responsesCallableToolType(toolType) {
		return [][]byte{[]byte(tool.Raw)}
	}
	name := responsesToolName(tool)
	if strings.TrimSpace(name) == "" {
		return nil
	}
	normalized := []byte(tool.Raw)
	if tool.Get("function").IsObject() {
		normalized = []byte(`{"type":"function","name":"","description":"","parameters":{}}`)
		normalized, _ = sjson.SetBytes(normalized, "name", qualifyResponsesNamespaceToolName(namespace, name))
		if description := responsesToolDescription(tool); description != "" {
			normalized, _ = sjson.SetBytes(normalized, "description", description)
		}
		if parameters := responsesToolParameters(tool); parameters.Exists() {
			normalized, _ = sjson.SetRawBytes(normalized, "parameters", []byte(parameters.Raw))
		}
		return [][]byte{normalized}
	}
	if strings.TrimSpace(toolType) == "" {
		normalized, _ = sjson.SetBytes(normalized, "type", "function")
	}
	normalized, _ = sjson.SetBytes(normalized, "name", qualifyResponsesNamespaceToolName(namespace, name))
	normalized, _ = sjson.DeleteBytes(normalized, "namespace")
	return [][]byte{normalized}
}

func convertResponsesToolToOpenAIChatTools(tool gjson.Result) [][]byte {
	toolType := strings.TrimSpace(tool.Get("type").String())
	switch toolType {
	case "", "function":
		if tJSON, ok := convertResponsesFunctionToolToOpenAIChat(tool, ""); ok {
			return [][]byte{tJSON}
		}
	case "namespace":
		return convertResponsesNamespaceToolToOpenAIChat(tool)
	case "custom":
		if tJSON, ok := convertResponsesCustomToolToOpenAIChat(tool, ""); ok {
			return [][]byte{tJSON}
		}
	}
	return nil
}

func convertResponsesNamespaceToolToOpenAIChat(tool gjson.Result) [][]byte {
	namespaceName := tool.Get("name").String()
	children := tool.Get("tools")
	if !children.IsArray() {
		return nil
	}
	var out [][]byte
	children.ForEach(func(_, child gjson.Result) bool {
		name := qualifyResponsesNamespaceToolName(namespaceName, responsesToolName(child))
		switch strings.TrimSpace(child.Get("type").String()) {
		case "", "function":
			if converted, ok := convertResponsesFunctionToolToOpenAIChat(child, name); ok {
				out = append(out, converted)
			}
		case "custom":
			if converted, ok := convertResponsesCustomToolToOpenAIChat(child, name); ok {
				out = append(out, converted)
			}
		}
		return true
	})
	return out
}

func convertResponsesFunctionToolToOpenAIChat(tool gjson.Result, overrideName string) ([]byte, bool) {
	name := overrideName
	if name == "" {
		name = responsesToolName(tool)
	}
	if strings.TrimSpace(name) == "" {
		return nil, false
	}
	chatTool := []byte(`{"type":"function","function":{"name":"","description":"","parameters":{}}}`)
	chatTool, _ = sjson.SetBytes(chatTool, "function.name", name)
	if description := responsesToolDescription(tool); description != "" {
		chatTool, _ = sjson.SetBytes(chatTool, "function.description", description)
	}
	if parameters := responsesToolParameters(tool); parameters.Exists() {
		chatTool, _ = sjson.SetRawBytes(chatTool, "function.parameters", []byte(parameters.Raw))
	}
	return chatTool, true
}

func convertResponsesCustomToolToOpenAIChat(tool gjson.Result, overrideName string) ([]byte, bool) {
	name := overrideName
	if name == "" {
		name = responsesToolName(tool)
	}
	if strings.TrimSpace(name) == "" {
		return nil, false
	}
	chatTool := []byte(`{"type":"function","function":{"name":"","description":"","parameters":{"type":"object","properties":{"input":{"type":"string"}},"required":["input"]}}}`)
	chatTool, _ = sjson.SetBytes(chatTool, "function.name", name)
	if description := responsesToolDescription(tool); description != "" {
		chatTool, _ = sjson.SetBytes(chatTool, "function.description", description)
	}
	return chatTool, true
}

func responsesToolName(tool gjson.Result) string {
	if name := tool.Get("name"); name.Exists() {
		return name.String()
	}
	return tool.Get("function.name").String()
}

func responsesToolDescription(tool gjson.Result) string {
	if description := tool.Get("description"); description.Exists() {
		return description.String()
	}
	return tool.Get("function.description").String()
}

func responsesToolParameters(tool gjson.Result) gjson.Result {
	for _, path := range []string{"parameters", "parametersJsonSchema", "input_schema", "function.parameters", "function.parametersJsonSchema"} {
		if parameters := tool.Get(path); parameters.Exists() {
			return parameters
		}
	}
	return gjson.Result{}
}

func responsesCustomToolNames(requestRawJSON []byte) map[string]struct{} {
	names := make(map[string]struct{})
	table, err := BuildResponsesToolDeclarationTable(requestRawJSON)
	if err != nil {
		return names
	}
	for _, declaration := range table.declarations {
		if responsesDeclarationKind(declaration.Type) == "custom" {
			names[declaration.EffectiveName] = struct{}{}
		}
	}
	return names
}

func responsesSingleCustomToolName(requestRawJSON []byte) (string, bool) {
	table, err := BuildResponsesToolDeclarationTable(requestRawJSON)
	if err != nil || len(table.declarations) != 1 || responsesDeclarationKind(table.declarations[0].Type) != "custom" {
		return "", false
	}
	return table.declarations[0].EffectiveName, true
}

type responsesCustomToolInputState struct {
	raw       bool
	sent      int
	finalized bool
}

func unwrapCustomToolInput(arguments string) string {
	if input, ok := exactCustomToolInputWrapper(arguments); ok {
		return input
	}
	return arguments
}

func responsesCustomToolInputDelta(arguments string, state *responsesCustomToolInputState, final bool) string {
	if state == nil || state.finalized {
		return ""
	}
	if state.raw {
		if state.sent > len(arguments) {
			state.sent = 0
		}
		delta := arguments[state.sent:]
		state.sent = len(arguments)
		if final {
			state.finalized = true
		}
		return delta
	}
	if final {
		state.finalized = true
		if input, ok := exactCustomToolInputWrapper(arguments); ok {
			return input
		}
		state.raw = true
		state.sent = len(arguments)
		return arguments
	}
	if customToolInputWrapperPrefix(arguments) {
		return ""
	}
	state.raw = true
	state.sent = len(arguments)
	return arguments
}

func exactCustomToolInputWrapper(arguments string) (string, bool) {
	decoder := json.NewDecoder(strings.NewReader(arguments))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') || !decoder.More() {
		return "", false
	}
	key, err := decoder.Token()
	if err != nil || key != "input" {
		return "", false
	}
	var rawInput json.RawMessage
	if err = decoder.Decode(&rawInput); err != nil || decoder.More() {
		return "", false
	}
	trimmedInput := bytes.TrimSpace(rawInput)
	if len(trimmedInput) == 0 || trimmedInput[0] != '"' {
		return "", false
	}
	var input string
	if err = json.Unmarshal(trimmedInput, &input); err != nil {
		return "", false
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return "", false
	}
	if _, err = decoder.Token(); err != io.EOF {
		return "", false
	}
	return input, true
}

func customToolInputWrapperPrefix(arguments string) bool {
	i := skipCustomToolJSONWhitespace(arguments, 0)
	if i >= len(arguments) {
		return true
	}
	if arguments[i] != '{' {
		return false
	}
	i = skipCustomToolJSONWhitespace(arguments, i+1)
	if i >= len(arguments) {
		return true
	}
	keyStart := i
	keyEnd, complete, valid := scanCustomToolJSONString(arguments, i)
	if !valid {
		return false
	}
	if !complete {
		return true
	}
	var key string
	if err := json.Unmarshal([]byte(arguments[keyStart:keyEnd]), &key); err != nil || key != "input" {
		return false
	}
	i = skipCustomToolJSONWhitespace(arguments, keyEnd)
	if i >= len(arguments) {
		return true
	}
	if arguments[i] != ':' {
		return false
	}
	i = skipCustomToolJSONWhitespace(arguments, i+1)
	if i >= len(arguments) {
		return true
	}
	valueEnd, complete, valid := scanCustomToolJSONString(arguments, i)
	if !valid {
		return false
	}
	if !complete {
		return true
	}
	i = skipCustomToolJSONWhitespace(arguments, valueEnd)
	if i >= len(arguments) {
		return true
	}
	if arguments[i] != '}' {
		return false
	}
	i = skipCustomToolJSONWhitespace(arguments, i+1)
	return i == len(arguments)
}

func skipCustomToolJSONWhitespace(value string, start int) int {
	for start < len(value) {
		switch value[start] {
		case ' ', '\t', '\r', '\n':
			start++
		default:
			return start
		}
	}
	return start
}

func scanCustomToolJSONString(value string, start int) (end int, complete bool, valid bool) {
	if start >= len(value) || value[start] != '"' {
		return start, false, false
	}
	for i := start + 1; i < len(value); i++ {
		switch value[i] {
		case '"':
			return i + 1, true, true
		case '\\':
			i++
			if i >= len(value) {
				return len(value), false, true
			}
			switch value[i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				for j := 0; j < 4; j++ {
					i++
					if i >= len(value) {
						return len(value), false, true
					}
					if !strings.ContainsRune("0123456789abcdefABCDEF", rune(value[i])) {
						return i, false, false
					}
				}
			default:
				return i, false, false
			}
		default:
			if value[i] < 0x20 {
				return i, false, false
			}
		}
	}
	return len(value), false, true
}

func qualifyResponsesNamespaceToolName(namespaceName, childName string) string {
	if childName == "" || namespaceName == "" || strings.HasPrefix(childName, "mcp__") {
		return childName
	}
	prefix := namespaceName
	if !strings.HasSuffix(prefix, "__") {
		prefix += "__"
	}
	if strings.HasPrefix(childName, prefix) {
		return childName
	}
	return prefix + childName
}

func pickRequestJSON(originalRequestRawJSON, requestRawJSON []byte) []byte {
	if len(originalRequestRawJSON) > 0 && gjson.ValidBytes(originalRequestRawJSON) {
		return originalRequestRawJSON
	}
	if len(requestRawJSON) > 0 && gjson.ValidBytes(requestRawJSON) {
		return requestRawJSON
	}
	return nil
}

func applyResponsesFunctionCallNamespaceFields(item []byte, requestRawJSON []byte, qualifiedName string, itemPath string) []byte {
	table, err := BuildResponsesToolDeclarationTable(requestRawJSON)
	if err != nil {
		return item
	}
	prefix := ""
	if itemPath != "" {
		prefix = itemPath + "."
	}
	item, _ = sjson.SetBytes(item, prefix+"name", qualifiedName)
	return table.restoreCallItem(item, itemPath)
}

func responsesCallableToolType(toolType string) bool {
	switch strings.TrimSpace(toolType) {
	case "", "function", "custom":
		return true
	default:
		return false
	}
}

func responsesDeclarationKind(toolType string) string {
	if strings.TrimSpace(toolType) == "custom" {
		return "custom"
	}
	return "function"
}

func responsesCallReferenceKind(callType string) string {
	switch strings.TrimSpace(callType) {
	case "custom", "custom_tool_call":
		return "custom"
	case "function", "function_call":
		return "function"
	default:
		return ""
	}
}

func sameResponsesDeclarationIdentity(a, b ResponsesToolDeclaration) bool {
	return a.Namespace == b.Namespace && a.Name == b.Name && a.Type == b.Type
}

func responsesDeclarationIdentity(declaration ResponsesToolDeclaration) string {
	return declaration.Namespace + "\x00" + declaration.Name + "\x00" + declaration.Type
}
