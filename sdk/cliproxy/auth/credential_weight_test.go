package auth

import (
	"encoding/json"
	"testing"
)

func TestCredentialWeightDefaultsAndParses(t *testing.T) {
	if got := CredentialWeight(nil); got != DefaultCredentialWeight {
		t.Fatalf("CredentialWeight(nil) = %d, want %d", got, DefaultCredentialWeight)
	}
	auth := &Auth{Metadata: map[string]any{"weight": json.Number("7")}}
	if got := CredentialWeight(auth); got != 7 {
		t.Fatalf("CredentialWeight() = %d, want 7", got)
	}
	auth.Attributes = map[string]string{"weight": "9"}
	if got := CredentialWeight(auth); got != 9 {
		t.Fatalf("CredentialWeight() with attrs = %d, want 9", got)
	}
}

func TestValidateCredentialWeightRejectsInvalidValues(t *testing.T) {
	for _, weight := range []int64{0, -1, MaxCredentialWeight + 1} {
		if err := ValidateCredentialWeight(weight); err == nil {
			t.Fatalf("ValidateCredentialWeight(%d) error = nil", weight)
		}
	}
	if err := ValidateCredentialWeight(DefaultCredentialWeight); err != nil {
		t.Fatalf("ValidateCredentialWeight(default) error = %v", err)
	}
}

func TestApplyCredentialWeightFromMetadata(t *testing.T) {
	auth := &Auth{Metadata: map[string]any{"weight": "5"}}
	if err := ApplyCredentialWeightFromMetadata(auth); err != nil {
		t.Fatalf("ApplyCredentialWeightFromMetadata() error = %v", err)
	}
	if got := auth.Attributes["weight"]; got != "5" {
		t.Fatalf("weight attr = %q, want 5", got)
	}
	auth.Metadata["weight"] = float64(1)
	if err := ApplyCredentialWeightFromMetadata(auth); err != nil {
		t.Fatalf("ApplyCredentialWeightFromMetadata(default) error = %v", err)
	}
	if _, ok := auth.Attributes["weight"]; ok {
		t.Fatalf("default weight should not be stored in attributes: %#v", auth.Attributes)
	}
	auth.Metadata["weight"] = float64(1.5)
	if err := ApplyCredentialWeightFromMetadata(auth); err == nil {
		t.Fatalf("fractional weight error = nil")
	}
}
