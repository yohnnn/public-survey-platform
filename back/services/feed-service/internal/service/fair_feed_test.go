package service

import (
	"context"
	"testing"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
	mockrepo "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service/mock"
	"go.uber.org/mock/gomock"
)

func TestGetFairFeedMergesDiscoveryAndChronological(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := mockrepo.NewMockFeedRepository(ctrl)

	repo.EXPECT().GetDiscoveryFeed(gomock.Any(), gomock.Any()).Return([]models.FeedItem{
		{ID: "d1", ImpressionCount: 1, CreatedAt: now},
		{ID: "d2", ImpressionCount: 2, CreatedAt: now.Add(-time.Minute)},
	}, nil)
	repo.EXPECT().GetFeed(gomock.Any(), gomock.Any()).Return([]models.FeedItem{
		{ID: "c1", CreatedAt: now.Add(2 * time.Minute)},
		{ID: "c2", CreatedAt: now.Add(1 * time.Minute)},
		{ID: "c3", CreatedAt: now},
	}, nil)
	repo.EXPECT().GetOptionsByFeedItemIDs(gomock.Any(), gomock.Any()).Return(map[string][]models.FeedItemOption{}, nil).AnyTimes()
	repo.EXPECT().GetTagsByFeedItemIDs(gomock.Any(), gomock.Any()).Return(map[string][]string{}, nil).AnyTimes()

	svc := NewFeedService(repo, nil, FeedRankingConfig{
		DiscoverySlotsPerPage:     1,
		ChronologicalSlotsPerPage: 1,
	})

	items, cursor, hasMore, err := svc.GetFeed(context.Background(), "", 2, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "d1" || items[1].ID != "c1" {
		t.Fatalf("unexpected merge order: %#v", items)
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true")
	}
	if cursor == "" || len(cursor) < 5 {
		t.Fatalf("expected fair cursor, got %q", cursor)
	}
}

func TestMergeFairFeedItemsTracksBothCursors(t *testing.T) {
	now := time.Now().UTC()
	discovery := []models.FeedItem{{ID: "d1", ImpressionCount: 3, CreatedAt: now}}
	chron := []models.FeedItem{{ID: "c1", CreatedAt: now.Add(time.Hour)}}

	merged := mergeFairFeedItems(discovery, chron, 1, FeedRankingConfig{
		DiscoverySlotsPerPage:     1,
		ChronologicalSlotsPerPage: 1,
	})
	if merged.nextDiscovery == nil || merged.nextDiscovery.ID != "d1" {
		t.Fatalf("expected discovery cursor, got %#v", merged.nextDiscovery)
	}
	if merged.nextChron == nil || merged.nextChron.id != "c1" {
		t.Fatalf("expected chron cursor, got %#v", merged.nextChron)
	}
}

func TestDecodeFairCursorRoundTrip(t *testing.T) {
	created := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	discovery := &repository.DiscoveryCursor{ImpressionCount: 4, CreatedAt: created, ID: "poll-1"}
	chron := &chronCursorParts{createdAt: &created, id: "poll-2"}

	encoded := encodeFairCursor(discovery, chron)
	gotDiscovery, gotChron, err := decodeFairCursor(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if gotDiscovery.ImpressionCount != 4 || gotDiscovery.ID != "poll-1" {
		t.Fatalf("unexpected discovery cursor: %#v", gotDiscovery)
	}
	if gotChron.id != "poll-2" {
		t.Fatalf("unexpected chron cursor: %#v", gotChron)
	}
}
