package tui

// i18n provides a simple string lookup for the TUI.

var currentLocale = "en"

// SetLocale is a no-op kept for API compatibility; only English is supported.
func SetLocale(locale string) {}

// CurrentLocale returns the active locale code.
func CurrentLocale() string {
	return currentLocale
}

// ToggleLocale is a no-op kept for API compatibility; only English is supported.
func ToggleLocale() {}

// T returns the string for the given key.
func T(key string) string {
	if v, ok := enStrings[key]; ok {
		return v
	}
	return key
}

var enTabNames = []string{"Dashboard", "Config", "Auth Files", "API Keys", "OAuth", "Logs"}

// TabNames returns tab names.
func TabNames() []string {
	return enTabNames
}

var enStrings = map[string]string{
	// ── Common ──
	"loading":      "Loading...",
	"refresh":      "Refresh",
	"save":         "Save",
	"cancel":       "Cancel",
	"confirm":      "Confirm",
	"yes":          "Yes",
	"no":           "No",
	"error":        "Error",
	"success":      "Success",
	"navigate":     "Navigate",
	"scroll":       "Scroll",
	"enter_save":   "Enter: Save",
	"esc_cancel":   "Esc: Cancel",
	"enter_submit": "Enter: Submit",
	"press_r":      "[r] Refresh",
	"press_scroll": "[↑↓] Scroll",
	"not_set":      "(not set)",
	"error_prefix": "⚠ Error: ",

	// ── Status bar ──
	"status_left":                 " LLMHub Management TUI",
	"status_right":                "Tab/Shift+Tab: switch • q/Ctrl+C: quit ",
	"initializing_tui":            "Initializing...",
	"auth_gate_title":             "🔐 Connect Management API",
	"auth_gate_help":              " Enter management password and press Enter to connect",
	"auth_gate_password":          "Password",
	"auth_gate_enter":             " Enter: connect • q/Ctrl+C: quit",
	"auth_gate_connecting":        "Connecting...",
	"auth_gate_connect_fail":      "Connection failed: %s",
	"auth_gate_password_required": "password is required",

	// ── Dashboard ──
	"dashboard_title":  "📊 Dashboard",
	"dashboard_help":   " [r] Refresh • [↑↓] Scroll",
	"connected":        "● Connected",
	"mgmt_keys":        "Mgmt Keys",
	"auth_files_label": "Auth Files",
	"active_suffix":    "active",
	"total_requests":   "Requests",
	"success_label":    "Success",
	"failure_label":    "Failed",
	"total_tokens":     "Total Tokens",
	"current_config":   "Current Config",
	"debug_mode":       "Debug Mode",
	"usage_stats":      "Usage Statistics",
	"log_to_file":      "Log to File",
	"retry_count":      "Retry Count",
	"proxy_url":        "Proxy URL",
	"routing_strategy": "Routing Strategy",
	"model_stats":      "Model Stats",
	"model":            "Model",
	"requests":         "Requests",
	"tokens":           "Tokens",
	"bool_yes":         "Yes ✓",
	"bool_no":          "No",

	// ── Config ──
	"config_title":      "⚙ Configuration",
	"config_help1":      "  [↑↓/jk] Navigate • [Enter/Space] Edit • [r] Refresh",
	"config_help2":      "  Bool: Enter to toggle • String/Int: Enter to type, Enter to confirm, Esc to cancel",
	"updated_ok":        "✓ Updated successfully",
	"no_config":         "  No configuration loaded",
	"invalid_int":       "invalid integer",
	"section_server":    "Server",
	"section_logging":   "Logging & Stats",
	"section_quota":     "Quota Exceeded Handling",
	"section_routing":   "Routing",
	"section_websocket": "WebSocket",
	"section_other":     "Other",

	// ── Auth Files ──
	"auth_title":      "🔑 Auth Files",
	"auth_help1":      " [↑↓/jk] Navigate • [Enter] Expand • [e] Enable/Disable • [d] Delete • [r] Refresh",
	"auth_help2":      " [1] Edit prefix • [2] Edit proxy_url • [3] Edit priority",
	"no_auth_files":   "  No auth files found",
	"confirm_delete":  "⚠ Delete %s? [y/n]",
	"deleted":         "Deleted %s",
	"enabled":         "Enabled",
	"disabled":        "Disabled",
	"updated_field":   "Updated %s on %s",
	"status_active":   "active",
	"status_disabled": "disabled",

	// ── API Keys ──
	"keys_title":         "🔐 API Keys",
	"keys_help":          " [↑↓/jk] Navigate • [a] Add • [e] Edit • [d] Delete • [c] Copy • [r] Refresh",
	"no_keys":            "  No API Keys. Press [a] to add",
	"access_keys":        "Access API Keys",
	"confirm_delete_key": "⚠ Delete %s? [y/n]",
	"key_added":          "API Key added",
	"key_updated":        "API Key updated",
	"key_deleted":        "API Key deleted",
	"copied":             "✓ Copied to clipboard",
	"copy_failed":        "✗ Copy failed",
	"new_key_prompt":     "  New Key: ",
	"edit_key_prompt":    "  Edit Key: ",
	"enter_add":          "    Enter: Add • Esc: Cancel",
	"enter_save_esc":     "    Enter: Save • Esc: Cancel",

	// ── OAuth ──
	"oauth_title":        "🔐 OAuth Login",
	"oauth_select":       "  Select a provider and press [Enter] to start OAuth login:",
	"oauth_help":         "  [↑↓/jk] Navigate • [Enter] Login • [Esc] Clear status",
	"oauth_initiating":   "⏳ Initiating %s login...",
	"oauth_success":      "Authentication successful! Refresh Auth Files tab to see the new credential.",
	"oauth_completed":    "Authentication flow completed.",
	"oauth_failed":       "Authentication failed",
	"oauth_timeout":      "OAuth flow timed out (5 minutes)",
	"oauth_press_esc":    "  Press [Esc] to cancel",
	"oauth_auth_url":     "  Authorization URL:",
	"oauth_remote_hint":  "  Remote browser mode: Open the URL above in browser, paste the callback URL below after authorization.",
	"oauth_callback_url": "  Callback URL:",
	"oauth_press_c":      "  Press [c] to enter callback URL • [Esc] to go back",
	"oauth_submitting":   "⏳ Submitting callback...",
	"oauth_submit_ok":    "✓ Callback submitted, waiting...",
	"oauth_submit_fail":  "✗ Callback submission failed",
	"oauth_waiting":      "  Waiting for authentication...",

	// ── Usage ──
	"usage_title":         "📈 Usage Statistics",
	"usage_help":          " [r] Refresh • [↑↓] Scroll",
	"usage_no_data":       "  Usage data not available",
	"usage_total_reqs":    "Total Requests",
	"usage_total_tokens":  "Total Tokens",
	"usage_success":       "Success",
	"usage_failure":       "Failed",
	"usage_total_token_l": "Total Tokens",
	"usage_rpm":           "RPM",
	"usage_tpm":           "TPM",
	"usage_req_by_hour":   "Requests by Hour",
	"usage_tok_by_hour":   "Token Usage by Hour",
	"usage_req_by_day":    "Requests by Day",
	"usage_api_detail":    "API Detail Statistics",
	"usage_input":         "Input",
	"usage_output":        "Output",
	"usage_cached":        "Cached",
	"usage_reasoning":     "Reasoning",
	"usage_time":          "Time",

	// ── Logs ──
	"logs_title":       "📋 Logs",
	"logs_auto_scroll": "● AUTO-SCROLL",
	"logs_paused":      "○ PAUSED",
	"logs_filter":      "Filter",
	"logs_lines":       "Lines",
	"logs_help":        " [a] Auto-scroll • [c] Clear • [1] All [2] info+ [3] warn+ [4] error • [↑↓] Scroll",
	"logs_waiting":     "  Waiting for log output...",
}
