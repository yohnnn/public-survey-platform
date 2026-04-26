package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, session models.RefreshSession) (models.RefreshSession, error) {
	const query = `
		INSERT INTO refresh_sessions (id, user_id, refresh_token_hash, expires_at, created_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, refresh_token_hash, expires_at, created_at, revoked_at
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.RefreshSession
	err := exec.QueryRow(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.RefreshTokenHash,
		session.ExpiresAt,
		session.CreatedAt,
		session.RevokedAt,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.RefreshTokenHash,
		&out.ExpiresAt,
		&out.CreatedAt,
		&out.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RefreshSession{}, models.ErrSessionNotFound
		}
		return models.RefreshSession{}, err
	}

	return out, nil
}

func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (models.RefreshSession, error) {
	const query = `
		SELECT id, user_id, refresh_token_hash, expires_at, created_at, revoked_at
		FROM refresh_sessions
		WHERE refresh_token_hash = $1
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.RefreshSession
	err := exec.QueryRow(ctx, query, tokenHash).Scan(
		&out.ID,
		&out.UserID,
		&out.RefreshTokenHash,
		&out.ExpiresAt,
		&out.CreatedAt,
		&out.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RefreshSession{}, models.ErrSessionNotFound
		}
		return models.RefreshSession{}, err
	}

	return out, nil
}

func (r *SessionRepository) RevokeByID(ctx context.Context, sessionID string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_sessions
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	exec := tx.Executor(ctx, r.pool)
	cmd, err := exec.Exec(ctx, query, sessionID, revokedAt)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrSessionNotFound
	}
	return nil
}

func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_sessions
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, userID, revokedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	const query = `DELETE FROM refresh_sessions WHERE expires_at <= $1`

	exec := tx.Executor(ctx, r.pool)
	cmd, err := exec.Exec(ctx, query, now)
	if err != nil {
		return 0, err
	}

	return cmd.RowsAffected(), nil
}
