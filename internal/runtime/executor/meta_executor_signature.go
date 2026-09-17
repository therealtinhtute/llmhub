package executor

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/therealtinhtute/llmhub/internal/runtime/executor/helps"
	"github.com/therealtinhtute/llmhub/internal/util"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// This file ports upstream CLIProxyAPI's OpenAI-Responses reasoning
// encrypted_content sanitizer (internal/runtime/executor/
// openai_responses_signature.go, commit 65348b959425) scoped to the meta
// executor. The upstream helper depends on internal/signature.
// InspectGPTReasoningSignature, which the fork does not have; the equivalent
// Fernet-envelope transport check is kept private here
// (metaInspectGPTReasoningSignature, ported from upstream
// internal/signature/gpt_validation.go) rather than expanding the signature
// package outside this phase's surface.

func openaiResponsesReasoningSummaryIsEmpty(summary gjson.Result) bool {
	if !summary.Exists() || summary.Type == gjson.Null {
		return true
	}
	return summary.IsArray() && len(summary.Array()) == 0
}

func promoteOpenAIResponsesReasoningTextToSummary(itemRaw string, content gjson.Result) (string, error) {
	var b strings.Builder
	b.WriteByte('[')
	n := 0
	for _, part := range content.Array() {
		if strings.TrimSpace(part.Get("type").String()) != "reasoning_text" {
			continue
		}
		text := part.Get("text").String()
		if text == "" {
			continue
		}
		partJSON, err := sjson.Set(`{"type":"summary_text"}`, "text", text)
		if err != nil {
			return itemRaw, err
		}
		if n > 0 {
			b.WriteByte(',')
		}
		b.WriteString(partJSON)
		n++
	}
	b.WriteByte(']')
	if n == 0 {
		return itemRaw, nil
	}
	return sjson.SetRaw(itemRaw, "summary", b.String())
}

func sanitizeOpenAIResponsesReasoningEncryptedContent(ctx context.Context, provider string, body []byte) []byte {
	inputResult := util.GetGJSONBytesNoCopy(body, "input")
	if !inputResult.Exists() || !inputResult.IsArray() {
		return body
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "openai responses upstream"
	}

	// Codex backend rejects store=true and does not persist items when store=false.
	// A reasoning item that still carries an id without usable encrypted_content is
	// treated as a store lookup and returns:
	//   Item with id '...' not found. Items are not persisted when `store` is set to false.
	// Strip those orphan ids unless the request explicitly opts into store=true.
	stripOrphanReasoningIDs := !gjson.GetBytes(body, "store").Bool()

	items := inputResult.Array()

	// rebuilt accumulates the edited "input" array as JSON array bytes. It
	// stays nil while no item needs editing so the common case (nothing to
	// sanitize) does no allocation or rebuilding. Edits are applied directly
	// to each item's own raw JSON rather than re-parsing the whole body,
	// keeping the cost proportional to the item being edited.
	var rebuilt []byte
	itemsWritten := 0
	keep := func(raw string) {
		if rebuilt == nil {
			return
		}
		if itemsWritten > 0 {
			rebuilt = append(rebuilt, ',')
		}
		rebuilt = append(rebuilt, raw...)
		itemsWritten++
	}
	startRebuild := func(index int) {
		if rebuilt != nil {
			return
		}
		// First item that needs editing: start the buffer and backfill
		// it with the raw JSON of every preceding item.
		rebuilt = make([]byte, 0, len(inputResult.Raw))
		rebuilt = append(rebuilt, '[')
		for i := range index {
			keep(items[i].Raw)
		}
	}

	for index, item := range items {
		if strings.TrimSpace(item.Get("type").String()) != "reasoning" {
			keep(item.Raw)
			continue
		}

		encryptedContent := item.Get("encrypted_content")
		itemID := strings.TrimSpace(item.Get("id").String())
		if itemID == "" {
			itemID = fmt.Sprintf("input[%d]", index)
		}

		nextItem := item.Raw
		changed := false

		// Official Codex schema sets maxItems: 0 on reasoning.content. Third-party
		// channels replay cleartext thinking there; promote it into summary when
		// summary is empty, then force content to [].
		content := item.Get("content")
		if content.IsArray() && len(content.Array()) > 0 {
			if openaiResponsesReasoningSummaryIsEmpty(item.Get("summary")) {
				promoted, errPromote := promoteOpenAIResponsesReasoningTextToSummary(nextItem, content)
				if errPromote != nil {
					helps.LogWithRequestID(ctx).Debugf("%s: failed to promote reasoning_text into summary at input[%d]: %v", provider, index, errPromote)
				} else {
					nextItem = promoted
				}
			}
			cleared, errClear := sjson.SetRaw(nextItem, "content", "[]")
			if errClear != nil {
				helps.LogWithRequestID(ctx).Debugf("%s: failed to clear reasoning content at input[%d]: %v", provider, index, errClear)
			} else {
				nextItem = cleared
				changed = true
				helps.LogWithRequestID(ctx).Debugf("%s: cleared reasoning content at input[%d] item_id=%q", provider, index, itemID)
			}
		}

		if !encryptedContent.Exists() {
			if stripOrphanReasoningIDs && item.Get("id").Exists() {
				dropped, err := sjson.Delete(nextItem, "id")
				if err != nil {
					helps.LogWithRequestID(ctx).Debugf("%s: failed to drop orphan reasoning id at input[%d]: %v", provider, index, err)
				} else {
					nextItem = dropped
					changed = true
					helps.LogWithRequestID(ctx).Debugf("%s: dropped orphan reasoning id at input[%d] item_id=%q reason=missing encrypted_content with store disabled", provider, index, itemID)
				}
			}
			if !changed {
				keep(item.Raw)
				continue
			}
			startRebuild(index)
			keep(nextItem)
			continue
		}

		reason := ""
		switch encryptedContent.Type {
		case gjson.String:
			rawSignature := encryptedContent.String()
			if rawSignature != strings.TrimSpace(rawSignature) {
				reason = "encrypted_content has leading or trailing whitespace"
			} else if err := metaInspectGPTReasoningSignature(rawSignature); err != nil {
				reason = err.Error()
			}
		case gjson.Null:
			reason = "encrypted_content is null"
		default:
			reason = fmt.Sprintf("encrypted_content must be a string, got %s", encryptedContent.Type.String())
		}
		if reason == "" {
			if !changed {
				keep(item.Raw)
				continue
			}
			startRebuild(index)
			keep(nextItem)
			continue
		}

		dropped, err := sjson.Delete(nextItem, "encrypted_content")
		if err != nil {
			helps.LogWithRequestID(ctx).Debugf("%s: failed to drop invalid reasoning encrypted_content at input[%d]: %v", provider, index, err)
			if !changed {
				keep(item.Raw)
				continue
			}
			startRebuild(index)
			keep(nextItem)
			continue
		}
		nextItem = dropped
		changed = true
		if stripOrphanReasoningIDs && item.Get("id").Exists() {
			if nextID, errID := sjson.Delete(nextItem, "id"); errID != nil {
				helps.LogWithRequestID(ctx).Debugf("%s: failed to drop reasoning id after invalid encrypted_content at input[%d]: %v", provider, index, errID)
			} else {
				nextItem = nextID
			}
		}

		startRebuild(index)
		keep(nextItem)

		helps.LogWithRequestID(ctx).Debugf("%s: dropped invalid reasoning encrypted_content at input[%d] item_id=%q reason=%s", provider, index, itemID, reason)
	}

	if rebuilt == nil {
		return body
	}
	rebuilt = append(rebuilt, ']')

	updated, err := sjson.SetRawBytes(body, "input", rebuilt)
	if err != nil {
		helps.LogWithRequestID(ctx).Debugf("%s: failed to rebuild input array while sanitizing reasoning encrypted_content: %v", provider, err)
		return body
	}
	return updated
}

const metaMaxGPTReasoningSignatureLen = 32 * 1024 * 1024

// metaInspectGPTReasoningSignature validates the Fernet-like outer format used
// by GPT/Codex reasoning encrypted_content. This is only a transport-shape
// check; it does not prove decryptability.
// Ported from upstream CLIProxyAPI internal/signature/gpt_validation.go
// (InspectGPTReasoningSignature).
func metaInspectGPTReasoningSignature(rawSignature string) error {
	sig := strings.TrimSpace(rawSignature)
	if sig == "" {
		return fmt.Errorf("empty GPT reasoning signature")
	}
	if len(sig) > metaMaxGPTReasoningSignatureLen {
		return fmt.Errorf("GPT reasoning signature exceeds maximum length (%d bytes)", metaMaxGPTReasoningSignatureLen)
	}
	// The literal prefix is the cheapest discriminator and rejects every other
	// provider's envelope outright, so it runs before the full charset scan.
	if !strings.HasPrefix(sig, "gAAAA") {
		return fmt.Errorf("invalid GPT reasoning signature: expected gAAAA prefix")
	}
	if index, r, ok := firstInvalidGPTReasoningSignatureChar(sig); ok {
		return fmt.Errorf("invalid GPT reasoning signature: contains non-base64url character U+%04X at byte %d", r, index)
	}

	decoded, err := decodeGPTReasoningSignature(sig)
	if err != nil {
		return err
	}
	if len(decoded) < 73 {
		return fmt.Errorf("invalid GPT reasoning signature: decoded payload too short")
	}
	if decoded[0] != 0x80 {
		return fmt.Errorf("invalid GPT reasoning signature: expected version 0x80, got 0x%02x", decoded[0])
	}

	ciphertextLen := len(decoded) - 1 - 8 - 16 - 32
	if ciphertextLen <= 0 || ciphertextLen%16 != 0 {
		return fmt.Errorf("invalid GPT reasoning signature: ciphertext length %d is not a positive AES block multiple", ciphertextLen)
	}
	return nil
}

func decodeGPTReasoningSignature(sig string) ([]byte, error) {
	if decoded, err := base64.RawURLEncoding.DecodeString(sig); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(sig); err == nil {
		return decoded, nil
	}
	return nil, fmt.Errorf("invalid GPT reasoning signature: base64url decode failed")
}

// gptReasoningSignatureCharSet is the base64url alphabet, padding included.
// Upstream shares base64AlphabetSet across validators; the fork only needs the
// GPT table here.
var gptReasoningSignatureCharSet = func() [256]bool {
	var set [256]bool
	for i := 'A'; i <= 'Z'; i++ {
		set[i] = true
	}
	for i := 'a'; i <= 'z'; i++ {
		set[i] = true
	}
	for i := '0'; i <= '9'; i++ {
		set[i] = true
	}
	for _, c := range "-_=" {
		set[c] = true
	}
	return set
}()

// firstInvalidGPTReasoningSignatureChar scans bytes against a lookup table:
// every legal character is ASCII, and a comparison chain mispredicts on nearly
// every byte of a multi-kilobyte reasoning blob. The offending rune is decoded
// only for the error message.
func firstInvalidGPTReasoningSignatureChar(sig string) (int, rune, bool) {
	for index := 0; index < len(sig); index++ {
		if !gptReasoningSignatureCharSet[sig[index]] {
			r, _ := utf8.DecodeRuneInString(sig[index:])
			return index, r, true
		}
	}
	return 0, 0, false
}
