package claude

import (
	"encoding/base64"
	"fmt"

	"github.com/therealtinhtute/llmhub/internal/cache"
	internalsignature "github.com/therealtinhtute/llmhub/internal/signature"
)

const maxBypassSignatureLen = internalsignature.MaxClaudeThinkingSignatureLen

type claudeSignatureTree = internalsignature.ClaudeSignatureTree

func StripEmptySignatureThinkingBlocks(payload []byte) []byte {
	return internalsignature.StripInvalidClaudeThinkingBlocks(payload, internalsignature.ClaudeSignatureValidationOptions{PrefixOnly: true})
}

func hasValidClaudeSignature(rawSignature string) bool {
	return internalsignature.HasClaudeThinkingSignaturePrefix(rawSignature)
}

func ValidateClaudeBypassSignatures(inputRawJSON []byte) error {
	return internalsignature.ValidateClaudeThinkingSignatures(inputRawJSON, claudeBypassValidationOptions())
}

func normalizeClaudeBypassSignature(rawSignature string) (string, error) {
	return internalsignature.NormalizeClaudeThinkingSignature(rawSignature, claudeBypassValidationOptions())
}

func validateDoubleLayerSignature(rawSignature string) error {
	decoded, err := base64.StdEncoding.DecodeString(rawSignature)
	if err != nil {
		return fmt.Errorf("invalid double-layer signature: base64 decode failed: %w", err)
	}
	if len(decoded) == 0 {
		return fmt.Errorf("invalid double-layer signature: empty after decode")
	}
	if decoded[0] != 'E' {
		return fmt.Errorf("invalid double-layer signature: inner does not start with 'E', got 0x%02x", decoded[0])
	}
	return validateSingleLayerSignatureContent(string(decoded), 2)
}

func validateSingleLayerSignature(rawSignature string) error {
	return validateSingleLayerSignatureContent(rawSignature, 1)
}

func validateSingleLayerSignatureContent(rawSignature string, encodingLayers int) error {
	decoded, err := base64.StdEncoding.DecodeString(rawSignature)
	if err != nil {
		return fmt.Errorf("invalid single-layer signature: base64 decode failed: %w", err)
	}
	if len(decoded) == 0 {
		return fmt.Errorf("invalid single-layer signature: empty after decode")
	}
	if decoded[0] != 0x12 {
		return fmt.Errorf("invalid Claude signature: expected first byte 0x12, got 0x%02x", decoded[0])
	}
	if !cache.SignatureBypassStrictMode() {
		return nil
	}
	_, err = internalsignature.InspectClaudeSignaturePayload(decoded, encodingLayers)
	return err
}

func inspectDoubleLayerSignature(rawSignature string) (*claudeSignatureTree, error) {
	return internalsignature.InspectClaudeDoubleLayerSignature(rawSignature)
}

func inspectSingleLayerSignature(rawSignature string) (*claudeSignatureTree, error) {
	return inspectSingleLayerSignatureWithLayers(rawSignature, 1)
}

func inspectSingleLayerSignatureWithLayers(rawSignature string, encodingLayers int) (*claudeSignatureTree, error) {
	decoded, err := base64.StdEncoding.DecodeString(rawSignature)
	if err != nil {
		return nil, fmt.Errorf("invalid single-layer signature: base64 decode failed: %w", err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("invalid single-layer signature: empty after decode")
	}
	return internalsignature.InspectClaudeSignaturePayload(decoded, encodingLayers)
}

func inspectClaudeSignaturePayload(payload []byte, encodingLayers int) (*claudeSignatureTree, error) {
	return internalsignature.InspectClaudeSignaturePayload(payload, encodingLayers)
}

func claudeBypassValidationOptions() internalsignature.ClaudeSignatureValidationOptions {
	return internalsignature.ClaudeSignatureValidationOptions{Strict: cache.SignatureBypassStrictMode()}
}
