package quotaalert

import (
	"context"
	"errors"
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
	observation.FailureCode = FailureUpstreamHTTP

	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{observation},
		PreviousStates: []CurrentState{previous},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	state := result.States[0]
	if len(result.States) != 1 || state.Alert != AlertWarning || state.Health != CollectionUnknown {
		t.Fatalf("retained state = %#v", result.States)
	}
	if state.FailureCode != FailureUpstreamHTTP {
		t.Fatalf("failure code = %q, want %q", state.FailureCode, FailureUpstreamHTTP)
	}
	// The last reliable observation is the retained state's observed time.
	if !state.LastReliableObservedAt.Equal(previous.ObservedAt) {
		t.Fatalf("last reliable observed at = %v, want %v", state.LastReliableObservedAt, previous.ObservedAt)
	}
	if !state.TransitionedAt.Equal(previous.TransitionedAt) || state.ObservedAt != previous.ObservedAt {
		t.Fatalf("retained timestamps changed incorrectly: %#v", state)
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

func TestEvaluatorUnknownPropagatesFailureCodeAndClearsOnRecovery(t *testing.T) {
	now := time.Date(2026, time.July, 29, 3, 45, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	previous := evaluatorState("auth-1", ProviderKiro, AlertUnknown, 0, now.Add(-2*time.Hour))
	previous.Health = CollectionUnknown
	previous.RemainingKnown = false
	previous.FailureCode = FailureDecode
	previous.LastReliableObservedAt = now.Add(-3 * time.Hour)

	unknown := evaluatorObservation("auth-1", ProviderKiro, "messages", "weekly", 0, now)
	unknown.Health = CollectionUnknown
	unknown.RemainingKnown = false
	unknown.FailureCode = FailureTimeout

	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{unknown},
		PreviousStates: []CurrentState{previous},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	state := result.States[0]
	if state.FailureCode != FailureTimeout || !state.LastReliableObservedAt.Equal(previous.LastReliableObservedAt) {
		t.Fatalf("unknown state did not carry failure metadata: %#v", state)
	}

	recovered := evaluatorObservation("auth-1", ProviderKiro, "messages", "weekly", 50, now.Add(time.Minute))
	result, err = EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{recovered},
		PreviousStates: []CurrentState{state},
		EvaluatedAt:    now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("recovery EvaluateObservations() error = %v", err)
	}
	state = result.States[0]
	if state.FailureCode != FailureNone || state.Health != CollectionReliable {
		t.Fatalf("reliable state retained failure metadata: %#v", state)
	}
	if !state.LastReliableObservedAt.Equal(recovered.ObservedAt) {
		t.Fatalf("last reliable observed at = %v, want %v", state.LastReliableObservedAt, recovered.ObservedAt)
	}
}

func TestClassifyCollectorErrorMapsBoundedCodes(t *testing.T) {
	t.Parallel()
	cases := map[error]CollectionFailureCode{
		nil:                                   FailureNone,
		errors.New("access token is missing"): FailureCredentialMissing,
		errors.New("project id is missing"):   FailureCredentialMissing,
		errors.New("refresh failed: unauthorized"):       FailureCredentialRejected,
		errors.New("usage request failed: http 401"):     FailureCredentialRejected,
		errors.New("usage request failed: http 500"):     FailureUpstreamHTTP,
		errors.New("decode usage payload: invalid json"): FailureDecode,
		errors.New("no recognized windows in response"):  FailureDecode,
		errors.New("context deadline exceeded"):          FailureTimeout,
		errors.New("dial tcp: connection refused"):       FailureTransport,
		errors.New("something odd happened"):             FailureInternal,
	}
	for err, want := range cases {
		if got := classifyCollectorError(err); got != want {
			t.Fatalf("classifyCollectorError(%v) = %q, want %q", err, got, want)
		}
	}
	if got := classifyCollectorError(context.DeadlineExceeded); got != FailureTimeout {
		t.Fatalf("classifyCollectorError(deadline) = %q, want %q", got, FailureTimeout)
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

func TestEvaluatorConfirmationSamplesGateWarning(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.ConfirmationSamples = 2

	first, err := EvaluateObservations(EvaluationInput{
		Settings:     settings,
		Observations: []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 5, now)},
		EvaluatedAt:  now,
	})
	if err != nil {
		t.Fatalf("first EvaluateObservations() error = %v", err)
	}
	if len(first.States) != 1 || first.States[0].Alert != AlertHealthy || first.States[0].ConsecutiveBelow != 1 {
		t.Fatalf("first below-threshold state = %#v", first.States)
	}
	if len(first.Events) != 0 || len(first.Batches) != 0 {
		t.Fatalf("first sample emitted events=%#v", first.Events)
	}

	second, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 5, now.Add(time.Minute))},
		PreviousStates: first.States,
		EvaluatedAt:    now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("second EvaluateObservations() error = %v", err)
	}
	if len(second.States) != 1 || second.States[0].Alert != AlertWarning || second.States[0].ConsecutiveBelow != 2 {
		t.Fatalf("second below-threshold state = %#v", second.States)
	}
	if len(second.Events) != 1 || second.Events[0].Kind != TransitionWarning {
		t.Fatalf("confirmed warning events = %#v", second.Events)
	}
}

func TestEvaluatorConfirmationResetsAboveThreshold(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 10, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.ConfirmationSamples = 2

	prior := evaluatorState("auth-1", ProviderClaude, AlertHealthy, 5, now.Add(-2*time.Minute))
	prior.Identity.Window = "five-hour"
	prior.ConsecutiveBelow = 1

	recovered, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 50, now.Add(-time.Minute))},
		PreviousStates: []CurrentState{prior},
		EvaluatedAt:    now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("recover EvaluateObservations() error = %v", err)
	}
	if recovered.States[0].ConsecutiveBelow != 0 {
		t.Fatalf("counter not reset above threshold: %#v", recovered.States[0])
	}

	again, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 5, now)},
		PreviousStates: recovered.States,
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("again EvaluateObservations() error = %v", err)
	}
	if again.States[0].Alert != AlertHealthy || again.States[0].ConsecutiveBelow != 1 || len(again.Events) != 0 {
		t.Fatalf("counter did not restart after healthy sample: %#v events=%#v", again.States[0], again.Events)
	}
}

func TestEvaluatorUnknownDoesNotIncrementConfirmation(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 20, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.ConfirmationSamples = 2

	prior := evaluatorState("auth-1", ProviderCodex, AlertHealthy, 5, now.Add(-2*time.Minute))
	prior.Identity.Window = "weekly"
	prior.ConsecutiveBelow = 1

	unknown := evaluatorObservation("auth-1", ProviderCodex, "messages", "weekly", 0, now.Add(-time.Minute))
	unknown.Health = CollectionUnknown
	unknown.RemainingKnown = false
	unknown.FailureCode = FailureTimeout

	middle, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{unknown},
		PreviousStates: []CurrentState{prior},
		EvaluatedAt:    now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("unknown EvaluateObservations() error = %v", err)
	}
	if middle.States[0].ConsecutiveBelow != 1 || middle.States[0].Health != CollectionUnknown {
		t.Fatalf("unknown collection mutated confirmation counter: %#v", middle.States[0])
	}

	confirmed, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderCodex, "messages", "weekly", 5, now)},
		PreviousStates: middle.States,
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("confirmed EvaluateObservations() error = %v", err)
	}
	if confirmed.States[0].Alert != AlertWarning || confirmed.States[0].ConsecutiveBelow != 2 {
		t.Fatalf("confirmation did not resume across unknown: %#v", confirmed.States[0])
	}
	if len(confirmed.Events) != 1 || confirmed.Events[0].Kind != TransitionWarning {
		t.Fatalf("confirmation events = %#v", confirmed.Events)
	}
}

func TestEvaluatorExplicitExhaustionBypassesConfirmation(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 30, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.ConfirmationSamples = 3

	observation := evaluatorObservation("auth-1", ProviderCodex, "code", "monthly", 0, now)
	observation.ExplicitlyExhausted = true
	result, err := EvaluateObservations(EvaluationInput{
		Settings:     settings,
		Observations: []Observation{observation},
		EvaluatedAt:  now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	if len(result.States) != 1 || result.States[0].Alert != AlertExhausted {
		t.Fatalf("explicit exhaustion held by confirmation: %#v", result.States)
	}
	if len(result.Events) != 1 || result.Events[0].Kind != TransitionExhausted {
		t.Fatalf("explicit exhaustion events = %#v", result.Events)
	}
}

func TestEvaluatorRecoveryMarginBlocksFlap(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 40, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.NotifyRecovery = true
	settings.WarningThreshold = 10
	settings.RecoveryMargin = 5

	prior := evaluatorState("auth-1", ProviderClaude, AlertWarning, 5, now.Add(-time.Hour))
	prior.Identity.Window = "five-hour"
	prior.ConsecutiveBelow = 2

	// 12% remaining is above the 10% threshold but inside the 15% recovery level.
	held, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 12, now)},
		PreviousStates: []CurrentState{prior},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("held EvaluateObservations() error = %v", err)
	}
	if held.States[0].Alert != AlertWarning || len(held.Events) != 0 {
		t.Fatalf("recovery margin not applied: %#v events=%#v", held.States[0], held.Events)
	}

	recovered, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 16, now.Add(time.Minute))},
		PreviousStates: held.States,
		EvaluatedAt:    now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("recovered EvaluateObservations() error = %v", err)
	}
	if recovered.States[0].Alert != AlertHealthy || recovered.States[0].ConsecutiveBelow != 0 {
		t.Fatalf("recovery above threshold+margin failed: %#v", recovered.States[0])
	}
	if len(recovered.Events) != 1 || recovered.Events[0].Kind != TransitionRecovery {
		t.Fatalf("recovery events = %#v", recovered.Events)
	}
}

func TestEvaluatorZeroRecoveryMarginPreservesRecovery(t *testing.T) {
	now := time.Date(2026, time.July, 29, 5, 50, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.NotifyRecovery = true
	settings.WarningThreshold = 10
	settings.RecoveryMargin = 0

	prior := evaluatorState("auth-1", ProviderClaude, AlertWarning, 5, now.Add(-time.Hour))
	prior.Identity.Window = "five-hour"
	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   []Observation{evaluatorObservation("auth-1", ProviderClaude, "messages", "five-hour", 11, now)},
		PreviousStates: []CurrentState{prior},
		EvaluatedAt:    now,
	})
	if err != nil {
		t.Fatalf("EvaluateObservations() error = %v", err)
	}
	if result.States[0].Alert != AlertHealthy || len(result.Events) != 1 || result.Events[0].Kind != TransitionRecovery {
		t.Fatalf("zero-margin recovery changed: %#v events=%#v", result.States[0], result.Events)
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
