// sentiric-user-service/internal/app/app.go
package app

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
	"github.com/sentiric/sentiric-user-service/internal/repository"
	"github.com/sentiric/sentiric-user-service/internal/repository/postgres"
	"github.com/sentiric/sentiric-user-service/internal/server"
	"github.com/sentiric/sentiric-user-service/internal/service"
)

type App struct {
	Cfg *config.Config
	Log zerolog.Logger
}

func NewApp(cfg *config.Config, log zerolog.Logger) *App {
	return &App{Cfg: cfg, Log: log}
}

func (a *App) Run() {
	// 1. Altyapı Bağlantısı
	db, err := database.Connect(a.Cfg.DatabaseURL, a.Cfg.MaxDBRetries, a.Log)
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

	// 2. DI: Repository -> Service -> Handler
	var userRepo repository.UserRepository = postgres.NewPostgresRepository(db, a.Log)
	userService := service.NewUserService(userRepo, a.Cfg, a.Log)

	// 3. Server Katmanı
	grpcServer := server.NewGrpcServer(userService, a.Cfg, a.Log)
	httpServer := a.startHttpServer(a.Cfg.HttpPort)

	// 4. Sunucuyu Başlat
	go func() {
		a.Log.Info().Str("port", a.Cfg.GRPCPort).Msg("gRPC sunucusu dinleniyor...")
		if err := server.Start(grpcServer, a.Cfg.GRPCPort); err != nil && err.Error() != "http: Server closed" {
			a.Log.Error().Err(err).Msg("gRPC sunucusu başlatılamadı")
		}
	}()

	// 5. Graceful Shutdown
	a.waitForShutdown(grpcServer, httpServer)
}

func (a *App) startHttpServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok"}`)
	})

	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		a.Log.Info().Str("port", port).Msg("HTTP sunucusu (health) dinleniyor")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Log.Fatal().Err(err).Msg("HTTP sunucusu başlatılamadı")
		}
	}()
	return srv
}

func (a *App) waitForShutdown(grpcSrv *server.GrpcServer, httpSrv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Log.Warn().Msg("Kapatma sinyali alındı, servisler durduruluyor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server.Stop(grpcSrv)
	a.Log.Info().Msg("gRPC sunucusu durduruldu.")

	if err := httpSrv.Shutdown(ctx); err != nil {
		a.Log.Error().Err(err).Msg("HTTP sunucusu düzgün kapatılamadı.")
	} else {
		a.Log.Info().Msg("HTTP sunucusu durduruldu.")
	}

	a.Log.Info().Msg("Servis başarıyla durduruldu.")
}
