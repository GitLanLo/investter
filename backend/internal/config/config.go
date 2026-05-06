package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv                     string
	BackendHost                string
	BackendPort                string
	PostgresHost               string
	PostgresPort               string
	PostgresDB                 string
	PostgresUser               string
	PostgresPass               string
	PostgresSSL                string
	MLDataRoot                 string
	MLModelRoot                string
	MLResearchRoot             string
	MLUniverseConfigPath       string
	OutcomeSchedulerEnabled    bool
	OutcomeSchedulerInterval   time.Duration
	OutcomeSchedulerLimit      int
	OutcomeSchedulerRunOnStart bool
	TinkoffInvestToken         string
	TinkoffInvestTarget        string
	TinkoffCACertFile          string

	WatchlistRefreshSchedulerEnabled bool
	WatchlistRefreshInterval         time.Duration
	WatchlistRefreshLimit            int
	WatchlistSignalSchedulerEnabled  bool
	WatchlistSignalInterval          time.Duration
}

func Load() Config {
	return Config{
		AppEnv:                     getEnv("APP_ENV", "dev"),
		BackendHost:                getEnv("BACKEND_HOST", "0.0.0.0"),
		BackendPort:                getEnv("BACKEND_PORT", "8080"),
		PostgresHost:               getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:               getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:                 getEnv("POSTGRES_DB", "invest"),
		PostgresUser:               getEnv("POSTGRES_USER", "invest"),
		PostgresPass:               getEnv("POSTGRES_PASSWORD", "invest"),
		PostgresSSL:                getEnv("POSTGRES_SSLMODE", "disable"),
		MLDataRoot:                 getEnv("ML_DATA_ROOT", "../data"),
		MLModelRoot:                getEnv("ML_MODEL_ROOT", "../artifacts/models"),
		MLResearchRoot:             getEnv("ML_RESEARCH_ROOT", "../artifacts/research"),
		MLUniverseConfigPath:       getEnv("ML_UNIVERSE_CONFIG", "../configs/mvp_universe_v1.json"),
		OutcomeSchedulerEnabled:    getEnvBool("OUTCOME_SCHEDULER_ENABLED", false),
		OutcomeSchedulerInterval:   getEnvDuration("OUTCOME_SCHEDULER_INTERVAL", 15*time.Minute),
		OutcomeSchedulerLimit:      getEnvInt("OUTCOME_SCHEDULER_LIMIT", 1000),
		OutcomeSchedulerRunOnStart: getEnvBool("OUTCOME_SCHEDULER_RUN_ON_START", false),
		TinkoffInvestToken:         getEnv("TINKOFF_INVEST_TOKEN", ""),
		TinkoffInvestTarget:        getEnv("TINKOFF_INVEST_TARGET", "sandbox"),
		TinkoffCACertFile:          getEnv("TINKOFF_CA_CERT_FILE", ""),

		WatchlistRefreshSchedulerEnabled: getEnvBool("WATCHLIST_REFRESH_SCHEDULER_ENABLED", false),
		WatchlistRefreshInterval:         getEnvDuration("WATCHLIST_REFRESH_INTERVAL", 30*time.Minute),
		WatchlistRefreshLimit:            getEnvInt("WATCHLIST_REFRESH_LIMIT", 50),
		WatchlistSignalSchedulerEnabled:  getEnvBool("WATCHLIST_SIGNAL_SCHEDULER_ENABLED", false),
		WatchlistSignalInterval:          getEnvDuration("WATCHLIST_SIGNAL_INTERVAL", 30*time.Minute),
	}
}

func (c Config) HTTPAddress() string {
	return c.BackendHost + ":" + c.BackendPort
}

func (c Config) PostgresDSN() string {
	return "postgres://" + c.PostgresUser + ":" + c.PostgresPass + "@" +
		c.PostgresHost + ":" + c.PostgresPort + "/" + c.PostgresDB +
		"?sslmode=" + c.PostgresSSL
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
