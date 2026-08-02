package signature

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ClaudeMessagesSignatureSanitizeOptions struct {
	TargetProvider SignatureProvider
	TargetModel    string
}

type SignatureSanitizeReport struct {
	Changed            bool
	PreservedBlocks    int
	DroppedBlocks      int
	DroppedSignatures  int
	ReplacedSignatures int
}

func SanitizeClaudeMessagesSignaturesForModel(payload []byte, targetModel string) ([]byte, SignatureSanitizeReport) {
	return SanitizeClaudeMessagesSignaturesForTarget(payload, ClaudeMessagesSignatureSanitizeOptions{
		TargetProvider: SignatureProviderFromModelName(targetModel),
		TargetModel:    targetModel,
	})
}

func SanitizeClaudeMessagesForClaudeUpstream(payload []byte, targetModel string) ([]byte, SignatureSanitizeReport) {
	return SanitizeClaudeMessagesSignaturesForTarget(payload, ClaudeMessagesSignatureSanitizeOptions{
		TargetProvider: SignatureProviderClaude,
		TargetModel:    targetModel,
	})
}

func SanitizeClaudeMessagesSignaturesForTarget(payload []byte, opts ClaudeMessagesSignatureSanitizeOptions) ([]byte, SignatureSanitizeReport) {
	targetProvider := opts.TargetProvider
	if targetProvider == SignatureProviderUnknown {
		targetProvider = SignatureProviderFromModelName(opts.TargetModel)
	}
	if targetProvider == SignatureProviderUnknown {
		return payload, SignatureSanitizeReport{}
	}

	messages := gjson.GetBytes(payload, "messages")
	if !messages.IsArray() {
		return payload, SignatureSanitizeReport{}
	}

	report := SignatureSanitizeReport{}
	messageResults := messages.Array()
	for i, msg := range messageResults {
		content := msg.Get("content")
		if !content.IsArray() {
			continue
		}

		kept := make([]string, 0, len(content.Array()))
		changedContent := false
		for _, part := range content.Array() {
			if part.Get("type").String() != "thinking" {
				kept = append(kept, part.Raw)
				continue
			}

			rawSignature := strings.TrimSpace(part.Get("signature").String())
			decision := DecideSignatureCompatibility(targetProvider, rawSignature, SignatureBlockKindClaudeThinking)
			switch decision.Action {
			case SignatureActionPreserve:
				normalized := decision.NormalizedSignature
				if normalized == "" {
					normalized = rawSignature
				}
				if normalized != rawSignature {
					updated, err := sjson.Set(part.Raw, "signature", normalized)
					if err == nil {
						part.Raw = updated
						changedContent = true
					}
				}
				report.PreservedBlocks++
				kept = append(kept, part.Raw)
			case SignatureActionDropSignature:
				updated, err := sjson.Delete(part.Raw, "signature")
				if err == nil {
					part.Raw = updated
					changedContent = true
					report.DroppedSignatures++
				}
				kept = append(kept, part.Raw)
			case SignatureActionReplaceWithGeminiBypass:
				updated, err := sjson.Set(part.Raw, "signature", decision.ReplacementSignature)
				if err == nil {
					part.Raw = updated
					changedContent = true
					report.ReplacedSignatures++
				}
				kept = append(kept, part.Raw)
			default:
				changedContent = true
				report.DroppedBlocks++
			}
		}

		if !changedContent {
			continue
		}
		report.Changed = true
		contentPath := fmt.Sprintf("messages.%d.content", i)
		if len(kept) == 0 {
			payload, _ = sjson.SetRawBytes(payload, contentPath, []byte("[]"))
			continue
		}
		payload, _ = sjson.SetRawBytes(payload, contentPath, []byte("["+strings.Join(kept, ",")+"]"))
	}

	return payload, report
}
