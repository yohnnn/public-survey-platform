package grpc

import (
	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapTokens(tokens service.AuthTokens) *authv1.AuthTokens {
	return &authv1.AuthTokens{
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		ExpiresInSeconds: tokens.ExpiresInSecond,
	}
}

func mapUser(user models.User) *authv1.User {
	return &authv1.User{
		Id:        user.ID,
		Email:     user.Email,
		Country:   user.Country,
		Gender:    user.Gender,
		BirthYear: user.BirthYear,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}
