// sentiric-user-service/internal/config/config.go
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
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
	// YENİ ALAN: Log seviyesini de config'den okuyacağız.
	LogLevel     string
	Env          string
}

func Load() (*Config, error) {
	godotenv.Load()

	maxRetries, err := strconv.Atoi(GetEnv("MAX_DB_RETRIES", "10"))
	if err != nil {
		log.Warn().Str("value", GetEnv("MAX_DB_RETRIES", "10")).Msg("Geçersiz MAX_DB_RETRIES değeri, varsayılan (10) kullanılıyor.")
		maxRetries = 10
	}

	return &Config{
		DatabaseURL:  GetEnvOrFail("POSTGRES_URL"),
		GRPCPort:     GetEnv("USER_SERVICE_GRPC_PORT", "12011"),
		HttpPort:     GetEnv("USER_SERVICE_HTTP_PORT", "12010"),
		CertPath:     GetEnvOrFail("USER_SERVICE_CERT_PATH"),
		KeyPath:      GetEnvOrFail("USER_SERVICE_KEY_PATH"),
		CaPath:       GetEnvOrFail("GRPC_TLS_CA_PATH"),
		SipRealm:     GetEnvOrFail("SIP_SIGNALING_REALM"),
		MaxDBRetries: maxRetries,
		// YENİ ALANLARI DOLDURUYORUZ
		LogLevel:     GetEnv("LOG_LEVEL", "info"),
		Env:          GetEnv("ENV", "production"),
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
		log.Fatal().Str("variable", key).Msg("Gerekli ortam değişkeni tanımlı değil")
	}
	return value
}

// GetEnv metodunu Config struct'ından kaldırıyoruz çünkü artık doğrudan kullanılıyor.
// func (c *Config) GetEnv(key, fallback string) string {
// 	return GetEnv(key, fallback)
// }