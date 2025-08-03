package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	userv1.UnimplementedUserServiceServer
	db *sql.DB
}

func main() {
	log.Println("Sentiric User Service başlatılıyor...")

	// .env yükle
	_ = godotenv.Load()

	// Ortam değişkenlerini al ve doğrula
	dbURL := getEnvOrFail("POSTGRES_URL")
	port := getEnv("USER_SERVICE_GRPC_PORT", "50053")
	certPath := getEnvOrFail("USER_SERVICE_CERT_PATH")
	keyPath := getEnvOrFail("USER_SERVICE_KEY_PATH")
	caPath := getEnvOrFail("GRPC_TLS_CA_PATH")

	db := connectToDBWithRetry(dbURL, 10)
	defer db.Close()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("gRPC portu dinlenemedi: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(loadServerTLS(certPath, keyPath, caPath)))
	userv1.RegisterUserServiceServer(grpcServer, &server{db: db})
	reflection.Register(grpcServer)

	log.Printf("gRPC sunucusu %s portunda dinleniyor...", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("gRPC sunucusu başlatılamadı: %v", err)
	}
}

// =============================
// === Database ve TLS Setup ===
// =============================

func connectToDBWithRetry(url string, maxRetries int) *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", url)
		if err == nil && db.Ping() == nil {
			log.Println("Veritabanına bağlantı başarılı.")
			return db
		}
		log.Printf("Veritabanına bağlanılamadı (deneme %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(5 * time.Second)
	}
	log.Fatalf("Veritabanına bağlanılamadı (%d deneme): %v", maxRetries, err)
	return nil
}

func loadServerTLS(certPath, keyPath, caPath string) credentials.TransportCredentials {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatalf("Sunucu sertifikası yüklenemedi: %v", err)
	}
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		log.Fatalf("CA sertifikası okunamadı: %v", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		log.Fatal("CA sertifikası havuza eklenemedi.")
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	}
	return credentials.NewTLS(tlsConfig)
}

// =====================
// === gRPC Servisler ===
// =====================

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	log.Printf("GetUser: %s", req.GetId())
	query := "SELECT id, name, tenant_id, user_type FROM users WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, req.GetId())

	var user userv1.User
	var name sql.NullString
	err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetId())
	}
	if err != nil {
		log.Printf("DB hatası: %v", err)
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası: %v", err)
	}
	if name.Valid {
		user.Name = name.String
	}
	return &userv1.GetUserResponse{User: &user}, nil
}

func (s *server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	log.Printf("CreateUser: %s", req.GetId())
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
		log.Printf("Kullanıcı oluşturulamadı: %v", err)
		return nil, status.Errorf(codes.Internal, "Kullanıcı oluşturulamadı: %v", err)
	}
	return &userv1.CreateUserResponse{User: user}, nil
}

// ============================
// === Yardımcı Fonksiyonlar ===
// ============================

func getEnv(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvOrFail(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Ortam değişkeni tanımlı değil: %s", key)
	}
	return val
}
