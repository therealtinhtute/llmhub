package helps

import (
	"bytes"

	"github.com/therealtinhtute/llmhub/internal/util"
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
	if (lastRole != "model" && lastRole != "assistant") || contentHasFunctionResponse(lastContent) {
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
