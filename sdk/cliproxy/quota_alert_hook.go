package cliproxy

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/therealtinhtute/llmhub/internal/quotaalert"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// quotaAlertWakeMinInterval bounds how often request-path quota evidence can
// trigger an accelerated collection for the same auth.
const quotaAlertWakeMinInterval = time.Minute

// quotaAlertWaker is the wake surface consumed from the quota alert service.
type quotaAlertWaker interface {
	Wake()
}

// quotaAlertResultHook forwards qualified request-path quota evidence to the
// quota alert service so collection does not wait for the poll timer.
type quotaAlertResultHook struct {
	coreauth.NoopHook
	mu       sync.Mutex
	target   quotaAlertWaker
	lastWake map[string]time.Time
	now      func() time.Time
}

func newQuotaAlertResultHook() *quotaAlertResultHook {
	return &quotaAlertResultHook{lastWake: make(map[string]time.Time), now: time.Now}
}

// SetTarget binds the quota alert service after it is constructed; OnResult is
// a no-op until then so early request results are safe.
func (h *quotaAlertResultHook) SetTarget(service *quotaalert.Service) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if service == nil {
		h.target = nil
		return
	}
	h.target = service
}

// OnResult implements coreauth.Hook. It must never call back into the manager:
// MarkResult invokes hooks while holding scheduler state.
func (h *quotaAlertResultHook) OnResult(_ context.Context, result coreauth.Result) {
	if h == nil || result.Success || result.AuthID == "" {
		return
	}
	if _, ok := quotaAlertProvider(result.Provider); !ok {
		return
	}
	if !quotaAlertQualifiedEvidence(result) {
		return
	}
	now := h.now()
	h.mu.Lock()
	service := h.target
	if service == nil {
		h.mu.Unlock()
		return
	}
	if last, ok := h.lastWake[result.AuthID]; ok && now.Sub(last) < quotaAlertWakeMinInterval {
		h.mu.Unlock()
		return
	}
	if len(h.lastWake) > 256 {
		for authID, last := range h.lastWake {
			if now.Sub(last) >= quotaAlertWakeMinInterval {
				delete(h.lastWake, authID)
			}
		}
	}
	h.lastWake[result.AuthID] = now
	h.mu.Unlock()
	service.Wake()
}

// quotaAlertQualifiedEvidence reports whether a request failure carries
// provider-specific quota evidence. A bare failure without a quota signal
// never wakes collection.
func quotaAlertQualifiedEvidence(result coreauth.Result) bool {
	if result.CredentialScope {
		return true
	}
	if result.Error == nil {
		return false
	}
	if result.Error.HTTPStatus == http.StatusTooManyRequests {
		return true
	}
	code := strings.ToLower(result.Error.Code)
	return strings.Contains(code, "quota") || strings.Contains(code, "rate_limit")
}
