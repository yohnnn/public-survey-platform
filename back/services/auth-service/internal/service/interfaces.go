package service

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
)

type AuthService interface {
	Register(ctx context.Context, email, password, country, gender string, birthYear int32) (AuthTokens, error)
	Login(ctx context.Context, email, password string) (AuthTokens, error)
	RefreshToken(ctx context.Context, refreshToken string) (AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	ValidateToken(ctx context.Context, accessToken string) (string, error)
	GetUser(ctx context.Context, userID string) (models.User, error)
	UpdateUser(ctx context.Context, userID string, input UpdateUserInput) (models.User, error)
}

type UpdateUserInput struct {
	Email     *string
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
