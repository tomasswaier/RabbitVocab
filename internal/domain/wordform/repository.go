package wordform

import "context"

type Repository interface {
	Insert(ctx context.Context, wordID int64, subject, form string, tense *string) (*WordForm, error)
	GetRandom(ctx context.Context, languageID int64, count int32) ([]*WordForm, error)
	Delete(ctx context.Context, wordFormID, userID int64) (bool, error)
	List(ctx context.Context, languageID int64, limit, offset int32) ([]*WordForm, error)
	Count(ctx context.Context, languageID int64) (int64, error)
}
