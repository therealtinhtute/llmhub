// Package quotaalert defines the server-side quota alert domain contracts.
package quotaalert

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPollInterval          = 5 * time.Minute
	MinPollInterval              = time.Minute
	MaxPollInterval              = 24 * time.Hour
	DefaultWarningThreshold      = Percentage(10)
	DefaultPageSize              = 50
	MaxPageSize                  = 100
	MaxIdentityFieldLength       = 256
	MaxAuthLabelLength           = 256
	MaxTelegramChatIDLength      = 256
	MaxTransitionEventIDLength   = 256
	MaxNotificationBatchEvents   = 100
	MinNotificationLeaseDuration = time.Second
	MaxNotificationLeaseDuration = 24 * time.Hour
)

// Provider identifies a quota collector supported by the monitor.
type Provider string

const (
	ProviderClaude      Provider = "claude"
	ProviderCodex       Provider = "codex"
	ProviderGeminiCLI   Provider = "gemini-cli"
	ProviderAntigravity Provider = "antigravity"
	ProviderKimi        Provider = "kimi"
	ProviderXAI         Provider = "xai"
	ProviderKiro        Provider = "kiro"
)

var supportedProviders = []Provider{
	ProviderClaude,
	ProviderCodex,
	ProviderGeminiCLI,
	ProviderAntigravity,
	ProviderKimi,
	ProviderXAI,
	ProviderKiro,
}

// SupportedProviders returns the provider keys accepted by settings and collectors.
func SupportedProviders() []Provider {
	return slices.Clone(supportedProviders)
}

// Validate verifies that p is a supported provider key.
func (p Provider) Validate() error {
	if !slices.Contains(supportedProviders, p) {
		return fmt.Errorf("unsupported quota provider %q", p)
	}
	return nil
}

// Percentage is a normalized quota percentage in the inclusive range 0-100.
type Percentage float64

// Validate verifies that p is finite and in the inclusive range 0-100.
func (p Percentage) Validate(field string) error {
	value := float64(p)
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100", field)
	}
	return nil
}

// NormalizePercentage bounds a finite provider percentage to the domain range.
func NormalizePercentage(value float64) (Percentage, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("quota percentage must be finite")
	}
	return Percentage(min(100, max(0, value))), nil
}

// ProviderOverride controls one supported provider when a global setting is enabled.
type ProviderOverride struct {
	Provider         Provider
	Enabled          bool
	WarningThreshold *Percentage
}

// Validate verifies the provider and optional threshold override.
func (o ProviderOverride) Validate() error {
	if err := o.Provider.Validate(); err != nil {
		return err
	}
	if o.WarningThreshold != nil {
		if err := o.WarningThreshold.Validate("provider warning threshold"); err != nil {
			return err
		}
	}
	return nil
}

// TelegramDestination describes the single Telegram destination supported by the MVP.
// Bot token material is intentionally absent from this type.
type TelegramDestination struct {
	Enabled         bool
	ChatID          string
	TokenConfigured bool
}

// Validate verifies the destination state without accepting a readable bot token.
func (d TelegramDestination) Validate() error {
	if len(d.ChatID) > MaxTelegramChatIDLength {
		return fmt.Errorf("telegram chat ID must not exceed %d bytes", MaxTelegramChatIDLength)
	}
	chatID := strings.TrimSpace(d.ChatID)
	if !d.Enabled {
		return nil
	}
	if chatID == "" {
		return fmt.Errorf("telegram chat ID is required when Telegram is enabled")
	}
	if !d.TokenConfigured {
		return fmt.Errorf("telegram bot token must be configured when Telegram is enabled")
	}
	return nil
}

// Settings contains database-backed global settings and provider overrides.
type Settings struct {
	Revision          int64
	Enabled           bool
	PollInterval      time.Duration
	WarningThreshold  Percentage
	NotifyRecovery    bool
	ReminderInterval  time.Duration
	ProviderOverrides []ProviderOverride
	Telegram          TelegramDestination
}

// DefaultSettings returns the safe disabled settings seeded by persistence.
func DefaultSettings() Settings {
	return Settings{
		PollInterval:     DefaultPollInterval,
		WarningThreshold: DefaultWarningThreshold,
	}
}

// Validate verifies settings before they are persisted or activated.
func (s Settings) Validate() error {
	if s.Revision < 0 {
		return fmt.Errorf("settings revision must not be negative")
	}
	if s.PollInterval < MinPollInterval || s.PollInterval > MaxPollInterval {
		return fmt.Errorf("poll interval must be between %s and %s", MinPollInterval, MaxPollInterval)
	}
	if err := s.WarningThreshold.Validate("warning threshold"); err != nil {
		return err
	}
	if s.ReminderInterval < 0 {
		return fmt.Errorf("reminder interval must not be negative")
	}
	if s.ReminderInterval > 0 && s.ReminderInterval < s.PollInterval {
		return fmt.Errorf("reminder interval must be zero or at least the poll interval")
	}

	seen := make(map[Provider]struct{}, len(s.ProviderOverrides))
	for _, override := range s.ProviderOverrides {
		if err := override.Validate(); err != nil {
			return err
		}
		if _, exists := seen[override.Provider]; exists {
			return fmt.Errorf("duplicate provider override %q", override.Provider)
		}
		seen[override.Provider] = struct{}{}
	}
	return s.Telegram.Validate()
}

// StateIdentity is the durable identity of one quota state row.
// It deliberately excludes auth filenames and derived runtime auth indexes.
type StateIdentity struct {
	AuthID   string
	Provider Provider
	Resource string
	Window   string
}

// Normalize trims identity components and canonicalizes the provider key.
func (i StateIdentity) Normalize() (StateIdentity, error) {
	i.AuthID = strings.TrimSpace(i.AuthID)
	i.Provider = Provider(strings.ToLower(strings.TrimSpace(string(i.Provider))))
	i.Resource = strings.TrimSpace(i.Resource)
	i.Window = strings.TrimSpace(i.Window)

	if i.AuthID == "" {
		return StateIdentity{}, fmt.Errorf("persisted auth ID is required")
	}
	if len(i.AuthID) > MaxIdentityFieldLength {
		return StateIdentity{}, fmt.Errorf("persisted auth ID must not exceed %d bytes", MaxIdentityFieldLength)
	}
	if err := i.Provider.Validate(); err != nil {
		return StateIdentity{}, err
	}
	if i.Resource == "" {
		return StateIdentity{}, fmt.Errorf("quota resource is required")
	}
	if len(i.Resource) > MaxIdentityFieldLength {
		return StateIdentity{}, fmt.Errorf("quota resource must not exceed %d bytes", MaxIdentityFieldLength)
	}
	if i.Window == "" {
		return StateIdentity{}, fmt.Errorf("quota window is required")
	}
	if len(i.Window) > MaxIdentityFieldLength {
		return StateIdentity{}, fmt.Errorf("quota window must not exceed %d bytes", MaxIdentityFieldLength)
	}
	return i, nil
}

// StableKey returns an unambiguous deterministic key over all durable identity fields.
func (i StateIdentity) StableKey() (string, error) {
	normalized, err := i.Normalize()
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	for _, part := range []string{
		normalized.AuthID,
		string(normalized.Provider),
		normalized.Resource,
		normalized.Window,
	} {
		_, _ = hash.Write([]byte(strconv.Itoa(len(part))))
		_, _ = hash.Write([]byte{':'})
		_, _ = hash.Write([]byte(part))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// CollectionHealth records whether an observation is safe to evaluate.
type CollectionHealth string

const (
	CollectionReliable CollectionHealth = "reliable"
	CollectionUnknown  CollectionHealth = "unknown"
)

// Validate verifies the collection-health value.
func (h CollectionHealth) Validate() error {
	switch h {
	case CollectionReliable, CollectionUnknown:
		return nil
	default:
		return fmt.Errorf("invalid collection health %q", h)
	}
}

// AlertState is the normalized alert level persisted for an identity.
type AlertState string

const (
	AlertHealthy   AlertState = "healthy"
	AlertWarning   AlertState = "warning"
	AlertExhausted AlertState = "exhausted"
	AlertUnknown   AlertState = "unknown"
)

// Validate verifies the alert-state value.
func (s AlertState) Validate() error {
	switch s {
	case AlertHealthy, AlertWarning, AlertExhausted, AlertUnknown:
		return nil
	default:
		return fmt.Errorf("invalid alert state %q", s)
	}
}

// Observation is a provider-independent quota measurement.
type Observation struct {
	Identity            StateIdentity
	AuthLabel           string
	Health              CollectionHealth
	Remaining           Percentage
	RemainingKnown      bool
	ExplicitlyExhausted bool
	ResetAt             time.Time
	ResetKnown          bool
	ObservedAt          time.Time
}

// Normalize validates an observation and returns its canonical, bounded form.
func (o Observation) Normalize() (Observation, error) {
	identity, err := o.Identity.Normalize()
	if err != nil {
		return Observation{}, err
	}
	o.Identity = identity
	o.AuthLabel = strings.TrimSpace(o.AuthLabel)
	if o.AuthLabel == "" {
		return Observation{}, fmt.Errorf("redacted auth label is required")
	}
	if len(o.AuthLabel) > MaxAuthLabelLength {
		return Observation{}, fmt.Errorf("redacted auth label must not exceed %d bytes", MaxAuthLabelLength)
	}
	if err := o.Health.Validate(); err != nil {
		return Observation{}, err
	}
	if o.ObservedAt.IsZero() {
		return Observation{}, fmt.Errorf("observation time is required")
	}
	o.ObservedAt = o.ObservedAt.UTC().Truncate(time.Microsecond)
	if o.ResetKnown {
		if o.ResetAt.IsZero() {
			return Observation{}, fmt.Errorf("known reset time must not be zero")
		}
		o.ResetAt = o.ResetAt.UTC().Truncate(time.Microsecond)
	} else {
		o.ResetAt = time.Time{}
	}

	if o.Health == CollectionUnknown {
		if o.RemainingKnown || o.ExplicitlyExhausted {
			return Observation{}, fmt.Errorf("unknown collection cannot contain reliable quota evidence")
		}
		o.Remaining = 0
		return o, nil
	}
	if !o.RemainingKnown && !o.ExplicitlyExhausted {
		return Observation{}, fmt.Errorf("reliable collection requires remaining quota or explicit exhaustion evidence")
	}
	if o.RemainingKnown {
		o.Remaining, err = NormalizePercentage(float64(o.Remaining))
		if err != nil {
			return Observation{}, err
		}
	} else {
		o.Remaining = 0
	}
	if o.ExplicitlyExhausted && o.RemainingKnown && o.Remaining > 0 {
		return Observation{}, fmt.Errorf("explicit exhaustion conflicts with positive remaining quota")
	}
	return o, nil
}

// CurrentState is the latest persisted evaluation for one durable identity.
type CurrentState struct {
	Identity       StateIdentity
	AuthLabel      string
	Alert          AlertState
	Health         CollectionHealth
	Remaining      Percentage
	RemainingKnown bool
	ResetAt        time.Time
	ResetKnown     bool
	ObservedAt     time.Time
	TransitionedAt time.Time
	UpdatedAt      time.Time
	Revision       int64
}

// Normalize validates a current state and returns its canonical durable form.
func (s CurrentState) Normalize() (CurrentState, error) {
	identity, err := s.Identity.Normalize()
	if err != nil {
		return CurrentState{}, err
	}
	s.Identity = identity
	s.AuthLabel = strings.TrimSpace(s.AuthLabel)
	if s.AuthLabel == "" {
		return CurrentState{}, fmt.Errorf("current state auth label is required")
	}
	if len(s.AuthLabel) > MaxAuthLabelLength {
		return CurrentState{}, fmt.Errorf("current state auth label must not exceed %d bytes", MaxAuthLabelLength)
	}
	if err = s.Alert.Validate(); err != nil {
		return CurrentState{}, err
	}
	if err = s.Health.Validate(); err != nil {
		return CurrentState{}, err
	}
	if s.Revision < 0 {
		return CurrentState{}, fmt.Errorf("current state revision must not be negative")
	}
	if s.ObservedAt.IsZero() || s.TransitionedAt.IsZero() || s.UpdatedAt.IsZero() {
		return CurrentState{}, fmt.Errorf("current state timestamps are required")
	}
	s.ObservedAt = s.ObservedAt.UTC().Truncate(time.Microsecond)
	s.TransitionedAt = s.TransitionedAt.UTC().Truncate(time.Microsecond)
	s.UpdatedAt = s.UpdatedAt.UTC().Truncate(time.Microsecond)
	if s.TransitionedAt.After(s.ObservedAt) || s.ObservedAt.After(s.UpdatedAt) {
		return CurrentState{}, fmt.Errorf("current state timestamps must satisfy transitioned <= observed <= updated")
	}
	if s.Health == CollectionReliable && s.Alert != AlertExhausted && !s.RemainingKnown {
		return CurrentState{}, fmt.Errorf("reliable current state requires known remaining quota unless exhausted")
	}
	if s.RemainingKnown {
		s.Remaining, err = NormalizePercentage(float64(s.Remaining))
		if err != nil {
			return CurrentState{}, err
		}
	} else {
		s.Remaining = 0
	}
	if s.ResetKnown {
		if s.ResetAt.IsZero() {
			return CurrentState{}, fmt.Errorf("known current state reset time must not be zero")
		}
		s.ResetAt = s.ResetAt.UTC().Truncate(time.Microsecond)
	} else {
		s.ResetAt = time.Time{}
	}
	if s.Alert == AlertUnknown && s.Health != CollectionUnknown {
		return CurrentState{}, fmt.Errorf("unknown alert state requires unknown collection health")
	}
	if s.Alert == AlertUnknown && s.RemainingKnown {
		return CurrentState{}, fmt.Errorf("unknown alert state cannot contain reliable remaining quota")
	}
	if s.RemainingKnown && s.Remaining == 0 && s.Alert != AlertExhausted {
		return CurrentState{}, fmt.Errorf("zero remaining quota requires exhausted alert state")
	}
	if s.Alert == AlertExhausted && s.RemainingKnown && s.Remaining > 0 {
		return CurrentState{}, fmt.Errorf("exhausted alert state conflicts with positive remaining quota")
	}
	return s, nil
}

// TransitionKind classifies transition-driven events and optional reminders.
type TransitionKind string

const (
	TransitionWarning   TransitionKind = "warning"
	TransitionExhausted TransitionKind = "exhausted"
	TransitionRecovery  TransitionKind = "recovery"
	TransitionReminder  TransitionKind = "reminder"
)

// Validate verifies the transition kind.
func (k TransitionKind) Validate() error {
	switch k {
	case TransitionWarning, TransitionExhausted, TransitionRecovery, TransitionReminder:
		return nil
	default:
		return fmt.Errorf("invalid transition kind %q", k)
	}
}

// TransitionEvent is a durable, acknowledgement-capable in-app alert event.
type TransitionEvent struct {
	ID             string
	Identity       StateIdentity
	AuthLabel      string
	Kind           TransitionKind
	From           AlertState
	To             AlertState
	Remaining      Percentage
	RemainingKnown bool
	ResetAt        time.Time
	ResetKnown     bool
	OccurredAt     time.Time
	AcknowledgedAt time.Time
}

// Normalize validates an event and returns its canonical durable form.
func (e TransitionEvent) Normalize() (TransitionEvent, error) {
	e.ID = strings.TrimSpace(e.ID)
	if e.ID == "" {
		return TransitionEvent{}, fmt.Errorf("transition event ID is required")
	}
	if len(e.ID) > MaxTransitionEventIDLength {
		return TransitionEvent{}, fmt.Errorf("transition event ID must not exceed %d bytes", MaxTransitionEventIDLength)
	}
	identity, err := e.Identity.Normalize()
	if err != nil {
		return TransitionEvent{}, err
	}
	e.Identity = identity
	e.AuthLabel = strings.TrimSpace(e.AuthLabel)
	if e.AuthLabel == "" {
		return TransitionEvent{}, fmt.Errorf("transition event auth label is required")
	}
	if len(e.AuthLabel) > MaxAuthLabelLength {
		return TransitionEvent{}, fmt.Errorf("transition event auth label must not exceed %d bytes", MaxAuthLabelLength)
	}
	if err = e.Kind.Validate(); err != nil {
		return TransitionEvent{}, err
	}
	if err = e.From.Validate(); err != nil {
		return TransitionEvent{}, err
	}
	if err = e.To.Validate(); err != nil {
		return TransitionEvent{}, err
	}
	if !validTransition(e.Kind, e.From, e.To) {
		return TransitionEvent{}, fmt.Errorf("invalid %s transition from %s to %s", e.Kind, e.From, e.To)
	}
	if e.RemainingKnown {
		if err = e.Remaining.Validate("event remaining quota"); err != nil {
			return TransitionEvent{}, err
		}
		if e.Remaining == 0 && e.To != AlertExhausted {
			return TransitionEvent{}, fmt.Errorf("zero remaining quota requires exhausted event target")
		}
		if e.To == AlertExhausted && e.Remaining > 0 {
			return TransitionEvent{}, fmt.Errorf("exhausted event target conflicts with positive remaining quota")
		}
	} else {
		e.Remaining = 0
	}
	if e.ResetKnown {
		if e.ResetAt.IsZero() {
			return TransitionEvent{}, fmt.Errorf("known event reset time must not be zero")
		}
		e.ResetAt = e.ResetAt.UTC().Truncate(time.Microsecond)
	} else {
		e.ResetAt = time.Time{}
	}
	if e.OccurredAt.IsZero() {
		return TransitionEvent{}, fmt.Errorf("transition event time is required")
	}
	e.OccurredAt = e.OccurredAt.UTC().Truncate(time.Microsecond)
	if !e.AcknowledgedAt.IsZero() {
		e.AcknowledgedAt = e.AcknowledgedAt.UTC().Truncate(time.Microsecond)
	}
	return e, nil
}

func validTransition(kind TransitionKind, from, to AlertState) bool {
	switch kind {
	case TransitionWarning:
		return to == AlertWarning && (from == AlertHealthy || from == AlertUnknown)
	case TransitionExhausted:
		return to == AlertExhausted && (from == AlertHealthy || from == AlertWarning || from == AlertUnknown)
	case TransitionRecovery:
		return to == AlertHealthy && (from == AlertWarning || from == AlertExhausted)
	case TransitionReminder:
		return from == to && (to == AlertWarning || to == AlertExhausted)
	default:
		return false
	}
}

// NotificationBatch is an immutable provider-grouped delivery payload.
type NotificationBatch struct {
	id        string
	provider  Provider
	events    []TransitionEvent
	createdAt time.Time
}

// NewNotificationBatch validates, orders, copies, and identifies provider events.
func NewNotificationBatch(provider Provider, events []TransitionEvent, createdAt time.Time) (NotificationBatch, error) {
	if err := provider.Validate(); err != nil {
		return NotificationBatch{}, err
	}
	if len(events) == 0 {
		return NotificationBatch{}, fmt.Errorf("notification batch requires at least one event")
	}
	if len(events) > MaxNotificationBatchEvents {
		return NotificationBatch{}, fmt.Errorf("notification batch must not exceed %d events", MaxNotificationBatchEvents)
	}
	if createdAt.IsZero() {
		return NotificationBatch{}, fmt.Errorf("notification batch creation time is required")
	}

	canonical := make([]TransitionEvent, len(events))
	seen := make(map[string]struct{}, len(canonical))
	for index, event := range events {
		normalized, err := event.Normalize()
		if err != nil {
			return NotificationBatch{}, err
		}
		if normalized.Identity.Provider != provider {
			return NotificationBatch{}, fmt.Errorf("notification event provider %q does not match batch provider %q", normalized.Identity.Provider, provider)
		}
		if _, exists := seen[normalized.ID]; exists {
			return NotificationBatch{}, fmt.Errorf("duplicate transition event %q", normalized.ID)
		}
		normalized.AcknowledgedAt = time.Time{}
		seen[normalized.ID] = struct{}{}
		canonical[index] = normalized
	}
	sort.Slice(canonical, func(left, right int) bool {
		leftKey, _ := canonical[left].Identity.StableKey()
		rightKey, _ := canonical[right].Identity.StableKey()
		if leftKey == rightKey {
			return canonical[left].ID < canonical[right].ID
		}
		return leftKey < rightKey
	})

	hash := sha256.New()
	writeBatchHashField(hash, string(provider))
	for _, event := range canonical {
		for _, field := range []string{
			event.ID,
			event.Identity.AuthID,
			string(event.Identity.Provider),
			event.Identity.Resource,
			event.Identity.Window,
			event.AuthLabel,
			string(event.Kind),
			string(event.From),
			string(event.To),
			strconv.FormatBool(event.RemainingKnown),
			strconv.FormatFloat(float64(event.Remaining), 'g', -1, 64),
			strconv.FormatBool(event.ResetKnown),
			event.ResetAt.Format(time.RFC3339Nano),
			event.OccurredAt.Format(time.RFC3339Nano),
		} {
			writeBatchHashField(hash, field)
		}
	}
	return NotificationBatch{
		id:        hex.EncodeToString(hash.Sum(nil)),
		provider:  provider,
		events:    canonical,
		createdAt: createdAt.UTC().Truncate(time.Microsecond),
	}, nil
}

func writeBatchHashField(hash interface{ Write([]byte) (int, error) }, field string) {
	_, _ = hash.Write([]byte(strconv.Itoa(len(field))))
	_, _ = hash.Write([]byte{':'})
	_, _ = hash.Write([]byte(field))
}

func (b NotificationBatch) ID() string                { return b.id }
func (b NotificationBatch) Provider() Provider        { return b.provider }
func (b NotificationBatch) CreatedAt() time.Time      { return b.createdAt }
func (b NotificationBatch) Events() []TransitionEvent { return slices.Clone(b.events) }

// PageRequest is an opaque-cursor pagination request.
type PageRequest struct {
	Cursor string
	Limit  int
}

// Normalize trims the cursor and applies the bounded default page size.
func (r PageRequest) Normalize() (PageRequest, error) {
	r.Cursor = strings.TrimSpace(r.Cursor)
	if r.Limit == 0 {
		r.Limit = DefaultPageSize
	}
	if r.Limit < 1 || r.Limit > MaxPageSize {
		return PageRequest{}, fmt.Errorf("page limit must be between 1 and %d", MaxPageSize)
	}
	return r, nil
}

// Page contains one bounded result page and an opaque continuation cursor.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// AuthSnapshot is the read-only, runtime-only credential view supplied to collectors.
// Implementations must provide a cloned snapshot; secret-bearing values must not be logged or persisted.
type AuthSnapshot interface {
	AuthID() string
	Provider() Provider
	RedactedLabel() string
	ProxyURL() string
	Attribute(key string) (string, bool)
	Metadata(key string) (any, bool)
}

// Collector obtains normalized observations for one persisted auth snapshot.
type Collector interface {
	Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error)
}

// Sender delivers one immutable provider batch.
type Sender interface {
	Send(ctx context.Context, batch NotificationBatch) error
}

// CollectionLease owns one database-backed polling cycle.
type CollectionLease interface {
	Release(ctx context.Context) error
}

// CollectionCommit is atomically persisted after evaluation.
type CollectionCommit struct {
	SettingsRevision int64
	States           []CurrentState
	RemovedStates    []StateIdentity
	Events           []TransitionEvent
	Batches          []NotificationBatch
}

// NotificationClaim contains one leased immutable batch.
type NotificationClaim struct {
	Batch      NotificationBatch
	LeaseID    string
	Attempt    int
	LeaseUntil time.Time
}

// NotificationClaimOptions bounds a durable outbox claim.
type NotificationClaimOptions struct {
	Limit         int
	LeaseDuration time.Duration
}

// NotificationResult resolves one claimed delivery attempt.
type NotificationResult struct {
	BatchID          string
	LeaseID          string
	SentAt           time.Time
	RetryAt          time.Time
	FailureCode      string
	PermanentFailure bool
}

// Store is the database-only quota-alert persistence boundary.
type Store interface {
	LoadSettings(ctx context.Context) (Settings, error)
	LoadSettingsWithSecret(ctx context.Context) (Settings, *EncryptedSecret, error)
	SaveSettings(ctx context.Context, expectedRevision int64, settings Settings) (Settings, error)
	SaveSettingsWithSecret(ctx context.Context, expectedRevision int64, settings Settings, update SecretUpdate, cipher *SecretCipher, purpose string) (Settings, error)
	TryAcquireCollection(ctx context.Context) (CollectionLease, bool, error)
	LoadStates(ctx context.Context, identities []StateIdentity) ([]CurrentState, error)
	CommitCollection(ctx context.Context, lease CollectionLease, commit CollectionCommit) error
	ListStates(ctx context.Context, page PageRequest) (Page[CurrentState], error)
	ListEvents(ctx context.Context, page PageRequest) (Page[TransitionEvent], error)
	AcknowledgeEvent(ctx context.Context, eventID string, acknowledgedAt time.Time) error
	PruneEvents(ctx context.Context, before time.Time, limit int) (int64, error)
	PruneNotificationBatches(ctx context.Context, before time.Time, limit int) (int64, error)
	ClaimNotificationBatches(ctx context.Context, options NotificationClaimOptions) ([]NotificationClaim, error)
	ResolveNotification(ctx context.Context, result NotificationResult) error
}
