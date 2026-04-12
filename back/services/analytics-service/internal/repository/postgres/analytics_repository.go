package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

func (r *AnalyticsRepository) IncrementOptionVotes(ctx context.Context, pollID, optionID string, delta int64) error {
	const query = `
		INSERT INTO poll_option_stats (poll_id, option_id, votes_count)
		VALUES ($1, $2, $3)
		ON CONFLICT (poll_id, option_id)
		DO UPDATE SET votes_count = GREATEST(0, poll_option_stats.votes_count + EXCLUDED.votes_count)
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, strings.TrimSpace(pollID), strings.TrimSpace(optionID), delta)
	return err
}

func (r *AnalyticsRepository) IncrementCountryVotes(ctx context.Context, pollID, country string, delta int64) error {
	const query = `
		INSERT INTO poll_country_stats (poll_id, country, votes_count)
		VALUES ($1, $2, $3)
		ON CONFLICT (poll_id, country)
		DO UPDATE SET votes_count = GREATEST(0, poll_country_stats.votes_count + EXCLUDED.votes_count)
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, strings.TrimSpace(pollID), strings.TrimSpace(country), delta)
	return err
}

func (r *AnalyticsRepository) IncrementGenderVotes(ctx context.Context, pollID, gender string, delta int64) error {
	const query = `
		INSERT INTO poll_gender_stats (poll_id, gender, votes_count)
		VALUES ($1, $2, $3)
		ON CONFLICT (poll_id, gender)
		DO UPDATE SET votes_count = GREATEST(0, poll_gender_stats.votes_count + EXCLUDED.votes_count)
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, strings.TrimSpace(pollID), strings.TrimSpace(gender), delta)
	return err
}

func (r *AnalyticsRepository) IncrementAgeVotes(ctx context.Context, pollID, ageRange string, delta int64) error {
	const query = `
		INSERT INTO poll_age_stats (poll_id, age_range, votes_count)
		VALUES ($1, $2, $3)
		ON CONFLICT (poll_id, age_range)
		DO UPDATE SET votes_count = GREATEST(0, poll_age_stats.votes_count + EXCLUDED.votes_count)
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, strings.TrimSpace(pollID), strings.TrimSpace(ageRange), delta)
	return err
}

func (r *AnalyticsRepository) MarkEventProcessed(ctx context.Context, eventID, topic string) (bool, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return true, nil
	}

	const query = `
		INSERT INTO processed_events (event_id, topic)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
	`

	exec := tx.Executor(ctx, r.pool)
	cmd, err := exec.Exec(ctx, query, eventID, strings.TrimSpace(topic))
	if err != nil {
		return false, err
	}

	return cmd.RowsAffected() == 1, nil
}

func (r *AnalyticsRepository) GetPollAnalytics(ctx context.Context, pollID string) (models.PollAnalytics, error) {
	pollID = strings.TrimSpace(pollID)
	result := models.PollAnalytics{PollID: pollID}

	options, err := r.getOptionStats(ctx, pollID)
	if err != nil {
		return models.PollAnalytics{}, err
	}
	countries, err := r.GetCountryStats(ctx, pollID)
	if err != nil {
		return models.PollAnalytics{}, err
	}
	gender, err := r.GetGenderStats(ctx, pollID)
	if err != nil {
		return models.PollAnalytics{}, err
	}
	age, err := r.GetAgeStats(ctx, pollID)
	if err != nil {
		return models.PollAnalytics{}, err
	}

	var total int64
	for _, item := range options {
		total += item.Votes
	}

	result.TotalVotes = total
	result.Options = options
	result.Countries = countries
	result.Gender = gender
	result.Age = age

	return result, nil
}

func (r *AnalyticsRepository) GetCountryStats(ctx context.Context, pollID string) ([]models.CountryStat, error) {
	const query = `
		SELECT country, votes_count
		FROM poll_country_stats
		WHERE poll_id = $1
		ORDER BY votes_count DESC, country ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, strings.TrimSpace(pollID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.CountryStat, 0)
	for rows.Next() {
		var item models.CountryStat
		if scanErr := rows.Scan(&item.Country, &item.Votes); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *AnalyticsRepository) GetGenderStats(ctx context.Context, pollID string) ([]models.GenderStat, error) {
	const query = `
		SELECT gender, votes_count
		FROM poll_gender_stats
		WHERE poll_id = $1
		ORDER BY votes_count DESC, gender ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, strings.TrimSpace(pollID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.GenderStat, 0)
	for rows.Next() {
		var item models.GenderStat
		if scanErr := rows.Scan(&item.Gender, &item.Votes); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *AnalyticsRepository) GetAgeStats(ctx context.Context, pollID string) ([]models.AgeStat, error) {
	const query = `
		SELECT age_range, votes_count
		FROM poll_age_stats
		WHERE poll_id = $1
		ORDER BY votes_count DESC, age_range ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, strings.TrimSpace(pollID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.AgeStat, 0)
	for rows.Next() {
		var item models.AgeStat
		if scanErr := rows.Scan(&item.AgeRange, &item.Votes); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *AnalyticsRepository) getOptionStats(ctx context.Context, pollID string) ([]models.OptionStat, error) {
	const query = `
		SELECT option_id, votes_count
		FROM poll_option_stats
		WHERE poll_id = $1
		ORDER BY votes_count DESC, option_id ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, strings.TrimSpace(pollID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.OptionStat, 0)
	for rows.Next() {
		var item models.OptionStat
		if scanErr := rows.Scan(&item.OptionID, &item.Votes); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
