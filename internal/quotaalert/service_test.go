package quotaalert

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceRunCollectionOnceCommitsEvaluatedObservations(t *testing.T) {
	now := time.Date(2026, time.July, 29, 6, 0, 0, 0, time.UTC)
	store := newServiceTestStore()
	store.settings.Enabled = true
	store.settings.Revision = 11
	clock := &serviceTestClock{now: now}
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			return []Observation{serviceTestObservation("auth-1", ProviderClaude, 5, now)}, nil
		}), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service, err := NewService(ServiceConfig{
		Store:             store,
		AuthSource:        serviceTestAuthSource{auths: []AuthSnapshot{serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Primary"}}},
		CollectorRegistry: registry,
		Clock:             clock,
		CollectionTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if err = service.RunCollectionOnce(context.Background()); err != nil {
		t.Fatalf("RunCollectionOnce() error = %v", err)
	}
	if store.acquireCount != 1 || store.commitCount != 1 || store.releaseCount != 1 {
		t.Fatalf("store counts acquire=%d commit=%d release=%d", store.acquireCount, store.commitCount, store.releaseCount)
	}
	if len(store.lastCommit.States) != 1 || store.lastCommit.States[0].Alert != AlertWarning {
		t.Fatalf("committed states = %#v", store.lastCommit.States)
	}
	if len(store.lastCommit.Events) != 1 || store.lastCommit.Events[0].Kind != TransitionWarning {
		t.Fatalf("committed events = %#v", store.lastCommit.Events)
	}
	if len(store.lastCommit.Batches) != 1 || store.lastCommit.Batches[0].Provider() != ProviderClaude {
		t.Fatalf("committed batches = %#v", store.lastCommit.Batches)
	}
}

func TestServiceRunCollectionOnceSkipsWhenDisabledOrLeaseUnavailable(t *testing.T) {
	store := newServiceTestStore()
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			t.Fatal("collector should not run")
			return nil, nil
		}), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service, err := NewService(ServiceConfig{Store: store, AuthSource: serviceTestAuthSource{auths: []AuthSnapshot{serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Primary"}}}, CollectorRegistry: registry})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err = service.RunCollectionOnce(context.Background()); err != nil {
		t.Fatalf("RunCollectionOnce(disabled) error = %v", err)
	}
	if store.acquireCount != 0 || store.commitCount != 0 {
		t.Fatalf("disabled counts acquire=%d commit=%d", store.acquireCount, store.commitCount)
	}

	store.settings.Enabled = true
	store.acquireAvailable = false
	if err = service.RunCollectionOnce(context.Background()); err != nil {
		t.Fatalf("RunCollectionOnce(no lease) error = %v", err)
	}
	if store.commitCount != 0 {
		t.Fatalf("commit count = %d, want 0", store.commitCount)
	}
}

func TestServiceCollectionIsolatesProviderFailure(t *testing.T) {
	now := time.Date(2026, time.July, 29, 6, 10, 0, 0, time.UTC)
	store := newServiceTestStore()
	store.settings.Enabled = true
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			return nil, errors.New("provider failed")
		}), nil
	}); err != nil {
		t.Fatalf("Register(claude) error = %v", err)
	}
	if err := registry.Register(ProviderCodex, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			return []Observation{serviceTestObservation("auth-2", ProviderCodex, 50, now)}, nil
		}), nil
	}); err != nil {
		t.Fatalf("Register(codex) error = %v", err)
	}
	service, err := NewService(ServiceConfig{
		Store: store,
		AuthSource: serviceTestAuthSource{auths: []AuthSnapshot{
			serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Claude"},
			serviceTestAuth{id: "auth-2", provider: ProviderCodex, label: "Codex"},
		}},
		CollectorRegistry: registry,
		Clock:             &serviceTestClock{now: now},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err = service.RunCollectionOnce(context.Background()); err != nil {
		t.Fatalf("RunCollectionOnce() error = %v", err)
	}
	if len(store.lastCommit.States) != 2 {
		t.Fatalf("states = %#v", store.lastCommit.States)
	}
	alerts := map[Provider]AlertState{}
	codes := map[Provider]CollectionFailureCode{}
	for _, state := range store.lastCommit.States {
		alerts[state.Identity.Provider] = state.Alert
		codes[state.Identity.Provider] = state.FailureCode
	}
	if alerts[ProviderClaude] != AlertUnknown || alerts[ProviderCodex] != AlertHealthy {
		t.Fatalf("alerts = %#v", alerts)
	}
	if codes[ProviderClaude] != FailureInternal || codes[ProviderCodex] != FailureNone {
		t.Fatalf("failure codes = %#v", codes)
	}
}

func TestServiceCollectionUsesPerAuthTimeout(t *testing.T) {
	now := time.Date(2026, time.July, 29, 6, 15, 0, 0, time.UTC)
	store := newServiceTestStore()
	store.settings.Enabled = true
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(ctx context.Context, _ AuthSnapshot) ([]Observation, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}), nil
	}); err != nil {
		t.Fatalf("Register(claude) error = %v", err)
	}
	if err := registry.Register(ProviderCodex, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(ctx context.Context, _ AuthSnapshot) ([]Observation, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return []Observation{serviceTestObservation("auth-2", ProviderCodex, 50, now)}, nil
		}), nil
	}); err != nil {
		t.Fatalf("Register(codex) error = %v", err)
	}
	service, err := NewService(ServiceConfig{
		Store: store,
		AuthSource: serviceTestAuthSource{auths: []AuthSnapshot{
			serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Claude"},
			serviceTestAuth{id: "auth-2", provider: ProviderCodex, label: "Codex"},
		}},
		CollectorRegistry: registry,
		Clock:             &serviceTestClock{now: now},
		CollectionTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err = service.RunCollectionOnce(context.Background()); err != nil {
		t.Fatalf("RunCollectionOnce() error = %v", err)
	}
	alerts := map[Provider]AlertState{}
	for _, state := range store.lastCommit.States {
		alerts[state.Identity.Provider] = state.Alert
	}
	if alerts[ProviderClaude] != AlertUnknown || alerts[ProviderCodex] != AlertHealthy {
		t.Fatalf("alerts = %#v", alerts)
	}
}

func TestServiceRunCollectionOnceRemovesStaleStates(t *testing.T) {
	now := time.Date(2026, time.July, 29, 6, 18, 0, 0, time.UTC)
	store := newServiceTestStore()
	store.settings.Enabled = true
	store.states = []CurrentState{
		serviceTestState("auth-1", ProviderClaude, "messages", "weekly", AlertWarning, now.Add(-time.Hour)),
		serviceTestState("auth-1", ProviderClaude, "tokens", "monthly", AlertWarning, now.Add(-time.Hour)),
		serviceTestState("auth-2", ProviderCodex, "messages", "weekly", AlertExhausted, now.Add(-time.Hour)),
	}
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			return []Observation{serviceTestObservation("auth-1", ProviderClaude, 80, now)}, nil
		}), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service, err := NewService(ServiceConfig{
		Store:             store,
		AuthSource:        serviceTestAuthSource{auths: []AuthSnapshot{serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Claude"}}},
		CollectorRegistry: registry,
		Clock:             &serviceTestClock{now: now},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err = service.RunCollectionOnce(context.Background()); err != nil {
		t.Fatalf("RunCollectionOnce() error = %v", err)
	}
	removed := map[StateIdentity]struct{}{}
	for _, identity := range store.lastCommit.RemovedStates {
		removed[identity] = struct{}{}
	}
	for _, want := range []StateIdentity{
		{AuthID: "auth-1", Provider: ProviderClaude, Resource: "tokens", Window: "monthly"},
		{AuthID: "auth-2", Provider: ProviderCodex, Resource: "messages", Window: "weekly"},
	} {
		if _, ok := removed[want]; !ok {
			t.Fatalf("removed states = %#v, missing %#v", store.lastCommit.RemovedStates, want)
		}
	}
	if len(store.lastCommit.RemovedStates) != 2 {
		t.Fatalf("removed states = %#v", store.lastCommit.RemovedStates)
	}
}

func TestServiceMonitorDegradedSignalCrossesThresholdOncePerStreak(t *testing.T) {
	now := time.Date(2026, time.July, 29, 7, 0, 0, 0, time.UTC)
	store := newServiceTestStore()
	store.settings.Enabled = true
	store.settings.DegradedFailureThreshold = 3
	clock := &serviceTestClock{now: now}
	reliable := false
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			if reliable {
				return []Observation{serviceTestObservation("auth-1", ProviderClaude, 80, clock.Now())}, nil
			}
			return nil, errors.New("http 500: upstream exploded")
		}), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service, err := NewService(ServiceConfig{
		Store:             store,
		AuthSource:        serviceTestAuthSource{auths: []AuthSnapshot{serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Primary"}}},
		CollectorRegistry: registry,
		Clock:             clock,
		CollectionTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	runCycle := func() CollectionCommit {
		clock.now = clock.now.Add(time.Minute)
		if err := service.RunCollectionOnce(context.Background()); err != nil {
			t.Fatalf("RunCollectionOnce() error = %v", err)
		}
		return store.lastCommit
	}
	degradedCount := func(commit CollectionCommit) int {
		count := 0
		for _, event := range commit.Events {
			if event.Kind == TransitionMonitorDegraded {
				count++
			}
		}
		return count
	}
	healthOf := func() int {
		for _, record := range store.health {
			if record.Key.AuthID == "auth-1" && record.Key.Provider == ProviderClaude {
				return record.ConsecutiveFailures
			}
		}
		return 0
	}

	if commit := runCycle(); degradedCount(commit) != 0 || healthOf() != 1 {
		t.Fatalf("cycle 1: degraded=%d health=%d, want 0/1", degradedCount(commit), healthOf())
	}
	if commit := runCycle(); degradedCount(commit) != 0 || healthOf() != 2 {
		t.Fatalf("cycle 2: degraded=%d health=%d, want 0/2", degradedCount(commit), healthOf())
	}
	commit3 := runCycle()
	if degradedCount(commit3) != 1 || healthOf() != 3 {
		t.Fatalf("cycle 3: degraded=%d health=%d, want 1/3", degradedCount(commit3), healthOf())
	}
	var degradedEvent TransitionEvent
	for _, event := range commit3.Events {
		if event.Kind == TransitionMonitorDegraded {
			degradedEvent = event
		}
	}
	if degradedEvent.Identity.Resource != "collection" || degradedEvent.Identity.Window != "latest" ||
		degradedEvent.From != AlertUnknown || degradedEvent.To != AlertUnknown {
		t.Fatalf("degraded event identity = %#v", degradedEvent)
	}
	stateMatch := false
	for _, state := range commit3.States {
		if state.Identity == degradedEvent.Identity {
			stateMatch = true
		}
	}
	if !stateMatch {
		t.Fatalf("degraded event has no matching committed state: %#v", commit3.States)
	}
	batched := false
	for _, batch := range commit3.Batches {
		for _, event := range batch.Events() {
			if event.ID == degradedEvent.ID {
				batched = true
			}
		}
	}
	if !batched {
		t.Fatalf("degraded event missing from notification batches: %#v", commit3.Batches)
	}
	if commit := runCycle(); degradedCount(commit) != 0 || healthOf() != 4 {
		t.Fatalf("cycle 4 repeated the one-shot: degraded=%d health=%d", degradedCount(commit), healthOf())
	}

	reliable = true
	commit5 := runCycle()
	if healthOf() != 0 {
		t.Fatalf("reliable cycle did not reset the streak: health=%d", healthOf())
	}
	deleteMatch := false
	for _, key := range commit5.HealthDeletes {
		if key.AuthID == "auth-1" && key.Provider == ProviderClaude {
			deleteMatch = true
		}
	}
	if !deleteMatch {
		t.Fatalf("reliable cycle missing health delete: %#v", commit5.HealthDeletes)
	}

	reliable = false
	runCycle()
	runCycle()
	if commit := runCycle(); degradedCount(commit) != 1 {
		t.Fatalf("re-armed streak did not fire again: degraded=%d health=%d", degradedCount(commit), healthOf())
	}
}

func TestServiceMonitorDegradedDisabledNeverFires(t *testing.T) {
	now := time.Date(2026, time.July, 29, 7, 30, 0, 0, time.UTC)
	store := newServiceTestStore()
	store.settings.Enabled = true
	store.settings.DegradedFailureThreshold = 0
	clock := &serviceTestClock{now: now}
	registry := NewCollectorRegistry()
	if err := registry.Register(ProviderClaude, func(CollectorDependencies) (Collector, error) {
		return CollectFunc(func(context.Context, AuthSnapshot) ([]Observation, error) {
			return nil, errors.New("http 500")
		}), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service, err := NewService(ServiceConfig{
		Store:             store,
		AuthSource:        serviceTestAuthSource{auths: []AuthSnapshot{serviceTestAuth{id: "auth-1", provider: ProviderClaude, label: "Primary"}}},
		CollectorRegistry: registry,
		Clock:             clock,
		CollectionTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	for cycle := 0; cycle < 5; cycle++ {
		clock.now = clock.now.Add(time.Minute)
		if err := service.RunCollectionOnce(context.Background()); err != nil {
			t.Fatalf("cycle %d RunCollectionOnce() error = %v", cycle, err)
		}
		for _, event := range store.lastCommit.Events {
			if event.Kind == TransitionMonitorDegraded {
				t.Fatalf("monitor_degraded fired with threshold=0 on cycle %d: %#v", cycle, event)
			}
		}
	}
	if len(store.health) != 1 || store.health[0].ConsecutiveFailures != 5 {
		t.Fatalf("streak counter wrong with signal disabled: %#v", store.health)
	}
}

func TestServiceDeliverNotificationsResolvesSentRetryAndPermanentFailure(t *testing.T) {
	now := time.Date(2026, time.July, 29, 6, 20, 0, 0, time.UTC)
	batch := telegramTestBatch(t, now, []TransitionEvent{telegramTestEvent("event-1", ProviderClaude, TransitionWarning, AlertHealthy, AlertWarning, 5, now)})
	store := newServiceTestStore()
	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-1", Attempt: 1}}
	sender := &serviceTestSender{}
	service, err := NewService(ServiceConfig{Store: store, Sender: sender, Clock: &serviceTestClock{now: now}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(sent) error = %v", err)
	}
	if len(sender.sent) != 1 || len(store.results) != 1 || store.results[0].SentAt.IsZero() {
		t.Fatalf("sent=%d results=%#v", len(sender.sent), store.results)
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-2", Attempt: 1}}
	sender.err = errors.New("temporary outage")
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(retry) error = %v", err)
	}
	if len(store.results) != 2 || store.results[1].RetryAt.IsZero() || store.results[1].PermanentFailure {
		t.Fatalf("retry result = %#v", store.results[1])
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-3", Attempt: MaxNotificationAttempts}}
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(permanent) error = %v", err)
	}
	if len(store.results) != 3 || !store.results[2].PermanentFailure || store.results[2].FailureCode != "send_failed" {
		t.Fatalf("permanent result = %#v", store.results[2])
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-4", Attempt: 1}}
	sender.err = ErrTelegramUnavailable
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(unavailable) error = %v", err)
	}
	if len(store.results) != 4 || !store.results[3].PermanentFailure || store.results[3].FailureCode != "telegram_unavailable" {
		t.Fatalf("unavailable result = %#v", store.results[3])
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-5", Attempt: 1}}
	sender.err = ErrTelegramUnconfigured
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(unconfigured) error = %v", err)
	}
	if len(store.results) != 5 || !store.results[4].PermanentFailure || store.results[4].FailureCode != "telegram_unconfigured" {
		t.Fatalf("unconfigured result = %#v", store.results[4])
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-6", Attempt: 1}}
	sender.err = ErrSenderUnavailable
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(sender unavailable) error = %v", err)
	}
	if len(store.results) != 6 || !store.results[5].PermanentFailure || store.results[5].FailureCode != "sender_unavailable" {
		t.Fatalf("sender unavailable result = %#v", store.results[5])
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-7", Attempt: 1}}
	sender.err = context.DeadlineExceeded
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(timeout) error = %v", err)
	}
	if len(store.results) != 7 || store.results[6].RetryAt.IsZero() || store.results[6].PermanentFailure || store.results[6].FailureCode != "send_failed" {
		t.Fatalf("timeout result = %#v", store.results[6])
	}
}

func TestServiceStopBeforeStartDoesNotBlock(t *testing.T) {
	service, err := NewService(ServiceConfig{Store: newServiceTestStore()})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	done := make(chan struct{})
	go func() {
		service.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Stop before Start blocked")
	}
}

func TestServiceStartStopAndWakeAreIdempotent(t *testing.T) {
	store := newServiceTestStore()
	store.settings.Enabled = true
	service, err := NewService(ServiceConfig{
		Store:             store,
		AuthSource:        serviceTestAuthSource{},
		CollectorRegistry: NewCollectorRegistry(),
		PollInterval:      time.Hour,
		DeliveryInterval:  time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.Start(context.Background())
	service.Start(context.Background())
	service.Wake()
	service.Stop()
	service.Stop()
}

type serviceTestStore struct {
	mu               sync.Mutex
	settings         Settings
	acquireAvailable bool
	acquireCount     int
	commitCount      int
	releaseCount     int
	lastCommit       CollectionCommit
	states           []CurrentState
	health           []CollectionHealthRecord
	claims           []NotificationClaim
	results          []NotificationResult
	secret           *EncryptedSecret
}

func newServiceTestStore() *serviceTestStore {
	settings := DefaultSettings()
	settings.Revision = 1
	return &serviceTestStore{settings: settings, acquireAvailable: true}
}

func (s *serviceTestStore) LoadSettings(context.Context) (Settings, error) { return s.settings, nil }
func (s *serviceTestStore) LoadSettingsWithSecret(context.Context) (Settings, *EncryptedSecret, error) {
	return s.settings, s.secret, nil
}
func (s *serviceTestStore) SaveSettings(context.Context, int64, Settings) (Settings, error) {
	return Settings{}, nil
}
func (s *serviceTestStore) SaveSettingsWithSecret(context.Context, int64, Settings, SecretUpdate, *SecretCipher, string) (Settings, error) {
	return Settings{}, nil
}
func (s *serviceTestStore) TryAcquireCollection(context.Context) (CollectionLease, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acquireCount++
	if !s.acquireAvailable {
		return nil, false, nil
	}
	return serviceTestLease{store: s}, true, nil
}
func (s *serviceTestStore) LoadStates(context.Context, []StateIdentity) ([]CurrentState, error) {
	return nil, nil
}
func (s *serviceTestStore) ListCollectionHealth(context.Context) ([]CollectionHealthRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]CollectionHealthRecord(nil), s.health...), nil
}
func (s *serviceTestStore) CommitCollection(_ context.Context, _ CollectionLease, commit CollectionCommit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commitCount++
	s.lastCommit = commit
	s.states = commit.States
	deleteKeys := make(map[CollectionHealthKey]struct{}, len(commit.HealthDeletes))
	for _, key := range commit.HealthDeletes {
		deleteKeys[key] = struct{}{}
	}
	kept := s.health[:0]
	for _, record := range s.health {
		if _, deleted := deleteKeys[record.Key]; !deleted {
			kept = append(kept, record)
		}
	}
	s.health = kept
	for _, record := range commit.HealthUpserts {
		replaced := false
		for index, existing := range s.health {
			if existing.Key == record.Key {
				s.health[index] = record
				replaced = true
				break
			}
		}
		if !replaced {
			s.health = append(s.health, record)
		}
	}
	return nil
}
func (s *serviceTestStore) ListStates(context.Context, PageRequest) (Page[CurrentState], error) {
	return Page[CurrentState]{Items: append([]CurrentState(nil), s.states...)}, nil
}
func (s *serviceTestStore) ListEvents(context.Context, PageRequest) (Page[TransitionEvent], error) {
	return Page[TransitionEvent]{}, nil
}
func (s *serviceTestStore) AcknowledgeEvent(context.Context, string, time.Time) error  { return nil }
func (s *serviceTestStore) PruneEvents(context.Context, time.Time, int) (int64, error) { return 0, nil }
func (s *serviceTestStore) PruneNotificationBatches(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}
func (s *serviceTestStore) ClaimNotificationBatches(context.Context, NotificationClaimOptions) ([]NotificationClaim, error) {
	claims := s.claims
	s.claims = nil
	return claims, nil
}
func (s *serviceTestStore) ResolveNotification(_ context.Context, result NotificationResult) error {
	s.results = append(s.results, result)
	return nil
}

type serviceTestLease struct{ store *serviceTestStore }

func (l serviceTestLease) Release(context.Context) error {
	l.store.mu.Lock()
	defer l.store.mu.Unlock()
	l.store.releaseCount++
	return nil
}

type serviceTestAuthSource struct{ auths []AuthSnapshot }

func (s serviceTestAuthSource) ListQuotaAlertAuths(context.Context) ([]AuthSnapshot, error) {
	return s.auths, nil
}

type serviceTestAuth struct {
	id       string
	provider Provider
	label    string
}

func (a serviceTestAuth) AuthID() string        { return a.id }
func (a serviceTestAuth) Provider() Provider    { return a.provider }
func (a serviceTestAuth) RedactedLabel() string { return a.label }
func (a serviceTestAuth) ProxyURL() string      { return "" }
func (a serviceTestAuth) Attribute(string) (string, bool) {
	return "", false
}
func (a serviceTestAuth) Metadata(string) (any, bool) { return nil, false }

type serviceTestClock struct{ now time.Time }

func (c *serviceTestClock) Now() time.Time { return c.now }

type serviceTestSender struct {
	sent []NotificationBatch
	err  error
}

func (s *serviceTestSender) Send(_ context.Context, batch NotificationBatch) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, batch)
	return nil
}

func serviceTestObservation(authID string, provider Provider, remaining Percentage, observedAt time.Time) Observation {
	return Observation{
		Identity:       StateIdentity{AuthID: authID, Provider: provider, Resource: "messages", Window: "weekly"},
		AuthLabel:      authID + " label",
		Health:         CollectionReliable,
		Remaining:      remaining,
		RemainingKnown: true,
		ObservedAt:     observedAt,
	}
}

func serviceTestState(authID string, provider Provider, resource string, window string, alert AlertState, now time.Time) CurrentState {
	state := CurrentState{
		Identity:       StateIdentity{AuthID: authID, Provider: provider, Resource: resource, Window: window},
		AuthLabel:      authID + " label",
		Alert:          alert,
		Health:         CollectionReliable,
		Remaining:      50,
		RemainingKnown: true,
		ObservedAt:     now,
		TransitionedAt: now,
		UpdatedAt:      now,
		Revision:       1,
	}
	if alert == AlertExhausted {
		state.Remaining = 0
	}
	return state
}

func TestAdaptivePollIntervalShortensNearThresholdAndReset(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	base := 5 * time.Minute
	fast := time.Minute // base/5 = 1m, equals MinPollInterval floor
	settings := Settings{WarningThreshold: 10}

	hot := serviceTestState("a1", ProviderClaude, "messages", "weekly", AlertHealthy, now)
	hot.Remaining = 18 // within threshold 10 + headroom 10
	if got := adaptivePollInterval(now, []CurrentState{hot}, settings, base, MinPollInterval); got != fast {
		t.Fatalf("near-threshold interval = %s, want %s", got, fast)
	}

	cool := serviceTestState("a1", ProviderClaude, "messages", "weekly", AlertHealthy, now)
	cool.Remaining = 21 // just outside headroom
	if got := adaptivePollInterval(now, []CurrentState{cool}, settings, base, MinPollInterval); got != base {
		t.Fatalf("cool interval = %s, want base %s", got, base)
	}

	nearReset := cool
	nearReset.ResetAt = now.Add(4 * time.Minute)
	nearReset.ResetKnown = true
	if got := adaptivePollInterval(now, []CurrentState{nearReset}, settings, base, MinPollInterval); got != fast {
		t.Fatalf("near-reset interval = %s, want %s", got, fast)
	}

	farReset := cool
	farReset.ResetAt = now.Add(6 * time.Minute)
	farReset.ResetKnown = true
	if got := adaptivePollInterval(now, []CurrentState{farReset}, settings, base, MinPollInterval); got != base {
		t.Fatalf("far-reset interval = %s, want base %s", got, base)
	}

	staleReset := cool
	staleReset.ResetAt = now.Add(-2 * base)
	staleReset.ResetKnown = true
	if got := adaptivePollInterval(now, []CurrentState{staleReset}, settings, base, MinPollInterval); got != base {
		t.Fatalf("stale-reset interval = %s, want base %s", got, base)
	}

	unknown := hot
	unknown.Health = CollectionUnknown
	if got := adaptivePollInterval(now, []CurrentState{unknown}, settings, base, MinPollInterval); got != base {
		t.Fatalf("unknown-health interval = %s, want base %s", got, base)
	}

	disabled := Settings{WarningThreshold: 10, ProviderOverrides: []ProviderOverride{{Provider: ProviderClaude, Enabled: false}}}
	if got := adaptivePollInterval(now, []CurrentState{hot}, disabled, base, MinPollInterval); got != base {
		t.Fatalf("disabled-provider interval = %s, want base %s", got, base)
	}

	override := Settings{WarningThreshold: 10, ProviderOverrides: []ProviderOverride{{Provider: ProviderClaude, Enabled: true, WarningThreshold: percentagePtr(50)}}}
	borderline := serviceTestState("a1", ProviderClaude, "messages", "weekly", AlertHealthy, now)
	borderline.Remaining = 55 // within override threshold 50 + headroom 10, outside global 10 + 10
	if got := adaptivePollInterval(now, []CurrentState{borderline}, override, base, MinPollInterval); got != fast {
		t.Fatalf("override-threshold interval = %s, want %s", got, fast)
	}
	if got := adaptivePollInterval(now, []CurrentState{borderline}, settings, base, MinPollInterval); got != base {
		t.Fatalf("global-threshold interval = %s, want base %s", got, base)
	}
}

func percentagePtr(p Percentage) *Percentage { return &p }
