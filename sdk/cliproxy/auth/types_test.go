package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRequestRetryOverride(t *testing.T) {
	var unset *Auth
	if got, ok := unset.RequestRetryOverride(); ok || got != 0 {
		t.Fatalf("nil auth override = (%d, %t), want (0, false)", got, ok)
	}

	auth := &Auth{}
	if got, ok := auth.RequestRetryOverride(); ok || got != 0 {
		t.Fatalf("empty auth override = (%d, %t), want (0, false)", got, ok)
	}

	auth = &Auth{Metadata: map[string]any{"request_retry": 0}}
	if got, ok := auth.RequestRetryOverride(); !ok || got != 0 {
		t.Fatalf("request_retry=0 override = (%d, %t), want (0, true)", got, ok)
	}

	auth = &Auth{Metadata: map[string]any{"request_retry": 3}}
	if got, ok := auth.RequestRetryOverride(); !ok || got != 3 {
		t.Fatalf("request_retry=3 override = (%d, %t), want (3, true)", got, ok)
	}

	auth = &Auth{Metadata: map[string]any{"request_retry": -1}}
	if got, ok := auth.RequestRetryOverride(); ok || got != 0 {
		t.Fatalf("request_retry=-1 override = (%d, %t), want (0, false)", got, ok)
	}

	auth = &Auth{Metadata: map[string]any{"request-retry": 2}}
	if got, ok := auth.RequestRetryOverride(); !ok || got != 2 {
		t.Fatalf("request-retry=2 override = (%d, %t), want (2, true)", got, ok)
	}

	auth = &Auth{Metadata: map[string]any{"request-retry": -2}}
	if got, ok := auth.RequestRetryOverride(); ok || got != 0 {
		t.Fatalf("request-retry=-2 override = (%d, %t), want (0, false)", got, ok)
	}

	auth = &Auth{Metadata: map[string]any{"request_retry": "0"}}
	if got, ok := auth.RequestRetryOverride(); !ok || got != 0 {
		t.Fatalf("request_retry string 0 override = (%d, %t), want (0, true)", got, ok)
	}
}

func TestToolPrefixDisabled(t *testing.T) {
	var a *Auth
	if a.ToolPrefixDisabled() {
		t.Error("nil auth should return false")
	}

	a = &Auth{}
	if a.ToolPrefixDisabled() {
		t.Error("empty auth should return false")
	}

	a = &Auth{Metadata: map[string]any{"tool_prefix_disabled": true}}
	if !a.ToolPrefixDisabled() {
		t.Error("should return true when set to true")
	}

	a = &Auth{Metadata: map[string]any{"tool_prefix_disabled": "true"}}
	if !a.ToolPrefixDisabled() {
		t.Error("should return true when set to string 'true'")
	}

	a = &Auth{Metadata: map[string]any{"tool-prefix-disabled": true}}
	if !a.ToolPrefixDisabled() {
		t.Error("should return true with kebab-case key")
	}

	a = &Auth{Metadata: map[string]any{"tool_prefix_disabled": false}}
	if a.ToolPrefixDisabled() {
		t.Error("should return false when set to false")
	}
}

func TestEnsureIndexUsesCredentialIdentity(t *testing.T) {
	t.Parallel()

	geminiAuth := &Auth{
		Provider: "gemini",
		Attributes: map[string]string{
			"api_key": "shared-key",
			"source":  "config:gemini[abc123]",
		},
	}
	compatAuth := &Auth{
		Provider: "bohe",
		Attributes: map[string]string{
			"api_key":      "shared-key",
			"compat_name":  "bohe",
			"provider_key": "bohe",
			"source":       "config:bohe[def456]",
		},
	}
	geminiAltBase := &Auth{
		Provider: "gemini",
		Attributes: map[string]string{
			"api_key":  "shared-key",
			"base_url": "https://alt.example.com",
			"source":   "config:gemini[ghi789]",
		},
	}
	geminiDuplicate := &Auth{
		Provider: "gemini",
		Attributes: map[string]string{
			"api_key": "shared-key",
			"source":  "config:gemini[abc123-1]",
		},
	}

	geminiIndex := geminiAuth.EnsureIndex()
	compatIndex := compatAuth.EnsureIndex()
	altBaseIndex := geminiAltBase.EnsureIndex()
	duplicateIndex := geminiDuplicate.EnsureIndex()

	if geminiIndex == "" {
		t.Fatal("gemini index should not be empty")
	}
	if compatIndex == "" {
		t.Fatal("compat index should not be empty")
	}
	if altBaseIndex == "" {
		t.Fatal("alt base index should not be empty")
	}
	if duplicateIndex == "" {
		t.Fatal("duplicate index should not be empty")
	}
	if geminiIndex == compatIndex {
		t.Fatalf("shared api key produced duplicate auth_index %q", geminiIndex)
	}
	if geminiIndex == altBaseIndex {
		t.Fatalf("same provider/key with different base_url produced duplicate auth_index %q", geminiIndex)
	}
	if geminiIndex != duplicateIndex {
		t.Fatalf("same provider/key with different source should share auth_index, got %q vs %q", geminiIndex, duplicateIndex)
	}

	// Ported from upstream CLIProxyAPI commit d09042a54810 (types_test.go):
	// meta config keys must keep a stable auth_index across proxy/prefix/source
	// variations.
	metaAuth1 := &Auth{
		Provider: "meta",
		ID:       "meta:apikey:token1",
		Attributes: map[string]string{
			"api_key":   "meta-secret",
			"base_url":  "https://api.meta.ai/v1",
			"proxy_url": "socks5://127.0.0.1:1080",
			"prefix":    "fast-",
			"source":    "config:meta[token1]",
		},
	}
	metaAuth2 := &Auth{
		Provider: "meta",
		ID:       "meta:apikey:token2",
		Attributes: map[string]string{
			"api_key":   "meta-secret",
			"base_url":  "https://api.meta.ai/v1",
			"proxy_url": "",
			"prefix":    "",
			"source":    "config:meta[token2]",
		},
	}
	metaIndex1 := metaAuth1.EnsureIndex()
	metaIndex2 := metaAuth2.EnsureIndex()
	if metaIndex1 == "" {
		t.Fatal("meta index should not be empty")
	}
	if metaIndex1 != metaIndex2 {
		t.Fatalf("meta auth index should remain stable across proxy/prefix changes, got %q vs %q", metaIndex1, metaIndex2)
	}
}

func TestEnsureIndexUsesOAuthTypeAndAbsolutePath(t *testing.T) {
	t.Parallel()

	wd, errWd := os.Getwd()
	if errWd != nil {
		t.Fatalf("os.Getwd returned error: %v", errWd)
	}

	relPath := "test-oauth.json"
	absPath := filepath.Join(wd, relPath)
	expectedSeed := "gemini:" + filepath.Clean(absPath)
	expectedIndex := stableAuthIndex(expectedSeed)

	a := &Auth{
		Provider: "gemini-cli",
		Attributes: map[string]string{
			"path": relPath,
		},
		Metadata: map[string]any{
			"type": "gemini",
		},
	}

	got := a.EnsureIndex()
	if got == "" {
		t.Fatal("auth index should not be empty")
	}
	if got != expectedIndex {
		t.Fatalf("auth index = %q, want %q", got, expectedIndex)
	}
}

func TestRecentRequestsSnapshotEmptyReturnsTwentyBuckets(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).In(time.Local)
	a := &Auth{}

	got := a.RecentRequestsSnapshot(now)
	if len(got) != recentRequestBucketCount {
		t.Fatalf("len = %d, want %d", len(got), recentRequestBucketCount)
	}

	currentBucketID := now.Unix() / recentRequestBucketSeconds
	baseBucketID := currentBucketID - int64(recentRequestBucketCount-1)
	for i, bucket := range got {
		if bucket.Success != 0 || bucket.Failed != 0 {
			t.Fatalf("bucket[%d] counts = %d/%d, want 0/0", i, bucket.Success, bucket.Failed)
		}
		if strings.TrimSpace(bucket.Time) == "" {
			t.Fatalf("bucket[%d] time label is empty", i)
		}
		expectedBucketID := baseBucketID + int64(i)
		start := time.Unix(expectedBucketID*recentRequestBucketSeconds, 0).In(time.Local)
		end := start.Add(10 * time.Minute)
		expected := start.Format("15:04") + "-" + end.Format("15:04")
		if bucket.Time != expected {
			t.Fatalf("bucket[%d] time = %q, want %q", i, bucket.Time, expected)
		}
	}
}

func TestRecentRequestsSnapshotIncludesCounts(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).In(time.Local)
	a := &Auth{}

	a.recordRecentRequest(now, true)
	a.recordRecentRequest(now, false)

	got := a.RecentRequestsSnapshot(now)
	if len(got) != recentRequestBucketCount {
		t.Fatalf("len = %d, want %d", len(got), recentRequestBucketCount)
	}

	newest := got[len(got)-1]
	if newest.Success != 1 || newest.Failed != 1 {
		t.Fatalf("newest bucket = success=%d failed=%d, want 1/1", newest.Success, newest.Failed)
	}
}

func TestRecentRequestsSnapshotBucketAdvanceMovesCounts(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).In(time.Local)
	next := now.Add(10 * time.Minute)
	a := &Auth{}

	a.recordRecentRequest(now, true)
	a.recordRecentRequest(next, false)

	got := a.RecentRequestsSnapshot(next)
	if len(got) != recentRequestBucketCount {
		t.Fatalf("len = %d, want %d", len(got), recentRequestBucketCount)
	}

	secondNewest := got[len(got)-2]
	newest := got[len(got)-1]
	if secondNewest.Success != 1 || secondNewest.Failed != 0 {
		t.Fatalf("second newest bucket = success=%d failed=%d, want 1/0", secondNewest.Success, secondNewest.Failed)
	}
	if newest.Success != 0 || newest.Failed != 1 {
		t.Fatalf("newest bucket = success=%d failed=%d, want 0/1", newest.Success, newest.Failed)
	}
}
func TestAuthClone_EmptyMapsIsolation(t *testing.T) {
	orig := &Auth{
		Attributes:  map[string]string{},
		Metadata:    map[string]any{},
		ModelStates: map[string]*ModelState{},
	}

	cloned := orig.Clone()
	if cloned == nil {
		t.Fatal("Clone() returned nil")
	}

	cloned.Attributes["test"] = "val"
	cloned.Metadata["test"] = 123
	cloned.ModelStates["test"] = &ModelState{}

	if len(orig.Attributes) != 0 {
		t.Errorf("orig.Attributes modified, len = %d", len(orig.Attributes))
	}
	if len(orig.Metadata) != 0 {
		t.Errorf("orig.Metadata modified, len = %d", len(orig.Metadata))
	}
	if len(orig.ModelStates) != 0 {
		t.Errorf("orig.ModelStates modified, len = %d", len(orig.ModelStates))
	}
}

func TestAuthFileSnapshotBindingAndFreshness(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "account.json")
	data := []byte(`{"type":"codex","prefix":"pfx-1"}`)
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	auth := &Auth{
		ID:         "account.json",
		Attributes: map[string]string{AttributePath: filePath},
	}

	if auth.HasFileSnapshot() {
		t.Fatal("auth should not have file snapshot before binding")
	}
	if !auth.FileSnapshotMatchesDisk() {
		t.Fatal("unbound auth should report FileSnapshotMatchesDisk() true")
	}

	auth.BindFileSnapshot(filePath, data)
	if !auth.HasFileSnapshot() {
		t.Fatal("auth should have file snapshot after binding")
	}
	if !auth.FileSnapshotMatchesDisk() {
		t.Fatal("freshly bound snapshot should match disk")
	}

	// Test canonical JSON digest invariance (reordered keys and whitespace).
	reformatted := []byte("{\n  \"prefix\": \"pfx-1\",\n  \"type\": \"codex\"\n}\n")
	auth2 := &Auth{ID: "account2.json"}
	auth2.BindFileSnapshot(filePath, reformatted)
	if !auth.SameFileSnapshot(auth2) {
		t.Fatal("snapshots with equivalent JSON should have identical digests and match under SameFileSnapshot")
	}

	// Verify re-formatting on disk still matches snapshot.
	if err := os.WriteFile(filePath, reformatted, 0o600); err != nil {
		t.Fatal(err)
	}
	if !auth.FileSnapshotMatchesDisk() {
		t.Fatal("reformatted file on disk should still match canonical snapshot digest")
	}

	// Modify file content on disk -> snapshot is stale!
	modified := []byte(`{"type":"codex","prefix":"pfx-2"}`)
	if err := os.WriteFile(filePath, modified, 0o600); err != nil {
		t.Fatal(err)
	}
	if auth.FileSnapshotMatchesDisk() {
		t.Fatal("modified file on disk must cause FileSnapshotMatchesDisk() to return false")
	}

	// Remove file on disk -> snapshot is stale!
	if err := os.Remove(filePath); err != nil {
		t.Fatal(err)
	}
	if auth.FileSnapshotMatchesDisk() {
		t.Fatal("deleted file on disk must cause FileSnapshotMatchesDisk() to return false")
	}

	// Test committing/versioning clears unversioned snapshot status.
	auth.Generation = 1
	if auth.HasFileSnapshot() {
		t.Fatal("auth with Generation > 0 must not report HasFileSnapshot() true")
	}
	auth.Generation = 0
	auth.RegistrationEpoch = 1
	if auth.HasFileSnapshot() {
		t.Fatal("auth with RegistrationEpoch > 0 must not report HasFileSnapshot() true")
	}

	// Test ClearFileSnapshot.
	auth.RegistrationEpoch = 0
	auth.ClearFileSnapshot()
	if auth.HasFileSnapshot() {
		t.Fatal("auth after ClearFileSnapshot() must not report HasFileSnapshot() true")
	}
}
