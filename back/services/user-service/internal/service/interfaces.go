package service

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
)

type UserService interface {
	Register(ctx context.Context, email, password, nickname, country, gender string, birthYear int32) (AuthTokens, error)
	Login(ctx context.Context, email, password string) (AuthTokens, error)
	RefreshToken(ctx context.Context, refreshToken string) (AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	ValidateToken(ctx context.Context, accessToken string) (string, error)
	GetUser(ctx context.Context, userID string) (models.User, error)
	GetPublicProfile(ctx context.Context, requesterID, targetUserID string) (models.PublicProfile, error)
	ListUserSummaries(ctx context.Context, userIDs []string) ([]models.UserSummary, error)
	ListFollowingIDs(ctx context.Context, userID string) ([]string, error)
	UpdateUser(ctx context.Context, userID string, input UpdateUserInput) (models.User, error)
	FollowUser(ctx context.Context, followerID, followeeID string) error
	UnfollowUser(ctx context.Context, followerID, followeeID string) error
}

type UpdateUserInput struct {
	Email     *string
	Nickname  *string
	Country   *string
	Gender    *string
	BirthYear *int32
}

type AuthTokens struct {
	AccessToken     string
	RefreshToken    string
	ExpiresInSecond int64
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hash, password string) error
	HashToken(token string) string
}

type TokenManager interface {
	GenerateAccessToken(userID string) (token string, expiresAt time.Time, err error)
	GenerateRefreshToken(userID, sessionID string) (token string, expiresAt time.Time, err error)
	ValidateAccessToken(token string) (userID string, err error)
	ParseRefreshToken(token string) (userID string, sessionID string, err error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}
