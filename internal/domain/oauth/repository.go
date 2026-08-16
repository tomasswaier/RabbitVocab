package oauth

import (
	"context"
	"time"
)

type Repository interface {
	CreateClient(ctx context.Context, clientID string, clientName *string, redirectURIs []string) (*Client, error)
	GetClient(ctx context.Context, clientID string) (*Client, error)

	CreateAuthorizationCode(ctx context.Context, codeHash, clientID string, userID int64, redirectURI, codeChallenge, codeChallengeMethod string, expiresAt time.Time) (*AuthorizationCode, error)
	ConsumeAuthorizationCode(ctx context.Context, codeHash string) (*AuthorizationCode, error)

	CreateRefreshToken(ctx context.Context, tokenHash string, apiKeyID, userID int64, clientID string, expiresAt time.Time) (*RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
}
