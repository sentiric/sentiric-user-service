package main

import (
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/database"
	"github.com/sentiric/sentiric-user-service/internal/logger"
	"github.com/sentiric/sentiric-user-service/internal/server"
)

// YENİ: ldflags ile doldurulacak değişkenler
var (
	ServiceVersion string
	GitCommit      string
	BuildDate      string
)

const serviceName = "user-service"

func main() {
	log := logger.New(serviceName)

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Konfigürasyon yüklenemedi")
	}

	// YENİ: Başlangıçta versiyon bilgisini logla
	log.Info().
		Str("version", ServiceVersion).
		Str("commit", GitCommit).
		Str("build_date", BuildDate).
		Str("profile", config.GetEnv("ENV", "production")).
		Msg("🚀 Sentiric User Service başlatılıyor...")

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
