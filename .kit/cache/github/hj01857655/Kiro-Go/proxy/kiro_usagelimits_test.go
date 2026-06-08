package proxy

import (
	"encoding/json"
	"testing"
)

// TestUsageLimitsParsesEnterpriseISODate is the regression guard for the
// Enterprise account parse crash.
//
// Upstream returns nextDateReset (and freeTrialExpiry / bonus expiresAt) in TWO
// shapes depending on account/path: a scientific-notation epoch number
// (1.780272E9) AND an ISO-8601 string ("2026-06-01T00:00:00.000Z"). The struct
// used to type these as json.Number, which ONLY accepts numbers — so an ISO
// string made json.Unmarshal fail with "invalid number literal". That failure
// propagated up GetUsageLimits -> RefreshAccountInfo, whose ban-detection does
// strings.Contains(err, "invalid") and would then auto-disable + BAN the
// account. Switching these fields to json.RawMessage accepts any JSON value
// (number, string, null, missing) without loss.
//
// This body is the real Enterprise /getUsageLimits response the user provided.
func TestUsageLimitsParsesEnterpriseISODate(t *testing.T) {
	const enterpriseBody = `{"nextDateReset":"2026-06-01T00:00:00.000Z","overageConfiguration":{"overageStatus":"ENABLED"},"subscriptionInfo":{"overageCapability":"OVERAGE_CAPABLE","subscriptionManagementTarget":"MANAGE","subscriptionTitle":"KIRO PRO","type":"Q_DEVELOPER_STANDALONE_PRO","upgradeCapability":"UPGRADE_INCAPABLE"},"usageBreakdownList":[{"bonuses":[],"currency":"USD","currentOverages":311,"currentOveragesWithPrecision":311.45,"currentUsage":1311,"currentUsageWithPrecision":1311.45,"displayName":"Credit","displayNamePlural":"Credits","nextDateReset":"2026-06-01T00:00:00.000Z","overageCap":10000,"overageCapWithPrecision":10000,"overageCharges":12.458250431804,"overageRate":0.04,"resourceType":"CREDIT","unit":"INVOCATIONS","usageLimit":1000,"usageLimitWithPrecision":1000}],"userInfo":{"userId":"d-90660ceab3.548854a8-5041-707d-d194-dd8797af60e8"}}`

	var resp UsageLimitsResponse
	if err := json.Unmarshal([]byte(enterpriseBody), &resp); err != nil {
		t.Fatalf("Enterprise response must parse, got error: %v", err)
	}

	// Top-level ISO date preserved verbatim (as a quoted JSON string).
	if got := string(resp.NextDateReset); got != `"2026-06-01T00:00:00.000Z"` {
		t.Errorf("NextDateReset = %s, want quoted ISO string", got)
	}

	// Subscription fields intact.
	if resp.SubscriptionInfo == nil {
		t.Fatal("SubscriptionInfo is nil — fields were dropped")
	}
	if resp.SubscriptionInfo.Type != "Q_DEVELOPER_STANDALONE_PRO" {
		t.Errorf("Type = %q, want Q_DEVELOPER_STANDALONE_PRO", resp.SubscriptionInfo.Type)
	}
	if resp.SubscriptionInfo.OverageCapability != "OVERAGE_CAPABLE" {
		t.Errorf("OverageCapability = %q, want OVERAGE_CAPABLE", resp.SubscriptionInfo.OverageCapability)
	}

	// Overage switch (string-enum shape, the CURRENT upstream format).
	if resp.OverageConfiguration == nil || resp.OverageConfiguration.OverageStatus != "ENABLED" {
		t.Errorf("OverageConfiguration.OverageStatus not parsed as ENABLED: %+v", resp.OverageConfiguration)
	}

	// Breakdown numbers (overage accrual precision) intact.
	if len(resp.UsageBreakdownList) != 1 {
		t.Fatalf("UsageBreakdownList len = %d, want 1", len(resp.UsageBreakdownList))
	}
	bd := resp.UsageBreakdownList[0]
	if bd.CurrentUsageWithPrecision != 1311.45 {
		t.Errorf("CurrentUsageWithPrecision = %v, want 1311.45", bd.CurrentUsageWithPrecision)
	}
	if bd.OverageCharges != 12.458250431804 {
		t.Errorf("OverageCharges = %v, want 12.458250431804 (precision must survive)", bd.OverageCharges)
	}
	if bd.OverageRate != 0.04 {
		t.Errorf("OverageRate = %v, want 0.04", bd.OverageRate)
	}
	// Per-breakdown ISO date also preserved.
	if got := string(bd.NextDateReset); got != `"2026-06-01T00:00:00.000Z"` {
		t.Errorf("breakdown NextDateReset = %s, want quoted ISO string", got)
	}
}

// TestUsageLimitsParsesScientificNotationEpoch proves the OTHER upstream shape
// (epoch as a scientific-notation number) still parses after the switch to
// json.RawMessage — i.e. the fix does not regress the numeric form.
func TestUsageLimitsParsesScientificNotationEpoch(t *testing.T) {
	const numericBody = `{"nextDateReset":1.780272E9,"subscriptionInfo":{"type":"PRO"},"usageBreakdownList":[{"resourceType":"CREDIT","currentUsage":7,"usageLimit":100,"nextDateReset":1.780272E9}]}`

	var resp UsageLimitsResponse
	if err := json.Unmarshal([]byte(numericBody), &resp); err != nil {
		t.Fatalf("numeric epoch response must parse, got error: %v", err)
	}
	if got := string(resp.NextDateReset); got != "1.780272E9" {
		t.Errorf("NextDateReset = %s, want raw number 1.780272E9", got)
	}
	if resp.SubscriptionInfo == nil || resp.SubscriptionInfo.Type != "PRO" {
		t.Errorf("SubscriptionInfo.Type not parsed: %+v", resp.SubscriptionInfo)
	}
}

// TestUsageLimitsParsesFreeTrialISODate covers the FREE account shape from the
// archived samples (Google/BuilderId): freeTrialInfo.freeTrialExpiry is an ISO
// string, which json.Number would also have rejected.
func TestUsageLimitsParsesFreeTrialISODate(t *testing.T) {
	const freeBody = `{"nextDateReset":"2026-02-01T00:00:00+00:00","usageBreakdownList":[{"resourceType":"CREDIT","currentUsage":50,"usageLimit":50,"freeTrialInfo":{"currentUsage":500,"usageLimit":500,"freeTrialStatus":"ACTIVE","freeTrialExpiry":"2026-01-31T06:52:04.970000+00:00"},"nextDateReset":"2026-02-01T00:00:00+00:00"}]}`

	var resp UsageLimitsResponse
	if err := json.Unmarshal([]byte(freeBody), &resp); err != nil {
		t.Fatalf("FREE response with ISO freeTrialExpiry must parse, got error: %v", err)
	}
	if len(resp.UsageBreakdownList) != 1 || resp.UsageBreakdownList[0].FreeTrialInfo == nil {
		t.Fatalf("FreeTrialInfo dropped: %+v", resp.UsageBreakdownList)
	}
	ft := resp.UsageBreakdownList[0].FreeTrialInfo
	if ft.FreeTrialStatus != "ACTIVE" {
		t.Errorf("FreeTrialStatus = %q, want ACTIVE", ft.FreeTrialStatus)
	}
	if got := string(ft.FreeTrialExpiry); got != `"2026-01-31T06:52:04.970000+00:00"` {
		t.Errorf("FreeTrialExpiry = %s, want quoted ISO string", got)
	}
}
