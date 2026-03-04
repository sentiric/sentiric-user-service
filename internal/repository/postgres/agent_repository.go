package postgres

import (
	"context"
	"database/sql"
	"errors"

	userv1 "github.com/sentiric/sentiric-contracts/gen/go/sentiric/user/v1"
	"github.com/sentiric/sentiric-user-service/internal/repository"
)

// GetAgentProfile: Ajan profilini getirir.
func (r *PostgresRepository) GetAgentProfile(ctx context.Context, userID string) (*userv1.AgentProfile, error) {
	query := `
		SELECT user_id, display_name, max_concurrent_calls, status 
		FROM agent_profiles 
		WHERE user_id = $1`

	var profile userv1.AgentProfile
	var displayName sql.NullString
	var statusStr string

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.UserId,
		&displayName,
		&profile.MaxConcurrentCalls,
		&statusStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		r.log.Error().Err(err).Str("user_id", userID).Msg("Ajan profili sorgulanamadı")
		return nil, repository.ErrDatabase
	}

	if displayName.Valid {
		profile.DisplayName = displayName.String
	}
	profile.Status = statusStr

	return &profile, nil
}

// UpsertAgentProfile: Ajan profilini oluşturur veya günceller.
func (r *PostgresRepository) UpsertAgentProfile(ctx context.Context, profile *userv1.AgentProfile, tenantID string) error {
	query := `
		INSERT INTO agent_profiles (user_id, tenant_id, display_name, max_concurrent_calls, status, last_status_change)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			max_concurrent_calls = EXCLUDED.max_concurrent_calls,
			status = EXCLUDED.status,
			last_status_change = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		profile.UserId,
		tenantID,
		profile.DisplayName,
		profile.MaxConcurrentCalls,
		profile.Status,
	)

	if err != nil {
		r.log.Error().Err(err).Msg("Ajan profili kaydedilemedi")
		return repository.ErrDatabase
	}
	return nil
}
