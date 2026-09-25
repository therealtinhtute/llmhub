package quotaalert

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const telegramSecretPurpose = "telegram-bot-token"

// Typed sender errors classify delivery failures for the notification outbox.
// They carry no secret material and are safe to log and persist as failure
// codes on delivery records.
var (
	// ErrSenderUnavailable means no usable sender exists for the batch.
	ErrSenderUnavailable = errors.New("notification sender is unavailable")
	// ErrTelegramUnconfigured means the destination itself is not configured:
	// Telegram disabled, missing chat ID, or no stored bot token.
	ErrTelegramUnconfigured = errors.New("telegram destination is not configured")
	// ErrTelegramUnavailable means the configured destination cannot be used:
	// missing LLMHUB_QUOTA_SECRET_KEY_B64, no cipher, or decryption failure.
	ErrTelegramUnavailable = errors.New("telegram delivery unavailable")
)

// TelegramStoreSender loads the write-only Telegram destination from durable settings for each delivery.
type TelegramStoreSender struct {
	store   Store
	cipher  *SecretCipher
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// TelegramStoreSenderConfig configures DB-backed Telegram delivery.
type TelegramStoreSenderConfig struct {
	Store   Store
	Cipher  *SecretCipher
	BaseURL string
	Timeout time.Duration
	Client  *http.Client
}

// NewTelegramStoreSender creates a Sender that decrypts the Telegram token only during delivery.
func NewTelegramStoreSender(config TelegramStoreSenderConfig) (*TelegramStoreSender, error) {
	if config.Store == nil {
		return nil, fmt.Errorf("quota alert store is required")
	}
	return &TelegramStoreSender{
		store:   config.Store,
		cipher:  config.Cipher,
		baseURL: config.BaseURL,
		timeout: config.Timeout,
		client:  config.Client,
	}, nil
}

// Send delivers one provider batch when Telegram is enabled and decryptable.
// Returned errors classify the failure: ErrTelegramUnconfigured for missing
// destination configuration, ErrTelegramUnavailable for cipher/decryption
// problems, ErrSenderUnavailable when no sender exists; anything else is an
// upstream send failure.
func (s *TelegramStoreSender) Send(ctx context.Context, batch NotificationBatch) error {
	if s == nil || s.store == nil {
		return ErrSenderUnavailable
	}
	settings, secret, err := s.store.LoadSettingsWithSecret(ctx)
	if err != nil {
		return fmt.Errorf("load telegram destination: %w", err)
	}
	if !settings.Telegram.Enabled || settings.Telegram.ChatID == "" || secret == nil {
		return ErrTelegramUnconfigured
	}
	if s.cipher == nil {
		return ErrTelegramUnavailable
	}
	botToken, err := s.cipher.Decrypt(telegramSecretPurpose, *secret)
	if err != nil {
		return ErrTelegramUnavailable
	}
	sender, err := NewTelegramSender(TelegramSenderConfig{
		BotToken: string(botToken),
		ChatID:   settings.Telegram.ChatID,
		BaseURL:  s.baseURL,
		Timeout:  s.timeout,
		Client:   s.client,
	})
	if err != nil {
		return ErrTelegramUnconfigured
	}
	return sender.Send(ctx, batch)
}
