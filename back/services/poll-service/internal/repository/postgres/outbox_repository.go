package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Add(ctx context.Context, event outbox.Event) error {
	const query = `
		INSERT INTO outbox_events (id, topic, event_key, payload)
		VALUES ($1, $2, $3, $4)
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, event.ID, event.Topic, event.Key, []byte(event.Payload))
	return err
}

func (r *OutboxRepository) ListUnpublished(ctx context.Context, limit int) ([]outbox.Event, error) {
	if limit <= 0 {
		limit = 50
	}

	const query = `
		SELECT id, topic, event_key, payload
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]outbox.Event, 0, limit)
	for rows.Next() {
		var item outbox.Event
		var payload []byte
		if scanErr := rows.Scan(&item.ID, &item.Topic, &item.Key, &payload); scanErr != nil {
			return nil, scanErr
		}
		item.Payload = payload
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	const query = `
		UPDATE outbox_events
		SET published_at = $2, last_error = NULL
		WHERE id = $1
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, strings.TrimSpace(eventID), publishedAt)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID, reason string) error {
	const query = `
		UPDATE outbox_events
		SET last_error = $2, attempts = attempts + 1
		WHERE id = $1
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, strings.TrimSpace(eventID), strings.TrimSpace(reason))
	return err
}
