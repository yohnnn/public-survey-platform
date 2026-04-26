package servicecache

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	cachepkg "github.com/yohnnn/public-survey-platform/back/pkg/cache"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
)

func TestGetPollAnalyticsSkipsCachingEmptySnapshot(t *testing.T) {
	t.Parallel()

	next := &stubAnalyticsService{
		pollResults: []models.PollAnalytics{
			{PollID: "poll-1", TotalVotes: 0},
			{PollID: "poll-1", TotalVotes: 1, Options: []models.OptionStat{{OptionID: "o1", Votes: 1}}},
		},
	}
	store := newInMemoryStore()
	svc := NewAnalyticsService(next, store, Config{TTL: time.Minute})

	first, err := svc.GetPollAnalytics(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	if first.TotalVotes != 0 {
		t.Fatalf("first call totalVotes=%d, want 0", first.TotalVotes)
	}

	second, err := svc.GetPollAnalytics(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if second.TotalVotes != 1 {
		t.Fatalf("second call totalVotes=%d, want 1", second.TotalVotes)
	}

	third, err := svc.GetPollAnalytics(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("third call returned error: %v", err)
	}
	if third.TotalVotes != 1 {
		t.Fatalf("third call totalVotes=%d, want 1", third.TotalVotes)
	}

	if next.pollCalls != 2 {
		t.Fatalf("unexpected upstream call count: got=%d want=2", next.pollCalls)
	}
}

func TestGetCountryStatsSkipsCachingEmptySlice(t *testing.T) {
	t.Parallel()

	next := &stubAnalyticsService{
		countryResults: [][]models.CountryStat{
			{},
			{{Country: "RU", Votes: 1}},
		},
	}
	store := newInMemoryStore()
	svc := NewAnalyticsService(next, store, Config{TTL: time.Minute})

	first, err := svc.GetCountryStats(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	if len(first) != 0 {
		t.Fatalf("first call len=%d, want 0", len(first))
	}

	second, err := svc.GetCountryStats(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	expected := []models.CountryStat{{Country: "RU", Votes: 1}}
	if !reflect.DeepEqual(second, expected) {
		t.Fatalf("second call mismatch: got=%#v want=%#v", second, expected)
	}

	third, err := svc.GetCountryStats(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("third call returned error: %v", err)
	}
	if !reflect.DeepEqual(third, expected) {
		t.Fatalf("third call mismatch: got=%#v want=%#v", third, expected)
	}

	if next.countryCalls != 2 {
		t.Fatalf("unexpected upstream call count: got=%d want=2", next.countryCalls)
	}
}

type stubAnalyticsService struct {
	pollResults    []models.PollAnalytics
	countryResults [][]models.CountryStat
	genderResults  [][]models.GenderStat
	ageResults     [][]models.AgeStat

	pollCalls    int
	countryCalls int
	genderCalls  int
	ageCalls     int
}

func (s *stubAnalyticsService) GetPollAnalytics(_ context.Context, _ string) (models.PollAnalytics, error) {
	s.pollCalls++
	if len(s.pollResults) == 0 {
		return models.PollAnalytics{}, nil
	}
	idx := s.pollCalls - 1
	if idx >= len(s.pollResults) {
		idx = len(s.pollResults) - 1
	}
	return s.pollResults[idx], nil
}

func (s *stubAnalyticsService) GetCountryStats(_ context.Context, _ string) ([]models.CountryStat, error) {
	s.countryCalls++
	if len(s.countryResults) == 0 {
		return nil, nil
	}
	idx := s.countryCalls - 1
	if idx >= len(s.countryResults) {
		idx = len(s.countryResults) - 1
	}
	return s.countryResults[idx], nil
}

func (s *stubAnalyticsService) GetGenderStats(_ context.Context, _ string) ([]models.GenderStat, error) {
	s.genderCalls++
	if len(s.genderResults) == 0 {
		return nil, nil
	}
	idx := s.genderCalls - 1
	if idx >= len(s.genderResults) {
		idx = len(s.genderResults) - 1
	}
	return s.genderResults[idx], nil
}

func (s *stubAnalyticsService) GetAgeStats(_ context.Context, _ string) ([]models.AgeStat, error) {
	s.ageCalls++
	if len(s.ageResults) == 0 {
		return nil, nil
	}
	idx := s.ageCalls - 1
	if idx >= len(s.ageResults) {
		idx = len(s.ageResults) - 1
	}
	return s.ageResults[idx], nil
}

type inMemoryStore struct {
	values map[string][]byte
}

func newInMemoryStore() *inMemoryStore {
	return &inMemoryStore{values: map[string][]byte{}}
}

func (s *inMemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, cachepkg.ErrNotFound
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out, nil
}

func (s *inMemoryStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	out := make([]byte, len(value))
	copy(out, value)
	s.values[key] = out
	return nil
}

func (s *inMemoryStore) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(s.values, key)
	}
	return nil
}

func (s *inMemoryStore) Increment(_ context.Context, key string) (int64, error) {
	current := int64(0)
	if raw, ok := s.values[key]; ok {
		parsed, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil {
			return 0, err
		}
		current = parsed
	}
	current++
	s.values[key] = []byte(strconv.FormatInt(current, 10))
	return current, nil
}

func (s *inMemoryStore) Close() error { return nil }
