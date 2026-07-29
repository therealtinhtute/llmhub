// Package main provides the entry point for the CLI Proxy API server.
// This server acts as a proxy that provides OpenAI/Gemini/Claude compatible API interfaces
// for CLI models, allowing CLI models to be used with tools and libraries designed for standard AI APIs.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	configaccess "github.com/therealtinhtute/llmhub/internal/access/config_access"
	"github.com/therealtinhtute/llmhub/internal/buildinfo"
	"github.com/therealtinhtute/llmhub/internal/cmd"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/logging"
	"github.com/therealtinhtute/llmhub/internal/misc"
	"github.com/therealtinhtute/llmhub/internal/quotaalert"
	"github.com/therealtinhtute/llmhub/internal/redisqueue"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/runtimepolicy"
	_ "github.com/therealtinhtute/llmhub/internal/translator"
	"github.com/therealtinhtute/llmhub/internal/tui"
	"github.com/therealtinhtute/llmhub/internal/util"
	sdkAuth "github.com/therealtinhtute/llmhub/sdk/auth"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy"
)

var (
	Version           = "dev"
	Commit            = "none"
	BuildDate         = "unknown"
	DefaultConfigPath = ""
)

// init initializes the shared logger setup.
func init() {
	logging.SetupBaseLogger()
	buildinfo.Version = Version
	buildinfo.Commit = Commit
	buildinfo.BuildDate = BuildDate
}

func configureQuotaAlertRuntime(builder *cliproxy.Builder, pgStore quotaalert.Store) {
	cipher, err := loadQuotaSecretCipherFromEnv()
	if err != nil {
		log.Warnf("quota alert Telegram disabled: %v", err)
	}
	builder.WithQuotaAlertStore(pgStore).WithQuotaAlertSecretCipher(cipher)
}

// main is the entry point of the application.
// It parses command-line flags, loads configuration, and starts the appropriate
// service based on the provided flags (login, codex-login, or server mode).
func main() {
	fmt.Printf("LLMHub Version: %s, Commit: %s, BuiltAt: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init-db-from-env":
			os.Exit(runInitDBFromEnv(os.Args[2:]))
		case "migrate-local-to-db":
			os.Exit(runMigrateLocalToDB(os.Args[2:]))
		}
	}

	// Command-line flags to control the application's behavior.
	var login bool
	var codexLogin bool
	var codexDeviceLogin bool
	var claudeLogin bool
	var noBrowser bool
	var oauthCallbackPort int
	var antigravityLogin bool
	var kimiLogin bool
	var xaiLogin bool
	var projectID string
	var vertexImport string
	var vertexImportPrefix string
	var password string
	var tuiMode bool
	var standalone bool
	var localModel bool

	// Define command-line flags for different operation modes.
	flag.BoolVar(&login, "login", false, "Login Google Account")
	flag.BoolVar(&codexLogin, "codex-login", false, "Login to Codex using OAuth")
	flag.BoolVar(&codexDeviceLogin, "codex-device-login", false, "Login to Codex using device code flow")
	flag.BoolVar(&claudeLogin, "claude-login", false, "Login to Claude using OAuth")
	flag.BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically for OAuth")
	flag.IntVar(&oauthCallbackPort, "oauth-callback-port", 0, "Override OAuth callback port (defaults to provider-specific port)")
	flag.BoolVar(&antigravityLogin, "antigravity-login", false, "Login to Antigravity using OAuth")
	flag.BoolVar(&kimiLogin, "kimi-login", false, "Login to Kimi using OAuth")
	flag.BoolVar(&xaiLogin, "xai-login", false, "Login to xAI using OAuth")
	flag.StringVar(&projectID, "project_id", "", "Project ID (Gemini only, not required)")
	flag.StringVar(&vertexImport, "vertex-import", "", "Import Vertex service account key JSON file")
	flag.StringVar(&vertexImportPrefix, "vertex-import-prefix", "", "Prefix for Vertex model namespacing (use with -vertex-import)")
	flag.StringVar(&password, "password", "", "")
	flag.BoolVar(&tuiMode, "tui", false, "Start with terminal management UI")
	flag.BoolVar(&standalone, "standalone", false, "In TUI mode, start an embedded local server")
	flag.BoolVar(&localModel, "local-model", false, "Use embedded model catalog only, skip remote model fetching")

	flag.CommandLine.Usage = func() {
		out := flag.CommandLine.Output()
		_, _ = fmt.Fprintf(out, "Usage of %s\n", os.Args[0])
		flag.CommandLine.VisitAll(func(f *flag.Flag) {
			if f.Name == "password" {
				return
			}
			s := fmt.Sprintf("  -%s", f.Name)
			name, unquoteUsage := flag.UnquoteUsage(f)
			if name != "" {
				s += " " + name
			}
			if len(s) <= 4 {
				s += "	"
			} else {
				s += "\n    "
			}
			if unquoteUsage != "" {
				s += unquoteUsage
			}
			if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
				s += fmt.Sprintf(" (default %s)", f.DefValue)
			}
			_, _ = fmt.Fprint(out, s+"\n")
		})
	}

	// Parse the command-line flags.
	flag.Parse()

	// Core application variables.
	var err error
	var cfg *config.Config
	var (
		pgStoreInst           interface{ Close() error }
		pgStoreUsageRetention int
		configFilePath        = runtimeConfigLabel
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pgStore, loadedCfg, retention, err := loadRuntimeConfigFromPostgres(ctx)
	if err != nil {
		log.Errorf("failed to load runtime config: %v", err)
		return
	}
	defer pgStore.Close()
	pgStoreInst = pgStore
	_ = pgStoreInst
	cfg = loadedCfg
	pgStoreUsageRetention = retention
	if value, ok := lookupEnvTrimmed("LLMHUB_HOST", "llmhub_host"); ok {
		cfg.Host = value
	}
	if value, ok := lookupEnvTrimmed("LLMHUB_PORT", "llmhub_port"); ok {
		port, errParse := strconv.Atoi(value)
		if errParse != nil || port <= 0 || port > 65535 {
			log.Errorf("invalid LLMHUB_PORT: %q", value)
			return
		}
		cfg.Port = port
	}

	redisqueue.SetUsageStatisticsEnabled(cfg.UsageStatisticsEnabled)
	if pgStoreUsageRetention > 0 {
		cfg.RedisUsageQueueRetentionSeconds = pgStoreUsageRetention
	}
	redisqueue.SetRetentionSeconds(cfg.RedisUsageQueueRetentionSeconds)
	redisqueue.SetUsageStore(pgStore)
	runtimeStoragePolicy := runtimepolicy.RuntimeStorage{PostgresDurable: true}

	if err = logging.ConfigureLogOutputWithPolicy(cfg, runtimeStoragePolicy); err != nil {
		log.Errorf("failed to configure log output: %v", err)
		return
	}

	log.Infof("LLMHub Version: %s, Commit: %s, BuiltAt: %s", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
	if runtimeStoragePolicy.PostgresDurableMode() {
		log.Info("postgres durable runtime mode active: local file-backed application logs and request archives are disabled; use stdout/stderr instead")
	}

	// Set the log level based on the configuration.
	util.SetLogLevel(cfg)
	// Create login options to be used in authentication flows.
	options := &cmd.LoginOptions{
		NoBrowser:    noBrowser,
		CallbackPort: oauthCallbackPort,
	}

	// Register the shared Postgres token store once so all components use DB persistence.
	sdkAuth.RegisterTokenStore(pgStore)

	// Register built-in access providers before constructing services.
	configaccess.Register(&cfg.SDKConfig)

	// Handle different command modes based on the provided flags.

	if vertexImport != "" {
		// Handle Vertex service account import
		cmd.DoVertexImport(cfg, vertexImport, vertexImportPrefix)
	} else if login {
		// Handle Google/Gemini login
		cmd.DoLogin(cfg, projectID, options)
	} else if antigravityLogin {
		// Handle Antigravity login
		cmd.DoAntigravityLogin(cfg, options)
	} else if codexLogin {
		// Handle Codex login
		cmd.DoCodexLogin(cfg, options)
	} else if codexDeviceLogin {
		// Handle Codex device-code login
		cmd.DoCodexDeviceLogin(cfg, options)
	} else if claudeLogin {
		// Handle Claude login
		cmd.DoClaudeLogin(cfg, options)
	} else if kimiLogin {
		cmd.DoKimiLogin(cfg, options)
	} else if xaiLogin {
		cmd.DoXAILogin(cfg, options)
	} else {
		if localModel && (!tuiMode || standalone) {
			log.Info("Local model mode: using embedded model catalog, remote model updates disabled")
		}
		if tuiMode {
			if standalone {
				// Standalone mode: start an embedded local server and connect TUI client to it.
				misc.StartAntigravityVersionUpdater(context.Background())
				if !localModel {
					registry.StartModelsUpdater(context.Background())
				}
				hook := tui.NewLogHook(2000)
				hook.SetFormatter(&logging.LogFormatter{})
				log.AddHook(hook)

				origStdout := os.Stdout
				origStderr := os.Stderr
				origLogOutput := log.StandardLogger().Out
				log.SetOutput(io.Discard)

				devNull, errOpenDevNull := os.Open(os.DevNull)
				if errOpenDevNull == nil {
					os.Stdout = devNull
					os.Stderr = devNull
				}

				restoreIO := func() {
					os.Stdout = origStdout
					os.Stderr = origStderr
					log.SetOutput(origLogOutput)
					if devNull != nil {
						_ = devNull.Close()
					}
				}

				localMgmtPassword := fmt.Sprintf("tui-%d-%d", os.Getpid(), time.Now().UnixNano())
				if password == "" {
					password = localMgmtPassword
				}

				cancel, done := cmd.StartServiceBackgroundWithBuilder(cfg, configFilePath, password, func(builder *cliproxy.Builder) {
					builder.WithManagementConfigStore(pgStore).
						WithWatcherFactory(cliproxy.NewStorageWatcherFactory(pgStore)).
						WithRuntimeStoragePolicy(runtimeStoragePolicy)
					configureQuotaAlertRuntime(builder, pgStore)
				})

				client := tui.NewClient(cfg.Port, password)
				ready := false
				backoff := 100 * time.Millisecond
				for i := 0; i < 30; i++ {
					if _, errGetConfig := client.GetConfig(); errGetConfig == nil {
						ready = true
						break
					}
					time.Sleep(backoff)
					if backoff < time.Second {
						backoff = time.Duration(float64(backoff) * 1.5)
					}
				}

				if !ready {
					restoreIO()
					cancel()
					<-done
					fmt.Fprintf(os.Stderr, "TUI error: embedded server is not ready\n")
					return
				}

				if errRun := tui.Run(cfg.Port, password, hook, origStdout); errRun != nil {
					restoreIO()
					fmt.Fprintf(os.Stderr, "TUI error: %v\n", errRun)
				} else {
					restoreIO()
				}

				cancel()
				<-done
			} else {
				// Default TUI mode: pure management client.
				// The proxy server must already be running.
				if errRun := tui.Run(cfg.Port, password, nil, os.Stdout); errRun != nil {
					fmt.Fprintf(os.Stderr, "TUI error: %v\n", errRun)
				}
			}
		} else {
			// Start the main proxy service
			misc.StartAntigravityVersionUpdater(context.Background())
			if !localModel {
				registry.StartModelsUpdater(context.Background())
			}
			cmd.StartServiceWithBuilder(cfg, configFilePath, password, func(builder *cliproxy.Builder) {
				builder.WithManagementConfigStore(pgStore).
					WithWatcherFactory(cliproxy.NewStorageWatcherFactory(pgStore)).
					WithRuntimeStoragePolicy(runtimeStoragePolicy)
				configureQuotaAlertRuntime(builder, pgStore)
			})
		}
	}
}
