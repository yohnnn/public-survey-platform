package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository"
)

type pollService struct {
	polls repository.PollRepository
	tags  repository.TagRepository
	tx    tx.Manager
	clock Clock
	ids   IDGenerator
}

func NewPollService(
	polls repository.PollRepository,
	tags repository.TagRepository,
	tx tx.Manager,
	clock Clock,
	ids IDGenerator,
) PollService {
	return &pollService{
		polls: polls,
		tags:  tags,
		tx:    tx,
		clock: clock,
		ids:   ids,
	}
}

func (s *pollService) CreatePoll(ctx context.Context, userID, question string, pollType models.PollType, isAnonymous bool, endsAt *time.Time, options, tags []string) (models.Poll, error) {
	if strings.TrimSpace(userID) == "" {
		return models.Poll{}, models.ErrUnauthorized
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return models.Poll{}, models.ErrInvalidArgument
	}
	if pollType != models.PollTypeSingleChoice && pollType != models.PollTypeMultipleChoice {
		return models.Poll{}, models.ErrInvalidArgument
	}

	normalizedOptions := normalizeOptions(options)
	if len(normalizedOptions) < 2 {
		return models.Poll{}, models.ErrInvalidArgument
	}

	now := s.clock.Now().UTC()
	poll := models.Poll{
		ID:          s.ids.NewID(),
		CreatorID:   userID,
		Question:    question,
		Type:        pollType,
		IsAnonymous: isAnonymous,
		EndsAt:      normalizeEndsAt(endsAt),
		CreatedAt:   now,
	}

	pollOptions := make([]models.PollOption, 0, len(normalizedOptions))
	for i, optionText := range normalizedOptions {
		pollOptions = append(pollOptions, models.PollOption{
			ID:       s.ids.NewID(),
			Text:     optionText,
			Position: int32(i),
		})
	}

	normalizedTags := normalizeTags(tags)
	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		ensuredTags, ensureErr := s.tags.EnsureByNames(txCtx, normalizedTags)
		if ensureErr != nil {
			return ensureErr
		}

		tagIDs := make([]string, 0, len(ensuredTags))
		for _, tag := range ensuredTags {
			tagIDs = append(tagIDs, tag.ID)
		}

		if createErr := s.polls.Create(txCtx, poll, pollOptions, tagIDs); createErr != nil {
			return createErr
		}
		return nil
	})
	if err != nil {
		return models.Poll{}, err
	}

	return s.GetPoll(ctx, poll.ID)
}

func (s *pollService) GetPoll(ctx context.Context, id string) (models.Poll, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return models.Poll{}, models.ErrInvalidArgument
	}

	poll, err := s.polls.GetByID(ctx, id)
	if err != nil {
		return models.Poll{}, err
	}

	optionsMap, err := s.polls.GetOptionsByPollIDs(ctx, []string{id})
	if err != nil {
		return models.Poll{}, err
	}
	tagsMap, err := s.polls.GetTagsByPollIDs(ctx, []string{id})
	if err != nil {
		return models.Poll{}, err
	}

	poll.Options = optionsMap[id]
	poll.Tags = tagsMap[id]

	if poll.Options == nil {
		poll.Options = []models.PollOption{}
	}
	if poll.Tags == nil {
		poll.Tags = []string{}
	}

	return poll, nil
}

func (s *pollService) ListPolls(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.Poll, string, bool, error) {
	listLimit := int(limit)
	if listLimit <= 0 {
		listLimit = 20
	}
	if listLimit > 100 {
		listLimit = 100
	}

	filter := repository.PollListFilter{
		Limit: listLimit + 1,
		Tags:  normalizeTags(tags),
	}

	if strings.TrimSpace(cursor) != "" {
		createdAt, cursorID, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", false, models.ErrInvalidArgument
		}
		filter.CursorCreatedAt = &createdAt
		filter.CursorID = cursorID
	}

	items, err := s.polls.List(ctx, filter)
	if err != nil {
		return nil, "", false, err
	}

	hasMore := len(items) > listLimit
	if hasMore {
		items = items[:listLimit]
	}

	pollIDs := make([]string, 0, len(items))
	for _, item := range items {
		pollIDs = append(pollIDs, item.ID)
	}

	optionsMap, err := s.polls.GetOptionsByPollIDs(ctx, pollIDs)
	if err != nil {
		return nil, "", false, err
	}
	tagsMap, err := s.polls.GetTagsByPollIDs(ctx, pollIDs)
	if err != nil {
		return nil, "", false, err
	}

	for i := range items {
		items[i].Options = optionsMap[items[i].ID]
		items[i].Tags = tagsMap[items[i].ID]
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
	}

	return items, nextCursor, hasMore, nil
}

func (s *pollService) UpdatePoll(ctx context.Context, userID, id string, question *string, isAnonymous *bool, endsAt *time.Time, tags []string, tagsProvided bool) (models.Poll, error) {
	if strings.TrimSpace(userID) == "" {
		return models.Poll{}, models.ErrUnauthorized
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return models.Poll{}, models.ErrInvalidArgument
	}

	var patch repository.PollPatch
	if question != nil {
		q := strings.TrimSpace(*question)
		if q == "" {
			return models.Poll{}, models.ErrInvalidArgument
		}
		patch.Question = &q
	}
	if isAnonymous != nil {
		patch.IsAnonymous = isAnonymous
	}
	if endsAt != nil {
		normalized := endsAt.UTC()
		patch.EndsAt = &normalized
	}

	if patch.Question == nil && patch.IsAnonymous == nil && patch.EndsAt == nil && !tagsProvided {
		return models.Poll{}, models.ErrInvalidArgument
	}

	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		if updateErr := s.polls.UpdateByIDAndCreator(txCtx, id, userID, patch); updateErr != nil {
			if errors.Is(updateErr, models.ErrForbidden) {
				_, getErr := s.polls.GetByID(txCtx, id)
				if getErr != nil {
					return getErr
				}
			}
			return updateErr
		}

		if tagsProvided {
			normalizedTags := normalizeTags(tags)
			ensuredTags, ensureErr := s.tags.EnsureByNames(txCtx, normalizedTags)
			if ensureErr != nil {
				return ensureErr
			}
			tagIDs := make([]string, 0, len(ensuredTags))
			for _, tag := range ensuredTags {
				tagIDs = append(tagIDs, tag.ID)
			}
			if replaceErr := s.polls.ReplaceTags(txCtx, id, tagIDs); replaceErr != nil {
				return replaceErr
			}
		}
		return nil
	})
	if err != nil {
		return models.Poll{}, err
	}

	return s.GetPoll(ctx, id)
}

func (s *pollService) DeletePoll(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(userID) == "" {
		return models.ErrUnauthorized
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return models.ErrInvalidArgument
	}

	err := s.polls.DeleteByIDAndCreator(ctx, id, userID)
	if err == nil {
		return nil
	}
	if errors.Is(err, models.ErrForbidden) {
		if _, getErr := s.polls.GetByID(ctx, id); getErr != nil {
			return getErr
		}
	}
	return err
}

func (s *pollService) CreateTag(ctx context.Context, name string) (models.Tag, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return models.Tag{}, models.ErrInvalidArgument
	}

	tag := models.Tag{
		ID:        s.ids.NewID(),
		Name:      name,
		CreatedAt: s.clock.Now().UTC(),
	}
	return s.tags.Create(ctx, tag)
}

func (s *pollService) ListTags(ctx context.Context) ([]models.Tag, error) {
	return s.tags.List(ctx)
}

func normalizeOptions(options []string) []string {
	result := make([]string, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, raw := range options {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, v)
	}
	return result
}

func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		v := strings.TrimSpace(strings.ToLower(raw))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}

func normalizeEndsAt(endsAt *time.Time) *time.Time {
	if endsAt == nil {
		return nil
	}
	v := endsAt.UTC()
	return &v
}
