package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time" // Tekrar deneme mantığı için eklendi

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	_ "github.com/jackc/pgx/v5/stdlib"

	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
)

type server struct {
	userv1.UnimplementedUserServiceServer
	db *sql.DB
}

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	log.Printf("GetUser request received for user ID: %s", req.GetId())

	query := "SELECT id, name, email, tenant_id FROM users WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, req.GetId())

	var user userv1.User
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.TenantId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found for ID: %s", req.GetId())
			return nil, status.Errorf(codes.NotFound, "user with ID '%s' not found", req.GetId())
		}
		log.Printf("Database query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "database query failed: %v", err)
	}

	log.Printf("User found: %s", user.Name)
	return &userv1.GetUserResponse{User: &user}, nil
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	var db *sql.DB
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", dbURL)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("Successfully connected to the database")
				break
			}
		}
		if i == maxRetries-1 {
			log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in 5 seconds...", i+1, maxRetries, err)
		time.Sleep(5 * time.Second)
	}
	defer db.Close()

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
	// Bu satır, 'server' struct'ını ve metodlarını 'kullanılır' hale getirir.
	userv1.RegisterUserServiceServer(s, &server{db: db})
	reflection.Register(s)

	log.Printf("gRPC user-service listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
