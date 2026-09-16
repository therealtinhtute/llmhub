package responses

import (
	"strings"

	translatorcommon "github.com/therealtinhtute/llmhub/internal/translator/common"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConvertOpenAIResponsesRequestToOpenAIChatCompletions converts OpenAI responses format to OpenAI chat completions format.
// It transforms the OpenAI responses API format (with instructions and input array) into the standard
// OpenAI chat completions format (with messages array and system content).
//
// The conversion handles:
// 1. Model name and streaming configuration
// 2. Instructions to system message conversion
// 3. Input array to messages array transformation
// 4. Tool definitions and tool choice conversion
// 5. Function calls and function results handling
// 6. Generation parameters mapping (max_tokens, reasoning, etc.)
//
// Parameters:
//   - modelName: The name of the model to use for the request
//   - rawJSON: The raw JSON request data in OpenAI responses format
//   - stream: A boolean indicating if the request is for a streaming response
//
// Returns:
//   - []byte: The transformed request data in OpenAI chat completions format
func ConvertOpenAIResponsesRequestToOpenAIChatCompletions(modelName string, inputRawJSON []byte, stream bool) []byte {
	rawJSON := NormalizeResponsesToolsForCodex(inputRawJSON)
	// Base OpenAI chat completions template with default values
	out := []byte(`{"model":"","messages":[],"stream":false}`)

	root := gjson.ParseBytes(rawJSON)

	// Set model name
	out, _ = sjson.SetBytes(out, "model", modelName)

	// Set stream configuration
	out, _ = sjson.SetBytes(out, "stream", stream)

	// Map generation parameters from responses format to chat completions format
	if maxTokens := root.Get("max_output_tokens"); maxTokens.Exists() {
		out, _ = sjson.SetBytes(out, "max_tokens", maxTokens.Int())
	}

	// Convert instructions to system message
	if instructions := root.Get("instructions"); instructions.Exists() {
		systemMessage := []byte(`{"role":"system","content":""}`)
		systemMessage, _ = sjson.SetBytes(systemMessage, "content", instructions.String())
		out, _ = sjson.SetRawBytes(out, "messages.-1", systemMessage)
	}

	// Convert input array to messages.
	// Outputs missing call_id are paired with their pending calls first so the
	// awaiting/orphan bookkeeping below sees resolved identifiers
	// (upstream NormalizeResponsesToolCallOutputs).
	if input := root.Get("input"); input.Exists() && input.IsArray() {
		inputItems := translatorcommon.NormalizeResponsesToolCallOutputs(input.Array())
		outputCallIDs := make(map[string]struct{})
		for _, item := range inputItems {
			itemType := item.Get("type").String()
			if itemType != "function_call_output" && itemType != "custom_tool_call_output" {
				continue
			}
			callID := translatorcommon.ExtractResponsesCallID(item)
			if callID == "" {
				continue
			}
			outputCallIDs[callID] = struct{}{}
		}

		pendingToolCalls := make([]interface{}, 0)
		pendingToolCallIDs := make([]string, 0)
		awaitingToolOutputs := make(map[string]struct{})
		deferredMessages := make([][]byte, 0)

		flushPendingToolCalls := func() {
			if len(pendingToolCalls) == 0 {
				return
			}
			assistantMessage := []byte(`{"role":"assistant","tool_calls":[]}`)
			assistantMessage, _ = sjson.SetBytes(assistantMessage, "tool_calls", pendingToolCalls)
			out, _ = sjson.SetRawBytes(out, "messages.-1", assistantMessage)
			for _, id := range pendingToolCallIDs {
				trimmed := strings.TrimSpace(id)
				if trimmed == "" {
					continue
				}
				awaitingToolOutputs[trimmed] = struct{}{}
			}
			pendingToolCalls = pendingToolCalls[:0]
			pendingToolCallIDs = pendingToolCallIDs[:0]
		}
		flushDeferredMessages := func() {
			for _, message := range deferredMessages {
				out, _ = sjson.SetRawBytes(out, "messages.-1", message)
			}
			deferredMessages = deferredMessages[:0]
		}
		hasAwaitingToolOutput := func() bool {
			for id := range awaitingToolOutputs {
				if _, ok := outputCallIDs[id]; ok {
					return true
				}
			}
			return false
		}
		appendRegularMessage := func(message []byte) {
			// Keep tool-call adjacency strict for providers that require
			// assistant(tool_calls) -> tool(tool_call_id) with no message in between.
			if hasAwaitingToolOutput() {
				deferredMessages = append(deferredMessages, message)
				return
			}
			out, _ = sjson.SetRawBytes(out, "messages.-1", message)
		}

		for _, item := range inputItems {
			itemType := item.Get("type").String()
			if itemType == "" && item.Get("role").String() != "" {
				itemType = "message"
			}
			if itemType != "function_call" {
				flushPendingToolCalls()
			}

			switch itemType {
			case "message", "":
				// Handle regular message conversion
				role := item.Get("role").String()
				if role == "developer" {
					role = "user"
				}
				message := []byte(`{"role":"","content":[]}`)
				message, _ = sjson.SetBytes(message, "role", role)

				if content := item.Get("content"); content.Exists() && content.IsArray() {
					var messageContent string
					var toolCalls []interface{}

					content.ForEach(func(_, contentItem gjson.Result) bool {
						contentType := contentItem.Get("type").String()
						if contentType == "" {
							contentType = "input_text"
						}

						switch contentType {
						case "input_text", "output_text":
							text := contentItem.Get("text").String()
							contentPart := []byte(`{"type":"text","text":""}`)
							contentPart, _ = sjson.SetBytes(contentPart, "text", text)
							message, _ = sjson.SetRawBytes(message, "content.-1", contentPart)
						case "input_image":
							imageURL := contentItem.Get("image_url").String()
							contentPart := []byte(`{"type":"image_url","image_url":{"url":""}}`)
							contentPart, _ = sjson.SetBytes(contentPart, "image_url.url", imageURL)
							message, _ = sjson.SetRawBytes(message, "content.-1", contentPart)
						}
						return true
					})

					if messageContent != "" {
						message, _ = sjson.SetBytes(message, "content", messageContent)
					}

					if len(toolCalls) > 0 {
						message, _ = sjson.SetBytes(message, "tool_calls", toolCalls)
					}
				} else if content.Type == gjson.String {
					message, _ = sjson.SetBytes(message, "content", content.String())
				}

				appendRegularMessage(message)

			case "function_call":
				// Buffer consecutive function calls and emit them as one assistant message.
				toolCall := []byte(`{"id":"","type":"function","function":{"name":"","arguments":""}}`)

				if callId := translatorcommon.ExtractResponsesCallID(item); callId != "" {
					toolCall, _ = sjson.SetBytes(toolCall, "id", callId)
				}

				if name := item.Get("name"); name.Exists() {
					functionName := name.String()
					if namespace := strings.TrimSpace(item.Get("namespace").String()); namespace != "" {
						functionName = qualifyResponsesNamespaceToolName(namespace, functionName)
					} else {
						functionName = canonicalResponsesToolName(inputRawJSON, functionName)
					}
					toolCall, _ = sjson.SetBytes(toolCall, "function.name", functionName)
				}

				if arguments := item.Get("arguments"); arguments.Exists() {
					toolCall, _ = sjson.SetBytes(toolCall, "function.arguments", arguments.String())
				}
				pendingToolCalls = append(pendingToolCalls, gjson.ParseBytes(toolCall).Value())
				if callID := translatorcommon.ExtractResponsesCallID(item); callID != "" {
					pendingToolCallIDs = append(pendingToolCallIDs, callID)
				}

			case "custom_tool_call":
				// Codex freeform tool call replay: wrap the raw input so it
				// matches the {"input": string} function shape used when
				// converting custom tool definitions.
				toolCall := []byte(`{"id":"","type":"function","function":{"name":"","arguments":""}}`)
				toolCall, _ = sjson.SetBytes(toolCall, "id", translatorcommon.ExtractResponsesCallID(item))
				functionName := item.Get("name").String()
				if namespace := item.Get("namespace").String(); namespace != "" {
					functionName = qualifyResponsesNamespaceToolName(namespace, functionName)
				} else {
					functionName = canonicalResponsesToolName(inputRawJSON, functionName)
				}
				toolCall, _ = sjson.SetBytes(toolCall, "function.name", functionName)
				wrappedArgs, _ := sjson.SetBytes([]byte(`{"input":""}`), "input", item.Get("input").String())
				toolCall, _ = sjson.SetBytes(toolCall, "function.arguments", string(wrappedArgs))
				pendingToolCalls = append(pendingToolCalls, gjson.ParseBytes(toolCall).Value())
				if callID := translatorcommon.ExtractResponsesCallID(item); callID != "" {
					pendingToolCallIDs = append(pendingToolCallIDs, callID)
				}

			case "function_call_output":
				callID := translatorcommon.ExtractResponsesCallID(item)
				if _, awaiting := awaitingToolOutputs[callID]; !awaiting {
					// Orphan outputs (empty call_id or no matching assistant
					// tool_calls, e.g. Codex send_message_to_thread cards) must
					// not become tool messages. Emit as user text instead.
					appendStandaloneResponsesToolOutputAsUser(item.Get("output"), setFunctionCallOutputContent, appendRegularMessage)
				} else {
					toolMessage := []byte(`{"role":"tool","tool_call_id":"","content":""}`)
					toolMessage, _ = sjson.SetBytes(toolMessage, "tool_call_id", callID)
					delete(awaitingToolOutputs, callID)
					if output := item.Get("output"); output.Exists() {
						toolMessage = setFunctionCallOutputContent(toolMessage, output)
					}
					out, _ = sjson.SetRawBytes(out, "messages.-1", toolMessage)
				}
				if len(awaitingToolOutputs) == 0 && len(deferredMessages) > 0 {
					flushDeferredMessages()
				}

			case "custom_tool_call_output":
				callID := translatorcommon.ExtractResponsesCallID(item)
				if _, awaiting := awaitingToolOutputs[callID]; !awaiting {
					appendStandaloneResponsesToolOutputAsUser(item.Get("output"), setCustomToolCallOutputContent, appendRegularMessage)
				} else {
					toolMessage := []byte(`{"role":"tool","tool_call_id":"","content":""}`)
					toolMessage, _ = sjson.SetBytes(toolMessage, "tool_call_id", callID)
					delete(awaitingToolOutputs, callID)
					if output := item.Get("output"); output.Exists() {
						toolMessage = setCustomToolCallOutputContent(toolMessage, output)
					}
					out, _ = sjson.SetRawBytes(out, "messages.-1", toolMessage)
				}
				if len(awaitingToolOutputs) == 0 && len(deferredMessages) > 0 {
					flushDeferredMessages()
				}
			}

		}
		flushPendingToolCalls()
		flushDeferredMessages()
	} else if input.Type == gjson.String {
		msg := []byte(`{}`)
		msg, _ = sjson.SetBytes(msg, "role", "user")
		msg, _ = sjson.SetBytes(msg, "content", input.String())
		out, _ = sjson.SetRawBytes(out, "messages.-1", msg)
	}

	// Convert tools from responses format to chat completions format.
	// Codex Desktop can deliver tool definitions through an "additional_tools" input item.
	var chatCompletionsTools []interface{}
	appendChatTools := func(tools gjson.Result) {
		if !tools.Exists() || !tools.IsArray() {
			return
		}
		tools.ForEach(func(_, tool gjson.Result) bool {
			for _, chatTool := range convertResponsesToolToOpenAIChatTools(tool) {
				chatCompletionsTools = append(chatCompletionsTools, gjson.ParseBytes(chatTool).Value())
			}
			return true
		})
	}
	appendChatTools(root.Get("tools"))
	if input := root.Get("input"); input.Exists() && input.IsArray() {
		input.ForEach(func(_, item gjson.Result) bool {
			if item.Get("type").String() == "additional_tools" {
				appendChatTools(item.Get("tools"))
			}
			return true
		})
	}
	if len(chatCompletionsTools) > 0 {
		out, _ = sjson.SetBytes(out, "tools", chatCompletionsTools)
		if parallelToolCalls := root.Get("parallel_tool_calls"); parallelToolCalls.Exists() {
			out, _ = sjson.SetBytes(out, "parallel_tool_calls", parallelToolCalls.Bool())
		}
		if toolChoice := root.Get("tool_choice"); toolChoice.Exists() {
			out, _ = sjson.SetRawBytes(out, "tool_choice", convertResponsesToolChoiceToChatCompletions(toolChoice, inputRawJSON))
		}
	}

	if reasoningEffort := root.Get("reasoning.effort"); reasoningEffort.Exists() {
		effort := strings.ToLower(strings.TrimSpace(reasoningEffort.String()))
		if effort != "" {
			out, _ = sjson.SetBytes(out, "reasoning_effort", effort)
		}
	}

	return out
}

// appendStandaloneResponsesToolOutputAsUser surfaces a tool output that has no
// matching assistant tool_call as plain user text instead of an invalid tool
// message. Empty outputs produce no message.
// Ported from upstream CLIProxyAPI commit 8c984672a66a ("handle orphan function
// outputs as user text"); the setContent hook keeps structured tool output
// parts intact (upstream range diff through v7.3.3).
func appendStandaloneResponsesToolOutputAsUser(output gjson.Result, setContent func([]byte, gjson.Result) []byte, appendMessage func([]byte)) {
	userMessage := []byte(`{"role":"user","content":""}`)
	if output.Exists() {
		userMessage = setContent(userMessage, output)
	}
	content := gjson.GetBytes(userMessage, "content")
	if !content.Exists() {
		return
	}
	if content.Type == gjson.String && strings.TrimSpace(content.String()) == "" {
		return
	}
	if content.IsArray() && !content.Get("0").Exists() {
		return
	}
	appendMessage(userMessage)
}

// setFunctionCallOutputContent writes a function_call_output payload as the
// tool message content, preserving structured parts (including images) when the
// output carries them (upstream openai_openai-responses_request.go).
func setFunctionCallOutputContent(toolMessage []byte, output gjson.Result) []byte {
	structuredContent := output
	if output.Type == gjson.String {
		if !gjson.Valid(output.String()) {
			toolMessage, _ = sjson.SetBytes(toolMessage, "content", output.String())
			return toolMessage
		}
		structuredContent = gjson.Parse(output.String())
	}

	if hasChatToolOutputImagePart(structuredContent) {
		contentItems := make([][]byte, 0, len(structuredContent.Array()))
		for _, item := range structuredContent.Array() {
			contentItems = append(contentItems, chatToolOutputContentPart(item))
		}
		return translatorcommon.SetRawArrayItems(toolMessage, "content", contentItems)
	}

	toolMessage, _ = sjson.SetBytes(toolMessage, "content", output.String())
	return toolMessage
}

// setCustomToolCallOutputContent writes a custom_tool_call_output payload as the
// tool message content; image-bearing outputs share the structured path
// (upstream openai_openai-responses_request.go).
func setCustomToolCallOutputContent(toolMessage []byte, output gjson.Result) []byte {
	structuredContent := output
	if output.Type == gjson.String && gjson.Valid(output.String()) {
		structuredContent = gjson.Parse(output.String())
	}
	if hasChatToolOutputImagePart(structuredContent) {
		return setFunctionCallOutputContent(toolMessage, output)
	}

	toolMessage, _ = sjson.SetBytes(toolMessage, "content", responsesToolOutputText(output))
	return toolMessage
}

func chatToolOutputContentPart(item gjson.Result) []byte {
	itemType := item.Get("type").String()
	switch itemType {
	case "text", "input_text", "output_text":
		part := []byte(`{"type":"text","text":""}`)
		part, _ = sjson.SetBytes(part, "text", item.Get("text").String())
		return part
	case "image_url", "input_image":
		imageURL, detail, ok := chatToolOutputImageFields(item)
		if !ok {
			return chatToolOutputFallbackPart(item)
		}
		part := []byte(`{"type":"image_url","image_url":{"url":""}}`)
		part, _ = sjson.SetBytes(part, "image_url.url", imageURL)
		if detail != "" {
			part, _ = sjson.SetBytes(part, "image_url.detail", detail)
		}
		return part
	default:
		return chatToolOutputFallbackPart(item)
	}
}

func hasChatToolOutputImagePart(content gjson.Result) bool {
	if !content.IsArray() {
		return false
	}

	hasImage := false
	for _, item := range content.Array() {
		itemType := item.Get("type")
		if itemType.Type != gjson.String {
			continue
		}
		switch itemType.String() {
		case "text", "input_text", "output_text":
			if item.Get("text").Type != gjson.String {
				return false
			}
		case "image_url", "input_image":
			if _, _, ok := chatToolOutputImageFields(item); !ok {
				return false
			}
			hasImage = true
		}
	}
	return hasImage
}

func chatToolOutputImageFields(item gjson.Result) (imageURL, detail string, ok bool) {
	var imageURLValue gjson.Result
	var detailValue gjson.Result
	switch item.Get("type").String() {
	case "image_url":
		imageURLValue = item.Get("image_url.url")
		detailValue = item.Get("image_url.detail")
	case "input_image":
		imageURLValue = item.Get("image_url")
		detailValue = item.Get("detail")
	default:
		return "", "", false
	}

	if imageURLValue.Type != gjson.String {
		return "", "", false
	}
	imageURL = strings.TrimSpace(imageURLValue.String())
	if imageURL == "" {
		return "", "", false
	}

	detail, ok = normalizeChatImageDetail(detailValue)
	if !ok {
		return "", "", false
	}
	return imageURL, detail, true
}

func normalizeChatImageDetail(detailValue gjson.Result) (string, bool) {
	if !detailValue.Exists() {
		return "", true
	}
	if detailValue.Type != gjson.String {
		return "", false
	}

	normalizedDetail := strings.ToLower(strings.TrimSpace(detailValue.String()))
	switch normalizedDetail {
	case "auto", "low", "high":
		return normalizedDetail, true
	case "original":
		// Chat Completions does not support Codex's original detail value.
		return "high", true
	default:
		return "", true
	}
}

func chatToolOutputFallbackPart(item gjson.Result) []byte {
	text := item.Raw
	if item.Type == gjson.String || text == "" {
		text = item.String()
	}
	part := []byte(`{"type":"text","text":""}`)
	part, _ = sjson.SetBytes(part, "text", text)
	return part
}

// convertResponsesToolChoiceToChatCompletions flattens a function/custom
// tool_choice object into the Chat Completions {"type":"function","function":
// {"name":...}} shape, qualifying namespaced declarations
// (upstream openai_openai-responses_request.go).
func convertResponsesToolChoiceToChatCompletions(toolChoice gjson.Result, inputRawJSON []byte) []byte {
	if !toolChoice.IsObject() {
		return []byte(toolChoice.Raw)
	}

	choiceType := toolChoice.Get("type").String()
	if choiceType != "function" && choiceType != "custom" {
		return []byte(toolChoice.Raw)
	}

	name := toolChoice.Get("function.name").String()
	if name == "" {
		name = toolChoice.Get("custom.name").String()
	}
	if name == "" {
		name = toolChoice.Get("name").String()
	}
	if name == "" {
		return []byte(toolChoice.Raw)
	}

	namespace := strings.TrimSpace(toolChoice.Get("namespace").String())
	if namespace == "" {
		namespace = strings.TrimSpace(toolChoice.Get("function.namespace").String())
	}
	if namespace == "" {
		namespace = strings.TrimSpace(toolChoice.Get("custom.namespace").String())
	}
	if namespace != "" {
		name = qualifyResponsesNamespaceToolName(namespace, name)
	} else {
		name = canonicalResponsesToolName(inputRawJSON, name)
	}

	converted := []byte(`{"type":"function","function":{"name":""}}`)
	converted, _ = sjson.SetBytes(converted, "function.name", name)
	return converted
}

// canonicalResponsesToolName recovers the emitted Chat Completions name for an
// unqualified call reference: exact emitted names win, a bare local name is
// qualified to its unique declaration, and ambiguous local names stay
// unresolved (upstream canonicalResponsesToolName).
func canonicalResponsesToolName(requestRawJSON []byte, name string) string {
	if strings.TrimSpace(name) == "" {
		return name
	}
	table, err := BuildResponsesToolDeclarationTable(requestRawJSON)
	if err != nil || table == nil {
		return name
	}
	if _, ok := table.byEffective[name]; ok {
		return name
	}
	candidate := ""
	ambiguous := false
	for _, declaration := range table.declarations {
		if declaration.Name != name {
			continue
		}
		if candidate != "" && candidate != declaration.EffectiveName {
			ambiguous = true
		}
		candidate = declaration.EffectiveName
	}
	if candidate != "" && !ambiguous {
		return candidate
	}
	return name
}
