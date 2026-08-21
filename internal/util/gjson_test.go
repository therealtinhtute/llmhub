package util

import (
	"testing"
	"unsafe"
)

func TestGetGJSONBytesNoCopy(t *testing.T) {
	input := []byte(`{"request":{"contents":[{"role":"user"}]}}`)
	result := GetGJSONBytesNoCopy(input, "request.contents.0.role")
	if result.String() != "user" {
		t.Fatalf("result = %s, want user", result.Raw)
	}
}

func TestGetGJSONBytesNoCopyEmptyInput(t *testing.T) {
	if result := GetGJSONBytesNoCopy(nil, "request"); result.Exists() {
		t.Fatalf("empty input result = %s, want missing", result.Raw)
	}
}

func TestParseGJSONBytesNoCopy(t *testing.T) {
	input := []byte(`{"request":{"contents":[{"role":"user"}]}}`)
	root := ParseGJSONBytesNoCopy(input)
	if !root.IsObject() || root.Get("request.contents.0.role").String() != "user" {
		t.Fatalf("parsed root = %s, want user content array", root.Raw)
	}
}

func TestParseGJSONBytesNoCopyReferencesInput(t *testing.T) {
	input := []byte(`{"contents":[{"role":"user"}]}`)
	root := ParseGJSONBytesNoCopy(input)
	if len(root.Raw) != len(input) {
		t.Fatalf("raw length = %d, want %d", len(root.Raw), len(input))
	}
	if unsafe.StringData(root.Raw) != unsafe.SliceData(input) {
		t.Fatal("parsed result copied the input instead of referencing it")
	}
}

func TestParseGJSONBytesNoCopyEmptyInput(t *testing.T) {
	if result := ParseGJSONBytesNoCopy(nil); result.Exists() {
		t.Fatalf("empty input result = %s, want missing", result.Raw)
	}
}
