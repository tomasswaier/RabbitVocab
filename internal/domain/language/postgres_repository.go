package language

import (
	"context"

	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
)

type PostgresRepository struct {
	q *sqlc.Queries
}

func NewPostgresRepository(q *sqlc.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) Create(ctx context.Context, userID int64, name string) (*Language, error) {
	row, err := r.q.CreateLanguage(ctx, sqlc.CreateLanguageParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID int64) ([]*Language, error) {
	rows, err := r.q.ListLanguagesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*Language, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntity(row))
	}
	return out, nil
}

func (r *PostgresRepository) CountByUser(ctx context.Context, userID int64) (int64, error) {
	return r.q.CountLanguagesByUser(ctx, userID)
}

func toEntity(row sqlc.Language) *Language {
	return &Language{
		ID:        row.ID,
		UserID:    row.UserID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}
}
