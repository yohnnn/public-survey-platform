package service

import (
	"context"
	"errors"
	"testing"
	"time"

	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/models"
	mockrepo "github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/service/mock"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fixedVoteClock struct {
	now time.Time
}

func (c fixedVoteClock) Now() time.Time {
	return c.now
}

func newVoteServiceForTest(repo *mockrepo.MockVoteRepository, pollClient pollv1.PollServiceClient) VoteService {
	var txMgr tx.Manager
	return NewVoteService(repo, nil, nil, pollClient, txMgr, fixedVoteClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
}

func TestVoteRejectsInvalidArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockVoteRepository(ctrl)
	pollClient := mockrepo.NewMockPollServiceClient(ctrl)
	svc := newVoteServiceForTest(repo, pollClient)

	_, _, err := svc.Vote(context.Background(), "", "poll-1", []string{"o1"})
	if !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for empty user, got %v", err)
	}

	_, _, err = svc.Vote(context.Background(), "user-1", "", []string{"o1"})
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for empty poll, got %v", err)
	}

	_, _, err = svc.Vote(context.Background(), "user-1", "poll-1", []string{" ", ""})
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument for empty options, got %v", err)
	}
}

func TestVoteMapsPollNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockVoteRepository(ctrl)
	pollClient := mockrepo.NewMockPollServiceClient(ctrl)
	pollClient.EXPECT().GetPoll(gomock.Any(), &pollv1.GetPollRequest{Id: "poll-1"}).Return(nil, status.Error(codes.NotFound, "not found"))

	svc := newVoteServiceForTest(repo, pollClient)
	_, _, err := svc.Vote(context.Background(), "user-1", "poll-1", []string{"o1"})
	if !errors.Is(err, models.ErrPollNotFound) {
		t.Fatalf("expected ErrPollNotFound, got %v", err)
	}
}

func TestVoteRejectsInvalidOptionBeforeTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockVoteRepository(ctrl)
	pollClient := mockrepo.NewMockPollServiceClient(ctrl)
	pollClient.EXPECT().GetPoll(gomock.Any(), &pollv1.GetPollRequest{Id: "poll-1"}).Return(
		&pollv1.GetPollResponse{
			Poll: &pollv1.Poll{
				Id:      "poll-1",
				Type:    pollv1.PollType_POLL_TYPE_SINGLE_CHOICE,
				Options: []*pollv1.PollOption{{Id: "o1"}},
			},
		},
		nil,
	)

	svc := newVoteServiceForTest(repo, pollClient)
	_, _, err := svc.Vote(context.Background(), "user-1", "poll-1", []string{"o2"})
	if !errors.Is(err, models.ErrInvalidOption) {
		t.Fatalf("expected ErrInvalidOption, got %v", err)
	}
}

func TestGetUserVoteNoVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockrepo.NewMockVoteRepository(ctrl)
	repo.EXPECT().GetUserVote(gomock.Any(), "user-1", "poll-1").Return([]string{}, nil, nil)

	svc := newVoteServiceForTest(repo, mockrepo.NewMockPollServiceClient(ctrl))
	hasVote, options, votedAt, err := svc.GetUserVote(context.Background(), "user-1", "poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasVote {
		t.Fatalf("expected hasVote=false")
	}
	if options == nil || len(options) != 0 {
		t.Fatalf("expected empty non-nil options slice, got %#v", options)
	}
	if votedAt != nil {
		t.Fatalf("expected nil votedAt, got %v", votedAt)
	}
}

func TestGetUserVoteRejectsInvalidArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newVoteServiceForTest(mockrepo.NewMockVoteRepository(ctrl), mockrepo.NewMockPollServiceClient(ctrl))

	_, _, _, err := svc.GetUserVote(context.Background(), "", "poll-1")
	if !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	_, _, _, err = svc.GetUserVote(context.Background(), "user-1", "")
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestNormalizeAndDiffOptionIDs(t *testing.T) {
	normalized := normalizeOptionIDs([]string{" o2 ", "o1", "o1", ""})
	if len(normalized) != 2 || normalized[0] != "o1" || normalized[1] != "o2" {
		t.Fatalf("unexpected normalized options: %#v", normalized)
	}

	removed, added := diffOptionIDs([]string{"o1", "o2"}, []string{"o2", "o3"})
	if len(removed) != 1 || removed[0] != "o1" {
		t.Fatalf("unexpected removed diff: %#v", removed)
	}
	if len(added) != 1 || added[0] != "o3" {
		t.Fatalf("unexpected added diff: %#v", added)
	}
}
