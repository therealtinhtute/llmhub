package common

import (
	"testing"

	"github.com/tidwall/gjson"
)

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4 ("preserve Gemini prompt
// cache by demoting mid-session developer messages").
func TestMergeAdjacentGeminiUserContents(t *testing.T) {
	t.Run("merges consecutive pure text user turns", func(t *testing.T) {
		contents := [][]byte{
			[]byte(`{"role":"user","parts":[{"text":"prompt 1"}]}`),
			[]byte(`{"role":"user","parts":[{"text":"prompt 2"}]}`),
		}
		merged := MergeAdjacentGeminiUserContents(contents)
		if len(merged) != 1 {
			t.Fatalf("expected 1 merged user turn, got %d", len(merged))
		}
		parts := gjson.GetBytes(merged[0], "parts").Array()
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(parts))
		}
	})

	t.Run("does not merge across functionResponse boundaries", func(t *testing.T) {
		contents := [][]byte{
			[]byte(`{"role":"user","parts":[{"functionResponse":{"name":"test","response":{"result":"ok"}}}]}`),
			[]byte(`{"role":"user","parts":[{"text":"user note"}]}`),
			[]byte(`{"role":"user","parts":[{"function_response":{"name":"test2","response":{"result":"ok2"}}}]}`),
		}
		merged := MergeAdjacentGeminiUserContents(contents)
		if len(merged) != 3 {
			t.Fatalf("expected 3 separate turns preserving functionResponse, got %d", len(merged))
		}
	})
}

// Ported from upstream CLIProxyAPI commit 4fde97f4144a ("reorder trailing text
// before function responses in user turns").
func TestReorderGeminiUserParts(t *testing.T) {
	t.Run("text after functionResponse is moved before it", func(t *testing.T) {
		parts := [][]byte{
			[]byte(`{"functionResponse":{"name":"run","response":{"result":"ok"}}}`),
			[]byte(`{"text":"system reminder"}`),
		}
		reordered := ReorderGeminiUserParts(parts)
		if len(reordered) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(reordered))
		}
		if !gjson.GetBytes(reordered[0], "text").Exists() {
			t.Fatalf("expected text part first, got %s", reordered[0])
		}
		if !gjson.GetBytes(reordered[1], "functionResponse").Exists() {
			t.Fatalf("expected functionResponse part last, got %s", reordered[1])
		}
	})

	t.Run("no reordering needed without trailing text", func(t *testing.T) {
		parts := [][]byte{
			[]byte(`{"text":"hello"}`),
			[]byte(`{"functionResponse":{"name":"run","response":{"result":"ok"}}}`),
		}
		reordered := ReorderGeminiUserParts(parts)
		if !gjson.GetBytes(reordered[0], "text").Exists() || !gjson.GetBytes(reordered[1], "functionResponse").Exists() {
			t.Fatalf("unexpected reorder: %v", reordered)
		}
	})
}

// Ported from upstream CLIProxyAPI v7.3.3 (internal/translator/common/gemini.go,
// MergeAdjacentGeminiContents): consecutive user turns merge, and the merge
// reorders text before functionResponse parts.
func TestMergeAdjacentGeminiContents(t *testing.T) {
	t.Run("merges consecutive user turns", func(t *testing.T) {
		contents := [][]byte{
			[]byte(`{"role":"user","parts":[{"text":"prompt 1"}]}`),
			[]byte(`{"role":"user","parts":[{"text":"prompt 2"}]}`),
		}
		merged := MergeAdjacentGeminiContents(contents)
		if len(merged) != 1 {
			t.Fatalf("expected 1 merged user turn, got %d", len(merged))
		}
	})

	t.Run("does not merge consecutive model turns", func(t *testing.T) {
		contents := [][]byte{
			[]byte(`{"role":"model","parts":[{"text":"a"}]}`),
			[]byte(`{"role":"model","parts":[{"text":"b"}]}`),
		}
		merged := MergeAdjacentGeminiContents(contents)
		if len(merged) != 2 {
			t.Fatalf("model turns must stay unmerged, got %d", len(merged))
		}
	})

	t.Run("merged user turn reorders text before functionResponse", func(t *testing.T) {
		contents := [][]byte{
			[]byte(`{"role":"user","parts":[{"functionResponse":{"name":"run","response":{"result":"ok"}}}]}`),
			[]byte(`{"role":"user","parts":[{"text":"reminder"}]}`),
		}
		merged := MergeAdjacentGeminiContents(contents)
		if len(merged) != 1 {
			t.Fatalf("expected merged user turn, got %d", len(merged))
		}
		parts := gjson.GetBytes(merged[0], "parts").Array()
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(parts))
		}
		if !parts[0].Get("text").Exists() || !parts[1].Get("functionResponse").Exists() {
			t.Fatalf("expected text before functionResponse, got %s", gjson.GetBytes(merged[0], "parts").Raw)
		}
	})
}
