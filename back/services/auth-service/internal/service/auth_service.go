package service

import (
	"context"
	"errors"
	"strings"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/repository"
)

type authService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	tx       tx.Manager
	hasher   PasswordHasher
	tokens   TokenManager
	clock    Clock
	ids      IDGenerator
}

func NewAuthService(
	users repository.UserRepository,
	sessions repository.SessionRepository,
	tx tx.Manager,
	hasher PasswordHasher,
	tokens TokenManager,
	clock Clock,
	ids IDGenerator,
) AuthService {
	return &authService{
		users:    users,
		sessions: sessions,
		tx:       tx,
		hasher:   hasher,
		tokens:   tokens,
		clock:    clock,
		ids:      ids,
	}
}

func (s *authService) Register(ctx context.Context, email, password, country, gender string, birthYear int32) (AuthTokens, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return AuthTokens{}, models.ErrInvalidCredentials
	}

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return AuthTokens{}, err
	}
	if exists {
		return AuthTokens{}, models.ErrEmailAlreadyExists
	}

	now := s.clock.Now().UTC()
	passwordHash, err := s.hasher.HashPassword(password)
	if err != nil {
		return AuthTokens{}, err
	}

	user := models.User{
		ID:           s.ids.NewID(),
		Email:        email,
		PasswordHash: passwordHash,
		Country:      strings.TrimSpace(country),
		Gender:       strings.TrimSpace(gender),
		BirthYear:    birthYear,
		CreatedAt:    now,
	}

	var out AuthTokens
	err = s.tx.WithTx(ctx, func(txCtx context.Context) error {
		created, createErr := s.users.Create(txCtx, user)
		if createErr != nil {
			return createErr
		}

		tokens, issueErr := s.issueTokens(txCtx, created.ID)
		if issueErr != nil {
			return issueErr
		}
		out = tokens
		return nil
	})
	if err != nil {
		return AuthTokens{}, err
	}

	return out, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (AuthTokens, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return AuthTokens{}, models.ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return AuthTokens{}, models.ErrInvalidCredentials
		}
		return AuthTokens{}, err
	}

	if err := s.hasher.ComparePassword(user.PasswordHash, password); err != nil {
		return AuthTokens{}, models.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (AuthTokens, error) {
	userIDFromToken, sessionIDFromToken, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil {
		return AuthTokens{}, models.ErrInvalidToken
	}

	tokenHash := s.hasher.HashToken(refreshToken)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			return AuthTokens{}, models.ErrUnauthorized
		}
		return AuthTokens{}, err
	}

	now := s.clock.Now().UTC()
	if session.RevokedAt != nil {
		return AuthTokens{}, models.ErrSessionRevoked
	}
	if now.After(session.ExpiresAt) {
		return AuthTokens{}, models.ErrSessionExpired
	}
	if session.ID != sessionIDFromToken || session.UserID != userIDFromToken {
		return AuthTokens{}, models.ErrInvalidToken
	}

	var out AuthTokens
	err = s.tx.WithTx(ctx, func(txCtx context.Context) error {
		if revokeErr := s.sessions.RevokeByID(txCtx, session.ID, now); revokeErr != nil {
			return revokeErr
		}
		tokens, issueErr := s.issueTokens(txCtx, session.UserID)
		if issueErr != nil {
			return issueErr
		}
		out = tokens
		return nil
	})
	if err != nil {
		return AuthTokens{}, err
	}

	return out, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	_, sessionID, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil {
		return models.ErrInvalidToken
	}
	return s.sessions.RevokeByID(ctx, sessionID, s.clock.Now().UTC())
}

func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return models.ErrUnauthorized
	}
	return s.sessions.RevokeAllByUserID(ctx, userID, s.clock.Now().UTC())
}

func (s *authService) ValidateToken(_ context.Context, accessToken string) (string, error) {
	userID, err := s.tokens.ValidateAccessToken(accessToken)
	if err != nil {
		return "", models.ErrInvalidToken
	}
	if strings.TrimSpace(userID) == "" {
		return "", models.ErrInvalidToken
	}
	return userID, nil
}

func (s *authService) GetUser(ctx context.Context, userID string) (models.User, error) {
	if strings.TrimSpace(userID) == "" {
		return models.User{}, models.ErrUnauthorized
	}
	return s.users.GetByID(ctx, userID)
}

func (s *authService) UpdateUser(ctx context.Context, userID string, input UpdateUserInput) (models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return models.User{}, models.ErrUnauthorized
	}

	if input.Email == nil && input.Country == nil && input.Gender == nil && input.BirthYear == nil {
		return models.User{}, models.ErrInvalidArgument
	}

	patch := repository.UserUpdatePatch{}

	if input.Email != nil {
		normalizedEmail := normalizeEmail(*input.Email)
		if normalizedEmail == "" {
			return models.User{}, models.ErrInvalidArgument
		}
		patch.Email = &normalizedEmail
	}

	if input.Country != nil {
		country := strings.TrimSpace(*input.Country)
		patch.Country = &country
	}

	if input.Gender != nil {
		gender := strings.TrimSpace(*input.Gender)
		patch.Gender = &gender
	}

	if input.BirthYear != nil {
		if *input.BirthYear < 0 {
			return models.User{}, models.ErrInvalidArgument
		}
		birthYear := *input.BirthYear
		patch.BirthYear = &birthYear
	}

	return s.users.Update(ctx, userID, patch)
}
