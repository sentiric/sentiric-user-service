// sentiric-user-service/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	GRPCPort     string
	HttpPort     string
	CertPath     string
	KeyPath      string
	CaPath       string
	SipRealm     string
	MaxDBRetries int
	// SUTS v4.0 Alanları
	LogLevel       string
	LogFormat      string
	Env            string
	NodeHostname   string // YENİ
	ServiceVersion string
}

func Load() (*Config, error) {
	// Sessiz yükleme
	_ = godotenv.Load()

	maxRetries, err := strconv.Atoi(GetEnv("MAX_DB_RETRIES", "10"))
	if err != nil {
		maxRetries = 10
	}

	return &Config{
		DatabaseURL:  GetEnvOrFail("POSTGRES_URL"),
		GRPCPort:     GetEnv("USER_SERVICE_GRPC_PORT", "12011"),
		HttpPort:     GetEnv("USER_SERVICE_HTTP_PORT", "12010"),
		CertPath:     GetEnvOrFail("USER_SERVICE_CERT_PATH"),
		KeyPath:      GetEnvOrFail("USER_SERVICE_KEY_PATH"),
		CaPath:       GetEnvOrFail("GRPC_TLS_CA_PATH"),
		SipRealm:     GetEnvOrFail("SIP_SIGNALING_SERVICE_REALM"),
		MaxDBRetries: maxRetries,
		// SUTS Config
		LogLevel:       GetEnv("LOG_LEVEL", "info"),
		LogFormat:      GetEnv("LOG_FORMAT", "json"), // Prod default: json
		Env:            GetEnv("ENV", "production"),
		NodeHostname:   GetEnv("NODE_HOSTNAME", "localhost"),
		ServiceVersion: GetEnv("SERVICE_VERSION", "1.0.0"),
	}, nil
}

func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func GetEnvOrFail(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		fmt.Fprintf(os.Stderr, "Kritik Hata: Gerekli ortam değişkeni tanımlı değil: %s\n", key)
		os.Exit(1)
	}
	return value
}
