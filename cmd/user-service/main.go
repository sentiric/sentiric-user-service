// sentiric-user-service/cmd/user-service/main.go
package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/sentiric/sentiric-user-service/internal/app"
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/logger"
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

	log := logger.New(
		serviceName,
		cfg.ServiceVersion,
		cfg.Env,
		cfg.NodeHostname,
		cfg.LogLevel,
		cfg.LogFormat,
	)

	log.Info().
		Str("event", logger.EventSystemStartup).
		Dict("attributes", zerolog.Dict().
			Str("commit", GitCommit).
			Str("build_date", BuildDate).
			Str("profile", cfg.Env)).
		Msg("🚀 Sentiric User Service başlatılıyor (SUTS v4.0)...")

	application := app.NewApp(cfg, log)
	application.Run()
}
