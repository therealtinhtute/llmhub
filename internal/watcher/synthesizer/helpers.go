package synthesizer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/watcher/diff"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// StableIDGenerator generates stable, deterministic IDs for auth entries.
// It uses SHA256 hashing with collision handling via counters.
// It is not safe for concurrent use.
type StableIDGenerator struct {
	counters map[string]int
}

// NewStableIDGenerator creates a new StableIDGenerator instance.
func NewStableIDGenerator() *StableIDGenerator {
	return &StableIDGenerator{counters: make(map[string]int)}
}

// Next generates a stable ID based on the kind and parts.
// Returns the full ID (kind:hash) and the short hash portion.
func (g *StableIDGenerator) Next(kind string, parts ...string) (string, string) {
	if g == nil {
		return kind + ":000000000000", "000000000000"
	}
	hasher := sha256.New()
	hasher.Write([]byte(kind))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		hasher.Write([]byte{0})
		hasher.Write([]byte(trimmed))
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if len(digest) < 12 {
		digest = fmt.Sprintf("%012s", digest)
	}
	short := digest[:12]
	key := kind + ":" + short
	index := g.counters[key]
	g.counters[key] = index + 1
	if index > 0 {
		short = fmt.Sprintf("%s-%d", short, index)
	}
	return fmt.Sprintf("%s:%s", kind, short), short
}

// ApplyAuthExcludedModelsMeta applies excluded models metadata to an auth entry.
// It computes a hash of credential-specific excluded models and sets the auth_kind attribute.
// Nonempty per-account rules override the global provider rules; credentials without their own
// rules inherit global provider exclusions at runtime without baking them into auth attributes.
// Existing excluded_models attributes are never cleared, so callers rebuilding them on an
// existing auth must drop stale values first.
func ApplyAuthExcludedModelsMeta(auth *coreauth.Auth, perKey []string, authKind string) {
	if auth == nil {
		return
	}
	seen := make(map[string]struct{})
	add := func(list []string) {
		for _, entry := range list {
			if trimmed := strings.TrimSpace(entry); trimmed != "" {
				key := strings.ToLower(trimmed)
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
			}
		}
	}
	add(perKey)
	combined := make([]string, 0, len(seen))
	for k := range seen {
		combined = append(combined, k)
	}
	sort.Strings(combined)
	hash := diff.ComputeExcludedModelsHash(combined)
	if auth.Attributes == nil {
		auth.Attributes = make(map[string]string)
	}
	if hash != "" {
		auth.Attributes["excluded_models_hash"] = hash
	}
	// Store only the credential's own rules so global config changes apply without rewriting credentials.
	if len(combined) > 0 {
		auth.Attributes["excluded_models"] = strings.Join(combined, ",")
	}
	if authKind != "" {
		auth.Attributes["auth_kind"] = authKind
	}
}

// addRequestRetryToMetadata copies a per-credential request-retry override into metadata.
// Nil or negative values are treated as unset and are not written.
func addRequestRetryToMetadata(requestRetry *int, metadata map[string]any) {
	if requestRetry == nil || *requestRetry < 0 || metadata == nil {
		return
	}
	metadata["request_retry"] = *requestRetry
}

// addRequestScopedErrorsToMetadata copies per-credential request-scoped error rules into metadata.
func addRequestScopedErrorsToMetadata(rules []config.RequestScopedErrorRule, metadata map[string]any) {
	if len(rules) == 0 || metadata == nil {
		return
	}
	metadata["request_scoped_errors"] = rules
}

// addConfigHeadersToAttrs adds header configuration to auth attributes.
// Headers are prefixed with "header:" in the attributes map.
func addConfigHeadersToAttrs(headers map[string]string, attrs map[string]string) {
	if len(headers) == 0 || attrs == nil {
		return
	}
	for hk, hv := range headers {
		key := strings.TrimSpace(hk)
		val := strings.TrimSpace(hv)
		if key == "" || val == "" {
			continue
		}
		attrs["header:"+key] = val
	}
}
