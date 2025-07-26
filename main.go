package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	// PGX sürücüsünü standart database/sql arayüzü ile kullanmak için import ediyoruz.
	_ "github.com/jackc/pgx/v5/stdlib"

	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
)

// server struct'ına artık bir veritabanı bağlantısı (DB pool) ekliyoruz.
type server struct {
	userv1.UnimplementedUserServiceServer
	db *sql.DB
}

// GetUser RPC'si artık mock veri yerine veritabanından okuma yapacak.
func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	log.Printf("GetUser request received for user ID: %s", req.GetId())

	// Veritabanından kullanıcıyı sorgula
	query := "SELECT id, name, email, tenant_id FROM users WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, req.GetId())

	var user userv1.User
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.TenantId)

	if err != nil {
		// Eğer sorgu sonuç döndürmezse, bu "Not Found" durumudur.
		if err == sql.ErrNoRows {
			log.Printf("User not found for ID: %s", req.GetId())
			// gRPC standartlarına uygun olarak 'Not Found' hatası döndürüyoruz.
			return nil, status.Errorf(codes.NotFound, "user with ID '%s' not found", req.GetId())
		}
		// Diğer veritabanı hataları için Internal Server Error döndür.
		log.Printf("Database query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "database query failed: %v", err)
	}

	log.Printf("User found: %s", user.Name)
	return &userv1.GetUserResponse{User: &user}, nil
}

func main() {
	// Veritabanı bağlantı bilgisini ortam değişkeninden al.
	// Docker Compose'da bu değişkeni .env dosyasından ayarlayacağız.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Veritabanına bağlan
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Bağlantıyı test et
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// gRPC sunucusunu başlat
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50053"
	}
	listenAddr := fmt.Sprintf(":%s", port)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// server struct'ını oluştururken veritabanı bağlantısını da içine koyuyoruz.
	userv1.RegisterUserServiceServer(s, &server{db: db})
	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
