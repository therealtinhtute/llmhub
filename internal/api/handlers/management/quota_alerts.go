package management

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/quotaalert"
)

const quotaAlertTelegramSecretPurpose = "telegram-bot-token"

var quotaAlertSendTelegramTest = func(ctx context.Context, botToken, chatID string) error {
	sender, err := quotaalert.NewTelegramSender(quotaalert.TelegramSenderConfig{BotToken: botToken, ChatID: chatID})
	if err != nil {
		return err
	}
	return sender.SendTest(ctx)
}

type quotaAlertSettingsResponse struct {
	Revision         int64                          `json:"revision"`
	Enabled          bool                           `json:"enabled"`
	PollIntervalSec  int64                          `json:"poll_interval_seconds"`
	WarningThreshold float64                        `json:"warning_threshold"`
	NotifyRecovery   bool                           `json:"notify_recovery"`
	ReminderSec      int64                          `json:"reminder_interval_seconds"`
	Providers        []quotaAlertProviderResponse   `json:"providers"`
	Telegram         quotaAlertTelegramReadResponse `json:"telegram"`
}

type quotaAlertProviderResponse struct {
	Provider         string   `json:"provider"`
	Enabled          bool     `json:"enabled"`
	WarningThreshold *float64 `json:"warning_threshold,omitempty"`
}

type quotaAlertTelegramReadResponse struct {
	Enabled             bool   `json:"enabled"`
	ChatID              string `json:"chat_id"`
	TokenConfigured     bool   `json:"token_configured"`
	SecretKeyConfigured bool   `json:"secret_key_configured"`
}

type quotaAlertSettingsRequest struct {
	Revision         int64                       `json:"revision"`
	Enabled          bool                        `json:"enabled"`
	PollIntervalSec  int64                       `json:"poll_interval_seconds"`
	WarningThreshold float64                     `json:"warning_threshold"`
	NotifyRecovery   bool                        `json:"notify_recovery"`
	ReminderSec      int64                       `json:"reminder_interval_seconds"`
	Providers        []quotaAlertProviderRequest `json:"providers"`
}

type quotaAlertProviderRequest struct {
	Provider         string   `json:"provider"`
	Enabled          bool     `json:"enabled"`
	WarningThreshold *float64 `json:"warning_threshold"`
}

type quotaAlertTelegramRequest struct {
	Revision int64   `json:"revision"`
	Enabled  bool    `json:"enabled"`
	ChatID   string  `json:"chat_id"`
	Token    *string `json:"token"`
	Clear    bool    `json:"clear_token"`
}

type quotaAlertPageResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type quotaAlertStateResponse struct {
	AuthID         string   `json:"auth_id"`
	Provider       string   `json:"provider"`
	Resource       string   `json:"resource"`
	Window         string   `json:"window"`
	AuthLabel      string   `json:"auth_label"`
	Alert          string   `json:"alert"`
	Health         string   `json:"health"`
	Remaining      *float64 `json:"remaining,omitempty"`
	ResetAt        *string  `json:"reset_at,omitempty"`
	ObservedAt     string   `json:"observed_at"`
	TransitionedAt string   `json:"transitioned_at"`
	UpdatedAt      string   `json:"updated_at"`
	Revision       int64    `json:"revision"`
}

type quotaAlertEventResponse struct {
	ID             string                           `json:"id"`
	AuthID         string                           `json:"auth_id"`
	Provider       string                           `json:"provider"`
	Resource       string                           `json:"resource"`
	Window         string                           `json:"window"`
	AuthLabel      string                           `json:"auth_label"`
	Kind           string                           `json:"kind"`
	From           string                           `json:"from"`
	To             string                           `json:"to"`
	Remaining      *float64                         `json:"remaining,omitempty"`
	ResetAt        *string                          `json:"reset_at,omitempty"`
	OccurredAt     string                           `json:"occurred_at"`
	AcknowledgedAt *string                          `json:"acknowledged_at,omitempty"`
	Delivery       *quotaAlertEventDeliveryResponse `json:"delivery,omitempty"`
}

type quotaAlertEventDeliveryResponse struct {
	Status       string  `json:"status"`
	FailureCode  string  `json:"failure_code,omitempty"`
	AttemptCount int     `json:"attempt_count"`
	SentAt       *string `json:"sent_at,omitempty"`
}

type quotaAlertEventDeliveryStore interface {
	ListEventDeliveryStatuses(context.Context, []string) (map[string]quotaalert.EventDeliveryStatus, error)
}

// SetQuotaAlertStore configures database-backed quota-alert management APIs.
func (h *Handler) SetQuotaAlertStore(store quotaalert.Store) {
	if h == nil {
		return
	}
	h.quotaAlertStore = store
}

// SetQuotaAlertSecretCipher configures write-only Telegram token encryption for quota alerts.
func (h *Handler) SetQuotaAlertSecretCipher(cipher *quotaalert.SecretCipher) {
	if h == nil {
		return
	}
	h.quotaAlertCipher = cipher
}

// GetQuotaAlertSettings returns database-backed quota alert settings with Telegram secret redacted.
func (h *Handler) GetQuotaAlertSettings(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	settings, err := store.LoadSettings(c.Request.Context())
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "quota alert settings unavailable")
		return
	}
	c.JSON(http.StatusOK, h.quotaAlertSettingsDTO(settings))
}

// PutQuotaAlertSettings updates global and provider quota alert settings without touching YAML config.
func (h *Handler) PutQuotaAlertSettings(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	var req quotaAlertSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.quotaAlertError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	current, err := store.LoadSettings(c.Request.Context())
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "quota alert settings unavailable")
		return
	}
	settings, err := settingsFromRequest(req, current.Telegram)
	if err != nil {
		h.quotaAlertError(c, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := store.SaveSettings(c.Request.Context(), req.Revision, settings)
	if err != nil {
		h.quotaAlertError(c, quotaAlertWriteStatus(err), quotaAlertSafeError(err))
		return
	}
	c.JSON(http.StatusOK, h.quotaAlertSettingsDTO(saved))
}

// PutQuotaAlertTelegram updates the single Telegram destination and write-only bot token.
func (h *Handler) PutQuotaAlertTelegram(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	var req quotaAlertTelegramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.quotaAlertError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	current, err := store.LoadSettings(c.Request.Context())
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "quota alert settings unavailable")
		return
	}
	settings := current
	settings.Telegram.Enabled = req.Enabled
	settings.Telegram.ChatID = strings.TrimSpace(req.ChatID)
	settings.Telegram.TokenConfigured = current.Telegram.TokenConfigured
	update := quotaalert.PreserveSecret()
	cipher := h.quotaAlertCipher
	if req.Clear {
		update = quotaalert.ClearSecret()
		settings.Telegram.TokenConfigured = false
	} else if req.Token != nil {
		secretUpdate, errSecret := quotaalert.ReplaceSecret(*req.Token)
		if errSecret != nil {
			h.quotaAlertError(c, http.StatusBadRequest, errSecret.Error())
			return
		}
		cipher, err = h.quotaAlertCipherForWrite()
		if err != nil {
			h.quotaAlertError(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		update = secretUpdate
		settings.Telegram.TokenConfigured = true
	}
	if err = settings.Validate(); err != nil {
		h.quotaAlertError(c, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := store.SaveSettingsWithSecret(c.Request.Context(), req.Revision, settings, update, cipher, quotaAlertTelegramSecretPurpose)
	if err != nil {
		h.quotaAlertError(c, quotaAlertWriteStatus(err), quotaAlertSafeError(err))
		return
	}
	c.JSON(http.StatusOK, h.quotaAlertSettingsDTO(saved))
}

// GetQuotaAlertState lists current quota alert state rows.
func (h *Handler) GetQuotaAlertState(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	page, ok := quotaAlertPageRequest(c)
	if !ok {
		return
	}
	states, err := store.ListStates(c.Request.Context(), page)
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, quotaAlertSafeError(err))
		return
	}
	items := make([]quotaAlertStateResponse, 0, len(states.Items))
	for _, state := range states.Items {
		items = append(items, quotaAlertStateDTO(state))
	}
	c.JSON(http.StatusOK, quotaAlertPageResponse[quotaAlertStateResponse]{Items: items, NextCursor: states.NextCursor})
}

// GetQuotaAlertEvents lists recent in-app quota alert events.
func (h *Handler) GetQuotaAlertEvents(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	page, ok := quotaAlertPageRequest(c)
	if !ok {
		return
	}
	events, err := store.ListEvents(c.Request.Context(), page)
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, quotaAlertSafeError(err))
		return
	}
	deliveries, err := quotaAlertEventDeliveries(c.Request.Context(), store, events.Items)
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, quotaAlertSafeError(err))
		return
	}
	items := make([]quotaAlertEventResponse, 0, len(events.Items))
	for _, event := range events.Items {
		items = append(items, quotaAlertEventDTO(event, deliveries[event.ID]))
	}
	c.JSON(http.StatusOK, quotaAlertPageResponse[quotaAlertEventResponse]{Items: items, NextCursor: events.NextCursor})
}

// AckQuotaAlertEvent idempotently acknowledges one in-app quota alert event.
func (h *Handler) AckQuotaAlertEvent(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	eventID := strings.TrimSpace(c.Param("id"))
	if eventID == "" {
		h.quotaAlertError(c, http.StatusBadRequest, "event id is required")
		return
	}
	if err := store.AcknowledgeEvent(c.Request.Context(), eventID, time.Now().UTC()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.quotaAlertError(c, http.StatusNotFound, "quota alert event not found")
			return
		}
		h.quotaAlertError(c, http.StatusBadRequest, quotaAlertSafeError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TestQuotaAlertTelegram sends an explicit Telegram test message without creating alert events.
func (h *Handler) TestQuotaAlertTelegram(c *gin.Context) {
	store, ok := h.requireQuotaAlertStore(c)
	if !ok {
		return
	}
	settings, secret, err := store.LoadSettingsWithSecret(c.Request.Context())
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "quota alert settings unavailable")
		return
	}
	if !settings.Telegram.Enabled || strings.TrimSpace(settings.Telegram.ChatID) == "" || secret == nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "telegram destination is not fully configured")
		return
	}
	cipher, err := h.quotaAlertCipherForWrite()
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	token, err := cipher.Decrypt(quotaAlertTelegramSecretPurpose, *secret)
	if err != nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "telegram bot token cannot be decrypted")
		return
	}
	if err = quotaAlertSendTelegramTest(c.Request.Context(), string(token), settings.Telegram.ChatID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "required") || strings.Contains(strings.ToLower(err.Error()), "invalid") {
			h.quotaAlertError(c, http.StatusServiceUnavailable, "telegram destination is not fully configured")
			return
		}
		h.quotaAlertError(c, http.StatusBadGateway, "telegram test send failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) requireQuotaAlertStore(c *gin.Context) (quotaalert.Store, bool) {
	if h == nil || h.quotaAlertStore == nil {
		h.quotaAlertError(c, http.StatusServiceUnavailable, "quota alert store is not configured")
		return nil, false
	}
	return h.quotaAlertStore, true
}

func (h *Handler) quotaAlertCipherForWrite() (*quotaalert.SecretCipher, error) {
	if h == nil || h.quotaAlertCipher == nil {
		return nil, fmt.Errorf("quota alert secret key is not configured")
	}
	return h.quotaAlertCipher, nil
}

func (h *Handler) quotaAlertSettingsDTO(settings quotaalert.Settings) quotaAlertSettingsResponse {
	providers := make([]quotaAlertProviderResponse, 0, len(settings.ProviderOverrides))
	for _, override := range settings.ProviderOverrides {
		var threshold *float64
		if override.WarningThreshold != nil {
			value := float64(*override.WarningThreshold)
			threshold = &value
		}
		providers = append(providers, quotaAlertProviderResponse{Provider: string(override.Provider), Enabled: override.Enabled, WarningThreshold: threshold})
	}
	return quotaAlertSettingsResponse{
		Revision:         settings.Revision,
		Enabled:          settings.Enabled,
		PollIntervalSec:  int64(settings.PollInterval / time.Second),
		WarningThreshold: float64(settings.WarningThreshold),
		NotifyRecovery:   settings.NotifyRecovery,
		ReminderSec:      int64(settings.ReminderInterval / time.Second),
		Providers:        providers,
		Telegram: quotaAlertTelegramReadResponse{
			Enabled:             settings.Telegram.Enabled,
			ChatID:              settings.Telegram.ChatID,
			TokenConfigured:     settings.Telegram.TokenConfigured,
			SecretKeyConfigured: h != nil && h.quotaAlertCipher != nil,
		},
	}
}

func settingsFromRequest(req quotaAlertSettingsRequest, telegram quotaalert.TelegramDestination) (quotaalert.Settings, error) {
	settings := quotaalert.Settings{
		Revision:         req.Revision,
		Enabled:          req.Enabled,
		PollInterval:     time.Duration(req.PollIntervalSec) * time.Second,
		WarningThreshold: quotaalert.Percentage(req.WarningThreshold),
		NotifyRecovery:   req.NotifyRecovery,
		ReminderInterval: time.Duration(req.ReminderSec) * time.Second,
		Telegram:         telegram,
	}
	settings.ProviderOverrides = make([]quotaalert.ProviderOverride, 0, len(req.Providers))
	for _, provider := range req.Providers {
		override := quotaalert.ProviderOverride{Provider: quotaalert.Provider(strings.TrimSpace(provider.Provider)), Enabled: provider.Enabled}
		if provider.WarningThreshold != nil {
			threshold := quotaalert.Percentage(*provider.WarningThreshold)
			override.WarningThreshold = &threshold
		}
		settings.ProviderOverrides = append(settings.ProviderOverrides, override)
	}
	if err := settings.Validate(); err != nil {
		return quotaalert.Settings{}, err
	}
	return settings, nil
}

func quotaAlertStateDTO(state quotaalert.CurrentState) quotaAlertStateResponse {
	var remaining *float64
	if state.RemainingKnown {
		value := float64(state.Remaining)
		remaining = &value
	}
	return quotaAlertStateResponse{
		AuthID:         state.Identity.AuthID,
		Provider:       string(state.Identity.Provider),
		Resource:       state.Identity.Resource,
		Window:         state.Identity.Window,
		AuthLabel:      state.AuthLabel,
		Alert:          string(state.Alert),
		Health:         string(state.Health),
		Remaining:      remaining,
		ResetAt:        timePtr(state.ResetKnown, state.ResetAt),
		ObservedAt:     formatQuotaAlertTime(state.ObservedAt),
		TransitionedAt: formatQuotaAlertTime(state.TransitionedAt),
		UpdatedAt:      formatQuotaAlertTime(state.UpdatedAt),
		Revision:       state.Revision,
	}
}

func quotaAlertEventDeliveries(ctx context.Context, store quotaalert.Store, events []quotaalert.TransitionEvent) (map[string]quotaalert.EventDeliveryStatus, error) {
	deliveryStore, ok := store.(quotaAlertEventDeliveryStore)
	if !ok || len(events) == 0 {
		return nil, nil
	}
	eventIDs := make([]string, 0, len(events))
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
	}
	return deliveryStore.ListEventDeliveryStatuses(ctx, eventIDs)
}

func quotaAlertEventDTO(event quotaalert.TransitionEvent, delivery quotaalert.EventDeliveryStatus) quotaAlertEventResponse {
	var remaining *float64
	if event.RemainingKnown {
		value := float64(event.Remaining)
		remaining = &value
	}
	response := quotaAlertEventResponse{
		ID:             event.ID,
		AuthID:         event.Identity.AuthID,
		Provider:       string(event.Identity.Provider),
		Resource:       event.Identity.Resource,
		Window:         event.Identity.Window,
		AuthLabel:      event.AuthLabel,
		Kind:           string(event.Kind),
		From:           string(event.From),
		To:             string(event.To),
		Remaining:      remaining,
		ResetAt:        timePtr(event.ResetKnown, event.ResetAt),
		OccurredAt:     formatQuotaAlertTime(event.OccurredAt),
		AcknowledgedAt: timePtr(!event.AcknowledgedAt.IsZero(), event.AcknowledgedAt),
	}
	if delivery.Status != "" {
		response.Delivery = &quotaAlertEventDeliveryResponse{
			Status:       delivery.Status,
			FailureCode:  delivery.FailureCode,
			AttemptCount: delivery.AttemptCount,
			SentAt:       timePtr(!delivery.SentAt.IsZero(), delivery.SentAt),
		}
	}
	return response
}

func timePtr(ok bool, value time.Time) *string {
	if !ok || value.IsZero() {
		return nil
	}
	formatted := formatQuotaAlertTime(value)
	return &formatted
}

func formatQuotaAlertTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func quotaAlertPageRequest(c *gin.Context) (quotaalert.PageRequest, bool) {
	page := quotaalert.PageRequest{Cursor: c.Query("cursor")}
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return quotaalert.PageRequest{}, false
		}
		page.Limit = parsed
	}
	page, err := page.Normalize()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return quotaalert.PageRequest{}, false
	}
	return page, true
}

func (h *Handler) quotaAlertError(c *gin.Context, status int, message string) {
	if strings.TrimSpace(message) == "" {
		message = "quota alert request failed"
	}
	c.JSON(status, gin.H{"error": message})
}

func quotaAlertWriteStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if strings.Contains(strings.ToLower(err.Error()), "revision conflict") {
		return http.StatusConflict
	}
	return http.StatusBadRequest
}

func quotaAlertSafeError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	for _, forbidden := range []string{"/bot", "telegram_secret", "ciphertext", "nonce"} {
		if strings.Contains(strings.ToLower(message), forbidden) {
			return "quota alert request failed"
		}
	}
	return message
}
