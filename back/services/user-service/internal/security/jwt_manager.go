package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type tokenClaims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid,omitempty"`
	Type      string `json:"typ"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *JWTManager) GenerateAccessToken(userID string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.accessTTL)

	claims := tokenClaims{
		UserID: userID,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, exp, nil
}

func (m *JWTManager) GenerateRefreshToken(userID, sessionID string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.refreshTTL)

	claims := tokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, exp, nil
}

func (m *JWTManager) ValidateAccessToken(token string) (string, error) {
	claims, err := m.parse(token)
	if err != nil {
		return "", err
	}
	if claims.Type != "access" || claims.UserID == "" {
		return "", errors.New("invalid access token claims")
	}
	return claims.UserID, nil
}

func (m *JWTManager) ParseRefreshToken(token string) (string, string, error) {
	claims, err := m.parse(token)
	if err != nil {
		return "", "", err
	}
	if claims.Type != "refresh" || claims.UserID == "" || claims.SessionID == "" {
		return "", "", errors.New("invalid refresh token claims")
	}
	return claims.UserID, claims.SessionID, nil
}

func (m *JWTManager) parse(token string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
