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

func (r *PostgresRepository) Create(ctx context.Context, username, passwordHash, apiKeyHash string) (*User, error) {
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     username,
		PasswordHash: passwordHash,
		ApiKeyHash:   apiKeyHash,
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*User, error) {
	row, err := r.q.GetUserByAPIKeyHash(ctx, apiKeyHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) UpdateAPIKeyHash(ctx context.Context, id int64, newAPIKeyHash string) (*User, error) {
	row, err := r.q.UpdateUserAPIKeyHash(ctx, sqlc.UpdateUserAPIKeyHashParams{
		ID:         id,
		ApiKeyHash: newAPIKeyHash,
	})
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
		APIKeyHash:   row.ApiKeyHash,
		CreatedAt:    row.CreatedAt.Time,
	}
}
