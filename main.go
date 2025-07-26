package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// DEĞİŞİKLİK: Artık yerel 'gen' klasörü yerine Go modülü olarak indirilen
	// merkezi kontrat reposundan import ediyoruz.
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
)

// Servisimiz için bir `server` struct'ı tanımla.
type server struct {
	userv1.UnimplementedUserServiceServer
}

// Proto dosyasında tanımladığımız `GetUser` RPC'sini implemente et.
func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	log.Printf("GetUser request received for user ID: %s", req.GetId())

	// Şimdilik sahte (dummy) bir kullanıcı veritabanımız var.
	// Gerçek bir uygulamada burada veritabanına gidilir.
	mockUsers := map[string]*userv1.User{
		"1001":         {Id: "1001", Name: "Alice", Email: "alice@sentiric.com"},
		"1002":         {Id: "1002", Name: "Bob", Email: "bob@sentiric.com"},
		"902124548590": {Id: "902124548590", Name: "Main IVR Account", Email: "ivr@sentiric.com"},
	}

	if user, ok := mockUsers[req.GetId()]; ok {
		log.Printf("User found: %s", user.Name)
		return &userv1.GetUserResponse{User: user}, nil
	}

	log.Printf("User not found for ID: %s", req.GetId())
	// Kullanıcı bulunamazsa, 'user' alanı 'nil' olan boş bir yanıt döndürürüz.
	// İstemci bu durumu kontrol etmelidir.
	return &userv1.GetUserResponse{User: nil}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	userv1.RegisterUserServiceServer(s, &server{})
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
