package signature

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

type geminiFunctionCallRef struct {
	id   string
	name string
	path string
}

type geminiFunctionResponseRef struct {
	part gjson.Result
	path string
}

// ValidateGeminiFunctionCallPairing validates the replay shape around Gemini
// functionCall and functionResponse parts. It checks id/name pairing and
// prevents response parts from being interleaved inside the same content as
// function calls. It allows a final pending functionCall group because callers
// may validate a freshly returned model step before tool outputs exist.
func ValidateGeminiFunctionCallPairing(inputRawJSON []byte) error {
	contents, contentsPath := geminiContents(inputRawJSON)
	if !contents.IsArray() {
		return nil
	}

	var pending []geminiFunctionCallRef
	contentResults := contents.Array()
	for i := 0; i < len(contentResults); i++ {
		parts := contentResults[i].Get("parts")
		if !parts.IsArray() {
			if len(pending) > 0 {
				return fmt.Errorf("%s[%d]: content appears before %d pending functionResponse part(s)", contentsPath, i, len(pending))
			}
			continue
		}

		var calls []geminiFunctionCallRef
		var responses []geminiFunctionResponseRef
		partResults := parts.Array()
		for j := 0; j < len(partResults); j++ {
			part := partResults[j]
			partPath := fmt.Sprintf("%s[%d].parts[%d]", contentsPath, i, j)
			if call := part.Get("functionCall"); call.Exists() {
				if call.Get("name").String() == "" {
					return fmt.Errorf("%s: missing functionCall.name", partPath)
				}
				calls = append(calls, geminiFunctionCallRef{
					id:   call.Get("id").String(),
					name: call.Get("name").String(),
					path: partPath,
				})
			}
			if response := part.Get("functionResponse"); response.Exists() {
				responses = append(responses, geminiFunctionResponseRef{
					part: part,
					path: partPath,
				})
			}
		}

		if len(calls) > 0 && len(responses) > 0 {
			return fmt.Errorf("%s[%d]: functionCall and functionResponse parts must not be interleaved in the same content", contentsPath, i)
		}

		if len(calls) > 0 {
			if len(pending) > 0 {
				return fmt.Errorf("%s[%d]: functionCall appears before %d pending functionResponse part(s)", contentsPath, i, len(pending))
			}
			pending = calls
			continue
		}

		if len(responses) == 0 {
			if len(pending) > 0 {
				return fmt.Errorf("%s[%d]: content appears before %d pending functionResponse part(s)", contentsPath, i, len(pending))
			}
			continue
		}
		if len(pending) == 0 {
			return fmt.Errorf("%s[%d]: functionResponse without preceding functionCall", contentsPath, i)
		}
		if len(responses) != len(pending) {
			return fmt.Errorf("%s[%d]: functionResponse count %d does not match pending functionCall count %d", contentsPath, i, len(responses), len(pending))
		}

		for j := 0; j < len(responses); j++ {
			partPath := responses[j].path
			response := responses[j].part.Get("functionResponse")
			call := pending[j]
			responseID := response.Get("id").String()
			responseName := response.Get("name").String()

			if call.id != "" && responseID == "" {
				return fmt.Errorf("%s: missing functionResponse.id for %s", partPath, call.path)
			}
			if call.id != "" && responseID != call.id {
				return fmt.Errorf("%s: functionResponse.id %q does not match functionCall.id %q at %s", partPath, responseID, call.id, call.path)
			}
			if responseName == "" {
				return fmt.Errorf("%s: missing functionResponse.name", partPath)
			}
			if call.name != "" && responseName != call.name {
				return fmt.Errorf("%s: functionResponse.name %q does not match functionCall.name %q at %s", partPath, responseName, call.name, call.path)
			}
		}

		pending = nil
	}

	return nil
}

func geminiContents(inputRawJSON []byte) (gjson.Result, string) {
	if contents := gjson.GetBytes(inputRawJSON, "contents"); contents.Exists() {
		return contents, "contents"
	}
	return gjson.GetBytes(inputRawJSON, "request.contents"), "request.contents"
}

func IsGeminiThoughtSignatureBypass(rawSignature string) bool {
	switch strings.TrimSpace(rawSignature) {
	case "skip_thought_signature_validator", "context_engineering_is_the_way_to_go":
		return true
	default:
		return false
	}
}
