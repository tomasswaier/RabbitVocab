package session

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*Session, error)
	GetValidByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}
