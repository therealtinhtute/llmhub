package quotaalert

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func TestValidateSettings(t *testing.T) {
	valid := DefaultSettings()
	valid.Enabled = true
	if err := valid.Validate(); err != nil {
		t.Fatalf("DefaultSettings().Validate() error = %v", err)
	}

	thresholdZero := Percentage(0)
	thresholdHundred := Percentage(100)
	for _, test := range []struct {
		name   string
		mutate func(*Settings)
		valid  bool
	}{
		{
			name: "minimum poll interval",
			mutate: func(settings *Settings) {
				settings.PollInterval = MinPollInterval
			},
			valid: true,
		},
		{
			name: "maximum poll interval",
			mutate: func(settings *Settings) {
				settings.PollInterval = MaxPollInterval
			},
			valid: true,
		},
		{
			name: "poll interval below minimum",
			mutate: func(settings *Settings) {
				settings.PollInterval = MinPollInterval - time.Nanosecond
			},
		},
		{
			name: "poll interval above maximum",
			mutate: func(settings *Settings) {
				settings.PollInterval = MaxPollInterval + time.Nanosecond
			},
		},
		{
			name: "zero warning threshold",
			mutate: func(settings *Settings) {
				settings.WarningThreshold = thresholdZero
			},
			valid: true,
		},
		{
			name: "hundred warning threshold",
			mutate: func(settings *Settings) {
				settings.WarningThreshold = thresholdHundred
			},
			valid: true,
		},
		{
			name: "warning threshold below zero",
			mutate: func(settings *Settings) {
				settings.WarningThreshold = -0.1
			},
		},
		{
			name: "warning threshold above hundred",
			mutate: func(settings *Settings) {
				settings.WarningThreshold = 100.1
			},
		},
		{
			name: "warning threshold not finite",
			mutate: func(settings *Settings) {
				settings.WarningThreshold = Percentage(math.NaN())
			},
		},
		{
			name: "reminders disabled",
			mutate: func(settings *Settings) {
				settings.ReminderInterval = 0
			},
			valid: true,
		},
		{
			name: "reminder at poll interval",
			mutate: func(settings *Settings) {
				settings.ReminderInterval = settings.PollInterval
			},
			valid: true,
		},
		{
			name: "negative reminder",
			mutate: func(settings *Settings) {
				settings.ReminderInterval = -time.Second
			},
		},
		{
			name: "reminder below poll interval",
			mutate: func(settings *Settings) {
				settings.ReminderInterval = settings.PollInterval - time.Second
			},
		},
		{
			name: "supported provider override",
			mutate: func(settings *Settings) {
				settings.ProviderOverrides = []ProviderOverride{{
					Provider:         ProviderClaude,
					Enabled:          true,
					WarningThreshold: &thresholdZero,
				}}
			},
			valid: true,
		},
		{
			name: "unsupported provider override",
			mutate: func(settings *Settings) {
				settings.ProviderOverrides = []ProviderOverride{{Provider: "other", Enabled: true}}
			},
		},
		{
			name: "duplicate provider override",
			mutate: func(settings *Settings) {
				settings.ProviderOverrides = []ProviderOverride{
					{Provider: ProviderCodex, Enabled: true},
					{Provider: ProviderCodex, Enabled: false},
				}
			},
		},
		{
			name: "invalid provider threshold",
			mutate: func(settings *Settings) {
				settings.ProviderOverrides = []ProviderOverride{{
					Provider:         ProviderKiro,
					Enabled:          true,
					WarningThreshold: func() *Percentage { value := Percentage(101); return &value }(),
				}}
			},
		},
		{
			name: "telegram disabled without destination",
			mutate: func(settings *Settings) {
				settings.Telegram = TelegramDestination{}
			},
			valid: true,
		},
		{
			name: "one telegram destination",
			mutate: func(settings *Settings) {
				settings.Telegram = TelegramDestination{
					Enabled:         true,
					ChatID:          "-1001234567890",
					TokenConfigured: true,
				}
			},
			valid: true,
		},
		{
			name: "telegram chat ID over limit",
			mutate: func(settings *Settings) {
				settings.Telegram = TelegramDestination{ChatID: strings.Repeat("1", MaxTelegramChatIDLength+1)}
			},
		},
		{
			name: "telegram enabled without chat",
			mutate: func(settings *Settings) {
				settings.Telegram = TelegramDestination{Enabled: true, TokenConfigured: true}
			},
		},
		{
			name: "telegram enabled without token",
			mutate: func(settings *Settings) {
				settings.Telegram = TelegramDestination{Enabled: true, ChatID: "123"}
			},
		},
		{
			name: "negative revision",
			mutate: func(settings *Settings) {
				settings.Revision = -1
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			settings := valid
			settings.ProviderOverrides = nil
			settings.Telegram = TelegramDestination{}
			settings.ReminderInterval = 0
			test.mutate(&settings)
			err := settings.Validate()
			if (err == nil) != test.valid {
				t.Fatalf("Settings.Validate() error = %v, want valid = %t", err, test.valid)
			}
		})
	}
}

func TestValidateSupportedProviders(t *testing.T) {
	providers := SupportedProviders()
	if len(providers) != 7 {
		t.Fatalf("len(SupportedProviders()) = %d, want 7", len(providers))
	}
	for _, provider := range providers {
		if err := provider.Validate(); err != nil {
			t.Fatalf("Provider(%q).Validate() error = %v", provider, err)
		}
	}
	providers[0] = "mutated"
	if SupportedProviders()[0] != ProviderClaude {
		t.Fatal("SupportedProviders() returned mutable package state")
	}
}

func TestNormalizePercentage(t *testing.T) {
	for _, test := range []struct {
		name  string
		input float64
		want  Percentage
		valid bool
	}{
		{name: "below range", input: -20, want: 0, valid: true},
		{name: "within range", input: 12.5, want: 12.5, valid: true},
		{name: "above range", input: 120, want: 100, valid: true},
		{name: "nan", input: math.NaN()},
		{name: "positive infinity", input: math.Inf(1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizePercentage(test.input)
			if (err == nil) != test.valid {
				t.Fatalf("NormalizePercentage() error = %v, want valid = %t", err, test.valid)
			}
			if err == nil && got != test.want {
				t.Fatalf("NormalizePercentage() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNormalizeObservation(t *testing.T) {
	observedAt := time.Date(2026, time.July, 28, 11, 30, 0, 0, time.FixedZone("ICT", 7*60*60))
	resetAt := observedAt.Add(time.Hour)
	base := Observation{
		Identity: StateIdentity{
			AuthID:   " auth-123 ",
			Provider: " CLAUDE ",
			Resource: " messages ",
			Window:   " five-hour ",
		},
		AuthLabel:      " Primary account ",
		Health:         CollectionReliable,
		Remaining:      120,
		RemainingKnown: true,
		ResetAt:        resetAt,
		ResetKnown:     true,
		ObservedAt:     observedAt,
	}

	normalized, err := base.Normalize()
	if err != nil {
		t.Fatalf("Observation.Normalize() error = %v", err)
	}
	if normalized.Identity != (StateIdentity{AuthID: "auth-123", Provider: ProviderClaude, Resource: "messages", Window: "five-hour"}) {
		t.Fatalf("normalized identity = %#v", normalized.Identity)
	}
	if normalized.AuthLabel != "Primary account" || normalized.Remaining != 100 {
		t.Fatalf("normalized observation = %#v", normalized)
	}
	if normalized.ObservedAt.Location() != time.UTC || normalized.ResetAt.Location() != time.UTC {
		t.Fatalf("normalized times are not UTC: observed=%v reset=%v", normalized.ObservedAt, normalized.ResetAt)
	}

	for _, test := range []struct {
		name   string
		mutate func(*Observation)
		valid  bool
	}{
		{
			name: "explicit exhaustion without percentage",
			mutate: func(observation *Observation) {
				observation.RemainingKnown = false
				observation.ExplicitlyExhausted = true
			},
			valid: true,
		},
		{
			name: "unknown reset-only observation",
			mutate: func(observation *Observation) {
				observation.Health = CollectionUnknown
				observation.RemainingKnown = false
			},
			valid: true,
		},
		{
			name: "unknown with remaining quota",
			mutate: func(observation *Observation) {
				observation.Health = CollectionUnknown
			},
		},
		{
			name: "unknown with exhaustion evidence",
			mutate: func(observation *Observation) {
				observation.Health = CollectionUnknown
				observation.RemainingKnown = false
				observation.ExplicitlyExhausted = true
			},
		},
		{
			name: "reliable without evidence",
			mutate: func(observation *Observation) {
				observation.RemainingKnown = false
			},
		},
		{
			name: "conflicting exhaustion evidence",
			mutate: func(observation *Observation) {
				observation.Remaining = 1
				observation.ExplicitlyExhausted = true
			},
		},
		{
			name: "missing redacted label",
			mutate: func(observation *Observation) {
				observation.AuthLabel = " "
			},
		},
		{
			name: "missing observation time",
			mutate: func(observation *Observation) {
				observation.ObservedAt = time.Time{}
			},
		},
		{
			name: "non-finite remaining quota",
			mutate: func(observation *Observation) {
				observation.Remaining = Percentage(math.NaN())
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			observation := base
			test.mutate(&observation)
			_, err := observation.Normalize()
			if (err == nil) != test.valid {
				t.Fatalf("Observation.Normalize() error = %v, want valid = %t", err, test.valid)
			}
		})
	}
}

func TestNormalizeCurrentState(t *testing.T) {
	zone := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, time.July, 28, 15, 0, 0, 123, zone)
	base := CurrentState{
		Identity: StateIdentity{
			AuthID:   " auth-1 ",
			Provider: " CLAUDE ",
			Resource: " messages ",
			Window:   " five-hour ",
		},
		AuthLabel:      " Primary ",
		Alert:          AlertWarning,
		Health:         CollectionReliable,
		Remaining:      5,
		RemainingKnown: true,
		ResetAt:        now.Add(time.Hour),
		ResetKnown:     true,
		ObservedAt:     now,
		TransitionedAt: now,
		UpdatedAt:      now,
	}
	normalized, err := base.Normalize()
	if err != nil {
		t.Fatalf("CurrentState.Normalize() error = %v", err)
	}
	if normalized.Identity != (StateIdentity{AuthID: "auth-1", Provider: ProviderClaude, Resource: "messages", Window: "five-hour"}) || normalized.AuthLabel != "Primary" {
		t.Fatalf("normalized current state identity = %#v, label %q", normalized.Identity, normalized.AuthLabel)
	}
	if normalized.ObservedAt.Location() != time.UTC || normalized.TransitionedAt.Location() != time.UTC || normalized.UpdatedAt.Location() != time.UTC || normalized.ResetAt.Location() != time.UTC {
		t.Fatalf("normalized current state times are not UTC: %#v", normalized)
	}
	if normalized.ObservedAt.Nanosecond()%1000 != 0 || normalized.TransitionedAt.Nanosecond()%1000 != 0 || normalized.UpdatedAt.Nanosecond()%1000 != 0 || normalized.ResetAt.Nanosecond()%1000 != 0 {
		t.Fatalf("normalized current state times exceed PostgreSQL precision: %#v", normalized)
	}

	for _, test := range []struct {
		name   string
		mutate func(*CurrentState)
	}{
		{name: "oversized label", mutate: func(state *CurrentState) { state.AuthLabel = strings.Repeat("a", MaxAuthLabelLength+1) }},
		{name: "unknown alert with reliable health", mutate: func(state *CurrentState) {
			state.Alert = AlertUnknown
		}},
		{name: "unknown alert with remaining quota", mutate: func(state *CurrentState) {
			state.Alert = AlertUnknown
			state.Health = CollectionUnknown
		}},
		{name: "reliable warning without remaining evidence", mutate: func(state *CurrentState) {
			state.RemainingKnown = false
		}},
		{name: "zero remaining without exhaustion", mutate: func(state *CurrentState) { state.Remaining = 0 }},
		{name: "exhausted with positive remaining", mutate: func(state *CurrentState) { state.Alert = AlertExhausted }},
		{name: "transition after observation", mutate: func(state *CurrentState) {
			state.TransitionedAt = state.ObservedAt.Add(time.Second)
		}},
		{name: "observation after update", mutate: func(state *CurrentState) {
			state.UpdatedAt = state.ObservedAt.Add(-time.Second)
		}},
		{name: "negative revision", mutate: func(state *CurrentState) { state.Revision = -1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := base
			test.mutate(&state)
			if _, err := state.Normalize(); err == nil {
				t.Fatal("CurrentState.Normalize() error = nil")
			}
		})
	}
}

func TestNormalizePageRequest(t *testing.T) {
	request, err := (PageRequest{Cursor: " next ", Limit: 0}).Normalize()
	if err != nil {
		t.Fatalf("PageRequest.Normalize() error = %v", err)
	}
	if request.Cursor != "next" || request.Limit != DefaultPageSize {
		t.Fatalf("PageRequest.Normalize() = %#v", request)
	}
	for _, limit := range []int{-1, MaxPageSize + 1} {
		if _, err := (PageRequest{Limit: limit}).Normalize(); err == nil {
			t.Fatalf("PageRequest{Limit: %d}.Normalize() error = nil", limit)
		}
	}
}

func TestIdentityStable(t *testing.T) {
	identity := StateIdentity{
		AuthID:   " auth-123 ",
		Provider: " CLAUDE ",
		Resource: "messages",
		Window:   "five-hour",
	}
	got, err := identity.StableKey()
	if err != nil {
		t.Fatalf("StateIdentity.StableKey() error = %v", err)
	}
	canonical, err := (StateIdentity{
		AuthID:   "auth-123",
		Provider: ProviderClaude,
		Resource: "messages",
		Window:   "five-hour",
	}).StableKey()
	if err != nil {
		t.Fatalf("canonical StateIdentity.StableKey() error = %v", err)
	}
	if got != canonical {
		t.Fatalf("normalized stable key = %q, want %q", got, canonical)
	}

	for _, changed := range []StateIdentity{
		{AuthID: "auth-456", Provider: ProviderClaude, Resource: "messages", Window: "five-hour"},
		{AuthID: "auth-123", Provider: ProviderCodex, Resource: "messages", Window: "five-hour"},
		{AuthID: "auth-123", Provider: ProviderClaude, Resource: "tokens", Window: "five-hour"},
		{AuthID: "auth-123", Provider: ProviderClaude, Resource: "messages", Window: "weekly"},
	} {
		key, err := changed.StableKey()
		if err != nil {
			t.Fatalf("changed StateIdentity.StableKey() error = %v", err)
		}
		if key == canonical {
			t.Fatalf("identity change did not change stable key: %#v", changed)
		}
	}
}

func TestIdentityRejectsMissingDurableComponents(t *testing.T) {
	valid := StateIdentity{AuthID: "auth-123", Provider: ProviderClaude, Resource: "messages", Window: "five-hour"}
	for _, mutate := range []func(*StateIdentity){
		func(identity *StateIdentity) { identity.AuthID = "" },
		func(identity *StateIdentity) { identity.Provider = "" },
		func(identity *StateIdentity) { identity.Resource = "" },
		func(identity *StateIdentity) { identity.Window = "" },
	} {
		identity := valid
		mutate(&identity)
		if _, err := identity.StableKey(); err == nil {
			t.Fatalf("StateIdentity.StableKey() error = nil for %#v", identity)
		}
	}
}

func TestValidateNotificationBatchIsImmutableAndProviderGrouped(t *testing.T) {
	now := time.Date(2026, time.July, 28, 4, 0, 0, 0, time.UTC)
	events := []TransitionEvent{
		{
			ID: "event-2",
			Identity: StateIdentity{
				AuthID:   "auth-2",
				Provider: ProviderClaude,
				Resource: "messages",
				Window:   "weekly",
			},
			AuthLabel:  "Secondary",
			Kind:       TransitionExhausted,
			From:       AlertWarning,
			To:         AlertExhausted,
			OccurredAt: now,
		},
		{
			ID: "event-1",
			Identity: StateIdentity{
				AuthID:   "auth-1",
				Provider: ProviderClaude,
				Resource: "messages",
				Window:   "five-hour",
			},
			AuthLabel:      "Primary",
			Kind:           TransitionWarning,
			From:           AlertHealthy,
			To:             AlertWarning,
			Remaining:      5,
			RemainingKnown: true,
			OccurredAt:     now,
		},
	}

	batch, err := NewNotificationBatch(ProviderClaude, events, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	reversed, err := NewNotificationBatch(ProviderClaude, []TransitionEvent{events[1], events[0]}, now)
	if err != nil {
		t.Fatalf("reversed NewNotificationBatch() error = %v", err)
	}
	if batch.ID() == "" || batch.ID() != reversed.ID() {
		t.Fatalf("notification batch IDs are not deterministic: %q and %q", batch.ID(), reversed.ID())
	}
	if batch.Provider() != ProviderClaude || !batch.CreatedAt().Equal(now) {
		t.Fatalf("notification batch metadata = provider %q, created %v", batch.Provider(), batch.CreatedAt())
	}

	events[0].AuthLabel = "mutated input"
	copyOne := batch.Events()
	if copyOne[0].AuthLabel == "mutated input" || copyOne[1].AuthLabel == "mutated input" {
		t.Fatal("notification batch retained caller-owned event storage")
	}
	copyOne[0].AuthLabel = "mutated output"
	if batch.Events()[0].AuthLabel == "mutated output" {
		t.Fatal("NotificationBatch.Events() exposed mutable batch storage")
	}

	foreign := events[0]
	foreign.ID = "event-foreign"
	foreign.Identity.Provider = ProviderCodex
	if _, err := NewNotificationBatch(ProviderClaude, []TransitionEvent{foreign}, now); err == nil {
		t.Fatal("NewNotificationBatch() error = nil for mixed providers")
	}
	if _, err := NewNotificationBatch(ProviderClaude, nil, now); err == nil {
		t.Fatal("NewNotificationBatch() error = nil for empty events")
	}
}

func TestNormalizeObservationZerosIgnoredReliableRemaining(t *testing.T) {
	observation := Observation{
		Identity: StateIdentity{
			AuthID:   "auth-123",
			Provider: ProviderClaude,
			Resource: "messages",
			Window:   "five-hour",
		},
		AuthLabel:           "Primary",
		Health:              CollectionReliable,
		Remaining:           87,
		ExplicitlyExhausted: true,
		ObservedAt:          time.Now(),
	}

	normalized, err := observation.Normalize()
	if err != nil {
		t.Fatalf("Observation.Normalize() error = %v", err)
	}
	if normalized.Remaining != 0 {
		t.Fatalf("normalized ignored remaining = %v, want 0", normalized.Remaining)
	}
}

func TestNormalizeTransitionEventMatrix(t *testing.T) {
	type transition struct {
		kind TransitionKind
		from AlertState
		to   AlertState
	}
	allowed := map[transition]bool{
		{TransitionWarning, AlertHealthy, AlertWarning}:      true,
		{TransitionWarning, AlertUnknown, AlertWarning}:      true,
		{TransitionExhausted, AlertHealthy, AlertExhausted}:  true,
		{TransitionExhausted, AlertWarning, AlertExhausted}:  true,
		{TransitionExhausted, AlertUnknown, AlertExhausted}:  true,
		{TransitionRecovery, AlertWarning, AlertHealthy}:     true,
		{TransitionRecovery, AlertExhausted, AlertHealthy}:   true,
		{TransitionReminder, AlertWarning, AlertWarning}:     true,
		{TransitionReminder, AlertExhausted, AlertExhausted}: true,
	}
	states := []AlertState{AlertHealthy, AlertWarning, AlertExhausted, AlertUnknown}
	kinds := []TransitionKind{TransitionWarning, TransitionExhausted, TransitionRecovery, TransitionReminder}
	now := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)

	for _, kind := range kinds {
		for _, from := range states {
			for _, to := range states {
				candidate := transition{kind: kind, from: from, to: to}
				t.Run(fmt.Sprintf("%s_%s_%s", kind, from, to), func(t *testing.T) {
					event := testTransitionEvent(now)
					event.Kind = kind
					event.From = from
					event.To = to
					_, err := event.Normalize()
					if (err == nil) != allowed[candidate] {
						t.Fatalf("TransitionEvent.Normalize() error = %v, want valid = %t", err, allowed[candidate])
					}
				})
			}
		}
	}
}

func TestNormalizeTransitionEventRemainingConsistency(t *testing.T) {
	now := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)
	warningAtZero := testTransitionEvent(now)
	warningAtZero.RemainingKnown = true
	if _, err := warningAtZero.Normalize(); err == nil {
		t.Fatal("TransitionEvent.Normalize() error = nil for non-exhausted event at zero remaining")
	}

	exhaustedWithRemaining := testTransitionEvent(now)
	exhaustedWithRemaining.Kind = TransitionExhausted
	exhaustedWithRemaining.From = AlertWarning
	exhaustedWithRemaining.To = AlertExhausted
	exhaustedWithRemaining.Remaining = 5
	exhaustedWithRemaining.RemainingKnown = true
	if _, err := exhaustedWithRemaining.Normalize(); err == nil {
		t.Fatal("TransitionEvent.Normalize() error = nil for exhausted event with positive remaining")
	}

	exhaustedWithoutPercentage := exhaustedWithRemaining
	exhaustedWithoutPercentage.Remaining = 0
	exhaustedWithoutPercentage.RemainingKnown = false
	if _, err := exhaustedWithoutPercentage.Normalize(); err != nil {
		t.Fatalf("TransitionEvent.Normalize() explicit exhaustion error = %v", err)
	}
}

func TestNormalizeTransitionEventCanonicalizesFields(t *testing.T) {
	zone := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, time.July, 28, 15, 0, 0, 123, zone)
	event := testTransitionEvent(now)
	event.ID = " event-1 "
	event.Identity = StateIdentity{
		AuthID:   " auth-1 ",
		Provider: " CLAUDE ",
		Resource: " messages ",
		Window:   " five-hour ",
	}
	event.AuthLabel = " Primary "
	event.From = AlertUnknown
	event.Remaining = 75
	event.RemainingKnown = false
	event.ResetAt = now.Add(time.Hour)
	event.ResetKnown = false
	event.AcknowledgedAt = now.Add(time.Minute)

	normalized, err := event.Normalize()
	if err != nil {
		t.Fatalf("TransitionEvent.Normalize() error = %v", err)
	}
	if normalized.ID != "event-1" || normalized.AuthLabel != "Primary" {
		t.Fatalf("normalized event strings = ID %q, label %q", normalized.ID, normalized.AuthLabel)
	}
	if normalized.Identity != (StateIdentity{AuthID: "auth-1", Provider: ProviderClaude, Resource: "messages", Window: "five-hour"}) {
		t.Fatalf("normalized event identity = %#v", normalized.Identity)
	}
	if normalized.Remaining != 0 || !normalized.ResetAt.IsZero() {
		t.Fatalf("normalized ignored fields = remaining %v, reset %v", normalized.Remaining, normalized.ResetAt)
	}
	if normalized.OccurredAt.Location() != time.UTC || normalized.AcknowledgedAt.Location() != time.UTC {
		t.Fatalf("normalized event times are not UTC: occurred=%v acknowledged=%v", normalized.OccurredAt, normalized.AcknowledgedAt)
	}
	if normalized.OccurredAt.Nanosecond()%1000 != 0 || normalized.AcknowledgedAt.Nanosecond()%1000 != 0 {
		t.Fatalf("normalized event times exceed PostgreSQL precision: occurred=%v acknowledged=%v", normalized.OccurredAt, normalized.AcknowledgedAt)
	}

	batch, err := NewNotificationBatch(ProviderClaude, []TransitionEvent{event}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	batchEvent := normalized
	batchEvent.AcknowledgedAt = time.Time{}
	if got := batch.Events()[0]; got != batchEvent {
		t.Fatalf("batch event = %#v, want immutable canonical %#v", got, batchEvent)
	}
	if !batch.CreatedAt().Equal(now.UTC().Truncate(time.Microsecond)) {
		t.Fatalf("batch creation time = %v, want PostgreSQL precision", batch.CreatedAt())
	}
}

func TestDomainStringBounds(t *testing.T) {
	identity := StateIdentity{AuthID: "auth", Provider: ProviderClaude, Resource: "messages", Window: "five-hour"}
	for _, test := range []struct {
		name string
		set  func(*StateIdentity, string)
	}{
		{name: "auth ID", set: func(identity *StateIdentity, value string) { identity.AuthID = value }},
		{name: "resource", set: func(identity *StateIdentity, value string) { identity.Resource = value }},
		{name: "window", set: func(identity *StateIdentity, value string) { identity.Window = value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			atLimit := identity
			test.set(&atLimit, strings.Repeat("a", MaxIdentityFieldLength))
			if _, err := atLimit.Normalize(); err != nil {
				t.Fatalf("StateIdentity.Normalize() at limit error = %v", err)
			}
			overLimit := identity
			test.set(&overLimit, strings.Repeat("a", MaxIdentityFieldLength+1))
			if _, err := overLimit.Normalize(); err == nil {
				t.Fatal("StateIdentity.Normalize() over limit error = nil")
			}
		})
	}

	now := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)
	observation := Observation{
		Identity:       identity,
		AuthLabel:      strings.Repeat("a", MaxAuthLabelLength),
		Health:         CollectionReliable,
		Remaining:      10,
		RemainingKnown: true,
		ObservedAt:     now,
	}
	if _, err := observation.Normalize(); err != nil {
		t.Fatalf("Observation.Normalize() label at limit error = %v", err)
	}
	observation.AuthLabel += "a"
	if _, err := observation.Normalize(); err == nil {
		t.Fatal("Observation.Normalize() label over limit error = nil")
	}

	event := testTransitionEvent(now)
	event.ID = strings.Repeat("a", MaxTransitionEventIDLength)
	event.AuthLabel = strings.Repeat("a", MaxAuthLabelLength)
	if _, err := event.Normalize(); err != nil {
		t.Fatalf("TransitionEvent.Normalize() at limits error = %v", err)
	}
	event.ID += "a"
	if _, err := event.Normalize(); err == nil {
		t.Fatal("TransitionEvent.Normalize() ID over limit error = nil")
	}
	event = testTransitionEvent(now)
	event.AuthLabel = strings.Repeat("a", MaxAuthLabelLength+1)
	if _, err := event.Normalize(); err == nil {
		t.Fatal("TransitionEvent.Normalize() label over limit error = nil")
	}
}

func TestNotificationBatchEventCountBound(t *testing.T) {
	now := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)
	events := make([]TransitionEvent, MaxNotificationBatchEvents+1)
	for index := range events {
		events[index] = testTransitionEvent(now)
		events[index].ID = fmt.Sprintf("event-%d", index)
	}
	if _, err := NewNotificationBatch(ProviderClaude, events[:MaxNotificationBatchEvents], now); err != nil {
		t.Fatalf("NewNotificationBatch() at limit error = %v", err)
	}
	if _, err := NewNotificationBatch(ProviderClaude, events, now); err == nil {
		t.Fatal("NewNotificationBatch() over limit error = nil")
	}
}

func TestNotificationBatchIDCommitsToCanonicalContent(t *testing.T) {
	now := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)
	base := testTransitionEvent(now)
	base.Remaining = 5
	base.RemainingKnown = true
	base.ResetAt = now.Add(time.Hour)
	base.ResetKnown = true

	baseBatch, err := NewNotificationBatch(ProviderClaude, []TransitionEvent{base}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch(base) error = %v", err)
	}
	for _, test := range []struct {
		name     string
		provider Provider
		mutate   func(*TransitionEvent)
	}{
		{name: "provider", provider: ProviderCodex, mutate: func(event *TransitionEvent) { event.Identity.Provider = ProviderCodex }},
		{name: "auth ID", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.Identity.AuthID = "auth-2" }},
		{name: "resource", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.Identity.Resource = "tokens" }},
		{name: "window", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.Identity.Window = "weekly" }},
		{name: "auth label", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.AuthLabel = "Secondary" }},
		{name: "transition", provider: ProviderClaude, mutate: func(event *TransitionEvent) {
			event.Kind = TransitionRecovery
			event.From = AlertExhausted
			event.To = AlertHealthy
		}},
		{name: "from state", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.From = AlertUnknown }},
		{name: "remaining", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.Remaining = 4 }},
		{name: "remaining known", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.RemainingKnown = false }},
		{name: "reset", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.ResetAt = event.ResetAt.Add(time.Minute) }},
		{name: "reset known", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.ResetKnown = false }},
		{name: "occurred", provider: ProviderClaude, mutate: func(event *TransitionEvent) { event.OccurredAt = event.OccurredAt.Add(time.Second) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.mutate(&changed)
			batch, err := NewNotificationBatch(test.provider, []TransitionEvent{changed}, now)
			if err != nil {
				t.Fatalf("NewNotificationBatch() error = %v", err)
			}
			if batch.ID() == baseBatch.ID() {
				t.Fatalf("batch ID did not change for %s", test.name)
			}
		})
	}

	acknowledged := base
	acknowledged.AcknowledgedAt = now
	acknowledgedBatch, err := NewNotificationBatch(ProviderClaude, []TransitionEvent{acknowledged}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch(acknowledged) error = %v", err)
	}
	if acknowledgedBatch.ID() != baseBatch.ID() || !acknowledgedBatch.Events()[0].AcknowledgedAt.IsZero() {
		t.Fatalf("acknowledgement changed delivery batch identity or payload: %#v", acknowledgedBatch.Events()[0])
	}
}

func testTransitionEvent(now time.Time) TransitionEvent {
	return TransitionEvent{
		ID: "event-1",
		Identity: StateIdentity{
			AuthID:   "auth-1",
			Provider: ProviderClaude,
			Resource: "messages",
			Window:   "five-hour",
		},
		AuthLabel:  "Primary",
		Kind:       TransitionWarning,
		From:       AlertHealthy,
		To:         AlertWarning,
		OccurredAt: now,
	}
}

type secretAwareStore interface {
	LoadSettingsWithSecret(context.Context) (Settings, *EncryptedSecret, error)
	SaveSettingsWithSecret(context.Context, int64, Settings, SecretUpdate, *SecretCipher, string) (Settings, error)
}

var _ secretAwareStore = (Store)(nil)
