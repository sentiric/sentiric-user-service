package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config struct holds all configuration for the application.
type Config struct {
	DatabaseURL  string
	GRPCPort     string
	CertPath     string
	KeyPath      string
	CaPath       string
	MaxDBRetries int
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	godotenv.Load()

	return &Config{
		DatabaseURL:  GetEnvOrFail("POSTGRES_URL"),
		GRPCPort:     GetEnv("USER_SERVICE_GRPC_PORT", "50053"),
		CertPath:     GetEnvOrFail("USER_SERVICE_CERT_PATH"),
		KeyPath:      GetEnvOrFail("USER_SERVICE_KEY_PATH"),
		CaPath:       GetEnvOrFail("GRPC_TLS_CA_PATH"),
		MaxDBRetries: 10,
	}, nil
}

// GetEnv retrieves an environment variable or returns a fallback.
func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// GetEnvOrFail retrieves an environment variable or fails fatally.
func GetEnvOrFail(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatal().Str("variable", key).Msg("Gerekli ortam değişkeni tanımlı değil")
	}
	return value
}
