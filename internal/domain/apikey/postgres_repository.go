package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tomasswaier/RabbitVocab/internal/db/pgtypeconv"
	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
)

var ErrNotFound = errors.New("api key not found")

type PostgresRepository struct {
	q *sqlc.Queries
}

func NewPostgresRepository(q *sqlc.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) Create(ctx context.Context, userID int64, keyHash string, label, clientID *string, expiresAt *time.Time) (*APIKey, error) {
	row, err := r.q.CreateAPIKey(ctx, sqlc.CreateAPIKeyParams{
		UserID:    userID,
		KeyHash:   keyHash,
		Label:     pgtypeconv.TextFromPtr(label),
		ClientID:  pgtypeconv.TextFromPtr(clientID),
		ExpiresAt: pgtypeconv.TimestamptzFromTimePtr(expiresAt),
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) GetUserByHash(ctx context.Context, keyHash string) (*user.User, error) {
	row, err := r.q.GetUserByAPIKeyHash(ctx, keyHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user.User{
		ID:           row.ID,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID int64) ([]*APIKey, error) {
	rows, err := r.q.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*APIKey, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntity(row))
	}
	return out, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	rows, err := r.q.DeleteAPIKey(ctx, sqlc.DeleteAPIKeyParams{ID: id, UserID: userID})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func toEntity(row sqlc.ApiKey) *APIKey {
	return &APIKey{
		ID:        row.ID,
		UserID:    row.UserID,
		KeyHash:   row.KeyHash,
		Label:     pgtypeconv.PtrFromText(row.Label),
		ClientID:  pgtypeconv.PtrFromText(row.ClientID),
		ExpiresAt: pgtypeconv.PtrFromTimestamptz(row.ExpiresAt),
		CreatedAt: row.CreatedAt.Time,
	}
}
