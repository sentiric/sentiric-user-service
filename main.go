// main.go
package main

import (
	"context"
	"log"
	"net"

	// 1. Gerekli gRPC kütüphanelerini import et
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection" // gRPC sunucusunu test etmek için harika bir araç

	// 2. OTOMATİK ÜRETİLEN KODLARI IMPORT ETME
	// Bu yol, proto dosyasındaki `option go_package` direktifinden gelir.
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
)

// 3. Servisimiz için bir `server` struct'ı tanımla.
// Bu struct, `UserServiceServer` arayüzünü (interface) implemente edecek.
// `userv1.UnimplementedUserServiceServer`'ı embed etmek, geriye dönük uyumluluk sağlar.
// Gelecekte arayüze yeni metodlar eklenirse, servisimiz derlenmeye devam eder.
type server struct {
	userv1.UnimplementedUserServiceServer
}

// 4. Proto dosyasında tanımladığımız `GetUser` RPC'sini implemente et.
// Bu metodun imzası, otomatik üretilen kodlardan gelir.
func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	log.Printf("GetUser request received for user ID: %s", req.GetId())

	// Şimdilik sahte (dummy) bir kullanıcı verisi döndürelim.
	// Gerçek bir uygulamada burada veritabanına gidilir.
	dummyUser := &userv1.User{
		Id:    req.GetId(),
		Name:  "Sentiric User",
		Email: req.GetId() + "@sentiric.com",
	}

	return &userv1.GetUserResponse{User: dummyUser}, nil
}

func main() {
	// 5. Sunucunun dinleyeceği portu belirle
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 6. Yeni bir gRPC sunucusu oluştur
	s := grpc.NewServer()

	// 7. Oluşturduğumuz `server` struct'ını gRPC sunucusuna kaydet
	userv1.RegisterUserServiceServer(s, &server{})

	// 8. Sunucu yansımasını (reflection) kaydet. Bu, gRPC istemcilerinin
	// (grpcurl gibi) sunucunun hangi servisleri sunduğunu dinamik olarak keşfetmesini sağlar.
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())

	// 9. Sunucuyu başlat ve gelen istekleri dinle
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
