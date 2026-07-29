package quotaalert

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const telegramSecretPurpose = "telegram-bot-token"

var ErrTelegramUnavailable = errors.New("telegram delivery unavailable")

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
func (s *TelegramStoreSender) Send(ctx context.Context, batch NotificationBatch) error {
	if s == nil || s.store == nil {
		return ErrTelegramUnavailable
	}
	settings, secret, err := s.store.LoadSettingsWithSecret(ctx)
	if err != nil {
		return fmt.Errorf("load telegram destination: %w", err)
	}
	if !settings.Telegram.Enabled || settings.Telegram.ChatID == "" || secret == nil || s.cipher == nil {
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
		return ErrTelegramUnavailable
	}
	return sender.Send(ctx, batch)
}
