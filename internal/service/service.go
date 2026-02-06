// sentiric-user-service/internal/service/service.go
package service

import (
	"context"

	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
)

// UserService defines the business logic methods for the User Service.
type UserService interface {
	GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error)
	FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error)
	CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error)

	// SIP
	GetSipCredentials(ctx context.Context, req *userv1.GetSipCredentialsRequest) (*userv1.GetSipCredentialsResponse, error)
	CreateSipCredential(ctx context.Context, req *userv1.CreateSipCredentialRequest) (*userv1.CreateSipCredentialResponse, error)
	DeleteSipCredential(ctx context.Context, req *userv1.DeleteSipCredentialRequest) (*userv1.DeleteSipCredentialResponse, error)
}
