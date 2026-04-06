package grpc

import (
	"context"

	votev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/vote/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/service"
)

type Handler struct {
	votev1.UnimplementedVoteServiceServer
	svc service.VoteService
}

func NewHandler(svc service.VoteService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Vote(ctx context.Context, req *votev1.VoteRequest) (*votev1.VoteResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	optionIDs, votedAt, err := h.svc.Vote(ctx, userID, req.GetPollId(), req.GetOptionIds())
	if err != nil {
		return nil, toStatusError(err)
	}

	return mapVoteResponse(req.GetPollId(), optionIDs, votedAt), nil
}

func (h *Handler) RemoveVote(ctx context.Context, req *votev1.RemoveVoteRequest) (*votev1.RemoveVoteResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.RemoveVote(ctx, userID, req.GetPollId()); err != nil {
		return nil, toStatusError(err)
	}

	return &votev1.RemoveVoteResponse{Success: true}, nil
}

func (h *Handler) GetUserVote(ctx context.Context, req *votev1.GetUserVoteRequest) (*votev1.GetUserVoteResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	hasVoted, optionIDs, votedAt, err := h.svc.GetUserVote(ctx, userID, req.GetPollId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return mapGetUserVoteResponse(req.GetPollId(), hasVoted, optionIDs, votedAt), nil
}
