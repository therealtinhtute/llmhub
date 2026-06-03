# Execplan

## Steps

1. Identify the Codex OAuth callback path in management UI/TUI mode.
2. Replace auth-dir marker-file handoff with session-store callback payloads.
3. Update provider goroutines and redirect routes to use the new handoff.
4. Add regression coverage for missing synthetic auth-dir behavior.
5. Run focused and broader Go verification.

## Status

Implemented.
