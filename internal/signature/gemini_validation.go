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
	var validationErr error
	contents.ForEach(func(contentIndex, content gjson.Result) bool {
		i := int(contentIndex.Int())
		parts := content.Get("parts")
		if !parts.IsArray() {
			if len(pending) > 0 {
				validationErr = fmt.Errorf(
					"%s[%d]: content appears before %d pending functionResponse part(s)",
					contentsPath,
					i,
					len(pending),
				)
			}
			return validationErr == nil
		}

		var calls []geminiFunctionCallRef
		var responses []geminiFunctionResponseRef
		parts.ForEach(func(partIndex, part gjson.Result) bool {
			j := int(partIndex.Int())
			partPath := fmt.Sprintf("%s[%d].parts[%d]", contentsPath, i, j)
			if call := part.Get("functionCall"); call.Exists() {
				if call.Get("name").String() == "" {
					validationErr = fmt.Errorf("%s: missing functionCall.name", partPath)
					return false
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
			return true
		})
		if validationErr != nil {
			return false
		}

		switch {
		case len(calls) > 0 && len(responses) > 0:
			validationErr = fmt.Errorf(
				"%s[%d]: functionCall and functionResponse parts must not be interleaved in the same content",
				contentsPath,
				i,
			)
		case len(calls) > 0 && len(pending) > 0:
			validationErr = fmt.Errorf(
				"%s[%d]: functionCall appears before %d pending functionResponse part(s)",
				contentsPath,
				i,
				len(pending),
			)
		case len(calls) > 0:
			pending = calls
			return true
		case len(responses) == 0 && len(pending) > 0:
			validationErr = fmt.Errorf(
				"%s[%d]: content appears before %d pending functionResponse part(s)",
				contentsPath,
				i,
				len(pending),
			)
		case len(responses) == 0:
			return true
		case len(pending) == 0:
			validationErr = fmt.Errorf("%s[%d]: functionResponse without preceding functionCall", contentsPath, i)
		case len(responses) != len(pending):
			validationErr = fmt.Errorf(
				"%s[%d]: functionResponse count %d does not match pending functionCall count %d",
				contentsPath,
				i,
				len(responses),
				len(pending),
			)
		}
		if validationErr != nil {
			return false
		}

		for responseIndex, responseRef := range responses {
			partPath := responseRef.path
			response := responseRef.part.Get("functionResponse")
			call := pending[responseIndex]
			responseID := response.Get("id").String()
			responseName := response.Get("name").String()

			switch {
			case call.id != "" && responseID == "":
				validationErr = fmt.Errorf("%s: missing functionResponse.id for %s", partPath, call.path)
			case call.id != "" && responseID != call.id:
				validationErr = fmt.Errorf(
					"%s: functionResponse.id %q does not match functionCall.id %q at %s",
					partPath,
					responseID,
					call.id,
					call.path,
				)
			case responseName == "":
				validationErr = fmt.Errorf("%s: missing functionResponse.name", partPath)
			case call.name != "" && responseName != call.name:
				validationErr = fmt.Errorf(
					"%s: functionResponse.name %q does not match functionCall.name %q at %s",
					partPath,
					responseName,
					call.name,
					call.path,
				)
			}
			if validationErr != nil {
				return false
			}
		}

		pending = nil
		return true
	})
	return validationErr
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
