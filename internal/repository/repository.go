// sentiric-user-service/internal/repository/repository.go
package repository

import (
	"context"

	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
)

// Hata sabitleri burada tanımlanabilir, ancak şimdilik Service katmanında tutuluyor.

// UserRepository, User Service domain'i için gerekli tüm CRUD ve sorgulama işlemlerini soyutlar.
type UserRepository interface {
	// User CRUD
	FetchUserByID(ctx context.Context, userID string) (*userv1.User, error)
	FetchUserByContact(ctx context.Context, contactType, contactValue string) (*userv1.User, error)
	CreateUser(ctx context.Context, user *userv1.User, initialContact *userv1.CreateUserRequest_InitialContact, normalizedContactValue string) (*userv1.User, error)

	// Sip Credentials
	FetchSipCredentials(ctx context.Context, sipUsername string) (userID, tenantID, ha1Hash string, err error)
	CreateSipCredential(ctx context.Context, userID, sipUsername, ha1Hash string) error
	DeleteSipCredential(ctx context.Context, sipUsername string) error

	// Helper
	FetchContactsForUser(ctx context.Context, userID string) ([]*userv1.Contact, error)
}
