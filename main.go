package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"

	// Projemizin içine kopyaladığımız üretilmiş gRPC kodunu import ediyoruz
	userv1 "github.com/sentiric/sentiric-user-service/gen/user/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// --- Veri Yapıları ve Mock Veritabanı ---

type User struct {
	ID       string
	TenantID string
	// Gelecekte şifre/secret gibi alanlar eklenebilir
}

// DİKKAT: Bu mock veri sadece ilk geliştirme ve entegrasyon fazı içindir.
// TODO: Bu yapıyı PostgreSQL veritabanı ile değiştir.
//
//	Detaylar için projenin ana dizinindeki TASKS.md dosyasına bakınız (Görev ID: user-task-001).
var mockUsers = map[string]User{
	"1001":  {ID: "usr-1234", TenantID: "tnt-abcd"},
	"1002":  {ID: "usr-5678", TenantID: "tnt-abcd"},
	"alice": {ID: "usr-9101", TenantID: "tnt-efgh"},
}

// --- gRPC Sunucu Implementasyonu ---

// userv1.UserServiceServer arayüzünü implemente eden struct
type server struct {
	userv1.UnimplementedUserServiceServer // İleriye dönük uyumluluk için
	logger                                *zap.Logger
}

// AuthenticateUser RPC'sini implemente eden fonksiyon
func (s *server) AuthenticateUser(ctx context.Context, req *userv1.AuthenticateUserRequest) (*userv1.AuthenticateUserResponse, error) {
	// SIP URI'sinden kullanıcı adını/numarasını ayıklayalım. Örn: "sip:1001@domain.com" -> "1001"
	re := regexp.MustCompile(`sip:([^@;]+)`)
	matches := re.FindStringSubmatch(req.GetFromUri())

	if len(matches) < 2 {
		s.logger.Warn("Geçersiz From URI formatı", zap.String("uri", req.GetFromUri()))
		return &userv1.AuthenticateUserResponse{Status: userv1.AuthenticateUserResponse_STATUS_FAILED}, nil
	}
	username := matches[1]

	s.logger.Info("AuthenticateUser isteği alındı", zap.String("username", username))

	// Mock veritabanımızda kullanıcıyı ara
	if user, ok := mockUsers[username]; ok {
		s.logger.Info("Kullanıcı bulundu", zap.String("userID", user.ID), zap.String("tenantID", user.TenantID))
		return &userv1.AuthenticateUserResponse{
			Status:   userv1.AuthenticateUserResponse_STATUS_OK,
			UserId:   user.ID,
			TenantId: user.TenantID,
		}, nil
	}

	s.logger.Warn("Kullanıcı bulunamadı", zap.String("username", username))
	return &userv1.AuthenticateUserResponse{Status: userv1.AuthenticateUserResponse_STATUS_NOT_FOUND}, nil
}

// --- Ana Fonksiyon ---

func main() {
	// Üretim ortamına uygun, yapılandırılmış JSON loglama
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger oluşturulamadı: %v", err)
	}
	defer logger.Sync() // Uygulama kapanırken buffer'daki logları yaz

	// Ortam değişkeninden portu al, yoksa varsayılanı kullan
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50053"
	}
	listenAddr := fmt.Sprintf(":%s", port)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal("TCP dinleme başlatılamadı", zap.String("address", listenAddr), zap.Error(err))
	}

	s := grpc.NewServer()
	userv1.RegisterUserServiceServer(s, &server{logger: logger})

	logger.Info("gRPC sunucusu dinlemede", zap.String("address", listenAddr))
	if err := s.Serve(lis); err != nil {
		logger.Fatal("Sunucu başlatılamadı", zap.Error(err))
	}
}
