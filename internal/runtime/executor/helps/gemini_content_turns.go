package helps

import (
	"bytes"
	"context"

	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/usage"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var emptyGeminiUserTurnJSON = []byte(`{"role":"user","parts":[{"text":""}]}`)

// EnsureGeminiLeadingUserContent prepends an empty user turn when the contents
// array starts with a model turn. Valid user-first payloads are returned unchanged.
func EnsureGeminiLeadingUserContent(payload []byte, path string) []byte {
	contents := util.GetGJSONBytesNoCopy(payload, path)
	if !contents.IsArray() || contents.Get("0.role").String() != "model" {
		return payload
	}
	contentArray := contents.Array()
	if len(contentArray) == 0 {
		return payload
	}

	var contentJSON bytes.Buffer
	contentJSON.Grow(len(contents.Raw) + len(emptyGeminiUserTurnJSON) + 1)
	contentJSON.WriteByte('[')
	contentJSON.Write(emptyGeminiUserTurnJSON)
	for _, content := range contentArray {
		contentJSON.WriteByte(',')
		contentJSON.WriteString(content.Raw)
	}
	contentJSON.WriteByte(']')

	updated, err := sjson.SetRawBytes(payload, path, contentJSON.Bytes())
	if err != nil {
		return payload
	}
	return updated
}

func contentHasFunctionResponse(content gjson.Result) bool {
	parts := content.Get("parts")
	if !parts.IsArray() {
		return false
	}
	for _, part := range parts.Array() {
		if part.Get("functionResponse").Exists() {
			return true
		}
	}
	return false
}

func contentHasFunctionResponseMedia(content gjson.Result) bool {
	parts := content.Get("parts")
	if !parts.IsArray() {
		return false
	}
	for _, part := range parts.Array() {
		for _, nested := range part.Get("functionResponse.parts").Array() {
			if nested.Get("inlineData").Exists() || nested.Get("inline_data").Exists() ||
				nested.Get("fileData").Exists() || nested.Get("file_data").Exists() {
				return true
			}
		}
	}
	return false
}

// EnsureGeminiTrailingUserContent appends an empty user turn when the contents
// array ends with a model turn. A trailing turn that contains a functionResponse
// is preserved because upstream expects the model to answer it. Ported from
// upstream commit 5dc428f39270.
func EnsureGeminiTrailingUserContent(payload []byte, path string) []byte {
	contents := util.GetGJSONBytesNoCopy(payload, path)
	if !contents.IsArray() {
		return payload
	}
	contentArray := contents.Array()
	if len(contentArray) == 0 {
		return payload
	}
	lastContent := contentArray[len(contentArray)-1]
	lastRole := lastContent.Get("role").String()
	if (lastRole != "model" && lastRole != "assistant") || (contentHasFunctionResponse(lastContent) && !contentHasFunctionResponseMedia(lastContent)) {
		return payload
	}

	var contentJSON bytes.Buffer
	contentJSON.Grow(len(contents.Raw) + len(emptyGeminiUserTurnJSON) + 1)
	contentJSON.WriteByte('[')
	for _, content := range contentArray {
		contentJSON.WriteString(content.Raw)
		contentJSON.WriteByte(',')
	}
	contentJSON.Write(emptyGeminiUserTurnJSON)
	contentJSON.WriteByte(']')

	updated, err := sjson.SetRawBytes(payload, path, contentJSON.Bytes())
	if err != nil {
		return payload
	}
	return updated
}

// EnsureGeminiBoundaryUserContent ensures that the contents array at the given path
// both starts and ends with a user turn when sending to Gemini/Antigravity upstreams.
func EnsureGeminiBoundaryUserContent(payload []byte, path string) []byte {
	payload = EnsureGeminiLeadingUserContent(payload, path)
	return EnsureGeminiTrailingUserContent(payload, path)
}

// DropGeminiTrailingThoughtOnlyContent removes trailing model turns made only of
// thought parts, keeping at least one turn. Such a tail comes from a replayed
// trailing reasoning item; dropping it avoids relying only on the synthetic
// empty user turn, which did not prevent Antigravity (#6530) or Vertex (#6630)
// from rejecting the request as ending with a model turn.
func DropGeminiTrailingThoughtOnlyContent(payload []byte, path string) []byte {
	contents := util.GetGJSONBytesNoCopy(payload, path)
	if !contents.IsArray() {
		return payload
	}
	contentArray := contents.Array()
	keep := len(contentArray)
	for keep > 1 && contentIsThoughtOnlyModelTurn(contentArray[keep-1]) {
		keep--
	}
	if keep == len(contentArray) {
		return payload
	}

	contentItems := make([][]byte, 0, keep)
	for _, content := range contentArray[:keep] {
		contentItems = append(contentItems, []byte(content.Raw))
	}
	var contentJSON bytes.Buffer
	contentJSON.WriteByte('[')
	for i, item := range contentItems {
		if i > 0 {
			contentJSON.WriteByte(',')
		}
		contentJSON.Write(item)
	}
	contentJSON.WriteByte(']')

	out, errSet := sjson.SetRawBytes(payload, path, contentJSON.Bytes())
	if errSet != nil {
		return payload
	}
	return out
}

func contentIsThoughtOnlyModelTurn(content gjson.Result) bool {
	role := content.Get("role").String()
	if role != "model" && role != "assistant" {
		return false
	}
	parts := content.Get("parts").Array()
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !part.Get("thought").Bool() || part.Get("functionCall").Exists() || part.Get("functionResponse").Exists() {
			return false
		}
	}
	return true
}

// TranslatedClaudeErrorPayload returns the raw JSON payload when a non-streaming
// translation to Claude format produced a terminal error object instead of a message.
func TranslatedClaudeErrorPayload(responseFormat sdktranslator.Format, out []byte) (string, bool) {
	if responseFormat != sdktranslator.FormatClaude || len(out) == 0 {
		return "", false
	}
	if gjson.GetBytes(out, "type").String() != "error" {
		return "", false
	}
	return string(out), true
}

// TranslatedClaudeStreamErrorPayload returns the raw JSON data payload when a translated
// Claude SSE chunk contains an error event.
func TranslatedClaudeStreamErrorPayload(responseFormat sdktranslator.Format, lines [][]byte) (string, bool) {
	if responseFormat != sdktranslator.FormatClaude {
		return "", false
	}
	for _, chunk := range lines {
		for _, line := range bytes.Split(chunk, []byte("\n")) {
			payload := JSONPayload(line)
			if len(payload) == 0 {
				continue
			}
			if gjson.GetBytes(payload, "type").String() == "error" {
				return string(payload), true
			}
		}
	}
	return "", false
}

// CaptureTranslatedClaudeStreamError retains the first SSE error event payload
// so failure usage and terminal StreamChunk.Err can be emitted after any trailing usage chunk.
func CaptureTranslatedClaudeStreamError(responseFormat sdktranslator.Format, lines [][]byte, retainedErr *error, buildErr func(string) error) {
	if retainedErr == nil || *retainedErr != nil {
		return
	}
	errPayload, okErr := TranslatedClaudeStreamErrorPayload(responseFormat, lines)
	if !okErr {
		return
	}
	if buildErr != nil {
		*retainedErr = buildErr(errPayload)
	}
}

// StopTranslatedClaudeStream publishes failure usage and propagates a terminal StreamChunk.Err
// once after a translated Claude stream emitted an SSE error event.
func StopTranslatedClaudeStreamWithDetail(ctx context.Context, retainedErr *error, reporter *UsageReporter, detail usage.Detail, out chan<- cliproxyexecutor.StreamChunk) bool {
	if retainedErr == nil || *retainedErr == nil {
		return false
	}
	err := *retainedErr
	*retainedErr = nil
	if reporter != nil {
		if detail.TotalTokens > 0 || detail.InputTokens > 0 || detail.OutputTokens > 0 {
			reporter.PublishFailureWithDetail(ctx, detail, err)
		} else {
			reporter.PublishFailure(ctx, err)
		}
	}
	select {
	case out <- cliproxyexecutor.StreamChunk{Err: err}:
		return true
	default:
	}
	select {
	case out <- cliproxyexecutor.StreamChunk{Err: err}:
	case <-ctx.Done():
	}
	return true
}

func StopTranslatedClaudeStream(ctx context.Context, retainedErr *error, reporter *UsageReporter, out chan<- cliproxyexecutor.StreamChunk) bool {
	return StopTranslatedClaudeStreamWithDetail(ctx, retainedErr, reporter, usage.Detail{}, out)
}

// PublishFailureWithDetail records a failure alongside usage statistics.
func (r *UsageReporter) PublishFailureWithDetail(ctx context.Context, detail usage.Detail, errs ...error) {
	if r == nil {
		return
	}
	r.publishWithOutcome(ctx, detail, true, failFromErrors(errs...))
}
