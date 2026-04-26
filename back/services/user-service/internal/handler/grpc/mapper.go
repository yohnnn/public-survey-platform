package grpc

import (
	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapTokens(tokens service.AuthTokens) *userv1.AuthTokens {
	return &userv1.AuthTokens{
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		ExpiresInSeconds: tokens.ExpiresInSecond,
	}
}

func mapUser(user models.User) *userv1.User {
	return &userv1.User{
		Id:        user.ID,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
		Gender:    user.Gender,
		BirthYear: user.BirthYear,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}

func mapPublicProfile(profile models.PublicProfile) *userv1.PublicUserProfile {
	return &userv1.PublicUserProfile{
		Id:             profile.ID,
		Nickname:       profile.Nickname,
		FollowersCount: profile.FollowersCount,
		FollowingCount: profile.FollowingCount,
		IsFollowing:    profile.IsFollowing,
	}
}

func mapUserSummaries(items []models.UserSummary) []*userv1.UserSummary {
	out := make([]*userv1.UserSummary, 0, len(items))
	for _, item := range items {
		out = append(out, &userv1.UserSummary{
			Id:       item.ID,
			Nickname: item.Nickname,
		})
	}
	return out
}
