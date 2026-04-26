package grpc

import (
	"context"
	"strings"

	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/service"
)

type Handler struct {
	userv1.UnimplementedUserServiceServer
	svc service.UserService
}

func NewHandler(svc service.UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	tokens, err := h.svc.Register(
		ctx,
		req.GetEmail(),
		req.GetPassword(),
		req.GetNickname(),
		req.GetCountry(),
		req.GetGender(),
		req.GetBirthYear(),
	)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.RegisterResponse{Tokens: mapTokens(tokens)}, nil
}

func (h *Handler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	tokens, err := h.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.LoginResponse{Tokens: mapTokens(tokens)}, nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	tokens, err := h.svc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.RefreshTokenResponse{Tokens: mapTokens(tokens)}, nil
}

func (h *Handler) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	if err := h.svc.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.LogoutResponse{Success: true}, nil
}

func (h *Handler) LogoutAll(ctx context.Context, _ *userv1.LogoutAllRequest) (*userv1.LogoutAllResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.LogoutAll(ctx, userID); err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.LogoutAllResponse{Success: true}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	userID, err := h.svc.ValidateToken(ctx, req.GetAccessToken())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.ValidateTokenResponse{Valid: true, UserId: userID}, nil
}

func (h *Handler) GetMyUser(ctx context.Context, _ *userv1.GetMyUserRequest) (*userv1.GetMyUserResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.GetMyUserResponse{User: mapUser(user)}, nil
}

func (h *Handler) UpdateMyUser(ctx context.Context, req *userv1.UpdateMyUserRequest) (*userv1.UpdateMyUserResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	input := service.UpdateUserInput{}
	if req.Email != nil {
		email := req.GetEmail()
		input.Email = &email
	}
	if req.Nickname != nil {
		nickname := req.GetNickname()
		input.Nickname = &nickname
	}
	if req.Country != nil {
		country := req.GetCountry()
		input.Country = &country
	}
	if req.Gender != nil {
		gender := req.GetGender()
		input.Gender = &gender
	}
	if req.BirthYear != nil {
		birthYear := req.GetBirthYear()
		input.BirthYear = &birthYear
	}

	user, err := h.svc.UpdateUser(ctx, userID, input)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.UpdateMyUserResponse{User: mapUser(user)}, nil
}

func (h *Handler) GetPublicProfile(ctx context.Context, req *userv1.GetPublicProfileRequest) (*userv1.GetPublicProfileResponse, error) {
	requesterID := ""
	if userID, ok := grpcinterceptor.UserIDFromContext(ctx); ok {
		requesterID = strings.TrimSpace(userID)
	}

	profile, err := h.svc.GetPublicProfile(ctx, requesterID, req.GetUserId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.GetPublicProfileResponse{Profile: mapPublicProfile(profile)}, nil
}

func (h *Handler) FollowUser(ctx context.Context, req *userv1.FollowUserRequest) (*userv1.FollowUserResponse, error) {
	followerID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(followerID) == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.FollowUser(ctx, followerID, req.GetUserId()); err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.FollowUserResponse{Success: true}, nil
}

func (h *Handler) UnfollowUser(ctx context.Context, req *userv1.UnfollowUserRequest) (*userv1.UnfollowUserResponse, error) {
	followerID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(followerID) == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.UnfollowUser(ctx, followerID, req.GetUserId()); err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.UnfollowUserResponse{Success: true}, nil
}

func (h *Handler) ListMyFollowing(ctx context.Context, _ *userv1.ListMyFollowingRequest) (*userv1.ListMyFollowingResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(userID) == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	userIDs, err := h.svc.ListFollowingIDs(ctx, userID)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.ListMyFollowingResponse{UserIds: userIDs}, nil
}

func (h *Handler) BatchGetUserSummaries(ctx context.Context, req *userv1.BatchGetUserSummariesRequest) (*userv1.BatchGetUserSummariesResponse, error) {
	items, err := h.svc.ListUserSummaries(ctx, req.GetUserIds())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &userv1.BatchGetUserSummariesResponse{Items: mapUserSummaries(items)}, nil
}
