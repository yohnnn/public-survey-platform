package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
)

func (s *authService) issueTokens(ctx context.Context, userID string) (AuthTokens, error) {
	sessionID := s.ids.NewID()
	accessToken, accessExp, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return AuthTokens{}, err
	}

	refreshToken, refreshExp, err := s.tokens.GenerateRefreshToken(userID, sessionID)
	if err != nil {
		return AuthTokens{}, err
	}

	session := models.RefreshSession{
		ID:               sessionID,
		UserID:           userID,
		RefreshTokenHash: s.hasher.HashToken(refreshToken),
		ExpiresAt:        refreshExp,
		CreatedAt:        s.clock.Now().UTC(),
	}
	if _, err := s.sessions.Create(ctx, session); err != nil {
		return AuthTokens{}, err
	}

	expiresIn := int64(time.Until(accessExp).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	return AuthTokens{
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		ExpiresInSecond: expiresIn,
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

type randomIDGenerator struct{}

func (randomIDGenerator) NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func NewSystemClock() Clock {
	return systemClock{}
}

func NewRandomIDGenerator() IDGenerator {
	return randomIDGenerator{}
}
