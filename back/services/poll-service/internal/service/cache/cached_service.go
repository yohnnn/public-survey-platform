package servicecache

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	cachepkg "github.com/yohnnn/public-survey-platform/back/pkg/cache"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/service"
)

type Config struct {
	PollTTL     time.Duration
	PollListTTL time.Duration
	TagsTTL     time.Duration
}

func DefaultConfig() Config {
	return Config{
		PollTTL:     30 * time.Second,
		PollListTTL: 30 * time.Second,
		TagsTTL:     5 * time.Minute,
	}
}

type pollService struct {
	next  service.PollService
	store cachepkg.Store
	cfg   Config
}

func NewPollService(next service.PollService, store cachepkg.Store, cfg Config) service.PollService {
	if next == nil || store == nil {
		return next
	}

	if cfg.PollTTL <= 0 {
		cfg.PollTTL = 30 * time.Second
	}
	if cfg.PollListTTL <= 0 {
		cfg.PollListTTL = 30 * time.Second
	}
	if cfg.TagsTTL <= 0 {
		cfg.TagsTTL = 5 * time.Minute
	}

	return &pollService{next: next, store: store, cfg: cfg}
}

func (s *pollService) CreatePoll(ctx context.Context, userID, question string, pollType models.PollType, isAnonymous bool, endsAt *time.Time, options, tags []string) (models.Poll, error) {
	poll, err := s.next.CreatePoll(ctx, userID, question, pollType, isAnonymous, endsAt, options, tags)
	if err != nil {
		return poll, err
	}

	_ = cachepkg.SetJSON(ctx, s.store, pollCacheKey(poll.ID), poll, s.cfg.PollTTL)
	_, _ = s.store.Increment(ctx, pollListVersionKey)
	_, _ = s.store.Increment(ctx, tagsVersionKey)

	return poll, nil
}

func (s *pollService) GetPoll(ctx context.Context, id string) (models.Poll, error) {
	key := pollCacheKey(id)
	var cached models.Poll

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	poll, err := s.next.GetPoll(ctx, id)
	if err != nil {
		return poll, err
	}

	_ = cachepkg.SetJSON(ctx, s.store, key, poll, s.cfg.PollTTL)
	return poll, nil
}

func (s *pollService) ListPolls(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.Poll, string, bool, error) {
	version := int64(0)
	versionValue, foundVersion, err := cachepkg.GetInt64(ctx, s.store, pollListVersionKey)
	if err == nil && foundVersion {
		version = versionValue
	}

	if err == nil {
		key := pollListCacheKey(cursor, limit, tags, version)
		var cached pollListCacheValue
		found, readErr := cachepkg.GetJSON(ctx, s.store, key, &cached)
		if readErr == nil && found {
			return cached.Items, cached.NextCursor, cached.HasMore, nil
		}

		items, nextCursor, hasMore, listErr := s.next.ListPolls(ctx, cursor, limit, tags)
		if listErr != nil {
			return nil, "", false, listErr
		}

		_ = cachepkg.SetJSON(ctx, s.store, key, pollListCacheValue{Items: items, NextCursor: nextCursor, HasMore: hasMore}, s.cfg.PollListTTL)
		return items, nextCursor, hasMore, nil
	}

	return s.next.ListPolls(ctx, cursor, limit, tags)
}

func (s *pollService) UpdatePoll(ctx context.Context, userID, id string, question *string, isAnonymous *bool, endsAt *time.Time, tags []string, tagsProvided bool) (models.Poll, error) {
	poll, err := s.next.UpdatePoll(ctx, userID, id, question, isAnonymous, endsAt, tags, tagsProvided)
	if err != nil {
		return poll, err
	}

	_ = cachepkg.SetJSON(ctx, s.store, pollCacheKey(poll.ID), poll, s.cfg.PollTTL)
	_, _ = s.store.Increment(ctx, pollListVersionKey)
	if tagsProvided {
		_, _ = s.store.Increment(ctx, tagsVersionKey)
	}

	return poll, nil
}

func (s *pollService) DeletePoll(ctx context.Context, userID, id string) error {
	if err := s.next.DeletePoll(ctx, userID, id); err != nil {
		return err
	}

	_ = s.store.Delete(ctx, pollCacheKey(id))
	_, _ = s.store.Increment(ctx, pollListVersionKey)
	return nil
}

func (s *pollService) CreateTag(ctx context.Context, name string) (models.Tag, error) {
	tag, err := s.next.CreateTag(ctx, name)
	if err != nil {
		return tag, err
	}

	_, _ = s.store.Increment(ctx, tagsVersionKey)
	return tag, nil
}

func (s *pollService) ListTags(ctx context.Context) ([]models.Tag, error) {
	version := int64(0)
	versionValue, foundVersion, err := cachepkg.GetInt64(ctx, s.store, tagsVersionKey)
	if err == nil && foundVersion {
		version = versionValue
	}

	if err == nil {
		key := tagsCacheKey(version)
		var cached []models.Tag
		found, readErr := cachepkg.GetJSON(ctx, s.store, key, &cached)
		if readErr == nil && found {
			return cached, nil
		}

		items, listErr := s.next.ListTags(ctx)
		if listErr != nil {
			return nil, listErr
		}

		_ = cachepkg.SetJSON(ctx, s.store, key, items, s.cfg.TagsTTL)
		return items, nil
	}

	return s.next.ListTags(ctx)
}

type pollListCacheValue struct {
	Items      []models.Poll `json:"items"`
	NextCursor string        `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

const (
	prefix             = "poll-service"
	pollListVersionKey = prefix + ":poll-list:version"
	tagsVersionKey     = prefix + ":tags-list:version"
)

func pollCacheKey(id string) string {
	return prefix + ":poll:" + strings.TrimSpace(id)
}

func pollListCacheKey(cursor string, limit uint32, tags []string, version int64) string {
	fingerprint := cachepkg.HashParts(
		strings.TrimSpace(cursor),
		strconv.Itoa(normalizedLimit(limit)),
		strings.Join(normalizedTags(tags), ","),
	)

	return fmt.Sprintf("%s:poll-list:%d:%s", prefix, version, fingerprint)
}

func tagsCacheKey(version int64) string {
	return fmt.Sprintf("%s:tags:%d", prefix, version)
}

func normalizedLimit(limit uint32) int {
	v := int(limit)
	if v <= 0 {
		v = 20
	}
	if v > 100 {
		v = 100
	}
	return v
}

func normalizedTags(tags []string) []string {
	out := make([]string, 0, len(tags))
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
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
