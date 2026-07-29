package api

import (
	"context"
	"time"

	"github.com/therealtinhtute/llmhub/internal/quotaalert"
)

type apiQuotaAlertRouteStore struct {
	settings          quotaalert.Settings
	loadSettingsCount int
}

func (s *apiQuotaAlertRouteStore) LoadSettings(context.Context) (quotaalert.Settings, error) {
	s.loadSettingsCount++
	return s.settings, nil
}

func (s *apiQuotaAlertRouteStore) LoadSettingsWithSecret(context.Context) (quotaalert.Settings, *quotaalert.EncryptedSecret, error) {
	return s.settings, nil, nil
}

func (s *apiQuotaAlertRouteStore) SaveSettings(context.Context, int64, quotaalert.Settings) (quotaalert.Settings, error) {
	return s.settings, nil
}

func (s *apiQuotaAlertRouteStore) SaveSettingsWithSecret(context.Context, int64, quotaalert.Settings, quotaalert.SecretUpdate, *quotaalert.SecretCipher, string) (quotaalert.Settings, error) {
	return s.settings, nil
}

func (s *apiQuotaAlertRouteStore) TryAcquireCollection(context.Context) (quotaalert.CollectionLease, bool, error) {
	return nil, false, nil
}

func (s *apiQuotaAlertRouteStore) LoadStates(context.Context, []quotaalert.StateIdentity) ([]quotaalert.CurrentState, error) {
	return nil, nil
}

func (s *apiQuotaAlertRouteStore) CommitCollection(context.Context, quotaalert.CollectionLease, quotaalert.CollectionCommit) error {
	return nil
}

func (s *apiQuotaAlertRouteStore) ListStates(context.Context, quotaalert.PageRequest) (quotaalert.Page[quotaalert.CurrentState], error) {
	return quotaalert.Page[quotaalert.CurrentState]{}, nil
}

func (s *apiQuotaAlertRouteStore) ListEvents(context.Context, quotaalert.PageRequest) (quotaalert.Page[quotaalert.TransitionEvent], error) {
	return quotaalert.Page[quotaalert.TransitionEvent]{}, nil
}

func (s *apiQuotaAlertRouteStore) AcknowledgeEvent(context.Context, string, time.Time) error {
	return nil
}

func (s *apiQuotaAlertRouteStore) PruneEvents(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (s *apiQuotaAlertRouteStore) PruneNotificationBatches(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (s *apiQuotaAlertRouteStore) ClaimNotificationBatches(context.Context, quotaalert.NotificationClaimOptions) ([]quotaalert.NotificationClaim, error) {
	return nil, nil
}

func (s *apiQuotaAlertRouteStore) ResolveNotification(context.Context, quotaalert.NotificationResult) error {
	return nil
}
