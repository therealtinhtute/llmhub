package quotaalert

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTelegramSendsProviderGroupedBatch(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 0, 0, 0, time.UTC)
	batch := telegramTestBatch(t, now, []TransitionEvent{
		telegramTestEvent("event-1", ProviderClaude, TransitionWarning, AlertHealthy, AlertWarning, 5, now),
		telegramTestEvent("event-2", ProviderClaude, TransitionExhausted, AlertWarning, AlertExhausted, 0, now),
	})
	var got struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/botsecret-token/sendMessage" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	sender, err := NewTelegramSender(TelegramSenderConfig{
		BotToken: "secret-token",
		ChatID:   "-100123",
		BaseURL:  server.URL,
		Client:   server.Client(),
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("NewTelegramSender() error = %v", err)
	}
	if err = sender.Send(context.Background(), batch); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got.ChatID != "-100123" {
		t.Fatalf("chat ID = %q", got.ChatID)
	}
	for _, want := range []string{"LLMHub quota alert", "Provider: claude", "auth-1 label", "messages / weekly", "warning", "exhausted", "5%", "0%"} {
		if !strings.Contains(got.Text, want) {
			t.Fatalf("message missing %q: %s", want, got.Text)
		}
	}
	if strings.Contains(got.Text, "secret-token") {
		t.Fatalf("message contains token: %s", got.Text)
	}
}

func TestTelegramTestSendCreatesNoAlertTransitionPayload(t *testing.T) {
	var got struct {
		Text string `json:"text"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	sender, err := NewTelegramSender(TelegramSenderConfig{BotToken: "secret-token", ChatID: "123", BaseURL: server.URL, Client: server.Client(), Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewTelegramSender() error = %v", err)
	}
	if err = sender.SendTest(context.Background()); err != nil {
		t.Fatalf("SendTest() error = %v", err)
	}
	if !strings.Contains(got.Text, "test notification") {
		t.Fatalf("test message = %q", got.Text)
	}
	for _, forbidden := range []string{"warning", "exhausted", "recovery", "auth-", "secret-token"} {
		if strings.Contains(got.Text, forbidden) {
			t.Fatalf("test message contains transition/token text %q: %s", forbidden, got.Text)
		}
	}
}

func TestTelegramSanitizesErrorsAndBoundsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret-token leaked by upstream", http.StatusBadGateway)
	}))
	defer server.Close()
	sender, err := NewTelegramSender(TelegramSenderConfig{BotToken: "secret-token", ChatID: "123", BaseURL: server.URL, Client: server.Client(), Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewTelegramSender() error = %v", err)
	}
	err = sender.Send(context.Background(), telegramTestBatch(t, time.Now(), []TransitionEvent{telegramTestEvent("event-1", ProviderCodex, TransitionWarning, AlertHealthy, AlertWarning, 1, time.Now())}))
	if err == nil {
		t.Fatal("Send() error = nil")
	}
	if strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "/bot") || strings.Contains(err.Error(), server.URL) {
		t.Fatalf("telegram error leaked sensitive data: %v", err)
	}

	largeEvents := make([]TransitionEvent, 0, MaxNotificationBatchEvents)
	for i := 0; i < MaxNotificationBatchEvents; i++ {
		event := telegramTestEvent("event-large-"+strings.Repeat("x", 10)+string(rune('a'+i%26))+"-"+string(rune('0'+i/26)), ProviderKimi, TransitionWarning, AlertHealthy, AlertWarning, 1, time.Now())
		event.AuthLabel = strings.Repeat("l", MaxAuthLabelLength)
		largeEvents = append(largeEvents, event)
	}
	largeBatch := telegramTestBatch(t, time.Now(), largeEvents)
	if text := RenderTelegramMessage(largeBatch); len([]byte(text)) > DefaultTelegramMaxMessageBytes {
		t.Fatalf("message length = %d, want <= %d", len([]byte(text)), DefaultTelegramMaxMessageBytes)
	}
}

func TestTelegramValidatesConfig(t *testing.T) {
	for _, config := range []TelegramSenderConfig{
		{},
		{BotToken: "token", ChatID: ""},
		{BotToken: strings.Repeat("t", MaxSecretValueLength+1), ChatID: "123"},
		{BotToken: "token", ChatID: strings.Repeat("1", MaxTelegramChatIDLength+1)},
		{BotToken: "token", ChatID: "123", BaseURL: "https://api.telegram.org/path"},
	} {
		if _, err := NewTelegramSender(config); err == nil {
			t.Fatalf("NewTelegramSender(%#v) error = nil", config)
		}
	}
}

func TestTelegramStoreSenderDecryptsConfiguredToken(t *testing.T) {
	now := time.Date(2026, time.July, 29, 7, 0, 0, 0, time.UTC)
	cipher, err := NewSecretCipher("runtime", bytes.Repeat([]byte{1}, SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	secret, err := cipher.Encrypt(telegramSecretPurpose, []byte("secret-token"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	store := newServiceTestStore()
	store.settings.Telegram.Enabled = true
	store.settings.Telegram.ChatID = "123"
	store.settings.Telegram.TokenConfigured = true
	store.secret = &secret

	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	sender, err := NewTelegramStoreSender(TelegramStoreSenderConfig{Store: store, Cipher: cipher, BaseURL: server.URL, Client: server.Client(), Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewTelegramStoreSender() error = %v", err)
	}
	if err = sender.Send(context.Background(), telegramTestBatch(t, now, []TransitionEvent{telegramTestEvent("event-db", ProviderClaude, TransitionWarning, AlertHealthy, AlertWarning, 5, now)})); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if path != "/botsecret-token/sendMessage" {
		t.Fatalf("Telegram path = %q", path)
	}
}

func TestTelegramStoreSenderUnavailableWithoutMatchingCipher(t *testing.T) {
	cipher, err := NewSecretCipher("runtime", bytes.Repeat([]byte{1}, SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	wrongCipher, err := NewSecretCipher("wrong", bytes.Repeat([]byte{2}, SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher(wrong) error = %v", err)
	}
	secret, err := cipher.Encrypt(telegramSecretPurpose, []byte("secret-token"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	store := newServiceTestStore()
	store.settings.Telegram.Enabled = true
	store.settings.Telegram.ChatID = "123"
	store.settings.Telegram.TokenConfigured = true
	store.secret = &secret

	for name, testCipher := range map[string]*SecretCipher{"missing": nil, "wrong": wrongCipher} {
		t.Run(name, func(t *testing.T) {
			sender, err := NewTelegramStoreSender(TelegramStoreSenderConfig{Store: store, Cipher: testCipher})
			if err != nil {
				t.Fatalf("NewTelegramStoreSender() error = %v", err)
			}
			err = sender.Send(context.Background(), telegramTestBatch(t, time.Now(), []TransitionEvent{telegramTestEvent("event-"+name, ProviderClaude, TransitionWarning, AlertHealthy, AlertWarning, 5, time.Now())}))
			if !errors.Is(err, ErrTelegramUnavailable) {
				t.Fatalf("Send() error = %v, want ErrTelegramUnavailable", err)
			}
		})
	}
}

func telegramTestBatch(t *testing.T, now time.Time, events []TransitionEvent) NotificationBatch {
	t.Helper()
	batch, err := NewNotificationBatch(events[0].Identity.Provider, events, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	return batch
}

func telegramTestEvent(id string, provider Provider, kind TransitionKind, from AlertState, to AlertState, remaining Percentage, now time.Time) TransitionEvent {
	return TransitionEvent{
		ID: id,
		Identity: StateIdentity{
			AuthID:   "auth-1",
			Provider: provider,
			Resource: "messages",
			Window:   "weekly",
		},
		AuthLabel:      "auth-1 label",
		Kind:           kind,
		From:           from,
		To:             to,
		Remaining:      remaining,
		RemainingKnown: true,
		OccurredAt:     now,
	}
}
