package session

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tomasswaier/RabbitVocab/internal/db/pgtypeconv"
	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
)

var ErrNotFound = errors.New("session not found or expired")

type PostgresRepository struct {
	q *sqlc.Queries
}

func NewPostgresRepository(q *sqlc.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*Session, error) {
	row, err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: pgtypeconv.TimestamptzFromTime(expiresAt),
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) GetValidByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	row, err := r.q.GetValidSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	return r.q.DeleteSessionByTokenHash(ctx, tokenHash)
}

func toEntity(row sqlc.Session) *Session {
	return &Session{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time, // adjust if sqlc generates plain time.Time instead of pgtype.Timestamptz
		CreatedAt: row.CreatedAt.Time,
	}
}
