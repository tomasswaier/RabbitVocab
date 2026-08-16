package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
)

var ErrNotFound = errors.New("user not found")

type PostgresRepository struct {
	q *sqlc.Queries
}

func NewPostgresRepository(q *sqlc.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) Create(ctx context.Context, username, passwordHash string) (*User, error) {
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     username,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	row, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toEntity(row), nil
}

func toEntity(row sqlc.User) *User {
	return &User{
		ID:           row.ID,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
	}
}
