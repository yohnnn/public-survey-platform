package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
	mockrepo "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service/mock"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

type followingReaderStub struct {
	userIDs   []string
	summaries []*userv1.UserSummary
	err       error
}

func (s followingReaderStub) ListMyFollowing(_ context.Context, _ *userv1.ListMyFollowingRequest, _ ...grpc.CallOption) (*userv1.ListMyFollowingResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &userv1.ListMyFollowingResponse{UserIds: s.userIDs}, nil
}

func (s followingReaderStub) BatchGetUserSummaries(_ context.Context, _ *userv1.BatchGetUserSummariesRequest, _ ...grpc.CallOption) (*userv1.BatchGetUserSummariesResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &userv1.BatchGetUserSummariesResponse{Items: s.summaries}, nil
}

func TestGetFeedInvalidCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockFeedRepository(ctrl)
	svc := NewFeedService(repo, nil)

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

	svc := NewFeedService(repo, nil)
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
	svc := NewFeedService(repo, nil)

	_, _, _, err := svc.GetTrending(context.Background(), "bad-cursor", 10)
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetUserPollsInvalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockFeedRepository(ctrl)
	svc := NewFeedService(repo, nil)

	_, _, _, err := svc.GetUserPolls(context.Background(), "   ", "", 10)
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetFollowingFeedPaginatesAndEnriches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := mockrepo.NewMockFeedRepository(ctrl)

	capturedFilter := repository.FeedListFilter{}
	repo.EXPECT().GetFollowingFeed(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filter repository.FeedListFilter) ([]models.FeedItem, error) {
			capturedFilter = filter
			return []models.FeedItem{
				{ID: "a", CreatorID: "u2", CreatedAt: now.Add(2 * time.Minute)},
				{ID: "b", CreatorID: "u3", CreatedAt: now.Add(1 * time.Minute)},
				{ID: "c", CreatorID: "u2", CreatedAt: now},
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

	svc := NewFeedService(repo, followingReaderStub{
		userIDs: []string{"u2", "u3", "u2"},
		summaries: []*userv1.UserSummary{
			{Id: "u2", Nickname: "alice"},
			{Id: "u3", Nickname: "bob"},
		},
	})
	items, cursor, hasMore, err := svc.GetFollowingFeed(context.Background(), "u1", "", 2)
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
	if !reflect.DeepEqual(capturedFilter.CreatorIDs, []string{"u2", "u3"}) {
		t.Fatalf("unexpected following ids: %#v", capturedFilter.CreatorIDs)
	}
	if capturedFilter.Limit != 3 {
		t.Fatalf("expected filter limit=3, got %d", capturedFilter.Limit)
	}
	if len(items[0].Options) != 1 || items[0].Options[0].ID != "o1" {
		t.Fatalf("first item options were not enriched: %#v", items[0].Options)
	}
	if items[0].Author.Nickname != "alice" {
		t.Fatalf("expected author nickname alice, got %#v", items[0].Author)
	}
}
