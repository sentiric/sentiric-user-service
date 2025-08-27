package database

import (
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

// Connect connects to the database with a retry mechanism.
func Connect(url string, maxRetries int, log zerolog.Logger) (*sql.DB, error) {
	var db *sql.DB
	var err error

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatal().Err(err).Msg("PostgreSQL URL parse edilemedi")
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	finalURL := stdlib.RegisterConnConfig(config.ConnConfig)

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", finalURL)
		if err == nil {
			db.SetConnMaxLifetime(time.Minute * 3)
			db.SetMaxIdleConns(2)
			db.SetMaxOpenConns(5)
			if pingErr := db.Ping(); pingErr == nil {
				log.Info().Msg("Veritabanına bağlantı başarılı (Simple Protocol Mode).")
				return db, nil
			} else {
				err = pingErr
				db.Close()
			}
		}
		log.Warn().Err(err).Int("attempt", i+1).Int("max_attempts", maxRetries).Msg("Veritabanına bağlanılamadı, 5 saniye sonra tekrar denenecek...")
		time.Sleep(5 * time.Second)
	}

	log.Fatal().Err(err).Msgf("Veritabanına bağlanılamadı (%d deneme)", maxRetries)
	return nil, err
}
