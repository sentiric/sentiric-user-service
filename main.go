package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/sentiric/sentiric-user-service/internal/logger" // YENİ

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog" // YENİ
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Sabitler ve Global Değişkenler
const serviceName = "user-service"

var log zerolog.Logger

type server struct {
	userv1.UnimplementedUserServiceServer
	db *sql.DB
}

func main() {
	godotenv.Load()
	log = logger.New(serviceName)

	log.Info().Msg("Sentiric User Service başlatılıyor...")

	dbURL := getEnvOrFail("POSTGRES_URL")
	port := getEnv("USER_SERVICE_GRPC_PORT", "50053")
	certPath := getEnvOrFail("USER_SERVICE_CERT_PATH")
	keyPath := getEnvOrFail("USER_SERVICE_KEY_PATH")
	caPath := getEnvOrFail("GRPC_TLS_CA_PATH")

	db := connectToDBWithRetry(dbURL, 10)
	defer db.Close()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal().Err(err).Msg("gRPC portu dinlenemedi")
	}

	grpcServer := grpc.NewServer(grpc.Creds(loadServerTLS(certPath, keyPath, caPath)))
	userv1.RegisterUserServiceServer(grpcServer, &server{db: db})
	reflection.Register(grpcServer)

	log.Info().Str("port", port).Msg("gRPC sunucusu dinleniyor...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().Err(err).Msg("gRPC sunucusu başlatılamadı")
	}
}

// ... (Database ve TLS fonksiyonları aynı, sadece loglama çağrıları güncellendi) ...

func connectToDBWithRetry(url string, maxRetries int) *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", url)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				log.Info().Msg("Veritabanına bağlantı başarılı.")
				return db
			} else {
				err = pingErr
			}
		}
		log.Warn().Err(err).Int("attempt", i+1).Int("max_attempts", maxRetries).Msg("Veritabanına bağlanılamadı, 5 saniye sonra tekrar denenecek...")
		time.Sleep(5 * time.Second)
	}
	log.Fatal().Err(err).Msgf("Veritabanına bağlanılamadı (%d deneme)", maxRetries)
	return nil
}

func loadServerTLS(certPath, keyPath, caPath string) credentials.TransportCredentials {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Sunucu sertifikası yüklenemedi")
	}
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		log.Fatal().Err(err).Msg("CA sertifikası okunamadı")
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		log.Fatal().Msg("CA sertifikası havuza eklenemedi.")
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	}
	return credentials.NewTLS(tlsConfig)
}

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	l := log.With().Str("method", "GetUser").Str("user_id", req.GetId()).Logger()
	l.Info().Msg("İstek alındı")

	query := "SELECT id, name, tenant_id, user_type FROM users WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, req.GetId())

	var user userv1.User
	var name sql.NullString
	err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType)
	if err != nil {
		if err == sql.ErrNoRows {
			l.Warn().Msg("Kullanıcı bulunamadı")
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetId())
		}
		l.Error().Err(err).Msg("Veritabanı sorgu hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası: %v", err)
	}
	if name.Valid {
		user.Name = name.String
	}
	l.Info().Msg("Kullanıcı başarıyla bulundu")
	return &userv1.GetUserResponse{User: &user}, nil
}

func (s *server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	l := log.With().Str("method", "CreateUser").Str("user_id", req.GetId()).Str("tenant_id", req.GetTenantId()).Logger()
	l.Info().Msg("İstek alındı")

	user := &userv1.User{
		Id:       req.GetId(),
		Name:     req.GetName(),
		TenantId: req.GetTenantId(),
		UserType: req.GetUserType(),
	}
	var sqlName sql.NullString
	if user.Name != "" {
		sqlName = sql.NullString{String: user.Name, Valid: true}
	}
	query := `
		INSERT INTO users (id, name, tenant_id, user_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			tenant_id = EXCLUDED.tenant_id,
			user_type = EXCLUDED.user_type
		RETURNING id`
	err := s.db.QueryRowContext(ctx, query, user.Id, sqlName, user.TenantId, user.UserType).Scan(&user.Id)
	if err != nil {
		l.Error().Err(err).Msg("Kullanıcı oluşturulamadı/güncellenemedi")
		return nil, status.Errorf(codes.Internal, "Kullanıcı oluşturulamadı: %v", err)
	}
	l.Info().Msg("Kullanıcı başarıyla oluşturuldu/güncellendi")
	return &userv1.CreateUserResponse{User: user}, nil
}

func getEnv(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvOrFail(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatal().Str("variable", key).Msg("Gerekli ortam değişkeni tanımlı değil")
	}
	return val
}
