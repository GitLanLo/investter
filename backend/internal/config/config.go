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
	OutcomeSchedulerEnabled    bool
	OutcomeSchedulerInterval   time.Duration
	OutcomeSchedulerLimit      int
	OutcomeSchedulerRunOnStart bool
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
		OutcomeSchedulerEnabled:    getEnvBool("OUTCOME_SCHEDULER_ENABLED", false),
		OutcomeSchedulerInterval:   getEnvDuration("OUTCOME_SCHEDULER_INTERVAL", 15*time.Minute),
		OutcomeSchedulerLimit:      getEnvInt("OUTCOME_SCHEDULER_LIMIT", 1000),
		OutcomeSchedulerRunOnStart: getEnvBool("OUTCOME_SCHEDULER_RUN_ON_START", false),
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
