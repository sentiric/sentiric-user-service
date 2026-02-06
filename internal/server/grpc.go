// sentiric-user-service/internal/server/grpc.go
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/rs/zerolog"
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

// GrpcServer'ı bir Go'nun yerel gRPC sunucusunu kapsayacak şekilde tip alias yapalım.
type GrpcServer = grpc.Server

// server struct'ı, Service Layer'ı (İş Mantığı) tutar.
type server struct {
	userv1.UnimplementedUserServiceServer
	svc service.UserService
	log zerolog.Logger
}

// NewGrpcServer, Service Layer'ı alarak Handler'ı oluşturur.
func NewGrpcServer(svc service.UserService, cfg *config.Config, log zerolog.Logger) *GrpcServer {
	creds, err := loadServerTLS(cfg.CertPath, cfg.KeyPath, cfg.CaPath, log)
	if err != nil {
		log.Fatal().Err(err).Msg("TLS kimlik bilgileri yüklenemedi")
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	userv1.RegisterUserServiceServer(grpcServer, &server{svc: svc, log: log})
	reflection.Register(grpcServer)
	return grpcServer
}

// Start, verilen gRPC sunucusunu belirtilen portta dinlemeye başlar.
func Start(grpcServer *GrpcServer, port string) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("gRPC portu dinlenemedi: %w", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC sunucusu başlatılamadı: %w", err)
	}
	return nil
}

// Stop, gRPC sunucusunu zarif bir şekilde (gracefully) durdurur.
func Stop(grpcServer *GrpcServer) {
	grpcServer.GracefulStop()
}

// --- gRPC Metot Implementasyonları (Handler) ---

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "GetUser").Logger()
	l.Info().Str("user_id", req.GetUserId()).Msg("İstek Service Layer'a devrediliyor")
	return s.svc.GetUser(ctx, req)
}

func (s *server) FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "FindUserByContact").Logger()
	l.Info().Str("contact_value", req.GetContactValue()).Msg("İstek Service Layer'a devrediliyor")
	return s.svc.FindUserByContact(ctx, req)
}

func (s *server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "CreateUser").Logger()
	l.Info().Str("tenant_id", req.GetTenantId()).Msg("İstek Service Layer'a devrediliyor")
	return s.svc.CreateUser(ctx, req)
}

func (s *server) GetSipCredentials(ctx context.Context, req *userv1.GetSipCredentialsRequest) (*userv1.GetSipCredentialsResponse, error) {
	return s.svc.GetSipCredentials(ctx, req)
}

func (s *server) CreateSipCredential(ctx context.Context, req *userv1.CreateSipCredentialRequest) (*userv1.CreateSipCredentialResponse, error) {
	return s.svc.CreateSipCredential(ctx, req)
}

func (s *server) DeleteSipCredential(ctx context.Context, req *userv1.DeleteSipCredentialRequest) (*userv1.DeleteSipCredentialResponse, error) {
	return s.svc.DeleteSipCredential(ctx, req)
}

// --- Helper Functions ---

func loadServerTLS(certPath, keyPath, caPath string, log zerolog.Logger) (credentials.TransportCredentials, error) {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("sunucu sertifikası yüklenemedi: %w", err)
	}
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("CA sertifikası okunamadı: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("CA sertifikası havuza eklenemedi")
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	}
	return credentials.NewTLS(tlsConfig), nil
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
