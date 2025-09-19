// sentiric-user-service/cmd/user-service/main.go
package main

import (
	"fmt"
	"net/http" // YENİ
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
		return
	}

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
	
	// YENİ: HTTP Sunucusunu başlat
	go startHttpServer(cfg.HttpPort, log)

	if err := server.Start(cfg.GRPCPort, db, cfg.CertPath, cfg.KeyPath, cfg.CaPath, log); err != nil {
		log.Fatal().Err(err).Msg("Sunucu başlatılamadı")
	}
}

// YENİ: HTTP sunucusunu başlatan fonksiyon
func startHttpServer(port string, log zerolog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok"}`)
	})

	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("port", port).Msg("HTTP sunucusu (health) dinleniyor")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error().Err(err).Msg("HTTP sunucusu başlatılamadı")
	}
}