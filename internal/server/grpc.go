// File: sentiric-user-service/internal/server/grpc.go
package server

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/rs/zerolog"
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	userv1.UnimplementedUserServiceServer
	db  *sql.DB
	log zerolog.Logger
}

func Start(port string, db *sql.DB, certPath, keyPath, caPath string, log zerolog.Logger) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("gRPC portu dinlenemedi: %w", err)
	}

	creds, err := loadServerTLS(certPath, keyPath, caPath, log)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	userv1.RegisterUserServiceServer(grpcServer, &server{db: db, log: log})
	reflection.Register(grpcServer)

	log.Info().Str("port", port).Msg("gRPC sunucusu dinleniyor...")
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC sunucusu başlatılamadı: %w", err)
	}
	return nil
}

func normalizePhoneNumber(phone string) string {
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "0") {
		return "90" + phone[1:]
	}
	return phone
}

// (GetUser, FindUserByContact, CreateUser, fetchUserByID, fetchContactsForUser fonksiyonları aynı kalacak...)

func (s *server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "GetUser").Str("user_id", req.GetUserId()).Logger()
	l.Info().Msg("Kullanıcı ID ile istek alındı")

	user, err := s.fetchUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &userv1.GetUserResponse{User: user}, nil
}

func (s *server) FindUserByContact(ctx context.Context, req *userv1.FindUserByContactRequest) (*userv1.FindUserByContactResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "FindUserByContact").Str("contact_value", req.GetContactValue()).Logger()
	l.Info().Msg("İletişim bilgisi ile kullanıcı arama isteği alındı")

	normalizedValue := req.GetContactValue()
	if req.GetContactType() == "phone" {
		normalizedValue = normalizePhoneNumber(req.GetContactValue())
		l.Info().Str("original", req.GetContactValue()).Str("normalized", normalizedValue).Msg("Telefon numarası sorgu için normalize edildi.")
	}

	query := `
		SELECT u.id, u.name, u.tenant_id, u.user_type, u.preferred_language_code
		FROM users u
		JOIN contacts c ON u.id = c.user_id
		WHERE c.contact_type = $1 AND c.contact_value = $2
	`
	row := s.db.QueryRowContext(ctx, query, req.GetContactType(), normalizedValue)
	var user userv1.User
	var name, langCode sql.NullString
	err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType, &langCode)
	if err != nil {
		if err == sql.ErrNoRows {
			l.Warn().Msg("İletişim bilgisiyle eşleşen kullanıcı bulunamadı")
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", req.GetContactValue())
		}
		l.Error().Err(err).Msg("Veritabanı sorgu hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası: %v", err)
	}
	if name.Valid {
		user.Name = &name.String
	}
	if langCode.Valid {
		user.PreferredLanguageCode = &langCode.String
	}
	contacts, err := s.fetchContactsForUser(ctx, user.Id)
	if err != nil {
		return nil, err
	}
	user.Contacts = contacts
	l.Info().Msg("Kullanıcı başarıyla bulundu")
	return &userv1.FindUserByContactResponse{User: &user}, nil
}

func (s *server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "CreateUser").Str("tenant_id", req.GetTenantId()).Logger()
	l.Info().Msg("Kullanıcı oluşturma isteği alındı")

	normalizedValue := req.InitialContact.GetContactValue()
	if req.InitialContact.GetContactType() == "phone" {
		normalizedValue = normalizePhoneNumber(req.InitialContact.GetContactValue())
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		l.Error().Err(err).Msg("Veritabanı transaction başlatılamadı")
		return nil, status.Error(codes.Internal, "Veritabanı hatası")
	}
	defer tx.Rollback()
	userQuery := `INSERT INTO users (name, tenant_id, user_type, preferred_language_code) VALUES ($1, $2, $3, $4) RETURNING id`
	var newUserID string
	err = tx.QueryRowContext(ctx, userQuery, req.Name, req.TenantId, req.UserType, req.PreferredLanguageCode).Scan(&newUserID)
	if err != nil {
		l.Error().Err(err).Msg("Yeni kullanıcı kaydı başarısız")
		return nil, status.Errorf(codes.Internal, "Kullanıcı oluşturulamadı: %v", err)
	}
	contactQuery := `INSERT INTO contacts (user_id, contact_type, contact_value, is_primary) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, contactQuery, newUserID, req.InitialContact.GetContactType(), normalizedValue, true)
	if err != nil {
		l.Error().Err(err).Msg("Yeni kullanıcının iletişim bilgisi kaydedilemedi")
		return nil, status.Errorf(codes.Internal, "İletişim bilgisi oluşturulamadı: %v", err)
	}
	if err := tx.Commit(); err != nil {
		l.Error().Err(err).Msg("Veritabanı transaction commit edilemedi")
		return nil, status.Error(codes.Internal, "Veritabanı hatası")
	}

	createdUser, err := s.fetchUserByID(ctx, newUserID)
	if err != nil {
		return nil, err
	}
	l.Info().Str("user_id", newUserID).Msg("Kullanıcı ve iletişim bilgisi başarıyla oluşturuldu")
	return &userv1.CreateUserResponse{User: createdUser}, nil
}

// --- YENİ SIP YÖNETİM FONKSİYONLARI ---

func (s *server) CreateSipCredential(ctx context.Context, req *userv1.CreateSipCredentialRequest) (*userv1.CreateSipCredentialResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "CreateSipCredential").Str("user_id", req.UserId).Str("sip_username", req.SipUsername).Logger()
	l.Info().Msg("SIP kimlik bilgisi oluşturma isteği alındı")

	// 1. Kullanıcının tenant_id'sini ve realm'ı bul (realm şimdilik statik)
	var tenantId string
	err := s.db.QueryRowContext(ctx, "SELECT tenant_id FROM users WHERE id = $1", req.UserId).Scan(&tenantId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "İlişkili kullanıcı bulunamadı: %s", req.UserId)
		}
		return nil, status.Errorf(codes.Internal, "Kullanıcı sorgulanamadı: %v", err)
	}

	// Bu sabit bir değer, gelecekte tenant bazlı olabilir.
	realm := "sentiric_demo"

	// 2. HA1 hash'ini hesapla: MD5(username:realm:password)
	h := md5.New()
	io.WriteString(h, fmt.Sprintf("%s:%s:%s", req.SipUsername, realm, req.Password))
	ha1Hash := fmt.Sprintf("%x", h.Sum(nil))

	// YENİ GEÇİCİ LOG
	l.Info().Str("username", req.SipUsername).Str("realm", realm).Str("password", req.Password).Str("generated_ha1_hash", ha1Hash).Msg("HA1 HASH HESAPLANDI")

	// 3. Veritabanına ekle
	query := `INSERT INTO sip_credentials (user_id, sip_username, ha1_hash) VALUES ($1, $2, $3)`
	_, err = s.db.ExecContext(ctx, query, req.UserId, req.SipUsername, ha1Hash)
	if err != nil {
		// PostgreSQL'in unique constraint hatasını kontrol et
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, status.Errorf(codes.AlreadyExists, "Bu SIP kullanıcı adı zaten mevcut: %s", req.SipUsername)
		}
		l.Error().Err(err).Msg("SIP kimlik bilgisi veritabanına yazılamadı")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	l.Info().Msg("SIP kimlik bilgisi başarıyla oluşturuldu.")
	return &userv1.CreateSipCredentialResponse{Success: true}, nil
}

func (s *server) DeleteSipCredential(ctx context.Context, req *userv1.DeleteSipCredentialRequest) (*userv1.DeleteSipCredentialResponse, error) {
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "DeleteSipCredential").Str("sip_username", req.SipUsername).Logger()
	l.Info().Msg("SIP kimlik bilgisi silme isteği alındı")

	query := `DELETE FROM sip_credentials WHERE sip_username = $1`
	result, err := s.db.ExecContext(ctx, query, req.SipUsername)
	if err != nil {
		l.Error().Err(err).Msg("SIP kimlik bilgisi silinemedi")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "Silinecek SIP kullanıcısı bulunamadı: %s", req.SipUsername)
	}

	l.Info().Int64("rows_affected", rowsAffected).Msg("SIP kimlik bilgisi başarıyla silindi.")
	return &userv1.DeleteSipCredentialResponse{Success: true}, nil
}

// (GetSipCredentials, fetchUserByID, fetchContactsForUser, loadServerTLS, getLoggerWithTraceID aynı kalacak...)
func (s *server) fetchUserByID(ctx context.Context, userID string) (*userv1.User, error) {
	l := getLoggerWithTraceID(ctx, s.log)
	query := "SELECT id, name, tenant_id, user_type, preferred_language_code FROM users WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, userID)
	var user userv1.User
	var name, langCode sql.NullString
	if err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType, &langCode); err != nil {
		if err == sql.ErrNoRows {
			l.Warn().Str("user_id", userID).Msg("Kullanıcı ID ile bulunamadı")
			return nil, status.Errorf(codes.NotFound, "Kullanıcı bulunamadı: %s", userID)
		}
		l.Error().Err(err).Str("user_id", userID).Msg("Kullanıcı sorgu hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}
	if name.Valid {
		user.Name = &name.String
	}
	if langCode.Valid {
		user.PreferredLanguageCode = &langCode.String
	}
	contacts, err := s.fetchContactsForUser(ctx, user.Id)
	if err != nil {
		return nil, err
	}
	user.Contacts = contacts
	return &user, nil
}

func (s *server) fetchContactsForUser(ctx context.Context, userID string) ([]*userv1.Contact, error) {
	query := `SELECT id, user_id, contact_type, contact_value, is_primary FROM contacts WHERE user_id = $1`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "İletişim bilgileri sorgulanamadı: %v", err)
	}
	defer rows.Close()
	var contacts []*userv1.Contact
	for rows.Next() {
		var c userv1.Contact
		if err := rows.Scan(&c.Id, &c.UserId, &c.ContactType, &c.ContactValue, &c.IsPrimary); err != nil {
			return nil, status.Errorf(codes.Internal, "İletişim bilgisi satırı okunamadı: %v", err)
		}
		contacts = append(contacts, &c)
	}
	return contacts, nil
}

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

func getLoggerWithTraceID(ctx context.Context, baseLogger zerolog.Logger) zerolog.Logger {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return baseLogger
	}
	traceIDValues := md.Get("x-trace-id")
	if len(traceIDValues) > 0 {
		return baseLogger.With().Str("trace_id", traceIDValues[0]).Logger()
	}
	return baseLogger
}

// Sadece değişen ve hata veren fonksiyonun nihai halini ekliyorum:
func (s *server) GetSipCredentials(ctx context.Context, req *userv1.GetSipCredentialsRequest) (*userv1.GetSipCredentialsResponse, error) {
	// Artık bu satır hata vermeyecektir çünkü go.mod v1.8.7'yi kullanıyor.
	l := getLoggerWithTraceID(ctx, s.log).With().Str("method", "GetSipCredentials").Str("sip_username", req.GetSipUsername()).Str("realm", req.GetRealm()).Logger()
	l.Info().Msg("SIP kimlik bilgisi isteği alındı")

	query := `
        SELECT sc.user_id, u.tenant_id, sc.ha1_hash
        FROM sip_credentials sc
        JOIN users u ON sc.user_id = u.id
        WHERE sc.sip_username = $1
    `
	row := s.db.QueryRowContext(ctx, query, req.GetSipUsername())

	var res userv1.GetSipCredentialsResponse
	err := row.Scan(&res.UserId, &res.TenantId, &res.Ha1Hash)

	if err != nil {
		if err == sql.ErrNoRows {
			l.Warn().Msg("SIP kullanıcısı bulunamadı")
			return nil, status.Errorf(codes.NotFound, "SIP kullanıcısı bulunamadı: %s", req.GetSipUsername())
		}
		l.Error().Err(err).Msg("Veritabanı sorgu hatası")
		return nil, status.Errorf(codes.Internal, "Veritabanı hatası")
	}

	l.Info().Str("ha1_hash_retrieved", res.Ha1Hash).Msg("SIP kimlik bilgileri başarıyla bulundu")
	return &res, nil
}
