package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository"
)

type pollService struct {
	polls  repository.PollRepository
	tags   repository.TagRepository
	outbox repository.OutboxRepository
	tx     tx.Manager
	clock  Clock
	ids    IDGenerator
	images PollImageUploader
}

func NewPollService(
	polls repository.PollRepository,
	tags repository.TagRepository,
	outbox repository.OutboxRepository,
	tx tx.Manager,
	clock Clock,
	ids IDGenerator,
	images PollImageUploader,
) PollService {
	return &pollService{
		polls:  polls,
		tags:   tags,
		outbox: outbox,
		tx:     tx,
		clock:  clock,
		ids:    ids,
		images: images,
	}
}

func (s *pollService) CreatePoll(ctx context.Context, userID, question string, pollType models.PollType, options, tags []string, imageURL string) (models.Poll, error) {
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

	normalizedImageURL, err := normalizeImageURL(imageURL)
	if err != nil {
		return models.Poll{}, err
	}

	now := s.clock.Now().UTC()
	poll := models.Poll{
		ID:        s.ids.NewID(),
		CreatorID: userID,
		Question:  question,
		Type:      pollType,
		ImageURL:  normalizedImageURL,
		CreatedAt: now,
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
	err = s.tx.WithTx(ctx, func(txCtx context.Context) error {
		eventID := s.ids.NewID()

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

		payload, eventErr := marshalPollCreatedPayload(eventID, poll, pollOptions, normalizedTags)
		if eventErr != nil {
			return eventErr
		}

		if outboxErr := s.outbox.Add(txCtx, outbox.Event{
			ID:      eventID,
			Topic:   events.TopicPollCreated,
			Key:     poll.ID,
			Payload: payload,
		}); outboxErr != nil {
			return outboxErr
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

func (s *pollService) UpdatePoll(ctx context.Context, userID, id string, question *string, tags []string, tagsProvided bool, imageURL *string) (models.Poll, error) {
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
	if imageURL != nil {
		normalizedImageURL, err := normalizeImageURL(*imageURL)
		if err != nil {
			return models.Poll{}, err
		}
		patch.ImageURL = &normalizedImageURL
	}

	if patch.Question == nil && patch.ImageURL == nil && !tagsProvided {
		return models.Poll{}, models.ErrInvalidArgument
	}

	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		eventID := s.ids.NewID()

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

		updatedPoll, getErr := s.polls.GetByID(txCtx, id)
		if getErr != nil {
			return getErr
		}
		tagsMap, getTagsErr := s.polls.GetTagsByPollIDs(txCtx, []string{id})
		if getTagsErr != nil {
			return getTagsErr
		}
		updatedPoll.Tags = tagsMap[id]
		if updatedPoll.Tags == nil {
			updatedPoll.Tags = []string{}
		}

		payload, marshalErr := marshalPollUpdatedPayload(eventID, updatedPoll, s.clock.Now().UTC())
		if marshalErr != nil {
			return marshalErr
		}

		return s.outbox.Add(txCtx, outbox.Event{
			ID:      eventID,
			Topic:   events.TopicPollUpdated,
			Key:     id,
			Payload: payload,
		})
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

	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		eventID := s.ids.NewID()

		deleteErr := s.polls.DeleteByIDAndCreator(txCtx, id, userID)
		if deleteErr != nil {
			if errors.Is(deleteErr, models.ErrForbidden) {
				_, getErr := s.polls.GetByID(txCtx, id)
				if getErr != nil {
					return getErr
				}
			}
			return deleteErr
		}

		payload, marshalErr := marshalPollDeletedPayload(eventID, id, s.clock.Now().UTC())
		if marshalErr != nil {
			return marshalErr
		}

		return s.outbox.Add(txCtx, outbox.Event{
			ID:      eventID,
			Topic:   events.TopicPollDeleted,
			Key:     id,
			Payload: payload,
		})
	})
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

func (s *pollService) CreatePollImageUploadURL(ctx context.Context, userID, fileName, contentType string, sizeBytes int64) (models.PollImageUpload, error) {
	if strings.TrimSpace(userID) == "" {
		return models.PollImageUpload{}, models.ErrUnauthorized
	}
	if s.images == nil {
		return models.PollImageUpload{}, models.ErrImageUploadOff
	}

	upload, err := s.images.CreatePollImageUploadURL(ctx, userID, fileName, contentType, sizeBytes)
	if err != nil {
		return models.PollImageUpload{}, err
	}

	upload.ObjectKey = strings.TrimSpace(upload.ObjectKey)
	upload.UploadURL = strings.TrimSpace(upload.UploadURL)
	upload.ImageURL = strings.TrimSpace(upload.ImageURL)
	if upload.ObjectKey == "" || upload.UploadURL == "" || upload.ImageURL == "" || upload.ExpiresInSeconds <= 0 {
		return models.PollImageUpload{}, models.ErrInvalidArgument
	}

	return upload, nil
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

func normalizeImageURL(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", nil
	}

	if len(v) > 2048 {
		return "", models.ErrInvalidImageURL
	}

	parsed, err := url.Parse(v)
	if err != nil {
		return "", models.ErrInvalidImageURL
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", models.ErrInvalidImageURL
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", models.ErrInvalidImageURL
	}

	return v, nil
}

type pollCreatedPayload struct {
	EventID   string       `json:"event_id"`
	PollID    string       `json:"poll_id"`
	CreatorID string       `json:"creator_id"`
	Question  string       `json:"question"`
	ImageURL  string       `json:"image_url,omitempty"`
	Type      int32        `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
	Options   []pollOption `json:"options"`
	Tags      []string     `json:"tags"`
}

type pollUpdatedPayload struct {
	EventID   string    `json:"event_id"`
	PollID    string    `json:"poll_id"`
	Question  string    `json:"question"`
	ImageURL  string    `json:"image_url,omitempty"`
	Tags      []string  `json:"tags"`
	UpdatedAt time.Time `json:"updated_at"`
}

type pollDeletedPayload struct {
	EventID   string    `json:"event_id"`
	PollID    string    `json:"poll_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type pollOption struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Position int32  `json:"position"`
}

func marshalPollCreatedPayload(eventID string, poll models.Poll, options []models.PollOption, tags []string) ([]byte, error) {
	payloadOptions := make([]pollOption, 0, len(options))
	for _, option := range options {
		payloadOptions = append(payloadOptions, pollOption{
			ID:       option.ID,
			Text:     option.Text,
			Position: option.Position,
		})
	}

	payload, err := json.Marshal(pollCreatedPayload{
		EventID:   eventID,
		PollID:    poll.ID,
		CreatorID: poll.CreatorID,
		Question:  poll.Question,
		ImageURL:  poll.ImageURL,
		Type:      int32(poll.Type),
		CreatedAt: poll.CreatedAt,
		Options:   payloadOptions,
		Tags:      tags,
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func marshalPollUpdatedPayload(eventID string, poll models.Poll, updatedAt time.Time) ([]byte, error) {
	payload, err := json.Marshal(pollUpdatedPayload{
		EventID:   eventID,
		PollID:    poll.ID,
		Question:  poll.Question,
		ImageURL:  poll.ImageURL,
		Tags:      poll.Tags,
		UpdatedAt: updatedAt,
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func marshalPollDeletedPayload(eventID, pollID string, deletedAt time.Time) ([]byte, error) {
	payload, err := json.Marshal(pollDeletedPayload{
		EventID:   eventID,
		PollID:    pollID,
		DeletedAt: deletedAt,
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}
