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
	"strings"
	"time"

	"github.com/sentiric/sentiric-user-service/internal/logger"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const serviceName = "user-service"

var log zerolog.Logger

type server struct {
	userv1.UnimplementedUserServiceServer
	db *sql.DB
}

func getLoggerWithTraceID(ctx context.Context, baseLogger zerolog.Logger) zerolog.Logger {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return baseLogger
	}
	traceIDValues := md.Get("x-trace-id")
	if len(traceIDValues) > 0 {
		return baseLogger.With().Str("trace_id", traceIDValues[0]).Logger()
	}
	return baseLogger
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

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	l := getLoggerWithTraceID(ctx, log).With().Str("method", "GetUser").Str("user_id", req.GetUserId()).Logger()
	l.Info().Msg("Kullanıcı ID ile istek alındı")
	user, err := s.fetchUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &userv1.GetUserResponse{User: user}, nil
}

func (s *server) FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error) {
	l := getLoggerWithTraceID(ctx, log).With().Str("method", "FindUserByContact").Str("contact_value", req.GetContactValue()).Logger()
	l.Info().Msg("İletişim bilgisi ile kullanıcı arama isteği alındı")

	query := `
		SELECT u.id, u.name, u.tenant_id, u.user_type, u.preferred_language_code
		FROM users u
		JOIN contacts c ON u.id = c.user_id
		WHERE c.contact_type = $1 AND c.contact_value = $2
	`
	row := s.db.QueryRowContext(ctx, query, req.GetContactType(), req.GetContactValue())
	var user userv1.User
	var name, langCode sql.NullString
	err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType, &langCode)
	if err != nil {
		if err == sql.ErrNoRows {
			l.Warn().Msg("İletişim bilgisiyle eşleşen kullanıcı bulunamadı")
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetContactValue())
		}
		l.Error().Err(err).Msg("Veritabanı sorgu hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası: %v", err)
	}
	if name.Valid {
		user.Name = &name.String
	}
	if langCode.Valid {
		user.PreferredLanguageCode = &langCode.String
	}
	contacts, err := s.fetchContactsForUser(ctx, user.Id)
	if err != nil {
		return nil, err
	}
	user.Contacts = contacts
	l.Info().Msg("Kullanıcı başarıyla bulundu")
	return &userv1.FindUserByContactResponse{User: &user}, nil
}

func (s *server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	l := getLoggerWithTraceID(ctx, log).With().Str("method", "CreateUser").Str("tenant_id", req.GetTenantId()).Logger()
	l.Info().Msg("Kullanıcı oluşturma isteği alındı")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		l.Error().Err(err).Msg("Veritabanı transaction başlatılamadı")
		return nil, status.Error(codes.Internal, "Veritabanı hatası")
	}
	defer tx.Rollback()
	userQuery := `INSERT INTO users (name, tenant_id, user_type, preferred_language_code) VALUES ($1, $2, $3, $4) RETURNING id`
	var newUserID string
	err = tx.QueryRowContext(ctx, userQuery, req.Name, req.TenantId, req.UserType, req.PreferredLanguageCode).Scan(&newUserID)
	if err != nil {
		l.Error().Err(err).Msg("Yeni kullanıcı kaydı başarısız")
		return nil, status.Errorf(codes.Internal, "Kullanıcı oluşturulamadı: %v", err)
	}
	contactQuery := `INSERT INTO contacts (user_id, contact_type, contact_value, is_primary) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, contactQuery, newUserID, req.InitialContact.GetContactType(), req.InitialContact.GetContactValue(), true)
	if err != nil {
		l.Error().Err(err).Msg("Yeni kullanıcının iletişim bilgisi kaydedilemedi")
		return nil, status.Errorf(codes.Internal, "İletişim bilgisi oluşturulamadı: %v", err)
	}
	if err := tx.Commit(); err != nil {
		l.Error().Err(err).Msg("Veritabanı transaction commit edilemedi")
		return nil, status.Error(codes.Internal, "Veritabanı hatası")
	}
	createdUser, err := s.fetchUserByID(ctx, newUserID)
	if err != nil {
		return nil, err
	}
	l.Info().Str("user_id", newUserID).Msg("Kullanıcı ve iletişim bilgisi başarıyla oluşturuldu")
	return &userv1.CreateUserResponse{User: createdUser}, nil
}

func (s *server) fetchUserByID(ctx context.Context, userID string) (*userv1.User, error) {
	l := getLoggerWithTraceID(ctx, log)
	query := "SELECT id, name, tenant_id, user_type, preferred_language_code FROM users WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, userID)
	var user userv1.User
	var name, langCode sql.NullString
	if err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType, &langCode); err != nil {
		if err == sql.ErrNoRows {
			l.Warn().Str("user_id", userID).Msg("Kullanıcı ID ile bulunamadı")
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", userID)
		}
		l.Error().Err(err).Str("user_id", userID).Msg("Kullanıcı sorgu hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}
	if name.Valid {
		user.Name = &name.String
	}
	if langCode.Valid {
		user.PreferredLanguageCode = &langCode.String
	}
	contacts, err := s.fetchContactsForUser(ctx, user.Id)
	if err != nil {
		return nil, err
	}
	user.Contacts = contacts
	return &user, nil
}

func (s *server) fetchContactsForUser(ctx context.Context, userID string) ([]*userv1.Contact, error) {
	query := `SELECT id, user_id, contact_type, contact_value, is_primary FROM contacts WHERE user_id = $1`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "İletişim bilgileri sorgulanamadı: %v", err)
	}
	defer rows.Close()
	var contacts []*userv1.Contact
	for rows.Next() {
		var c userv1.Contact
		if err := rows.Scan(&c.Id, &c.UserId, &c.ContactType, &c.ContactValue, &c.IsPrimary); err != nil {
			return nil, status.Errorf(codes.Internal, "İletişim bilgisi satırı okunamadı: %v", err)
		}
		contacts = append(contacts, &c)
	}
	return contacts, nil
}

func connectToDBWithRetry(url string, maxRetries int) *sql.DB {
	var db *sql.DB
	var err error
	finalURL := url

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", finalURL)
		if err == nil {
			db.SetConnMaxLifetime(time.Minute * 3)
			db.SetMaxIdleConns(2)
			db.SetMaxOpenConns(5)
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
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: caPool}
	return credentials.NewTLS(tlsConfig)
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
