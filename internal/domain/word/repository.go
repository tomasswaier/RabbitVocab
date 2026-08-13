package word

import "context"

type Repository interface {
	Insert(ctx context.Context, languageID int64, nativeWord, learningWord string, article *string) (*Word, error)
	GetRandom(ctx context.Context, languageID int64, count int32) ([]*Word, error)
	UpdateState(ctx context.Context, id int64, state State) (*Word, error)
	Search(ctx context.Context, languageID int64, query string) ([]*Word, error)
	Delete(ctx context.Context, wordID, userID int64) (bool, error)
	List(ctx context.Context, languageID int64, limit, offset int32) ([]*Word, error)
	Count(ctx context.Context, languageID int64) (int64, error)
}
