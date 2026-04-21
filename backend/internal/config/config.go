package config

import "os"

type Config struct {
	AppEnv         string
	BackendHost    string
	BackendPort    string
	PostgresHost   string
	PostgresPort   string
	PostgresDB     string
	PostgresUser   string
	PostgresPass   string
	PostgresSSL    string
	MLDataRoot     string
	MLModelRoot    string
	MLResearchRoot string
}

func Load() Config {
	return Config{
		AppEnv:         getEnv("APP_ENV", "dev"),
		BackendHost:    getEnv("BACKEND_HOST", "0.0.0.0"),
		BackendPort:    getEnv("BACKEND_PORT", "8080"),
		PostgresHost:   getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:   getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:     getEnv("POSTGRES_DB", "invest"),
		PostgresUser:   getEnv("POSTGRES_USER", "invest"),
		PostgresPass:   getEnv("POSTGRES_PASSWORD", "invest"),
		PostgresSSL:    getEnv("POSTGRES_SSLMODE", "disable"),
		MLDataRoot:     getEnv("ML_DATA_ROOT", "../data"),
		MLModelRoot:    getEnv("ML_MODEL_ROOT", "../artifacts/models"),
		MLResearchRoot: getEnv("ML_RESEARCH_ROOT", "../artifacts/research"),
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
