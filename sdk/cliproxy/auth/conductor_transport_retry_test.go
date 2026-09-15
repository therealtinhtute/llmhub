package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"testing"
	"time"
)

// The following tests cover the transient pre-HTTP transport retry semantics
// ported from upstream CLIProxyAPI commit bef1f65c6c1d ("feat(auth): add retry
// logic for pre-HTTP transport failures"). Local symbols under test:
// isTransientTransportError, isTransientTransportResultError,
// isTransientSyscallError, isTransientTransportMessage, isRequestRetryRoundError,
// resultErrorFromError, shouldSkipCredentialCooldown,
// Manager.shouldRetryAfterError, ErrorCodeTransientTransport.

func TestIsTransientTransportError_Classification(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"http status stays on credential path", &Error{HTTPStatus: http.StatusBadGateway, Message: "bad gateway"}, false},
		{"context canceled excluded", context.Canceled, false},
		{"deadline exceeded excluded", context.DeadlineExceeded, false},
		{"io.EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"dns timeout", &net.DNSError{Err: "timeout", Name: "example.com", IsTimeout: true}, true},
		{"dns temporary", &net.DNSError{Err: "temporary", Name: "example.com", IsTemporary: true}, true},
		{"net timeout", &net.DNSError{Err: "i/o timeout", IsTimeout: true}, true},
		{"op error", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, true},
		{"syscall conn reset", syscall.ECONNRESET, true},
		{"syscall conn refused", syscall.ECONNREFUSED, true},
		{"syscall timed out", syscall.ETIMEDOUT, true},
		{"syscall host unreachable", syscall.EHOSTUNREACH, true},
		{"syscall broken pipe", syscall.EPIPE, true},
		{"wrapped url error", &url.Error{Op: "Get", URL: "https://example.com", Err: syscall.ECONNRESET}, true},
		{"tls record header message", &tls.RecordHeaderError{Msg: "tls handshake timeout"}, true},
		{"message connection reset", errors.New("read tcp 1.2.3.4: connection reset by peer"), true},
		{"message no such host", errors.New("dial tcp: lookup api.example.com: no such host"), true},
		{"message use of closed connection", errors.New("use of closed network connection"), true},
		{"generic error not classified", errors.New("some random failure"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransientTransportError(tc.err); got != tc.want {
				t.Fatalf("isTransientTransportError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestResultErrorFromError_TransientTransportCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"connection refused op error", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, ErrorCodeTransientTransport},
		{"tls message", errors.New("tls handshake timeout"), ErrorCodeTransientTransport},
		{"eof is lifecycle first", io.EOF, ErrorCodeConnectionLifecycle},
		{"http status untouched", &Error{HTTPStatus: http.StatusTooManyRequests, Message: "rate limit"}, ""},
		{"generic error untouched", errors.New("unrelated failure"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resultErrorFromError(tc.err)
			if got == nil {
				t.Fatal("resultErrorFromError() = nil")
			}
			if got.Code != tc.want {
				t.Fatalf("resultErrorFromError().Code = %q, want %q", got.Code, tc.want)
			}
		})
	}
}

func TestShouldSkipCredentialCooldown_TransientTransport(t *testing.T) {
	cases := []struct {
		name string
		err  *Error
		want bool
	}{
		{"coded transport error", &Error{Code: ErrorCodeTransientTransport, Message: "dial tcp: connection refused"}, true},
		{"message-only transport error", &Error{Message: "read tcp: connection reset by peer"}, true},
		{"statused error not skipped by transport", &Error{HTTPStatus: http.StatusTooManyRequests, Message: "rate limit"}, false},
		{"force cooldown wins", &Error{Code: ErrorCodeForceCooldown, Message: "connection reset"}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSkipCredentialCooldown(tc.err); got != tc.want {
				t.Fatalf("shouldSkipCredentialCooldown(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// Transport failures must participate in request-retry rounds without waiting
// on credential cooldown, matching upstream isRequestRetryRoundError usage.
func TestShouldRetryAfterError_TransientTransportRetriesWithoutCooldown(t *testing.T) {
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.SetRetryConfig(3, time.Minute, 0)

	auth := &Auth{ID: "transport-retry", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register: %v", err)
	}

	transportErr := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	wait, shouldRetry := manager.shouldRetryAfterError(transportErr, 0, []string{"codex"}, "model", time.Minute)
	if !shouldRetry {
		t.Fatal("shouldRetryAfterError() = false, want true for transient transport error")
	}
	if wait != 0 {
		t.Fatalf("wait = %v, want immediate retry (0)", wait)
	}

	// The credential must not be cooled by the retry decision.
	updated, _ := manager.GetByID("transport-retry")
	if updated.Unavailable {
		t.Fatal("credential cooled by transient transport retry decision")
	}
}

func TestShouldRetryAfterError_TransientTransportHonorsRetryLimit(t *testing.T) {
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.SetRetryConfig(1, time.Minute, 0)
	if _, err := manager.Register(context.Background(), &Auth{ID: "transport-cap", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}); err != nil {
		t.Fatalf("register: %v", err)
	}

	transportErr := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNRESET}
	if _, shouldRetry := manager.shouldRetryAfterError(transportErr, 5, []string{"codex"}, "model", time.Minute); shouldRetry {
		t.Fatal("shouldRetryAfterError() = true past retry limit, want false")
	}
}

func TestIsRequestRetryRoundError_IncludesTransport(t *testing.T) {
	if !isRequestRetryRoundError(&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}) {
		t.Fatal("isRequestRetryRoundError() = false for transport error, want true")
	}
	if !isRequestRetryRoundError(&Error{HTTPStatus: http.StatusServiceUnavailable, Message: "unavailable"}) {
		t.Fatal("isRequestRetryRoundError() = false for 503, want true")
	}
	if isRequestRetryRoundError(&Error{HTTPStatus: http.StatusBadRequest, Message: "bad request"}) {
		t.Fatal("isRequestRetryRoundError() = true for 400, want false")
	}
	if isRequestRetryRoundError(nil) {
		t.Fatal("isRequestRetryRoundError(nil) = true, want false")
	}
}

// End-to-end: MarkResult must not cool the credential for transport failures.
func TestMarkResult_TransientTransportSkipsCooldown(t *testing.T) {
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	auth := &Auth{ID: "transport-mark", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register: %v", err)
	}

	manager.MarkResult(context.Background(), Result{
		AuthID:   "transport-mark",
		Provider: "codex",
		Model:    "model",
		Success:  false,
		Error:    &Error{Code: ErrorCodeTransientTransport, Message: fmt.Sprintf("dial tcp: %v", syscall.ECONNREFUSED)},
	})

	updated, _ := manager.GetByID("transport-mark")
	if updated.Unavailable {
		t.Fatal("MarkResult cooled credential for transient transport error")
	}
	if updated.Status == StatusError {
		t.Fatalf("Status = %v, want not StatusError for transient transport", updated.Status)
	}
}
