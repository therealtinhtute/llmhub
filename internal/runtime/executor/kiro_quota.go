package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/therealtinhtute/llmhub/internal/auth/kiro"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor/helps"
	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

const (
	kiroUsageLimitsPath             = "/getUsageLimits"
	kiroCodeWhispererUsageBaseURL   = "https://codewhisperer.us-east-1.amazonaws.com"
	kiroCodeWhispererUsageLimitsURL = kiroCodeWhispererUsageBaseURL + kiroUsageLimitsPath
	kiroCodeWhispererGetUsageTarget = "AmazonCodeWhispererService.GetUsageLimits"
	kiroProviderQuotaKey            = "kiro_quota"
	kiroProviderQuotaReason         = "provider_quota"
)

// KiroQuotaRow is one normalized provider quota bucket from Kiro getUsageLimits.
type KiroQuotaRow struct {
	ID                string     `json:"id"`
	ResourceType      string     `json:"resource_type"`
	Name              string     `json:"name"`
	Current           *float64   `json:"current,omitempty"`
	Limit             *float64   `json:"limit,omitempty"`
	Used              *float64   `json:"used,omitempty"`
	Total             *float64   `json:"total,omitempty"`
	Remaining         *float64   `json:"remaining,omitempty"`
	Percent           *float64   `json:"percent,omitempty"`
	RemainingPercent  *float64   `json:"remaining_percent,omitempty"`
	ResetAt           *time.Time `json:"reset_at,omitempty"`
	Unlimited         bool       `json:"unlimited"`
	FreeTrial         bool       `json:"free_trial,omitempty"`
	TrialStatus       string     `json:"trial_status,omitempty"`
	SubscriptionTitle string     `json:"subscription_title,omitempty"`
	SubscriptionType  string     `json:"subscription_type,omitempty"`
	OverageStatus     string     `json:"overage_status,omitempty"`
	OverageCap        *float64   `json:"overage_cap,omitempty"`
	OverageRate       *float64   `json:"overage_rate,omitempty"`
	CurrentOverages   *float64   `json:"current_overages,omitempty"`
}

// KiroQuotaState is the normalized account quota snapshot persisted on auth metadata.
type KiroQuotaState struct {
	ProviderQuotaAvailable bool           `json:"provider_quota_available"`
	Message                string         `json:"message,omitempty"`
	Plan                   string         `json:"plan,omitempty"`
	Quotas                 []KiroQuotaRow `json:"quotas,omitempty"`
	Current                *float64       `json:"current,omitempty"`
	Limit                  *float64       `json:"limit,omitempty"`
	Percent                *float64       `json:"percent,omitempty"`
	Remaining              *float64       `json:"remaining,omitempty"`
	NextResetAt            *time.Time     `json:"next_reset_at,omitempty"`
	SubscriptionType       string         `json:"subscription_type,omitempty"`
	SubscriptionTitle      string         `json:"subscription_title,omitempty"`
	TrialCurrent           *float64       `json:"trial_current,omitempty"`
	TrialLimit             *float64       `json:"trial_limit,omitempty"`
	TrialPercent           *float64       `json:"trial_percent,omitempty"`
	TrialStatus            string         `json:"trial_status,omitempty"`
	TrialExpiresAt         *time.Time     `json:"trial_expires_at,omitempty"`
	OverageStatus          string         `json:"overage_status,omitempty"`
	OverageCap             *float64       `json:"overage_cap,omitempty"`
	OverageRate            *float64       `json:"overage_rate,omitempty"`
	CurrentOverages        *float64       `json:"current_overages,omitempty"`
	CheckedAt              time.Time      `json:"checked_at"`
}

func (q KiroQuotaState) Exhausted(now time.Time) bool {
	if !q.ProviderQuotaAvailable || q.Current == nil || q.Limit == nil || *q.Limit <= 0 {
		return false
	}
	if q.NextResetAt != nil && !q.NextResetAt.IsZero() && !q.NextResetAt.After(now) {
		return false
	}
	return *q.Current >= *q.Limit
}

func (q KiroQuotaState) NextRecoverAt() time.Time {
	if q.NextResetAt == nil {
		return time.Time{}
	}
	return q.NextResetAt.UTC()
}

type kiroUsageLimitsResponse struct {
	UsageBreakdownList []kiroUsageBreakdown `json:"usageBreakdownList"`
	NextDateReset      any                  `json:"nextDateReset"`
	SubscriptionInfo   map[string]any       `json:"subscriptionInfo"`
	UserInfo           map[string]any       `json:"userInfo"`
	FreeTrialInfo      map[string]any       `json:"freeTrialInfo"`
	OverageStatus      any                  `json:"overageStatus"`
	OverageCapability  any                  `json:"overageCapability"`
	SubscriptionTitle  any                  `json:"subscriptionTitle"`
	OverageCap         any                  `json:"overageCap"`
	OverageRate        any                  `json:"overageRate"`
	CurrentOverages    any                  `json:"currentOverages"`
}

type kiroUsageBreakdown struct {
	ResourceType              any            `json:"resourceType"`
	CurrentUsage              any            `json:"currentUsage"`
	CurrentUsageWithPrecision any            `json:"currentUsageWithPrecision"`
	UsageCurrent              any            `json:"usageCurrent"`
	UsageLimit                any            `json:"usageLimit"`
	UsageLimitWithPrecision   any            `json:"usageLimitWithPrecision"`
	Limit                     any            `json:"limit"`
	FreeTrialInfo             map[string]any `json:"freeTrialInfo"`
	OverageStatus             any            `json:"overageStatus"`
	OverageCap                any            `json:"overageCap"`
	OverageRate               any            `json:"overageRate"`
	CurrentOverages           any            `json:"currentOverages"`
}

func (e *KiroExecutor) FetchQuota(ctx context.Context, auth *cliproxyauth.Auth) (KiroQuotaState, *cliproxyauth.Auth, error) {
	if auth == nil {
		return KiroQuotaState{}, nil, fmt.Errorf("kiro quota: auth is nil")
	}
	if metadataString(auth, "access_token") == "" {
		return KiroQuotaState{}, nil, statusErr{code: http.StatusUnauthorized, msg: "kiro quota: missing access token"}
	}
	quota, err := e.fetchKiroQuota(ctx, auth)
	if err != nil {
		if status, ok := err.(interface{ StatusCode() int }); ok && status.StatusCode() == http.StatusUnauthorized && metadataString(auth, "refresh_token") != "" {
			refreshed, errRefresh := e.Refresh(ctx, auth)
			if errRefresh != nil {
				return KiroQuotaState{}, nil, errRefresh
			}
			quota, err = e.fetchKiroQuota(ctx, refreshed)
			if err != nil {
				return KiroQuotaState{}, refreshed, err
			}
			return quota, refreshed, nil
		}
		return KiroQuotaState{}, nil, err
	}
	return quota, nil, nil
}

func (e *KiroExecutor) fetchKiroQuota(ctx context.Context, auth *cliproxyauth.Auth) (KiroQuotaState, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 30*time.Second)
	attempts := kiroUsageLimitAttempts(auth)
	var lastStatus int
	var lastBody string
	var lastErr error
	for _, attempt := range attempts {
		req, err := attempt.request(ctx, auth)
		if err != nil {
			lastErr = fmt.Errorf("kiro quota: create %s request: %w", attempt.name, err)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("kiro quota: %s request failed: %w", attempt.name, err)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("kiro quota: read %s response: %w", attempt.name, readErr)
			continue
		}
		if closeErr != nil {
			lastErr = fmt.Errorf("kiro quota: close %s response: %w", attempt.name, closeErr)
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			lastStatus = resp.StatusCode
			lastBody = strings.TrimSpace(string(body))
			lastErr = statusErr{code: resp.StatusCode, msg: fmt.Sprintf("kiro quota: %s status %d: %s", attempt.name, resp.StatusCode, lastBody)}
			continue
		}
		return ParseKiroUsageLimits(body, time.Now().UTC())
	}
	if lastStatus > 0 {
		return KiroQuotaState{}, statusErr{code: lastStatus, msg: fmt.Sprintf("kiro quota: status %d: %s", lastStatus, lastBody)}
	}
	if lastErr != nil {
		return KiroQuotaState{}, lastErr
	}
	return KiroQuotaState{}, fmt.Errorf("kiro quota: no usage endpoint attempts configured")
}

func ParseKiroUsageLimits(body []byte, checkedAt time.Time) (KiroQuotaState, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload kiroUsageLimitsResponse
	if err := decoder.Decode(&payload); err != nil {
		return KiroQuotaState{}, fmt.Errorf("kiro quota: decode response: %w", err)
	}
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	quota := KiroQuotaState{CheckedAt: checkedAt.UTC()}
	quota.NextResetAt = timePtr(payload.NextDateReset)
	quota.SubscriptionType = firstString(payload.SubscriptionInfo, "subscriptionType", "type", "tier", "planType")
	quota.SubscriptionTitle = firstString(payload.SubscriptionInfo, "subscriptionTitle", "title", "planTitle", "name")
	if quota.SubscriptionTitle == "" {
		quota.SubscriptionTitle = kiroQuotaStringValue(payload.SubscriptionTitle)
	}
	if quota.SubscriptionType == "" {
		quota.SubscriptionType = firstString(payload.UserInfo, "subscriptionType", "type", "tier", "planType")
	}
	quota.Plan = quota.SubscriptionTitle
	if quota.Plan == "" {
		quota.Plan = "Kiro"
	}

	for _, breakdown := range payload.UsageBreakdownList {
		row := buildKiroQuotaRow(breakdown, quota.NextResetAt, quota.SubscriptionType, quota.SubscriptionTitle, payload)
		if row == nil {
			continue
		}
		quota.Quotas = append(quota.Quotas, *row)
		if row.Current != nil || row.Limit != nil {
			quota.ProviderQuotaAvailable = true
		}
		if quota.Current == nil && !row.FreeTrial {
			quota.Current = row.Current
			quota.Limit = row.Limit
			quota.Remaining = row.Remaining
			quota.Percent = row.Percent
		}
		trialRow := buildKiroFreeTrialQuotaRow(breakdown, row, quota.NextResetAt, quota.SubscriptionType, quota.SubscriptionTitle)
		if trialRow != nil {
			quota.Quotas = append(quota.Quotas, *trialRow)
			if quota.TrialCurrent == nil {
				quota.TrialCurrent = trialRow.Current
				quota.TrialLimit = trialRow.Limit
				quota.TrialPercent = trialRow.Percent
				quota.TrialStatus = trialRow.TrialStatus
				quota.TrialExpiresAt = trialRow.ResetAt
			}
		}
	}

	trialCurrent := firstNumber(payload.FreeTrialInfo, "currentUsage", "usageCurrent", "current")
	trialLimit := firstNumber(payload.FreeTrialInfo, "usageLimit", "limit")
	if trialCurrent != nil || trialLimit != nil {
		quota.TrialCurrent = trialCurrent
		quota.TrialLimit = trialLimit
		quota.TrialPercent = percentPtr(trialCurrent, trialLimit)
		quota.TrialStatus = firstString(payload.FreeTrialInfo, "status", "trialStatus", "freeTrialStatus")
		quota.TrialExpiresAt = timePtr(firstMapValue(payload.FreeTrialInfo, "expiresAt", "expiredAt", "expirationDate", "trialExpiresAt", "endDate", "freeTrialExpiry"))
		if trialRow := buildTopLevelKiroFreeTrialQuotaRow(payload.FreeTrialInfo, quota.NextResetAt, quota.SubscriptionType, quota.SubscriptionTitle); trialRow != nil {
			quota.Quotas = append(quota.Quotas, *trialRow)
		}
	}

	quota.OverageStatus = kiroQuotaStringValue(firstNonNil(payload.OverageStatus, firstMapValue(payload.SubscriptionInfo, "overageStatus")))
	quota.OverageCap = numberPtr(firstNonNil(payload.OverageCap, firstMapValue(payload.SubscriptionInfo, "overageCap")))
	quota.OverageRate = numberPtr(firstNonNil(payload.OverageRate, firstMapValue(payload.SubscriptionInfo, "overageRate")))
	quota.CurrentOverages = numberPtr(firstNonNil(payload.CurrentOverages, firstMapValue(payload.SubscriptionInfo, "currentOverages")))

	return quota, nil
}

func ApplyKiroQuotaToAuth(auth *cliproxyauth.Auth, quota KiroQuotaState, now time.Time) *cliproxyauth.Auth {
	if auth == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	updated := auth.Clone()
	if updated.Metadata == nil {
		updated.Metadata = make(map[string]any)
	}
	updated.Metadata[kiroProviderQuotaKey] = quota
	if quota.Exhausted(now) {
		updated.Quota.Exceeded = true
		updated.Quota.Reason = kiroProviderQuotaReason
		updated.Quota.NextRecoverAt = quota.NextRecoverAt()
	} else if updated.Quota.Reason == kiroProviderQuotaReason {
		updated.Quota = cliproxyauth.QuotaState{}
	}
	updated.UpdatedAt = now.UTC()
	return updated
}

func KiroQuotaFromAuth(auth *cliproxyauth.Auth) (KiroQuotaState, bool) {
	if auth == nil || auth.Metadata == nil {
		return KiroQuotaState{}, false
	}
	return normalizeKiroQuotaState(auth.Metadata[kiroProviderQuotaKey])
}

func normalizeKiroQuotaState(raw any) (KiroQuotaState, bool) {
	if raw == nil {
		return KiroQuotaState{}, false
	}
	switch typed := raw.(type) {
	case KiroQuotaState:
		return typed, true
	case map[string]any:
		quota := KiroQuotaState{
			ProviderQuotaAvailable: boolValue(firstMapValue(typed, "provider_quota_available", "providerQuotaAvailable", "available")),
			Current:                numberPtr(firstMapValue(typed, "current")),
			Limit:                  numberPtr(firstMapValue(typed, "limit")),
			Percent:                numberPtr(firstMapValue(typed, "percent")),
			Remaining:              numberPtr(firstMapValue(typed, "remaining")),
			Message:                kiroQuotaStringValue(firstMapValue(typed, "message")),
			Plan:                   kiroQuotaStringValue(firstMapValue(typed, "plan")),
			Quotas:                 normalizeKiroQuotaRows(firstMapValue(typed, "quotas")),
			NextResetAt:            timePtr(firstMapValue(typed, "next_reset_at", "nextResetAt")),
			SubscriptionType:       kiroQuotaStringValue(firstMapValue(typed, "subscription_type", "subscriptionType")),
			SubscriptionTitle:      kiroQuotaStringValue(firstMapValue(typed, "subscription_title", "subscriptionTitle")),
			TrialCurrent:           numberPtr(firstMapValue(typed, "trial_current", "trialCurrent")),
			TrialLimit:             numberPtr(firstMapValue(typed, "trial_limit", "trialLimit")),
			TrialPercent:           numberPtr(firstMapValue(typed, "trial_percent", "trialPercent")),
			TrialStatus:            kiroQuotaStringValue(firstMapValue(typed, "trial_status", "trialStatus")),
			TrialExpiresAt:         timePtr(firstMapValue(typed, "trial_expires_at", "trialExpiresAt")),
			OverageStatus:          kiroQuotaStringValue(firstMapValue(typed, "overage_status", "overageStatus")),
			OverageCap:             numberPtr(firstMapValue(typed, "overage_cap", "overageCap")),
			OverageRate:            numberPtr(firstMapValue(typed, "overage_rate", "overageRate")),
			CurrentOverages:        numberPtr(firstMapValue(typed, "current_overages", "currentOverages")),
			CheckedAt:              timeValue(firstMapValue(typed, "checked_at", "checkedAt")),
		}
		if quota.Current != nil || quota.Limit != nil || len(quota.Quotas) > 0 {
			quota.ProviderQuotaAvailable = true
		}
		return quota, true
	default:
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			return KiroQuotaState{}, false
		}
		var decoded map[string]any
		if err := json.Unmarshal(rawJSON, &decoded); err != nil {
			return KiroQuotaState{}, false
		}
		return normalizeKiroQuotaState(decoded)
	}
}

func normalizeKiroQuotaRows(raw any) []KiroQuotaRow {
	if raw == nil {
		return nil
	}
	var values []any
	switch typed := raw.(type) {
	case []any:
		values = typed
	default:
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			return nil
		}
		if err := json.Unmarshal(rawJSON, &values); err != nil {
			return nil
		}
	}
	rows := make([]KiroQuotaRow, 0, len(values))
	for _, value := range values {
		source, ok := value.(map[string]any)
		if !ok {
			rawJSON, err := json.Marshal(value)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(rawJSON, &source); err != nil {
				continue
			}
		}
		row := KiroQuotaRow{
			ID:                kiroQuotaStringValue(firstMapValue(source, "id")),
			ResourceType:      kiroQuotaStringValue(firstMapValue(source, "resource_type", "resourceType")),
			Name:              kiroQuotaStringValue(firstMapValue(source, "name")),
			Current:           numberPtr(firstMapValue(source, "current")),
			Limit:             numberPtr(firstMapValue(source, "limit")),
			Used:              numberPtr(firstMapValue(source, "used")),
			Total:             numberPtr(firstMapValue(source, "total")),
			Remaining:         numberPtr(firstMapValue(source, "remaining")),
			Percent:           numberPtr(firstMapValue(source, "percent")),
			RemainingPercent:  numberPtr(firstMapValue(source, "remaining_percent", "remainingPercent")),
			ResetAt:           timePtr(firstMapValue(source, "reset_at", "resetAt")),
			Unlimited:         boolValue(firstMapValue(source, "unlimited")),
			FreeTrial:         boolValue(firstMapValue(source, "free_trial", "freeTrial")),
			TrialStatus:       kiroQuotaStringValue(firstMapValue(source, "trial_status", "trialStatus")),
			SubscriptionTitle: kiroQuotaStringValue(firstMapValue(source, "subscription_title", "subscriptionTitle")),
			SubscriptionType:  kiroQuotaStringValue(firstMapValue(source, "subscription_type", "subscriptionType")),
			OverageStatus:     kiroQuotaStringValue(firstMapValue(source, "overage_status", "overageStatus")),
			OverageCap:        numberPtr(firstMapValue(source, "overage_cap", "overageCap")),
			OverageRate:       numberPtr(firstMapValue(source, "overage_rate", "overageRate")),
			CurrentOverages:   numberPtr(firstMapValue(source, "current_overages", "currentOverages")),
		}
		if row.ID == "" {
			row.ID = row.ResourceType
		}
		if row.Name == "" {
			row.Name = row.ResourceType
		}
		if row.Current == nil {
			row.Current = row.Used
		}
		if row.Limit == nil {
			row.Limit = row.Total
		}
		if row.Used == nil {
			row.Used = row.Current
		}
		if row.Total == nil {
			row.Total = row.Limit
		}
		if row.Percent == nil {
			row.Percent = percentPtr(row.Current, row.Limit)
		}
		if row.Remaining == nil {
			row.Remaining = remainingPtr(row.Current, row.Limit)
		}
		if row.RemainingPercent == nil {
			row.RemainingPercent = remainingPercentPtr(row.Remaining, row.Limit)
		}
		rows = append(rows, row)
	}
	return rows
}

type kiroUsageLimitAttempt struct {
	name    string
	request func(context.Context, *cliproxyauth.Auth) (*http.Request, error)
}

func kiroUsageLimitAttempts(auth *cliproxyauth.Auth) []kiroUsageLimitAttempt {
	if rawURL := metadataString(auth, "quota_url"); rawURL != "" {
		return []kiroUsageLimitAttempt{kiroUsageLimitGetAttempt("custom-get", rawURL, true)}
	}
	if rawURL := metadataString(auth, "usage_limits_url"); rawURL != "" {
		return []kiroUsageLimitAttempt{kiroUsageLimitGetAttempt("custom-get", rawURL, true)}
	}
	return []kiroUsageLimitAttempt{
		kiroUsageLimitGetAttempt("codewhisperer-get", kiroCodeWhispererUsageURL(auth), false),
		{
			name: "codewhisperer-post",
			request: func(ctx context.Context, auth *cliproxyauth.Auth) (*http.Request, error) {
				body := map[string]any{
					"origin":       "AI_EDITOR",
					"resourceType": "AGENTIC_REQUEST",
				}
				if profileARN := metadataString(auth, "profile_arn"); profileARN != "" {
					body["profileArn"] = profileARN
				}
				data, err := json.Marshal(body)
				if err != nil {
					return nil, err
				}
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, kiroCodeWhispererUsagePostURL(auth), bytes.NewReader(data))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+metadataString(auth, "access_token"))
				req.Header.Set("Content-Type", "application/x-amz-json-1.0")
				req.Header.Set("Accept", "application/json")
				req.Header.Set("x-amz-target", kiroCodeWhispererGetUsageTarget)
				if auth != nil {
					util.ApplyCustomHeadersFromAttrs(req, auth.Attributes)
				}
				return req, nil
			},
		},
		kiroUsageLimitGetAttempt("q-get", kiroQUsageURL(auth), true),
	}
}

func kiroUsageLimitGetAttempt(name, rawURL string, includeProfileARN bool) kiroUsageLimitAttempt {
	return kiroUsageLimitAttempt{
		name: name,
		request: func(ctx context.Context, auth *cliproxyauth.Auth) (*http.Request, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, kiroUsageLimitsURLWithBase(auth, rawURL, includeProfileARN), nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+metadataString(auth, "access_token"))
			applyKiroUsageLimitsHeaders(req, auth)
			return req, nil
		},
	}
}

func kiroUsageLimitsURL(auth *cliproxyauth.Auth) string {
	rawURL := metadataString(auth, "quota_url")
	if rawURL == "" {
		rawURL = metadataString(auth, "usage_limits_url")
	}
	if rawURL == "" {
		region := metadataString(auth, "region")
		if region == "" {
			region = regionFromKiroProfileARN(metadataString(auth, "profile_arn"))
		}
		if region == "" {
			region = kiro.DefaultRegion
		}
		rawURL = fmt.Sprintf("https://q.%s.amazonaws.com%s", region, kiroUsageLimitsPath)
	}
	return kiroUsageLimitsURLWithBase(auth, rawURL, true)
}

func kiroCodeWhispererUsageURL(auth *cliproxyauth.Auth) string {
	if rawURL := metadataString(auth, "codewhisperer_usage_url"); rawURL != "" {
		return rawURL
	}
	return kiroCodeWhispererUsageLimitsURL
}

func kiroCodeWhispererUsagePostURL(auth *cliproxyauth.Auth) string {
	if rawURL := metadataString(auth, "codewhisperer_usage_base_url"); rawURL != "" {
		return rawURL
	}
	return kiroCodeWhispererUsageBaseURL
}

func kiroQUsageURL(auth *cliproxyauth.Auth) string {
	if rawURL := metadataString(auth, "q_usage_url"); rawURL != "" {
		return rawURL
	}
	return kiroUsageLimitsURL(auth)
}

func kiroUsageLimitsURLWithBase(auth *cliproxyauth.Auth, rawURL string, includeProfileARN bool) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	if query.Get("origin") == "" {
		query.Set("origin", "AI_EDITOR")
	}
	if query.Get("resourceType") == "" {
		query.Set("resourceType", "AGENTIC_REQUEST")
	}
	if query.Get("isEmailRequired") == "" {
		query.Set("isEmailRequired", "true")
	}
	if profileARN := metadataString(auth, "profile_arn"); includeProfileARN && profileARN != "" && query.Get("profileArn") == "" {
		query.Set("profileArn", profileARN)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func buildKiroQuotaRow(breakdown kiroUsageBreakdown, resetAt *time.Time, subscriptionType, subscriptionTitle string, payload kiroUsageLimitsResponse) *KiroQuotaRow {
	resourceType := strings.ToLower(kiroQuotaStringValue(breakdown.ResourceType))
	if resourceType == "" {
		resourceType = "unknown"
	}
	current := numberPtr(firstNonNil(breakdown.CurrentUsageWithPrecision, breakdown.CurrentUsage, breakdown.UsageCurrent))
	limit := numberPtr(firstNonNil(breakdown.UsageLimitWithPrecision, breakdown.UsageLimit, breakdown.Limit))
	if current == nil && limit == nil {
		return nil
	}
	row := &KiroQuotaRow{
		ID:                resourceType,
		ResourceType:      resourceType,
		Name:              resourceType,
		Current:           current,
		Limit:             limit,
		Used:              current,
		Total:             limit,
		Remaining:         remainingPtr(current, limit),
		Percent:           percentPtr(current, limit),
		ResetAt:           resetAt,
		SubscriptionType:  subscriptionType,
		SubscriptionTitle: subscriptionTitle,
		OverageStatus:     kiroQuotaStringValue(firstNonNil(breakdown.OverageStatus, payload.OverageStatus, firstMapValue(payload.SubscriptionInfo, "overageStatus"))),
		OverageCap:        numberPtr(firstNonNil(breakdown.OverageCap, payload.OverageCap, firstMapValue(payload.SubscriptionInfo, "overageCap"))),
		OverageRate:       numberPtr(firstNonNil(breakdown.OverageRate, payload.OverageRate, firstMapValue(payload.SubscriptionInfo, "overageRate"))),
		CurrentOverages:   numberPtr(firstNonNil(breakdown.CurrentOverages, payload.CurrentOverages, firstMapValue(payload.SubscriptionInfo, "currentOverages"))),
	}
	row.RemainingPercent = remainingPercentPtr(row.Remaining, limit)
	return row
}

func buildKiroFreeTrialQuotaRow(breakdown kiroUsageBreakdown, parent *KiroQuotaRow, defaultResetAt *time.Time, subscriptionType, subscriptionTitle string) *KiroQuotaRow {
	if parent == nil || len(breakdown.FreeTrialInfo) == 0 {
		return nil
	}
	current := firstNumber(breakdown.FreeTrialInfo, "currentUsageWithPrecision", "currentUsage", "usageCurrent", "current")
	limit := firstNumber(breakdown.FreeTrialInfo, "usageLimitWithPrecision", "usageLimit", "limit")
	if current == nil && limit == nil {
		return nil
	}
	resetAt := timePtr(firstMapValue(breakdown.FreeTrialInfo, "freeTrialExpiry", "expiresAt", "expiredAt", "expirationDate", "trialExpiresAt", "endDate"))
	if resetAt == nil {
		resetAt = defaultResetAt
	}
	row := &KiroQuotaRow{
		ID:                parent.ID + "_freetrial",
		ResourceType:      parent.ResourceType,
		Name:              parent.Name + "_freetrial",
		Current:           current,
		Limit:             limit,
		Used:              current,
		Total:             limit,
		Remaining:         remainingPtr(current, limit),
		Percent:           percentPtr(current, limit),
		ResetAt:           resetAt,
		Unlimited:         false,
		FreeTrial:         true,
		TrialStatus:       firstString(breakdown.FreeTrialInfo, "status", "trialStatus", "freeTrialStatus"),
		SubscriptionType:  subscriptionType,
		SubscriptionTitle: subscriptionTitle,
	}
	row.RemainingPercent = remainingPercentPtr(row.Remaining, limit)
	return row
}

func buildTopLevelKiroFreeTrialQuotaRow(info map[string]any, defaultResetAt *time.Time, subscriptionType, subscriptionTitle string) *KiroQuotaRow {
	if len(info) == 0 {
		return nil
	}
	current := firstNumber(info, "currentUsageWithPrecision", "currentUsage", "usageCurrent", "current")
	limit := firstNumber(info, "usageLimitWithPrecision", "usageLimit", "limit")
	if current == nil && limit == nil {
		return nil
	}
	resetAt := timePtr(firstMapValue(info, "freeTrialExpiry", "expiresAt", "expiredAt", "expirationDate", "trialExpiresAt", "endDate"))
	if resetAt == nil {
		resetAt = defaultResetAt
	}
	row := &KiroQuotaRow{
		ID:                "free_trial",
		ResourceType:      "free_trial",
		Name:              "free_trial",
		Current:           current,
		Limit:             limit,
		Used:              current,
		Total:             limit,
		Remaining:         remainingPtr(current, limit),
		Percent:           percentPtr(current, limit),
		ResetAt:           resetAt,
		FreeTrial:         true,
		TrialStatus:       firstString(info, "status", "trialStatus", "freeTrialStatus"),
		SubscriptionType:  subscriptionType,
		SubscriptionTitle: subscriptionTitle,
	}
	row.RemainingPercent = remainingPercentPtr(row.Remaining, limit)
	return row
}

func applyKiroUsageLimitsHeaders(req *http.Request, auth *cliproxyauth.Auth) {
	req.Header.Set("Accept", "application/json")
	applyKiroFingerprintHeaders(req, auth, "codewhispererruntime", "1.0.0", "m/N,E", 1)
	if auth != nil {
		util.ApplyCustomHeadersFromAttrs(req, auth.Attributes)
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstMapValue(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, key := range keys {
		if value, ok := m[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func firstString(m map[string]any, keys ...string) string {
	return kiroQuotaStringValue(firstMapValue(m, keys...))
}

func firstNumber(m map[string]any, keys ...string) *float64 {
	return numberPtr(firstMapValue(m, keys...))
}

func numberPtr(v any) *float64 {
	n, ok := numberValue(v)
	if !ok {
		return nil
	}
	return &n
}

func numberValue(v any) (float64, bool) {
	switch typed := v.(type) {
	case nil:
		return 0, false
	case json.Number:
		n, err := typed.Float64()
		return n, err == nil && finiteFloat(n)
	case float64:
		return typed, finiteFloat(typed)
	case float32:
		n := float64(typed)
		return n, finiteFloat(n)
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		n, err := strconv.ParseFloat(trimmed, 64)
		return n, err == nil && finiteFloat(n)
	default:
		return 0, false
	}
}

func finiteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func percentPtr(current, limit *float64) *float64 {
	if current == nil || limit == nil || *limit <= 0 {
		return nil
	}
	value := (*current / *limit) * 100
	return &value
}

func remainingPercentPtr(remaining, limit *float64) *float64 {
	if remaining == nil || limit == nil || *limit <= 0 {
		return nil
	}
	value := (*remaining / *limit) * 100
	return &value
}

func remainingPtr(current, limit *float64) *float64 {
	if current == nil || limit == nil {
		return nil
	}
	value := *limit - *current
	if value < 0 {
		value = 0
	}
	return &value
}

func timePtr(v any) *time.Time {
	ts := timeValue(v)
	if ts.IsZero() {
		return nil
	}
	return &ts
}

func timeValue(v any) time.Time {
	switch typed := v.(type) {
	case nil:
		return time.Time{}
	case time.Time:
		return typed.UTC()
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return unixTimeValue(i)
		}
	case float64:
		return unixTimeValue(int64(typed))
	case int64:
		return unixTimeValue(typed)
	case int:
		return unixTimeValue(int64(typed))
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" || strings.HasPrefix(trimmed, "0001-01-01") {
			return time.Time{}
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if parsed, err := time.Parse(layout, trimmed); err == nil {
				return parsed.UTC()
			}
		}
		if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return unixTimeValue(i)
		}
	}
	return time.Time{}
}

func unixTimeValue(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value > 1e12 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

func kiroQuotaStringValue(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func boolValue(v any) bool {
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	case float64:
		return typed != 0
	case json.Number:
		n, _ := typed.Int64()
		return n != 0
	}
	return false
}
