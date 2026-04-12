package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
	mockrepo "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service/mock"
	"go.uber.org/mock/gomock"
)

func TestGetFeedInvalidCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockFeedRepository(ctrl)
	svc := NewFeedService(repo)

	_, _, _, err := svc.GetFeed(context.Background(), "bad-cursor", 20, nil)
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetFeedPaginatesAndEnriches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := mockrepo.NewMockFeedRepository(ctrl)

	capturedFilter := repository.FeedListFilter{}
	repo.EXPECT().GetFeed(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filter repository.FeedListFilter) ([]models.FeedItem, error) {
			capturedFilter = filter
			return []models.FeedItem{
				{ID: "a", CreatedAt: now.Add(2 * time.Minute)},
				{ID: "b", CreatedAt: now.Add(1 * time.Minute)},
				{ID: "c", CreatedAt: now},
			}, nil
		},
	)
	repo.EXPECT().GetOptionsByFeedItemIDs(gomock.Any(), []string{"a", "b"}).Return(
		map[string][]models.FeedItemOption{
			"a": {{ID: "o1", Text: "A"}},
			"b": {{ID: "o2", Text: "B"}},
		},
		nil,
	)
	repo.EXPECT().GetTagsByFeedItemIDs(gomock.Any(), []string{"a", "b"}).Return(
		map[string][]string{
			"a": {"x"},
			"b": {"y"},
		},
		nil,
	)

	svc := NewFeedService(repo)
	items, cursor, hasMore, err := svc.GetFeed(context.Background(), "", 2, []string{" Tag ", "tag", "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if cursor == "" {
		t.Fatalf("expected non-empty cursor")
	}

	cursorTime, cursorID, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("cursor must be decodable: %v", err)
	}
	if cursorID != "b" {
		t.Fatalf("expected cursorID=b, got %s", cursorID)
	}
	if !cursorTime.Equal(now.Add(1 * time.Minute)) {
		t.Fatalf("unexpected cursor time: %v", cursorTime)
	}

	if !reflect.DeepEqual(capturedFilter.Tags, []string{"tag", "x"}) {
		t.Fatalf("unexpected normalized tags: %#v", capturedFilter.Tags)
	}
	if capturedFilter.Limit != 3 {
		t.Fatalf("expected filter limit=3, got %d", capturedFilter.Limit)
	}

	if len(items[0].Options) != 1 || items[0].Options[0].ID != "o1" {
		t.Fatalf("first item options were not enriched: %#v", items[0].Options)
	}
	if len(items[1].Tags) != 1 || items[1].Tags[0] != "y" {
		t.Fatalf("second item tags were not enriched: %#v", items[1].Tags)
	}
}

func TestGetTrendingInvalidCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockFeedRepository(ctrl)
	svc := NewFeedService(repo)

	_, _, _, err := svc.GetTrending(context.Background(), "bad-cursor", 10)
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetUserPollsInvalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockFeedRepository(ctrl)
	svc := NewFeedService(repo)

	_, _, _, err := svc.GetUserPolls(context.Background(), "   ", "", 10)
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}
