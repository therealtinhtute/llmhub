package executor

import "context"

type downstreamWebsocketContextKey struct{}
type codexCloakingDisabledContextKey struct{}

// WithDownstreamWebsocket marks the current request as coming from a downstream websocket connection.
func WithDownstreamWebsocket(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, downstreamWebsocketContextKey{}, true)
}

// DownstreamWebsocket reports whether the current request originates from a downstream websocket connection.
func DownstreamWebsocket(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	raw := ctx.Value(downstreamWebsocketContextKey{})
	enabled, ok := raw.(bool)
	return ok && enabled
}

// WithCodexCloakingDisabled marks the current request as disabling Codex identity cloaking.
func WithCodexCloakingDisabled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, codexCloakingDisabledContextKey{}, true)
}

// CodexCloakingDisabled reports whether Codex identity cloaking is disabled for this request.
func CodexCloakingDisabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	raw := ctx.Value(codexCloakingDisabledContextKey{})
	disabled, ok := raw.(bool)
	return ok && disabled
}
