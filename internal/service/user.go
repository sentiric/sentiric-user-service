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
	"github.com/sentiric/sentiric-user-service/internal/logger"
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

func (s *userService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	l := logger.ContextLogger(ctx, s.log)

	user, err := s.repo.FetchUserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.Warn().
				Str("event", logger.EventUserLookupFailed).
				Dict("attributes", zerolog.Dict().
					Str("reason", "not_found").
					Str("user_id", req.GetUserId())).
				Msg("Kullanıcı bulunamadı")
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetUserId())
		}
		l.Error().
			Str("event", "DB_ERROR").
			Err(err).
			Msg("Veritabanı hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	l.Debug().
		Str("event", logger.EventUserLookup).
		Str("tenant_id", user.TenantId).
		Dict("attributes", zerolog.Dict().
			Str("user_id", user.Id)).
		Msg("Kullanıcı ID ile getirildi")

	return &userv1.GetUserResponse{User: user}, nil
}

func (s *userService) FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error) {
	l := logger.ContextLogger(ctx, s.log)
	contactValue := normalizePhoneNumber(req.GetContactValue())

	user, err := s.repo.FetchUserByContact(ctx, req.GetContactType(), contactValue)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.Info().
				Str("event", logger.EventUserLookupFailed).
				Dict("attributes", zerolog.Dict().
					Str("contact_type", req.GetContactType()).
					Str("contact_value", contactValue)).
				Msg("İletişim bilgisine ait kullanıcı yok")

			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetContactValue())
		}
		l.Error().
			Str("event", "DB_ERROR").
			Err(err).
			Msg("Veritabanı hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	// [SUTS]: Kullanıcı bulundu
	l.Info().
		Str("event", logger.EventUserLookup).
		Str("tenant_id", user.TenantId).
		Dict("attributes", zerolog.Dict().
			Str("user_id", user.Id).
			Str("contact_type", req.GetContactType())).
		Msg("Kullanıcı iletişim bilgisiyle bulundu")

	return &userv1.FindUserByContactResponse{User: user}, nil
}

func (s *userService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	l := logger.ContextLogger(ctx, s.log)

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
			l.Warn().
				Str("event", logger.EventUserConflict).
				Dict("attributes", zerolog.Dict().
					Str("contact_value", req.InitialContact.GetContactValue())).
				Msg("Kullanıcı/Kontak zaten mevcut")
			return nil, status.Errorf(codes.AlreadyExists, "Bu iletişim bilgisi zaten kayıtlı: %s", req.InitialContact.GetContactValue())
		}
		l.Error().
			Str("event", "USER_CREATION_FAIL").
			Err(err).
			Msg("Kullanıcı oluşturma hatası")
		return nil, status.Errorf(codes.Internal, "Kullanıcı oluşturulamadı: %v", err)
	}

	// [SUTS]: AUDIT LOG - Explicit Tenant Override
	l.Info().
		Str("event", logger.EventUserCreated).
		Str("tenant_id", user.TenantId).
		Dict("attributes", zerolog.Dict().
			Str("user_id", user.Id).
			Str("user_type", user.UserType)).
		Msg("Yeni kullanıcı başarıyla oluşturuldu")

	return &userv1.CreateUserResponse{User: user}, nil
}

func (s *userService) GetSipCredentials(ctx context.Context, req *userv1.GetSipCredentialsRequest) (*userv1.GetSipCredentialsResponse, error) {
	l := logger.ContextLogger(ctx, s.log)

	l.Debug().
		Str("event", logger.EventSipAuthAttempt).
		Dict("attributes", zerolog.Dict().
			Str("username", req.SipUsername).
			Str("requested_realm", req.Realm)).
		Msg("SIP Kimlik Bilgileri İsteniyor")

	userID, tenantID, ha1Hash, err := s.repo.FetchSipCredentials(ctx, req.GetSipUsername())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.Warn().
				Str("event", logger.EventSipAuthFailure).
				Dict("attributes", zerolog.Dict().
					Str("reason", "user_not_found").
					Str("username", req.GetSipUsername())).
				Msg("SIP Auth Başarısız: Kullanıcı yok")

			return nil, status.Errorf(codes.NotFound, "SIP kullanıcısı bulunamadı: %s", req.GetSipUsername())
		}
		l.Error().
			Str("event", "DB_ERROR").
			Err(err).
			Msg("Veritabanı hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	// Realm Check
	if req.Realm != "" && req.Realm != s.config.SipRealm {
		l.Warn().
			Str("event", logger.EventSipAuthFailure).
			Dict("attributes", zerolog.Dict().
				Str("reason", "realm_mismatch").
				Str("expected", s.config.SipRealm).
				Str("received", req.Realm)).
			Msg("SIP Auth Uyarısı: Realm uyuşmazlığı")
	} else {
		l.Info().
			Str("event", logger.EventSipAuthSuccess).
			Str("tenant_id", tenantID).
			Dict("attributes", zerolog.Dict().
				Str("user_id", userID)).
			Msg("SIP Kimlik Bilgileri Sağlandı")
	}

	return &userv1.GetSipCredentialsResponse{
		UserId:   userID,
		TenantId: tenantID,
		Ha1Hash:  ha1Hash,
	}, nil
}

func (s *userService) CreateSipCredential(ctx context.Context, req *userv1.CreateSipCredentialRequest) (*userv1.CreateSipCredentialResponse, error) {
	l := logger.ContextLogger(ctx, s.log)

	// Önce kullanıcının varlığını ve tenant_id'sini doğrula
	user, err := s.repo.FetchUserByID(ctx, req.UserId)
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

	err = s.repo.CreateSipCredential(ctx, req.UserId, req.SipUsername, ha1Hash)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			l.Warn().
				Str("event", "SIP_CRED_CONFLICT").
				Dict("attributes", zerolog.Dict().
					Str("username", req.SipUsername)).
				Msg("SIP kullanıcı adı çakışması")
			return nil, status.Errorf(codes.AlreadyExists, "Bu SIP kullanıcı adı zaten mevcut")
		}
		l.Error().Err(err).Msg("Veritabanı hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	l.Info().
		Str("event", logger.EventSipCredCreated).
		Str("tenant_id", user.TenantId).
		Dict("attributes", zerolog.Dict().
			Str("user_id", req.UserId).
			Str("sip_username", req.SipUsername).
			Str("realm", realm)).
		Msg("Yeni SIP kimliği oluşturuldu")

	return &userv1.CreateSipCredentialResponse{Success: true}, nil
}

func (s *userService) DeleteSipCredential(ctx context.Context, req *userv1.DeleteSipCredentialRequest) (*userv1.DeleteSipCredentialResponse, error) {
	l := logger.ContextLogger(ctx, s.log)

	err := s.repo.DeleteSipCredential(ctx, req.SipUsername)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Silinecek SIP kullanıcısı bulunamadı")
		}
		l.Error().Err(err).Msg("Veritabanı hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	l.Info().
		Str("event", "SIP_CRED_DELETED").
		Dict("attributes", zerolog.Dict().
			Str("sip_username", req.SipUsername)).
		Msg("SIP kimliği silindi")

	return &userv1.DeleteSipCredentialResponse{Success: true}, nil
}

func normalizePhoneNumber(phone string) string {
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "0") {
		return "90" + phone[1:]
	}
	return phone
}
