package main

import (
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/database"
	"github.com/sentiric/sentiric-user-service/internal/logger"
	"github.com/sentiric/sentiric-user-service/internal/server"
)

var (
	ServiceVersion string
	GitCommit      string
	BuildDate      string
)

const serviceName = "user-service"

func main() {
	cfg, err := config.Load()
	if err != nil {
		// config.Load zaten loglama yapıyor.
		return
	}

	// Bu çağrı artık doğru imzayla eşleşiyor: New(serviceName, env)
	log := logger.New(serviceName, cfg.GetEnv("ENV", "production"))

	log.Info().
		Str("version", ServiceVersion).
		Str("commit", GitCommit).
		Str("build_date", BuildDate).
		Str("profile", cfg.GetEnv("ENV", "production")).
		Msg("🚀 Sentiric User Service başlatılıyor...")

	db, err := database.Connect(cfg.DatabaseURL, cfg.MaxDBRetries, log)
	if err != nil {
		return
	}
	defer db.Close()

	if err := server.Start(cfg.GRPCPort, db, cfg.CertPath, cfg.KeyPath, cfg.CaPath, log); err != nil {
		log.Fatal().Err(err).Msg("Sunucu başlatılamadı")
	}
}
