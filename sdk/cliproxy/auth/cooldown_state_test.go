package auth

import (
	"testing"
	"time"
)

func TestCooldownStateRecordNormalize(t *testing.T) {
	fallback := time.Date(2026, time.July, 31, 3, 0, 0, 123456789, time.FixedZone("offset", 3600))
	retry := fallback.Add(time.Hour)
	recoverAt := fallback.Add(2 * time.Hour)
	record, err := (CooldownStateRecord{
		Provider:       " CODEX ",
		AuthID:         " auth-1 ",
		Model:          " gpt-test ",
		Status:         " error ",
		Reason:         " rate limited ",
		NextRetryAfter: retry,
		Quota:          QuotaState{NextRecoverAt: recoverAt},
	}).Normalize(fallback)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if record.Provider != "codex" || record.AuthID != "auth-1" || record.Model != "gpt-test" || record.Status != "error" || record.Reason != "rate limited" {
		t.Fatalf("normalized identity = %#v", record)
	}
	if record.UpdatedAt.Location() != time.UTC || record.UpdatedAt.Nanosecond()%1000 != 0 {
		t.Fatalf("UpdatedAt = %v, want UTC microsecond precision", record.UpdatedAt)
	}
	if record.NextRetryAfter.Location() != time.UTC || record.Quota.NextRecoverAt.Location() != time.UTC {
		t.Fatalf("normalized deadlines = retry %v recover %v", record.NextRetryAfter, record.Quota.NextRecoverAt)
	}
}

func TestCooldownStateRecordNormalizeRequiresAuthID(t *testing.T) {
	if _, err := (CooldownStateRecord{}).Normalize(time.Now()); err == nil {
		t.Fatal("Normalize() error = nil for empty auth ID")
	}
}

func TestCooldownStateRecordExpired(t *testing.T) {
	now := time.Date(2026, time.July, 31, 3, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name    string
		retry   time.Time
		expired bool
	}{
		{name: "zero", expired: true},
		{name: "past", retry: now.Add(-time.Second), expired: true},
		{name: "equal", retry: now, expired: true},
		{name: "future", retry: now.Add(time.Second), expired: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := (CooldownStateRecord{NextRetryAfter: test.retry}).Expired(now); got != test.expired {
				t.Fatalf("Expired() = %v, want %v", got, test.expired)
			}
		})
	}
}
