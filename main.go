package main

import (
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/database"
	"github.com/sentiric/sentiric-user-service/internal/logger"
	"github.com/sentiric/sentiric-user-service/internal/server"
)

const serviceName = "user-service"

func main() {
	log := logger.New(serviceName)
	log.Info().Msg("Sentiric User Service başlatılıyor...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Konfigürasyon yüklenemedi")
	}

	db, err := database.Connect(cfg.DatabaseURL, cfg.MaxDBRetries, log)
	if err != nil {
		// Connect function already logs fatally, but we exit just in case.
		return
	}
	defer db.Close()

	if err := server.Start(cfg.GRPCPort, db, cfg.CertPath, cfg.KeyPath, cfg.CaPath, log); err != nil {
		log.Fatal().Err(err).Msg("Sunucu başlatılamadı")
	}
}
