package quotaalert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	DefaultTelegramAPIBase         = "https://api.telegram.org"
	DefaultTelegramTimeout         = 10 * time.Second
	DefaultTelegramMaxMessageBytes = 3900
)

// TelegramSenderConfig configures the single Telegram destination supported by quota alerts.
type TelegramSenderConfig struct {
	BotToken string
	ChatID   string
	BaseURL  string
	Timeout  time.Duration
	Client   *http.Client
}

// TelegramSender sends provider-grouped quota alert batches to Telegram.
type TelegramSender struct {
	botToken string
	chatID   string
	baseURL  *url.URL
	timeout  time.Duration
	client   *http.Client
}

// NewTelegramSender creates a fixed-host Telegram sender.
func NewTelegramSender(config TelegramSenderConfig) (*TelegramSender, error) {
	botToken := strings.TrimSpace(config.BotToken)
	chatID := strings.TrimSpace(config.ChatID)
	if botToken == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	if len(botToken) > MaxSecretValueLength {
		return nil, fmt.Errorf("telegram bot token must not exceed %d bytes", MaxSecretValueLength)
	}
	if chatID == "" {
		return nil, fmt.Errorf("telegram chat ID is required")
	}
	if len(chatID) > MaxTelegramChatIDLength {
		return nil, fmt.Errorf("telegram chat ID must not exceed %d bytes", MaxTelegramChatIDLength)
	}
	baseText := strings.TrimSpace(config.BaseURL)
	if baseText == "" {
		baseText = DefaultTelegramAPIBase
	}
	baseURL, err := url.Parse(baseText)
	if err != nil || baseURL == nil || baseURL.Scheme != "https" && baseURL.Scheme != "http" || baseURL.Host == "" {
		return nil, fmt.Errorf("telegram API base URL is invalid")
	}
	baseURL.RawQuery = ""
	baseURL.Fragment = ""
	if strings.Trim(baseURL.Path, "/") != "" {
		return nil, fmt.Errorf("telegram API base URL must not include a path")
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTelegramTimeout
	}
	if timeout < time.Second || timeout > time.Minute {
		return nil, fmt.Errorf("telegram timeout must be between 1s and 1m")
	}
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &TelegramSender{botToken: botToken, chatID: chatID, baseURL: baseURL, timeout: timeout, client: client}, nil
}

// Send delivers one provider-grouped notification batch.
func (s *TelegramSender) Send(ctx context.Context, batch NotificationBatch) error {
	message := RenderTelegramMessage(batch)
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("telegram message is empty")
	}
	return s.sendMessage(ctx, message)
}

// SendTest sends an explicit test notification without creating any alert transition payload.
func (s *TelegramSender) SendTest(ctx context.Context) error {
	return s.sendMessage(ctx, "LLMHub quota alert test notification. If you received this, Telegram delivery is configured.")
}

// RenderTelegramMessage renders a bounded provider-grouped quota alert message.
func RenderTelegramMessage(batch NotificationBatch) string {
	events := batch.Events()
	sort.Slice(events, func(left, right int) bool {
		leftEvent := events[left]
		rightEvent := events[right]
		if leftEvent.AuthLabel != rightEvent.AuthLabel {
			return leftEvent.AuthLabel < rightEvent.AuthLabel
		}
		if leftEvent.Identity.Resource != rightEvent.Identity.Resource {
			return leftEvent.Identity.Resource < rightEvent.Identity.Resource
		}
		if leftEvent.Identity.Window != rightEvent.Identity.Window {
			return leftEvent.Identity.Window < rightEvent.Identity.Window
		}
		return leftEvent.ID < rightEvent.ID
	})
	lines := []string{
		"LLMHub quota alert",
		"Provider: " + string(batch.Provider()),
	}
	for _, event := range events {
		line := fmt.Sprintf("- %s: %s %s / %s", event.AuthLabel, event.Kind, event.Identity.Resource, event.Identity.Window)
		if event.RemainingKnown {
			line += fmt.Sprintf(" (%g%% remaining)", event.Remaining)
		}
		if event.ResetKnown {
			line += "; resets " + event.ResetAt.UTC().Format(time.RFC3339)
		}
		lines = append(lines, line)
	}
	return boundTelegramMessage(strings.Join(lines, "\n"))
}

func (s *TelegramSender) sendMessage(ctx context.Context, message string) error {
	if s == nil || s.client == nil || s.baseURL == nil {
		return fmt.Errorf("telegram sender is not initialized")
	}
	if len([]byte(message)) > DefaultTelegramMaxMessageBytes {
		message = boundTelegramMessage(message)
	}
	payload := map[string]string{
		"chat_id": s.chatID,
		"text":    message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram payload is invalid")
	}
	requestURL := *s.baseURL
	requestURL.Path = "/bot" + s.botToken + "/sendMessage"
	requestCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request is invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed: %s", s.sanitize(err.Error()))
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("telegram request returned HTTP %d: %s", resp.StatusCode, s.sanitize(string(responseBody)))
	}
	return nil
}

func (s *TelegramSender) sanitize(message string) string {
	message = strings.ReplaceAll(message, s.botToken, "[redacted]")
	if s.baseURL != nil {
		message = strings.ReplaceAll(message, s.baseURL.String(), "[redacted]")
	}
	message = strings.ReplaceAll(message, "/bot[redacted]/sendMessage", "[redacted]")
	if strings.Contains(message, "/bot") {
		message = "telegram request failed"
	}
	return strings.TrimSpace(message)
}

func boundTelegramMessage(message string) string {
	if len([]byte(message)) <= DefaultTelegramMaxMessageBytes {
		return message
	}
	marker := "\n... truncated"
	limit := DefaultTelegramMaxMessageBytes - len([]byte(marker))
	if limit < 0 {
		limit = DefaultTelegramMaxMessageBytes
		marker = ""
	}
	buffer := make([]byte, 0, DefaultTelegramMaxMessageBytes)
	for _, r := range message {
		next := string(r)
		if len(buffer)+len([]byte(next)) > limit {
			break
		}
		buffer = append(buffer, next...)
	}
	return string(buffer) + marker
}
