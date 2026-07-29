package quotaalert

import (
	"context"
	"fmt"
	"sync"
	"time"
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
	active := activeAuthProviderKeys(settings, auths)
	collection := s.collectObservations(ctx, auths, previous, active)
	result, err := EvaluateObservations(EvaluationInput{
		Settings:       settings,
		Observations:   collection.observations,
		PreviousStates: previous,
		EvaluatedAt:    s.clock.Now(),
	})
	if err != nil {
		return err
	}
	return s.store.CommitCollection(ctx, lease, CollectionCommit{
		SettingsRevision: settings.Revision,
		States:           result.States,
		RemovedStates:    removedStateIdentities(previous, active, collection),
		Events:           result.Events,
		Batches:          result.Batches,
	})
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
			result.FailureCode = "sender_unavailable"
		} else if sendErr := s.sender.Send(ctx, claim.Batch); sendErr != nil {
			result.FailureCode = "send_failed"
			if claim.Attempt >= MaxNotificationAttempts {
				result.PermanentFailure = true
			} else {
				result.RetryAt = s.clock.Now().Add(DefaultNotificationRetryDelay).UTC().Truncate(time.Microsecond)
			}
		} else {
			result.SentAt = s.clock.Now().UTC().Truncate(time.Microsecond)
		}
		if err = s.store.ResolveNotification(ctx, result); err != nil {
			return err
		}
	}
	return nil
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
			_ = s.RunCollectionOnce(ctx)
		case <-pollTimer.C:
			_ = s.RunCollectionOnce(ctx)
			resetTimer(pollTimer, s.pollInterval)
		case <-deliveryTimer.C:
			_ = s.DeliverNotificationsOnce(ctx)
			resetTimer(deliveryTimer, s.deliveryInterval)
		}
	}
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
		return s.unknownObservations(auth, previous), true
	}
	collectionCtx, cancel := context.WithTimeout(ctx, s.collectionTimeout)
	defer cancel()
	collected, err := collector.Collect(collectionCtx, auth)
	if err != nil {
		return s.unknownObservations(auth, previous), true
	}
	return collected, false
}

func (s *Service) unknownObservations(auth AuthSnapshot, previous []CurrentState) []Observation {
	observedAt := s.clock.Now()
	if len(previous) == 0 {
		return []Observation{unknownObservation(auth, observedAt)}
	}
	observations := make([]Observation, 0, len(previous))
	for _, state := range previous {
		observations = append(observations, Observation{
			Identity:   state.Identity,
			AuthLabel:  state.AuthLabel,
			Health:     CollectionUnknown,
			ObservedAt: observedAt,
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

func unknownObservation(auth AuthSnapshot, observedAt time.Time) Observation {
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
		AuthLabel:  auth.RedactedLabel(),
		Health:     CollectionUnknown,
		ObservedAt: observedAt,
	}
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }
