package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/repository"
	mocksvc "github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/service/mock"
	"go.uber.org/mock/gomock"
)

func newAuthServiceForTest(
	users *mocksvc.MockUserRepository,
	sessions *mocksvc.MockSessionRepository,
	hasher *mocksvc.MockPasswordHasher,
	tokens *mocksvc.MockTokenManager,
	clock *mocksvc.MockClock,
	ids *mocksvc.MockIDGenerator,
) AuthService {
	var txMgr tx.Manager
	return NewAuthService(users, sessions, txMgr, hasher, tokens, clock, ids)
}

func TestLoginRejectsEmptyCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newAuthServiceForTest(
		mocksvc.NewMockUserRepository(ctrl),
		mocksvc.NewMockSessionRepository(ctrl),
		mocksvc.NewMockPasswordHasher(ctrl),
		mocksvc.NewMockTokenManager(ctrl),
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	_, err := svc.Login(context.Background(), "", "")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginMapsUserNotFoundToInvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mocksvc.NewMockUserRepository(ctrl)
	users.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(models.User{}, models.ErrUserNotFound)

	svc := newAuthServiceForTest(
		users,
		mocksvc.NewMockSessionRepository(ctrl),
		mocksvc.NewMockPasswordHasher(ctrl),
		mocksvc.NewMockTokenManager(ctrl),
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	_, err := svc.Login(context.Background(), "user@example.com", "pass")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mocksvc.NewMockUserRepository(ctrl)
	users.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(models.User{ID: "u1", PasswordHash: "stored"}, nil)

	hasher := mocksvc.NewMockPasswordHasher(ctrl)
	hasher.EXPECT().ComparePassword("stored", "bad").Return(errors.New("wrong password"))

	svc := newAuthServiceForTest(
		users,
		mocksvc.NewMockSessionRepository(ctrl),
		hasher,
		mocksvc.NewMockTokenManager(ctrl),
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	_, err := svc.Login(context.Background(), "user@example.com", "bad")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginSuccessIssuesTokensAndCreatesSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	accessExp := time.Now().UTC().Add(30 * time.Minute)
	refreshExp := now.Add(24 * time.Hour)

	users := mocksvc.NewMockUserRepository(ctrl)
	hasher := mocksvc.NewMockPasswordHasher(ctrl)
	tokens := mocksvc.NewMockTokenManager(ctrl)
	sessions := mocksvc.NewMockSessionRepository(ctrl)
	clock := mocksvc.NewMockClock(ctrl)
	ids := mocksvc.NewMockIDGenerator(ctrl)

	gomock.InOrder(
		users.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(models.User{ID: "u1", PasswordHash: "stored"}, nil),
		hasher.EXPECT().ComparePassword("stored", "pass").Return(nil),
		ids.EXPECT().NewID().Return("session-1"),
		tokens.EXPECT().GenerateAccessToken("u1").Return("acc-token", accessExp, nil),
		tokens.EXPECT().GenerateRefreshToken("u1", "session-1").Return("ref-token", refreshExp, nil),
		hasher.EXPECT().HashToken("ref-token").Return("hash(ref-token)"),
		clock.EXPECT().Now().Return(now),
	)

	sessions.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(models.RefreshSession{})).DoAndReturn(
		func(_ context.Context, session models.RefreshSession) (models.RefreshSession, error) {
			if session.ID != "session-1" {
				t.Fatalf("expected session id=session-1, got %s", session.ID)
			}
			if session.UserID != "u1" {
				t.Fatalf("expected session user=u1, got %s", session.UserID)
			}
			if session.RefreshTokenHash != "hash(ref-token)" {
				t.Fatalf("unexpected refresh token hash: %s", session.RefreshTokenHash)
			}
			if !session.CreatedAt.Equal(now) {
				t.Fatalf("unexpected created at: %v", session.CreatedAt)
			}
			if !session.ExpiresAt.Equal(refreshExp) {
				t.Fatalf("unexpected expires at: %v", session.ExpiresAt)
			}
			return session, nil
		},
	)

	svc := newAuthServiceForTest(users, sessions, hasher, tokens, clock, ids)

	result, err := svc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "acc-token" || result.RefreshToken != "ref-token" {
		t.Fatalf("unexpected tokens: %#v", result)
	}
	if result.ExpiresInSecond <= 0 {
		t.Fatalf("expected positive expires_in, got %d", result.ExpiresInSecond)
	}
}

func TestValidateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokens := mocksvc.NewMockTokenManager(ctrl)
	tokens.EXPECT().ValidateAccessToken("ok").Return("user-1", nil)
	tokens.EXPECT().ValidateAccessToken("bad").Return("", errors.New("bad token"))

	svc := newAuthServiceForTest(
		mocksvc.NewMockUserRepository(ctrl),
		mocksvc.NewMockSessionRepository(ctrl),
		mocksvc.NewMockPasswordHasher(ctrl),
		tokens,
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	userID, err := svc.ValidateToken(context.Background(), "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("unexpected userID: %s", userID)
	}

	_, err = svc.ValidateToken(context.Background(), "bad")
	if !errors.Is(err, models.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestLogoutAndGuards(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokens := mocksvc.NewMockTokenManager(ctrl)
	tokens.EXPECT().ParseRefreshToken("bad").Return("", "", errors.New("bad token"))

	svc := newAuthServiceForTest(
		mocksvc.NewMockUserRepository(ctrl),
		mocksvc.NewMockSessionRepository(ctrl),
		mocksvc.NewMockPasswordHasher(ctrl),
		tokens,
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	err := svc.Logout(context.Background(), "bad")
	if !errors.Is(err, models.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}

	err = svc.LogoutAll(context.Background(), "")
	if !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	_, err = svc.GetUser(context.Background(), "")
	if !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestUpdateUserGuards(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newAuthServiceForTest(
		mocksvc.NewMockUserRepository(ctrl),
		mocksvc.NewMockSessionRepository(ctrl),
		mocksvc.NewMockPasswordHasher(ctrl),
		mocksvc.NewMockTokenManager(ctrl),
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	country := "RU"
	_, err := svc.UpdateUser(context.Background(), "", UpdateUserInput{Country: &country})
	if !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	_, err = svc.UpdateUser(context.Background(), "u1", UpdateUserInput{})
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for empty payload, got %v", err)
	}

	email := "   "
	_, err = svc.UpdateUser(context.Background(), "u1", UpdateUserInput{Email: &email})
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for empty email, got %v", err)
	}

	birthYear := int32(-1)
	_, err = svc.UpdateUser(context.Background(), "u1", UpdateUserInput{BirthYear: &birthYear})
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for negative birth year, got %v", err)
	}
}

func TestUpdateUserSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mocksvc.NewMockUserRepository(ctrl)
	users.EXPECT().Update(gomock.Any(), "u1", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, patch repository.UserUpdatePatch) (models.User, error) {
			if patch.Email == nil || *patch.Email != "new@example.com" {
				t.Fatalf("unexpected email patch: %#v", patch.Email)
			}
			if patch.Country == nil || *patch.Country != "RU" {
				t.Fatalf("unexpected country patch: %#v", patch.Country)
			}
			if patch.Gender == nil || *patch.Gender != "female" {
				t.Fatalf("unexpected gender patch: %#v", patch.Gender)
			}
			if patch.BirthYear == nil || *patch.BirthYear != 1997 {
				t.Fatalf("unexpected birthYear patch: %#v", patch.BirthYear)
			}

			return models.User{
				ID:        "u1",
				Email:     *patch.Email,
				Country:   *patch.Country,
				Gender:    *patch.Gender,
				BirthYear: *patch.BirthYear,
			}, nil
		},
	)

	svc := newAuthServiceForTest(
		users,
		mocksvc.NewMockSessionRepository(ctrl),
		mocksvc.NewMockPasswordHasher(ctrl),
		mocksvc.NewMockTokenManager(ctrl),
		mocksvc.NewMockClock(ctrl),
		mocksvc.NewMockIDGenerator(ctrl),
	)

	email := " New@Example.com "
	country := " RU "
	gender := " female "
	birthYear := int32(1997)

	user, err := svc.UpdateUser(context.Background(), "u1", UpdateUserInput{
		Email:     &email,
		Country:   &country,
		Gender:    &gender,
		BirthYear: &birthYear,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != "u1" || user.Email != "new@example.com" {
		t.Fatalf("unexpected updated user: %#v", user)
	}
}
