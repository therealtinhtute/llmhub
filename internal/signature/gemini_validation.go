package signature

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protowire"
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

// isGeminiField2WrappedSignature reports whether decoded bytes match Gemini's
// field-2 outer protobuf envelope wrapping a single field-1 opaque payload
// (Tink-style primitive output or an ASCII UUID), as carried by Gemini 3.x models.
func isGeminiField2WrappedSignature(decoded []byte) bool {
	value, ok := consumeGeminiField2Field1Value(decoded)
	if !ok {
		return false
	}
	return isLikelyGeminiOpaquePayload(value) || isASCIIUUIDBytes(value)
}

// consumeGeminiField2Field1Value parses the outer field-2 length-delimited
// record and returns its inner field-1 payload when the envelope is exact.
func consumeGeminiField2Field1Value(decoded []byte) ([]byte, bool) {
	num, typ, n := protowire.ConsumeTag(decoded)
	if n < 0 || num != 2 || typ != protowire.BytesType {
		return nil, false
	}
	offset := n
	container, n := protowire.ConsumeBytes(decoded[offset:])
	if n < 0 {
		return nil, false
	}
	offset += n
	if offset != len(decoded) {
		return nil, false
	}

	num, typ, n = protowire.ConsumeTag(container)
	if n < 0 || num != 1 || typ != protowire.BytesType {
		return nil, false
	}
	containerOffset := n
	value, n := protowire.ConsumeBytes(container[containerOffset:])
	if n < 0 {
		return nil, false
	}
	containerOffset += n
	if containerOffset != len(container) {
		return nil, false
	}
	return value, true
}

// isLikelyGeminiOpaquePayload checks only the Tink prefix-type byte (0x01);
// the four-byte key id that follows is rotated key material, so pinning it
// would reject every signature whenever Google rotates keys.
func isLikelyGeminiOpaquePayload(value []byte) bool {
	return len(value) > 0 && value[0] == 0x01
}

// isASCIIUUIDBytes reports whether decoded bytes form a canonical 36-char
// ASCII UUID (8-4-4-4-12 with hyphens at fixed offsets).
func isASCIIUUIDBytes(decoded []byte) bool {
	if len(decoded) != 36 {
		return false
	}
	for i, b := range decoded {
		switch i {
		case 8, 13, 18, 23:
			if b != '-' {
				return false
			}
		default:
			if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')) {
				return false
			}
		}
	}
	return true
}
