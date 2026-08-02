package signature

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protowire"
)

func buildTestClaudeSignaturePayload(t *testing.T, channelID uint64, modelText string) []byte {
	t.Helper()

	channelBlock := []byte{}
	channelBlock = protowire.AppendTag(channelBlock, 1, protowire.VarintType)
	channelBlock = protowire.AppendVarint(channelBlock, channelID)
	if modelText != "" {
		channelBlock = protowire.AppendTag(channelBlock, 6, protowire.BytesType)
		channelBlock = protowire.AppendString(channelBlock, modelText)
	}

	container := []byte{}
	container = protowire.AppendTag(container, 1, protowire.BytesType)
	container = protowire.AppendBytes(container, channelBlock)
	container = protowire.AppendTag(container, 2, protowire.BytesType)
	container = protowire.AppendBytes(container, bytes.Repeat([]byte{0x11}, 12))
	container = protowire.AppendTag(container, 3, protowire.BytesType)
	container = protowire.AppendBytes(container, bytes.Repeat([]byte{0x22}, 12))
	container = protowire.AppendTag(container, 4, protowire.BytesType)
	container = protowire.AppendBytes(container, bytes.Repeat([]byte{0x33}, 48))

	payload := []byte{}
	payload = protowire.AppendTag(payload, 2, protowire.BytesType)
	payload = protowire.AppendBytes(payload, container)
	payload = protowire.AppendTag(payload, 3, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 1)
	return payload
}

func buildTestClaudeSignature(t *testing.T) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(buildTestClaudeSignaturePayload(t, 12, "claude-sonnet-4-6"))
}

func buildTestClaudeCAISSignature(t *testing.T, modelText string, contextID string) string {
	t.Helper()

	channelBlock := []byte{}
	channelBlock = protowire.AppendTag(channelBlock, 1, protowire.VarintType)
	channelBlock = protowire.AppendVarint(channelBlock, 12)
	channelBlock = protowire.AppendTag(channelBlock, 3, protowire.VarintType)
	channelBlock = protowire.AppendVarint(channelBlock, 1)
	channelBlock = protowire.AppendTag(channelBlock, 5, protowire.BytesType)
	channelBlock = protowire.AppendBytes(channelBlock, bytes.Repeat([]byte{0x44}, 64))
	channelBlock = protowire.AppendTag(channelBlock, 6, protowire.BytesType)
	channelBlock = protowire.AppendString(channelBlock, modelText)
	channelBlock = protowire.AppendTag(channelBlock, 8, protowire.BytesType)
	channelBlock = protowire.AppendString(channelBlock, "thinking")
	if contextID != "" {
		channelBlock = protowire.AppendTag(channelBlock, 11, protowire.BytesType)
		channelBlock = protowire.AppendString(channelBlock, contextID)
	}

	container := []byte{}
	container = protowire.AppendTag(container, 1, protowire.BytesType)
	container = protowire.AppendBytes(container, channelBlock)

	payload := []byte{}
	payload = protowire.AppendTag(payload, 1, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 1)
	payload = protowire.AppendTag(payload, 2, protowire.BytesType)
	payload = protowire.AppendBytes(payload, container)

	sig := base64.StdEncoding.EncodeToString(payload)
	if sig[0] != 'C' {
		t.Fatalf("test setup: expected C-prefixed CAIS signature, got %q", sig[0])
	}
	return sig
}

func TestInspectClaudeCAISSignature_ValidEnvelope(t *testing.T) {
	contextID := "123e4567-e89b-12d3-a456-426614174000"
	sig := buildTestClaudeCAISSignature(t, "claude-sonnet-4-6", contextID)

	info, err := InspectClaudeCAISSignature("claude#" + sig)
	if err != nil {
		t.Fatalf("expected CAIS signature to inspect cleanly: %v", err)
	}
	if info.FirstByte != 0x08 {
		t.Fatalf("first byte = 0x%02x, want 0x08", info.FirstByte)
	}
	if info.EnvelopeVersion != 1 {
		t.Fatalf("envelope version = %d, want 1", info.EnvelopeVersion)
	}
	if info.ChannelID != 12 {
		t.Fatalf("channel id = %d, want 12", info.ChannelID)
	}
	if info.ModelText != "claude-sonnet-4-6" {
		t.Fatalf("model text = %q", info.ModelText)
	}
	if info.BlockKind != "thinking" {
		t.Fatalf("block kind = %q", info.BlockKind)
	}
	if info.ContextID != contextID {
		t.Fatalf("context id = %q", info.ContextID)
	}
	if info.SignatureLen != 64 {
		t.Fatalf("signature len = %d, want 64", info.SignatureLen)
	}
}

func TestInspectClaudeCAISSignature_RejectsInvalidModelAndContext(t *testing.T) {
	if _, err := InspectClaudeCAISSignature(buildTestClaudeCAISSignature(t, "gemini-2.5-pro", "")); err == nil || !strings.Contains(err.Error(), "claude-") {
		t.Fatalf("expected non-Claude model to be rejected, got %v", err)
	}
	if _, err := InspectClaudeCAISSignature(buildTestClaudeCAISSignature(t, "claude-sonnet-4-6", "not-a-uuid")); err == nil || !strings.Contains(err.Error(), "canonical UUID") {
		t.Fatalf("expected invalid context UUID to be rejected, got %v", err)
	}
}

func TestSignatureProviderDetection_ClassifiesClaudeCAISAndGemini(t *testing.T) {
	claudeSig := buildTestClaudeSignature(t)
	caisSig := buildTestClaudeCAISSignature(t, "claude-sonnet-4-6", "")
	geminiSig := base64.StdEncoding.EncodeToString(append([]byte{0x0a}, bytes.Repeat([]byte{0x55}, 48)...))

	tests := []struct {
		name string
		sig  string
		want SignatureProvider
	}{
		{"claude classic", claudeSig, SignatureProviderClaude},
		{"claude CAIS", caisSig, SignatureProviderClaude},
		{"prefixed CAIS", "model-cache#" + caisSig, SignatureProviderClaude},
		{"prefixed classic", "model-cache#" + claudeSig, SignatureProviderClaude},
		{"gemini native", geminiSig, SignatureProviderGemini},
		{"gemini bypass", GeminiThoughtSignatureBypass, SignatureProviderGeminiBypass},
		{"gpt prefix", "gpt#reasoning-token", SignatureProviderGPT},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectSignatureProvider(tt.sig); got != tt.want {
				t.Fatalf("provider = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSignatureCompatibility_ProviderActions(t *testing.T) {
	claudeSig := buildTestClaudeSignature(t)
	geminiSig := base64.StdEncoding.EncodeToString(append([]byte{0x0a}, bytes.Repeat([]byte{0x55}, 48)...))

	claudeDecision := DecideSignatureCompatibility(SignatureProviderClaude, claudeSig, SignatureBlockKindClaudeThinking)
	if !claudeDecision.Compatible || claudeDecision.Action != SignatureActionPreserve || claudeDecision.NormalizedSignature == claudeSig {
		t.Fatalf("expected Claude signature to be preserved in normalized R-form: %+v", claudeDecision)
	}

	crossProviderDecision := DecideSignatureCompatibility(SignatureProviderClaude, geminiSig, SignatureBlockKindClaudeThinking)
	if crossProviderDecision.Compatible || crossProviderDecision.Action != SignatureActionDropBlock {
		t.Fatalf("expected Gemini signature to be dropped for Claude thinking: %+v", crossProviderDecision)
	}

	geminiDecision := DecideSignatureCompatibility(SignatureProviderGemini, claudeSig, SignatureBlockKindGeminiModelPart)
	if !geminiDecision.Compatible || geminiDecision.Action != SignatureActionReplaceWithGeminiBypass || geminiDecision.ReplacementSignature != GeminiThoughtSignatureBypass {
		t.Fatalf("expected Claude signature to be replaced by Gemini bypass for Gemini model parts: %+v", geminiDecision)
	}
}

func TestSanitizeClaudeMessagesSignaturesForTarget_DropsIncompatibleProviderBlock(t *testing.T) {
	geminiSig := base64.StdEncoding.EncodeToString(append([]byte{0x0a}, bytes.Repeat([]byte{0x55}, 48)...))
	payload := []byte(`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"x","signature":"` + geminiSig + `"},{"type":"text","text":"answer"}]}]}`)

	output, report := SanitizeClaudeMessagesForClaudeUpstream(payload, "claude-sonnet-4-6")
	if !report.Changed || report.DroppedBlocks != 1 {
		t.Fatalf("expected one dropped block, got %+v", report)
	}
	content := gjson.GetBytes(output, "messages.0.content").Array()
	if len(content) != 1 || content[0].Get("text").String() != "answer" {
		t.Fatalf("unexpected sanitized content: %s", gjson.GetBytes(output, "messages.0.content").Raw)
	}
}
