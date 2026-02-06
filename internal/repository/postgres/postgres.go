// sentiric-user-service/internal/repository/postgres/postgres.go
package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/rs/zerolog"
	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"github.com/sentiric/sentiric-user-service/internal/repository"
)

// PostgresRepository, tüm veritabanı işlemlerini yürüten yapıdır.
type PostgresRepository struct {
	db  *sql.DB
	log zerolog.Logger
}

// NewPostgresRepository, Repository'yi başlatır.
func NewPostgresRepository(db *sql.DB, log zerolog.Logger) repository.UserRepository {
	return &PostgresRepository{db: db, log: log}
}

// --- User CRUD ---

func (r *PostgresRepository) FetchUserByID(ctx context.Context, userID string) (*userv1.User, error) {
	query := "SELECT id, name, tenant_id, user_type, preferred_language_code FROM users WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, userID)
	var user userv1.User
	var name, langCode sql.NullString
	if err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType, &langCode); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		r.log.Error().Err(err).Str("user_id", userID).Msg("Veritabanı sorgu hatası")
		return nil, repository.ErrDatabase
	}
	if name.Valid {
		user.Name = &name.String
	}
	if langCode.Valid {
		user.PreferredLanguageCode = &langCode.String
	}

	contacts, err := r.FetchContactsForUser(ctx, user.Id)
	if err != nil {
		return nil, err
	}
	user.Contacts = contacts

	return &user, nil
}

func (r *PostgresRepository) FetchUserByContact(ctx context.Context, contactType, contactValue string) (*userv1.User, error) {
	query := `
		SELECT u.id, u.name, u.tenant_id, u.user_type, u.preferred_language_code
		FROM users u
		JOIN contacts c ON u.id = c.user_id
		WHERE c.contact_type = $1 AND c.contact_value = $2
	`
	row := r.db.QueryRowContext(ctx, query, contactType, contactValue)
	var user userv1.User
	var name, langCode sql.NullString
	err := row.Scan(&user.Id, &name, &user.TenantId, &user.UserType, &langCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		r.log.Error().Err(err).Msg("Veritabanı sorgu hatası")
		return nil, repository.ErrDatabase
	}
	if name.Valid {
		user.Name = &name.String
	}
	if langCode.Valid {
		user.PreferredLanguageCode = &langCode.String
	}

	contacts, err := r.FetchContactsForUser(ctx, user.Id)
	if err != nil {
		return nil, err
	}
	user.Contacts = contacts

	return &user, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *userv1.User, initialContact *userv1.CreateUserRequest_InitialContact, normalizedContactValue string) (*userv1.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, repository.ErrDatabase
	}
	defer tx.Rollback()

	userQuery := `INSERT INTO users (name, tenant_id, user_type, preferred_language_code) VALUES ($1, $2, $3, $4) RETURNING id`
	var newUserID string
	err = tx.QueryRowContext(ctx, userQuery, user.Name, user.TenantId, user.UserType, user.PreferredLanguageCode).Scan(&newUserID)
	if err != nil {
		return nil, repository.ErrDatabase
	}

	contactQuery := `INSERT INTO contacts (user_id, contact_type, contact_value, is_primary) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, contactQuery, newUserID, initialContact.GetContactType(), normalizedContactValue, true)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, repository.ErrConflict
		}
		return nil, repository.ErrDatabase
	}

	if err := tx.Commit(); err != nil {
		return nil, repository.ErrDatabase
	}

	return r.FetchUserByID(ctx, newUserID)
}

// --- Sip Credentials ---

func (r *PostgresRepository) FetchSipCredentials(ctx context.Context, sipUsername string) (userID, tenantID, ha1Hash string, err error) {
	query := `SELECT sc.user_id, u.tenant_id, sc.ha1_hash FROM sip_credentials sc JOIN users u ON sc.user_id = u.id WHERE sc.sip_username = $1`
	row := r.db.QueryRowContext(ctx, query, sipUsername)

	var resUserID, resTenantID, resHA1Hash string
	err = row.Scan(&resUserID, &resTenantID, &resHA1Hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", repository.ErrNotFound
		}
		r.log.Error().Err(err).Msg("SIP kimlik sorgu hatası")
		return "", "", "", repository.ErrDatabase
	}
	return resUserID, resTenantID, resHA1Hash, nil
}

func (r *PostgresRepository) CreateSipCredential(ctx context.Context, userID, sipUsername, ha1Hash string) error {
	query := `INSERT INTO sip_credentials (user_id, sip_username, ha1_hash) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, userID, sipUsername, ha1Hash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return repository.ErrConflict
		}
		return repository.ErrDatabase
	}
	return nil
}

func (r *PostgresRepository) DeleteSipCredential(ctx context.Context, sipUsername string) error {
	query := `DELETE FROM sip_credentials WHERE sip_username = $1`
	result, err := r.db.ExecContext(ctx, query, sipUsername)
	if err != nil {
		return repository.ErrDatabase
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// --- Helper / Internal ---

func (r *PostgresRepository) FetchContactsForUser(ctx context.Context, userID string) ([]*userv1.Contact, error) {
	query := `SELECT id, user_id, contact_type, contact_value, is_primary FROM contacts WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, repository.ErrDatabase
	}
	defer rows.Close()

	var contacts []*userv1.Contact
	for rows.Next() {
		var c userv1.Contact
		if err := rows.Scan(&c.Id, &c.UserId, &c.ContactType, &c.ContactValue, &c.IsPrimary); err != nil {
			return nil, repository.ErrDatabase
		}
		contacts = append(contacts, &c)
	}
	return contacts, nil
}
