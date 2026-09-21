package executor

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"
)

// This file keeps the Fernet-envelope transport check used by the shared
// OpenAI-Responses reasoning encrypted_content sanitizer, which now lives in
// openai_responses_signature.go under its upstream file name (upstream
// CLIProxyAPI internal/runtime/executor/openai_responses_signature.go).
// Upstream routes the check through internal/signature.
// InspectGPTReasoningSignature, which the fork does not have; the equivalent
// stays private here (metaInspectGPTReasoningSignature, ported from upstream
// internal/signature/gpt_validation.go) rather than expanding the signature
// package.

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
