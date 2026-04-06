package grpc

import (
	"context"
	"time"

	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	grpcinterceptors "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/handler/grpc/interceptors"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/service"
)

type Handler struct {
	svc service.PollService
	pollv1.UnimplementedPollServiceServer
}

func NewHandler(svc service.PollService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreatePoll(ctx context.Context, req *pollv1.CreatePollRequest) (*pollv1.CreatePollResponse, error) {
	userID, ok := grpcinterceptors.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	endsAt, err := timestampToTime(req.GetEndsAt())
	if err != nil {
		return nil, toStatusError(err)
	}

	poll, err := h.svc.CreatePoll(
		ctx,
		userID,
		req.GetQuestion(),
		models.PollType(req.GetType()),
		req.GetIsAnonymous(),
		endsAt,
		req.GetOptions(),
		req.GetTags(),
	)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.CreatePollResponse{Poll: mapPoll(poll)}, nil
}

func (h *Handler) GetPoll(ctx context.Context, req *pollv1.GetPollRequest) (*pollv1.GetPollResponse, error) {
	poll, err := h.svc.GetPoll(ctx, req.GetId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.GetPollResponse{Poll: mapPoll(poll)}, nil
}

func (h *Handler) ListPolls(ctx context.Context, req *pollv1.ListPollsRequest) (*pollv1.ListPollsResponse, error) {
	items, nextCursor, hasMore, err := h.svc.ListPolls(ctx, req.GetCursor(), req.GetLimit(), req.GetTags())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.ListPollsResponse{
		Items:      mapPolls(items),
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (h *Handler) UpdatePoll(ctx context.Context, req *pollv1.UpdatePollRequest) (*pollv1.UpdatePollResponse, error) {
	userID, ok := grpcinterceptors.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	var question *string
	if req.Question != "" {
		q := req.Question
		question = &q
	}

	var endsAt *time.Time
	if req.EndsAt != nil {
		et, err := timestampToTime(req.EndsAt)
		if err != nil {
			return nil, toStatusError(err)
		}
		endsAt = et
	}

	var isAnonymous *bool
	if req.IsAnonymous != nil {
		anon := req.GetIsAnonymous()
		isAnonymous = &anon
	}

	poll, err := h.svc.UpdatePoll(
		ctx,
		userID,
		req.GetId(),
		question,
		isAnonymous,
		endsAt,
		req.GetTags(),
		req.Tags != nil,
	)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.UpdatePollResponse{Poll: mapPoll(poll)}, nil
}

func (h *Handler) DeletePoll(ctx context.Context, req *pollv1.DeletePollRequest) (*pollv1.DeletePollResponse, error) {
	userID, ok := grpcinterceptors.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.DeletePoll(ctx, userID, req.GetId()); err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.DeletePollResponse{Success: true}, nil
}

func (h *Handler) CreateTag(ctx context.Context, req *pollv1.CreateTagRequest) (*pollv1.CreateTagResponse, error) {
	userID, ok := grpcinterceptors.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	tag, err := h.svc.CreateTag(ctx, req.GetName())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.CreateTagResponse{Tag: mapTag(tag)}, nil
}

func (h *Handler) ListTags(ctx context.Context, _ *pollv1.ListTagsRequest) (*pollv1.ListTagsResponse, error) {
	items, err := h.svc.ListTags(ctx)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.ListTagsResponse{Items: mapTags(items)}, nil
}
