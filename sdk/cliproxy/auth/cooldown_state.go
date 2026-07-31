package auth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CooldownStateRecord is a persisted runtime cooldown snapshot for one auth/model pair.
type CooldownStateRecord struct {
	Provider       string     `json:"provider,omitempty"`
	AuthID         string     `json:"auth_id"`
	AuthFile       string     `json:"-"`
	Model          string     `json:"model,omitempty"`
	Status         string     `json:"status,omitempty"`
	NextRetryAfter time.Time  `json:"next_retry_after"`
	Reason         string     `json:"reason,omitempty"`
	Quota          QuotaState `json:"quota,omitempty"`
	LastError      *Error     `json:"last_error,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Normalize validates the durable identity and canonicalizes persisted timestamps.
func (r CooldownStateRecord) Normalize(fallback time.Time) (CooldownStateRecord, error) {
	r.Provider = strings.ToLower(strings.TrimSpace(r.Provider))
	r.AuthID = strings.TrimSpace(r.AuthID)
	r.Model = strings.TrimSpace(r.Model)
	r.Status = strings.TrimSpace(r.Status)
	r.Reason = strings.TrimSpace(r.Reason)
	if r.AuthID == "" {
		return CooldownStateRecord{}, fmt.Errorf("cooldown state auth ID is required")
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = fallback
	}
	if !r.UpdatedAt.IsZero() {
		r.UpdatedAt = r.UpdatedAt.UTC().Truncate(time.Microsecond)
	}
	if !r.NextRetryAfter.IsZero() {
		r.NextRetryAfter = r.NextRetryAfter.UTC().Truncate(time.Microsecond)
	}
	if !r.Quota.NextRecoverAt.IsZero() {
		r.Quota.NextRecoverAt = r.Quota.NextRecoverAt.UTC().Truncate(time.Microsecond)
	}
	return r, nil
}

// Expired reports whether the record is no longer restorable at now.
func (r CooldownStateRecord) Expired(now time.Time) bool {
	return r.NextRetryAfter.IsZero() || !r.NextRetryAfter.After(now)
}

// CooldownStateStore persists runtime cooldown state independently from auth tokens.
type CooldownStateStore interface {
	Load(context.Context) ([]CooldownStateRecord, error)
	Save(context.Context, []CooldownStateRecord) error
}

// CooldownStateStoreProvider exposes a backend-specific cooldown state store.
type CooldownStateStoreProvider interface {
	CooldownStateStore() CooldownStateStore
}
