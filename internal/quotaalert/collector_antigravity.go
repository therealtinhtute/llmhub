package quotaalert

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const antigravityQuotaPath = "/v1internal:fetchAvailableModels"

var antigravityCollectorBaseURLs = []string{
	"https://daily-cloudcode-pa.googleapis.com",
	"https://daily-cloudcode-pa.sandbox.googleapis.com",
	"https://cloudcode-pa.googleapis.com",
}

type AntigravityCollector struct {
	httpClients []*CollectorHTTPClient
	refresh     CollectorRefreshFunc
	now         func() time.Time
}

type antigravityQuotaPayload struct {
	Models map[string]antigravityModelQuota `json:"models"`
}

type antigravityModelQuota struct {
	DisplayName  string                `json:"displayName"`
	QuotaInfo    *antigravityQuotaInfo `json:"quotaInfo"`
	QuotaInfoAlt *antigravityQuotaInfo `json:"quota_info"`
}

type antigravityQuotaInfo struct {
	RemainingFraction    any `json:"remainingFraction"`
	RemainingFractionAlt any `json:"remaining_fraction"`
	Remaining            any `json:"remaining"`
	ResetTime            any `json:"resetTime"`
	ResetTimeAlt         any `json:"reset_time"`
}

type antigravityGroupDefinition struct {
	id             string
	label          string
	identifiers    []string
	labelFromModel bool
}

type antigravityQuotaEntry struct {
	id                string
	displayName       string
	remainingFraction float64
	resetTime         string
}

var antigravityGroups = []antigravityGroupDefinition{
	{id: "claude-gpt", label: "Claude/GPT", identifiers: []string{"claude-sonnet-4-6", "claude-opus-4-6-thinking", "gpt-oss-120b-medium"}},
	{id: "gemini-3-pro", label: "Gemini 3 Pro", identifiers: []string{"gemini-3-pro-high", "gemini-3-pro-low"}},
	{id: "gemini-3-1-pro-series", label: "Gemini 3.1 Pro Series", identifiers: []string{"gemini-3.1-pro-high", "gemini-3.1-pro-low"}},
	{id: "gemini-2-5-flash", label: "Gemini 2.5 Flash", identifiers: []string{"gemini-2.5-flash", "gemini-2.5-flash-thinking"}},
	{id: "gemini-2-5-flash-lite", label: "Gemini 2.5 Flash Lite", identifiers: []string{"gemini-2.5-flash-lite"}},
	{id: "gemini-2-5-cu", label: "Gemini 2.5 CU", identifiers: []string{"rev19-uic3-1p"}},
	{id: "gemini-3-flash", label: "Gemini 3 Flash", identifiers: []string{"gemini-3-flash"}},
	{id: "gemini-image", label: "gemini-3.1-flash-image", identifiers: []string{"gemini-3.1-flash-image"}, labelFromModel: true},
}

func NewAntigravityCollector(deps CollectorDependencies) (Collector, error) {
	if deps.HTTPClient != nil {
		return &AntigravityCollector{httpClients: []*CollectorHTTPClient{deps.HTTPClient}, refresh: deps.Refresh, now: time.Now}, nil
	}
	clients := make([]*CollectorHTTPClient, 0, len(antigravityCollectorBaseURLs))
	for _, baseURL := range antigravityCollectorBaseURLs {
		client, err := NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      baseURL,
			AllowedHosts: []string{"daily-cloudcode-pa.googleapis.com", "daily-cloudcode-pa.sandbox.googleapis.com", "cloudcode-pa.googleapis.com"},
		})
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return &AntigravityCollector{httpClients: clients, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *AntigravityCollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || len(c.httpClients) == 0 {
		return nil, fmt.Errorf("antigravity quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token", "project_id"}, []string{"access_token", "project_id"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok {
		return nil, fmt.Errorf("antigravity quota collector access token is missing")
	}
	projectID, ok := snapshotString(cloned, "project_id")
	if !ok {
		return nil, fmt.Errorf("antigravity quota collector project ID is missing")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
		"User-Agent":    "antigravity/1.11.5 windows/amd64",
	}
	body := map[string]string{"project": projectID}
	var lastErr error
	for _, client := range c.httpClients {
		var payload antigravityQuotaPayload
		err = client.JSONBody(ctx, cloned, http.MethodPost, antigravityQuotaPath, headers, body, &payload, c.refresh)
		if err != nil {
			lastErr = err
			continue
		}
		observations := buildAntigravityObservations(cloned, payload.Models, c.now().UTC())
		if len(observations) > 0 {
			return observations, nil
		}
		lastErr = fmt.Errorf("antigravity quota response contains no recognized models")
	}
	if lastErr != nil {
		return nil, fmt.Errorf("antigravity quota request failed: %s", RedactCollectorError(lastErr, cloned))
	}
	return nil, fmt.Errorf("antigravity quota response contains no recognized models")
}

func buildAntigravityObservations(auth AuthSnapshot, models map[string]antigravityModelQuota, observedAt time.Time) []Observation {
	if len(models) == 0 {
		return nil
	}
	order := make(map[string]int, len(antigravityGroups))
	for index, group := range antigravityGroups {
		order[group.id] = index
	}
	built := make(map[string]Observation)
	var geminiProResetTime string
	for _, group := range antigravityGroups {
		observation, resetTime, ok := buildAntigravityGroupObservation(auth, models, observedAt, group)
		if !ok {
			continue
		}
		if group.id == "gemini-3-1-pro-series" || (group.id == "gemini-3-pro" && geminiProResetTime == "") {
			geminiProResetTime = resetTime
		}
		built[group.id+"::default"] = observation
	}
	if image, ok := built["gemini-image::default"]; ok && !image.ResetKnown && geminiProResetTime != "" {
		if resetAt, resetKnown := parseProviderTime(geminiProResetTime, observedAt); resetKnown {
			image.ResetAt = resetAt
			image.ResetKnown = true
			if normalized, err := image.Normalize(); err == nil {
				built["gemini-image::default"] = normalized
			}
		}
	}
	keys := sortedGroupKeys(built, order)
	observations := make([]Observation, 0, len(keys))
	for _, key := range keys {
		observations = append(observations, built[key])
	}
	return observations
}

func buildAntigravityGroupObservation(auth AuthSnapshot, models map[string]antigravityModelQuota, observedAt time.Time, group antigravityGroupDefinition) (Observation, string, bool) {
	entries := make([]antigravityQuotaEntry, 0, len(group.identifiers))
	for _, identifier := range group.identifiers {
		id, model, ok := findAntigravityModel(models, identifier)
		if !ok {
			continue
		}
		entry, ok := normalizeAntigravityEntry(id, model)
		if ok {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return Observation{}, "", false
	}
	remainingFraction := entries[0].remainingFraction
	resetTime := entries[0].resetTime
	displayName := entries[0].displayName
	for _, entry := range entries[1:] {
		if entry.remainingFraction < remainingFraction {
			remainingFraction = entry.remainingFraction
		}
		resetTime = earlierProviderTime(resetTime, entry.resetTime)
		if displayName == "" {
			displayName = entry.displayName
		}
	}
	remaining, err := NormalizePercentage(remainingFraction * 100)
	if err != nil {
		return Observation{}, "", false
	}
	resetAt, resetKnown := parseProviderTime(resetTime, observedAt)
	resource := group.id
	if group.labelFromModel && strings.TrimSpace(displayName) != "" {
		resource = slugID(displayName)
		if resource == "" {
			resource = group.id
		}
	}
	observation, err := (Observation{
		Identity: StateIdentity{
			AuthID:   auth.AuthID(),
			Provider: ProviderAntigravity,
			Resource: resource,
			Window:   "default",
		},
		AuthLabel:           auth.RedactedLabel(),
		Health:              CollectionReliable,
		Remaining:           remaining,
		RemainingKnown:      true,
		ExplicitlyExhausted: remaining == 0,
		ResetAt:             resetAt,
		ResetKnown:          resetKnown,
		ObservedAt:          observedAt,
	}).Normalize()
	if err != nil {
		return Observation{}, "", false
	}
	return observation, resetTime, true
}

func findAntigravityModel(models map[string]antigravityModelQuota, identifier string) (string, antigravityModelQuota, bool) {
	if model, ok := models[identifier]; ok {
		return identifier, model, true
	}
	for id, model := range models {
		if strings.EqualFold(strings.TrimSpace(model.DisplayName), identifier) {
			return id, model, true
		}
	}
	return "", antigravityModelQuota{}, false
}

func normalizeAntigravityEntry(id string, model antigravityModelQuota) (antigravityQuotaEntry, bool) {
	quota := model.QuotaInfo
	if quota == nil {
		quota = model.QuotaInfoAlt
	}
	if quota == nil {
		return antigravityQuotaEntry{}, false
	}
	resetTime, _ := stringFromAny(firstAny(quota.ResetTime, quota.ResetTimeAlt))
	remainingFraction, ok := quotaFractionFromAny(firstAny(quota.RemainingFraction, quota.RemainingFractionAlt, quota.Remaining))
	if !ok {
		if strings.TrimSpace(resetTime) == "" {
			return antigravityQuotaEntry{}, false
		}
		remainingFraction = 0
	}
	return antigravityQuotaEntry{
		id:                id,
		displayName:       model.DisplayName,
		remainingFraction: remainingFraction,
		resetTime:         resetTime,
	}, true
}
