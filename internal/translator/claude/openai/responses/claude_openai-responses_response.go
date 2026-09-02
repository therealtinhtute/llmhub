package responses

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	translatorcommon "github.com/therealtinhtute/llmhub/internal/translator/common"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type claudeToResponsesState struct {
	Seq                int
	ResponseID         string
	CreatedAt          int64
	NextOutputIndex    int
	CurrentMsgID       string
	CurrentFCID        string
	InTextBlock        bool
	InFuncBlock        bool
	MessageOpen        bool
	ContentPartOpen    bool
	MessageOutputIndex int
	FuncArgsBuf        map[int]*strings.Builder // index -> args
	// function call bookkeeping for output aggregation
	FuncNames         map[int]string // Claude block index -> function name
	FuncCallIDs       map[int]string // Claude block index -> call id
	FuncOutputIndices map[int]int    // Claude block index -> Responses output index
	// message text aggregation
	TextBuf            strings.Builder
	CurrentTextBuf     strings.Builder
	MessageAnnotations []any
	MessageItems       []claudeResponsesMessageItem
	// reasoning state
	ReasoningActive    bool
	ReasoningItemID    string
	ReasoningBuf       strings.Builder
	ReasoningSignature string
	ReasoningIndex     int
	ReasoningItems     []claudeResponsesReasoningItem
	// server-side web search state: a Claude server_tool_use block and its
	// web_search_tool_result block fold into one Responses web_search_call item.
	WebSearchByBlock  map[int]*claudeResponsesWebSearchItem
	WebSearchByToolID map[string]*claudeResponsesWebSearchItem
	WebSearchItems    []*claudeResponsesWebSearchItem
	// usage aggregation
	InputTokens  int64
	OutputTokens int64
	UsageSeen    bool
}

type claudeResponsesWebSearchItem struct {
	ToolUseID   string
	OutputIndex int
	InputBuf    strings.Builder
	Results     []byte
	Emitted     bool
}

func (item *claudeResponsesWebSearchItem) render() []byte {
	return buildResponsesWebSearchCallItem(item.ToolUseID, claudeWebSearchQuery(item.InputBuf.String()), item.Results)
}

type claudeResponsesMessageItem struct {
	ID          string
	OutputIndex int
	Text        string
	Annotations []any
}

type claudeResponsesReasoningItem struct {
	ID          string
	OutputIndex int
	Text        string
	Signature   string
}

var dataTag = []byte("data:")

func pickRequestJSON(originalRequestRawJSON, requestRawJSON []byte) []byte {
	if len(originalRequestRawJSON) > 0 && gjson.ValidBytes(originalRequestRawJSON) {
		return originalRequestRawJSON
	}
	if len(requestRawJSON) > 0 && gjson.ValidBytes(requestRawJSON) {
		return requestRawJSON
	}
	return nil
}

func requestModelName(originalRequestRawJSON, requestRawJSON []byte) string {
	for _, rawJSON := range [][]byte{originalRequestRawJSON, requestRawJSON} {
		if len(rawJSON) == 0 || !gjson.ValidBytes(rawJSON) {
			continue
		}
		root := gjson.ParseBytes(rawJSON)
		for _, path := range []string{"model", "request.model"} {
			model := root.Get(path)
			if model.Type == gjson.String && strings.TrimSpace(model.String()) != "" {
				return model.String()
			}
		}
	}
	return ""
}

func emitEvent(event string, payload []byte) []byte {
	return translatorcommon.SSEEventData(event, payload)
}

func (st *claudeToResponsesState) appendMessageAnnotation(annotation any) {
	if annotation == nil {
		return
	}
	st.MessageAnnotations = append(st.MessageAnnotations, annotation)
}

func (st *claudeToResponsesState) allocateOutputIndex() int {
	index := st.NextOutputIndex
	st.NextOutputIndex++
	return index
}

func (st *claudeToResponsesState) messageOutputIndex() int {
	if st.MessageOutputIndex < 0 {
		st.MessageOutputIndex = st.allocateOutputIndex()
	}
	return st.MessageOutputIndex
}

func (st *claudeToResponsesState) functionOutputIndex(blockIndex int) int {
	if index, ok := st.FuncOutputIndices[blockIndex]; ok {
		return index
	}
	index := st.allocateOutputIndex()
	st.FuncOutputIndices[blockIndex] = index
	return index
}

func (st *claudeToResponsesState) startWebSearch(blockIndex int, toolUseID string) *claudeResponsesWebSearchItem {
	item := &claudeResponsesWebSearchItem{ToolUseID: toolUseID, OutputIndex: st.allocateOutputIndex()}
	st.WebSearchByBlock[blockIndex] = item
	st.WebSearchByToolID[toolUseID] = item
	st.WebSearchItems = append(st.WebSearchItems, item)
	return item
}

func (st *claudeToResponsesState) finalizeWebSearch(item *claudeResponsesWebSearchItem, nextSeq func() int) [][]byte {
	if item == nil || item.Emitted {
		return nil
	}
	item.Emitted = true
	done := []byte(`{"type":"response.output_item.done","sequence_number":0,"output_index":0,"item":{}}`)
	done, _ = sjson.SetBytes(done, "sequence_number", nextSeq())
	done, _ = sjson.SetBytes(done, "output_index", item.OutputIndex)
	done, _ = sjson.SetRawBytes(done, "item", item.render())
	return [][]byte{emitEvent("response.output_item.done", done)}
}

func (st *claudeToResponsesState) finalizeAssistantMessage(nextSeq func() int) [][]byte {
	if !st.MessageOpen {
		return nil
	}
	fullText := st.TextBuf.String()
	outputIndex := st.messageOutputIndex()
	var out [][]byte
	done := []byte(`{"type":"response.output_text.done","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"text":"","logprobs":[]}`)
	done, _ = sjson.SetBytes(done, "sequence_number", nextSeq())
	done, _ = sjson.SetBytes(done, "item_id", st.CurrentMsgID)
	done, _ = sjson.SetBytes(done, "output_index", outputIndex)
	done, _ = sjson.SetBytes(done, "text", fullText)
	out = append(out, emitEvent("response.output_text.done", done))

	partDone := []byte(`{"type":"response.content_part.done","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":""}}`)
	partDone, _ = sjson.SetBytes(partDone, "sequence_number", nextSeq())
	partDone, _ = sjson.SetBytes(partDone, "item_id", st.CurrentMsgID)
	partDone, _ = sjson.SetBytes(partDone, "output_index", outputIndex)
	partDone, _ = sjson.SetBytes(partDone, "part.text", fullText)
	if len(st.MessageAnnotations) > 0 {
		partDone, _ = sjson.SetBytes(partDone, "part.annotations", st.MessageAnnotations)
	}
	out = append(out, emitEvent("response.content_part.done", partDone))

	final := []byte(`{"type":"response.output_item.done","sequence_number":0,"output_index":0,"item":{"id":"","type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":""}],"role":"assistant"}}`)
	final, _ = sjson.SetBytes(final, "sequence_number", nextSeq())
	final, _ = sjson.SetBytes(final, "output_index", outputIndex)
	final, _ = sjson.SetBytes(final, "item.id", st.CurrentMsgID)
	final, _ = sjson.SetBytes(final, "item.content.0.text", fullText)
	if len(st.MessageAnnotations) > 0 {
		final, _ = sjson.SetBytes(final, "item.content.0.annotations", st.MessageAnnotations)
	}
	out = append(out, emitEvent("response.output_item.done", final))

	st.MessageItems = append(st.MessageItems, claudeResponsesMessageItem{
		ID:          st.CurrentMsgID,
		OutputIndex: outputIndex,
		Text:        fullText,
		Annotations: append([]any(nil), st.MessageAnnotations...),
	})
	st.InTextBlock = false
	st.MessageOpen = false
	st.ContentPartOpen = false
	st.CurrentMsgID = ""
	st.MessageOutputIndex = -1
	st.TextBuf.Reset()
	st.CurrentTextBuf.Reset()
	st.MessageAnnotations = nil
	return out
}

// ConvertClaudeResponseToOpenAIResponses converts Claude SSE to OpenAI Responses SSE events.
func ConvertClaudeResponseToOpenAIResponses(ctx context.Context, modelName string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) [][]byte {
	if *param == nil {
		*param = &claudeToResponsesState{
			MessageOutputIndex: -1,
			ReasoningIndex:     -1,
			FuncArgsBuf:        make(map[int]*strings.Builder),
			FuncNames:          make(map[int]string),
			FuncCallIDs:        make(map[int]string),
			FuncOutputIndices:  make(map[int]int),
			WebSearchByBlock:   make(map[int]*claudeResponsesWebSearchItem),
			WebSearchByToolID:  make(map[string]*claudeResponsesWebSearchItem),
		}
	}
	st := (*param).(*claudeToResponsesState)

	// Expect `data: {..}` from Claude clients
	if !bytes.HasPrefix(rawJSON, dataTag) {
		return [][]byte{}
	}
	rawJSON = bytes.TrimSpace(rawJSON[5:])
	root := gjson.ParseBytes(rawJSON)
	ev := root.Get("type").String()
	var out [][]byte

	nextSeq := func() int { st.Seq++; return st.Seq }

	switch ev {
	case "message_start":
		if msg := root.Get("message"); msg.Exists() {
			st.ResponseID = msg.Get("id").String()
			st.CreatedAt = time.Now().Unix()
			// Reset per-message aggregation state
			st.TextBuf.Reset()
			st.CurrentTextBuf.Reset()
			st.MessageAnnotations = nil
			st.MessageItems = nil
			st.ReasoningBuf.Reset()
			st.ReasoningActive = false
			st.NextOutputIndex = 0
			st.InTextBlock = false
			st.InFuncBlock = false
			st.MessageOpen = false
			st.ContentPartOpen = false
			st.CurrentMsgID = ""
			st.CurrentFCID = ""
			st.MessageOutputIndex = -1
			st.ReasoningItemID = ""
			st.ReasoningSignature = ""
			st.ReasoningIndex = -1
			st.ReasoningItems = nil
			st.FuncArgsBuf = make(map[int]*strings.Builder)
			st.FuncNames = make(map[int]string)
			st.FuncCallIDs = make(map[int]string)
			st.FuncOutputIndices = make(map[int]int)
			st.WebSearchByBlock = make(map[int]*claudeResponsesWebSearchItem)
			st.WebSearchByToolID = make(map[string]*claudeResponsesWebSearchItem)
			st.WebSearchItems = nil
			st.InputTokens = 0
			st.OutputTokens = 0
			st.UsageSeen = false
			if usage := msg.Get("usage"); usage.Exists() {
				if v := usage.Get("input_tokens"); v.Exists() {
					st.InputTokens = v.Int()
					st.UsageSeen = true
				}
				if v := usage.Get("output_tokens"); v.Exists() {
					st.OutputTokens = v.Int()
					st.UsageSeen = true
				}
			}
			// response.created
			created := []byte(`{"type":"response.created","sequence_number":0,"response":{"id":"","object":"response","created_at":0,"status":"in_progress","background":false,"error":null,"output":[]}}`)
			created, _ = sjson.SetBytes(created, "sequence_number", nextSeq())
			created, _ = sjson.SetBytes(created, "response.id", st.ResponseID)
			created, _ = sjson.SetBytes(created, "response.created_at", st.CreatedAt)
			reqModel := requestModelName(originalRequestRawJSON, requestRawJSON)
			if reqModel == "" {
				reqModel = modelName
			}
			if reqModel != "" {
				created, _ = sjson.SetBytes(created, "response.model", reqModel)
			}
			out = append(out, emitEvent("response.created", created))
			// response.in_progress
			inprog := []byte(`{"type":"response.in_progress","sequence_number":0,"response":{"id":"","object":"response","created_at":0,"status":"in_progress"}}`)
			inprog, _ = sjson.SetBytes(inprog, "sequence_number", nextSeq())
			inprog, _ = sjson.SetBytes(inprog, "response.id", st.ResponseID)
			inprog, _ = sjson.SetBytes(inprog, "response.created_at", st.CreatedAt)
			if reqModel != "" {
				inprog, _ = sjson.SetBytes(inprog, "response.model", reqModel)
			}
			out = append(out, emitEvent("response.in_progress", inprog))
		}
	case "content_block_start":
		cb := root.Get("content_block")
		if !cb.Exists() {
			return out
		}
		idx := int(root.Get("index").Int())
		typ := cb.Get("type").String()
		if typ == "text" {
			st.InTextBlock = true
			outputIndex := st.messageOutputIndex()
			if st.CurrentMsgID == "" {
				st.CurrentMsgID = fmt.Sprintf("msg_%s_%d", st.ResponseID, len(st.MessageItems))
			}
			if !st.MessageOpen {
				item := []byte(`{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"message","status":"in_progress","content":[],"role":"assistant"}}`)
				item, _ = sjson.SetBytes(item, "sequence_number", nextSeq())
				item, _ = sjson.SetBytes(item, "output_index", outputIndex)
				item, _ = sjson.SetBytes(item, "item.id", st.CurrentMsgID)
				out = append(out, emitEvent("response.output_item.added", item))
				st.MessageOpen = true
			}
			if !st.ContentPartOpen {
				part := []byte(`{"type":"response.content_part.added","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":""}}`)
				part, _ = sjson.SetBytes(part, "sequence_number", nextSeq())
				part, _ = sjson.SetBytes(part, "item_id", st.CurrentMsgID)
				part, _ = sjson.SetBytes(part, "output_index", outputIndex)
				out = append(out, emitEvent("response.content_part.added", part))
				st.ContentPartOpen = true
			}
		} else if typ == "tool_use" {
			out = append(out, st.finalizeAssistantMessage(nextSeq)...)
			st.InFuncBlock = true
			st.CurrentFCID = cb.Get("id").String()
			name := cb.Get("name").String()
			outputIndex := st.functionOutputIndex(idx)
			item := []byte(`{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"function_call","status":"in_progress","arguments":"","call_id":"","name":""}}`)
			item, _ = sjson.SetBytes(item, "sequence_number", nextSeq())
			item, _ = sjson.SetBytes(item, "output_index", outputIndex)
			item, _ = sjson.SetBytes(item, "item.id", fmt.Sprintf("fc_%s", st.CurrentFCID))
			item, _ = sjson.SetBytes(item, "item.call_id", st.CurrentFCID)
			item, _ = sjson.SetBytes(item, "item.name", name)
			out = append(out, emitEvent("response.output_item.added", item))
			if st.FuncArgsBuf[idx] == nil {
				st.FuncArgsBuf[idx] = &strings.Builder{}
			}
			// record function metadata for aggregation
			st.FuncCallIDs[idx] = st.CurrentFCID
			st.FuncNames[idx] = name
		} else if typ == "server_tool_use" {
			out = append(out, st.finalizeAssistantMessage(nextSeq)...)
			if name := cb.Get("name").String(); name != claudeWebSearchToolName {
				log.Debugf("claude->responses: unmapped server_tool_use %q at block %d", name, idx)
			} else {
				item := st.startWebSearch(idx, cb.Get("id").String())
				added := []byte(`{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"web_search_call","status":"in_progress","action":{"type":"search","query":""}}}`)
				added, _ = sjson.SetBytes(added, "sequence_number", nextSeq())
				added, _ = sjson.SetBytes(added, "output_index", item.OutputIndex)
				added, _ = sjson.SetBytes(added, "item.id", responsesWebSearchCallID(item.ToolUseID))
				out = append(out, emitEvent("response.output_item.added", added))
			}
		} else if typ == "web_search_tool_result" {
			if item := st.WebSearchByToolID[cb.Get("tool_use_id").String()]; item != nil {
				item.Results = claudeWebSearchResultsToResponses(cb.Get("content"))
				out = append(out, st.finalizeWebSearch(item, nextSeq)...)
			} else {
				log.Debugf("claude->responses: web_search_tool_result without matching server_tool_use at block %d", idx)
			}
		} else if typ == "thinking" {
			out = append(out, st.finalizeAssistantMessage(nextSeq)...)
			// start reasoning item
			st.ReasoningActive = true
			st.ReasoningIndex = st.allocateOutputIndex()
			st.ReasoningBuf.Reset()
			st.ReasoningSignature = ""
			if signature := cb.Get("signature"); signature.Exists() && signature.String() != "" {
				st.ReasoningSignature = signature.String()
			}
			st.ReasoningItemID = fmt.Sprintf("rs_%s_%d", st.ResponseID, idx)
			item := []byte(`{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"reasoning","status":"in_progress","encrypted_content":"","summary":[]}}`)
			item, _ = sjson.SetBytes(item, "sequence_number", nextSeq())
			item, _ = sjson.SetBytes(item, "output_index", st.ReasoningIndex)
			item, _ = sjson.SetBytes(item, "item.id", st.ReasoningItemID)
			item, _ = sjson.SetBytes(item, "item.encrypted_content", st.ReasoningSignature)
			out = append(out, emitEvent("response.output_item.added", item))
			// add a summary part placeholder
			part := []byte(`{"type":"response.reasoning_summary_part.added","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`)
			part, _ = sjson.SetBytes(part, "sequence_number", nextSeq())
			part, _ = sjson.SetBytes(part, "item_id", st.ReasoningItemID)
			part, _ = sjson.SetBytes(part, "output_index", st.ReasoningIndex)
			out = append(out, emitEvent("response.reasoning_summary_part.added", part))
		}
	case "content_block_delta":
		d := root.Get("delta")
		if !d.Exists() {
			return out
		}
		dt := d.Get("type").String()
		if dt == "text_delta" {
			if t := d.Get("text"); t.Exists() {
				msg := []byte(`{"type":"response.output_text.delta","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"delta":"","logprobs":[]}`)
				msg, _ = sjson.SetBytes(msg, "sequence_number", nextSeq())
				msg, _ = sjson.SetBytes(msg, "item_id", st.CurrentMsgID)
				msg, _ = sjson.SetBytes(msg, "output_index", st.messageOutputIndex())
				msg, _ = sjson.SetBytes(msg, "delta", t.String())
				out = append(out, emitEvent("response.output_text.delta", msg))
				// aggregate text for response.output
				st.TextBuf.WriteString(t.String())
				st.CurrentTextBuf.WriteString(t.String())
			}
		} else if dt == "input_json_delta" {
			if item := st.WebSearchByBlock[int(root.Get("index").Int())]; item != nil {
				if pj := d.Get("partial_json"); pj.Exists() {
					item.InputBuf.WriteString(pj.String())
				}
				return [][]byte{}
			}
			idx := int(root.Get("index").Int())
			if pj := d.Get("partial_json"); pj.Exists() {
				if st.FuncArgsBuf[idx] == nil {
					st.FuncArgsBuf[idx] = &strings.Builder{}
				}
				st.FuncArgsBuf[idx].WriteString(pj.String())
				outputIndex := st.functionOutputIndex(idx)
				msg := []byte(`{"type":"response.function_call_arguments.delta","sequence_number":0,"item_id":"","output_index":0,"delta":""}`)
				msg, _ = sjson.SetBytes(msg, "sequence_number", nextSeq())
				msg, _ = sjson.SetBytes(msg, "item_id", fmt.Sprintf("fc_%s", st.CurrentFCID))
				msg, _ = sjson.SetBytes(msg, "output_index", outputIndex)
				msg, _ = sjson.SetBytes(msg, "delta", pj.String())
				out = append(out, emitEvent("response.function_call_arguments.delta", msg))
			}
		} else if dt == "thinking_delta" {
			if st.ReasoningActive {
				if t := d.Get("thinking"); t.Exists() {
					st.ReasoningBuf.WriteString(t.String())
					msg := []byte(`{"type":"response.reasoning_summary_text.delta","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"delta":""}`)
					msg, _ = sjson.SetBytes(msg, "sequence_number", nextSeq())
					msg, _ = sjson.SetBytes(msg, "item_id", st.ReasoningItemID)
					msg, _ = sjson.SetBytes(msg, "output_index", st.ReasoningIndex)
					msg, _ = sjson.SetBytes(msg, "delta", t.String())
					out = append(out, emitEvent("response.reasoning_summary_text.delta", msg))
				}
			}
		} else if dt == "signature_delta" {
			if st.ReasoningActive {
				if signature := d.Get("signature"); signature.Exists() && signature.String() != "" {
					st.ReasoningSignature = signature.String()
				}
			}
		} else if dt == "citations_delta" {
			if citation := d.Get("citation"); citation.Exists() {
				st.appendMessageAnnotation(citation.Value())
			}
			return [][]byte{}
		}
	case "content_block_stop":
		idx := int(root.Get("index").Int())
		if st.InTextBlock {
			st.InTextBlock = false
		} else if st.InFuncBlock {
			outputIndex := st.functionOutputIndex(idx)
			args := "{}"
			if buf := st.FuncArgsBuf[idx]; buf != nil {
				if buf.Len() > 0 {
					args = buf.String()
				}
			}
			fcDone := []byte(`{"type":"response.function_call_arguments.done","sequence_number":0,"item_id":"","output_index":0,"arguments":""}`)
			fcDone, _ = sjson.SetBytes(fcDone, "sequence_number", nextSeq())
			fcDone, _ = sjson.SetBytes(fcDone, "item_id", fmt.Sprintf("fc_%s", st.CurrentFCID))
			fcDone, _ = sjson.SetBytes(fcDone, "output_index", outputIndex)
			fcDone, _ = sjson.SetBytes(fcDone, "arguments", args)
			out = append(out, emitEvent("response.function_call_arguments.done", fcDone))
			itemDone := []byte(`{"type":"response.output_item.done","sequence_number":0,"output_index":0,"item":{"id":"","type":"function_call","status":"completed","arguments":"","call_id":"","name":""}}`)
			itemDone, _ = sjson.SetBytes(itemDone, "sequence_number", nextSeq())
			itemDone, _ = sjson.SetBytes(itemDone, "output_index", outputIndex)
			itemDone, _ = sjson.SetBytes(itemDone, "item.id", fmt.Sprintf("fc_%s", st.CurrentFCID))
			itemDone, _ = sjson.SetBytes(itemDone, "item.arguments", args)
			itemDone, _ = sjson.SetBytes(itemDone, "item.call_id", st.CurrentFCID)
			itemDone, _ = sjson.SetBytes(itemDone, "item.name", st.FuncNames[idx])
			out = append(out, emitEvent("response.output_item.done", itemDone))
			st.InFuncBlock = false
		} else if st.ReasoningActive {
			full := st.ReasoningBuf.String()
			textDone := []byte(`{"type":"response.reasoning_summary_text.done","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"text":""}`)
			textDone, _ = sjson.SetBytes(textDone, "sequence_number", nextSeq())
			textDone, _ = sjson.SetBytes(textDone, "item_id", st.ReasoningItemID)
			textDone, _ = sjson.SetBytes(textDone, "output_index", st.ReasoningIndex)
			textDone, _ = sjson.SetBytes(textDone, "text", full)
			out = append(out, emitEvent("response.reasoning_summary_text.done", textDone))
			partDone := []byte(`{"type":"response.reasoning_summary_part.done","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`)
			partDone, _ = sjson.SetBytes(partDone, "sequence_number", nextSeq())
			partDone, _ = sjson.SetBytes(partDone, "item_id", st.ReasoningItemID)
			partDone, _ = sjson.SetBytes(partDone, "output_index", st.ReasoningIndex)
			partDone, _ = sjson.SetBytes(partDone, "part.text", full)
			out = append(out, emitEvent("response.reasoning_summary_part.done", partDone))
			itemDone := []byte(`{"type":"response.output_item.done","sequence_number":0,"output_index":0,"item":{"id":"","type":"reasoning","encrypted_content":"","summary":[]}}`)
			itemDone, _ = sjson.SetBytes(itemDone, "sequence_number", nextSeq())
			itemDone, _ = sjson.SetBytes(itemDone, "item.id", st.ReasoningItemID)
			itemDone, _ = sjson.SetBytes(itemDone, "output_index", st.ReasoningIndex)
			itemDone, _ = sjson.SetBytes(itemDone, "item.encrypted_content", st.ReasoningSignature)
			summary := []byte(`{"type":"summary_text","text":""}`)
			summary, _ = sjson.SetBytes(summary, "text", full)
			itemDone = translatorcommon.SetRawArrayItems(itemDone, "item.summary", [][]byte{summary})
			out = append(out, emitEvent("response.output_item.done", itemDone))
			st.ReasoningItems = append(st.ReasoningItems, claudeResponsesReasoningItem{
				ID:          st.ReasoningItemID,
				OutputIndex: st.ReasoningIndex,
				Text:        full,
				Signature:   st.ReasoningSignature,
			})
			st.ReasoningActive = false
			st.ReasoningItemID = ""
			st.ReasoningBuf.Reset()
			st.ReasoningSignature = ""
			st.ReasoningIndex = -1
		}
	case "message_delta":
		if usage := root.Get("usage"); usage.Exists() {
			if v := usage.Get("output_tokens"); v.Exists() {
				st.OutputTokens = v.Int()
				st.UsageSeen = true
			}
			if v := usage.Get("input_tokens"); v.Exists() {
				st.InputTokens = v.Int()
				st.UsageSeen = true
			}
		}
	case "message_stop":
		out = append(out, st.finalizeAssistantMessage(nextSeq)...)
		for _, item := range st.WebSearchItems {
			out = append(out, st.finalizeWebSearch(item, nextSeq)...)
		}

		completed := []byte(`{"type":"response.completed","sequence_number":0,"response":{"id":"","object":"response","created_at":0,"status":"completed","background":false,"error":null}}`)
		completed, _ = sjson.SetBytes(completed, "sequence_number", nextSeq())
		completed, _ = sjson.SetBytes(completed, "response.id", st.ResponseID)
		completed, _ = sjson.SetBytes(completed, "response.created_at", st.CreatedAt)
		// Inject original request fields into response as per docs/response.completed.json

		reqBytes := pickRequestJSON(originalRequestRawJSON, requestRawJSON)
		if len(reqBytes) > 0 {
			req := gjson.ParseBytes(reqBytes)
			if v := req.Get("instructions"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.instructions", v.String())
			}
			if v := req.Get("max_output_tokens"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.max_output_tokens", v.Int())
			}
			if v := req.Get("max_tool_calls"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.max_tool_calls", v.Int())
			}
			if v := req.Get("model"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.model", v.String())
			}
			if v := req.Get("parallel_tool_calls"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.parallel_tool_calls", v.Bool())
			}
			if v := req.Get("previous_response_id"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.previous_response_id", v.String())
			}
			if v := req.Get("prompt_cache_key"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.prompt_cache_key", v.String())
			}
			if v := req.Get("reasoning"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.reasoning", v.Value())
			}
			if v := req.Get("safety_identifier"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.safety_identifier", v.String())
			}
			if v := req.Get("service_tier"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.service_tier", v.String())
			}
			if v := req.Get("store"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.store", v.Bool())
			}
			if v := req.Get("temperature"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.temperature", v.Float())
			}
			if v := req.Get("text"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.text", v.Value())
			}
			if v := req.Get("tool_choice"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.tool_choice", v.Value())
			}
			if v := req.Get("tools"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.tools", v.Value())
			}
			if v := req.Get("top_logprobs"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.top_logprobs", v.Int())
			}
			if v := req.Get("top_p"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.top_p", v.Float())
			}
			if v := req.Get("truncation"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.truncation", v.String())
			}
			if v := req.Get("user"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.user", v.Value())
			}
			if v := req.Get("metadata"); v.Exists() {
				completed, _ = sjson.SetBytes(completed, "response.metadata", v.Value())
			}
		}

		// Build response.output from aggregated state
		outputsWrapper := []byte(`{"arr":[]}`)
		// reasoning items
		for _, reasoning := range st.ReasoningItems {
			item := []byte(`{"id":"","type":"reasoning","encrypted_content":"","summary":[]}`)
			item, _ = sjson.SetBytes(item, "id", reasoning.ID)
			item, _ = sjson.SetBytes(item, "encrypted_content", reasoning.Signature)
			summary := []byte(`{"type":"summary_text","text":""}`)
			summary, _ = sjson.SetBytes(summary, "text", reasoning.Text)
			item = translatorcommon.SetRawArrayItems(item, "summary", [][]byte{summary})
			outputsWrapper, _ = sjson.SetRawBytes(outputsWrapper, fmt.Sprintf("arr.%d", reasoning.OutputIndex), item)
		}
		// assistant message items
		for _, message := range st.MessageItems {
			item := []byte(`{"id":"","type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":""}],"role":"assistant"}`)
			item, _ = sjson.SetBytes(item, "id", message.ID)
			item, _ = sjson.SetBytes(item, "content.0.text", message.Text)
			if len(message.Annotations) > 0 {
				item, _ = sjson.SetBytes(item, "content.0.annotations", message.Annotations)
			}
			outputsWrapper, _ = sjson.SetRawBytes(outputsWrapper, fmt.Sprintf("arr.%d", message.OutputIndex), item)
		}
		// web_search_call items
		for _, item := range st.WebSearchItems {
			outputsWrapper, _ = sjson.SetRawBytes(outputsWrapper, fmt.Sprintf("arr.%d", item.OutputIndex), item.render())
		}
		// function_call items (in ascending index order for determinism)
		if len(st.FuncArgsBuf) > 0 {
			// collect indices
			idxs := make([]int, 0, len(st.FuncArgsBuf))
			for idx := range st.FuncArgsBuf {
				idxs = append(idxs, idx)
			}
			sort.Ints(idxs)
			for _, idx := range idxs {
				args := "{}"
				if b := st.FuncArgsBuf[idx]; b != nil && b.Len() > 0 {
					args = b.String()
				}
				callID := st.FuncCallIDs[idx]
				name := st.FuncNames[idx]
				if callID == "" && st.CurrentFCID != "" {
					callID = st.CurrentFCID
				}
				item := []byte(`{"id":"","type":"function_call","status":"completed","arguments":"","call_id":"","name":""}`)
				item, _ = sjson.SetBytes(item, "id", fmt.Sprintf("fc_%s", callID))
				item, _ = sjson.SetBytes(item, "arguments", args)
				item, _ = sjson.SetBytes(item, "call_id", callID)
				item, _ = sjson.SetBytes(item, "name", name)
				outputsWrapper, _ = sjson.SetRawBytes(outputsWrapper, fmt.Sprintf("arr.%d", st.functionOutputIndex(idx)), item)
			}
		}
		if gjson.GetBytes(outputsWrapper, "arr.#").Int() > 0 {
			completed, _ = sjson.SetRawBytes(completed, "response.output", []byte(gjson.GetBytes(outputsWrapper, "arr").Raw))
		}

		reasoningTokens := int64(0)
		for _, reasoning := range st.ReasoningItems {
			reasoningTokens += int64(len(reasoning.Text) / 4)
		}
		usagePresent := st.UsageSeen || reasoningTokens > 0
		if usagePresent {
			completed, _ = sjson.SetBytes(completed, "response.usage.input_tokens", st.InputTokens)
			completed, _ = sjson.SetBytes(completed, "response.usage.input_tokens_details.cached_tokens", 0)
			completed, _ = sjson.SetBytes(completed, "response.usage.output_tokens", st.OutputTokens)
			completed, _ = sjson.SetBytes(completed, "response.usage.output_tokens_details.reasoning_tokens", reasoningTokens)
			total := st.InputTokens + st.OutputTokens
			if total > 0 || st.UsageSeen {
				completed, _ = sjson.SetBytes(completed, "response.usage.total_tokens", total)
			}
		}
		out = append(out, emitEvent("response.completed", completed))
	}

	return out
}

// ConvertClaudeResponseToOpenAIResponsesNonStream aggregates Claude SSE into a single OpenAI Responses JSON.
func ConvertClaudeResponseToOpenAIResponsesNonStream(_ context.Context, _ string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, _ *any) []byte {
	// Aggregate Claude SSE lines into a single OpenAI Responses JSON (non-stream)
	// We follow the same aggregation logic as the streaming variant but produce
	// one final object matching docs/out.json structure.

	// Collect SSE data: lines start with "data: "; ignore others
	var chunks [][]byte
	remaining := rawJSON
	for len(remaining) > 0 {
		var line []byte
		idx := bytes.IndexByte(remaining, '\n')
		if idx >= 0 {
			line = remaining[:idx]
			remaining = remaining[idx+1:]
		} else {
			line = remaining
			remaining = nil
		}
		line = bytes.TrimRight(line, "\r")
		if !bytes.HasPrefix(line, dataTag) {
			continue
		}
		chunks = append(chunks, line[len(dataTag):])
	}

	// Base OpenAI Responses (non-stream) object
	out := []byte(`{"id":"","object":"response","created_at":0,"status":"completed","background":false,"error":null,"incomplete_details":null,"output":[],"usage":{"input_tokens":0,"input_tokens_details":{"cached_tokens":0},"output_tokens":0,"output_tokens_details":{},"total_tokens":0}}`)

	// Aggregation state
	var (
		responseID   string
		createdAt    int64
		inputTokens  int64
		outputTokens int64
	)

	type nonStreamOutputItem struct {
		outputIndex int
		itemType    string
		id          string
		callID      string
		name        string
		text        strings.Builder
		signature   string
		annotations []any
		args        strings.Builder
		results     []byte
	}

	// Preserve content block order: map each Claude block index to its output
	// item and keep the insertion order in outputItems.
	blockToItem := make(map[int]*nonStreamOutputItem)
	webSearchByToolID := make(map[string]*nonStreamOutputItem)
	outputItems := make([]*nonStreamOutputItem, 0)
	nextOutputIndex := 0
	messageCount := 0
	var activeMessageItem *nonStreamOutputItem
	var pendingAnnotations []any

	allocateOutputIndex := func() int {
		outputIndex := nextOutputIndex
		nextOutputIndex++
		return outputIndex
	}
	newOutputItem := func(itemType string, blockIndex int) *nonStreamOutputItem {
		item := &nonStreamOutputItem{
			outputIndex: allocateOutputIndex(),
			itemType:    itemType,
		}
		outputItems = append(outputItems, item)
		blockToItem[blockIndex] = item
		return item
	}

	// Walk through SSE chunks to fill state
	for _, ch := range chunks {
		root := gjson.ParseBytes(ch)
		ev := root.Get("type").String()

		switch ev {
		case "message_start":
			if msg := root.Get("message"); msg.Exists() {
				responseID = msg.Get("id").String()
				createdAt = time.Now().Unix()
				if usage := msg.Get("usage"); usage.Exists() {
					if v := usage.Get("input_tokens"); v.Exists() {
						inputTokens = v.Int()
					}
				}
			}

		case "content_block_start":
			cb := root.Get("content_block")
			if !cb.Exists() {
				continue
			}
			idx := int(root.Get("index").Int())
			typ := cb.Get("type").String()
			switch typ {
			case "text":
				item := newOutputItem("message", idx)
				item.id = fmt.Sprintf("msg_%s_%d", responseID, messageCount)
				messageCount++
				if len(pendingAnnotations) > 0 {
					item.annotations = append(item.annotations, pendingAnnotations...)
					pendingAnnotations = nil
				}
				activeMessageItem = item
			case "tool_use":
				activeMessageItem = nil
				item := newOutputItem("function_call", idx)
				item.callID = cb.Get("id").String()
				item.id = fmt.Sprintf("fc_%s", item.callID)
				item.name = cb.Get("name").String()
			case "server_tool_use":
				activeMessageItem = nil
				name := cb.Get("name").String()
				if name != claudeWebSearchToolName {
					log.Debugf("claude->responses: unmapped server_tool_use %q at block %d", name, idx)
					continue
				}
				toolUseID := cb.Get("id").String()
				item := newOutputItem("web_search_call", idx)
				item.id = responsesWebSearchCallID(toolUseID)
				item.callID = toolUseID
				webSearchByToolID[toolUseID] = item
				if input := cb.Get("input"); input.IsObject() && claudeWebSearchQuery(input.Raw) != "" {
					item.args.WriteString(input.Raw)
				}
			case "web_search_tool_result":
				if item := webSearchByToolID[cb.Get("tool_use_id").String()]; item != nil {
					item.results = claudeWebSearchResultsToResponses(cb.Get("content"))
				} else {
					log.Debugf("claude->responses: web_search_tool_result without matching server_tool_use at block %d", idx)
				}
			case "thinking":
				activeMessageItem = nil
				item := newOutputItem("reasoning", idx)
				item.id = fmt.Sprintf("rs_%s_%d", responseID, idx)
				if signature := cb.Get("signature"); signature.Exists() && signature.String() != "" {
					item.signature = signature.String()
				}
			}

		case "content_block_delta":
			d := root.Get("delta")
			if !d.Exists() {
				continue
			}
			idx := int(root.Get("index").Int())
			item := blockToItem[idx]
			dt := d.Get("type").String()
			switch dt {
			case "text_delta":
				if item != nil && item.itemType == "message" {
					if t := d.Get("text"); t.Exists() {
						item.text.WriteString(t.String())
					}
				}
			case "input_json_delta":
				if item != nil && (item.itemType == "function_call" || item.itemType == "web_search_call") {
					if pj := d.Get("partial_json"); pj.Exists() {
						item.args.WriteString(pj.String())
					}
				}
			case "thinking_delta":
				if item != nil && item.itemType == "reasoning" {
					if t := d.Get("thinking"); t.Exists() {
						item.text.WriteString(t.String())
					}
				}
			case "signature_delta":
				if item != nil && item.itemType == "reasoning" {
					if signature := d.Get("signature"); signature.Exists() && signature.String() != "" {
						item.signature = signature.String()
					}
				}
			case "citations_delta":
				if citation := d.Get("citation"); citation.Exists() {
					if item != nil && item.itemType == "message" {
						item.annotations = append(item.annotations, citation.Value())
					} else if activeMessageItem != nil {
						activeMessageItem.annotations = append(activeMessageItem.annotations, citation.Value())
					} else {
						pendingAnnotations = append(pendingAnnotations, citation.Value())
					}
				}
			}

		case "content_block_stop":
			// Output items are finalized after all deltas have been aggregated.

		case "message_delta":
			if usage := root.Get("usage"); usage.Exists() {
				if v := usage.Get("output_tokens"); v.Exists() {
					outputTokens = v.Int()
				}
			}
		}
	}

	// Populate base fields
	out, _ = sjson.SetBytes(out, "id", responseID)
	out, _ = sjson.SetBytes(out, "created_at", createdAt)

	// Inject request echo fields as top-level (similar to streaming variant)
	reqBytes := pickRequestJSON(originalRequestRawJSON, requestRawJSON)
	if len(reqBytes) > 0 {
		req := gjson.ParseBytes(reqBytes)
		if v := req.Get("instructions"); v.Exists() {
			out, _ = sjson.SetBytes(out, "instructions", v.String())
		}
		if v := req.Get("max_output_tokens"); v.Exists() {
			out, _ = sjson.SetBytes(out, "max_output_tokens", v.Int())
		}
		if v := req.Get("max_tool_calls"); v.Exists() {
			out, _ = sjson.SetBytes(out, "max_tool_calls", v.Int())
		}
		if v := req.Get("model"); v.Exists() {
			out, _ = sjson.SetBytes(out, "model", v.String())
		}
		if v := req.Get("parallel_tool_calls"); v.Exists() {
			out, _ = sjson.SetBytes(out, "parallel_tool_calls", v.Bool())
		}
		if v := req.Get("previous_response_id"); v.Exists() {
			out, _ = sjson.SetBytes(out, "previous_response_id", v.String())
		}
		if v := req.Get("prompt_cache_key"); v.Exists() {
			out, _ = sjson.SetBytes(out, "prompt_cache_key", v.String())
		}
		if v := req.Get("reasoning"); v.Exists() {
			out, _ = sjson.SetBytes(out, "reasoning", v.Value())
		}
		if v := req.Get("safety_identifier"); v.Exists() {
			out, _ = sjson.SetBytes(out, "safety_identifier", v.String())
		}
		if v := req.Get("service_tier"); v.Exists() {
			out, _ = sjson.SetBytes(out, "service_tier", v.String())
		}
		if v := req.Get("store"); v.Exists() {
			out, _ = sjson.SetBytes(out, "store", v.Bool())
		}
		if v := req.Get("temperature"); v.Exists() {
			out, _ = sjson.SetBytes(out, "temperature", v.Float())
		}
		if v := req.Get("text"); v.Exists() {
			out, _ = sjson.SetBytes(out, "text", v.Value())
		}
		if v := req.Get("tool_choice"); v.Exists() {
			out, _ = sjson.SetBytes(out, "tool_choice", v.Value())
		}
		if v := req.Get("tools"); v.Exists() {
			out, _ = sjson.SetBytes(out, "tools", v.Value())
		}
		if v := req.Get("top_logprobs"); v.Exists() {
			out, _ = sjson.SetBytes(out, "top_logprobs", v.Int())
		}
		if v := req.Get("top_p"); v.Exists() {
			out, _ = sjson.SetBytes(out, "top_p", v.Float())
		}
		if v := req.Get("truncation"); v.Exists() {
			out, _ = sjson.SetBytes(out, "truncation", v.String())
		}
		if v := req.Get("user"); v.Exists() {
			out, _ = sjson.SetBytes(out, "user", v.Value())
		}
		if v := req.Get("metadata"); v.Exists() {
			out, _ = sjson.SetBytes(out, "metadata", v.Value())
		}
	}

	// Build output array in the order of the original content blocks.
	outputs := make([][]byte, 0, len(outputItems))
	for _, outputItem := range outputItems {
		var item []byte
		switch outputItem.itemType {
		case "reasoning":
			item = []byte(`{"id":"","type":"reasoning","encrypted_content":"","summary":[]}`)
			item, _ = sjson.SetBytes(item, "id", outputItem.id)
			item, _ = sjson.SetBytes(item, "encrypted_content", outputItem.signature)
			summary := []byte(`{"type":"summary_text","text":""}`)
			summary, _ = sjson.SetBytes(summary, "text", outputItem.text.String())
			item = translatorcommon.SetRawArrayItems(item, "summary", [][]byte{summary})
		case "message":
			item = []byte(`{"id":"","type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":""}],"role":"assistant"}`)
			item, _ = sjson.SetBytes(item, "id", outputItem.id)
			item, _ = sjson.SetBytes(item, "content.0.text", outputItem.text.String())
			if len(outputItem.annotations) > 0 {
				item, _ = sjson.SetBytes(item, "content.0.annotations", outputItem.annotations)
			}
		case "web_search_call":
			item = buildResponsesWebSearchCallItem(outputItem.callID, claudeWebSearchQuery(outputItem.args.String()), outputItem.results)
		case "function_call":
			args := outputItem.args.String()
			if args == "" {
				args = "{}"
			}
			item = []byte(`{"id":"","type":"function_call","status":"completed","arguments":"","call_id":"","name":""}`)
			item, _ = sjson.SetBytes(item, "id", outputItem.id)
			item, _ = sjson.SetBytes(item, "arguments", args)
			item, _ = sjson.SetBytes(item, "call_id", outputItem.callID)
			item, _ = sjson.SetBytes(item, "name", outputItem.name)
		}
		if len(item) > 0 {
			outputs = append(outputs, item)
		}
	}
	if len(outputs) > 0 {
		out, _ = sjson.SetRawBytes(out, "output", translatorcommon.JoinRawArray(outputs))
	}

	// Usage
	total := inputTokens + outputTokens
	out, _ = sjson.SetBytes(out, "usage.input_tokens", inputTokens)
	out, _ = sjson.SetBytes(out, "usage.output_tokens", outputTokens)
	out, _ = sjson.SetBytes(out, "usage.total_tokens", total)
	reasoningLength := 0
	for _, outputItem := range outputItems {
		if outputItem.itemType == "reasoning" {
			reasoningLength += outputItem.text.Len()
		}
	}
	if reasoningLength > 0 {
		// Rough estimate similar to chat completions
		reasoningTokens := int64(reasoningLength / 4)
		if reasoningTokens > 0 {
			out, _ = sjson.SetBytes(out, "usage.output_tokens_details.reasoning_tokens", reasoningTokens)
		}
	}

	return out
}
