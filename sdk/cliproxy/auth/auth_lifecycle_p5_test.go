package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// testJWTWithExp builds an unsigned JWT carrying the given exp claim.
func testJWTWithExp(t *testing.T, exp any) string {
	t.Helper()
	encode := func(v string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(v))
	}
	payload := `{"sub":"test"}`
	if exp != nil {
		switch v := exp.(type) {
		case int64:
			payload = `{"sub":"test","exp":` + itoa(v) + `}`
		case string:
			payload = `{"sub":"test","exp":"` + v + `"}`
		}
	}
	return encode(`{"alg":"none"}`) + "." + encode(payload) + ".sig"
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestP5MinQuotaCooldownFloorOn429(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	auth, err := manager.Register(WithSkipPersist(context.Background()), &Auth{ID: "p5-floor", Provider: "codex"})
	if err != nil || auth == nil {
		t.Fatalf("register: %v", err)
	}
	subSecond := 500 * time.Millisecond
	before := time.Now()
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth.ID,
		Provider:   "codex",
		Model:      "gpt-5",
		RetryAfter: &subSecond,
		Error:      &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota exceeded"},
	})
	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil || updated.ModelStates["gpt-5"] == nil {
		t.Fatal("missing model state")
	}
	state := updated.ModelStates["gpt-5"]
	wait := state.NextRetryAfter.Sub(before)
	if wait < minQuotaCooldownFloor {
		t.Fatalf("model cooldown %v below floor %v", wait, minQuotaCooldownFloor)
	}
	if got := state.NextRetryAfter.Sub(time.Now()); got <= 0 {
		t.Fatal("expected future retry deadline")
	}
	if updated.ModelStates["gpt-5"].Quota.NextRecoverAt.IsZero() {
		t.Fatal("expected quota recover deadline")
	}

	// Credential-level path via empty model.
	auth2, err := manager.Register(WithSkipPersist(context.Background()), &Auth{ID: "p5-floor-cred", Provider: "codex"})
	if err != nil || auth2 == nil {
		t.Fatalf("register2: %v", err)
	}
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth2.ID,
		Provider:   "codex",
		RetryAfter: &subSecond,
		Error:      &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota exhausted"},
	})
	cred, ok := manager.GetByID(auth2.ID)
	if !ok || cred == nil {
		t.Fatal("missing cred")
	}
	if cred.NextRetryAfter.Sub(before) < minQuotaCooldownFloor {
		t.Fatalf("credential cooldown %v below floor", cred.NextRetryAfter.Sub(before))
	}
	// Wall-clock deadlines must not carry a monotonic reading.
	if !cred.NextRetryAfter.Equal(cred.NextRetryAfter.Round(0)) {
		t.Fatal("cooldown deadline carries a monotonic reading")
	}
}

func TestP5AttemptedCredentialTrackingOn429(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetRetryConfig(3, time.Minute, 5)
	for _, id := range []string{"p5-a1", "p5-a2"} {
		if _, err := manager.Register(WithSkipPersist(context.Background()), &Auth{ID: id, Provider: "codex"}); err != nil {
			t.Fatalf("register %s: %v", id, err)
		}
	}
	attempted := map[string]struct{}{"p5-a1": {}}
	wait, found := manager.closestCooldownWaitWithAttempted([]string{"codex"}, "gpt-5", 0, http.StatusTooManyRequests, attempted)
	if !found {
		t.Fatal("expected retry wait for attempted 429 round")
	}
	if wait < minQuotaCooldownFloor {
		t.Fatalf("attempted wait %v below floor", wait)
	}
	// Non-429 status keeps legacy behavior: no attempted creds cooling means no wait.
	if _, found := manager.closestCooldownWaitWithAttempted([]string{"codex"}, "gpt-5", 0, http.StatusInternalServerError, attempted); found {
		t.Fatal("non-429 round should not force a floor wait")
	}
	// Tracker records dispatched credentials and preserves prior callbacks.
	seen := make(map[string]struct{})
	var prevCalled []string
	opts := cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.SelectedAuthCallbackMetadataKey: func(id string) { prevCalled = append(prevCalled, id) },
	}}
	tracked := withAttemptedAuthTracker(opts, seen)
	cb, _ := tracked.Metadata[cliproxyexecutor.SelectedAuthCallbackMetadataKey].(func(string))
	if cb == nil {
		t.Fatal("tracker callback missing")
	}
	cb("p5-a1")
	if _, ok := seen["p5-a1"]; !ok {
		t.Fatal("attempted id not recorded")
	}
	if len(prevCalled) != 1 || prevCalled[0] != "p5-a1" {
		t.Fatalf("previous callback not preserved: %v", prevCalled)
	}
}

func TestP5TokenExpirationValidation(t *testing.T) {
	future := time.Now().Add(2 * time.Hour).Unix()
	past := time.Now().Add(-2 * time.Hour).Unix()
	valid := &Auth{ID: "p5-valid", Metadata: map[string]any{"access_token": testJWTWithExp(t, future)}}
	expired := &Auth{ID: "p5-expired", Metadata: map[string]any{"access_token": testJWTWithExp(t, past)}}
	now := time.Now()
	if !valid.HasValidAccessToken(now) {
		t.Fatal("valid JWT token reported invalid")
	}
	if expired.HasValidAccessToken(now) {
		t.Fatal("expired JWT token reported valid")
	}
	if exp, ok := valid.AccessTokenExpirationTime(); !ok || exp.Unix() != future {
		t.Fatalf("AccessTokenExpirationTime = %v ok=%v want unix %d", exp, ok, future)
	}
	// JWT exp takes priority over metadata expiry keys.
	legacy := &Auth{ID: "p5-legacy", Metadata: map[string]any{
		"access_token": testJWTWithExp(t, future),
		"expires_at":   time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
	}}
	if exp, ok := legacy.ExpirationTime(); !ok || exp.Unix() != future {
		t.Fatalf("JWT exp should take priority, got %v ok=%v", exp, ok)
	}
	// Expired tokens block selection.
	blocked, _, _ := isAuthBlockedForModel(expired, "gpt-5", now)
	if !blocked {
		t.Fatal("expired token should block selection")
	}
	blocked, _, _ = isAuthBlockedForModel(valid, "gpt-5", now)
	if blocked {
		t.Fatal("valid token should not block selection")
	}
}

func TestP5ActiveCooldownDeadlinePreservation(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	auth, err := manager.Register(WithSkipPersist(context.Background()), &Auth{ID: "p5-preserve", Provider: "codex"})
	if err != nil || auth == nil {
		t.Fatalf("register: %v", err)
	}
	longRetry := 10 * time.Minute
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth.ID,
		Provider:   "codex",
		Model:      "gpt-5",
		RetryAfter: &longRetry,
		Error:      &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	})
	first, _ := manager.GetByID(auth.ID)
	firstDeadline := first.ModelStates["gpt-5"].NextRetryAfter
	// A subsequent failure with a shorter hint must not shorten the live deadline.
	shortRetry := time.Minute
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth.ID,
		Provider:   "codex",
		Model:      "gpt-5",
		RetryAfter: &shortRetry,
		Error:      &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	})
	second, _ := manager.GetByID(auth.ID)
	if !second.ModelStates["gpt-5"].NextRetryAfter.Equal(firstDeadline) {
		t.Fatalf("cooldown shortened: %v -> %v", firstDeadline, second.ModelStates["gpt-5"].NextRetryAfter)
	}
	// Credential-level preservation.
	auth2, _ := manager.Register(WithSkipPersist(context.Background()), &Auth{ID: "p5-preserve-cred", Provider: "codex"})
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth2.ID,
		Provider:   "codex",
		RetryAfter: &longRetry,
		Error:      &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	})
	c1, _ := manager.GetByID(auth2.ID)
	c1Deadline := c1.NextRetryAfter
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth2.ID,
		Provider:   "codex",
		RetryAfter: &shortRetry,
		Error:      &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	})
	c2, _ := manager.GetByID(auth2.ID)
	if !c2.NextRetryAfter.Equal(c1Deadline) {
		t.Fatalf("credential cooldown shortened: %v -> %v", c1Deadline, c2.NextRetryAfter)
	}
}

func TestP5RefreshLoopNextWaitCapped(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	loop := newAuthAutoRefreshLoop(manager, time.Second, 1)
	loop.upsert("far-future", time.Now().Add(10*time.Minute))
	wait, ok := loop.nextWait(time.Now())
	if !ok {
		t.Fatal("expected wait")
	}
	if wait > maxRefreshTimerWait {
		t.Fatalf("wait %v exceeds cap %v", wait, maxRefreshTimerWait)
	}
}

func TestP5SchedulerDemotesExpiredTokens(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	past := time.Now().Add(-time.Hour).Unix()
	auth, err := manager.Register(WithSkipPersist(context.Background()), &Auth{
		ID:       "p5-demote",
		Provider: "codex",
		Metadata: map[string]any{"access_token": testJWTWithExp(t, past)},
	})
	if err != nil || auth == nil {
		t.Fatalf("register: %v", err)
	}
	shard := &modelScheduler{modelKey: "gpt-5", entries: map[string]*scheduledAuth{
		auth.ID: {auth: auth.Clone(), state: scheduledStateReady},
	}}
	shard.demoteExpiredTokensLocked(time.Now())
	if shard.entries[auth.ID].state == scheduledStateReady {
		t.Fatal("expired token left in ready state")
	}
}

func TestP5HasUnauthorizedRequiresTerminalState(t *testing.T) {
	active := &Auth{LastError: &Error{HTTPStatus: http.StatusUnauthorized, Message: "unauthorized"}}
	if hasUnauthorizedAuthFailure(active) {
		t.Fatal("non-terminal 401 should not count as terminal unauthorized failure")
	}
	terminal := &Auth{
		Unavailable: true,
		Status:      StatusError,
		LastError:   &Error{HTTPStatus: http.StatusUnauthorized, Message: "unauthorized"},
	}
	if !hasUnauthorizedAuthFailure(terminal) {
		t.Fatal("terminal 401 should count")
	}
}
