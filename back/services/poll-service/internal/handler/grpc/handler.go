package grpc

import (
	"context"

	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
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
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	poll, err := h.svc.CreatePoll(
		ctx,
		userID,
		req.GetQuestion(),
		models.PollType(req.GetType()),
		req.GetOptions(),
		req.GetTags(),
		req.GetImageUrl(),
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
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	var question *string
	if req.Question != "" {
		q := req.Question
		question = &q
	}

	var imageURL *string
	if req.ImageUrl != nil {
		imageValue := req.GetImageUrl()
		imageURL = &imageValue
	}

	poll, err := h.svc.UpdatePoll(
		ctx,
		userID,
		req.GetId(),
		question,
		req.GetTags(),
		req.Tags != nil,
		imageURL,
	)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.UpdatePollResponse{Poll: mapPoll(poll)}, nil
}

func (h *Handler) DeletePoll(ctx context.Context, req *pollv1.DeletePollRequest) (*pollv1.DeletePollResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	if err := h.svc.DeletePoll(ctx, userID, req.GetId()); err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.DeletePollResponse{Success: true}, nil
}

func (h *Handler) CreateTag(ctx context.Context, req *pollv1.CreateTagRequest) (*pollv1.CreateTagResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
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

func (h *Handler) CreatePollImageUploadURL(ctx context.Context, req *pollv1.CreatePollImageUploadURLRequest) (*pollv1.CreatePollImageUploadURLResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	upload, err := h.svc.CreatePollImageUploadURL(ctx, userID, req.GetFilename(), req.GetContentType(), req.GetSizeBytes())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &pollv1.CreatePollImageUploadURLResponse{
		ObjectKey:        upload.ObjectKey,
		UploadUrl:        upload.UploadURL,
		ImageUrl:         upload.ImageURL,
		ExpiresInSeconds: upload.ExpiresInSeconds,
	}, nil
}
