package quotaalert

import (
	"testing"
	"time"
)

func TestEvaluatorEvaluatesStatesTransitionsAndBatches(t *testing.T) {
	now := time.Date(2026, time.July, 29, 3, 20, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.Revision = 7
	settings.NotifyRecovery = true

	previousRecovered := evaluatorState("auth-recovered", ProviderClaude, AlertExhausted, 0, now.Add(-time.Hour))
	previousRecovered.Identity.Resource = "messages"
	previousRecovered.Identity.Window = "five-hour"
	observations := []Observation{
		evaluatorObservation("auth-healthy", ProviderClaude, "messages", "five-hour", 80, now),
		evaluatorObservation("auth-warning", ProviderClaude, "messages", "five-hour", 10, now),
		evaluatorObservation("auth-exhausted", ProviderClaude, "messages", "five-hour", 0, now),
		evaluatorObservation("auth-recovered", ProviderClaude, "messages", "five-hour", 90, now),
	}

	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   observations,
		PreviousStates: []CurrentState{previousRecovered},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	if len(result.States) != 4 {
		t.Fatalf("len(states) = %d, want 4", len(result.States))
	}
	stateByAuth := make(map[string]CurrentState, len(result.States))
	for _, state := range result.States {
		stateByAuth[state.Identity.AuthID] = state
		if state.Revision != settings.Revision || !state.UpdatedAt.Equal(now) {
			t.Fatalf("state metadata = %#v", state)
		}
	}
	if stateByAuth["auth-healthy"].Alert != AlertHealthy {
		t.Fatalf("healthy alert = %s", stateByAuth["auth-healthy"].Alert)
	}
	if stateByAuth["auth-warning"].Alert != AlertWarning {
		t.Fatalf("warning alert = %s", stateByAuth["auth-warning"].Alert)
	}
	if stateByAuth["auth-exhausted"].Alert != AlertExhausted {
		t.Fatalf("exhausted alert = %s", stateByAuth["auth-exhausted"].Alert)
	}
	if stateByAuth["auth-recovered"].Alert != AlertHealthy {
		t.Fatalf("recovered alert = %s", stateByAuth["auth-recovered"].Alert)
	}

	if len(result.Events) != 3 {
		t.Fatalf("len(events) = %d, want 3: %#v", len(result.Events), result.Events)
	}
	kinds := map[TransitionKind]bool{}
	for _, event := range result.Events {
		kinds[event.Kind] = true
		if event.ID == "" || !event.OccurredAt.Equal(now) {
			t.Fatalf("event metadata = %#v", event)
		}
	}
	for _, kind := range []TransitionKind{TransitionWarning, TransitionExhausted, TransitionRecovery} {
		if !kinds[kind] {
			t.Fatalf("missing transition kind %s in %#v", kind, result.Events)
		}
	}
	if len(result.Batches) != 1 || result.Batches[0].Provider() != ProviderClaude || len(result.Batches[0].Events()) != 3 {
		t.Fatalf("batches = %#v", result.Batches)
	}
}

func TestEvaluatorRetainsPriorStateOnUnknownCollection(t *testing.T) {
	now := time.Date(2026, time.July, 29, 3, 30, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.Revision = 8
	previous := evaluatorState("auth-1", ProviderCodex, AlertWarning, 4, now.Add(-time.Hour))
	observation := evaluatorObservation("auth-1", ProviderCodex, "messages", "weekly", 0, now)
	observation.Health = CollectionUnknown
	observation.RemainingKnown = false

	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{observation},
		PreviousStates: []CurrentState{previous},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	if len(result.States) != 1 || result.States[0].Alert != AlertWarning || result.States[0].Health != CollectionReliable {
		t.Fatalf("retained state = %#v", result.States)
	}
	if !result.States[0].TransitionedAt.Equal(previous.TransitionedAt) || result.States[0].ObservedAt != previous.ObservedAt {
		t.Fatalf("retained timestamps changed incorrectly: %#v", result.States[0])
	}
	if len(result.Events) != 0 || len(result.Batches) != 0 {
		t.Fatalf("unknown collection emitted events=%#v batches=%#v", result.Events, result.Batches)
	}
}

func TestEvaluatorUnknownWithoutPriorStateAndBareFailureDoesNotExhaust(t *testing.T) {
	now := time.Date(2026, time.July, 29, 3, 35, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	observation := evaluatorObservation("auth-1", ProviderKiro, "requests", "daily", 0, now)
	observation.Health = CollectionUnknown
	observation.RemainingKnown = false

	result, err := EvaluateObservations(EvaluationInput{Settings: settings, Observations: []Observation{observation}, EvaluatedAt: now})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	if len(result.States) != 1 || result.States[0].Alert != AlertUnknown || result.States[0].Health != CollectionUnknown || result.States[0].RemainingKnown {
		t.Fatalf("unknown state = %#v", result.States)
	}
	if len(result.Events) != 0 || len(result.Batches) != 0 {
		t.Fatalf("unknown collection emitted events=%#v batches=%#v", result.Events, result.Batches)
	}
}

func TestEvaluatorReminderAndProviderOverrides(t *testing.T) {
	now := time.Date(2026, time.July, 29, 4, 0, 0, 0, time.UTC)
	threshold := Percentage(20)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.Revision = 10
	settings.ReminderInterval = time.Hour
	settings.ProviderOverrides = []ProviderOverride{
		{Provider: ProviderClaude, Enabled: false},
		{Provider: ProviderCodex, Enabled: true, WarningThreshold: &threshold},
	}
	previous := evaluatorState("auth-codex", ProviderCodex, AlertWarning, 15, now.Add(-2*time.Hour))
	previous.TransitionedAt = now.Add(-2 * time.Hour)
	previous.UpdatedAt = now.Add(-2 * time.Hour)
	observation := evaluatorObservation("auth-codex", ProviderCodex, "messages", "weekly", 15, now)
	disabledProviderObservation := evaluatorObservation("auth-claude", ProviderClaude, "messages", "daily", 0, now)

	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{disabledProviderObservation, observation},
		PreviousStates: []CurrentState{previous},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	if len(result.States) != 1 || result.States[0].Identity.Provider != ProviderCodex {
		t.Fatalf("provider override states = %#v", result.States)
	}
	if len(result.Events) != 1 || result.Events[0].Kind != TransitionReminder {
		t.Fatalf("reminder events = %#v", result.Events)
	}
}

func TestEvaluatorIsDeterministicAndDeduplicatesInputs(t *testing.T) {
	now := time.Date(2026, time.July, 29, 4, 30, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	first := evaluatorObservation("auth-2", ProviderXAI, "credits", "monthly", 5, now)
	second := evaluatorObservation("auth-1", ProviderGeminiCLI, "requests", "daily", 5, now)
	duplicateOlder := first
	duplicateOlder.Remaining = 80
	duplicateOlder.ObservedAt = now.Add(-time.Minute)

	left, err := EvaluateObservations(EvaluationInput{Settings: settings, Observations: []Observation{first, second, duplicateOlder}, EvaluatedAt: now})
	if err != nil {
		t.Fatalf("left EvaluateObservations() error = %v", err)
	}
	right, err := EvaluateObservations(EvaluationInput{Settings: settings, Observations: []Observation{duplicateOlder, second, first}, EvaluatedAt: now})
	if err != nil {
		t.Fatalf("right EvaluateObservations() error = %v", err)
	}
	if len(left.Events) != len(right.Events) || len(left.Batches) != len(right.Batches) || len(left.States) != len(right.States) {
		t.Fatalf("deterministic lengths differ: left=%#v right=%#v", left, right)
	}
	for index := range left.Events {
		if left.Events[index].ID != right.Events[index].ID {
			t.Fatalf("event order/id differs at %d: %q vs %q", index, left.Events[index].ID, right.Events[index].ID)
		}
	}
	if left.States[1].Remaining != 5 {
		t.Fatalf("newest duplicate was not retained: %#v", left.States)
	}
}

func evaluatorObservation(authID string, provider Provider, resource string, window string, remaining Percentage, observedAt time.Time) Observation {
	return Observation{
		Identity: StateIdentity{
			AuthID:   authID,
			Provider: provider,
			Resource: resource,
			Window:   window,
		},
		AuthLabel:      authID + " label",
		Health:         CollectionReliable,
		Remaining:      remaining,
		RemainingKnown: true,
		ObservedAt:     observedAt,
	}
}

func evaluatorState(authID string, provider Provider, alert AlertState, remaining Percentage, observedAt time.Time) CurrentState {
	state := CurrentState{
		Identity: StateIdentity{
			AuthID:   authID,
			Provider: provider,
			Resource: "messages",
			Window:   "weekly",
		},
		AuthLabel:      authID + " label",
		Alert:          alert,
		Health:         CollectionReliable,
		Remaining:      remaining,
		RemainingKnown: alert != AlertExhausted,
		ObservedAt:     observedAt,
		TransitionedAt: observedAt,
		UpdatedAt:      observedAt,
	}
	if alert == AlertExhausted {
		state.RemainingKnown = true
		state.Remaining = 0
	}
	return state
}
