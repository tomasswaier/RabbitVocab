package language

import "context"

type Repository interface {
	Create(ctx context.Context, userID int64, name string) (*Language, error)
	ListByUser(ctx context.Context, userID int64) ([]*Language, error)
	CountByUser(ctx context.Context, userID int64) (int64, error)
}
