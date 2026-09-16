package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/buildinfo"
	"github.com/therealtinhtute/llmhub/internal/cmd"
)

// This file ports upstream CLIProxyAPI's `discover` entry points
// (3428110d49be; flag/JSON-cleanliness hardening from 13af6c002bd0 and
// 7d687054329d). Two invocation styles are supported:
//   - `llmhub discover [flags]` — early positional command dispatched ahead of
//     the startup banner and Postgres loading via dispatchEarlyCommand.
//   - `llmhub --discover [--discover-json] ...` — legacy flag mode handled
//     after flag.Parse, still before runtime config loading.

// runDiscoverCommand implements `llmhub discover` (R11): a one-shot LAN scan
// for advertised AI gateways. The banner goes to stderr and logrus is moved to
// stderr so `--json` output stays machine-readable on stdout.
// Exit 0 success, 1 scan error, 2 usage.
func runDiscoverCommand(args []string) int {
	log.SetOutput(os.Stderr)
	discoverFlags := flag.NewFlagSet("discover", flag.ContinueOnError)
	discoverFlags.SetOutput(os.Stderr)
	timeoutSec := discoverFlags.Int("timeout", 3, "Discovery timeout in seconds")
	jsonOut := discoverFlags.Bool("json", false, "Output in JSON format")
	serviceType := discoverFlags.String("service-type", "", "DNS-SD service type (default _ai-gateway._tcp)")
	configPath := discoverFlags.String("config", DefaultConfigPath, "Configure File Path")
	var include, exclude []string
	discoverFlags.Func("include", "Comma-separated interface names to scan (overrides default physical LAN filter)", appendCSV(&include))
	discoverFlags.Func("exclude", "Comma-separated interface names to skip", appendCSV(&exclude))
	if err := discoverFlags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintf(os.Stderr, "LLMHub Version: %s, Commit: %s, BuiltAt: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
	}
	cfgInclude, cfgExclude := cmd.LoadDiscoveryScanFilters(*configPath)
	include, exclude = cmd.ResolveDiscoveryInterfaceFilters(include, exclude, cfgInclude, cfgExclude)
	return cmd.DoDiscoverWithOptions(cmd.DiscoverOptions{
		Timeout:     time.Duration(*timeoutSec) * time.Second,
		JSONOutput:  *jsonOut,
		ServiceType: *serviceType,
		Include:     include,
		Exclude:     exclude,
	})
}

// runDiscoverFlags implements the legacy `--discover`/`--discover-json` flag
// mode with the same option resolution as the positional command.
func runDiscoverFlags(timeoutSec int, jsonOutput bool, serviceType, configPath string, include, exclude []string) int {
	log.SetOutput(os.Stderr)
	cfgInclude, cfgExclude := cmd.LoadDiscoveryScanFilters(configPath)
	include, exclude = cmd.ResolveDiscoveryInterfaceFilters(include, exclude, cfgInclude, cfgExclude)
	return cmd.DoDiscoverWithOptions(cmd.DiscoverOptions{
		Timeout:     time.Duration(timeoutSec) * time.Second,
		JSONOutput:  jsonOutput,
		ServiceType: serviceType,
		Include:     include,
		Exclude:     exclude,
	})
}

func appendCSV(dst *[]string) func(string) error {
	return func(raw string) error {
		*dst = append(*dst, cmd.ParseInterfaceList(raw)...)
		return nil
	}
}

// argvEnablesBoolFlag pre-scans argv for a boolean flag so JSON-mode discovery
// can keep stdout clean before flag.Parse runs. Scanning stops at the first
// non-flag argument or "--" terminator, and a following token is skipped when
// the preceding flag is known to consume a separate value.
func argvEnablesBoolFlag(args []string, name string) bool {
	enabled := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			break
		}
		flagName, value, hasValue := splitArgvFlag(arg)
		if flagName == name {
			if !hasValue {
				enabled = true
			} else if parsed, errParse := strconv.ParseBool(value); errParse == nil {
				enabled = parsed
			}
		}
		if !hasValue && argvFlagConsumesValue(flagName) {
			if i+1 < len(args) && args[i+1] != "--" {
				i++
			}
		}
	}
	return enabled
}

// argvFlagConsumesValue reports whether the named llmhub flag takes a separate
// value token. Boolean flags return false; every other known flag consumes one.
func argvFlagConsumesValue(name string) bool {
	switch name {
	case "login", "codex-login", "codex-device-login", "claude-login", "no-browser",
		"antigravity-login", "kimi-login", "xai-login",
		"discover", "discover-json",
		"tui", "standalone", "local-model":
		return false
	default:
		return name != ""
	}
}

func splitArgvFlag(arg string) (name, value string, hasValue bool) {
	if !strings.HasPrefix(arg, "-") {
		return "", "", false
	}
	arg = strings.TrimPrefix(arg, "-")
	arg = strings.TrimPrefix(arg, "-")
	name, value, hasValue = strings.Cut(arg, "=")
	return name, value, hasValue
}
