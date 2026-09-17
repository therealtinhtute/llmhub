package management

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// oauthSessionTTL must cover device-code flows (xAI ~30m, Kimi ~15m).
	// Ported from upstream CLIProxyAPI 6e819ab62257.
	oauthSessionTTL = 30 * time.Minute
	// oauthCompletedSessionTTL retains a completed-session tombstone so a
	// replayed callback is rejected with 409 instead of 404 (upstream
	// 7115e7e00c4d/d1ef06cb5e34 idempotent-completion slice).
	oauthCompletedSessionTTL = time.Minute
	maxOAuthStateLength      = 128
)

var (
	errInvalidOAuthState      = errors.New("invalid oauth state")
	errUnsupportedOAuthFlow   = errors.New("unsupported oauth provider")
	errOAuthSessionNotPending = errors.New("oauth session is not pending")
)

type oauthSession struct {
	Provider  string
	Status    string
	Callback  *oauthCallbackFilePayload
	Completed bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

type oauthSessionStore struct {
	mu           sync.RWMutex
	ttl          time.Duration
	completedTTL time.Duration
	sessions     map[string]oauthSession
}

func newOAuthSessionStore(ttl time.Duration) *oauthSessionStore {
	if ttl <= 0 {
		ttl = oauthSessionTTL
	}
	completedTTL := oauthCompletedSessionTTL
	if ttl < completedTTL {
		completedTTL = ttl
	}
	return &oauthSessionStore{
		ttl:          ttl,
		completedTTL: completedTTL,
		sessions:     make(map[string]oauthSession),
	}
}

func (s *oauthSessionStore) purgeExpiredLocked(now time.Time) {
	for state, session := range s.sessions {
		if !session.ExpiresAt.IsZero() && now.After(session.ExpiresAt) {
			delete(s.sessions, state)
		}
	}
}

func (s *oauthSessionStore) Register(state, provider string) {
	state = strings.TrimSpace(state)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if state == "" || provider == "" {
		return
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	s.sessions[state] = oauthSession{
		Provider:  provider,
		Status:    "",
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}
}

func (s *oauthSessionStore) SetError(state, message string) {
	state = strings.TrimSpace(state)
	message = strings.TrimSpace(message)
	if state == "" {
		return
	}
	if message == "" {
		message = "Authentication failed"
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	if !ok || session.Completed {
		return
	}
	session.Status = message
	session.ExpiresAt = now.Add(s.ttl)
	s.sessions[state] = session
}

func (s *oauthSessionStore) SetCallback(state, provider string, payload oauthCallbackFilePayload) error {
	state = strings.TrimSpace(state)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if state == "" {
		return errOAuthSessionNotPending
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	if !ok || session.Completed || session.Status != "" {
		return errOAuthSessionNotPending
	}
	if provider != "" && !strings.EqualFold(session.Provider, provider) {
		return fmt.Errorf("oauth provider mismatch")
	}
	payload.State = strings.TrimSpace(payload.State)
	if payload.State == "" {
		payload.State = state
	}
	session.Callback = &payload
	session.ExpiresAt = now.Add(s.ttl)
	s.sessions[state] = session
	return nil
}

func (s *oauthSessionStore) TakeCallback(state, provider string) (oauthCallbackFilePayload, bool, error) {
	state = strings.TrimSpace(state)
	provider = strings.ToLower(strings.TrimSpace(provider))
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	if !ok || session.Completed || session.Status != "" {
		return oauthCallbackFilePayload{}, false, errOAuthSessionNotPending
	}
	if provider != "" && !strings.EqualFold(session.Provider, provider) {
		return oauthCallbackFilePayload{}, false, fmt.Errorf("oauth provider mismatch")
	}
	if session.Callback == nil {
		return oauthCallbackFilePayload{}, false, nil
	}
	payload := *session.Callback
	session.Callback = nil
	s.sessions[state] = session
	return payload, true, nil
}

// Complete marks a session completed and retains a short-lived tombstone
// (completedTTL) instead of deleting it, so replayed callbacks are rejected
// with 409 and completion is idempotent. Ported from upstream CLIProxyAPI
// internal/api/handlers/management/oauth_sessions.go (7115e7e00c4d).
func (s *oauthSessionStore) Complete(state string) {
	state = strings.TrimSpace(state)
	if state == "" {
		return
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	if !ok || session.Completed {
		return
	}
	session.Status = ""
	session.Callback = nil
	session.Completed = true
	session.ExpiresAt = now.Add(s.completedTTL)
	s.sessions[state] = session
}

func (s *oauthSessionStore) CompleteProvider(provider string) int {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return 0
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	removed := 0
	for state, session := range s.sessions {
		if !session.Completed && strings.EqualFold(session.Provider, provider) {
			session.Status = ""
			session.Callback = nil
			session.Completed = true
			session.ExpiresAt = now.Add(s.completedTTL)
			s.sessions[state] = session
			removed++
		}
	}
	return removed
}

func (s *oauthSessionStore) Get(state string) (oauthSession, bool) {
	state = strings.TrimSpace(state)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	return session, ok
}

func (s *oauthSessionStore) IsPending(state, provider string) bool {
	state = strings.TrimSpace(state)
	provider = strings.ToLower(strings.TrimSpace(provider))
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	if !ok {
		return false
	}
	if session.Completed || session.Status != "" {
		return false
	}
	if provider == "" {
		return true
	}
	return strings.EqualFold(session.Provider, provider)
}

// Cancel removes a pending OAuth session so background callback and device-code
// waiters observe IsOAuthSessionPending as false and exit without saving
// credentials. Returns true when a pending session was cancelled. Completed
// tombstones and errored sessions are not cancellable. Ported from upstream
// CLIProxyAPI internal/api/handlers/management/oauth_sessions.go (6e819ab62257).
func (s *oauthSessionStore) Cancel(state string) bool {
	state = strings.TrimSpace(state)
	if state == "" {
		return false
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked(now)
	session, ok := s.sessions[state]
	if !ok || session.Completed || session.Status != "" {
		return false
	}
	delete(s.sessions, state)
	return true
}

var oauthSessions = newOAuthSessionStore(oauthSessionTTL)

func RegisterOAuthSession(state, provider string) { oauthSessions.Register(state, provider) }

func SetOAuthSessionError(state, message string) { oauthSessions.SetError(state, message) }

func CompleteOAuthSession(state string) { oauthSessions.Complete(state) }

func CompleteOAuthSessionsByProvider(provider string) int {
	return oauthSessions.CompleteProvider(provider)
}

// GetOAuthSession hides completed tombstones from legacy readers
// (upstream d1ef06cb5e34); use GetOAuthSessionDetails when the completed flag
// must be observed.
func GetOAuthSession(state string) (provider string, status string, ok bool) {
	session, ok := oauthSessions.Get(state)
	if !ok || session.Completed {
		return "", "", false
	}
	return session.Provider, session.Status, true
}

// GetOAuthSessionDetails additionally reports the completed tombstone flag so
// handlers can distinguish a completed session (409 replay) from an unknown or
// expired one (404). Local variant of the upstream details getter (v7.3.4):
// plugin source/metadata are omitted because llmhub has no plugin OAuth
// sessions.
func GetOAuthSessionDetails(state string) (provider string, status string, completed bool, ok bool) {
	session, ok := oauthSessions.Get(state)
	if !ok {
		return "", "", false, false
	}
	return session.Provider, session.Status, session.Completed, true
}

func IsOAuthSessionPending(state, provider string) bool {
	return oauthSessions.IsPending(state, provider)
}

// CancelOAuthSession cancels a pending OAuth session by state. Background
// callback and device-code waiters observe IsOAuthSessionPending as false and
// exit without saving credentials. Returns true when a pending session was
// cancelled.
func CancelOAuthSession(state string) bool {
	return oauthSessions.Cancel(state)
}

// guardOAuthSessionPendingForSave returns errOAuthSessionNotPending when the
// session is no longer pending (cancelled, completed, errored, or expired).
// Call immediately before persisting credentials so a cancel that races with
// token exchange or metadata fetch cannot save credentials for a cancelled
// flow.
func guardOAuthSessionPendingForSave(state, provider string) error {
	if IsOAuthSessionPending(state, provider) {
		return nil
	}
	return errOAuthSessionNotPending
}

func SubmitOAuthCallbackForPendingSession(provider, state, code, errorMessage string) error {
	canonicalProvider, err := NormalizeOAuthProvider(provider)
	if err != nil {
		return err
	}
	if err := ValidateOAuthState(state); err != nil {
		return err
	}
	payload := oauthCallbackFilePayload{
		Code:  strings.TrimSpace(code),
		State: strings.TrimSpace(state),
		Error: strings.TrimSpace(errorMessage),
	}
	return oauthSessions.SetCallback(state, canonicalProvider, payload)
}

func WaitOAuthCallbackForPendingSession(provider, state string, timeout time.Duration) (map[string]string, error) {
	canonicalProvider, err := NormalizeOAuthProvider(provider)
	if err != nil {
		return nil, err
	}
	if err := ValidateOAuthState(state); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	for {
		payload, ok, errTake := oauthSessions.TakeCallback(state, canonicalProvider)
		if errTake != nil {
			return nil, errTake
		}
		if ok {
			return map[string]string{
				"code":  strings.TrimSpace(payload.Code),
				"state": strings.TrimSpace(payload.State),
				"error": strings.TrimSpace(payload.Error),
			}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for OAuth callback")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func oauthSessionErrorWithCause(message string, cause error) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Authentication failed"
	}
	if cause == nil {
		return message
	}
	detail := strings.TrimSpace(cause.Error())
	if detail == "" {
		return message
	}
	return message + ": " + detail
}

func ValidateOAuthState(state string) error {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		return fmt.Errorf("%w: empty", errInvalidOAuthState)
	}
	if len(trimmed) > maxOAuthStateLength {
		return fmt.Errorf("%w: too long", errInvalidOAuthState)
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return fmt.Errorf("%w: contains path separator", errInvalidOAuthState)
	}
	if strings.Contains(trimmed, "..") {
		return fmt.Errorf("%w: contains '..'", errInvalidOAuthState)
	}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return fmt.Errorf("%w: invalid character", errInvalidOAuthState)
		}
	}
	return nil
}

func NormalizeOAuthProvider(provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic", "claude":
		return "anthropic", nil
	case "codex", "openai":
		return "codex", nil
	case "gemini", "google":
		return "gemini", nil
	case "antigravity", "anti-gravity":
		return "antigravity", nil
	case "xai", "x-ai", "x.ai", "grok":
		return "xai", nil
	case "devin", "cognition":
		return "devin", nil
	// Meta/Muse provider normalization ported from upstream CLIProxyAPI commit e475807a.
	case "meta", "muse":
		return "meta", nil
	default:
		return "", errUnsupportedOAuthFlow
	}
}

type oauthCallbackFilePayload struct {
	Code  string `json:"code"`
	State string `json:"state"`
	Error string `json:"error"`
}

func WriteOAuthCallbackFile(authDir, provider, state, code, errorMessage string) (string, error) {
	if strings.TrimSpace(authDir) == "" {
		return "", fmt.Errorf("auth dir is empty")
	}
	canonicalProvider, err := NormalizeOAuthProvider(provider)
	if err != nil {
		return "", err
	}
	if err := ValidateOAuthState(state); err != nil {
		return "", err
	}

	fileName := fmt.Sprintf(".oauth-%s-%s.oauth", canonicalProvider, state)
	filePath := filepath.Join(authDir, fileName)
	payload := oauthCallbackFilePayload{
		Code:  strings.TrimSpace(code),
		State: strings.TrimSpace(state),
		Error: strings.TrimSpace(errorMessage),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal oauth callback payload: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return "", fmt.Errorf("write oauth callback file: %w", err)
	}
	return filePath, nil
}

func WriteOAuthCallbackFileForPendingSession(authDir, provider, state, code, errorMessage string) (string, error) {
	canonicalProvider, err := NormalizeOAuthProvider(provider)
	if err != nil {
		return "", err
	}
	if !IsOAuthSessionPending(state, canonicalProvider) {
		return "", errOAuthSessionNotPending
	}
	if err := SubmitOAuthCallbackForPendingSession(canonicalProvider, state, code, errorMessage); err != nil {
		return "", err
	}
	if strings.TrimSpace(authDir) == "" {
		return "", nil
	}
	return WriteOAuthCallbackFile(authDir, canonicalProvider, state, code, errorMessage)
}
