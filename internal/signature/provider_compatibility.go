package signature

import (
	"encoding/base64"
	"strings"
)

type SignatureProvider string

const (
	SignatureProviderUnknown      SignatureProvider = "unknown"
	SignatureProviderClaude       SignatureProvider = "claude"
	SignatureProviderGemini       SignatureProvider = "gemini"
	SignatureProviderGeminiBypass SignatureProvider = "gemini_bypass"
	SignatureProviderGPT          SignatureProvider = "gpt"
)

type SignatureBlockKind string

const (
	SignatureBlockKindUnknown            SignatureBlockKind = "unknown"
	SignatureBlockKindClaudeThinking     SignatureBlockKind = "claude_thinking"
	SignatureBlockKindGeminiModelPart    SignatureBlockKind = "gemini_model_part"
	SignatureBlockKindGeminiFunctionCall SignatureBlockKind = "gemini_function_call"
	SignatureBlockKindGPTReasoning       SignatureBlockKind = "gpt_reasoning"
)

type SignatureCompatibilityAction string

const (
	SignatureActionPreserve                SignatureCompatibilityAction = "preserve"
	SignatureActionDropBlock               SignatureCompatibilityAction = "drop_block"
	SignatureActionDropSignature           SignatureCompatibilityAction = "drop_signature"
	SignatureActionReplaceWithGeminiBypass SignatureCompatibilityAction = "replace_with_gemini_bypass"
	SignatureActionNoCompatibleReplacement SignatureCompatibilityAction = "no_compatible_replacement"
)

type SignatureCompatibilityDecision struct {
	TargetProvider       SignatureProvider
	DetectedProvider     SignatureProvider
	BlockKind            SignatureBlockKind
	Compatible           bool
	Action               SignatureCompatibilityAction
	ReplacementSignature string
	NormalizedSignature  string
	Reason               string
}

const GeminiThoughtSignatureBypass = "skip_thought_signature_validator"

func SignatureProviderFromModelName(modelName string) SignatureProvider {
	model := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case model == "":
		return SignatureProviderUnknown
	case strings.Contains(model, "claude"):
		return SignatureProviderClaude
	case strings.Contains(model, "gemini") || strings.Contains(model, "learnlm"):
		return SignatureProviderGemini
	case strings.Contains(model, "gpt") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4") || strings.HasPrefix(model, "o5"):
		return SignatureProviderGPT
	default:
		return SignatureProviderUnknown
	}
}

func DetectSignatureProvider(rawSignature string) SignatureProvider {
	return DetectSignatureProviderForBlock(rawSignature, SignatureBlockKindUnknown)
}

func DetectSignatureProviderForBlock(rawSignature string, blockKind SignatureBlockKind) SignatureProvider {
	prefixProvider, payload, hasProviderPrefix := SplitSignatureProviderPrefix(rawSignature)
	if hasProviderPrefix && prefixProvider != SignatureProviderUnknown && strings.TrimSpace(payload) == "" {
		return SignatureProviderUnknown
	}
	if hasProviderPrefix && prefixProvider == SignatureProviderGPT {
		return SignatureProviderGPT
	}
	if payload == "" {
		payload = strings.TrimSpace(rawSignature)
	}
	payload = signaturePayloadAfterAnyPrefix(payload)

	if IsGeminiThoughtSignatureBypass(payload) {
		return SignatureProviderGeminiBypass
	}
	// Gemini wire-shape matching runs before the Claude probes: Gemini's
	// exact single-record envelope requirement keeps the families separable,
	// while a loose Claude decodability probe can otherwise claim a native
	// Gemini 3.x field-2 signature.
	if isGeminiNativeSignature(payload) {
		return SignatureProviderGemini
	}
	if IsValidClaudeCAISSignature(payload) || HasDecodableClaudeThinkingSignature(payload) {
		return SignatureProviderClaude
	}
	if hasProviderPrefix && prefixProvider != SignatureProviderUnknown {
		return prefixProvider
	}
	if blockKind == SignatureBlockKindGPTReasoning && payload != "" {
		return SignatureProviderGPT
	}
	return SignatureProviderUnknown
}

func DecideSignatureCompatibility(targetProvider SignatureProvider, rawSignature string, blockKind SignatureBlockKind) SignatureCompatibilityDecision {
	detected := DetectSignatureProviderForBlock(rawSignature, blockKind)
	decision := SignatureCompatibilityDecision{
		TargetProvider:   targetProvider,
		DetectedProvider: detected,
		BlockKind:        blockKind,
		Action:           SignatureActionNoCompatibleReplacement,
		Reason:           "signature is not compatible with target provider",
	}

	if strings.TrimSpace(rawSignature) == "" {
		decision.Reason = "empty signature"
		return decision
	}

	switch targetProvider {
	case SignatureProviderClaude:
		if detected != SignatureProviderClaude {
			decision.Action = SignatureActionDropBlock
			return decision
		}
		if blockKind == SignatureBlockKindClaudeThinking || blockKind == SignatureBlockKindUnknown {
			if normalized, ok := CompatibleAntigravityClaudeThinkingSignature(rawSignature); ok {
				decision.Compatible = true
				decision.Action = SignatureActionPreserve
				decision.NormalizedSignature = normalized
				decision.Reason = "Claude thinking signature is compatible"
				return decision
			}
			if IsValidClaudeCAISSignature(rawSignature) {
				decision.Compatible = true
				decision.Action = SignatureActionPreserve
				decision.NormalizedSignature = signaturePayloadAfterAnyPrefix(rawSignature)
				decision.Reason = "Claude CAIS signature is compatible with Claude upstream"
				return decision
			}
		}
		decision.Action = SignatureActionDropBlock
	case SignatureProviderGemini:
		if detected == SignatureProviderGemini || detected == SignatureProviderGeminiBypass {
			decision.Compatible = true
			decision.Action = SignatureActionPreserve
			decision.NormalizedSignature = SignaturePayloadWithoutProviderPrefix(rawSignature)
			decision.Reason = "Gemini signature is compatible"
			return decision
		}
		if blockKind == SignatureBlockKindGeminiModelPart || blockKind == SignatureBlockKindGeminiFunctionCall {
			decision.Compatible = true
			decision.Action = SignatureActionReplaceWithGeminiBypass
			decision.ReplacementSignature = GeminiThoughtSignatureBypass
			decision.NormalizedSignature = GeminiThoughtSignatureBypass
			decision.Reason = "Gemini accepts the bypass sentinel for incompatible replay signatures"
			return decision
		}
		decision.Action = SignatureActionDropBlock
	case SignatureProviderGPT:
		if detected == SignatureProviderGPT {
			decision.Compatible = true
			decision.Action = SignatureActionPreserve
			decision.NormalizedSignature = SignaturePayloadWithoutProviderPrefix(rawSignature)
			decision.Reason = "GPT signature is compatible"
			return decision
		}
		decision.Action = SignatureActionDropBlock
	default:
		decision.Action = SignatureActionNoCompatibleReplacement
	}
	return decision
}

func DecideSignatureCompatibilityForModel(targetProvider SignatureProvider, targetModel string, rawSignature string, blockKind SignatureBlockKind) SignatureCompatibilityDecision {
	if targetProvider == SignatureProviderUnknown {
		targetProvider = SignatureProviderFromModelName(targetModel)
	}
	return DecideSignatureCompatibility(targetProvider, rawSignature, blockKind)
}

func CompatibleSignatureForProvider(targetProvider SignatureProvider, rawSignature string) (string, bool) {
	decision := DecideSignatureCompatibility(targetProvider, rawSignature, SignatureBlockKindUnknown)
	if !decision.Compatible {
		return "", false
	}
	if decision.ReplacementSignature != "" {
		return decision.ReplacementSignature, true
	}
	return decision.NormalizedSignature, true
}

func CompatibleSignatureForProviderBlock(targetProvider SignatureProvider, rawSignature string, blockKind SignatureBlockKind) (string, bool) {
	decision := DecideSignatureCompatibility(targetProvider, rawSignature, blockKind)
	if !decision.Compatible {
		return "", false
	}
	if decision.ReplacementSignature != "" {
		return decision.ReplacementSignature, true
	}
	return decision.NormalizedSignature, true
}

func CompatibleAntigravityClaudeThinkingSignature(rawSignature string) (string, bool) {
	normalized, err := NormalizeClaudeThinkingSignature(rawSignature, ClaudeSignatureValidationOptions{Strict: true})
	if err != nil {
		return "", false
	}
	return normalized, true
}

func SplitSignatureProviderPrefix(rawSignature string) (SignatureProvider, string, bool) {
	sig := strings.TrimSpace(rawSignature)
	idx := strings.IndexByte(sig, '#')
	if idx < 0 {
		return SignatureProviderUnknown, sig, false
	}
	prefix := strings.ToLower(strings.TrimSpace(sig[:idx]))
	payload := strings.TrimSpace(sig[idx+1:])
	provider := SignatureProviderFromCachePrefix(prefix)
	return provider, payload, provider != SignatureProviderUnknown
}

func SignatureProviderFromCachePrefix(prefix string) SignatureProvider {
	switch strings.ToLower(strings.TrimSpace(prefix)) {
	case "claude", "anthropic", "antigravity-claude":
		return SignatureProviderClaude
	case "gemini", "google", "antigravity-gemini":
		return SignatureProviderGemini
	case "gemini_bypass", "gemini-bypass":
		return SignatureProviderGeminiBypass
	case "gpt", "openai":
		return SignatureProviderGPT
	default:
		return SignatureProviderUnknown
	}
}

func SignaturePayloadWithoutProviderPrefix(rawSignature string) string {
	return signaturePayloadAfterAnyPrefix(rawSignature)
}

func signaturePayloadAfterAnyPrefix(rawSignature string) string {
	sig := strings.TrimSpace(rawSignature)
	if idx := strings.IndexByte(sig, '#'); idx >= 0 {
		return strings.TrimSpace(sig[idx+1:])
	}
	return sig
}

func isGeminiNativeSignature(rawSignature string) bool {
	sig := strings.TrimSpace(rawSignature)
	if sig == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(sig)
	if err != nil || len(decoded) == 0 {
		return false
	}
	if decoded[0] == 0x0a {
		return true
	}
	return isGeminiField2WrappedSignature(decoded)
}
