package helps

import (
	"context"

	"github.com/therealtinhtute/llmhub/internal/logging"
	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// EnsureSessionContext ensures that ctx carries the internal session identity
// for $CPA-SESSION-ID expansion in custom headers and for executor-side
// session/cascade resolution.
// Ported from upstream CLIProxyAPI internal/runtime/executor/helps/cpa_session.go (v7.3.3).
func EnsureSessionContext(ctx context.Context, opts cliproxyexecutor.Options, payload []byte) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if id := util.SessionIDFromContext(ctx); id != "" {
		return ctx
	}
	if util.HasExplicitSessionID(ctx) {
		return ctx
	}
	if meta := logging.GetClientRequestMetadata(ctx); meta.SessionID != "" {
		return util.WithSessionID(ctx, meta.SessionID)
	}
	evalPayload := opts.OriginalRequest
	if len(evalPayload) == 0 {
		evalPayload = payload
	}
	canonical := cliproxyauth.CanonicalSessionID(opts.Headers, evalPayload, opts.Metadata)
	if canonical != "" {
		return util.WithSessionID(ctx, canonical)
	}
	return ctx
}
