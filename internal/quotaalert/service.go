package quotaalert

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultCollectionTimeout      = 30 * time.Second
	DefaultCollectionWorkers      = 4
	DefaultDeliveryInterval       = 30 * time.Second
	DefaultNotificationClaimLimit = 10
	DefaultNotificationLease      = 2 * time.Minute
	DefaultNotificationRetryDelay = time.Minute
	MaxNotificationAttempts       = 3
	DefaultRetentionPruneLimit    = 100
	DefaultRetentionAge           = 30 * 24 * time.Hour

	// adaptivePollDivisor is the fast-poll factor applied while any reliable
	// quota window is hot (reset due within one base interval, or remaining
	// within adaptiveThresholdHeadroom of the effective warning threshold).
	adaptivePollDivisor = 5
	// adaptiveThresholdHeadroom is the remaining-percentage headroom above the
	// warning threshold that keeps a window hot.
	adaptiveThresholdHeadroom = Percentage(10)
)

// AuthSource lists persisted auth snapshots eligible for quota monitoring.
type AuthSource interface {
	ListQuotaAlertAuths(context.Context) ([]AuthSnapshot, error)
}

// Clock supplies service time for tests.
type Clock interface {
	Now() time.Time
}

// ServiceConfig contains quota monitor service dependencies.
type ServiceConfig struct {
	Store             Store
	AuthSource        AuthSource
	CollectorRegistry *CollectorRegistry
	CollectorDeps     CollectorDependencies
	Sender            Sender
	Clock             Clock
	PollInterval      time.Duration
	DeliveryInterval  time.Duration
	CollectionTimeout time.Duration
}

// Service runs quota collection and durable notification delivery.
type Service struct {
	store             Store
	authSource        AuthSource
	collectorRegistry *CollectorRegistry
	collectorDeps     CollectorDependencies
	sender            Sender
	clock             Clock
	pollInterval      time.Duration
	deliveryInterval  time.Duration
	collectionTimeout time.Duration

	startOnce sync.Once
	mu        sync.Mutex
	cancel    context.CancelFunc
	wake      chan struct{}
	done      chan struct{}
}

// NewService validates dependencies and constructs a quota monitor service.
func NewService(config ServiceConfig) (*Service, error) {
	if config.Store == nil {
		return nil, fmt.Errorf("quota alert store is required")
	}
	pollInterval := config.PollInterval
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}
	if pollInterval < MinPollInterval || pollInterval > MaxPollInterval {
		return nil, fmt.Errorf("quota alert poll interval must be between %s and %s", MinPollInterval, MaxPollInterval)
	}
	deliveryInterval := config.DeliveryInterval
	if deliveryInterval == 0 {
		deliveryInterval = DefaultDeliveryInterval
	}
	if deliveryInterval < time.Second || deliveryInterval > time.Hour {
		return nil, fmt.Errorf("quota alert delivery interval must be between 1s and 1h")
	}
	collectionTimeout := config.CollectionTimeout
	if collectionTimeout == 0 {
		collectionTimeout = DefaultCollectionTimeout
	}
	if collectionTimeout < time.Second || collectionTimeout > time.Minute {
		return nil, fmt.Errorf("quota alert collection timeout must be between 1s and 1m")
	}
	clock := config.Clock
	if clock == nil {
		clock = realClock{}
	}
	registry := config.CollectorRegistry
	if registry == nil {
		registry = NewCollectorRegistry()
	}
	return &Service{
		store:             config.Store,
		authSource:        config.AuthSource,
		collectorRegistry: registry,
		collectorDeps:     config.CollectorDeps,
		sender:            config.Sender,
		clock:             clock,
		pollInterval:      pollInterval,
		deliveryInterval:  deliveryInterval,
		collectionTimeout: collectionTimeout,
		wake:              make(chan struct{}, 1),
	}, nil
}

// Start launches collection and delivery loops. Calling Start multiple times is safe.
func (s *Service) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		workerCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		s.mu.Lock()
		s.cancel = cancel
		s.done = done
		s.mu.Unlock()
		go s.run(workerCtx, done)
	})
}

// Stop cancels loops and waits for workers to exit. Calling Stop multiple times is safe.
func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()
	if cancel == nil || done == nil {
		return
	}
	cancel()
	<-done
}

// Wake requests an immediate collection cycle.
func (s *Service) Wake() {
	if s == nil {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// RunCollectionOnce runs one single-owner collection cycle.
func (s *Service) RunCollectionOnce(ctx context.Context) error {
	settings, err := s.store.LoadSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}
	if err = settings.Validate(); err != nil {
		return err
	}
	lease, acquired, err := s.store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		return err
	}
	defer func() { _ = lease.Release(context.Background()) }()

	auths, err := s.listAuths(ctx)
	if err != nil {
		return err
	}
	previous, err := s.listAllStates(ctx)
	if err != nil {
		return err
	}
	healthRows, err := s.store.ListCollectionHealth(ctx)
	if err != nil {
		return err
	}
	evaluatedAt := s.clock.Now()
	active := activeAuthProviderKeys(settings, auths)
	collection := s.collectObservations(ctx, auths, previous, active)
	degradedEvents, healthUpserts, healthDeletes := advanceCollectionHealth(
		settings, auths, active, &collection, healthRows, evaluatedAt)
	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   collection.observations,
		PreviousStates: previous,
		EvaluatedAt:    evaluatedAt,
	})
	if err != nil {
		return err
	}
	events := append(result.Events, degradedEvents...)
	batches, err := groupTransitionEvents(events, evaluatedAt)
	if err != nil {
		return err
	}
	return s.store.CommitCollection(ctx, lease, CollectionCommit{
		SettingsRevision: settings.Revision,
		States:           result.States,
		RemovedStates:    removedStateIdentities(previous, active, collection),
		Events:           events,
		Batches:          batches,
		HealthUpserts:    healthUpserts,
		HealthDeletes:    healthDeletes,
	})
}

// advanceCollectionHealth tracks consecutive collection failures per
// (auth, provider) and emits one monitor_degraded event per streak when the
// count crosses settings.DegradedFailureThreshold. Crossing auths also get a
// collection/latest unknown observation appended to the cycle so the event's
// durable state row exists; the row is auto-removed on the next reliable cycle
// by removedStateIdentities. Reliable cycles delete the streak row, re-arming
// the one-shot.
func advanceCollectionHealth(
	settings Settings,
	auths []AuthSnapshot,
	active map[authProviderKey]struct{},
	collection *collectionCycle,
	healthRows []CollectionHealthRecord,
	evaluatedAt time.Time,
) ([]TransitionEvent, []CollectionHealthRecord, []CollectionHealthKey) {
	healthByKey := make(map[authProviderKey]CollectionHealthRecord, len(healthRows))
	for _, row := range healthRows {
		key := authProviderKey{authID: row.Key.AuthID, provider: row.Key.Provider}
		healthByKey[key] = row
	}
	authByKey := make(map[authProviderKey]AuthSnapshot, len(auths))
	for _, auth := range auths {
		key, ok := authKey(auth)
		if !ok {
			continue
		}
		authByKey[key] = auth
	}
	failureCodeByKey := make(map[authProviderKey]CollectionFailureCode)
	for _, observation := range collection.observations {
		if observation.Health != CollectionUnknown {
			continue
		}
		key := authProviderKey{authID: observation.Identity.AuthID, provider: observation.Identity.Provider}
		if _, exists := failureCodeByKey[key]; !exists {
			failureCodeByKey[key] = observation.FailureCode
		}
	}

	threshold := settings.DegradedFailureThreshold
	upserts := make([]CollectionHealthRecord, 0, len(collection.failed))
	deletes := make([]CollectionHealthKey, 0, len(healthByKey))
	degraded := make([]TransitionEvent, 0)
	keys := sortedAuthProviderKeys(active)
	for _, key := range keys {
		if _, failed := collection.failed[key]; !failed {
			if _, tracked := healthByKey[key]; tracked {
				deletes = append(deletes, CollectionHealthKey{AuthID: key.authID, Provider: key.provider})
			}
			continue
		}
		previous := healthByKey[key].ConsecutiveFailures
		count := previous + 1
		code := failureCodeByKey[key]
		if code == "" {
			code = FailureInternal
		}
		upserts = append(upserts, CollectionHealthRecord{
			Key:                 CollectionHealthKey{AuthID: key.authID, Provider: key.provider},
			ConsecutiveFailures: count,
			LastFailureCode:     code,
			UpdatedAt:           evaluatedAt,
		})
		if threshold <= 0 || previous >= threshold || count < threshold {
			continue
		}
		auth, ok := authByKey[key]
		if !ok {
			continue
		}
		observation := unknownObservation(auth, evaluatedAt, code)
		if _, observed := collection.observed[observation.Identity]; !observed {
			collection.observed[observation.Identity] = struct{}{}
			collection.observations = append(collection.observations, observation)
		}
		event, err := (TransitionEvent{
			ID:         eventID(TransitionMonitorDegraded, AlertUnknown, AlertUnknown, observation.Identity, evaluatedAt),
			Identity:   observation.Identity,
			AuthLabel:  observation.AuthLabel,
			Kind:       TransitionMonitorDegraded,
			From:       AlertUnknown,
			To:         AlertUnknown,
			OccurredAt: evaluatedAt,
		}).Normalize()
		if err == nil {
			degraded = append(degraded, event)
		}
	}
	for key := range healthByKey {
		if _, isActive := active[key]; !isActive {
			deletes = append(deletes, CollectionHealthKey{AuthID: key.authID, Provider: key.provider})
		}
	}
	return degraded, upserts, deletes
}

func sortedAuthProviderKeys(active map[authProviderKey]struct{}) []authProviderKey {
	keys := make([]authProviderKey, 0, len(active))
	for key := range active {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].authID != keys[right].authID {
			return keys[left].authID < keys[right].authID
		}
		return keys[left].provider < keys[right].provider
	})
	return keys
}

// DeliverNotificationsOnce claims and resolves currently due notification batches.
func (s *Service) DeliverNotificationsOnce(ctx context.Context) error {
	claims, err := s.store.ClaimNotificationBatches(ctx, NotificationClaimOptions{Limit: DefaultNotificationClaimLimit, LeaseDuration: DefaultNotificationLease})
	if err != nil || len(claims) == 0 {
		return err
	}
	for _, claim := range claims {
		result := NotificationResult{BatchID: claim.Batch.ID(), LeaseID: claim.LeaseID}
		if s.sender == nil {
			result.PermanentFailure = true
			result.FailureCode = string(DeliverySenderUnavailable)
		} else if sendErr := s.sender.Send(ctx, claim.Batch); sendErr != nil {
			code, permanent := classifyDeliveryError(sendErr)
			result.FailureCode = string(code)
			if permanent || claim.Attempt >= MaxNotificationAttempts {
				result.PermanentFailure = true
			} else {
				result.RetryAt = s.clock.Now().Add(DefaultNotificationRetryDelay).UTC().Truncate(time.Microsecond)
			}
			log.WithError(sendErr).WithFields(log.Fields{
				"failure_code": code,
				"provider":     claim.Batch.Provider(),
				"batch_id":     claim.Batch.ID(),
			}).Warn("quota alert: notification delivery failed")
		} else {
			result.SentAt = s.clock.Now().UTC().Truncate(time.Microsecond)
		}
		if err = s.store.ResolveNotification(ctx, result); err != nil {
			return err
		}
	}
	return nil
}

// classifyDeliveryError maps sender errors onto the bounded delivery failure
// codes persisted on notification batches. Configuration and decryption
// problems are permanent — retrying cannot fix them; upstream send failures
// stay retryable until MaxNotificationAttempts.
func classifyDeliveryError(err error) (DeliveryFailureCode, bool) {
	switch {
	case errors.Is(err, ErrSenderUnavailable):
		return DeliverySenderUnavailable, true
	case errors.Is(err, ErrTelegramUnconfigured):
		return DeliveryTelegramUnconfigured, true
	case errors.Is(err, ErrTelegramUnavailable):
		return DeliveryTelegramUnavailable, true
	default:
		return DeliverySendFailed, false
	}
}

func (s *Service) run(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	pollTimer := time.NewTimer(0)
	deliveryTimer := time.NewTimer(0)
	defer pollTimer.Stop()
	defer deliveryTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
			if err := s.RunCollectionOnce(ctx); err != nil {
				log.WithError(err).Error("quota alert: collection cycle failed")
			}
			resetTimer(pollTimer, s.nextPollInterval(ctx))
		case <-pollTimer.C:
			if err := s.RunCollectionOnce(ctx); err != nil {
				log.WithError(err).Error("quota alert: collection cycle failed")
			}
			resetTimer(pollTimer, s.nextPollInterval(ctx))
		case <-deliveryTimer.C:
			if err := s.DeliverNotificationsOnce(ctx); err != nil {
				log.WithError(err).Error("quota alert: notification delivery cycle failed")
			}
			resetTimer(deliveryTimer, s.deliveryInterval)
		}
	}
}

// nextPollInterval computes the delay before the next scheduled collection
// cycle. When monitoring is disabled or state is unreadable it falls back to
// the configured base interval.
func (s *Service) nextPollInterval(ctx context.Context) time.Duration {
	settings, err := s.store.LoadSettings(ctx)
	if err != nil || !settings.Enabled {
		return s.pollInterval
	}
	states, err := s.listAllStates(ctx)
	if err != nil {
		return s.pollInterval
	}
	return adaptivePollInterval(s.clock.Now(), states, settings, s.pollInterval, MinPollInterval)
}

// adaptivePollInterval shortens the base interval while any reliable quota
// window is hot: its reset lands within one base interval (so recovery is
// observed promptly instead of a full interval late), or its remaining value
// sits within adaptiveThresholdHeadroom of the effective warning threshold.
// Unknown-health rows and disabled providers never shorten the interval.
func adaptivePollInterval(now time.Time, states []CurrentState, settings Settings, base, min time.Duration) time.Duration {
	fast := base / adaptivePollDivisor
	if fast < min {
		fast = min
	}
	configs := evaluationProviderSettings(settings)
	for _, state := range states {
		if state.Health != CollectionReliable {
			continue
		}
		config, ok := configs[state.Identity.Provider]
		if !ok || !config.enabled {
			continue
		}
		if state.RemainingKnown && state.Remaining <= config.threshold+adaptiveThresholdHeadroom {
			return fast
		}
		if state.ResetKnown {
			until := state.ResetAt.Sub(now)
			if until <= base && until > -base {
				return fast
			}
		}
	}
	return base
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}

func (s *Service) listAuths(ctx context.Context) ([]AuthSnapshot, error) {
	if s.authSource == nil {
		return nil, nil
	}
	return s.authSource.ListQuotaAlertAuths(ctx)
}

func (s *Service) listAllStates(ctx context.Context) ([]CurrentState, error) {
	var states []CurrentState
	cursor := ""
	for {
		page, err := s.store.ListStates(ctx, PageRequest{Cursor: cursor, Limit: MaxPageSize})
		if err != nil {
			return nil, err
		}
		for _, state := range page.Items {
			normalized, err := state.Normalize()
			if err != nil {
				return nil, err
			}
			states = append(states, normalized)
		}
		if page.NextCursor == "" {
			return states, nil
		}
		if page.NextCursor == cursor {
			return nil, fmt.Errorf("quota alert state pagination did not advance")
		}
		cursor = page.NextCursor
	}
}

type authProviderKey struct {
	authID   string
	provider Provider
}

type collectionCycle struct {
	observations []Observation
	observed     map[StateIdentity]struct{}
	failed       map[authProviderKey]struct{}
}

type collectionJob struct {
	index    int
	auth     AuthSnapshot
	key      authProviderKey
	previous []CurrentState
}

type collectionResult struct {
	index        int
	key          authProviderKey
	observations []Observation
	failed       bool
}

func activeAuthProviderKeys(settings Settings, auths []AuthSnapshot) map[authProviderKey]struct{} {
	providerSettings := evaluationProviderSettings(settings)
	active := make(map[authProviderKey]struct{}, len(auths))
	for _, auth := range auths {
		key, ok := authKey(auth)
		if !ok {
			continue
		}
		config, ok := providerSettings[key.provider]
		if ok && config.enabled {
			active[key] = struct{}{}
		}
	}
	return active
}

func authKey(auth AuthSnapshot) (authProviderKey, bool) {
	if auth == nil {
		return authProviderKey{}, false
	}
	identity, err := (StateIdentity{AuthID: auth.AuthID(), Provider: auth.Provider(), Resource: "collection", Window: "latest"}).Normalize()
	if err != nil {
		return authProviderKey{}, false
	}
	return authProviderKey{authID: identity.AuthID, provider: identity.Provider}, true
}

func stateKey(state CurrentState) authProviderKey {
	return authProviderKey{authID: state.Identity.AuthID, provider: state.Identity.Provider}
}

func previousStatesByAuthProvider(states []CurrentState) map[authProviderKey][]CurrentState {
	grouped := make(map[authProviderKey][]CurrentState)
	for _, state := range states {
		grouped[stateKey(state)] = append(grouped[stateKey(state)], state)
	}
	return grouped
}

func (s *Service) collectObservations(ctx context.Context, auths []AuthSnapshot, previous []CurrentState, active map[authProviderKey]struct{}) collectionCycle {
	cycle := collectionCycle{
		observed: make(map[StateIdentity]struct{}),
		failed:   make(map[authProviderKey]struct{}),
	}
	if len(auths) == 0 || len(active) == 0 {
		return cycle
	}
	previousByKey := previousStatesByAuthProvider(previous)
	jobs := make([]collectionJob, 0, len(auths))
	for _, auth := range auths {
		key, ok := authKey(auth)
		if !ok {
			continue
		}
		if _, ok = active[key]; !ok {
			continue
		}
		jobs = append(jobs, collectionJob{index: len(jobs), auth: auth, key: key, previous: previousByKey[key]})
	}
	if len(jobs) == 0 {
		return cycle
	}

	workerCount := min(DefaultCollectionWorkers, len(jobs))
	jobCh := make(chan collectionJob)
	resultCh := make(chan collectionResult, len(jobs))
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				observations, failed := s.collectAuthObservations(ctx, job.auth, job.previous)
				resultCh <- collectionResult{index: job.index, key: job.key, observations: observations, failed: failed}
			}
		}()
	}
	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)
	wg.Wait()
	close(resultCh)

	ordered := make([][]Observation, len(jobs))
	for result := range resultCh {
		ordered[result.index] = result.observations
		if result.failed {
			cycle.failed[result.key] = struct{}{}
		}
	}
	for _, observations := range ordered {
		for _, observation := range observations {
			if identity, err := observation.Identity.Normalize(); err == nil {
				cycle.observed[identity] = struct{}{}
			}
			cycle.observations = append(cycle.observations, observation)
		}
	}
	return cycle
}

func (s *Service) collectAuthObservations(ctx context.Context, auth AuthSnapshot, previous []CurrentState) ([]Observation, bool) {
	collector, err := s.collectorRegistry.Collector(auth.Provider(), s.collectorDeps)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"provider": auth.Provider(), "auth_id": auth.AuthID()}).
			Warn("quota alert: no collector for provider")
		return s.unknownObservations(auth, previous, FailureCollectorMissing), true
	}
	collectionCtx, cancel := context.WithTimeout(ctx, s.collectionTimeout)
	defer cancel()
	collected, err := collector.Collect(collectionCtx, auth)
	if err != nil {
		code := classifyCollectorError(err)
		log.WithError(err).WithFields(log.Fields{"provider": auth.Provider(), "auth_id": auth.AuthID(), "failure_code": code}).
			Warn("quota alert: collector failed")
		return s.unknownObservations(auth, previous, code), true
	}
	return collected, false
}

// classifyCollectorError maps collector errors onto the bounded sanitized
// failure-code enum. Raw error text is logged server-side only; the code is
// what reaches persisted state, APIs, and the UI.
func classifyCollectorError(err error) CollectionFailureCode {
	if err == nil {
		return FailureNone
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureTimeout
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "missing") && (strings.Contains(msg, "token") || strings.Contains(msg, "credential") || strings.Contains(msg, "account") || strings.Contains(msg, "project")):
		return FailureCredentialMissing
	case strings.Contains(msg, "refresh failed"):
		return FailureCredentialRejected
	case strings.Contains(msg, "http 401") || strings.Contains(msg, "http 403"):
		return FailureCredentialRejected
	case strings.Contains(msg, "http "):
		return FailureUpstreamHTTP
	case strings.Contains(msg, "no recognized windows") || strings.Contains(msg, "json") || strings.Contains(msg, "decode"):
		return FailureDecode
	case strings.Contains(msg, "deadline") || strings.Contains(msg, "timeout"):
		return FailureTimeout
	case strings.Contains(msg, "dial") || strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") || strings.Contains(msg, "eof"):
		return FailureTransport
	default:
		return FailureInternal
	}
}

func (s *Service) unknownObservations(auth AuthSnapshot, previous []CurrentState, code CollectionFailureCode) []Observation {
	observedAt := s.clock.Now()
	if len(previous) == 0 {
		return []Observation{unknownObservation(auth, observedAt, code)}
	}
	observations := make([]Observation, 0, len(previous))
	for _, state := range previous {
		observations = append(observations, Observation{
			Identity:    state.Identity,
			AuthLabel:   state.AuthLabel,
			Health:      CollectionUnknown,
			FailureCode: code,
			ObservedAt:  observedAt,
		})
	}
	return observations
}

func removedStateIdentities(previous []CurrentState, active map[authProviderKey]struct{}, collection collectionCycle) []StateIdentity {
	removed := make([]StateIdentity, 0)
	for _, state := range previous {
		key := stateKey(state)
		if _, ok := active[key]; !ok {
			removed = append(removed, state.Identity)
			continue
		}
		if _, failed := collection.failed[key]; failed {
			continue
		}
		if _, observed := collection.observed[state.Identity]; !observed {
			removed = append(removed, state.Identity)
		}
	}
	return removed
}

func unknownObservation(auth AuthSnapshot, observedAt time.Time, code CollectionFailureCode) Observation {
	provider := auth.Provider()
	if err := provider.Validate(); err != nil {
		provider = ProviderClaude
	}
	return Observation{
		Identity: StateIdentity{
			AuthID:   auth.AuthID(),
			Provider: provider,
			Resource: "collection",
			Window:   "latest",
		},
		AuthLabel:   auth.RedactedLabel(),
		Health:      CollectionUnknown,
		FailureCode: code,
		ObservedAt:  observedAt,
	}
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }
