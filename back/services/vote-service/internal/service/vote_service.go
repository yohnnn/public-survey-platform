package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type voteService struct {
	repo       repository.VoteRepository
	outbox     repository.OutboxRepository
	authClient userv1.UserServiceClient
	pollClient pollv1.PollServiceClient
	tx         tx.Manager
	clock      Clock
}

func NewVoteService(repo repository.VoteRepository, outbox repository.OutboxRepository, authClient userv1.UserServiceClient, pollClient pollv1.PollServiceClient, tx tx.Manager, clock Clock) VoteService {
	return &voteService{repo: repo, outbox: outbox, authClient: authClient, pollClient: pollClient, tx: tx, clock: clock}
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
	voterMeta := s.getVoterMeta(ctx)
	responseVotedAt := now
	if err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		existingOptionIDs, existingVotedAt, err := s.repo.GetUserVote(txCtx, userID, pollID)
		if err != nil {
			return err
		}

		removedOptionIDs, addedOptionIDs := diffOptionIDs(existingOptionIDs, normalizedOptions)
		if len(removedOptionIDs) == 0 && len(addedOptionIDs) == 0 {
			if existingVotedAt != nil {
				responseVotedAt = existingVotedAt.UTC()
			}
			return nil
		}

		if err := s.repo.ReplaceUserVote(txCtx, userID, pollID, normalizedOptions, now); err != nil {
			return err
		}

		if len(removedOptionIDs) > 0 {
			removedAt := now
			removedEventID := newEventID()
			payload, marshalErr := marshalVoteRemovedPayload(
				removedEventID,
				userID,
				pollID,
				removedOptionIDs,
				voterMeta,
				existingVotedAt,
				removedAt,
			)
			if marshalErr != nil {
				return marshalErr
			}

			if addErr := s.outbox.Add(txCtx, outbox.Event{
				ID:      removedEventID,
				Topic:   events.TopicVoteRemoved,
				Key:     pollID,
				Payload: payload,
			}); addErr != nil {
				return addErr
			}
		}

		if len(addedOptionIDs) == 0 {
			return nil
		}

		castEventID := newEventID()
		payload, err := marshalVoteCastPayload(castEventID, userID, pollID, addedOptionIDs, voterMeta, now)
		if err != nil {
			return err
		}

		return s.outbox.Add(txCtx, outbox.Event{
			ID:      castEventID,
			Topic:   events.TopicVoteCast,
			Key:     pollID,
			Payload: payload,
		})
	}); err != nil {
		return nil, time.Time{}, err
	}

	return normalizedOptions, responseVotedAt, nil
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

		voterMeta := s.getVoterMeta(ctx)

		eventID := newEventID()
		payload, err := marshalVoteRemovedPayload(eventID, userID, pollID, optionIDs, voterMeta, votedAt, removedAt)
		if err != nil {
			return err
		}

		return s.outbox.Add(txCtx, outbox.Event{
			ID:      eventID,
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
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	PollID    string    `json:"poll_id"`
	OptionIDs []string  `json:"option_ids"`
	Country   string    `json:"country,omitempty"`
	Gender    string    `json:"gender,omitempty"`
	BirthYear *int32    `json:"birth_year,omitempty"`
	VotedAt   time.Time `json:"voted_at"`
}

type voteRemovedPayload struct {
	EventID   string     `json:"event_id"`
	UserID    string     `json:"user_id"`
	PollID    string     `json:"poll_id"`
	OptionIDs []string   `json:"option_ids"`
	Country   string     `json:"country,omitempty"`
	Gender    string     `json:"gender,omitempty"`
	BirthYear *int32     `json:"birth_year,omitempty"`
	VotedAt   *time.Time `json:"voted_at,omitempty"`
	RemovedAt time.Time  `json:"removed_at"`
}

type voterMeta struct {
	country   string
	gender    string
	birthYear *int32
}

func marshalVoteCastPayload(eventID, userID, pollID string, optionIDs []string, meta voterMeta, votedAt time.Time) ([]byte, error) {
	return json.Marshal(voteCastPayload{
		EventID:   eventID,
		UserID:    userID,
		PollID:    pollID,
		OptionIDs: optionIDs,
		Country:   meta.country,
		Gender:    meta.gender,
		BirthYear: meta.birthYear,
		VotedAt:   votedAt,
	})
}

func marshalVoteRemovedPayload(eventID, userID, pollID string, optionIDs []string, meta voterMeta, votedAt *time.Time, removedAt time.Time) ([]byte, error) {
	return json.Marshal(voteRemovedPayload{
		EventID:   eventID,
		UserID:    userID,
		PollID:    pollID,
		OptionIDs: optionIDs,
		Country:   meta.country,
		Gender:    meta.gender,
		BirthYear: meta.birthYear,
		VotedAt:   votedAt,
		RemovedAt: removedAt,
	})
}

func diffOptionIDs(previous, current []string) ([]string, []string) {
	previousSet := make(map[string]struct{}, len(previous))
	currentSet := make(map[string]struct{}, len(current))

	for _, optionID := range previous {
		optionID = strings.TrimSpace(optionID)
		if optionID == "" {
			continue
		}
		previousSet[optionID] = struct{}{}
	}
	for _, optionID := range current {
		optionID = strings.TrimSpace(optionID)
		if optionID == "" {
			continue
		}
		currentSet[optionID] = struct{}{}
	}

	removed := make([]string, 0, len(previousSet))
	for optionID := range previousSet {
		if _, ok := currentSet[optionID]; ok {
			continue
		}
		removed = append(removed, optionID)
	}

	added := make([]string, 0, len(currentSet))
	for optionID := range currentSet {
		if _, ok := previousSet[optionID]; ok {
			continue
		}
		added = append(added, optionID)
	}

	sort.Strings(removed)
	sort.Strings(added)

	return removed, added
}

func (s *voteService) getVoterMeta(ctx context.Context) voterMeta {
	if s.authClient == nil {
		return voterMeta{}
	}

	inMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return voterMeta{}
	}

	authVals := inMD.Get("authorization")
	if len(authVals) == 0 || strings.TrimSpace(authVals[0]) == "" {
		return voterMeta{}
	}

	outMD := metadata.Pairs("authorization", authVals[0])
	outCtx := metadata.NewOutgoingContext(ctx, outMD)

	resp, err := s.authClient.GetMyUser(outCtx, &userv1.GetMyUserRequest{})
	if err != nil || resp.GetUser() == nil {
		return voterMeta{}
	}

	user := resp.GetUser()
	meta := voterMeta{
		country: strings.TrimSpace(user.GetCountry()),
		gender:  strings.TrimSpace(user.GetGender()),
	}

	if birthYear := user.GetBirthYear(); birthYear > 0 {
		v := birthYear
		meta.birthYear = &v
	}

	return meta
}

func newEventID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
