package user

import "context"

type Repository interface {
	Create(ctx context.Context, username, passwordHash string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
}
