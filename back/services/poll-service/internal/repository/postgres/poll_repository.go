package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository"
)

type PollRepository struct {
	pool *pgxpool.Pool
}

func NewPollRepository(pool *pgxpool.Pool) *PollRepository {
	return &PollRepository{pool: pool}
}

func (r *PollRepository) Create(ctx context.Context, poll models.Poll, options []models.PollOption, tagIDs []string) error {
	const insertPollQuery = `
		INSERT INTO polls (id, creator_id, question, type, is_anonymous, ends_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	exec := tx.Executor(ctx, r.pool)
	if _, err := exec.Exec(ctx, insertPollQuery,
		poll.ID,
		poll.CreatorID,
		poll.Question,
		poll.Type,
		poll.IsAnonymous,
		poll.EndsAt,
		poll.CreatedAt,
	); err != nil {
		return err
	}

	const insertOptionQuery = `
		INSERT INTO poll_options (id, poll_id, text, votes_count, position)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, option := range options {
		if _, err := exec.Exec(ctx, insertOptionQuery, option.ID, poll.ID, option.Text, option.VotesCount, option.Position); err != nil {
			return err
		}
	}

	if len(tagIDs) > 0 {
		const insertPollTagQuery = `
			INSERT INTO poll_tags (poll_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`
		for _, tagID := range tagIDs {
			if _, err := exec.Exec(ctx, insertPollTagQuery, poll.ID, tagID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *PollRepository) GetByID(ctx context.Context, id string) (models.Poll, error) {
	const query = `
		SELECT p.id, p.creator_id, p.question, p.type, p.is_anonymous, p.ends_at, p.created_at,
		       COALESCE(SUM(po.votes_count), 0) AS total_votes
		FROM polls p
		LEFT JOIN poll_options po ON po.poll_id = p.id
		WHERE p.id = $1
		GROUP BY p.id
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.Poll
	var endsAt *time.Time
	err := exec.QueryRow(ctx, query, id).Scan(
		&out.ID,
		&out.CreatorID,
		&out.Question,
		&out.Type,
		&out.IsAnonymous,
		&endsAt,
		&out.CreatedAt,
		&out.TotalVotes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Poll{}, models.ErrPollNotFound
		}
		return models.Poll{}, err
	}
	out.EndsAt = endsAt

	return out, nil
}

func (r *PollRepository) List(ctx context.Context, filter repository.PollListFilter) ([]models.Poll, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	base := `
		SELECT p.id, p.creator_id, p.question, p.type, p.is_anonymous, p.ends_at, p.created_at,
		       COALESCE(SUM(po.votes_count), 0) AS total_votes
		FROM polls p
		LEFT JOIN poll_options po ON po.poll_id = p.id
	`

	where := make([]string, 0, 2)
	args := make([]any, 0, 8)
	argPos := 1

	if filter.CursorCreatedAt != nil && strings.TrimSpace(filter.CursorID) != "" {
		where = append(where, fmt.Sprintf("(p.created_at < $%d OR (p.created_at = $%d AND p.id < $%d))", argPos, argPos, argPos+1))
		args = append(args, *filter.CursorCreatedAt, filter.CursorID)
		argPos += 2
	}

	if len(filter.Tags) > 0 {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM poll_tags ptf
			JOIN tags tf ON tf.id = ptf.tag_id
			WHERE ptf.poll_id = p.id AND tf.name = ANY($%d)
		)`, argPos))
		args = append(args, filter.Tags)
		argPos++
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " GROUP BY p.id ORDER BY p.created_at DESC, p.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, filter.Limit)

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Poll, 0, filter.Limit)
	for rows.Next() {
		var item models.Poll
		var endsAt *time.Time
		if scanErr := rows.Scan(
			&item.ID,
			&item.CreatorID,
			&item.Question,
			&item.Type,
			&item.IsAnonymous,
			&endsAt,
			&item.CreatedAt,
			&item.TotalVotes,
		); scanErr != nil {
			return nil, scanErr
		}
		item.EndsAt = endsAt
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *PollRepository) UpdateByIDAndCreator(ctx context.Context, pollID, creatorID string, patch repository.PollPatch) error {
	sets := make([]string, 0, 3)
	args := make([]any, 0, 5)
	argPos := 1

	if patch.Question != nil {
		sets = append(sets, fmt.Sprintf("question = $%d", argPos))
		args = append(args, *patch.Question)
		argPos++
	}
	if patch.IsAnonymous != nil {
		sets = append(sets, fmt.Sprintf("is_anonymous = $%d", argPos))
		args = append(args, *patch.IsAnonymous)
		argPos++
	}
	if patch.EndsAt != nil {
		sets = append(sets, fmt.Sprintf("ends_at = $%d", argPos))
		args = append(args, *patch.EndsAt)
		argPos++
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE polls
		SET %s
		WHERE id = $%d AND creator_id = $%d
	`, strings.Join(sets, ", "), argPos, argPos+1)
	args = append(args, pollID, creatorID)

	exec := tx.Executor(ctx, r.pool)
	cmd, err := exec.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrForbidden
	}

	return nil
}

func (r *PollRepository) DeleteByIDAndCreator(ctx context.Context, pollID, creatorID string) error {
	const query = `DELETE FROM polls WHERE id = $1 AND creator_id = $2`

	exec := tx.Executor(ctx, r.pool)
	cmd, err := exec.Exec(ctx, query, pollID, creatorID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrForbidden
	}

	return nil
}

func (r *PollRepository) ReplaceTags(ctx context.Context, pollID string, tagIDs []string) error {
	exec := tx.Executor(ctx, r.pool)

	if _, err := exec.Exec(ctx, `DELETE FROM poll_tags WHERE poll_id = $1`, pollID); err != nil {
		return err
	}

	if len(tagIDs) == 0 {
		return nil
	}

	const query = `
		INSERT INTO poll_tags (poll_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	for _, tagID := range tagIDs {
		if _, err := exec.Exec(ctx, query, pollID, tagID); err != nil {
			return err
		}
	}

	return nil
}

func (r *PollRepository) GetOptionsByPollIDs(ctx context.Context, pollIDs []string) (map[string][]models.PollOption, error) {
	if len(pollIDs) == 0 {
		return map[string][]models.PollOption{}, nil
	}

	const query = `
		SELECT id, poll_id, text, votes_count, position
		FROM poll_options
		WHERE poll_id = ANY($1)
		ORDER BY poll_id, position ASC, id ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, pollIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]models.PollOption, len(pollIDs))
	for _, id := range pollIDs {
		out[id] = []models.PollOption{}
	}
	for rows.Next() {
		var option models.PollOption
		var pollID string
		if scanErr := rows.Scan(&option.ID, &pollID, &option.Text, &option.VotesCount, &option.Position); scanErr != nil {
			return nil, scanErr
		}
		out[pollID] = append(out[pollID], option)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PollRepository) GetTagsByPollIDs(ctx context.Context, pollIDs []string) (map[string][]string, error) {
	if len(pollIDs) == 0 {
		return map[string][]string{}, nil
	}

	const query = `
		SELECT pt.poll_id, t.name
		FROM poll_tags pt
		JOIN tags t ON t.id = pt.tag_id
		WHERE pt.poll_id = ANY($1)
		ORDER BY pt.poll_id, t.name ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, pollIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]string, len(pollIDs))
	for _, id := range pollIDs {
		out[id] = []string{}
	}
	for rows.Next() {
		var pollID, tagName string
		if scanErr := rows.Scan(&pollID, &tagName); scanErr != nil {
			return nil, scanErr
		}
		out[pollID] = append(out[pollID], tagName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
