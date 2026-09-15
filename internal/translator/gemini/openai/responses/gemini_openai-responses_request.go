package responses

import (
	"strings"

	translatorcommon "github.com/therealtinhtute/llmhub/internal/translator/common"
	"github.com/therealtinhtute/llmhub/internal/translator/gemini/common"
	"github.com/therealtinhtute/llmhub/internal/util"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const geminiResponsesThoughtSignature = "skip_thought_signature_validator"

func ConvertOpenAIResponsesRequestToGemini(modelName string, inputRawJSON []byte, stream bool) []byte {
	rawJSON := inputRawJSON

	// Note: modelName and stream parameters are part of the fixed method signature
	_ = modelName // Unused but required by interface
	_ = stream    // Unused but required by interface

	// Base Gemini API template (do not include thinkingConfig by default)
	out := []byte(`{"contents":[]}`)

	root := gjson.ParseBytes(rawJSON)

	// Extract system instruction from OpenAI "instructions" field
	if instructions := root.Get("instructions"); instructions.Exists() {
		systemInstr := []byte(`{"parts":[{"text":""}]}`)
		systemInstr, _ = sjson.SetBytes(systemInstr, "parts.0.text", instructions.String())
		out, _ = sjson.SetRawBytes(out, "systemInstruction", systemInstr)
	}

	// Convert input messages to Gemini contents format
	if input := root.Get("input"); input.Exists() && input.IsArray() {
		items := input.Array()

		// Normalize consecutive function calls and outputs so each call is immediately followed by its response
		normalized := make([]gjson.Result, 0, len(items))
		contentItems := make([][]byte, 0, len(items))
		// pendingFunctionCallIDs tracks function calls whose outputs have not been
		// emitted yet. pendingDeveloperParts buffers mid-session developer/system
		// notices while calls are pending so they land after the functionResponse.
		// Ported from upstream CLIProxyAPI commit 0fe19ede90a4 ("preserve Gemini
		// prompt cache by demoting mid-session developer messages").
		pendingFunctionCallIDs := make([]string, 0)
		var pendingDeveloperParts [][]byte
		hasEncounteredConversation := false
		for i := 0; i < len(items); {
			item := items[i]
			itemType := item.Get("type").String()
			itemRole := item.Get("role").String()
			if itemType == "" && itemRole != "" {
				itemType = "message"
			}

			if itemType == "function_call" {
				var calls []gjson.Result
				var outputs []gjson.Result

				for i < len(items) {
					next := items[i]
					nextType := next.Get("type").String()
					nextRole := next.Get("role").String()
					if nextType == "" && nextRole != "" {
						nextType = "message"
					}
					if nextType != "function_call" {
						break
					}
					calls = append(calls, next)
					i++
				}

				for i < len(items) {
					next := items[i]
					nextType := next.Get("type").String()
					nextRole := next.Get("role").String()
					if nextType == "" && nextRole != "" {
						nextType = "message"
					}
					if nextType != "function_call_output" {
						break
					}
					outputs = append(outputs, next)
					i++
				}

				if len(calls) > 0 {
					outputMap := make(map[string]gjson.Result, len(outputs))
					for _, outItem := range outputs {
						outputMap[outItem.Get("call_id").String()] = outItem
					}
					for _, call := range calls {
						normalized = append(normalized, call)
						callID := call.Get("call_id").String()
						if resp, ok := outputMap[callID]; ok {
							normalized = append(normalized, resp)
							delete(outputMap, callID)
						}
					}
					for _, outItem := range outputs {
						if _, ok := outputMap[outItem.Get("call_id").String()]; ok {
							normalized = append(normalized, outItem)
						}
					}
					continue
				}
			}

			if itemType == "function_call_output" {
				normalized = append(normalized, item)
				i++
				continue
			}

			normalized = append(normalized, item)
			i++
		}

		for i, item := range normalized {
			itemType := item.Get("type").String()
			itemRole := item.Get("role").String()
			if itemType == "" && itemRole != "" {
				itemType = "message"
			}

			switch itemType {
			case "message":
				if strings.EqualFold(itemRole, "system") || strings.EqualFold(itemRole, "developer") {
					// Leading system/developer messages hoist into
					// systemInstruction, keeping the token-0 prompt-cache
					// prefix immutable. Mid-session ones demote to user turns.
					// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
					if !hasEncounteredConversation {
						pendingFunctionCallIDs = nil
						if contentArray := item.Get("content"); contentArray.Exists() {
							systemInstr := []byte(`{"parts":[]}`)
							if systemInstructionResult := gjson.GetBytes(out, "systemInstruction"); systemInstructionResult.Exists() {
								systemInstr = []byte(systemInstructionResult.Raw)
							}

							if contentArray.IsArray() {
								contentArray.ForEach(func(_, contentItem gjson.Result) bool {
									part := []byte(`{"text":""}`)
									text := contentItem.Get("text").String()
									part, _ = sjson.SetBytes(part, "text", text)
									systemInstr, _ = sjson.SetRawBytes(systemInstr, "parts.-1", part)
									return true
								})
							} else if contentArray.Type == gjson.String {
								part := []byte(`{"text":""}`)
								part, _ = sjson.SetBytes(part, "text", contentArray.String())
								systemInstr, _ = sjson.SetRawBytes(systemInstr, "parts.-1", part)
							}

							if gjson.GetBytes(systemInstr, "parts.#").Int() > 0 {
								out, _ = sjson.SetRawBytes(out, "systemInstruction", systemInstr)
							}
						}
						continue
					}

					var devParts [][]byte
					if contentArray := item.Get("content"); contentArray.Exists() {
						if contentArray.IsArray() {
							contentArray.ForEach(func(_, contentItem gjson.Result) bool {
								text := contentItem.Get("text").String()
								if text != "" {
									part := []byte(`{"text":""}`)
									part, _ = sjson.SetBytes(part, "text", text)
									devParts = append(devParts, part)
								}
								return true
							})
						} else if contentArray.Type == gjson.String && contentArray.String() != "" {
							part := []byte(`{"text":""}`)
							part, _ = sjson.SetBytes(part, "text", contentArray.String())
							devParts = append(devParts, part)
						}
					}
					if len(devParts) > 0 {
						if len(pendingFunctionCallIDs) > 0 {
							pendingDeveloperParts = append(pendingDeveloperParts, devParts...)
						} else {
							contentItems = append(contentItems, geminiResponsesContent("user", devParts))
						}
					}
					continue
				}

				hasEncounteredConversation = true
				if _, isAssistantOutput := openAIResponsesAssistantVisibleText(item); !isAssistantOutput {
					if len(pendingFunctionCallIDs) > 0 {
						anyHasFutureOutput := false
						for _, callID := range pendingFunctionCallIDs {
							if openAIResponsesHasMatchingOutput(normalized[i:], callID) {
								anyHasFutureOutput = true
								break
							}
						}
						if !anyHasFutureOutput {
							pendingFunctionCallIDs = nil
						}
					}
					// Flush buffered developer notices before the intervening
					// user turn so ordering stays call -> notice -> user -> response.
					// Ported from upstream CLIProxyAPI commit e56fae88c0ac
					// ("flush pending developer notice before intervening user turn").
					if len(pendingDeveloperParts) > 0 {
						contentItems = append(contentItems, geminiResponsesContent("user", pendingDeveloperParts))
						pendingDeveloperParts = nil
					}
				}

				// Handle regular messages
				// Note: In Responses format, model outputs may appear as content items with type "output_text"
				// even when the message.role is "user". We split such items into distinct Gemini messages
				// with roles derived from the content type to match docs/convert-2.md.
				if contentArray := item.Get("content"); contentArray.Exists() && contentArray.IsArray() {
					currentRole := ""
					currentParts := make([][]byte, 0)

					flush := func() {
						if currentRole == "" || len(currentParts) == 0 {
							currentParts = currentParts[:0]
							return
						}
						contentItems = append(contentItems, geminiResponsesContent(currentRole, currentParts))
						currentParts = currentParts[:0]
					}

					contentArray.ForEach(func(_, contentItem gjson.Result) bool {
						contentType := contentItem.Get("type").String()
						if contentType == "" {
							contentType = "input_text"
						}

						effRole := "user"
						if itemRole != "" {
							switch strings.ToLower(itemRole) {
							case "assistant", "model":
								effRole = "model"
							default:
								effRole = strings.ToLower(itemRole)
							}
						}
						if contentType == "output_text" {
							effRole = "model"
						}
						if effRole == "assistant" {
							effRole = "model"
						}

						if currentRole != "" && effRole != currentRole {
							flush()
							currentRole = ""
						}
						if currentRole == "" {
							currentRole = effRole
						}

						var partJSON []byte
						switch contentType {
						case "input_text", "output_text":
							if text := contentItem.Get("text"); text.Exists() {
								partJSON = []byte(`{"text":""}`)
								partJSON, _ = sjson.SetBytes(partJSON, "text", text.String())
							}
						case "input_image":
							imageURL := contentItem.Get("image_url").String()
							if imageURL == "" {
								imageURL = contentItem.Get("url").String()
							}
							if imageURL != "" {
								mimeType, data := parseOpenAIResponsesDataURL(imageURL)
								if data != "" {
									partJSON = geminiResponsesInlineDataPart(mimeType, data)
								}
							}
						case "input_audio":
							audioData := contentItem.Get("data").String()
							audioFormat := contentItem.Get("format").String()
							if audioData != "" {
								audioMimeMap := map[string]string{
									"mp3":       "audio/mpeg",
									"wav":       "audio/wav",
									"ogg":       "audio/ogg",
									"flac":      "audio/flac",
									"aac":       "audio/aac",
									"webm":      "audio/webm",
									"pcm16":     "audio/pcm",
									"g711_ulaw": "audio/basic",
									"g711_alaw": "audio/basic",
								}
								mimeType := "audio/wav"
								if audioFormat != "" {
									if mapped, ok := audioMimeMap[audioFormat]; ok {
										mimeType = mapped
									} else {
										mimeType = "audio/" + audioFormat
									}
								}
								partJSON = []byte(`{"inline_data":{"mime_type":"","data":""}}`)
								partJSON, _ = sjson.SetBytes(partJSON, "inline_data.mime_type", mimeType)
								partJSON, _ = sjson.SetBytes(partJSON, "inline_data.data", audioData)
							}
						}

						if len(partJSON) > 0 {
							currentParts = append(currentParts, partJSON)
						}
						return true
					})

					flush()
				} else if contentArray.Type == gjson.String {
					effRole := "user"
					if itemRole != "" {
						switch strings.ToLower(itemRole) {
						case "assistant", "model":
							effRole = "model"
						default:
							effRole = strings.ToLower(itemRole)
						}
					}

					one := []byte(`{"role":"","parts":[{"text":""}]}`)
					one, _ = sjson.SetBytes(one, "role", effRole)
					one, _ = sjson.SetBytes(one, "parts.0.text", contentArray.String())
					contentItems = append(contentItems, one)
				}

			case "function_call":
				hasEncounteredConversation = true
				// Handle function calls - convert to model message with functionCall
				name := util.SanitizeFunctionName(item.Get("name").String())
				arguments := item.Get("arguments").String()

				modelContent := []byte(`{"role":"model","parts":[]}`)
				functionCall := []byte(`{"functionCall":{"name":"","args":{}}}`)
				functionCall, _ = sjson.SetBytes(functionCall, "functionCall.name", name)
				functionCall, _ = sjson.SetBytes(functionCall, "thoughtSignature", geminiResponsesThoughtSignature)
				functionCall, _ = sjson.SetBytes(functionCall, "functionCall.id", item.Get("call_id").String())

				// Parse arguments JSON string and set as args object
				if arguments != "" {
					argsResult := gjson.Parse(arguments)
					functionCall, _ = sjson.SetRawBytes(functionCall, "functionCall.args", []byte(argsResult.Raw))
				}

				modelContent, _ = sjson.SetRawBytes(modelContent, "parts.-1", functionCall)
				contentItems = append(contentItems, modelContent)
				if callID := item.Get("call_id").String(); callID != "" {
					pendingFunctionCallIDs = append(pendingFunctionCallIDs, callID)
				}

			case "function_call_output":
				hasEncounteredConversation = true
				// Handle function call outputs - convert to function message with functionResponse
				callID := item.Get("call_id").String()

				// Find the corresponding function call by matching call_id.
				// Orphan outputs (empty call_id or no matching function_call,
				// e.g. Codex send_message_to_thread cards) must not become
				// unpaired functionResponse parts; surface them as user text.
				functionName := ""
				matchedCall := false
				if callID != "" {
					if inputArray := root.Get("input"); inputArray.Exists() && inputArray.IsArray() {
						inputArray.ForEach(func(_, prevItem gjson.Result) bool {
							if prevItem.Get("type").String() == "function_call" && prevItem.Get("call_id").String() == callID {
								functionName = prevItem.Get("name").String()
								matchedCall = true
								return false // Stop iteration
							}
							return true
						})
					}
				}
				if !matchedCall {
					if parts := buildOpenAIResponsesStandaloneToolOutputTextParts(item); len(parts) > 0 {
						userContent := []byte(`{"role":"user","parts":[]}`)
						for _, part := range parts {
							userContent, _ = sjson.SetRawBytes(userContent, "parts.-1", part)
						}
						contentItems = append(contentItems, userContent)
					}
				} else {
					if functionName == "" {
						functionName = "unknown"
					}
					functionName = util.SanitizeFunctionName(functionName)

					functionContent := []byte(`{"role":"user","parts":[]}`)
					functionResponse := buildOpenAIResponsesFunctionResponsePart(functionName, callID, item.Get("output"))
					functionContent, _ = sjson.SetRawBytes(functionContent, "parts.-1", functionResponse)
					contentItems = append(contentItems, functionContent)

					// This call is resolved; drop it from the pending set.
					if len(pendingFunctionCallIDs) > 0 {
						stillPending := make([]string, 0, len(pendingFunctionCallIDs))
						for _, pendingID := range pendingFunctionCallIDs {
							if pendingID != callID {
								stillPending = append(stillPending, pendingID)
							}
						}
						pendingFunctionCallIDs = stillPending
					}
				}
				// Emit buffered developer notices once the pending calls have
				// been answered so they land after the functionResponse turn.
				// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
				if len(pendingFunctionCallIDs) == 0 && len(pendingDeveloperParts) > 0 {
					contentItems = append(contentItems, geminiResponsesContent("user", pendingDeveloperParts))
					pendingDeveloperParts = nil
				}

			case "reasoning":
				hasEncounteredConversation = true
				thoughtContent := []byte(`{"role":"model","parts":[]}`)
				thought := []byte(`{"text":"","thoughtSignature":"","thought":true}`)
				thought, _ = sjson.SetBytes(thought, "text", item.Get("summary.0.text").String())
				thought, _ = sjson.SetBytes(thought, "thoughtSignature", item.Get("encrypted_content").String())

				thoughtContent, _ = sjson.SetRawBytes(thoughtContent, "parts.-1", thought)
				contentItems = append(contentItems, thoughtContent)
			}
		}
		// Flush any developer notices still buffered behind unanswered calls.
		// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
		if len(pendingDeveloperParts) > 0 {
			contentItems = append(contentItems, geminiResponsesContent("user", pendingDeveloperParts))
			pendingDeveloperParts = nil
		}
		// Merge adjacent user turns (demoted developer notices merge into
		// neighboring user turns) without crossing functionResponse boundaries.
		// Ported from upstream CLIProxyAPI commit 0fe19ede90a4.
		contentItems = translatorcommon.MergeAdjacentGeminiUserContents(contentItems)
		out = translatorcommon.SetRawArrayItems(out, "contents", contentItems)
	} else if input.Exists() && input.Type == gjson.String {
		// Simple string input conversion to user message
		userContent := []byte(`{"role":"user","parts":[{"text":""}]}`)
		userContent, _ = sjson.SetBytes(userContent, "parts.0.text", input.String())
		out, _ = sjson.SetRawBytes(out, "contents.-1", userContent)
	}

	// Convert tools to Gemini functionDeclarations format
	if tools := root.Get("tools"); tools.Exists() && tools.IsArray() {
		geminiTools := []byte(`[{"functionDeclarations":[]}]`)

		tools.ForEach(func(_, tool gjson.Result) bool {
			if tool.Get("type").String() == "function" {
				funcDecl := []byte(`{"name":"","description":"","parametersJsonSchema":{}}`)

				if name := tool.Get("name"); name.Exists() {
					funcDecl, _ = sjson.SetBytes(funcDecl, "name", util.SanitizeFunctionName(name.String()))
				}
				if desc := tool.Get("description"); desc.Exists() {
					funcDecl, _ = sjson.SetBytes(funcDecl, "description", desc.String())
				}
				if params := tool.Get("parameters"); params.Exists() {
					funcDecl, _ = sjson.SetRawBytes(funcDecl, "parametersJsonSchema", []byte(params.Raw))
				}

				geminiTools, _ = sjson.SetRawBytes(geminiTools, "0.functionDeclarations.-1", funcDecl)
			}
			return true
		})

		// Only add tools if there are function declarations
		if funcDecls := gjson.GetBytes(geminiTools, "0.functionDeclarations"); funcDecls.Exists() && len(funcDecls.Array()) > 0 {
			out, _ = sjson.SetRawBytes(out, "tools", geminiTools)
		}
	}

	// Handle generation config from OpenAI format
	if maxOutputTokens := root.Get("max_output_tokens"); maxOutputTokens.Exists() {
		genConfig := []byte(`{"maxOutputTokens":0}`)
		genConfig, _ = sjson.SetBytes(genConfig, "maxOutputTokens", maxOutputTokens.Int())
		out, _ = sjson.SetRawBytes(out, "generationConfig", genConfig)
	}

	// Handle temperature if present
	if temperature := root.Get("temperature"); temperature.Exists() {
		if !gjson.GetBytes(out, "generationConfig").Exists() {
			out, _ = sjson.SetRawBytes(out, "generationConfig", []byte(`{}`))
		}
		out, _ = sjson.SetBytes(out, "generationConfig.temperature", temperature.Float())
	}

	// Handle top_p if present
	if topP := root.Get("top_p"); topP.Exists() {
		if !gjson.GetBytes(out, "generationConfig").Exists() {
			out, _ = sjson.SetRawBytes(out, "generationConfig", []byte(`{}`))
		}
		out, _ = sjson.SetBytes(out, "generationConfig.topP", topP.Float())
	}

	// Handle stop sequences
	if stopSequences := root.Get("stop_sequences"); stopSequences.Exists() && stopSequences.IsArray() {
		if !gjson.GetBytes(out, "generationConfig").Exists() {
			out, _ = sjson.SetRawBytes(out, "generationConfig", []byte(`{}`))
		}
		var sequences []string
		stopSequences.ForEach(func(_, seq gjson.Result) bool {
			sequences = append(sequences, seq.String())
			return true
		})
		out, _ = sjson.SetBytes(out, "generationConfig.stopSequences", sequences)
	}

	// Apply thinking configuration: convert OpenAI Responses API reasoning.effort to Gemini thinkingConfig.
	// Inline translation-only mapping; capability checks happen later in ApplyThinking.
	re := root.Get("reasoning.effort")
	if re.Exists() {
		effort := strings.ToLower(strings.TrimSpace(re.String()))
		if effort != "" {
			thinkingPath := "generationConfig.thinkingConfig"
			if effort == "auto" {
				out, _ = sjson.SetBytes(out, thinkingPath+".thinkingBudget", -1)
				out, _ = sjson.SetBytes(out, thinkingPath+".includeThoughts", true)
			} else {
				out, _ = sjson.SetBytes(out, thinkingPath+".thinkingLevel", effort)
				out, _ = sjson.SetBytes(out, thinkingPath+".includeThoughts", effort != "none")
			}
		}
	}

	result := out
	result = common.AttachDefaultSafetySettings(result, "safetySettings")
	return result
}

// buildOpenAIResponsesStandaloneToolOutputTextParts renders a tool output that
// has no matching function_call as plain Gemini user text parts. Empty outputs
// produce no parts.
// Ported from upstream CLIProxyAPI commit 8c984672a66a ("handle orphan function
// outputs as user text").
func buildOpenAIResponsesStandaloneToolOutputTextParts(item gjson.Result) [][]byte {
	output := item.Get("output")
	if !output.Exists() {
		return nil
	}
	if output.IsArray() {
		var parts [][]byte
		output.ForEach(func(_, part gjson.Result) bool {
			text := part.Get("text").String()
			if strings.TrimSpace(text) == "" {
				return true
			}
			textPart := []byte(`{"text":""}`)
			textPart, _ = sjson.SetBytes(textPart, "text", text)
			parts = append(parts, textPart)
			return true
		})
		return parts
	}
	text := output.String()
	if strings.TrimSpace(text) == "" {
		return nil
	}
	textPart := []byte(`{"text":""}`)
	textPart, _ = sjson.SetBytes(textPart, "text", text)
	return [][]byte{textPart}
}

// geminiResponsesInlineDataPart builds a snake_case inline_data part for a
// Gemini content part.
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go).
func geminiResponsesInlineDataPart(mimeType, data string) []byte {
	partJSON := []byte(`{"inline_data":{"mime_type":"","data":""}}`)
	partJSON, _ = sjson.SetBytes(partJSON, "inline_data.mime_type", mimeType)
	partJSON, _ = sjson.SetBytes(partJSON, "inline_data.data", data)
	return partJSON
}

// parseOpenAIResponsesDataURL extracts (mimeType, base64 data) from a data: URL.
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go).
func parseOpenAIResponsesDataURL(imageURL string) (string, string) {
	mimeType := "application/octet-stream"
	data := ""
	if strings.HasPrefix(imageURL, "data:") {
		trimmed := strings.TrimPrefix(imageURL, "data:")
		mediaAndData := strings.SplitN(trimmed, ";base64,", 2)
		if len(mediaAndData) == 2 {
			if mediaAndData[0] != "" {
				mimeType = mediaAndData[0]
			}
			data = mediaAndData[1]
		} else {
			mediaAndData = strings.SplitN(trimmed, ",", 2)
			if len(mediaAndData) == 2 {
				if mediaAndData[0] != "" {
					mimeType = mediaAndData[0]
				}
				data = mediaAndData[1]
			}
		}
	}
	return mimeType, data
}

// openAIResponsesImageFromBlock extracts inline image data from a content/output
// block across the shapes clients use (input_image, image_url, image, or a
// Claude-style base64 source block).
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go).
func openAIResponsesImageFromBlock(block gjson.Result) (mimeType string, data string, ok bool) {
	blockType := block.Get("type").String()
	switch blockType {
	case "input_image", "image_url", "image":
		imageURL := ""
		if block.Get("image_url.url").Exists() {
			imageURL = block.Get("image_url.url").String()
		} else if block.Get("image_url").Type == gjson.String {
			imageURL = block.Get("image_url").String()
		} else if block.Get("url").Exists() {
			imageURL = block.Get("url").String()
		}
		if imageURL != "" {
			mimeType, data = parseOpenAIResponsesDataURL(imageURL)
			if data != "" {
				return mimeType, data, true
			}
		}
		if block.Get("source.type").String() == "base64" {
			data = block.Get("source.data").String()
			mimeType = block.Get("source.media_type").String()
			if mimeType == "" {
				mimeType = "image/png"
			}
			if data != "" {
				return mimeType, data, true
			}
		}
	}
	return "", "", false
}

type openAIResponsesOutputBlock struct {
	text   string
	isText bool
	raw    string
}

// parseOpenAIResponsesArrayOutput splits a function_call_output array output into
// the textual/result payload and any embedded image blocks.
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go).
func parseOpenAIResponsesArrayOutput(outputResult gjson.Result) (result string, isRaw bool, images [][]byte) {
	var imageParts [][]byte
	var nonImageEntries []openAIResponsesOutputBlock
	var hasContentBlock bool
	var hasNonTextBlock bool

	outputResult.ForEach(func(_, block gjson.Result) bool {
		if mimeType, data, ok := openAIResponsesImageFromBlock(block); ok {
			hasContentBlock = true
			imageParts = append(imageParts, geminiResponsesInlineDataPart(mimeType, data))
			return true
		}
		bType := block.Get("type").String()
		if bType == "input_text" || bType == "output_text" || bType == "text" {
			hasContentBlock = true
			nonImageEntries = append(nonImageEntries, openAIResponsesOutputBlock{
				text:   block.Get("text").String(),
				isText: true,
				raw:    block.Raw,
			})
		} else if block.Type == gjson.String {
			nonImageEntries = append(nonImageEntries, openAIResponsesOutputBlock{
				text:   block.String(),
				isText: true,
				raw:    block.Raw,
			})
		} else {
			hasNonTextBlock = true
			nonImageEntries = append(nonImageEntries, openAIResponsesOutputBlock{
				text:   block.Raw,
				isText: false,
				raw:    block.Raw,
			})
		}
		return true
	})

	if !hasContentBlock {
		return outputResult.Raw, true, nil
	}

	switch len(nonImageEntries) {
	case 0:
		return "", false, imageParts
	case 1:
		if nonImageEntries[0].isText {
			return nonImageEntries[0].text, false, imageParts
		}
		return nonImageEntries[0].raw, true, imageParts
	default:
		if !hasNonTextBlock {
			texts := make([]string, len(nonImageEntries))
			for idx, e := range nonImageEntries {
				texts[idx] = e.text
			}
			return strings.Join(texts, "\n"), false, imageParts
		}
		rawItems := make([][]byte, len(nonImageEntries))
		for idx, e := range nonImageEntries {
			rawItems[idx] = []byte(e.raw)
		}
		return string(translatorcommon.JoinRawArray(rawItems)), true, imageParts
	}
}

// buildOpenAIResponsesFunctionResponsePart builds the functionResponse part for a
// matched function_call_output item. Image blocks embedded in the output are
// nested inside functionResponse.parts as inlineData instead of being appended
// as sibling parts next to the functionResponse.
// Ported from upstream CLIProxyAPI commit 728ea8b8557c ("nest image parts inside
// functionResponse").
func buildOpenAIResponsesFunctionResponsePart(functionName, callID string, outputResult gjson.Result) []byte {
	functionResponse := []byte(`{"functionResponse":{"name":"","response":{}}}`)
	functionResponse, _ = sjson.SetBytes(functionResponse, "functionResponse.name", functionName)
	functionResponse, _ = sjson.SetBytes(functionResponse, "functionResponse.id", callID)

	if outputResult.Type == gjson.String {
		str := outputResult.String()
		if str == "" || str == "null" {
			return functionResponse
		}
		// Keep it as a string instead of parsing it into JSON.
		// Parsing it as JSON, similar to reading a JSON file with readFile, may trigger an upstream 400 error.
		functionResponse, _ = sjson.SetBytes(functionResponse, "functionResponse.response.result", str)
		return functionResponse
	}

	var imageParts [][]byte
	switch {
	case outputResult.IsArray():
		result, isRaw, images := parseOpenAIResponsesArrayOutput(outputResult)
		imageParts = images
		if isRaw {
			functionResponse, _ = sjson.SetRawBytes(functionResponse, "functionResponse.response.result", []byte(result))
		} else {
			functionResponse, _ = sjson.SetBytes(functionResponse, "functionResponse.response.result", result)
		}
	case outputResult.IsObject():
		if mimeType, data, ok := openAIResponsesImageFromBlock(outputResult); ok {
			imageParts = append(imageParts, geminiResponsesInlineDataPart(mimeType, data))
			functionResponse, _ = sjson.SetBytes(functionResponse, "functionResponse.response.result", "")
		} else {
			functionResponse, _ = sjson.SetRawBytes(functionResponse, "functionResponse.response.result", []byte(outputResult.Raw))
		}
	case outputResult.Raw != "" && outputResult.Raw != "null":
		functionResponse, _ = sjson.SetBytes(functionResponse, "functionResponse.response.result", outputResult.String())
	}

	for _, imagePart := range imageParts {
		inlineData := []byte(`{"inlineData":{"mimeType":"","data":""}}`)
		inlineData, _ = sjson.SetBytes(inlineData, "inlineData.mimeType", gjson.GetBytes(imagePart, "inline_data.mime_type").String())
		inlineData, _ = sjson.SetBytes(inlineData, "inlineData.data", gjson.GetBytes(imagePart, "inline_data.data").String())
		functionResponse, _ = sjson.SetRawBytes(functionResponse, "functionResponse.parts.-1", inlineData)
	}
	return functionResponse
}

// geminiResponsesContent builds a Gemini content node with the given role and
// raw parts.
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go,
// helper geminiContent).
func geminiResponsesContent(role string, parts [][]byte) []byte {
	content := []byte(`{"role":"","parts":[]}`)
	content, _ = sjson.SetBytes(content, "role", role)
	content, _ = sjson.SetRawBytes(content, "parts", translatorcommon.JoinRawArray(parts))
	return content
}

// openAIResponsesAssistantVisibleText reports whether a Responses input message
// represents visible assistant output: either an assistant/model role message
// or a content array containing output_text items.
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go).
func openAIResponsesAssistantVisibleText(item gjson.Result) (string, bool) {
	itemType := item.Get("type").String()
	itemRole := item.Get("role").String()
	if itemType == "" && itemRole != "" {
		itemType = "message"
	}
	if itemType != "message" {
		return "", false
	}

	content := item.Get("content")
	if !content.Exists() {
		return "", false
	}
	if content.Type == gjson.String {
		switch strings.ToLower(strings.TrimSpace(itemRole)) {
		case "assistant", "model":
			return content.String(), true
		default:
			return "", false
		}
	}
	if !content.IsArray() {
		return "", false
	}

	var textParts []string
	hasOutputText := false
	content.ForEach(func(_, contentItem gjson.Result) bool {
		contentType := contentItem.Get("type").String()
		if contentType == "" {
			contentType = "input_text"
		}
		if contentType != "output_text" {
			return true
		}
		hasOutputText = true
		textParts = append(textParts, contentItem.Get("text").String())
		return true
	})
	if !hasOutputText {
		return "", false
	}
	// output_text marks model-visible content even when message.role is "user".
	return strings.Join(textParts, "\n"), true
}

// openAIResponsesHasMatchingOutput reports whether a function_call_output /
// custom_tool_call_output item matching callID exists in the remaining items.
// Ported from upstream CLIProxyAPI v7.3.3 (gemini_openai-responses_request.go,
// helper responsesHasMatchingOutput).
func openAIResponsesHasMatchingOutput(items []gjson.Result, callID string) bool {
	if callID == "" {
		return false
	}
	for _, item := range items {
		typ := item.Get("type").String()
		if typ == "function_call_output" || typ == "custom_tool_call_output" {
			if item.Get("call_id").String() == callID {
				return true
			}
		}
	}
	return false
}
