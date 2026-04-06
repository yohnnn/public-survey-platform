package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
)

type VoteRepository struct {
	pool *pgxpool.Pool
}

func NewVoteRepository(pool *pgxpool.Pool) *VoteRepository {
	return &VoteRepository{pool: pool}
}

func (r *VoteRepository) ReplaceUserVote(ctx context.Context, userID, pollID string, optionIDs []string, createdAt time.Time) error {
	exec := tx.Executor(ctx, r.pool)

	if _, err := exec.Exec(ctx, `DELETE FROM votes WHERE user_id = $1 AND poll_id = $2`, userID, pollID); err != nil {
		return err
	}

	const insertQuery = `
		INSERT INTO votes (user_id, poll_id, option_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	for _, optionID := range optionIDs {
		if _, err := exec.Exec(ctx, insertQuery, userID, pollID, optionID, createdAt); err != nil {
			return err
		}
	}

	return nil
}

func (r *VoteRepository) DeleteUserVote(ctx context.Context, userID, pollID string) error {
	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, `DELETE FROM votes WHERE user_id = $1 AND poll_id = $2`, userID, pollID)
	return err
}

func (r *VoteRepository) GetUserVote(ctx context.Context, userID, pollID string) ([]string, *time.Time, error) {
	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, `
		SELECT option_id, created_at
		FROM votes
		WHERE user_id = $1 AND poll_id = $2
		ORDER BY option_id ASC
	`, userID, pollID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	optionIDs := make([]string, 0)
	var votedAt *time.Time
	for rows.Next() {
		var optionID string
		var createdAt time.Time
		if scanErr := rows.Scan(&optionID, &createdAt); scanErr != nil {
			return nil, nil, scanErr
		}
		if votedAt == nil {
			v := createdAt.UTC()
			votedAt = &v
		}
		optionIDs = append(optionIDs, optionID)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return optionIDs, votedAt, nil
}
