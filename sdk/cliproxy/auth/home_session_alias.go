package auth

import (
	"strings"
	"sync"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	cliproxysession "github.com/therealtinhtute/llmhub/sdk/cliproxy/session"
)

const defaultHomeSessionAliasTTL = time.Hour

type homeSessionAliasEntry struct {
	canonical string
	expiresAt time.Time
	aliases   []string
}

type homeSessionAliasCache struct {
	mu      sync.Mutex
	entries map[string]homeSessionAliasEntry
}

func (c *homeSessionAliasCache) canonical(primary, fallback string, ttl time.Duration, now time.Time) string {
	primary = strings.TrimSpace(primary)
	fallback = strings.TrimSpace(fallback)
	if primary == "" {
		return ""
	}
	if ttl <= 0 {
		ttl = defaultHomeSessionAliasTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]homeSessionAliasEntry)
	}
	canonical := primary
	aliases := compactHomeSessionAliases(primary, fallback)
	if existing, ok := c.entryLocked(primary, now); ok {
		canonical = existing.canonical
		aliases = compactHomeSessionAliases(append(aliases, existing.aliases...)...)
	}
	if fallback != "" && fallback != primary {
		if existing, ok := c.entryLocked(fallback, now); ok {
			if canonical == primary {
				canonical = existing.canonical
			}
			aliases = compactHomeSessionAliases(append(aliases, existing.aliases...)...)
		}
	}
	aliases = compactHomeSessionAliases(append(aliases, canonical)...)
	entry := homeSessionAliasEntry{canonical: canonical, expiresAt: now.Add(ttl), aliases: aliases}
	for _, alias := range aliases {
		c.entries[alias] = entry
	}
	return canonical
}

func (c *homeSessionAliasCache) entryLocked(alias string, now time.Time) (homeSessionAliasEntry, bool) {
	entry, ok := c.entries[alias]
	if !ok {
		return homeSessionAliasEntry{}, false
	}
	if now.Before(entry.expiresAt) {
		return entry, true
	}
	for _, stale := range entry.aliases {
		delete(c.entries, stale)
	}
	return homeSessionAliasEntry{}, false
}

func (c *homeSessionAliasCache) clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries = nil
	c.mu.Unlock()
}

func compactHomeSessionAliases(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	aliases := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		aliases = append(aliases, value)
	}
	return aliases
}

func homeSessionAliasTTL(cfg *internalconfig.Config) time.Duration {
	if cfg == nil {
		return defaultHomeSessionAliasTTL
	}
	raw := strings.TrimSpace(cfg.Routing.SessionAffinityTTL)
	if raw == "" {
		return defaultHomeSessionAliasTTL
	}
	parsed, errParse := time.ParseDuration(raw)
	if errParse != nil || parsed <= 0 {
		return defaultHomeSessionAliasTTL
	}
	return parsed
}

// isHierarchyParent reports whether fallback is a parent lineage for primary.
// Same-namespace siblings (including bare IDs) and :agent: subagent paths both
// count as hierarchy, which keeps fork and subagent parents out of alias groups.
func isHierarchyParent(primary, fallback string) bool {
	if fallback == "" || primary == "" || primary == fallback {
		return false
	}
	if strings.Contains(primary, ":agent:") {
		return true
	}
	idx1 := strings.Index(primary, ":")
	idx2 := strings.Index(fallback, ":")
	if idx1 > 0 && idx2 > 0 && primary[:idx1] == fallback[:idx2] {
		return true
	}
	if idx1 == -1 && idx2 == -1 {
		return true
	}
	return false
}

// homeDispatchSessionIDs resolves the canonical session identity and any parent
// lineage used for Home dispatch and usage reporting. Parent lineage comes from
// the explicit extraction fallback or request metadata, with self-referential
// loops suppressed.
func (m *Manager) homeDispatchSessionIDs(opts cliproxyexecutor.Options) (string, string) {
	primary, fallback := extractExplicitSessionIDs(opts.Headers, opts.OriginalRequest, opts.Metadata)
	hasAuthoritativeInput := primary != ""
	if primary == "" {
		if canonicalID, ok := opts.Metadata[cliproxyexecutor.CanonicalSessionIDMetadataKey].(string); ok && strings.TrimSpace(canonicalID) != "" {
			primary = strings.TrimSpace(canonicalID)
		} else if lcpID, ok := opts.Metadata[cliproxyexecutor.LCPAffinitySessionIDMetadataKey].(string); ok && strings.TrimSpace(lcpID) != "" {
			primary = strings.TrimSpace(lcpID)
		} else {
			primary, fallback = extractSessionIDs(opts.Headers, opts.OriginalRequest, opts.Metadata)
			hasAuthoritativeInput = primary != ""
		}
	}
	if primary == "" || m == nil {
		return primary, ""
	}

	var parentSessionID string
	var aliasFallback string
	if fallback != "" && fallback != primary {
		if isHierarchyParent(primary, fallback) {
			parentSessionID = fallback
		} else {
			aliasFallback = fallback
		}
	}
	if !hasAuthoritativeInput && parentSessionID == "" && opts.Metadata != nil {
		if metaParent, ok := opts.Metadata[cliproxyexecutor.ParentSessionIDMetadataKey].(string); ok && strings.TrimSpace(metaParent) != "" {
			parentSessionID = strings.TrimSpace(metaParent)
		}
	}

	cfg, _ := m.runtimeConfig.Load().(*internalconfig.Config)
	ttl := homeSessionAliasTTL(cfg)
	now := time.Now()
	canonical := m.homeSessionAliases.canonical(primary, aliasFallback, ttl, now)
	if parentSessionID != "" {
		if parentSessionID == canonical || parentSessionID == primary || (aliasFallback != "" && parentSessionID == aliasFallback) {
			parentSessionID = ""
		} else {
			parentSessionID = m.homeSessionAliases.canonical(parentSessionID, "", ttl, now)
			if parentSessionID == canonical {
				parentSessionID = ""
			}
		}
	}
	canonical = cliproxysession.BoundSessionIdentity(canonical)
	if parentSessionID != "" {
		parentSessionID = cliproxysession.BoundSessionIdentity(parentSessionID)
	}
	if canonical == parentSessionID {
		parentSessionID = ""
	}
	return canonical, parentSessionID
}

func (m *Manager) homeDispatchSessionID(opts cliproxyexecutor.Options) string {
	sessionID, _ := m.homeDispatchSessionIDs(opts)
	return sessionID
}
