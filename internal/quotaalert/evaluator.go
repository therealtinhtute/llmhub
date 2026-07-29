package quotaalert

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"time"
)

// EvaluationInput contains one pure quota evaluation cycle.
type EvaluationInput struct {
	Settings       Settings
	Observations   []Observation
	PreviousStates []CurrentState
	EvaluatedAt    time.Time
}

// EvaluationResult is the durable output produced by quota evaluation.
type EvaluationResult struct {
	States  []CurrentState
	Events  []TransitionEvent
	Batches []NotificationBatch
}

// EvaluateObservations evaluates normalized provider observations into durable states, events, and batches.
func EvaluateObservations(input EvaluationInput) (EvaluationResult, error) {
	settings, err := normalizedEvaluationSettings(input.Settings)
	if err != nil {
		return EvaluationResult{}, err
	}
	evaluatedAt := input.EvaluatedAt.UTC().Truncate(time.Microsecond)
	if evaluatedAt.IsZero() {
		return EvaluationResult{}, fmt.Errorf("evaluation time is required")
	}
	if !settings.Enabled {
		return EvaluationResult{}, nil
	}

	previous, err := normalizePreviousStates(input.PreviousStates)
	if err != nil {
		return EvaluationResult{}, err
	}
	observations, err := normalizeLatestObservations(input.Observations)
	if err != nil {
		return EvaluationResult{}, err
	}
	providerSettings := evaluationProviderSettings(settings)

	states := make([]CurrentState, 0, len(observations))
	events := make([]TransitionEvent, 0)
	keys := sortedObservationKeys(observations)
	for _, key := range keys {
		observation := observations[key]
		providerConfig := providerSettings[observation.Identity.Provider]
		if !providerConfig.enabled {
			continue
		}
		prior, hadPrior := previous[key]
		state, event, hasEvent, err := evaluateObservation(settings, providerConfig.threshold, observation, prior, hadPrior, evaluatedAt)
		if err != nil {
			return EvaluationResult{}, err
		}
		states = append(states, state)
		if hasEvent {
			events = append(events, event)
		}
	}

	batches, err := groupTransitionEvents(events, evaluatedAt)
	if err != nil {
		return EvaluationResult{}, err
	}
	return EvaluationResult{States: states, Events: events, Batches: batches}, nil
}

type evaluatorProviderConfig struct {
	enabled   bool
	threshold Percentage
}

func normalizedEvaluationSettings(settings Settings) (Settings, error) {
	if settings.PollInterval == 0 {
		settings.PollInterval = DefaultPollInterval
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func evaluationProviderSettings(settings Settings) map[Provider]evaluatorProviderConfig {
	configs := make(map[Provider]evaluatorProviderConfig, len(supportedProviders))
	for _, provider := range SupportedProviders() {
		configs[provider] = evaluatorProviderConfig{enabled: true, threshold: settings.WarningThreshold}
	}
	for _, override := range settings.ProviderOverrides {
		config := configs[override.Provider]
		config.enabled = override.Enabled
		if override.WarningThreshold != nil {
			config.threshold = *override.WarningThreshold
		}
		configs[override.Provider] = config
	}
	return configs
}

func normalizePreviousStates(states []CurrentState) (map[string]CurrentState, error) {
	previous := make(map[string]CurrentState, len(states))
	for _, state := range states {
		normalized, err := state.Normalize()
		if err != nil {
			return nil, err
		}
		key, err := normalized.Identity.StableKey()
		if err != nil {
			return nil, err
		}
		previous[key] = normalized
	}
	return previous, nil
}

func normalizeLatestObservations(observations []Observation) (map[string]Observation, error) {
	latest := make(map[string]Observation, len(observations))
	for _, observation := range observations {
		normalized, err := observation.Normalize()
		if err != nil {
			return nil, err
		}
		key, err := normalized.Identity.StableKey()
		if err != nil {
			return nil, err
		}
		prior, exists := latest[key]
		if !exists || prior.ObservedAt.Before(normalized.ObservedAt) || (prior.ObservedAt.Equal(normalized.ObservedAt) && observationSortKey(normalized) > observationSortKey(prior)) {
			latest[key] = normalized
		}
	}
	return latest, nil
}

func sortedObservationKeys(observations map[string]Observation) []string {
	keys := make([]string, 0, len(observations))
	for key := range observations {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		leftObservation := observations[keys[left]]
		rightObservation := observations[keys[right]]
		if leftObservation.Identity.Provider != rightObservation.Identity.Provider {
			return leftObservation.Identity.Provider < rightObservation.Identity.Provider
		}
		if leftObservation.Identity.AuthID != rightObservation.Identity.AuthID {
			return leftObservation.Identity.AuthID < rightObservation.Identity.AuthID
		}
		if leftObservation.Identity.Resource != rightObservation.Identity.Resource {
			return leftObservation.Identity.Resource < rightObservation.Identity.Resource
		}
		return leftObservation.Identity.Window < rightObservation.Identity.Window
	})
	return keys
}

func observationSortKey(observation Observation) string {
	return string(observation.Identity.Provider) + "\x00" + observation.Identity.AuthID + "\x00" + observation.Identity.Resource + "\x00" + observation.Identity.Window
}

func evaluateObservation(settings Settings, threshold Percentage, observation Observation, prior CurrentState, hadPrior bool, evaluatedAt time.Time) (CurrentState, TransitionEvent, bool, error) {
	if observation.Health == CollectionUnknown {
		if hadPrior && prior.Alert != AlertUnknown {
			prior.UpdatedAt = evaluatedAt
			normalized, err := prior.Normalize()
			return normalized, TransitionEvent{}, false, err
		}
		state := CurrentState{
			Identity:       observation.Identity,
			AuthLabel:      observation.AuthLabel,
			Alert:          AlertUnknown,
			Health:         CollectionUnknown,
			ObservedAt:     observation.ObservedAt,
			TransitionedAt: observation.ObservedAt,
			UpdatedAt:      evaluatedAt,
			Revision:       settings.Revision,
		}
		normalized, err := state.Normalize()
		return normalized, TransitionEvent{}, false, err
	}

	alert := evaluateAlert(observation, threshold)
	transitionedAt := observation.ObservedAt
	from := AlertUnknown
	if hadPrior {
		from = prior.Alert
		if prior.Alert == alert {
			transitionedAt = prior.TransitionedAt
		}
	}
	state := CurrentState{
		Identity:       observation.Identity,
		AuthLabel:      observation.AuthLabel,
		Alert:          alert,
		Health:         CollectionReliable,
		Remaining:      observation.Remaining,
		RemainingKnown: observation.RemainingKnown,
		ResetAt:        observation.ResetAt,
		ResetKnown:     observation.ResetKnown,
		ObservedAt:     observation.ObservedAt,
		TransitionedAt: transitionedAt,
		UpdatedAt:      evaluatedAt,
		Revision:       settings.Revision,
	}
	if alert == AlertExhausted && !state.RemainingKnown {
		state.Remaining = 0
	}
	normalizedState, err := state.Normalize()
	if err != nil {
		return CurrentState{}, TransitionEvent{}, false, err
	}

	kind, emit := transitionKind(from, alert, settings.NotifyRecovery)
	if !emit && settings.ReminderInterval > 0 && hadPrior && prior.Alert == alert && (alert == AlertWarning || alert == AlertExhausted) && !prior.UpdatedAt.Add(settings.ReminderInterval).After(evaluatedAt) {
		kind = TransitionReminder
		emit = true
	}
	if !emit {
		return normalizedState, TransitionEvent{}, false, nil
	}
	event := TransitionEvent{
		ID:             eventID(kind, from, alert, normalizedState.Identity, evaluatedAt),
		Identity:       normalizedState.Identity,
		AuthLabel:      normalizedState.AuthLabel,
		Kind:           kind,
		From:           from,
		To:             alert,
		Remaining:      normalizedState.Remaining,
		RemainingKnown: normalizedState.RemainingKnown,
		ResetAt:        normalizedState.ResetAt,
		ResetKnown:     normalizedState.ResetKnown,
		OccurredAt:     evaluatedAt,
	}
	normalizedEvent, err := event.Normalize()
	if err != nil {
		return CurrentState{}, TransitionEvent{}, false, err
	}
	return normalizedState, normalizedEvent, true, nil
}

func evaluateAlert(observation Observation, threshold Percentage) AlertState {
	if observation.ExplicitlyExhausted || (observation.RemainingKnown && observation.Remaining == 0) {
		return AlertExhausted
	}
	if observation.RemainingKnown && observation.Remaining <= threshold {
		return AlertWarning
	}
	return AlertHealthy
}

func transitionKind(from, to AlertState, notifyRecovery bool) (TransitionKind, bool) {
	if from == to {
		return "", false
	}
	switch to {
	case AlertWarning:
		if from == AlertHealthy || from == AlertUnknown {
			return TransitionWarning, true
		}
	case AlertExhausted:
		if from == AlertHealthy || from == AlertWarning || from == AlertUnknown {
			return TransitionExhausted, true
		}
	case AlertHealthy:
		if notifyRecovery && (from == AlertWarning || from == AlertExhausted) {
			return TransitionRecovery, true
		}
	}
	return "", false
}

func eventID(kind TransitionKind, from, to AlertState, identity StateIdentity, occurredAt time.Time) string {
	key, _ := identity.StableKey()
	hash := sha256.New()
	for _, field := range []string{
		key,
		string(kind),
		string(from),
		string(to),
		occurredAt.Format(time.RFC3339Nano),
	} {
		_, _ = hash.Write([]byte(strconv.Itoa(len(field))))
		_, _ = hash.Write([]byte{':'})
		_, _ = hash.Write([]byte(field))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func groupTransitionEvents(events []TransitionEvent, createdAt time.Time) ([]NotificationBatch, error) {
	if len(events) == 0 {
		return nil, nil
	}
	byProvider := make(map[Provider][]TransitionEvent)
	for _, event := range events {
		normalized, err := event.Normalize()
		if err != nil {
			return nil, err
		}
		byProvider[normalized.Identity.Provider] = append(byProvider[normalized.Identity.Provider], normalized)
	}
	providers := make([]Provider, 0, len(byProvider))
	for provider := range byProvider {
		providers = append(providers, provider)
	}
	slices.Sort(providers)
	batches := make([]NotificationBatch, 0, len(providers))
	for _, provider := range providers {
		batch, err := NewNotificationBatch(provider, byProvider[provider], createdAt)
		if err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}
	return batches, nil
}
