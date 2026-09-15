package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// testJWT builds an unsigned JWT whose payload carries the supplied exp claim.
func testJWT(t *testing.T, exp int64) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, errJSON := json.Marshal(map[string]any{"exp": exp})
	if errJSON != nil {
		t.Fatalf("marshal jwt payload: %v", errJSON)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

// The following tests cover the access-token expiration semantics ported from
// upstream CLIProxyAPI commit 9812b1e76872 ("fix(auth): validate access token
// expiration and retain valid credentials on refresh failure").
// Local symbols under test: Auth.AccessTokenExpirationTime, Auth.HasValidAccessToken,
// Auth.ExpirationTime, parseJWTExp, Manager.refreshAuthForRequest.

func TestAuth_AccessTokenExpirationTime_JWTExpPriority(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Unix()
	auth := &Auth{Metadata: map[string]any{
		"access_token": testJWT(t, exp),
		// A misleading metadata expiry must not win over the JWT exp claim.
		"expires_at": time.Now().Add(-time.Hour).Format(time.RFC3339),
	}}
	got, ok := auth.AccessTokenExpirationTime()
	if !ok {
		t.Fatal("AccessTokenExpirationTime() ok = false, want true")
	}
	if got.Unix() != exp {
		t.Fatalf("AccessTokenExpirationTime() = %d, want exp %d", got.Unix(), exp)
	}
	if !auth.HasValidAccessToken(time.Now()) {
		t.Fatal("HasValidAccessToken(now) = false, want true for unexpired JWT")
	}
	if auth.HasValidAccessToken(time.Now().Add(3 * time.Hour)) {
		t.Fatal("HasValidAccessToken(+3h) = true, want false past exp")
	}
}

func TestAuth_AccessTokenExpirationTime_CamelCaseAndURLEncoding(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	header := base64.URLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload := base64.URLEncoding.EncodeToString([]byte(`{"exp":` + fmt.Sprintf("%d", exp) + `}`))
	token := header + "." + payload + ".sig"
	auth := &Auth{Metadata: map[string]any{"accessToken": token}}
	got, ok := auth.AccessTokenExpirationTime()
	if !ok || got.Unix() != exp {
		t.Fatalf("AccessTokenExpirationTime() = %v, %v; want exp %d", got, ok, exp)
	}
}

func TestAuth_AccessTokenExpirationTime_FallsBackToMetadata(t *testing.T) {
	expires := time.Now().Add(45 * time.Minute).Truncate(time.Second)
	auth := &Auth{Metadata: map[string]any{
		"access_token": "opaque-token-not-a-jwt",
		"expires_at":   expires.Format(time.RFC3339),
	}}
	got, ok := auth.AccessTokenExpirationTime()
	if !ok {
		t.Fatal("AccessTokenExpirationTime() ok = false, want metadata fallback true")
	}
	if !got.Equal(expires) {
		t.Fatalf("AccessTokenExpirationTime() = %s, want %s", got, expires)
	}
}

func TestAuth_HasValidAccessToken_NoToken(t *testing.T) {
	auth := &Auth{Metadata: map[string]any{"email": "x@example.com"}}
	if auth.HasValidAccessToken(time.Now()) {
		t.Fatal("HasValidAccessToken() = true, want false for missing access token")
	}
	var nilAuth *Auth
	if nilAuth.HasValidAccessToken(time.Now()) {
		t.Fatal("HasValidAccessToken() on nil auth = true, want false")
	}
}

type failingRefreshExecutor struct {
	schedulerProviderTestExecutor
	err error
}

func (e failingRefreshExecutor) Refresh(context.Context, *Auth) (*Auth, error) {
	return nil, e.err
}

// Upstream 9812b1e76872: when refresh fails but the access token is still
// unexpired, the credential stays active and a retry backoff is scheduled
// (capped at token expiry) instead of marking the credential unavailable.
func TestManager_RefreshAuth_RetainsValidAccessTokenOnFailure(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(failingRefreshExecutor{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
		err:                           errors.New("dial tcp: connection reset by peer"),
	})

	exp := time.Now().Add(2 * time.Hour)
	auth := &Auth{
		ID:       "valid-token-refresh-fail",
		Provider: "codex",
		Metadata: map[string]any{
			"access_token":  testJWT(t, exp.Unix()),
			"refresh_token": "rt-1",
		},
	}
	if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	manager.refreshAuth(ctx, auth.ID)

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatal("expected auth after failed refresh")
	}
	if updated.Unavailable {
		t.Fatal("expected credential to stay available while access token unexpired")
	}
	if updated.Status == StatusError {
		t.Fatalf("Status = %v, want not StatusError while token remains valid", updated.Status)
	}
	if updated.NextRefreshAfter.IsZero() {
		t.Fatal("expected NextRefreshAfter retry backoff to be scheduled")
	}
	if updated.NextRefreshAfter.After(exp) {
		t.Fatalf("NextRefreshAfter = %s, want capped at token expiry %s", updated.NextRefreshAfter, exp)
	}
	if updated.LastError == nil {
		t.Fatal("expected refresh error to be recorded")
	}
	if hasUnauthorizedAuthFailure(updated) {
		t.Fatal("hasUnauthorizedAuthFailure() = true, want false for retained valid token")
	}
}

// Upstream 9812b1e76872: refresh failure with an expired access token demotes
// the credential (unavailable + error), distinguishing unauthorized from
// token-expired.
func TestManager_RefreshAuth_ExpiredAccessTokenMarkedUnavailable(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(failingRefreshExecutor{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
		err:                           errors.New("upstream temporarily unavailable"),
	})

	expired := time.Now().Add(-time.Hour)
	auth := &Auth{
		ID:       "expired-token-refresh-fail",
		Provider: "codex",
		Metadata: map[string]any{
			"access_token":  testJWT(t, expired.Unix()),
			"refresh_token": "rt-1",
		},
	}
	if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	manager.refreshAuth(ctx, auth.ID)

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatal("expected auth after failed refresh")
	}
	if !updated.Unavailable || updated.Status != StatusError {
		t.Fatalf("expected unavailable/error credential, got unavailable=%v status=%v", updated.Unavailable, updated.Status)
	}
	if updated.StatusMessage != "token expired" {
		t.Fatalf("StatusMessage = %q, want %q", updated.StatusMessage, "token expired")
	}
	if updated.NextRefreshAfter.IsZero() {
		t.Fatal("expected retry backoff for expired-token refresh failure")
	}
}

// Upstream 9812b1e76872: expired access tokens are blocked from selection and
// demoted out of ready scheduler entries.
func TestManager_ExpiredAccessToken_BlockedFromScheduling(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(schedulerProviderTestExecutor{provider: "codex"})

	authID := "expired-token-pick"
	auth := &Auth{
		ID:       authID,
		Provider: "codex",
		Status:   StatusActive,
		Metadata: map[string]any{
			"access_token": testJWT(t, time.Now().Add(-time.Hour).Unix()),
		},
	}
	if _, errRegister := manager.Register(ctx, auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	picked, errPick := manager.scheduler.pickSingle(ctx, "codex", "", cliproxyexecutor.Options{}, nil)
	if errPick == nil {
		t.Fatalf("expected pick error for expired access token, got auth %v", picked)
	}

	blocked, _, _ := isAuthBlockedForModel(auth, "", time.Now())
	if !blocked {
		t.Fatal("isAuthBlockedForModel() = false, want true for expired access token")
	}
}

// The following tests cover the three-way merge semantics ported from upstream
// CLIProxyAPI commit 4c1bebe837a6 ("fix(auth): preserve concurrent modifications
// during auth refresh and preparation").
// Local symbols under test: MergeRefreshedAuth, MergePreparedAuth,
// Manager.UpdateRefreshedAuth, Manager.UpdatePreparedAuth, Manager.persist.

func TestMergeRefreshedAuth_PreservesConcurrentCooldown(t *testing.T) {
	base := &Auth{
		ID:                "merge-cd",
		RegistrationEpoch: 3,
		Metadata:          map[string]any{"access_token": "old"},
		Unavailable:       false,
		Status:            StatusActive,
		LastRefreshedAt:   time.Now().Add(-time.Hour),
	}
	current := base.Clone()
	current.Unavailable = true
	current.Status = StatusError
	current.StatusMessage = "quota exceeded"
	current.Quota = QuotaState{Exceeded: true, Reason: "credential_quota", NextRecoverAt: time.Now().Add(10 * time.Minute)}
	current.NextRetryAfter = time.Now().Add(10 * time.Minute)

	updated := base.Clone()
	updated.Metadata["access_token"] = "new-token"
	updated.LastRefreshedAt = time.Now()
	updated.Unavailable = false
	updated.Status = StatusActive

	merged := MergeRefreshedAuth(base, current, updated)
	if merged == nil {
		t.Fatal("MergeRefreshedAuth() = nil")
	}
	if got := merged.Metadata["access_token"]; got != "new-token" {
		t.Fatalf("merged access_token = %v, want new-token", got)
	}
	if !merged.Unavailable || merged.Status != StatusError {
		t.Fatalf("expected concurrent cooldown preserved, got unavailable=%v status=%v", merged.Unavailable, merged.Status)
	}
	if merged.StatusMessage != "quota exceeded" {
		t.Fatalf("StatusMessage = %q, want concurrent quota message", merged.StatusMessage)
	}
	if merged.LastRefreshedAt.IsZero() {
		t.Fatal("expected LastRefreshedAt from refresh result")
	}
}

func TestMergeRefreshedAuth_PreservesConcurrentProxyURLAndNotes(t *testing.T) {
	base := &Auth{
		ID:                "merge-proxy",
		RegistrationEpoch: 1,
		ProxyURL:          "http://old-proxy",
		Metadata:          map[string]any{"access_token": "old", "note": "a"},
	}
	current := base.Clone()
	current.ProxyURL = "http://user-proxy"
	current.Metadata["note"] = "user-edited"

	updated := base.Clone()
	updated.ProxyURL = "http://executor-proxy"
	updated.Metadata["access_token"] = "new-token"
	updated.Metadata["note"] = "executor-wrote"
	updated.Status = StatusActive
	updated.LastRefreshedAt = time.Now()

	merged := MergeRefreshedAuth(base, current, updated)
	if merged == nil {
		t.Fatal("MergeRefreshedAuth() = nil")
	}
	if merged.ProxyURL != "http://user-proxy" {
		t.Fatalf("ProxyURL = %q, want user-modified http://user-proxy", merged.ProxyURL)
	}
	if got := merged.Metadata["proxy_url"]; got != "http://user-proxy" {
		t.Fatalf("metadata proxy_url = %v, want user value", got)
	}
	if got := merged.Metadata["note"]; got != "user-edited" {
		t.Fatalf("metadata note = %v, want concurrent user edit", got)
	}
	if got := merged.Metadata["access_token"]; got != "new-token" {
		t.Fatalf("access_token = %v, want executor's refreshed token (token payload wins)", got)
	}
}

func TestMergeRefreshedAuth_StaleEpochKeepsCurrent(t *testing.T) {
	base := &Auth{ID: "merge-epoch", RegistrationEpoch: 1, Metadata: map[string]any{"access_token": "old"}}
	current := base.Clone()
	current.RegistrationEpoch = 2
	current.Metadata["access_token"] = "current-token"

	updated := base.Clone()
	updated.Metadata["access_token"] = "stale-executor-token"

	merged := MergeRefreshedAuth(base, current, updated)
	if merged == nil {
		t.Fatal("MergeRefreshedAuth() = nil")
	}
	if got := merged.Metadata["access_token"]; got != "current-token" {
		t.Fatalf("access_token = %v, want current state for stale epoch merge", got)
	}
}

func TestMergePreparedAuth_PreservesRefreshLifecycle(t *testing.T) {
	lastRefresh := time.Now().Add(-30 * time.Minute)
	base := &Auth{
		ID:                "merge-prepare",
		RegistrationEpoch: 2,
		LastRefreshedAt:   lastRefresh,
		Metadata:          map[string]any{"access_token": "tok"},
	}
	current := base.Clone()
	current.Unavailable = true
	current.NextRetryAfter = time.Now().Add(time.Minute)

	updated := base.Clone()
	updated.Metadata["prepared_field"] = "prepared"
	updated.Unavailable = false
	updated.NextRetryAfter = time.Time{}
	updated.LastRefreshedAt = time.Now()

	merged := MergePreparedAuth(base, current, updated)
	if merged == nil {
		t.Fatal("MergePreparedAuth() = nil")
	}
	if got := merged.Metadata["prepared_field"]; got != "prepared" {
		t.Fatalf("metadata prepared_field = %v, want prepared", got)
	}
	if !merged.Unavailable {
		t.Fatal("expected concurrent unavailability preserved by prepare merge")
	}
	if merged.NextRetryAfter.IsZero() {
		t.Fatal("expected concurrent NextRetryAfter preserved by prepare merge")
	}
	if !merged.LastRefreshedAt.Equal(lastRefresh) {
		t.Fatalf("LastRefreshedAt = %s, want unchanged %s", merged.LastRefreshedAt, lastRefresh)
	}
}

func TestManager_UpdateRefreshedAuth_RejectsStaleRegistrationEpoch(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)

	if _, errRegister := manager.Register(ctx, &Auth{ID: "epoch-guard", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}); errRegister != nil {
		t.Fatalf("register: %v", errRegister)
	}
	registered, _ := manager.GetByID("epoch-guard")
	base := registered.Clone()

	// Simulate re-registration bumping the epoch while a refresh is in flight.
	if _, errRegister := manager.Register(ctx, &Auth{ID: "epoch-guard", Provider: "codex", Metadata: map[string]any{"access_token": "tok2"}}); errRegister != nil {
		t.Fatalf("re-register: %v", errRegister)
	}

	updated := base.Clone()
	updated.Metadata["access_token"] = "stale"
	if _, errUpdate := manager.UpdateRefreshedAuth(ctx, base, updated); errUpdate == nil {
		t.Fatal("UpdateRefreshedAuth() error = nil, want stale registration epoch error")
	} else if !strings.Contains(errUpdate.Error(), "stale registration epoch") {
		t.Fatalf("UpdateRefreshedAuth() error = %v, want stale registration epoch", errUpdate)
	}

	current, _ := manager.GetByID("epoch-guard")
	if got := current.Metadata["access_token"]; got != "tok2" {
		t.Fatalf("access_token = %v, want re-registered token preserved", got)
	}
}

// Upstream 4c1bebe837a6: per-auth persistence drops writes whose
// (RegistrationEpoch, Generation) pair is older than the last persisted pair.
func TestManager_Persist_DropsOutOfOrderGenerations(t *testing.T) {
	store := &countingStore{}
	manager := NewManager(store, nil, nil)
	ctx := context.Background()

	if _, errRegister := manager.Register(ctx, &Auth{ID: "persist-order", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}); errRegister != nil {
		t.Fatalf("register: %v", errRegister)
	}
	current, _ := manager.GetByID("persist-order")
	if got := store.saveCount.Load(); got != 1 {
		t.Fatalf("save count after register = %d, want 1", got)
	}

	// A newer snapshot persists.
	newer := current.Clone()
	newer.Generation = current.Generation + 1
	if err := manager.persist(ctx, newer); err != nil {
		t.Fatalf("persist newer: %v", err)
	}
	if got := store.saveCount.Load(); got != 2 {
		t.Fatalf("save count after newer persist = %d, want 2", got)
	}

	// An older generation of the same epoch must be dropped.
	stale := current.Clone()
	stale.Metadata["marker"] = "stale-write"
	if err := manager.persist(ctx, stale); err != nil {
		t.Fatalf("persist stale: %v", err)
	}
	if got := store.saveCount.Load(); got != 2 {
		t.Fatalf("save count after stale persist = %d, want 2 (stale write dropped)", got)
	}

	// An older epoch must be dropped even with a higher generation.
	oldEpoch := newer.Clone()
	oldEpoch.RegistrationEpoch = newer.RegistrationEpoch - 1
	oldEpoch.Generation = newer.Generation + 99
	if err := manager.persist(ctx, oldEpoch); err != nil {
		t.Fatalf("persist old epoch: %v", err)
	}
	if got := store.saveCount.Load(); got != 2 {
		t.Fatalf("save count after old-epoch persist = %d, want 2", got)
	}
}

type recordingPreparer struct {
	schedulerProviderTestExecutor
	preparedMetadata map[string]any
}

func (e recordingPreparer) ShouldPrepareRequestAuth(*Auth) bool { return true }

func (e recordingPreparer) PrepareRequestAuth(_ context.Context, auth *Auth) (*Auth, error) {
	if auth.Metadata == nil {
		auth.Metadata = make(map[string]any)
	}
	for k, v := range e.preparedMetadata {
		auth.Metadata[k] = v
	}
	return auth, nil
}

// Upstream 4c1bebe837a6: request preparation merges into the latest runtime
// auth instead of overwriting it.
func TestManager_PrepareRequestAuth_MergesIntoCurrent(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(nil, &RoundRobinSelector{}, nil)
	manager.RegisterExecutor(recordingPreparer{
		schedulerProviderTestExecutor: schedulerProviderTestExecutor{provider: "codex"},
		preparedMetadata:              map[string]any{"prepared": "yes"},
	})

	if _, errRegister := manager.Register(ctx, &Auth{ID: "prepare-merge", Provider: "codex", Metadata: map[string]any{"access_token": "tok"}}); errRegister != nil {
		t.Fatalf("register: %v", errRegister)
	}
	// Concurrent user modification after the base snapshot would be clobbered by
	// a wholesale replace; the merge must keep it.
	current, _ := manager.GetByID("prepare-merge")
	currentWithUserEdit := current.Clone()
	currentWithUserEdit.Metadata["user_note"] = "keep-me"
	if _, errUpdate := manager.Update(ctx, currentWithUserEdit); errUpdate != nil {
		t.Fatalf("update: %v", errUpdate)
	}

	got, err := manager.prepareRequestAuth(ctx, manager.executors["codex"], current)
	if err != nil {
		t.Fatalf("prepareRequestAuth: %v", err)
	}
	if got == nil {
		t.Fatal("prepareRequestAuth returned nil auth")
	}
	stored, _ := manager.GetByID("prepare-merge")
	if stored.Metadata["prepared"] != "yes" {
		t.Fatalf("prepared metadata missing: %v", stored.Metadata["prepared"])
	}
	if stored.Metadata["user_note"] != "keep-me" {
		t.Fatalf("concurrent user metadata lost: %v", stored.Metadata["user_note"])
	}
}
