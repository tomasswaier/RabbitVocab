package user

import "context"

type Repository interface {
	Create(ctx context.Context, username, passwordHash, apiKeyHash string) (*User, error)
	GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	UpdateAPIKeyHash(ctx context.Context, id int64, newAPIKeyHash string) (*User, error)
}
