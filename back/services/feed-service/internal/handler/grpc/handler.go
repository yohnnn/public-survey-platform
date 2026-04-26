package grpc

import (
	"context"
	"strings"

	feedv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/feed/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service"
)

type Handler struct {
	svc service.FeedService
	feedv1.UnimplementedFeedServiceServer
}

func NewHandler(svc service.FeedService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetFeed(ctx context.Context, req *feedv1.GetFeedRequest) (*feedv1.GetFeedResponse, error) {
	items, nextCursor, hasMore, err := h.svc.GetFeed(ctx, req.GetCursor(), req.GetLimit(), req.GetTags())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &feedv1.GetFeedResponse{
		Items: mapFeedItems(items),
		Page:  mapCursorPageMeta(nextCursor, hasMore, req.GetLimit()),
	}, nil
}

func (h *Handler) GetTrending(ctx context.Context, req *feedv1.GetTrendingRequest) (*feedv1.GetTrendingResponse, error) {
	items, nextCursor, hasMore, err := h.svc.GetTrending(ctx, req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &feedv1.GetTrendingResponse{
		Items: mapFeedItems(items),
		Page:  mapCursorPageMeta(nextCursor, hasMore, req.GetLimit()),
	}, nil
}

func (h *Handler) GetUserPolls(ctx context.Context, req *feedv1.GetUserPollsRequest) (*feedv1.GetUserPollsResponse, error) {
	items, nextCursor, hasMore, err := h.svc.GetUserPolls(ctx, req.GetUserId(), req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &feedv1.GetUserPollsResponse{
		Items: mapFeedItems(items),
		Page:  mapCursorPageMeta(nextCursor, hasMore, req.GetLimit()),
	}, nil
}

func (h *Handler) GetMyPolls(ctx context.Context, req *feedv1.GetMyPollsRequest) (*feedv1.GetUserPollsResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(userID) == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	items, nextCursor, hasMore, err := h.svc.GetUserPolls(ctx, userID, req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &feedv1.GetUserPollsResponse{
		Items: mapFeedItems(items),
		Page:  mapCursorPageMeta(nextCursor, hasMore, req.GetLimit()),
	}, nil
}

func (h *Handler) GetFollowingFeed(ctx context.Context, req *feedv1.GetFollowingFeedRequest) (*feedv1.GetFeedResponse, error) {
	userID, ok := grpcinterceptor.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(userID) == "" {
		return nil, toStatusError(models.ErrUnauthorized)
	}

	items, nextCursor, hasMore, err := h.svc.GetFollowingFeed(ctx, userID, req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &feedv1.GetFeedResponse{
		Items: mapFeedItems(items),
		Page:  mapCursorPageMeta(nextCursor, hasMore, req.GetLimit()),
	}, nil
}
