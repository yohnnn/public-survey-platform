package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, country, gender, birth_year, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password_hash, country, gender, birth_year, created_at
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Country,
		user.Gender,
		user.BirthYear,
		user.CreatedAt,
	).Scan(
		&out.ID,
		&out.Email,
		&out.PasswordHash,
		&out.Country,
		&out.Gender,
		&out.BirthYear,
		&out.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, models.ErrEmailAlreadyExists
		}
		return models.User{}, err
	}

	return out, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (models.User, error) {
	const query = `
		SELECT id, email, country, gender, birth_year, created_at
		FROM users
		WHERE id = $1
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(ctx, query, id).Scan(
		&out.ID,
		&out.Email,
		&out.Country,
		&out.Gender,
		&out.BirthYear,
		&out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, models.ErrUserNotFound
		}
		return models.User{}, err
	}

	return out, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, country, gender, birth_year, created_at
		FROM users
		WHERE email = $1
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(ctx, query, email).Scan(
		&out.ID,
		&out.Email,
		&out.PasswordHash,
		&out.Country,
		&out.Gender,
		&out.BirthYear,
		&out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, models.ErrUserNotFound
		}
		return models.User{}, err
	}

	return out, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	exec := tx.Executor(ctx, r.pool)
	var exists bool
	if err := exec.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
