package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/gorilla/websocket"
)

// TestIsConnectionLifecycleError verifies that client-abort and request-scoped
// timeout failures are classified as connection lifecycle errors so they never
// cool credentials. Mirrors upstream 4b3cc55cdc93 deadline additions.
func TestIsConnectionLifecycleError(t *testing.T) {
	typedCases := []error{
		context.Canceled,
		context.DeadlineExceeded,
		io.EOF,
		io.ErrUnexpectedEOF,
		&url.Error{Op: "Post", URL: "https://example.com", Err: context.Canceled},
		&url.Error{Op: "Post", URL: "https://example.com", Err: context.DeadlineExceeded},
		fmt.Errorf("wrap: %w", context.Canceled),
		fmt.Errorf("wrap: %w", io.ErrUnexpectedEOF),
		&websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "normal"},
		&websocket.CloseError{Code: websocket.CloseGoingAway, Text: "bye"},
		&websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: "unexpected EOF"},
	}
	for _, err := range typedCases {
		if !isConnectionLifecycleError(err) {
			t.Fatalf("isConnectionLifecycleError(%v) = false, want true", err)
		}
	}

	messageCases := []string{
		"context canceled",
		"context deadline exceeded",
		"EOF",
		"unexpected EOF",
		"websocket: close 1000 (normal)",
		"websocket: close 1001 (going away)",
		"websocket: close 1006 (abnormal closure): unexpected EOF",
		"read tcp 127.0.0.1:1->127.0.0.1:2: unexpected EOF",
	}
	for _, msg := range messageCases {
		if !isConnectionLifecycleError(errors.New(msg)) {
			t.Fatalf("isConnectionLifecycleError(%q) = false, want true", msg)
		}
	}

	notLifecycle := []error{
		errors.New("boom"),
		errors.New("invalid token"),
		&Error{Message: "unauthorized", HTTPStatus: http.StatusUnauthorized},
		&Error{Message: "rate limited", HTTPStatus: http.StatusTooManyRequests},
		&Error{Message: "bad gateway", HTTPStatus: http.StatusBadGateway},
	}
	for _, err := range notLifecycle {
		if isConnectionLifecycleError(err) {
			t.Fatalf("isConnectionLifecycleError(%v) = true, want false", err)
		}
	}
}

// TestIsConnectionLifecycleMessage verifies the message-level fallback used when
// no typed cause is available.
func TestIsConnectionLifecycleMessage(t *testing.T) {
	for _, msg := range []string{
		"context canceled",
		"context deadline exceeded",
		"eof",
		"unexpected eof",
		"websocket: close 1006 (abnormal closure): unexpected EOF",
	} {
		if !isConnectionLifecycleMessage(msg) {
			t.Fatalf("isConnectionLifecycleMessage(%q) = false, want true", msg)
		}
	}
	for _, msg := range []string{
		"",
		"unauthorized",
		"rate limit reached",
		"internal error",
	} {
		if isConnectionLifecycleMessage(msg) {
			t.Fatalf("isConnectionLifecycleMessage(%q) = true, want false", msg)
		}
	}
}

// TestResultErrorFromError_ConnectionLifecycleDoesNotBecomeRequestScoped guards
// the ordering in resultErrorFromError: lifecycle failures keep their lifecycle
// classification instead of becoming request-scoped, so credential fallback and
// rotation remain intact.
func TestResultErrorFromError_ConnectionLifecycleDoesNotBecomeRequestScoped(t *testing.T) {
	cases := []error{
		context.Canceled,
		context.DeadlineExceeded,
		io.EOF,
		io.ErrUnexpectedEOF,
		&url.Error{Op: "Post", URL: "https://example.com", Err: context.Canceled},
		&url.Error{Op: "Post", URL: "https://example.com", Err: context.DeadlineExceeded},
		&websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "normal"},
		&websocket.CloseError{Code: websocket.CloseGoingAway, Text: "bye"},
		&websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: "unexpected EOF"},
		fmt.Errorf("wrap: %w", context.DeadlineExceeded),
		fmt.Errorf("wrap: %w", io.ErrUnexpectedEOF),
		errors.New("websocket: close 1006 (abnormal closure): unexpected EOF"),
		errors.New("context deadline exceeded"),
		errors.New("unexpected EOF"),
	}
	for _, err := range cases {
		resultErr := resultErrorFromError(err)
		if resultErr == nil {
			t.Fatalf("resultErrorFromError(%v) = nil", err)
		}
		if resultErr.Code == requestScopedErrorCode {
			t.Fatalf("resultErrorFromError(%v) = request-scoped code %q, want connection-lifecycle", err, resultErr.Code)
		}
		if resultErr.Code != connectionLifecycleErrorCode {
			t.Fatalf("resultErrorFromError(%v) code = %q, want %q", err, resultErr.Code, connectionLifecycleErrorCode)
		}
	}
}
