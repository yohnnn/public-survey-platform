package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository"
	mockrepo "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/service/mock"
	"go.uber.org/mock/gomock"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fixedIDGenerator struct {
	id string
}

func (g fixedIDGenerator) NewID() string {
	return g.id
}

func newPollServiceForTest(polls repository.PollRepository, tags repository.TagRepository, outboxRepo repository.OutboxRepository, clock Clock, ids IDGenerator) PollService {
	var txMgr tx.Manager
	return NewPollService(polls, tags, outboxRepo, txMgr, clock, ids, nil)
}

func TestGetPollInvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newPollServiceForTest(
		mockrepo.NewMockPollRepository(ctrl),
		mockrepo.NewMockTagRepository(ctrl),
		mockrepo.NewMockOutboxRepository(ctrl),
		fixedClock{},
		fixedIDGenerator{id: "id"},
	)

	_, err := svc.GetPoll(context.Background(), "   ")
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetPollFillsEmptyCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	polls := mockrepo.NewMockPollRepository(ctrl)
	tags := mockrepo.NewMockTagRepository(ctrl)
	outboxRepo := mockrepo.NewMockOutboxRepository(ctrl)

	polls.EXPECT().GetByID(gomock.Any(), "poll-1").Return(models.Poll{ID: "poll-1", Question: "Q"}, nil)
	polls.EXPECT().GetOptionsByPollIDs(gomock.Any(), []string{"poll-1"}).Return(map[string][]models.PollOption{}, nil)
	polls.EXPECT().GetTagsByPollIDs(gomock.Any(), []string{"poll-1"}).Return(map[string][]string{}, nil)

	svc := newPollServiceForTest(polls, tags, outboxRepo, fixedClock{}, fixedIDGenerator{id: "id"})
	poll, err := svc.GetPoll(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poll.Options == nil {
		t.Fatalf("expected non-nil options slice")
	}
	if poll.Tags == nil {
		t.Fatalf("expected non-nil tags slice")
	}
}

func TestListPollsInvalidCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newPollServiceForTest(
		mockrepo.NewMockPollRepository(ctrl),
		mockrepo.NewMockTagRepository(ctrl),
		mockrepo.NewMockOutboxRepository(ctrl),
		fixedClock{},
		fixedIDGenerator{id: "id"},
	)

	_, _, _, err := svc.ListPolls(context.Background(), "bad-cursor", 20, nil)
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestListPollsPaginatesAndEnriches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	polls := mockrepo.NewMockPollRepository(ctrl)
	tags := mockrepo.NewMockTagRepository(ctrl)
	outboxRepo := mockrepo.NewMockOutboxRepository(ctrl)

	capturedFilter := repository.PollListFilter{}
	polls.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filter repository.PollListFilter) ([]models.Poll, error) {
			capturedFilter = filter
			return []models.Poll{
				{ID: "a", CreatedAt: now.Add(2 * time.Minute)},
				{ID: "b", CreatedAt: now.Add(1 * time.Minute)},
				{ID: "c", CreatedAt: now},
			}, nil
		},
	)
	polls.EXPECT().GetOptionsByPollIDs(gomock.Any(), []string{"a", "b"}).Return(
		map[string][]models.PollOption{
			"a": []models.PollOption{{ID: "o1", Text: "A"}},
			"b": []models.PollOption{{ID: "o2", Text: "B"}},
		},
		nil,
	)
	polls.EXPECT().GetTagsByPollIDs(gomock.Any(), []string{"a", "b"}).Return(
		map[string][]string{
			"a": {"x"},
			"b": {"y"},
		},
		nil,
	)

	svc := newPollServiceForTest(polls, tags, outboxRepo, fixedClock{}, fixedIDGenerator{id: "id"})
	items, cursor, hasMore, err := svc.ListPolls(context.Background(), "", 2, []string{"Tag", "tag", "x"})
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
		t.Fatalf("expected next cursor")
	}
	if capturedFilter.Limit != 3 {
		t.Fatalf("expected filter limit=3, got %d", capturedFilter.Limit)
	}
	if !reflect.DeepEqual(capturedFilter.Tags, []string{"tag", "x"}) {
		t.Fatalf("unexpected normalized tags: %#v", capturedFilter.Tags)
	}

	cursorTime, cursorID, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("cursor should decode: %v", err)
	}
	if cursorID != "b" || !cursorTime.Equal(now.Add(1*time.Minute)) {
		t.Fatalf("unexpected cursor payload id=%s time=%v", cursorID, cursorTime)
	}
}

func TestCreateTagNormalizesName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	polls := mockrepo.NewMockPollRepository(ctrl)
	tags := mockrepo.NewMockTagRepository(ctrl)
	outboxRepo := mockrepo.NewMockOutboxRepository(ctrl)

	captured := models.Tag{}
	tags.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(models.Tag{})).DoAndReturn(
		func(_ context.Context, tag models.Tag) (models.Tag, error) {
			captured = tag
			return tag, nil
		},
	)

	svc := newPollServiceForTest(polls, tags, outboxRepo, fixedClock{now: now}, fixedIDGenerator{id: "tag-1"})
	tag, err := svc.CreateTag(context.Background(), "  TeSt  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag.Name != "test" {
		t.Fatalf("expected normalized tag name 'test', got %q", tag.Name)
	}
	if captured.Name != "test" {
		t.Fatalf("expected repository to receive normalized name, got %q", captured.Name)
	}
	if tag.ID != "tag-1" {
		t.Fatalf("expected generated id tag-1, got %s", tag.ID)
	}
}
