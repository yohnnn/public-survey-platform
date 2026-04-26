package repository

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	GetByID(ctx context.Context, id string) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	ListSummariesByIDs(ctx context.Context, ids []string) ([]models.UserSummary, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByNickname(ctx context.Context, nickname string) (bool, error)
	Update(ctx context.Context, id string, patch UserUpdatePatch) (models.User, error)
	CountFollowers(ctx context.Context, userID string) (int64, error)
	CountFollowing(ctx context.Context, userID string) (int64, error)
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
	ListFollowingIDs(ctx context.Context, userID string) ([]string, error)
	Follow(ctx context.Context, followerID, followeeID string, createdAt time.Time) error
	Unfollow(ctx context.Context, followerID, followeeID string) error
}

type UserUpdatePatch struct {
	Email     *string
	Nickname  *string
	Country   *string
	Gender    *string
	BirthYear *int32
}

type SessionRepository interface {
	Create(ctx context.Context, session models.RefreshSession) (models.RefreshSession, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (models.RefreshSession, error)
	RevokeByID(ctx context.Context, sessionID string, revokedAt time.Time) error
	RevokeAllByUserID(ctx context.Context, userID string, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
