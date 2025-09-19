// sentiric-user-service/cmd/user-service/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
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
		fmt.Fprintf(os.Stderr, "Kritik Hata: Konfigürasyon yüklenemedi: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}
	defer db.Close()

	// HTTP ve gRPC sunucularını oluştur
	grpcServer := server.NewGrpcServer(db, cfg.CertPath, cfg.KeyPath, cfg.CaPath, log, cfg)
	httpServer := startHttpServer(cfg.HttpPort, log)

	// gRPC sunucusunu bir goroutine'de başlat
	go func() {
		log.Info().Str("port", cfg.GRPCPort).Msg("gRPC sunucusu dinleniyor...")
		if err := server.Start(grpcServer, cfg.GRPCPort); err != nil && err.Error() != "http: Server closed" {
			log.Error().Err(err).Msg("gRPC sunucusu başlatılamadı")
		}
	}()

	// Graceful shutdown için sinyal dinleyicisi
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn().Msg("Kapatma sinyali alındı, servisler durduruluyor...")

	// Sunucuları zarifçe kapatmak için bir context oluştur
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// gRPC sunucusunu durdur
	server.Stop(grpcServer)
	log.Info().Msg("gRPC sunucusu durduruldu.")

	// HTTP sunucusunu durdur
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP sunucusu düzgün kapatılamadı.")
	} else {
		log.Info().Msg("HTTP sunucusu durduruldu.")
	}

	log.Info().Msg("Servis başarıyla durduruldu.")
}

func startHttpServer(port string, log zerolog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok"}`)
	})

	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Info().Str("port", port).Msg("HTTP sunucusu (health) dinleniyor")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP sunucusu başlatılamadı")
		}
	}()
	return srv
}