// sentiric-user-service/cmd/user-service/main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// --- YENİ IMPORT BLOKU BAŞLANGICI ---
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/database"
	"github.com/sentiric/sentiric-user-service/internal/logger"
	"github.com/sentiric/sentiric-user-service/internal/server"
	"github.com/rs/zerolog" // Bu satır doğrudan zerolog'u import etmese de,
	                        // logger paketini import ettiğimiz için gereklidir.
	// --- YENİ IMPORT BLOKU SONU ---
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
		// Loglama henüz hazır olmadığı için standart log kullanıyoruz.
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
		// Connect fonksiyonu zaten loglama yapıyor ve gerekirse programı sonlandırıyor.
		// Bu yüzden burada ek bir loga gerek yok, sadece çıkış yapabiliriz.
		os.Exit(1)
	}
	defer db.Close()
	
	go startHttpServer(cfg.HttpPort, log)

	// Graceful shutdown için sinyal dinleyicisi
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	grpcServer := server.NewGrpcServer(db, cfg.CertPath, cfg.KeyPath, cfg.CaPath, log)
	go func() {
		log.Info().Str("port", cfg.GRPCPort).Msg("gRPC sunucusu dinleniyor...")
		if err := server.Start(grpcServer, cfg.GRPCPort); err != nil {
			log.Error().Err(err).Msg("gRPC sunucusu başlatılamadı")
			stopChan <- syscall.SIGTERM // Başlatma hatası durumunda ana goroutine'i sonlandır
		}
	}()

	<-stopChan // Kapatma sinyali bekleniyor
	
	log.Warn().Msg("Kapatma sinyali alındı, servis durduruluyor...")
	server.Stop(grpcServer)
	log.Info().Msg("Servis başarıyla durduruldu.")
}

func startHttpServer(port string, log zerolog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok"}`)
	})

	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("port", port).Msg("HTTP sunucusu (health) dinleniyor")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error().Err(err).Msg("HTTP sunucusu başlatılamadı")
	}
}