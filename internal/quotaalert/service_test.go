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
	for _, state := range store.lastCommit.States {
		alerts[state.Identity.Provider] = state.Alert
	}
	if alerts[ProviderClaude] != AlertUnknown || alerts[ProviderCodex] != AlertHealthy {
		t.Fatalf("alerts = %#v", alerts)
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
	if len(store.results) != 4 || store.results[3].RetryAt.IsZero() || store.results[3].PermanentFailure {
		t.Fatalf("unavailable result = %#v", store.results[3])
	}

	store.claims = []NotificationClaim{{Batch: batch, LeaseID: "lease-5", Attempt: 1}}
	sender.err = context.DeadlineExceeded
	if err = service.DeliverNotificationsOnce(context.Background()); err != nil {
		t.Fatalf("DeliverNotificationsOnce(timeout) error = %v", err)
	}
	if len(store.results) != 5 || store.results[4].RetryAt.IsZero() || store.results[4].PermanentFailure {
		t.Fatalf("timeout result = %#v", store.results[4])
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
func (s *serviceTestStore) CommitCollection(_ context.Context, _ CollectionLease, commit CollectionCommit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commitCount++
	s.lastCommit = commit
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
