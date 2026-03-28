package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
)

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

func (r *TagRepository) Create(ctx context.Context, tag models.Tag) (models.Tag, error) {
	const query = `
		INSERT INTO tags (id, name, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, name, created_at
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.Tag
	err := exec.QueryRow(ctx, query, tag.ID, tag.Name, tag.CreatedAt).Scan(&out.ID, &out.Name, &out.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return models.Tag{}, models.ErrTagAlreadyExist
		}
		return models.Tag{}, err
	}

	return out, nil
}

func (r *TagRepository) List(ctx context.Context) ([]models.Tag, error) {
	const query = `
		SELECT id, name, created_at
		FROM tags
		ORDER BY name ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Tag, 0)
	for rows.Next() {
		var item models.Tag
		if scanErr := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *TagRepository) EnsureByNames(ctx context.Context, names []string) ([]models.Tag, error) {
	normalized := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		v := strings.TrimSpace(strings.ToLower(name))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		normalized = append(normalized, v)
	}

	if len(normalized) == 0 {
		return []models.Tag{}, nil
	}

	exec := tx.Executor(ctx, r.pool)
	for _, name := range normalized {
		_, err := exec.Exec(ctx, `
			INSERT INTO tags (id, name, created_at)
			VALUES (md5(random()::text || clock_timestamp()::text), $1, $2)
			ON CONFLICT (name) DO NOTHING
		`, name, time.Now().UTC())
		if err != nil {
			return nil, err
		}
	}

	rows, err := exec.Query(ctx, `
		SELECT id, name, created_at
		FROM tags
		WHERE name = ANY($1)
		ORDER BY name ASC
	`, normalized)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Tag, 0, len(normalized))
	for rows.Next() {
		var item models.Tag
		if scanErr := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
