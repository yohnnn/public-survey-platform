package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
)

type FeedRepository struct {
	pool *pgxpool.Pool
}

func NewFeedRepository(pool *pgxpool.Pool) *FeedRepository {
	return &FeedRepository{pool: pool}
}

func (r *FeedRepository) CreateFeedItem(ctx context.Context, item models.FeedItem, options []models.FeedItemOption, tags []string) error {
	const insertFeedItemQuery = `
		INSERT INTO feed_items (id, creator_id, question, total_votes, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	exec := tx.Executor(ctx, r.pool)
	if _, err := exec.Exec(ctx, insertFeedItemQuery,
		item.ID,
		item.CreatorID,
		item.Question,
		item.TotalVotes,
		item.CreatedAt,
	); err != nil {
		return err
	}

	const insertOptionQuery = `
		INSERT INTO feed_item_options (id, feed_item_id, text, votes_count, position)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, option := range options {
		if _, err := exec.Exec(ctx, insertOptionQuery, option.ID, item.ID, option.Text, option.VotesCount, option.Position); err != nil {
			return err
		}
	}

	if len(tags) > 0 {
		const insertTagQuery = `
			INSERT INTO feed_item_tags (feed_item_id, tag)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`
		for _, tag := range tags {
			if _, err := exec.Exec(ctx, insertTagQuery, item.ID, tag); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *FeedRepository) IncrementOptionVotes(ctx context.Context, optionID string, delta int64) error {
	const query = `
		UPDATE feed_item_options
		SET votes_count = votes_count + $1
		WHERE id = $2
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, delta, optionID)
	return err
}

func (r *FeedRepository) UpdateTotalVotes(ctx context.Context, feedItemID string, delta int64) error {
	const query = `
		UPDATE feed_items
		SET total_votes = total_votes + $1
		WHERE id = $2
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, delta, feedItemID)
	return err
}

func (r *FeedRepository) GetFeed(ctx context.Context, filter repository.FeedListFilter) ([]models.FeedItem, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	base := `
		SELECT id, creator_id, question, total_votes, created_at
		FROM feed_items
	`

	where := make([]string, 0, 3)
	args := make([]any, 0, 8)
	argPos := 1

	if filter.CursorCreatedAt != nil && strings.TrimSpace(filter.CursorID) != "" {
		where = append(where, fmt.Sprintf("(created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1))
		args = append(args, *filter.CursorCreatedAt, filter.CursorID)
		argPos += 2
	}

	if len(filter.Tags) > 0 {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM feed_item_tags fit
			WHERE fit.feed_item_id = feed_items.id AND fit.tag = ANY($%d)
		)`, argPos))
		args = append(args, filter.Tags)
		argPos++
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, filter.Limit)

	return r.queryItems(ctx, query, args)
}

func (r *FeedRepository) GetTrending(ctx context.Context, filter repository.FeedListFilter) ([]models.FeedItem, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	base := `
		SELECT id, creator_id, question, total_votes, created_at
		FROM feed_items
	`

	where := make([]string, 0, 2)
	args := make([]any, 0, 8)
	argPos := 1

	if filter.CursorVotes != nil && strings.TrimSpace(filter.CursorID) != "" {
		where = append(where, fmt.Sprintf("(total_votes < $%d OR (total_votes = $%d AND id < $%d))", argPos, argPos, argPos+1))
		args = append(args, *filter.CursorVotes, filter.CursorID)
		argPos += 2
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY total_votes DESC, id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, filter.Limit)

	return r.queryItems(ctx, query, args)
}

func (r *FeedRepository) GetUserPolls(ctx context.Context, filter repository.FeedListFilter) ([]models.FeedItem, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	base := `
		SELECT id, creator_id, question, total_votes, created_at
		FROM feed_items
	`

	where := make([]string, 0, 3)
	args := make([]any, 0, 8)
	argPos := 1

	where = append(where, fmt.Sprintf("creator_id = $%d", argPos))
	args = append(args, filter.CreatorID)
	argPos++

	if filter.CursorCreatedAt != nil && strings.TrimSpace(filter.CursorID) != "" {
		where = append(where, fmt.Sprintf("(created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1))
		args = append(args, *filter.CursorCreatedAt, filter.CursorID)
		argPos += 2
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, filter.Limit)

	return r.queryItems(ctx, query, args)
}

func (r *FeedRepository) queryItems(ctx context.Context, query string, args []any) ([]models.FeedItem, error) {
	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.FeedItem, 0)
	for rows.Next() {
		var item models.FeedItem
		if scanErr := rows.Scan(
			&item.ID,
			&item.CreatorID,
			&item.Question,
			&item.TotalVotes,
			&item.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *FeedRepository) GetOptionsByFeedItemIDs(ctx context.Context, feedItemIDs []string) (map[string][]models.FeedItemOption, error) {
	if len(feedItemIDs) == 0 {
		return map[string][]models.FeedItemOption{}, nil
	}

	const query = `
		SELECT id, feed_item_id, text, votes_count, position
		FROM feed_item_options
		WHERE feed_item_id = ANY($1)
		ORDER BY feed_item_id, position ASC, id ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, feedItemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]models.FeedItemOption, len(feedItemIDs))
	for _, id := range feedItemIDs {
		out[id] = []models.FeedItemOption{}
	}
	for rows.Next() {
		var option models.FeedItemOption
		var feedItemID string
		if scanErr := rows.Scan(&option.ID, &feedItemID, &option.Text, &option.VotesCount, &option.Position); scanErr != nil {
			return nil, scanErr
		}
		out[feedItemID] = append(out[feedItemID], option)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *FeedRepository) GetTagsByFeedItemIDs(ctx context.Context, feedItemIDs []string) (map[string][]string, error) {
	if len(feedItemIDs) == 0 {
		return map[string][]string{}, nil
	}

	const query = `
		SELECT feed_item_id, tag
		FROM feed_item_tags
		WHERE feed_item_id = ANY($1)
		ORDER BY feed_item_id, tag ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, feedItemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]string, len(feedItemIDs))
	for _, id := range feedItemIDs {
		out[id] = []string{}
	}
	for rows.Next() {
		var feedItemID, tag string
		if scanErr := rows.Scan(&feedItemID, &tag); scanErr != nil {
			return nil, scanErr
		}
		out[feedItemID] = append(out[feedItemID], tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

