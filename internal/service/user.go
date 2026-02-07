// sentiric-user-service/internal/service/user.go
package service

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"github.com/sentiric/sentiric-user-service/internal/config"
	"github.com/sentiric/sentiric-user-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userService struct {
	repo   repository.UserRepository
	config *config.Config
	log    zerolog.Logger
}

func NewUserService(repo repository.UserRepository, cfg *config.Config, log zerolog.Logger) UserService {
	return &userService{repo: repo, config: cfg, log: log}
}

// --- Business Logic ---

// GetUser: [FIXED Signature] Artık (Response, error) döndürüyor.
func (s *userService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := s.repo.FetchUserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetUserId())
		}
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}
	return &userv1.GetUserResponse{User: user}, nil
}

func (s *userService) FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error) {
	contactValue := normalizePhoneNumber(req.GetContactValue())

	user, err := s.repo.FetchUserByContact(ctx, req.GetContactType(), contactValue)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetContactValue())
		}
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}
	return &userv1.FindUserByContactResponse{User: user}, nil
}

func (s *userService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	normalizedValue := req.InitialContact.GetContactValue()
	if req.InitialContact.GetContactType() == "phone" {
		normalizedValue = normalizePhoneNumber(req.InitialContact.GetContactValue())
	}

	newUser := &userv1.User{
		Name:                  req.Name,
		TenantId:              req.TenantId,
		UserType:              req.UserType,
		PreferredLanguageCode: req.PreferredLanguageCode,
	}

	user, err := s.repo.CreateUser(ctx, newUser, req.InitialContact, normalizedValue)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, status.Errorf(codes.AlreadyExists, "Bu iletişim bilgisi zaten kayıtlı: %s", req.InitialContact.GetContactValue())
		}
		return nil, status.Errorf(codes.Internal, "Kullanıcı oluşturulamadı: %v", err)
	}

	return &userv1.CreateUserResponse{User: user}, nil
}

func (s *userService) GetSipCredentials(ctx context.Context, req *userv1.GetSipCredentialsRequest) (*userv1.GetSipCredentialsResponse, error) {
	s.log.Debug().
		Str("username", req.SipUsername).
		Str("requested_realm", req.Realm).
		Msg("🔑 Fetching SIP credentials")

	userID, tenantID, ha1Hash, err := s.repo.FetchSipCredentials(ctx, req.GetSipUsername())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "SIP kullanıcısı bulunamadı: %s", req.GetSipUsername())
		}
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	// [SECURITY]: Realm doğrulaması ve uyarı logu
	if req.Realm != "" && req.Realm != s.config.SipRealm {
		s.log.Warn().
			Str("expected", s.config.SipRealm).
			Str("received", req.Realm).
			Msg("⚠️ Realm mismatch detected during authentication!")
	}

	return &userv1.GetSipCredentialsResponse{
		UserId:   userID,
		TenantId: tenantID,
		Ha1Hash:  ha1Hash,
	}, nil
}

func (s *userService) CreateSipCredential(ctx context.Context, req *userv1.CreateSipCredentialRequest) (*userv1.CreateSipCredentialResponse, error) {
	_, err := s.repo.FetchUserByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "İlişkili kullanıcı bulunamadı")
		}
		return nil, status.Errorf(codes.Internal, "Kullanıcı sorgulanamadı")
	}

	realm := s.config.SipRealm
	h := md5.New()
	io.WriteString(h, fmt.Sprintf("%s:%s:%s", req.SipUsername, realm, req.Password))
	ha1Hash := fmt.Sprintf("%x", h.Sum(nil))

	s.log.Info().
		Str("username", req.SipUsername).
		Str("realm", realm).
		Msg("📝 Creating new SIP credential")

	err = s.repo.CreateSipCredential(ctx, req.UserId, req.SipUsername, ha1Hash)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, status.Errorf(codes.AlreadyExists, "Bu SIP kullanıcı adı zaten mevcut")
		}
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	return &userv1.CreateSipCredentialResponse{Success: true}, nil
}

func (s *userService) DeleteSipCredential(ctx context.Context, req *userv1.DeleteSipCredentialRequest) (*userv1.DeleteSipCredentialResponse, error) {
	err := s.repo.DeleteSipCredential(ctx, req.SipUsername)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Silinecek SIP kullanıcısı bulunamadı")
		}
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}
	return &userv1.DeleteSipCredentialResponse{Success: true}, nil
}

// --- Helpers ---

func normalizePhoneNumber(phone string) string {
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "0") {
		return "90" + phone[1:]
	}
	return phone
}
