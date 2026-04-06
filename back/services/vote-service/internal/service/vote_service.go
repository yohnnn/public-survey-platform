package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/api/events"
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type voteService struct {
	repo       repository.VoteRepository
	outbox     repository.OutboxRepository
	pollClient pollv1.PollServiceClient
	tx         tx.Manager
	clock      Clock
}

func NewVoteService(repo repository.VoteRepository, outbox repository.OutboxRepository, pollClient pollv1.PollServiceClient, tx tx.Manager, clock Clock) VoteService {
	return &voteService{repo: repo, outbox: outbox, pollClient: pollClient, tx: tx, clock: clock}
}

func (s *voteService) Vote(ctx context.Context, userID, pollID string, optionIDs []string) ([]string, time.Time, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, time.Time{}, models.ErrUnauthorized
	}
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, time.Time{}, models.ErrInvalidArgument
	}

	normalizedOptions := normalizeOptionIDs(optionIDs)
	if len(normalizedOptions) == 0 {
		return nil, time.Time{}, models.ErrInvalidArgument
	}

	poll, err := s.getPoll(ctx, pollID)
	if err != nil {
		return nil, time.Time{}, err
	}

	if err := validateSelectedOptions(poll, normalizedOptions); err != nil {
		return nil, time.Time{}, err
	}

	now := s.clock.Now().UTC()
	if err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.ReplaceUserVote(txCtx, userID, pollID, normalizedOptions, now); err != nil {
			return err
		}

		payload, err := marshalVoteCastPayload(userID, pollID, normalizedOptions, now)
		if err != nil {
			return err
		}

		return s.outbox.Add(txCtx, outbox.Event{
			ID:      newEventID(),
			Topic:   events.TopicVoteCast,
			Key:     pollID,
			Payload: payload,
		})
	}); err != nil {
		return nil, time.Time{}, err
	}

	return normalizedOptions, now, nil
}

func (s *voteService) RemoveVote(ctx context.Context, userID, pollID string) error {
	if strings.TrimSpace(userID) == "" {
		return models.ErrUnauthorized
	}
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return models.ErrInvalidArgument
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		optionIDs, votedAt, err := s.repo.GetUserVote(txCtx, userID, pollID)
		if err != nil {
			return err
		}

		if err := s.repo.DeleteUserVote(txCtx, userID, pollID); err != nil {
			return err
		}

		if len(optionIDs) == 0 {
			return nil
		}

		removedAt := s.clock.Now().UTC()
		if votedAt != nil && !votedAt.IsZero() {
			v := votedAt.UTC()
			votedAt = &v
		}

		payload, err := marshalVoteRemovedPayload(userID, pollID, optionIDs, votedAt, removedAt)
		if err != nil {
			return err
		}

		return s.outbox.Add(txCtx, outbox.Event{
			ID:      newEventID(),
			Topic:   events.TopicVoteRemoved,
			Key:     pollID,
			Payload: payload,
		})
	})
}

func (s *voteService) GetUserVote(ctx context.Context, userID, pollID string) (bool, []string, *time.Time, error) {
	if strings.TrimSpace(userID) == "" {
		return false, nil, nil, models.ErrUnauthorized
	}
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return false, nil, nil, models.ErrInvalidArgument
	}

	optionIDs, votedAt, err := s.repo.GetUserVote(ctx, userID, pollID)
	if err != nil {
		return false, nil, nil, err
	}
	if len(optionIDs) == 0 {
		return false, []string{}, nil, nil
	}

	return true, optionIDs, votedAt, nil
}

func (s *voteService) getPoll(ctx context.Context, pollID string) (*pollv1.Poll, error) {
	resp, err := s.pollClient.GetPoll(ctx, &pollv1.GetPollRequest{Id: pollID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, models.ErrPollNotFound
		}
		return nil, err
	}
	if resp.GetPoll() == nil {
		return nil, models.ErrPollNotFound
	}
	return resp.GetPoll(), nil
}

func validateSelectedOptions(poll *pollv1.Poll, selected []string) error {
	allowed := make(map[string]struct{}, len(poll.GetOptions()))
	for _, option := range poll.GetOptions() {
		allowed[strings.TrimSpace(option.GetId())] = struct{}{}
	}

	for _, optionID := range selected {
		if _, ok := allowed[optionID]; !ok {
			return models.ErrInvalidOption
		}
	}

	if poll.GetType() == pollv1.PollType_POLL_TYPE_SINGLE_CHOICE && len(selected) != 1 {
		return models.ErrInvalidArgument
	}

	return nil
}

func normalizeOptionIDs(optionIDs []string) []string {
	seen := make(map[string]struct{}, len(optionIDs))
	out := make([]string, 0, len(optionIDs))
	for _, optionID := range optionIDs {
		v := strings.TrimSpace(optionID)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

type voteCastPayload struct {
	UserID    string    `json:"user_id"`
	PollID    string    `json:"poll_id"`
	OptionIDs []string  `json:"option_ids"`
	VotedAt   time.Time `json:"voted_at"`
}

type voteRemovedPayload struct {
	UserID    string     `json:"user_id"`
	PollID    string     `json:"poll_id"`
	OptionIDs []string   `json:"option_ids"`
	VotedAt   *time.Time `json:"voted_at,omitempty"`
	RemovedAt time.Time  `json:"removed_at"`
}

func marshalVoteCastPayload(userID, pollID string, optionIDs []string, votedAt time.Time) ([]byte, error) {
	return json.Marshal(voteCastPayload{
		UserID:    userID,
		PollID:    pollID,
		OptionIDs: optionIDs,
		VotedAt:   votedAt,
	})
}

func marshalVoteRemovedPayload(userID, pollID string, optionIDs []string, votedAt *time.Time, removedAt time.Time) ([]byte, error) {
	return json.Marshal(voteRemovedPayload{
		UserID:    userID,
		PollID:    pollID,
		OptionIDs: optionIDs,
		VotedAt:   votedAt,
		RemovedAt: removedAt,
	})
}

func newEventID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
