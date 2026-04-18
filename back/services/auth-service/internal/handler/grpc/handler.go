package grpc

import (
	"context"

	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/service"
)

type Handler struct {
	authv1.UnimplementedAuthServiceServer
	svc service.AuthService
}

func NewHandler(svc service.AuthService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	tokens, err := h.svc.Register(
		ctx,
		req.GetEmail(),
		req.GetPassword(),
		req.GetCountry(),
		req.GetGender(),
		req.GetBirthYear(),
	)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.RegisterResponse{Tokens: mapTokens(tokens)}, nil
}

func (h *Handler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokens, err := h.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.LoginResponse{Tokens: mapTokens(tokens)}, nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokens, err := h.svc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.RefreshTokenResponse{Tokens: mapTokens(tokens)}, nil
}

func (h *Handler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := h.svc.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.LogoutResponse{Success: true}, nil
}

func (h *Handler) LogoutAll(ctx context.Context, _ *authv1.LogoutAllRequest) (*authv1.LogoutAllResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.LogoutAll(ctx, userID); err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.LogoutAllResponse{Success: true}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	userID, err := h.svc.ValidateToken(ctx, req.GetAccessToken())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.ValidateTokenResponse{Valid: true, UserId: userID}, nil
}

func (h *Handler) GetUser(ctx context.Context, _ *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authv1.GetUserResponse{User: mapUser(user)}, nil
}

func (h *Handler) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.UpdateUserResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	input := service.UpdateUserInput{}
	if req.Email != nil {
		email := req.GetEmail()
		input.Email = &email
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

	return &authv1.UpdateUserResponse{User: mapUser(user)}, nil
}
