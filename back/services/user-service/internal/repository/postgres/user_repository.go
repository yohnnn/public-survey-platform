package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
	repo "github.com/yohnnn/public-survey-platform/back/services/user-service/internal/repository"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		INSERT INTO users (id, nickname, email, password_hash, country, gender, birth_year, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, nickname, email, password_hash, country, gender, birth_year, created_at
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(
		ctx,
		query,
		user.ID,
		user.Nickname,
		user.Email,
		user.PasswordHash,
		user.Country,
		user.Gender,
		user.BirthYear,
		user.CreatedAt,
	).Scan(
		&out.ID,
		&out.Nickname,
		&out.Email,
		&out.PasswordHash,
		&out.Country,
		&out.Gender,
		&out.BirthYear,
		&out.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return models.User{}, models.ErrEmailAlreadyExists
		}
		if isUniqueViolation(err, "users_nickname_key") {
			return models.User{}, models.ErrNicknameAlreadyExists
		}
		return models.User{}, err
	}

	return out, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (models.User, error) {
	const query = `
		SELECT id, nickname, email, country, gender, birth_year, created_at
		FROM users
		WHERE id = $1
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(ctx, query, id).Scan(
		&out.ID,
		&out.Nickname,
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
		SELECT id, nickname, email, password_hash, country, gender, birth_year, created_at
		FROM users
		WHERE email = $1
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(ctx, query, email).Scan(
		&out.ID,
		&out.Nickname,
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

func (r *UserRepository) ListSummariesByIDs(ctx context.Context, ids []string) ([]models.UserSummary, error) {
	if len(ids) == 0 {
		return []models.UserSummary{}, nil
	}

	const query = `
		SELECT id, nickname
		FROM users
		WHERE id = ANY($1)
		ORDER BY nickname ASC, id ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.UserSummary, 0, len(ids))
	for rows.Next() {
		var item models.UserSummary
		if scanErr := rows.Scan(&item.ID, &item.Nickname); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func (r *UserRepository) ExistsByNickname(ctx context.Context, nickname string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE nickname = $1)`

	exec := tx.Executor(ctx, r.pool)
	var exists bool
	if err := exec.QueryRow(ctx, query, nickname).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, patch repo.UserUpdatePatch) (models.User, error) {
	const query = `
		UPDATE users
		SET
			email = COALESCE($2, email),
			nickname = COALESCE($3, nickname),
			country = COALESCE($4, country),
			gender = COALESCE($5, gender),
			birth_year = COALESCE($6, birth_year)
		WHERE id = $1
		RETURNING id, nickname, email, country, gender, birth_year, created_at
	`

	exec := tx.Executor(ctx, r.pool)
	var out models.User
	err := exec.QueryRow(
		ctx,
		query,
		id,
		patch.Email,
		patch.Nickname,
		patch.Country,
		patch.Gender,
		patch.BirthYear,
	).Scan(
		&out.ID,
		&out.Nickname,
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
		if isUniqueViolation(err, "users_email_key") {
			return models.User{}, models.ErrEmailAlreadyExists
		}
		if isUniqueViolation(err, "users_nickname_key") {
			return models.User{}, models.ErrNicknameAlreadyExists
		}
		return models.User{}, err
	}

	return out, nil
}

func (r *UserRepository) CountFollowers(ctx context.Context, userID string) (int64, error) {
	const query = `SELECT COUNT(*) FROM user_follows WHERE followee_id = $1`

	exec := tx.Executor(ctx, r.pool)
	var count int64
	if err := exec.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UserRepository) CountFollowing(ctx context.Context, userID string) (int64, error) {
	const query = `SELECT COUNT(*) FROM user_follows WHERE follower_id = $1`

	exec := tx.Executor(ctx, r.pool)
	var count int64
	if err := exec.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UserRepository) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM user_follows WHERE follower_id = $1 AND followee_id = $2)`

	exec := tx.Executor(ctx, r.pool)
	var exists bool
	if err := exec.QueryRow(ctx, query, followerID, followeeID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *UserRepository) ListFollowingIDs(ctx context.Context, userID string) ([]string, error) {
	const query = `
		SELECT followee_id
		FROM user_follows
		WHERE follower_id = $1
		ORDER BY created_at DESC, followee_id ASC
	`

	exec := tx.Executor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var followeeID string
		if scanErr := rows.Scan(&followeeID); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, followeeID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *UserRepository) Follow(ctx context.Context, followerID, followeeID string, createdAt time.Time) error {
	const query = `
		INSERT INTO user_follows (follower_id, followee_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (follower_id, followee_id) DO NOTHING
	`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, followerID, followeeID, createdAt)
	return err
}

func (r *UserRepository) Unfollow(ctx context.Context, followerID, followeeID string) error {
	const query = `DELETE FROM user_follows WHERE follower_id = $1 AND followee_id = $2`

	exec := tx.Executor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, followerID, followeeID)
	return err
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "23505" {
			return false
		}
		if constraint == "" {
			return true
		}
		return pgErr.ConstraintName == constraint
	}
	return false
}
