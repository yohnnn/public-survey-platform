package repository

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	GetByID(ctx context.Context, id string) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session models.RefreshSession) (models.RefreshSession, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (models.RefreshSession, error)
	RevokeByID(ctx context.Context, sessionID string, revokedAt time.Time) error
	RevokeAllByUserID(ctx context.Context, userID string, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
