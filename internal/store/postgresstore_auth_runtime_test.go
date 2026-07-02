package store

import (
	"testing"
	"time"

	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestPostgresAuthPayloadRoundTripsRuntimeCooldownState(t *testing.T) {
	s := &PostgresStore{}
	next := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	auth := &cliproxyauth.Auth{
		ID:             "cooldown-auth",
		Provider:       "gemini",
		FileName:       "cooldown-auth.json",
		Status:         cliproxyauth.StatusError,
		StatusMessage:  "quota exhausted",
		Unavailable:    true,
		NextRetryAfter: next,
		Quota:          cliproxyauth.QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: next, BackoffLevel: 3},
		Metadata: map[string]any{
			"type":  "gemini",
			"email": "cooldown@example.com",
		},
		ModelStates: map[string]*cliproxyauth.ModelState{
			"gemini-cooldown-model": {
				Status:         cliproxyauth.StatusError,
				StatusMessage:  "quota exhausted",
				Unavailable:    true,
				NextRetryAfter: next,
				Quota:          cliproxyauth.QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: next, BackoffLevel: 3},
			},
		},
	}

	payload, err := s.authPayload(auth)
	if err != nil {
		t.Fatalf("authPayload() error = %v", err)
	}
	roundTripped, err := authFromPayload(auth.ID, auth.Provider, payload, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("authFromPayload() error = %v", err)
	}

	if !roundTripped.Quota.Exceeded || roundTripped.Quota.Reason != "quota" || roundTripped.Quota.BackoffLevel != 3 {
		t.Fatalf("round-tripped auth quota = %+v, want persisted quota", roundTripped.Quota)
	}
	if roundTripped.NextRetryAfter.IsZero() || !roundTripped.Unavailable {
		t.Fatalf("round-tripped auth availability unavailable=%v next=%v", roundTripped.Unavailable, roundTripped.NextRetryAfter)
	}
	state := roundTripped.ModelStates["gemini-cooldown-model"]
	if state == nil {
		t.Fatalf("round-tripped model state missing")
	}
	if !state.Quota.Exceeded || state.Quota.Reason != "quota" || state.Quota.BackoffLevel != 3 || state.NextRetryAfter.IsZero() {
		t.Fatalf("round-tripped model state = %+v next=%v, want persisted cooldown", state.Quota, state.NextRetryAfter)
	}
}
