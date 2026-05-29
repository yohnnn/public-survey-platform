package repository

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
)

type DiscoveryCursor struct {
	ImpressionCount int64
	CreatedAt       time.Time
	ID              string
}

type FeedListFilter struct {
	CursorCreatedAt *time.Time
	CursorID        string
	CursorVotes     *int64
	DiscoveryCursor *DiscoveryCursor
	ExposureMaxAge  time.Duration
	CreatorID       string
	CreatorIDs      []string
	Limit           int
	Tags            []string
}

type FeedRepository interface {
	CreateFeedItem(ctx context.Context, item models.FeedItem, options []models.FeedItemOption, tags []string) error
	UpdateFeedItem(ctx context.Context, item models.FeedItem, tags []string) error
	DeleteFeedItem(ctx context.Context, feedItemID string) error
	IncrementOptionVotes(ctx context.Context, optionID string, delta int64) (bool, error)
	UpdateTotalVotes(ctx context.Context, feedItemID string, delta int64) (bool, error)
	AddPendingOptionVotes(ctx context.Context, pollID, optionID string, delta int64) error
	AddPendingTotalVotes(ctx context.Context, pollID string, delta int64) error
	ApplyPendingVotes(ctx context.Context, feedItemID string) error
	MarkEventProcessed(ctx context.Context, eventID, topic string) (bool, error)
	GetFeed(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetDiscoveryFeed(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	RecordFeedImpressions(ctx context.Context, viewerKey string, feedItemIDs []string) (int, error)
	GetTrending(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetUserPolls(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetFollowingFeed(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetOptionsByFeedItemIDs(ctx context.Context, feedItemIDs []string) (map[string][]models.FeedItemOption, error)
	GetTagsByFeedItemIDs(ctx context.Context, feedItemIDs []string) (map[string][]string, error)
}
