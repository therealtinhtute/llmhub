package helps

import (
	"strings"

	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
)

// IsNativeCodexRequest checks the client dialect for use inside Codex executors.
//
// Ported from upstream CLIProxyAPI internal/runtime/executor/helps/codex_native.go (f702bc1ac263).
func IsNativeCodexRequest(body []byte, opts cliproxyexecutor.Options) bool {
	for _, format := range []sdktranslator.Format{opts.SourceFormat, cliproxyexecutor.ResponseFormatOrSource(opts)} {
		name := strings.TrimSpace(format.String())
		if !strings.EqualFold(name, sdktranslator.FormatCodex.String()) && !strings.EqualFold(name, sdktranslator.FormatOpenAIResponse.String()) {
			return false
		}
	}
	return util.IsCodexResponsesLiteRequest(body, opts.Headers)
}
