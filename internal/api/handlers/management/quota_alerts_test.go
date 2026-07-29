package management

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/quotaalert"
)

type quotaAlertMemoryStore struct {
	settings               quotaalert.Settings
	secret                 *quotaalert.EncryptedSecret
	states                 []quotaalert.CurrentState
	events                 []quotaalert.TransitionEvent
	deliveries             map[string]quotaalert.EventDeliveryStatus
	loadSettingsErr        error
	saveSettingsErr        error
	listStatesErr          error
	listEventsErr          error
	ackErr                 error
	saveSettingsCount      int
	saveSettingsWithSecret int
	ackCount               int
	lastExpectedRevision   int64
	lastSavedSettings      quotaalert.Settings
	lastSecretMode         quotaalert.SecretUpdateMode
	lastSecretPurpose      string
	lastAckID              string
}

func newQuotaAlertMemoryStore() *quotaAlertMemoryStore {
	settings := quotaalert.DefaultSettings()
	settings.Revision = 7
	return &quotaAlertMemoryStore{settings: settings}
}

func (s *quotaAlertMemoryStore) LoadSettings(context.Context) (quotaalert.Settings, error) {
	if s.loadSettingsErr != nil {
		return quotaalert.Settings{}, s.loadSettingsErr
	}
	return s.settings, nil
}

func (s *quotaAlertMemoryStore) LoadSettingsWithSecret(context.Context) (quotaalert.Settings, *quotaalert.EncryptedSecret, error) {
	if s.loadSettingsErr != nil {
		return quotaalert.Settings{}, nil, s.loadSettingsErr
	}
	return s.settings, s.secret, nil
}

func (s *quotaAlertMemoryStore) SaveSettings(_ context.Context, expectedRevision int64, settings quotaalert.Settings) (quotaalert.Settings, error) {
	if s.saveSettingsErr != nil {
		return quotaalert.Settings{}, s.saveSettingsErr
	}
	s.saveSettingsCount++
	s.lastExpectedRevision = expectedRevision
	s.lastSavedSettings = settings
	settings.Revision = expectedRevision + 1
	s.settings = settings
	return settings, nil
}

func (s *quotaAlertMemoryStore) SaveSettingsWithSecret(_ context.Context, expectedRevision int64, settings quotaalert.Settings, update quotaalert.SecretUpdate, cipher *quotaalert.SecretCipher, purpose string) (quotaalert.Settings, error) {
	if s.saveSettingsErr != nil {
		return quotaalert.Settings{}, s.saveSettingsErr
	}
	nextSecret, err := update.Apply(s.secret, cipher, purpose)
	if err != nil {
		return quotaalert.Settings{}, err
	}
	s.saveSettingsWithSecret++
	s.lastExpectedRevision = expectedRevision
	s.lastSavedSettings = settings
	s.lastSecretMode = update.Mode()
	s.lastSecretPurpose = purpose
	s.secret = nextSecret
	settings.Telegram.TokenConfigured = nextSecret != nil
	settings.Revision = expectedRevision + 1
	s.settings = settings
	return settings, nil
}

func (s *quotaAlertMemoryStore) TryAcquireCollection(context.Context) (quotaalert.CollectionLease, bool, error) {
	return nil, false, nil
}

func (s *quotaAlertMemoryStore) LoadStates(context.Context, []quotaalert.StateIdentity) ([]quotaalert.CurrentState, error) {
	return nil, nil
}

func (s *quotaAlertMemoryStore) CommitCollection(context.Context, quotaalert.CollectionLease, quotaalert.CollectionCommit) error {
	return nil
}

func (s *quotaAlertMemoryStore) ListStates(_ context.Context, page quotaalert.PageRequest) (quotaalert.Page[quotaalert.CurrentState], error) {
	if _, err := page.Normalize(); err != nil {
		return quotaalert.Page[quotaalert.CurrentState]{}, err
	}
	if s.listStatesErr != nil {
		return quotaalert.Page[quotaalert.CurrentState]{}, s.listStatesErr
	}
	return quotaalert.Page[quotaalert.CurrentState]{Items: s.states, NextCursor: "next-state"}, nil
}

func (s *quotaAlertMemoryStore) ListEvents(_ context.Context, page quotaalert.PageRequest) (quotaalert.Page[quotaalert.TransitionEvent], error) {
	if _, err := page.Normalize(); err != nil {
		return quotaalert.Page[quotaalert.TransitionEvent]{}, err
	}
	if s.listEventsErr != nil {
		return quotaalert.Page[quotaalert.TransitionEvent]{}, s.listEventsErr
	}
	return quotaalert.Page[quotaalert.TransitionEvent]{Items: s.events, NextCursor: "next-event"}, nil
}

func (s *quotaAlertMemoryStore) ListEventDeliveryStatuses(context.Context, []string) (map[string]quotaalert.EventDeliveryStatus, error) {
	return s.deliveries, nil
}

func (s *quotaAlertMemoryStore) AcknowledgeEvent(_ context.Context, eventID string, _ time.Time) error {
	s.ackCount++
	s.lastAckID = eventID
	return s.ackErr
}

func (s *quotaAlertMemoryStore) PruneEvents(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}
func (s *quotaAlertMemoryStore) PruneNotificationBatches(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}
func (s *quotaAlertMemoryStore) ClaimNotificationBatches(context.Context, quotaalert.NotificationClaimOptions) ([]quotaalert.NotificationClaim, error) {
	return nil, nil
}
func (s *quotaAlertMemoryStore) ResolveNotification(context.Context, quotaalert.NotificationResult) error {
	return nil
}

func TestQuotaAlertSettingsGetRedactsTelegramSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newQuotaAlertMemoryStore()
	store.settings.Telegram = quotaalert.TelegramDestination{Enabled: true, ChatID: "123", TokenConfigured: true}
	h := &Handler{quotaAlertStore: store}

	rec := performQuotaAlertRequest(t, h.GetQuotaAlertSettings, http.MethodGet, "/v0/management/quota-alerts/settings", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, forbidden := range []string{"token\"", "ciphertext", "nonce", "telegram_secret", "/bot"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `"token_configured":true`) {
		t.Fatalf("response missing token_configured=true: %s", body)
	}
}

func TestQuotaAlertSettingsPutUsesQuotaStoreOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newQuotaAlertMemoryStore()
	store.settings.Telegram = quotaalert.TelegramDestination{Enabled: false, ChatID: "chat", TokenConfigured: true}
	configStore := &recordingConfigStore{}
	h := &Handler{quotaAlertStore: store, configStore: configStore}

	body := `{"revision":7,"enabled":true,"poll_interval_seconds":300,"warning_threshold":15,"notify_recovery":true,"reminder_interval_seconds":600,"providers":[{"provider":"claude","enabled":true,"warning_threshold":20}]}`
	rec := performQuotaAlertRequest(t, h.PutQuotaAlertSettings, http.MethodPut, "/v0/management/quota-alerts/settings", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if store.saveSettingsCount != 1 {
		t.Fatalf("SaveSettings calls = %d, want 1", store.saveSettingsCount)
	}
	if store.saveSettingsWithSecret != 0 {
		t.Fatalf("SaveSettingsWithSecret calls = %d, want 0", store.saveSettingsWithSecret)
	}
	if configStore.saveCount != 0 {
		t.Fatalf("config store saves = %d, want 0", configStore.saveCount)
	}
	if !store.lastSavedSettings.Telegram.TokenConfigured || store.lastSavedSettings.Telegram.ChatID != "chat" {
		t.Fatalf("telegram destination was not preserved: %#v", store.lastSavedSettings.Telegram)
	}
}

func TestQuotaAlertTelegramPutReplaceAndClearAreWriteOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cipher := testQuotaAlertCipher(t)
	store := newQuotaAlertMemoryStore()
	h := &Handler{quotaAlertStore: store, quotaAlertCipher: cipher}

	replace := `{"revision":7,"enabled":true,"chat_id":"123","token":"secret-token"}`
	rec := performQuotaAlertRequest(t, h.PutQuotaAlertTelegram, http.MethodPut, "/v0/management/quota-alerts/telegram", replace)
	if rec.Code != http.StatusOK {
		t.Fatalf("replace status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if store.saveSettingsWithSecret != 1 || store.lastSecretMode != quotaalert.SecretReplace || store.lastSecretPurpose != quotaAlertTelegramSecretPurpose {
		t.Fatalf("unexpected secret write: calls=%d mode=%d purpose=%q", store.saveSettingsWithSecret, store.lastSecretMode, store.lastSecretPurpose)
	}
	assertNoTelegramSecretInBody(t, rec.Body.String())
	if !strings.Contains(rec.Body.String(), `"token_configured":true`) {
		t.Fatalf("replace response missing token_configured=true: %s", rec.Body.String())
	}

	clear := `{"revision":8,"enabled":false,"chat_id":"","clear_token":true}`
	rec = performQuotaAlertRequest(t, h.PutQuotaAlertTelegram, http.MethodPut, "/v0/management/quota-alerts/telegram", clear)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if store.lastSecretMode != quotaalert.SecretClear {
		t.Fatalf("secret mode = %d, want clear", store.lastSecretMode)
	}
	assertNoTelegramSecretInBody(t, rec.Body.String())
	if !strings.Contains(rec.Body.String(), `"token_configured":false`) {
		t.Fatalf("clear response missing token_configured=false: %s", rec.Body.String())
	}
}

func TestQuotaAlertStateEventsAndAck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newQuotaAlertMemoryStore()
	now := time.Date(2026, 7, 29, 4, 0, 0, 0, time.UTC)
	identity := quotaalert.StateIdentity{AuthID: "auth-1", Provider: quotaalert.ProviderClaude, Resource: "messages", Window: "daily"}
	store.states = []quotaalert.CurrentState{{
		Identity: identity, AuthLabel: "Claude account", Alert: quotaalert.AlertWarning, Health: quotaalert.CollectionReliable,
		Remaining: 9, RemainingKnown: true, ResetAt: now.Add(time.Hour), ResetKnown: true, ObservedAt: now, TransitionedAt: now.Add(-time.Minute), UpdatedAt: now, Revision: 3,
	}}
	store.events = []quotaalert.TransitionEvent{{
		ID: "event-1", Identity: identity, AuthLabel: "Claude account", Kind: quotaalert.TransitionWarning,
		From: quotaalert.AlertHealthy, To: quotaalert.AlertWarning, Remaining: 9, RemainingKnown: true, OccurredAt: now,
	}}
	store.deliveries = map[string]quotaalert.EventDeliveryStatus{
		"event-1": {Status: "sent", AttemptCount: 1, SentAt: now},
	}
	h := &Handler{quotaAlertStore: store}

	state := performQuotaAlertRequest(t, h.GetQuotaAlertState, http.MethodGet, "/v0/management/quota-alerts/state?limit=10", "")
	if state.Code != http.StatusOK || !strings.Contains(state.Body.String(), `"next_cursor":"next-state"`) || !strings.Contains(state.Body.String(), `"remaining":9`) {
		t.Fatalf("unexpected state response status=%d body=%s", state.Code, state.Body.String())
	}
	events := performQuotaAlertRequest(t, h.GetQuotaAlertEvents, http.MethodGet, "/v0/management/quota-alerts/events?limit=10", "")
	if events.Code != http.StatusOK || !strings.Contains(events.Body.String(), `"next_cursor":"next-event"`) || !strings.Contains(events.Body.String(), `"kind":"warning"`) || !strings.Contains(events.Body.String(), `"delivery":{"status":"sent"`) {
		t.Fatalf("unexpected events response status=%d body=%s", events.Code, events.Body.String())
	}
	ack := performQuotaAlertRequest(t, h.AckQuotaAlertEvent, http.MethodPost, "/v0/management/quota-alerts/events/event-1/ack", "")
	if ack.Code != http.StatusOK || store.ackCount != 1 || store.lastAckID != "event-1" {
		t.Fatalf("unexpected ack status=%d body=%s count=%d id=%q", ack.Code, ack.Body.String(), store.ackCount, store.lastAckID)
	}
	store.ackErr = sql.ErrNoRows
	missing := performQuotaAlertRequest(t, h.AckQuotaAlertEvent, http.MethodPost, "/v0/management/quota-alerts/events/missing/ack", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing ack status=%d, want %d body=%s", missing.Code, http.StatusNotFound, missing.Body.String())
	}
}

func TestQuotaAlertEventsRejectPartialLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{quotaAlertStore: newQuotaAlertMemoryStore()}
	rec := performQuotaAlertRequest(t, h.GetQuotaAlertEvents, http.MethodGet, "/v0/management/quota-alerts/events?limit=1abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestQuotaAlertListStoreErrorsAreUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newQuotaAlertMemoryStore()
	store.listStatesErr = errors.New("database unavailable")
	store.listEventsErr = errors.New("database unavailable")
	h := &Handler{quotaAlertStore: store}

	state := performQuotaAlertRequest(t, h.GetQuotaAlertState, http.MethodGet, "/v0/management/quota-alerts/state?limit=10", "")
	if state.Code != http.StatusServiceUnavailable {
		t.Fatalf("state status = %d, want %d body=%s", state.Code, http.StatusServiceUnavailable, state.Body.String())
	}
	events := performQuotaAlertRequest(t, h.GetQuotaAlertEvents, http.MethodGet, "/v0/management/quota-alerts/events?limit=10", "")
	if events.Code != http.StatusServiceUnavailable {
		t.Fatalf("events status = %d, want %d body=%s", events.Code, http.StatusServiceUnavailable, events.Body.String())
	}
}

func TestQuotaAlertTelegramTestSendUsesSecretWithoutPersistingEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cipher := testQuotaAlertCipher(t)
	secret, err := cipher.Encrypt(quotaAlertTelegramSecretPurpose, []byte("secret-token"))
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	store := newQuotaAlertMemoryStore()
	store.settings.Telegram = quotaalert.TelegramDestination{Enabled: true, ChatID: "123", TokenConfigured: true}
	store.secret = &secret
	h := &Handler{quotaAlertStore: store, quotaAlertCipher: cipher}

	var gotToken, gotChatID string
	previous := quotaAlertSendTelegramTest
	quotaAlertSendTelegramTest = func(_ context.Context, botToken, chatID string) error {
		gotToken = botToken
		gotChatID = chatID
		return nil
	}
	t.Cleanup(func() { quotaAlertSendTelegramTest = previous })

	rec := performQuotaAlertRequest(t, h.TestQuotaAlertTelegram, http.MethodPost, "/v0/management/quota-alerts/telegram/test", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotToken != "secret-token" || gotChatID != "123" {
		t.Fatalf("test send got token=%q chat=%q", gotToken, gotChatID)
	}
	if store.saveSettingsCount != 0 || store.saveSettingsWithSecret != 0 || store.ackCount != 0 {
		t.Fatalf("test send persisted unexpectedly: save=%d saveSecret=%d ack=%d", store.saveSettingsCount, store.saveSettingsWithSecret, store.ackCount)
	}
}

func TestQuotaAlertTelegramTestSendSanitizesFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cipher := testQuotaAlertCipher(t)
	secret, err := cipher.Encrypt(quotaAlertTelegramSecretPurpose, []byte("secret-token"))
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	store := newQuotaAlertMemoryStore()
	store.settings.Telegram = quotaalert.TelegramDestination{Enabled: true, ChatID: "123", TokenConfigured: true}
	store.secret = &secret
	h := &Handler{quotaAlertStore: store, quotaAlertCipher: cipher}

	previous := quotaAlertSendTelegramTest
	quotaAlertSendTelegramTest = func(context.Context, string, string) error {
		return errors.New("https://api.telegram.org/botsecret-token/sendMessage failed")
	}
	t.Cleanup(func() { quotaAlertSendTelegramTest = previous })

	rec := performQuotaAlertRequest(t, h.TestQuotaAlertTelegram, http.MethodPost, "/v0/management/quota-alerts/telegram/test", "")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
	assertNoTelegramSecretInBody(t, rec.Body.String())
}

func performQuotaAlertRequest(t *testing.T, handler gin.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req
	if strings.Contains(target, "/events/") && strings.HasSuffix(target, "/ack") {
		parts := strings.Split(target, "/")
		ctx.Params = gin.Params{{Key: "id", Value: parts[len(parts)-2]}}
	}
	handler(ctx)
	return rec
}

func assertNoTelegramSecretInBody(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{"secret-token", "token\"", "ciphertext", "nonce", "telegram_secret", "/bot"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("response is not JSON: %v body=%s", err, body)
	}
}

func testQuotaAlertCipher(t *testing.T) *quotaalert.SecretCipher {
	t.Helper()
	key := bytes.Repeat([]byte{7}, quotaalert.SecretKeySize)
	cipher, err := quotaalert.NewSecretCipher("test", key)
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	return cipher
}
