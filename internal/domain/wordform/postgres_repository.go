package wordform

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

func (r *PostgresRepository) Insert(ctx context.Context, wordID int64, subject, form string, tense *string) (*WordForm, error) {
	row, err := r.q.InsertWordForm(ctx, sqlc.InsertWordFormParams{
		WordID:  wordID,
		Subject: subject,
		Form:    form,
		Tense:   pgtypeconv.TextFromPtr(tense),
	})
	if err != nil {
		return nil, err
	}
	return &WordForm{
		ID:        row.ID,
		WordID:    row.WordID,
		Subject:   row.Subject,
		Form:      row.Form,
		Tense:     pgtypeconv.PtrFromText(row.Tense),
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, wordFormID, userID int64) (bool, error) {
	rows, err := r.q.DeleteWordForm(ctx, sqlc.DeleteWordFormParams{
		ID:     wordFormID,
		UserID: userID,
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *PostgresRepository) GetRandom(ctx context.Context, languageID int64, count int32) ([]*WordForm, error) {
	rows, err := r.q.GetRandomWordForms(ctx, sqlc.GetRandomWordFormsParams{
		LanguageID: languageID,
		Limit:      count,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*WordForm, 0, len(rows))
	for _, row := range rows {
		out = append(out, &WordForm{
			ID:           row.ID,
			WordID:       row.WordID,
			Subject:      row.Subject,
			Form:         row.Form,
			NativeWord:   row.NativeWord,
			LearningWord: row.LearningWord,
			Tense:        pgtypeconv.PtrFromText(row.Tense),
			CreatedAt:    row.CreatedAt.Time,
		})
	}
	return out, nil
}
func (r *PostgresRepository) List(ctx context.Context, languageID int64, limit, offset int32) ([]*WordForm, error) {
	rows, err := r.q.ListWordForms(ctx, sqlc.ListWordFormsParams{
		LanguageID: languageID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*WordForm, 0, len(rows))
	for _, row := range rows {
		out = append(out, &WordForm{
			ID:        row.ID,
			WordID:    row.WordID,
			Subject:   row.Subject,
			Form:      row.Form,
			Tense:     pgtypeconv.PtrFromText(row.Tense),
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return out, nil
}

func (r *PostgresRepository) Count(ctx context.Context, languageID int64) (int64, error) {
	return r.q.CountWordForms(ctx, languageID)
}
