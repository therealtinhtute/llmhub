package util

import (
	"fmt"
	"regexp"
	"sync/atomic"
	"time"
)

var (
	claudeToolUseIDSanitizer    = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	claudeFunctionNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	claudeToolUseIDCounter      uint64
)

// SanitizeClaudeToolID ensures the given id conforms to Claude's
// tool_use.id regex ^[a-zA-Z0-9_-]+$.  Non-conforming characters are
// replaced with '_'; an empty result gets a generated fallback.
func SanitizeClaudeToolID(id string) string {
	s := claudeToolUseIDSanitizer.ReplaceAllString(id, "_")
	if s == "" {
		s = fmt.Sprintf("toolu_%d_%d", time.Now().UnixNano(), atomic.AddUint64(&claudeToolUseIDCounter, 1))
	}
	return s
}

// SanitizeClaudeFunctionName ensures a function or tool name conforms to Claude's
// requirements: ^[a-zA-Z0-9_-]{1,64}$. Non-conforming characters (including dots and colons
// commonly emitted by MCP tools) are replaced with '_'.
func SanitizeClaudeFunctionName(name string) string {
	if name == "" {
		return ""
	}
	s := claudeFunctionNameSanitizer.ReplaceAllString(name, "_")
	if len(s) > 64 {
		s = s[:64]
	}
	if s == "" {
		s = "_"
	}
	return s
}
