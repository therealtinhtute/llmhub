package responses

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	common "github.com/therealtinhtute/llmhub/internal/translator/common"
	"github.com/therealtinhtute/llmhub/internal/util"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	defaultClaudeResponsesMaxTokens = 32000
	defaultFableResponsesMaxTokens  = 64000
)

// ConvertOpenAIResponsesRequestToClaude transforms an OpenAI Responses API request
// into a Claude Messages API request using only gjson/sjson for JSON handling.
func ConvertOpenAIResponsesRequestToClaude(modelName string, inputRawJSON []byte, stream bool) []byte {
	return convertOpenAIResponsesRequestToClaude(modelName, inputRawJSON, stream)
}

// ConvertOpenAIResponsesRequestToClaudeWithCompat preserves reasoning items
// whose encrypted content is empty for configured compatibility endpoints.
func ConvertOpenAIResponsesRequestToClaudeWithCompat(modelName string, inputRawJSON []byte, stream bool) []byte {
	return convertOpenAIResponsesRequestToClaude(modelName, inputRawJSON, stream)
}

func convertOpenAIResponsesRequestToClaude(modelName string, inputRawJSON []byte, stream bool) []byte {
	rawJSON := inputRawJSON

	userID := common.DeriveClaudeUserID(rawJSON)

	// Base Claude message payload
	out := []byte(`{"model":"","max_tokens":32000,"messages":[],"metadata":{}}`)
	out, _ = sjson.SetBytes(out, "metadata.user_id", userID)
	out, _ = sjson.SetBytes(out, "max_tokens", defaultClaudeResponsesMaxTokensForModel(modelName))

	root := gjson.ParseBytes(rawJSON)

	// Convert OpenAI Responses reasoning.effort to Claude thinking config.
	if v := root.Get("reasoning.effort"); v.Exists() {
		effort := strings.ToLower(strings.TrimSpace(v.String()))
		if effort != "" {
			mi := registry.LookupModelInfo(modelName, "claude")
			supportsAdaptive := mi != nil && mi.Thinking != nil && len(mi.Thinking.Levels) > 0
			supportsMax := supportsAdaptive && thinking.HasLevel(mi.Thinking.Levels, string(thinking.LevelMax))

			if supportsAdaptive {
				switch effort {
				case "none":
					out, _ = sjson.SetBytes(out, "thinking.type", "disabled")
					out, _ = sjson.DeleteBytes(out, "thinking.budget_tokens")
					out, _ = sjson.DeleteBytes(out, "output_config.effort")
				case "auto":
					out, _ = sjson.SetBytes(out, "thinking.type", "adaptive")
					out, _ = sjson.DeleteBytes(out, "thinking.budget_tokens")
					out, _ = sjson.DeleteBytes(out, "output_config.effort")
				default:
					if mapped, ok := thinking.MapToClaudeEffort(effort, supportsMax); ok {
						effort = mapped
					}
					out, _ = sjson.SetBytes(out, "thinking.type", "adaptive")
					out, _ = sjson.DeleteBytes(out, "thinking.budget_tokens")
					out, _ = sjson.SetBytes(out, "output_config.effort", effort)
				}
			} else {
				// Legacy/manual thinking (budget_tokens).
				budget, ok := thinking.ConvertLevelToBudget(effort)
				if ok {
					switch budget {
					case 0:
						out, _ = sjson.SetBytes(out, "thinking.type", "disabled")
					case -1:
						out, _ = sjson.SetBytes(out, "thinking.type", "enabled")
					default:
						if budget > 0 {
							out, _ = sjson.SetBytes(out, "thinking.type", "enabled")
							out, _ = sjson.SetBytes(out, "thinking.budget_tokens", budget)
						}
					}
				}
			}
		}
	}

	// Helper for generating tool call IDs when missing
	genToolCallID := func() string {
		const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		var b strings.Builder
		for range 24 {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
			b.WriteByte(letters[n.Int64()])
		}
		return "toolu_" + b.String()
	}

	// Model
	out, _ = sjson.SetBytes(out, "model", modelName)

	// Max tokens
	if mot := root.Get("max_output_tokens"); mot.Exists() && mot.Type != gjson.Null {
		val := mot.Int()
		if info := registry.LookupModelInfo(modelName, "claude"); info != nil && info.MaxCompletionTokens > 0 && val > int64(info.MaxCompletionTokens) {
			val = int64(info.MaxCompletionTokens)
		}
		out, _ = sjson.SetBytes(out, "max_tokens", val)
	}

	// Service Tier -> Speed
	if st := root.Get("service_tier"); st.Type == gjson.String && st.String() == "priority" {
		out, _ = sjson.SetBytes(out, "speed", "fast")
	}

	// Stream
	out, _ = sjson.SetBytes(out, "stream", stream)

	var messageBlocks [][]byte

	// instructions -> as a leading message (use role user for Claude API compatibility)
	instructionsText := ""
	extractedFromSystem := false
	if instr := root.Get("instructions"); instr.Exists() && instr.Type == gjson.String {
		instructionsText = instr.String()
		if instructionsText != "" {
			sysMsg := []byte(`{"role":"user","content":""}`)
			sysMsg, _ = sjson.SetBytes(sysMsg, "content", instructionsText)
			messageBlocks = append(messageBlocks, sysMsg)
		}
	}

	if instructionsText == "" {
		if input := root.Get("input"); input.Exists() && input.IsArray() {
			input.ForEach(func(_, item gjson.Result) bool {
				if strings.EqualFold(item.Get("role").String(), "system") {
					var builder strings.Builder
					if parts := item.Get("content"); parts.Exists() && parts.IsArray() {
						parts.ForEach(func(_, part gjson.Result) bool {
							textResult := part.Get("text")
							text := textResult.String()
							if builder.Len() > 0 && text != "" {
								builder.WriteByte('\n')
							}
							builder.WriteString(text)
							return true
						})
					} else if parts.Type == gjson.String {
						builder.WriteString(parts.String())
					}
					instructionsText = builder.String()
					if instructionsText != "" {
						sysMsg := []byte(`{"role":"user","content":""}`)
						sysMsg, _ = sjson.SetBytes(sysMsg, "content", instructionsText)
						messageBlocks = append(messageBlocks, sysMsg)
						extractedFromSystem = true
					}
				}
				return instructionsText == ""
			})
		}
	}

	// input array processing with contiguous role grouping
	var pendingRole string
	var pendingParts [][]byte
	var pendingToolUseParts [][]byte

	// Tool-call pairing state. Outputs pair against the raw call id so two
	// distinct ids that sanitize to the same Claude id are not mistaken for
	// the same call.
	emittedToolResults := map[string]struct{}{}
	emittedRawToolUses := map[string]struct{}{}
	unmappedItemTypes := map[string]int{}

	appendMessage := func(msg []byte) {
		messageBlocks = append(messageBlocks, msg)
	}

	flushPendingMessage := func() {
		if pendingRole == "" {
			return
		}

		parts := pendingParts
		if pendingRole == "assistant" && len(pendingToolUseParts) > 0 {
			combined := make([][]byte, 0, len(pendingParts)+len(pendingToolUseParts))
			combined = append(combined, pendingParts...)
			combined = append(combined, pendingToolUseParts...)
			parts = combined
		}
		if len(parts) > 0 {
			msg := []byte(`{"role":"","content":[]}`)
			msg, _ = sjson.SetBytes(msg, "role", pendingRole)
			if len(parts) == 1 {
				part := gjson.ParseBytes(parts[0])
				if part.Get("type").String() == "text" && !part.Get("cache_control").Exists() && !part.Get("citations").Exists() {
					msg, _ = sjson.SetBytes(msg, "content", part.Get("text").String())
				} else {
					msg, _ = sjson.SetRawBytes(msg, "content", common.JoinRawArray(parts))
				}
			} else {
				msg, _ = sjson.SetRawBytes(msg, "content", common.JoinRawArray(parts))
			}
			appendMessage(msg)
		}

		pendingRole = ""
		pendingParts = nil
		pendingToolUseParts = nil
	}

	appendParts := func(role string, parts ...[]byte) {
		if role == "" || len(parts) == 0 {
			return
		}
		if pendingRole != "" && pendingRole != role {
			flushPendingMessage()
		}
		pendingRole = role
		pendingParts = append(pendingParts, parts...)
	}

	appendToolUse := func(toolUse []byte) {
		if len(toolUse) == 0 {
			return
		}
		if pendingRole != "" && pendingRole != "assistant" {
			flushPendingMessage()
		}
		pendingRole = "assistant"
		pendingToolUseParts = append(pendingToolUseParts, toolUse)
	}

	if input := root.Get("input"); input.Exists() && input.IsArray() {
		input.ForEach(func(_, item gjson.Result) bool {
			if extractedFromSystem && strings.EqualFold(item.Get("role").String(), "system") {
				return true
			}
			typ := item.Get("type").String()
			if typ == "" && item.Get("role").String() != "" {
				typ = "message"
			}
			switch typ {
			case "message":
				var role string
				var partsJSON [][]byte
				if parts := item.Get("content"); parts.Exists() && parts.IsArray() {
					parts.ForEach(func(_, part gjson.Result) bool {
						ptype := part.Get("type").String()
						switch ptype {
						case "input_text", "output_text":
							if t := part.Get("text"); t.Exists() {
								txt := t.String()
								contentPart := []byte(`{"type":"text","text":""}`)
								contentPart, _ = sjson.SetBytes(contentPart, "text", txt)
								contentPart = attachClaudeCitations(contentPart, part.Get("annotations"))
								partsJSON = append(partsJSON, contentPart)
							}
							if ptype == "input_text" {
								role = "user"
							} else {
								role = "assistant"
							}
						case "refusal":
							if t := part.Get("refusal"); t.Exists() && t.String() != "" {
								contentPart := []byte(`{"type":"text","text":""}`)
								contentPart, _ = sjson.SetBytes(contentPart, "text", t.String())
								partsJSON = append(partsJSON, contentPart)
							}
							role = "assistant"
						case "input_image":
							url := part.Get("image_url").String()
							if url == "" {
								url = part.Get("url").String()
							}
							if url != "" {
								var contentPart []byte
								if strings.HasPrefix(url, "data:") {
									mediaType, data, ok := parseResponsesBase64DataURL(url)
									if ok {
										contentPart = []byte(`{"type":"image","source":{"type":"base64","media_type":"","data":""}}`)
										contentPart, _ = sjson.SetBytes(contentPart, "source.media_type", mediaType)
										contentPart, _ = sjson.SetBytes(contentPart, "source.data", data)
									}
								} else {
									contentPart = []byte(`{"type":"image","source":{"type":"url","url":""}}`)
									contentPart, _ = sjson.SetBytes(contentPart, "source.url", url)
								}
								if len(contentPart) > 0 {
									partsJSON = append(partsJSON, contentPart)
									if role == "" {
										role = "user"
									}
								}
							}
						case "input_file":
							fileData := part.Get("file_data").String()
							if fileData != "" {
								mediaType := "application/octet-stream"
								data := fileData
								if strings.HasPrefix(fileData, "data:") {
									if m, d, ok := parseResponsesBase64DataURL(fileData); ok {
										mediaType = m
										data = d
									}
								}
								contentPart := []byte(`{"type":"document","source":{"type":"base64","media_type":"","data":""}}`)
								contentPart, _ = sjson.SetBytes(contentPart, "source.media_type", mediaType)
								contentPart, _ = sjson.SetBytes(contentPart, "source.data", data)
								partsJSON = append(partsJSON, contentPart)
								if role == "" {
									role = "user"
								}
							}
						}
						return true
					})
				} else if parts.Type == gjson.String && parts.String() != "" {
					contentPart := []byte(`{"type":"text","text":""}`)
					contentPart, _ = sjson.SetBytes(contentPart, "text", parts.String())
					partsJSON = append(partsJSON, contentPart)
				}

				if role == "" {
					r := item.Get("role").String()
					switch r {
					case "user", "assistant", "system":
						role = r
					default:
						role = "user"
					}
				}

				if len(partsJSON) > 0 {
					appendParts(role, partsJSON...)
				}

			case "web_search_call":
				if blocks := convertResponsesWebSearchCallToClaudeBlocks(item); len(blocks) > 0 {
					appendParts("assistant", blocks...)
				}

			case "reasoning":
				if thinkingPart := convertResponsesReasoningToClaudeThinking(item); len(thinkingPart) > 0 {
					appendParts("assistant", thinkingPart)
				}

			case "function_call":
				rawCallID := item.Get("call_id").String()
				callID := rawCallID
				if callID == "" {
					callID = genToolCallID()
				}
				callID = util.SanitizeClaudeToolID(callID)
				if rawCallID != "" {
					emittedRawToolUses[rawCallID] = struct{}{}
				}
				name := item.Get("name").String()
				argsStr := item.Get("arguments").String()

				toolUse := []byte(`{"type":"tool_use","id":"","name":"","input":{}}`)
				toolUse, _ = sjson.SetBytes(toolUse, "id", callID)
				toolUse, _ = sjson.SetBytes(toolUse, "name", name)
				if argsStr != "" && gjson.Valid(argsStr) {
					argsJSON := gjson.Parse(argsStr)
					if argsJSON.IsObject() {
						toolUse, _ = sjson.SetRawBytes(toolUse, "input", []byte(argsJSON.Raw))
					}
				}

				appendToolUse(toolUse)
			case "function_call_output":
				rawID := item.Get("call_id").String()
				if rawID != "" {
					if _, exists := emittedToolResults[rawID]; exists {
						return true
					}
					emittedToolResults[rawID] = struct{}{}
				}
				output := item.Get("output")
				// Standalone outputs (no call_id, or one that never paired with
				// a function_call in this input) have no tool_use to attach to.
				// Claude rejects orphan tool_result blocks, so surface them as
				// plain user text instead. Pairing is decided on the raw id so
				// two distinct ids that sanitize to the same Claude id are not
				// mistaken for the same call.
				if _, paired := emittedRawToolUses[rawID]; rawID == "" || !paired {
					appendParts("user", convertResponsesStandaloneToolOutputToClaudeText(output)...)
					return true
				}
				callID := util.SanitizeClaudeToolID(rawID)
				toolResult := []byte(`{"type":"tool_result","tool_use_id":"","content":""}`)
				toolResult, _ = sjson.SetBytes(toolResult, "tool_use_id", callID)
				toolResult = applyResponsesToolResultContent(toolResult, output)

				appendParts("user", toolResult)

			default:
				// A new item type means the client gained a capability whose
				// Claude counterpart still has to be decided, so make the gap
				// visible instead of dropping the turn content in silence.
				if typ := item.Get("type").String(); typ != "" {
					unmappedItemTypes[typ]++
				}
			}
			return true
		})
	}
	flushPendingMessage()
	if len(unmappedItemTypes) > 0 {
		log.Warnf("responses->claude: dropped input items of unmapped types %v (model=%s)", unmappedItemTypes, modelName)
	}
	// Answer dangling tool_use blocks so an interrupted turn ends with a
	// synthesized user tool_result instead of an assistant prefill Anthropic
	// rejects.
	messageBlocks = repairClaudeToolPairing(messageBlocks)
	if problems := claudeMessageInvariantProblems(messageBlocks); len(problems) > 0 {
		log.Warnf("responses->claude: message invariants violated after repair (model=%s): %s", modelName, strings.Join(problems, "; "))
	}

	if len(messageBlocks) > 0 {
		out, _ = sjson.SetRawBytes(out, "messages", common.JoinRawArray(messageBlocks))
	}

	includedToolNames := map[string]struct{}{}
	toolNameMap := map[string]string{}

	// tools mapping: parameters -> input_schema
	if tools := root.Get("tools"); tools.Exists() && tools.IsArray() {
		var toolItems [][]byte
		tools.ForEach(func(_, tool gjson.Result) bool {
			convertedTools := convertResponsesToolToClaudeTools(tool, toolNameMap)
			for _, tJSON := range convertedTools {
				toolName := gjson.GetBytes(tJSON, "name").String()
				if toolName != "" {
					includedToolNames[toolName] = struct{}{}
				}
				toolItems = append(toolItems, tJSON)
			}
			return true
		})
		if len(toolItems) > 0 {
			out, _ = sjson.SetRawBytes(out, "tools", common.JoinRawArray(toolItems))
		}
	}

	// Map tool_choice similar to Chat Completions translator (optional in docs, safe to handle)
	if toolChoice := root.Get("tool_choice"); toolChoice.Exists() {
		switch toolChoice.Type {
		case gjson.String:
			switch toolChoice.String() {
			case "auto":
				out, _ = sjson.SetRawBytes(out, "tool_choice", []byte(`{"type":"auto"}`))
			case "none":
				// Leave unset; implies no tools
			case "required":
				if len(includedToolNames) > 0 {
					out, _ = sjson.SetRawBytes(out, "tool_choice", []byte(`{"type":"any"}`))
				}
			}
		case gjson.JSON:
			if toolChoice.Get("type").String() == "function" {
				fn := toolChoice.Get("function.name").String()
				if fn == "" {
					fn = toolChoice.Get("name").String()
				}
				if mappedName := toolNameMap[fn]; mappedName != "" {
					fn = mappedName
				}
				if _, ok := includedToolNames[fn]; ok {
					toolChoiceJSON := []byte(`{"name":"","type":"tool"}`)
					toolChoiceJSON, _ = sjson.SetBytes(toolChoiceJSON, "name", fn)
					out, _ = sjson.SetRawBytes(out, "tool_choice", toolChoiceJSON)
				}
			}
		default:

		}
	}

	return out
}

func defaultClaudeResponsesMaxTokensForModel(modelName string) int {
	maxTokens := defaultClaudeResponsesMaxTokens
	if strings.Contains(strings.ToLower(strings.TrimSpace(modelName)), "fable") {
		maxTokens = defaultFableResponsesMaxTokens
	}
	if info := registry.LookupModelInfo(modelName, "claude"); info != nil && info.MaxCompletionTokens > 0 && info.MaxCompletionTokens < maxTokens {
		return info.MaxCompletionTokens
	}
	return maxTokens
}

func convertResponsesReasoningToClaudeThinking(item gjson.Result) []byte {
	signature := item.Get("encrypted_content").String()
	if signature == "" {
		return nil
	}

	thinkingText := responsesReasoningSummaryText(item)
	if thinkingText == "" {
		thinkingText = responsesReasoningText(item)
	}
	thinkingPart := []byte(`{"type":"thinking","thinking":"","signature":""}`)
	thinkingPart, _ = sjson.SetBytes(thinkingPart, "thinking", thinkingText)
	thinkingPart, _ = sjson.SetBytes(thinkingPart, "signature", signature)
	return thinkingPart
}

func responsesReasoningText(item gjson.Result) string {
	if text := responsesReasoningPartsText(item.Get("summary")); text != "" {
		return text
	}
	return responsesReasoningPartsText(item.Get("content"))
}

func responsesReasoningPartsText(parts gjson.Result) string {
	if !parts.Exists() || !parts.IsArray() {
		return ""
	}
	var builder strings.Builder
	parts.ForEach(func(_, part gjson.Result) bool {
		if text := part.Get("text"); text.Exists() {
			builder.WriteString(text.String())
		} else if part.Type == gjson.String {
			builder.WriteString(part.String())
		}
		return true
	})
	return builder.String()
}

func responsesReasoningSummaryText(item gjson.Result) string {
	var builder strings.Builder
	if summary := item.Get("summary"); summary.Exists() && summary.IsArray() {
		summary.ForEach(func(_, part gjson.Result) bool {
			if text := part.Get("text"); text.Exists() {
				builder.WriteString(text.String())
			} else if part.Type == gjson.String {
				builder.WriteString(part.String())
			}
			return true
		})
	}
	return builder.String()
}

func applyResponsesToolResultContent(toolResult []byte, output gjson.Result) []byte {
	if output.Type == gjson.String {
		toolResult, _ = sjson.SetBytes(toolResult, "content", output.String())
		return toolResult
	}
	if output.IsArray() {
		var partsJSON [][]byte
		valid := true
		output.ForEach(func(_, part gjson.Result) bool {
			partJSON := convertResponsesToolResultContentPartToClaude(part)
			if len(partJSON) == 0 {
				valid = false
				return false
			}
			partsJSON = append(partsJSON, partJSON)
			return true
		})
		if valid {
			toolResult, _ = sjson.SetRawBytes(toolResult, "content", common.JoinRawArray(partsJSON))
			return toolResult
		}
	}
	toolResult, _ = sjson.SetBytes(toolResult, "content", compactResponsesJSONText(output))
	return toolResult
}

func convertResponsesToolResultContentPartToClaude(part gjson.Result) []byte {
	if !part.IsObject() {
		return nil
	}

	switch part.Get("type").String() {
	case "input_text", "output_text":
		if text := part.Get("text"); text.Type == gjson.String {
			contentPart := []byte(`{"type":"text","text":""}`)
			contentPart, _ = sjson.SetBytes(contentPart, "text", text.String())
			return contentPart
		}
	case "input_image":
		url := part.Get("image_url").String()
		if url == "" {
			url = part.Get("url").String()
		}
		if url == "" {
			return nil
		}
		if strings.HasPrefix(url, "data:") {
			mediaType, data, ok := parseResponsesBase64DataURL(url)
			if !ok || !validClaudeImageMediaType(mediaType) {
				return nil
			}
			contentPart := []byte(`{"type":"image","source":{"type":"base64","media_type":"","data":""}}`)
			contentPart, _ = sjson.SetBytes(contentPart, "source.media_type", mediaType)
			contentPart, _ = sjson.SetBytes(contentPart, "source.data", data)
			return contentPart
		}
		contentPart := []byte(`{"type":"image","source":{"type":"url","url":""}}`)
		contentPart, _ = sjson.SetBytes(contentPart, "source.url", url)
		return contentPart
	case "input_file":
		fileData := part.Get("file_data").String()
		if fileData == "" {
			return nil
		}
		mediaType := part.Get("media_type").String()
		data := fileData
		if strings.HasPrefix(fileData, "data:") {
			var ok bool
			mediaType, data, ok = parseResponsesBase64DataURL(fileData)
			if !ok {
				return nil
			}
		} else if mediaType == "" && strings.HasSuffix(strings.ToLower(part.Get("filename").String()), ".pdf") {
			mediaType = "application/pdf"
		}
		if mediaType != "application/pdf" || !validResponsesBase64(data) {
			return nil
		}
		contentPart := []byte(`{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":""}}`)
		contentPart, _ = sjson.SetBytes(contentPart, "source.data", data)
		return contentPart
	case "text":
		if part.Get("text").Type == gjson.String {
			return compactResponsesJSON([]byte(part.Raw))
		}
	case "image":
		if validClaudeToolResultImageSource(part.Get("source")) {
			return compactResponsesJSON([]byte(part.Raw))
		}
	case "document":
		if validClaudeToolResultDocumentSource(part.Get("source")) {
			return compactResponsesJSON([]byte(part.Raw))
		}
	case "search_result":
		if validClaudeToolResultSearchResult(part) {
			return compactResponsesJSON([]byte(part.Raw))
		}
	case "tool_reference":
		if part.Get("tool_name").Type == gjson.String && part.Get("tool_name").String() != "" {
			return compactResponsesJSON([]byte(part.Raw))
		}
	}
	return nil
}

// claudeMessageInvariantProblems reports shapes Anthropic rejects. It only
// describes; repairClaudeToolPairing already fixes the cases it knows about,
// so anything reported here is a new history shape worth investigating from
// the proxy log instead of from an upstream 400.
// Ported from upstream CLIProxyAPI commit 2bcebaa89c98 ("repair tool call
// pairing and handle standalone tool outputs").
func claudeMessageInvariantProblems(messages [][]byte) []string {
	var problems []string
	if len(messages) > 0 && gjson.GetBytes(messages[0], "role").String() != "user" {
		problems = append(problems, "first message is not user")
	}
	for i, msg := range messages {
		role := gjson.GetBytes(msg, "role").String()
		content := gjson.GetBytes(msg, "content")
		switch role {
		case "assistant":
			var toolUseIDs []string
			content.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "tool_use" {
					toolUseIDs = append(toolUseIDs, block.Get("id").String())
				}
				return true
			})
			if len(toolUseIDs) == 0 {
				continue
			}
			if i+1 >= len(messages) || gjson.GetBytes(messages[i+1], "role").String() != "user" {
				problems = append(problems, "messages["+strconv.Itoa(i)+"] tool_use has no following user message")
				continue
			}
			next := gjson.GetBytes(messages[i+1], "content")
			answered := map[string]struct{}{}
			next.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "tool_result" {
					answered[block.Get("tool_use_id").String()] = struct{}{}
				}
				return true
			})
			for _, id := range toolUseIDs {
				if _, ok := answered[id]; !ok {
					problems = append(problems, "messages["+strconv.Itoa(i)+"] tool_use "+id+" has no tool_result in messages["+strconv.Itoa(i+1)+"]")
				}
			}
		case "user":
			var previous gjson.Result
			if i > 0 {
				previous = gjson.GetBytes(messages[i-1], "content")
			}
			toolUses := map[string]struct{}{}
			previous.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "tool_use" {
					toolUses[block.Get("id").String()] = struct{}{}
				}
				return true
			})
			leading := true
			content.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "tool_result" {
					if !leading {
						problems = append(problems, "messages["+strconv.Itoa(i)+"] tool_result after non-tool_result block")
					}
					if _, ok := toolUses[block.Get("tool_use_id").String()]; !ok {
						problems = append(problems, "messages["+strconv.Itoa(i)+"] tool_result "+block.Get("tool_use_id").String()+" has no tool_use in the previous message")
					}
				} else {
					leading = false
				}
				return true
			})
		}
	}
	return problems
}

// repairClaudeToolPairing enforces the Anthropic invariant that every
// assistant tool_use is answered by a tool_result at the start of the very
// next user message, and that every tool_result references a tool_use in the
// immediately preceding assistant message. Histories break it in practice
// when a session dies while a tool runs (tool_use with no output), when
// standalone context is injected ahead of a real tool output (text before
// tool_result), or when a delayed output lands after an intervening
// assistant message. Missing results are synthesized as errors and orphan
// results fold into plain text so the model can carry on.
func repairClaudeToolPairing(messages [][]byte) [][]byte {
	if len(messages) == 0 {
		return messages
	}
	prevToolUseIDs := map[string]struct{}{}
	out := make([][]byte, 0, len(messages)+1)
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		role := gjson.GetBytes(msg, "role").String()
		content := gjson.GetBytes(msg, "content")

		if role == "user" {
			// Fold tool_result blocks that do not answer a tool_use in the
			// immediately preceding assistant message into plain text, and move
			// the real ones ahead of any other content.
			rebuilt, changed := normalizeClaudeToolResultMessage(msg, prevToolUseIDs)
			if changed {
				msg = rebuilt
				content = gjson.GetBytes(msg, "content")
				if i < len(messages) {
					messages[i] = msg
				}
			}
		}

		out = append(out, msg)

		prevToolUseIDs = map[string]struct{}{}
		if role == "assistant" {
			var toolUseIDs []string
			content.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "tool_use" {
					if id := block.Get("id").String(); id != "" {
						toolUseIDs = append(toolUseIDs, id)
						prevToolUseIDs[id] = struct{}{}
					}
				}
				return true
			})
			if len(toolUseIDs) == 0 {
				continue
			}

			hasNextUser := i+1 < len(messages) && gjson.GetBytes(messages[i+1], "role").String() == "user"
			answered := map[string]struct{}{}
			if hasNextUser {
				gjson.GetBytes(messages[i+1], "content").ForEach(func(_, block gjson.Result) bool {
					if block.Get("type").String() == "tool_result" {
						answered[block.Get("tool_use_id").String()] = struct{}{}
					}
					return true
				})
			}

			var synthesized [][]byte
			for _, id := range toolUseIDs {
				if _, ok := answered[id]; ok {
					continue
				}
				part := []byte(`{"type":"tool_result","tool_use_id":"","is_error":true,"content":"Tool call was interrupted before any output was recorded."}`)
				part, _ = sjson.SetBytes(part, "tool_use_id", id)
				synthesized = append(synthesized, part)
			}
			if len(synthesized) == 0 {
				continue
			}

			if hasNextUser {
				// Prepend the missing results so tool_result blocks still lead
				// the existing user message.
				next := messages[i+1]
				nextContent := gjson.GetBytes(next, "content")
				parts := synthesized
				if nextContent.IsArray() {
					nextContent.ForEach(func(_, block gjson.Result) bool {
						parts = append(parts, []byte(block.Raw))
						return true
					})
				} else if nextContent.Type == gjson.String {
					textPart := []byte(`{"type":"text","text":""}`)
					textPart, _ = sjson.SetBytes(textPart, "text", nextContent.String())
					parts = append(parts, textPart)
				}
				userMsg := []byte(`{"role":"user","content":[]}`)
				userMsg, _ = sjson.SetRawBytes(userMsg, "content", common.JoinRawArray(parts))
				messages[i+1] = userMsg
			} else {
				userMsg := []byte(`{"role":"user","content":[]}`)
				userMsg, _ = sjson.SetRawBytes(userMsg, "content", common.JoinRawArray(synthesized))
				out = append(out, userMsg)
			}
		}
	}
	return out
}

// normalizeClaudeToolResultMessage rebuilds a user message so tool_result
// blocks lead the content and any tool_result that does not answer a tool_use
// in the immediately preceding assistant message folds into plain text.
// answeredIDs lists the ids allowed to stay tool_result blocks.
func normalizeClaudeToolResultMessage(msg []byte, answeredIDs map[string]struct{}) ([]byte, bool) {
	content := gjson.GetBytes(msg, "content")
	if !content.IsArray() {
		return msg, false
	}
	var resultParts, otherParts [][]byte
	seenOther := false
	changed := false
	content.ForEach(func(_, block gjson.Result) bool {
		raw := []byte(block.Raw)
		if block.Get("type").String() == "tool_result" {
			if _, ok := answeredIDs[block.Get("tool_use_id").String()]; !ok {
				changed = true
				seenOther = true
				if textParts := toolResultTextParts(block); len(textParts) > 0 {
					otherParts = append(otherParts, textParts...)
				} else {
					// An empty orphan result folds to nothing, so keep an
					// explicit marker instead of producing an empty user
					// message that Anthropic also rejects.
					otherParts = append(otherParts, []byte(`{"type":"text","text":"Tool result was empty."}`))
				}
				return true
			}
			resultParts = append(resultParts, raw)
			if seenOther {
				changed = true
			}
			return true
		}
		seenOther = true
		otherParts = append(otherParts, raw)
		return true
	})
	if !changed {
		return msg, false
	}
	parts := make([][]byte, 0, len(resultParts)+len(otherParts))
	parts = append(parts, resultParts...)
	parts = append(parts, otherParts...)
	userMsg := []byte(`{"role":"user","content":[]}`)
	userMsg, _ = sjson.SetRawBytes(userMsg, "content", common.JoinRawArray(parts))
	return userMsg, true
}

// toolResultTextParts folds an orphan tool_result block into plain text parts
// so it can ride along as ordinary user content. Empty text parts are
// filtered out; when nothing visible remains it returns nil so the caller
// can substitute a marker instead of emitting empty text blocks that
// Anthropic rejects.
func toolResultTextParts(block gjson.Result) [][]byte {
	content := block.Get("content")
	if content.IsArray() {
		var parts [][]byte
		content.ForEach(func(_, part gjson.Result) bool {
			raw := []byte(part.Raw)
			if part.Get("type").String() == "" {
				raw, _ = sjson.SetBytes(raw, "type", "text")
			}
			if !contentPartHasVisibleContent(raw) {
				return true
			}
			parts = append(parts, raw)
			return true
		})
		if len(parts) > 0 {
			return parts
		}
		return nil
	}
	text := content.String()
	if strings.TrimSpace(text) == "" {
		return nil
	}
	part := []byte(`{"type":"text","text":""}`)
	part, _ = sjson.SetBytes(part, "text", text)
	return [][]byte{part}
}

// convertResponsesStandaloneToolOutputToClaudeText renders a tool output that
// has no matching tool_use as ordinary user content blocks.
func convertResponsesStandaloneToolOutputToClaudeText(output gjson.Result) [][]byte {
	if output.Exists() && output.IsArray() {
		var partsJSON [][]byte
		output.ForEach(func(_, part gjson.Result) bool {
			if partJSON := convertResponsesToolResultContentPartToClaude(part); len(partJSON) > 0 {
				// Drop empty text parts so a mixed array never emits blocks
				// Anthropic rejects; images and documents always count.
				if !contentPartHasVisibleContent(partJSON) {
					return true
				}
				partsJSON = append(partsJSON, partJSON)
			}
			return true
		})
		if len(partsJSON) > 0 {
			return partsJSON
		}
	}
	text := output.String()
	if output.IsArray() || strings.TrimSpace(text) == "" {
		// An empty standalone output still needs a block so the user message
		// it joins never degenerates to an empty content array. The IsArray
		// guard keeps the raw array dump from leaking in as literal text when
		// every converted part was empty.
		return [][]byte{[]byte(`{"type":"text","text":"Tool result was empty."}`)}
	}
	contentPart := []byte(`{"type":"text","text":""}`)
	contentPart, _ = sjson.SetBytes(contentPart, "text", text)
	return [][]byte{contentPart}
}

// contentPartHasVisibleContent reports whether a converted Claude content
// block carries user-visible payload (non-empty text, image, or document).
func contentPartHasVisibleContent(part []byte) bool {
	switch gjson.GetBytes(part, "type").String() {
	case "text":
		return strings.TrimSpace(gjson.GetBytes(part, "text").String()) != ""
	default:
		// Non-text blocks (image, document, ...) always carry content.
		return true
	}
}

func parseResponsesBase64DataURL(value string) (string, string, bool) {
	mediaAndData := strings.SplitN(strings.TrimPrefix(value, "data:"), ";base64,", 2)
	if len(mediaAndData) != 2 || mediaAndData[0] == "" || !validResponsesBase64(mediaAndData[1]) {
		return "", "", false
	}
	return mediaAndData[0], mediaAndData[1], true
}

func validResponsesBase64(data string) bool {
	if data == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(data)
	return err == nil
}

func validClaudeImageMediaType(mediaType string) bool {
	switch mediaType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func validClaudeToolResultImageSource(source gjson.Result) bool {
	if !source.IsObject() {
		return false
	}
	switch source.Get("type").String() {
	case "base64":
		return validClaudeImageMediaType(source.Get("media_type").String()) &&
			source.Get("data").Type == gjson.String && validResponsesBase64(source.Get("data").String())
	case "url":
		return source.Get("url").Type == gjson.String && source.Get("url").String() != ""
	default:
		return false
	}
}

func validClaudeToolResultDocumentSource(source gjson.Result) bool {
	if !source.IsObject() {
		return false
	}
	switch source.Get("type").String() {
	case "base64":
		return source.Get("media_type").String() == "application/pdf" &&
			source.Get("data").Type == gjson.String && validResponsesBase64(source.Get("data").String())
	case "url":
		return source.Get("url").Type == gjson.String && source.Get("url").String() != ""
	case "text":
		return source.Get("media_type").String() == "text/plain" && source.Get("data").Type == gjson.String
	case "content":
		content := source.Get("content")
		return content.Type == gjson.String || validClaudeToolResultDocumentContent(content)
	default:
		return false
	}
}

func validClaudeToolResultDocumentContent(content gjson.Result) bool {
	if !content.IsArray() {
		return false
	}
	valid := true
	content.ForEach(func(_, block gjson.Result) bool {
		switch block.Get("type").String() {
		case "text":
			valid = block.IsObject() && block.Get("text").Type == gjson.String
		case "image":
			valid = block.IsObject() && validClaudeToolResultImageSource(block.Get("source"))
		default:
			valid = false
		}
		return valid
	})
	return valid
}

func validClaudeToolResultSearchResult(part gjson.Result) bool {
	return part.Get("source").Type == gjson.String && part.Get("source").String() != "" &&
		part.Get("title").Type == gjson.String && part.Get("title").String() != "" &&
		validClaudeToolResultTextBlocks(part.Get("content"))
}

func validClaudeToolResultTextBlocks(content gjson.Result) bool {
	if !content.IsArray() {
		return false
	}
	valid := true
	content.ForEach(func(_, block gjson.Result) bool {
		valid = block.IsObject() && block.Get("type").String() == "text" && block.Get("text").Type == gjson.String
		return valid
	})
	return valid
}

func compactResponsesJSONText(value gjson.Result) string {
	raw := strings.TrimSpace(value.Raw)
	if raw == "" {
		return value.String()
	}
	return string(compactResponsesJSON([]byte(raw)))
}

func compactResponsesJSON(raw []byte) []byte {
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return raw
	}
	return compact.Bytes()
}

func convertResponsesToolToClaudeTools(tool gjson.Result, toolNameMap map[string]string) [][]byte {
	toolType := strings.TrimSpace(tool.Get("type").String())
	switch toolType {
	case "", "function":
		if tJSON, ok := convertResponsesFunctionToolToClaude(tool, ""); ok {
			return [][]byte{tJSON}
		}
	case "namespace":
		return convertResponsesNamespaceToolToClaude(tool, toolNameMap)
	case "web_search":
		if tJSON, ok := convertResponsesWebSearchToolToClaude(tool); ok {
			if name := gjson.GetBytes(tJSON, "name").String(); name != "" {
				toolNameMap[name] = name
			}
			return [][]byte{tJSON}
		}
	default:
		if isUnsupportedOpenAIBuiltinToolType(toolType) {
			return nil
		}
		if tool.Get("name").String() != "" {
			return [][]byte{[]byte(tool.Raw)}
		}
	}
	return nil
}

func convertResponsesNamespaceToolToClaude(tool gjson.Result, toolNameMap map[string]string) [][]byte {
	namespaceName := strings.TrimSpace(tool.Get("name").String())
	children := tool.Get("tools")
	if !children.Exists() || !children.IsArray() {
		return nil
	}

	var out [][]byte
	children.ForEach(func(_, child gjson.Result) bool {
		childName := responsesToolName(child)
		qualifiedName := qualifyResponsesNamespaceToolName(namespaceName, childName)
		if tJSON, ok := convertResponsesFunctionToolToClaude(child, qualifiedName); ok {
			out = append(out, tJSON)
			toolNameMap[qualifiedName] = qualifiedName
			if childName != "" {
				toolNameMap[childName] = qualifiedName
			}
		}
		return true
	})
	return out
}

func convertResponsesFunctionToolToClaude(tool gjson.Result, overrideName string) ([]byte, bool) {
	name := strings.TrimSpace(overrideName)
	if name == "" {
		name = responsesToolName(tool)
	}
	if name == "" {
		return nil, false
	}

	tJSON := []byte(`{"name":"","description":"","input_schema":{}}`)
	tJSON, _ = sjson.SetBytes(tJSON, "name", name)
	if d := responsesToolDescription(tool); d != "" {
		tJSON, _ = sjson.SetBytes(tJSON, "description", d)
	}
	tJSON, _ = sjson.SetRawBytes(tJSON, "input_schema", normalizeClaudeToolInputSchema(responsesToolParameters(tool)))
	return tJSON, true
}

func convertResponsesWebSearchToolToClaude(tool gjson.Result) ([]byte, bool) {
	if externalWebAccess := tool.Get("external_web_access"); externalWebAccess.Exists() && !externalWebAccess.Bool() {
		return nil, false
	}

	name := strings.TrimSpace(tool.Get("name").String())
	if name == "" {
		name = "web_search"
	}
	tJSON := []byte(`{"type":"web_search_20250305","name":""}`)
	tJSON, _ = sjson.SetBytes(tJSON, "name", name)
	if maxUses := tool.Get("max_uses"); maxUses.Exists() {
		tJSON, _ = sjson.SetBytes(tJSON, "max_uses", maxUses.Int())
	}
	if allowedDomains := tool.Get("filters.allowed_domains"); allowedDomains.Exists() && allowedDomains.IsArray() {
		tJSON, _ = sjson.SetRawBytes(tJSON, "allowed_domains", []byte(allowedDomains.Raw))
	}
	if userLocation := tool.Get("user_location"); userLocation.Exists() && userLocation.IsObject() {
		tJSON, _ = sjson.SetRawBytes(tJSON, "user_location", []byte(userLocation.Raw))
	}
	return tJSON, true
}

func responsesToolName(tool gjson.Result) string {
	if name := strings.TrimSpace(tool.Get("name").String()); name != "" {
		return name
	}
	return strings.TrimSpace(tool.Get("function.name").String())
}

func responsesToolDescription(tool gjson.Result) string {
	if description := tool.Get("description").String(); description != "" {
		return description
	}
	return tool.Get("function.description").String()
}

func responsesToolParameters(tool gjson.Result) gjson.Result {
	for _, path := range []string{
		"parameters",
		"parametersJsonSchema",
		"input_schema",
		"function.parameters",
		"function.parametersJsonSchema",
	} {
		if parameters := tool.Get(path); parameters.Exists() {
			return parameters
		}
	}
	return gjson.Result{}
}

func normalizeClaudeToolInputSchema(parameters gjson.Result) []byte {
	raw := strings.TrimSpace(parameters.Raw)
	if raw == "" || raw == "null" || !gjson.Valid(raw) {
		return []byte(`{"type":"object","properties":{}}`)
	}
	result := gjson.Parse(raw)
	if !result.IsObject() {
		return []byte(`{"type":"object","properties":{}}`)
	}
	schema := []byte(raw)
	schemaType := result.Get("type").String()
	if schemaType == "" {
		schema, _ = sjson.SetBytes(schema, "type", "object")
		schemaType = "object"
	}
	if schemaType == "object" && !result.Get("properties").Exists() {
		schema, _ = sjson.SetRawBytes(schema, "properties", []byte(`{}`))
	}
	return schema
}

func qualifyResponsesNamespaceToolName(namespaceName, childName string) string {
	childName = strings.TrimSpace(childName)
	if childName == "" || namespaceName == "" || strings.HasPrefix(childName, "mcp__") {
		return childName
	}
	if strings.HasPrefix(childName, namespaceName) {
		return childName
	}
	if strings.HasSuffix(namespaceName, "__") {
		return namespaceName + childName
	}
	return namespaceName + "__" + childName
}

func isUnsupportedOpenAIBuiltinToolType(toolType string) bool {
	switch toolType {
	case "image_generation", "file_search", "code_interpreter", "computer_use_preview":
		return true
	default:
		return false
	}
}
