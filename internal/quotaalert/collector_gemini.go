package quotaalert

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	geminiCLICollectorBaseURL    = "https://cloudcode-pa.googleapis.com"
	geminiCLIQuotaPath           = "/v1internal:retrieveUserQuota"
	geminiCLICodeAssistPath      = "/v1internal:loadCodeAssist"
	geminiCLIGoogleOneCreditType = "GOOGLE_ONE_AI"
)

type GeminiCLICollector struct {
	httpClient *CollectorHTTPClient
	refresh    CollectorRefreshFunc
	now        func() time.Time
}

type geminiCLIQuotaPayload struct {
	Buckets []geminiCLIQuotaBucket `json:"buckets"`
}

type geminiCLIQuotaBucket struct {
	ModelID              any `json:"modelId"`
	ModelIDAlt           any `json:"model_id"`
	TokenType            any `json:"tokenType"`
	TokenTypeAlt         any `json:"token_type"`
	RemainingFraction    any `json:"remainingFraction"`
	RemainingFractionAlt any `json:"remaining_fraction"`
	RemainingAmount      any `json:"remainingAmount"`
	RemainingAmountAlt   any `json:"remaining_amount"`
	ResetTime            any `json:"resetTime"`
	ResetTimeAlt         any `json:"reset_time"`
}

type geminiCLIParsedBucket struct {
	modelID           string
	tokenType         string
	remainingFraction *float64
	remainingAmount   *float64
	resetTime         string
}

type geminiCLIGroup struct {
	id               string
	label            string
	preferredModelID string
	modelIDs         []string
}

type geminiCLIGroupBucket struct {
	id                        string
	tokenType                 string
	modelIDs                  []string
	preferredModelID          string
	preferred                 *geminiCLIParsedBucket
	fallbackRemainingFraction *float64
	fallbackRemainingAmount   *float64
	fallbackResetTime         string
}

type geminiCLICodeAssistPayload struct {
	CurrentTier *geminiCLITier `json:"currentTier"`
	CurrentAlt  *geminiCLITier `json:"current_tier"`
	PaidTier    *geminiCLITier `json:"paidTier"`
	PaidAlt     *geminiCLITier `json:"paid_tier"`
}

type geminiCLITier struct {
	ID                  any                `json:"id"`
	AvailableCredits    []geminiCLICredits `json:"availableCredits"`
	AvailableCreditsAlt []geminiCLICredits `json:"available_credits"`
}

type geminiCLICredits struct {
	CreditType      any `json:"creditType"`
	CreditTypeAlt   any `json:"credit_type"`
	CreditAmount    any `json:"creditAmount"`
	CreditAmountAlt any `json:"credit_amount"`
}

var geminiCLIGroups = []geminiCLIGroup{
	{id: "gemini-flash-lite-series", label: "Gemini Flash Lite Series", preferredModelID: "gemini-2.5-flash-lite", modelIDs: []string{"gemini-2.5-flash-lite"}},
	{id: "gemini-flash-series", label: "Gemini Flash Series", preferredModelID: "gemini-3-flash-preview", modelIDs: []string{"gemini-3-flash-preview", "gemini-2.5-flash"}},
	{id: "gemini-pro-series", label: "Gemini Pro Series", preferredModelID: "gemini-3.1-pro-preview", modelIDs: []string{"gemini-3.1-pro-preview", "gemini-3-pro-preview", "gemini-2.5-pro"}},
}

func NewGeminiCLICollector(deps CollectorDependencies) (Collector, error) {
	client := deps.HTTPClient
	var err error
	if client == nil {
		client, err = NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      geminiCLICollectorBaseURL,
			AllowedHosts: []string{"cloudcode-pa.googleapis.com"},
		})
		if err != nil {
			return nil, err
		}
	}
	return &GeminiCLICollector{httpClient: client, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *GeminiCLICollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("gemini CLI quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token", "project_id"}, []string{"access_token", "project_id"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok {
		return nil, fmt.Errorf("gemini CLI quota collector access token is missing")
	}
	projectID, ok := snapshotString(cloned, "project_id")
	if !ok {
		return nil, fmt.Errorf("gemini CLI quota collector project ID is missing")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
	}
	var payload geminiCLIQuotaPayload
	if err = c.httpClient.JSONBody(ctx, cloned, http.MethodPost, geminiCLIQuotaPath, headers, map[string]string{"project": projectID}, &payload, c.refresh); err != nil {
		return nil, fmt.Errorf("gemini CLI quota request failed: %s", RedactCollectorError(err, cloned))
	}
	observedAt := c.now().UTC()
	observations := buildGeminiCLIObservations(cloned, payload.Buckets, observedAt)
	if len(observations) == 0 {
		return nil, fmt.Errorf("gemini CLI quota response contains no recognized buckets")
	}

	var ignored geminiCLICodeAssistPayload
	_ = c.httpClient.JSONBody(ctx, cloned, http.MethodPost, geminiCLICodeAssistPath, headers, map[string]any{
		"cloudaicompanionProject": projectID,
		"metadata": map[string]string{
			"ideType":     "IDE_UNSPECIFIED",
			"platform":    "PLATFORM_UNSPECIFIED",
			"pluginType":  "GEMINI",
			"duetProject": projectID,
		},
	}, &ignored, c.refresh)
	return observations, nil
}

func buildGeminiCLIObservations(auth AuthSnapshot, buckets []geminiCLIQuotaBucket, observedAt time.Time) []Observation {
	parsed := make([]geminiCLIParsedBucket, 0, len(buckets))
	for _, bucket := range buckets {
		modelID, ok := stringFromAny(firstAny(bucket.ModelID, bucket.ModelIDAlt))
		if !ok {
			continue
		}
		modelID = normalizeGeminiCLIModelID(modelID)
		if modelID == "" || isIgnoredGeminiCLIModel(modelID) {
			continue
		}
		tokenType, _ := stringFromAny(firstAny(bucket.TokenType, bucket.TokenTypeAlt))
		remainingFraction, remainingFractionKnown := numberFromAny(firstAny(bucket.RemainingFraction, bucket.RemainingFractionAlt))
		remainingAmount, remainingAmountKnown := numberFromAny(firstAny(bucket.RemainingAmount, bucket.RemainingAmountAlt))
		resetTime, _ := stringFromAny(firstAny(bucket.ResetTime, bucket.ResetTimeAlt))
		parsedBucket := geminiCLIParsedBucket{modelID: modelID, tokenType: tokenType, resetTime: resetTime}
		if remainingFractionKnown {
			parsedBucket.remainingFraction = &remainingFraction
		} else if strings.HasSuffix(strings.TrimSpace(fmt.Sprint(firstAny(bucket.RemainingFraction, bucket.RemainingFractionAlt))), "%") {
			percentText := strings.TrimSuffix(strings.TrimSpace(fmt.Sprint(firstAny(bucket.RemainingFraction, bucket.RemainingFractionAlt))), "%")
			if percent, ok := numberFromAny(percentText); ok {
				fraction := percent / 100
				parsedBucket.remainingFraction = &fraction
			}
		}
		if remainingAmountKnown {
			parsedBucket.remainingAmount = &remainingAmount
		}
		parsed = append(parsed, parsedBucket)
	}
	return observationsFromGeminiCLIParsedBuckets(auth, parsed, observedAt)
}

func observationsFromGeminiCLIParsedBuckets(auth AuthSnapshot, buckets []geminiCLIParsedBucket, observedAt time.Time) []Observation {
	groupByModel := map[string]geminiCLIGroup{}
	order := map[string]int{}
	for index, group := range geminiCLIGroups {
		order[group.id] = index
		for _, modelID := range group.modelIDs {
			groupByModel[modelID] = group
		}
	}
	grouped := map[string]*geminiCLIGroupBucket{}
	for _, bucket := range buckets {
		group, ok := groupByModel[bucket.modelID]
		if !ok {
			group = geminiCLIGroup{id: bucket.modelID, label: bucket.modelID}
		}
		key := group.id + "::" + bucket.tokenType
		current := grouped[key]
		if current == nil {
			current = &geminiCLIGroupBucket{id: group.id, tokenType: bucket.tokenType, preferredModelID: group.preferredModelID}
			grouped[key] = current
		}
		current.modelIDs = append(current.modelIDs, bucket.modelID)
		current.fallbackRemainingFraction = minFloatPointer(current.fallbackRemainingFraction, bucket.remainingFraction)
		current.fallbackRemainingAmount = minFloatPointer(current.fallbackRemainingAmount, bucket.remainingAmount)
		current.fallbackResetTime = earlierProviderTime(current.fallbackResetTime, bucket.resetTime)
		if group.preferredModelID != "" && bucket.modelID == group.preferredModelID {
			copyBucket := bucket
			current.preferred = &copyBucket
		}
	}

	keys := sortedGroupKeys(grouped, order)
	observations := make([]Observation, 0, len(keys))
	for _, key := range keys {
		group := grouped[key]
		remainingFraction := group.fallbackRemainingFraction
		remainingAmount := group.fallbackRemainingAmount
		resetTime := group.fallbackResetTime
		if group.preferred != nil {
			remainingFraction = group.preferred.remainingFraction
			remainingAmount = group.preferred.remainingAmount
			resetTime = group.preferred.resetTime
		}
		if remainingFraction == nil {
			if remainingAmount != nil && *remainingAmount <= 0 {
				zero := 0.0
				remainingFraction = &zero
			} else if strings.TrimSpace(resetTime) != "" {
				zero := 0.0
				remainingFraction = &zero
			}
		}
		if remainingFraction == nil {
			continue
		}
		remaining, err := NormalizePercentage(*remainingFraction * 100)
		if err != nil {
			continue
		}
		window := group.tokenType
		if window == "" {
			window = "default"
		}
		resetAt, resetKnown := parseProviderTime(resetTime, observedAt)
		observation, err := (Observation{
			Identity: StateIdentity{
				AuthID:   auth.AuthID(),
				Provider: ProviderGeminiCLI,
				Resource: group.id,
				Window:   window,
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
		if err == nil {
			observations = append(observations, observation)
		}
	}
	return observations
}

func normalizeGeminiCLIModelID(value string) string {
	value = strings.TrimSpace(value)
	return strings.TrimSuffix(value, "_vertex")
}

func isIgnoredGeminiCLIModel(modelID string) bool {
	return modelID == "gemini-2.0-flash" || strings.HasPrefix(modelID, "gemini-2.0-flash-")
}
