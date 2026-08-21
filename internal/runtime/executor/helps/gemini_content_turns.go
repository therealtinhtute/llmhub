package helps

import (
	"bytes"

	"github.com/therealtinhtute/llmhub/internal/util"
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
