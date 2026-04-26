package service

import (
	"context"
	"time"

	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"google.golang.org/grpc"
)

type FeedService interface {
	GetFeed(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.FeedItem, string, bool, error)
	GetTrending(ctx context.Context, cursor string, limit uint32) ([]models.FeedItem, string, bool, error)
	GetUserPolls(ctx context.Context, userID, cursor string, limit uint32) ([]models.FeedItem, string, bool, error)
	GetFollowingFeed(ctx context.Context, userID, cursor string, limit uint32) ([]models.FeedItem, string, bool, error)
}

type FollowingReader interface {
	ListMyFollowing(ctx context.Context, in *userv1.ListMyFollowingRequest, opts ...grpc.CallOption) (*userv1.ListMyFollowingResponse, error)
	BatchGetUserSummaries(ctx context.Context, in *userv1.BatchGetUserSummariesRequest, opts ...grpc.CallOption) (*userv1.BatchGetUserSummariesResponse, error)
}

type Clock interface {
	Now() time.Time
}
