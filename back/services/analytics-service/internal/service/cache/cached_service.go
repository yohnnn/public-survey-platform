package servicecache

import (
	"context"
	"fmt"
	"strings"
	"time"

	cachepkg "github.com/yohnnn/public-survey-platform/back/pkg/cache"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/service"
)

type Config struct {
	TTL time.Duration
}

func DefaultConfig() Config {
	return Config{TTL: 60 * time.Second}
}

type analyticsService struct {
	next  service.AnalyticsService
	store cachepkg.Store
	ttl   time.Duration
}

func NewAnalyticsService(next service.AnalyticsService, store cachepkg.Store, cfg Config) service.AnalyticsService {
	if next == nil || store == nil {
		return next
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	return &analyticsService{next: next, store: store, ttl: ttl}
}

func (s *analyticsService) GetPollAnalytics(ctx context.Context, pollID string, from, to *time.Time, interval string) (models.PollAnalytics, error) {
	key := cacheKey("poll", pollID, from, to, interval)
	var cached models.PollAnalytics

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	result, err := s.next.GetPollAnalytics(ctx, pollID, from, to, interval)
	if err != nil {
		return result, err
	}

	if shouldCachePollAnalytics(result) {
		_ = cachepkg.SetJSON(ctx, s.store, key, result, s.ttl)
	}
	return result, nil
}

func (s *analyticsService) GetCountryStats(ctx context.Context, pollID string, from, to *time.Time, interval string) ([]models.CountryStat, error) {
	key := cacheKey("country", pollID, from, to, interval)
	var cached []models.CountryStat

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	result, err := s.next.GetCountryStats(ctx, pollID, from, to, interval)
	if err != nil {
		return result, err
	}

	if len(result) > 0 {
		_ = cachepkg.SetJSON(ctx, s.store, key, result, s.ttl)
	}
	return result, nil
}

func (s *analyticsService) GetGenderStats(ctx context.Context, pollID string, from, to *time.Time, interval string) ([]models.GenderStat, error) {
	key := cacheKey("gender", pollID, from, to, interval)
	var cached []models.GenderStat

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	result, err := s.next.GetGenderStats(ctx, pollID, from, to, interval)
	if err != nil {
		return result, err
	}

	if len(result) > 0 {
		_ = cachepkg.SetJSON(ctx, s.store, key, result, s.ttl)
	}
	return result, nil
}

func (s *analyticsService) GetAgeStats(ctx context.Context, pollID string, from, to *time.Time, interval string) ([]models.AgeStat, error) {
	key := cacheKey("age", pollID, from, to, interval)
	var cached []models.AgeStat

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	result, err := s.next.GetAgeStats(ctx, pollID, from, to, interval)
	if err != nil {
		return result, err
	}

	if len(result) > 0 {
		_ = cachepkg.SetJSON(ctx, s.store, key, result, s.ttl)
	}
	return result, nil
}

func shouldCachePollAnalytics(result models.PollAnalytics) bool {
	if result.TotalVotes > 0 {
		return true
	}

	return len(result.Options) > 0 || len(result.Countries) > 0 || len(result.Gender) > 0 || len(result.Age) > 0
}

func cacheKey(kind, pollID string, from, to *time.Time, interval string) string {
	return fmt.Sprintf(
		"analytics-service:%s:%s",
		kind,
		cachepkg.HashParts(strings.TrimSpace(pollID), formatTime(from), formatTime(to), strings.TrimSpace(interval)),
	)
}

func formatTime(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.UTC().Format(time.RFC3339Nano)
}
