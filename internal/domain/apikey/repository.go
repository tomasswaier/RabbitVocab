package apikey

import (
	"context"
	"time"

	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
)

type Repository interface {
	Create(ctx context.Context, userID int64, keyHash string, label, clientID *string, expiresAt *time.Time) (*APIKey, error)
	GetUserByHash(ctx context.Context, keyHash string) (*user.User, error)
	ListByUser(ctx context.Context, userID int64) ([]*APIKey, error)
	Delete(ctx context.Context, id, userID int64) (bool, error)
}
