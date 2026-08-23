package executor

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// codexBootstrapMaxBufferedEvents bounds how many handshake metadata events may be held
// back while probing for an upstream rejection embedded in an HTTP 200 stream. The websocket
// transport prefixes response events with codex.response.metadata and codex.rate_limits frames,
// so the limit must comfortably exceed the four handshake frames observed in practice. Once the
// limit is reached the stream is released and the original unbuffered semantics apply.
const codexBootstrapMaxBufferedEvents = 16

// isCodexHandshakeMetadataEvent reports whether an event carries no generated output and is
// therefore safe to hold back before the downstream response headers are committed. Keeping a
// type allow-list rather than a fixed event count matters for the websocket transport, where
// handshake frames arrive before response.created and would otherwise exhaust a small counter
// before the rejection event is seen.
func isCodexHandshakeMetadataEvent(eventType string) bool {
	switch eventType {
	case "response.created", "response.in_progress", "codex.rate_limits", "codex.response.metadata":
		return true
	default:
		return false
	}
}

// newCodexBootstrapOverloadErr reports a buffered overload rejection with its real status.
//
// The status is deliberately produced here instead of in the shared terminal-failure mapping:
// that mapping serves the unbuffered path, where the rejection is delivered in-stream and a
// status change would alter cooldown classification and retry-after parsing for everyone.
// Keeping 503 scoped to this path means disabling the feature restores previous behaviour exactly.
func newCodexBootstrapOverloadErr(body []byte) statusErr {
	return newCodexStatusErr(http.StatusServiceUnavailable, body)
}

// isCodexOverloadBootstrapFailure reports whether a terminal failure delivered inside an HTTP 200
// stream is a transient capacity rejection that a different credential may be able to serve.
// Only these failures justify replacing the whole attempt during bootstrap; every other terminal
// failure keeps the original in-stream delivery semantics so downstream behaviour is unchanged.
func isCodexOverloadBootstrapFailure(body []byte) bool {
	errorType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "error.type").String()))
	errorCode := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "error.code").String()))
	switch {
	case errorType == "service_unavailable_error", errorCode == "server_is_overloaded":
		return true
	case errorType == "rate_limit_error", errorCode == "rate_limit_exceeded":
		return true
	default:
		return false
	}
}

// codexBootstrapTerminalFailure extracts a terminal error payload from a buffered stream event.
// Mirrors the local unbuffered handling of "error" and "response.failed" events so buffered and
// unbuffered paths classify identically; returns ok=false for non-terminal events.
func codexBootstrapTerminalFailure(eventData []byte) (statusErr, []byte, bool) {
	eventType := gjson.GetBytes(eventData, "type").String()
	var body []byte
	switch eventType {
	case "error":
		body = codexTerminalErrorBody(eventData, "error")
		if len(body) == 0 {
			body = codexTerminalTopLevelErrorBody(eventData)
		}
	case "response.failed":
		body = codexTerminalErrorBody(eventData, "response.error")
		if len(body) == 0 {
			body = codexTerminalErrorBody(eventData, "error")
		}
	default:
		return statusErr{}, nil, false
	}
	if len(body) == 0 {
		return statusErr{}, nil, false
	}
	return newCodexStatusErr(http.StatusInternalServerError, body), body, true
}
