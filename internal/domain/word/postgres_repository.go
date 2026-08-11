package word

import (
	"context"

	"github.com/tomasswaier/RabbitVocab/internal/db/pgtypeconv"
	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
)

type PostgresRepository struct {
	q *sqlc.Queries
}

func NewPostgresRepository(q *sqlc.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) Insert(ctx context.Context, languageID int64, nativeWord, learningWord string, article *string) (*Word, error) {
	row, err := r.q.InsertWord(ctx, sqlc.InsertWordParams{
		LanguageID:   languageID,
		NativeWord:   nativeWord,
		LearningWord: learningWord,
		Article:      pgtypeconv.TextFromPtr(article),
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *PostgresRepository) GetRandom(ctx context.Context, languageID int64, count int32) ([]*Word, error) {
	rows, err := r.q.GetRandomWords(ctx, sqlc.GetRandomWordsParams{
		LanguageID: languageID,
		Limit:      count,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*Word, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntity(row))
	}
	return out, nil
}

func (r *PostgresRepository) UpdateState(ctx context.Context, id int64, state State) (*Word, error) {
	row, err := r.q.UpdateWordState(ctx, sqlc.UpdateWordStateParams{
		ID:    id,
		State: sqlc.WordState(state),
	})
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func toEntity(row sqlc.Word) *Word {
	return &Word{
		ID:           row.ID,
		LanguageID:   row.LanguageID,
		NativeWord:   row.NativeWord,
		LearningWord: row.LearningWord,
		Article:      pgtypeconv.PtrFromText(row.Article),
		State:        State(row.State),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
