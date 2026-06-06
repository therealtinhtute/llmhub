package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/store"
	"github.com/therealtinhtute/llmhub/internal/util"
)

const runtimeConfigLabel = "postgres://runtime-config"

func lookupEnvTrimmed(keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

func loadDotEnvForSetup(path string) error {
	envPath := strings.TrimSpace(path)
	if envPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		envPath = filepath.Join(wd, ".env")
	}
	if err := godotenv.Load(envPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func loadPostgresStoreFromEnv(ctx context.Context) (*store.PostgresStore, int, error) {
	dsn, ok := lookupEnvTrimmed("PGSTORE_DSN", "pgstore_dsn")
	if !ok {
		return nil, 0, fmt.Errorf("PGSTORE_DSN is required")
	}
	storeCfg := store.PostgresStoreConfig{
		DSN: dsn,
	}
	if value, ok := lookupEnvTrimmed("PGSTORE_SCHEMA", "pgstore_schema"); ok {
		storeCfg.Schema = value
	}
	if value, ok := lookupEnvTrimmed("PGSTORE_USAGE_RETENTION_SECONDS", "pgstore_usage_retention_seconds"); ok {
		retention, err := strconv.Atoi(value)
		if err != nil || retention < 0 {
			return nil, 0, fmt.Errorf("invalid PGSTORE_USAGE_RETENTION_SECONDS: %q", value)
		}
		pgStore, err := store.NewPostgresStore(ctx, storeCfg)
		return pgStore, retention, err
	}
	pgStore, err := store.NewPostgresStore(ctx, storeCfg)
	return pgStore, 0, err
}

func loadInitConfigBytesFromEnv() ([]byte, error) {
	if raw, ok := lookupEnvTrimmed("LLMHUB_INIT_CONFIG_B64"); ok {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("decode LLMHUB_INIT_CONFIG_B64: %w", err)
		}
		if _, err := config.ParseConfigBytes(decoded); err != nil {
			return nil, fmt.Errorf("parse init config from LLMHUB_INIT_CONFIG_B64: %w", err)
		}
		return decoded, nil
	}
	if raw, ok := lookupEnvTrimmed("LLMHUB_INIT_CONFIG_YAML"); ok {
		data := []byte(raw)
		if _, err := config.ParseConfigBytes(data); err != nil {
			return nil, fmt.Errorf("parse init config from LLMHUB_INIT_CONFIG_YAML: %w", err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("missing LLMHUB_INIT_CONFIG_YAML or LLMHUB_INIT_CONFIG_B64")
}

func runInitDBFromEnv(args []string) int {
	flags := flag.NewFlagSet("init-db-from-env", flag.ContinueOnError)
	envFile := flags.String("env-file", "", "Optional env file to load before reading init variables")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := loadDotEnvForSetup(*envFile); err != nil {
		log.Errorf("failed to load env file: %v", err)
		return 1
	}
	configBytes, err := loadInitConfigBytesFromEnv()
	if err != nil {
		log.Errorf("failed to load init config from env: %v", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pgStore, _, err := loadPostgresStoreFromEnv(ctx)
	if err != nil {
		log.Errorf("failed to initialize postgres store: %v", err)
		return 1
	}
	defer pgStore.Close()
	seeded, version, err := pgStore.InitializeConfig(ctx, configBytes)
	if err != nil {
		log.Errorf("failed to initialize postgres config: %v", err)
		return 1
	}
	if !seeded {
		log.Infof("postgres config already initialized; current version remains %d", version)
		return 0
	}
	log.Infof("postgres config initialized from env at version %d", version)
	return 0
}

func runMigrateLocalToDB(args []string) int {
	flags := flag.NewFlagSet("migrate-local-to-db", flag.ContinueOnError)
	envFile := flags.String("env-file", "", "Optional env file to load before opening Postgres")
	configPath := flags.String("config", "", "Legacy local config.yaml path to import")
	authDir := flags.String("auth-dir", "", "Legacy auth directory to import")
	overwrite := flags.Bool("overwrite-auth", false, "Overwrite existing auth IDs already present in Postgres")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*configPath) == "" {
		log.Error("missing -config for migrate-local-to-db")
		return 1
	}
	if err := loadDotEnvForSetup(*envFile); err != nil {
		log.Errorf("failed to load env file: %v", err)
		return 1
	}
	configBytes, err := os.ReadFile(strings.TrimSpace(*configPath))
	if err != nil {
		log.Errorf("failed to read local config: %v", err)
		return 1
	}
	cfg, err := config.ParseConfigBytes(configBytes)
	if err != nil {
		log.Errorf("failed to parse local config: %v", err)
		return 1
	}
	resolvedAuthDir := strings.TrimSpace(*authDir)
	if resolvedAuthDir == "" && cfg != nil {
		resolvedAuthDir = cfg.AuthDir
	}
	if resolvedAuthDir != "" {
		resolvedAuthDir, err = util.ResolveAuthDir(resolvedAuthDir)
		if err != nil {
			log.Errorf("failed to resolve auth dir: %v", err)
			return 1
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pgStore, _, err := loadPostgresStoreFromEnv(ctx)
	if err != nil {
		log.Errorf("failed to initialize postgres store: %v", err)
		return 1
	}
	defer pgStore.Close()
	seeded, version, err := pgStore.InitializeConfig(ctx, configBytes)
	if err != nil {
		log.Errorf("failed to initialize postgres config: %v", err)
		return 1
	}
	imported, skipped, err := pgStore.ImportAuthFromDirectory(ctx, resolvedAuthDir, *overwrite)
	if err != nil {
		log.Errorf("failed to import auth records: %v", err)
		return 1
	}
	if seeded {
		log.Infof("migrated config to postgres at version %d", version)
	} else {
		log.Infof("postgres config already existed; left unchanged at version %d", version)
	}
	log.Infof("imported %d auth records, skipped %d", imported, skipped)
	return 0
}

func legacyRuntimeModeError() error {
	legacyKeys := []string{
		"HOME_JWT",
		"home_jwt",
		"GITSTORE_GIT_URL",
		"gitstore_git_url",
		"OBJECTSTORE_ENDPOINT",
		"objectstore_endpoint",
	}
	for _, key := range legacyKeys {
		if _, ok := lookupEnvTrimmed(key); ok {
			return fmt.Errorf("%s is no longer supported; runtime is Postgres-only", key)
		}
	}
	return nil
}

func loadRuntimeConfigFromPostgres(ctx context.Context) (*store.PostgresStore, *config.Config, int, error) {
	if err := legacyRuntimeModeError(); err != nil {
		return nil, nil, 0, err
	}
	pgStore, retention, err := loadPostgresStoreFromEnv(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	snapshot, err := pgStore.LoadConfig(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = pgStore.Close()
			return nil, nil, 0, fmt.Errorf("postgres runtime config is empty; run `llmhub init-db-from-env` or `llmhub migrate-local-to-db` first")
		}
		_ = pgStore.Close()
		return nil, nil, 0, err
	}
	cfg, err := config.ParseConfigBytes(snapshot.Content)
	if err != nil {
		_ = pgStore.Close()
		return nil, nil, 0, err
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.Home.Enabled {
		_ = pgStore.Close()
		return nil, nil, 0, fmt.Errorf("home runtime mode is no longer supported")
	}
	return pgStore, cfg, retention, nil
}
