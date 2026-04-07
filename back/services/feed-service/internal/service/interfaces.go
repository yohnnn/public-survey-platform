package service

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
)

type FeedService interface {
	GetFeed(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.FeedItem, string, bool, error)
	GetTrending(ctx context.Context, cursor string, limit uint32) ([]models.FeedItem, string, bool, error)
	GetUserPolls(ctx context.Context, userID, cursor string, limit uint32) ([]models.FeedItem, string, bool, error)
}

type Clock interface {
	Now() time.Time
}
