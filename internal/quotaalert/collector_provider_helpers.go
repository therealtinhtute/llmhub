package quotaalert

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

func snapshotString(auth AuthSnapshot, key string) (string, bool) {
	if value, ok := auth.Attribute(key); ok {
		value = strings.TrimSpace(value)
		return value, value != ""
	}
	if value, ok := auth.Metadata(key); ok {
		switch typed := value.(type) {
		case string:
			typed = strings.TrimSpace(typed)
			return typed, typed != ""
		case json.Number:
			text := strings.TrimSpace(typed.String())
			return text, text != ""
		case fmt.Stringer:
			text := strings.TrimSpace(typed.String())
			return text, text != ""
		}
	}
	return "", false
}

func numberFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, !math.IsNaN(typed) && !math.IsInf(typed, 0)
	case float32:
		value := float64(typed)
		return value, !math.IsNaN(value) && !math.IsInf(value, 0)
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	default:
		return 0, false
	}
}

func quotaFractionFromAny(value any) (float64, bool) {
	if text, ok := value.(string); ok {
		trimmed := strings.TrimSpace(text)
		if strings.HasSuffix(trimmed, "%") {
			percent, ok := numberFromAny(strings.TrimSuffix(trimmed, "%"))
			return percent / 100, ok
		}
	}
	return numberFromAny(value)
}

func boolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y", "on":
			return true, true
		case "false", "0", "no", "n", "off":
			return false, true
		}
	case float64:
		if typed == 0 || typed == 1 {
			return typed != 0, true
		}
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil && (parsed == 0 || parsed == 1) {
			return parsed != 0, true
		}
	}
	return false, false
}

func stringFromAny(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		return trimmed, trimmed != ""
	case json.Number:
		text := strings.TrimSpace(typed.String())
		return text, text != ""
	case fmt.Stringer:
		text := strings.TrimSpace(typed.String())
		return text, text != ""
	default:
		return "", false
	}
}

func parseProviderTime(value string, now time.Time) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), true
		}
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds > 0 {
		return time.Unix(seconds, 0).UTC(), true
	}
	return time.Time{}, false
}

func parseUnixOrRelativeReset(resetAt, resetAfter any, now time.Time) (time.Time, bool) {
	if seconds, ok := numberFromAny(resetAt); ok && seconds > 0 {
		return time.Unix(int64(seconds), 0).UTC(), true
	}
	if seconds, ok := numberFromAny(resetAfter); ok && seconds > 0 {
		return now.Add(time.Duration(seconds) * time.Second).UTC(), true
	}
	return time.Time{}, false
}

func minFloatPointer(current, next *float64) *float64 {
	if current == nil {
		return next
	}
	if next == nil {
		return current
	}
	if *next < *current {
		return next
	}
	return current
}

func earlierProviderTime(current, next string) string {
	if strings.TrimSpace(current) == "" {
		return next
	}
	if strings.TrimSpace(next) == "" {
		return current
	}
	currentTime, currentOK := parseProviderTime(current, time.Time{})
	nextTime, nextOK := parseProviderTime(next, time.Time{})
	if !currentOK {
		return next
	}
	if !nextOK {
		return current
	}
	if currentTime.Before(nextTime) || currentTime.Equal(nextTime) {
		return current
	}
	return next
}

func sortedGroupKeys[T any](grouped map[string]T, order map[string]int) []string {
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		leftID := strings.Split(keys[left], "::")[0]
		rightID := strings.Split(keys[right], "::")[0]
		leftOrder, leftOK := order[leftID]
		rightOrder, rightOK := order[rightID]
		if leftOK && rightOK && leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		if leftOK != rightOK {
			return leftOK
		}
		return keys[left] < keys[right]
	})
	return keys
}

func slugID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
