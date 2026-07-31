package auth

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultCredentialWeight = int64(1)
	MaxCredentialWeight     = int64(1_000_000)
)

func CredentialWeight(auth *Auth) int64 {
	weight, ok := CredentialWeightValue(auth)
	if !ok {
		return DefaultCredentialWeight
	}
	return weight
}

func CredentialWeightValue(auth *Auth) (int64, bool) {
	if auth == nil {
		return 0, false
	}
	if auth.Attributes != nil {
		if weight, ok := ParseCredentialWeight(auth.Attributes["weight"]); ok {
			return weight, true
		}
	}
	if auth.Metadata != nil {
		if weight, ok := ParseCredentialWeight(auth.Metadata["weight"]); ok {
			return weight, true
		}
	}
	return 0, false
}

func ParseCredentialWeight(value any) (int64, bool) {
	switch typed := value.(type) {
	case nil:
		return 0, false
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if typed != float64(int64(typed)) {
			return 0, false
		}
		return int64(typed), true
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return i, true
		}
	case string:
		if i, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

func ValidateCredentialWeight(weight int64) error {
	if weight < DefaultCredentialWeight || weight > MaxCredentialWeight {
		return fmt.Errorf("credential weight must be between %d and %d", DefaultCredentialWeight, MaxCredentialWeight)
	}
	return nil
}

func ApplyCredentialWeightFromMetadata(auth *Auth) error {
	if auth == nil || len(auth.Metadata) == 0 {
		return nil
	}
	raw, exists := auth.Metadata["weight"]
	if !exists || raw == nil {
		return nil
	}
	weight, ok := ParseCredentialWeight(raw)
	if !ok {
		return fmt.Errorf("credential weight must be an integer")
	}
	if err := ValidateCredentialWeight(weight); err != nil {
		return err
	}
	if auth.Attributes == nil {
		auth.Attributes = make(map[string]string)
	}
	if weight == DefaultCredentialWeight {
		delete(auth.Attributes, "weight")
		return nil
	}
	auth.Attributes["weight"] = strconv.FormatInt(weight, 10)
	return nil
}
