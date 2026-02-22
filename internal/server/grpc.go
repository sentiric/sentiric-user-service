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
	"github.com/sentiric/sentiric-user-service/internal/logger"
	"github.com/sentiric/sentiric-user-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

type GrpcServer = grpc.Server

type server struct {
	userv1.UnimplementedUserServiceServer
	svc service.UserService
	log zerolog.Logger
}

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

func Stop(grpcServer *GrpcServer) {
	grpcServer.GracefulStop()
}

// --- Handler Implementations ---

func (s *server) propagateTrace(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	l := logger.ContextLogger(ctx, s.log)
	l.Info().
		Str("event", logger.EventGrpcRequest).
		Dict("attributes", zerolog.Dict().
			Str("method", "GetUser").
			Str("user_id", req.GetUserId())).
		Msg("gRPC İstek Alındı")

	ctx = s.propagateTrace(ctx)
	return s.svc.GetUser(ctx, req)
}

func (s *server) FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error) {
	l := logger.ContextLogger(ctx, s.log)
	l.Info().
		Str("event", logger.EventGrpcRequest).
		Dict("attributes", zerolog.Dict().
			Str("method", "FindUserByContact").
			Str("contact_type", req.GetContactType()).
			Str("contact_value", req.GetContactValue())).
		Msg("gRPC İstek Alındı")

	ctx = s.propagateTrace(ctx)
	return s.svc.FindUserByContact(ctx, req)
}

func (s *server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	l := logger.ContextLogger(ctx, s.log)
	l.Info().
		Str("event", logger.EventGrpcRequest).
		Dict("attributes", zerolog.Dict().
			Str("method", "CreateUser").
			Str("tenant_id", req.GetTenantId())).
		Msg("gRPC İstek Alındı")

	ctx = s.propagateTrace(ctx)
	return s.svc.CreateUser(ctx, req)
}

func (s *server) GetSipCredentials(ctx context.Context, req *userv1.GetSipCredentialsRequest) (*userv1.GetSipCredentialsResponse, error) {
	l := logger.ContextLogger(ctx, s.log)
	// Password/Auth isteklerini INFO seviyesinde basarken dikkatli olunmalı.
	// Kullanıcı adı güvenlidir.
	l.Info().
		Str("event", logger.EventGrpcRequest).
		Dict("attributes", zerolog.Dict().
			Str("method", "GetSipCredentials").
			Str("username", req.GetSipUsername())).
		Msg("SIP Auth İsteği Alındı")

	ctx = s.propagateTrace(ctx)
	return s.svc.GetSipCredentials(ctx, req)
}

func (s *server) CreateSipCredential(ctx context.Context, req *userv1.CreateSipCredentialRequest) (*userv1.CreateSipCredentialResponse, error) {
	ctx = s.propagateTrace(ctx)
	return s.svc.CreateSipCredential(ctx, req)
}

func (s *server) DeleteSipCredential(ctx context.Context, req *userv1.DeleteSipCredentialRequest) (*userv1.DeleteSipCredentialResponse, error) {
	ctx = s.propagateTrace(ctx)
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
