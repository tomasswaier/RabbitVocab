package oauth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tomasswaier/RabbitVocab/internal/db/pgtypeconv"
	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
)

var ErrNotFound = errors.New("not found or expired")

type PostgresRepository struct {
	q *sqlc.Queries
}

func NewPostgresRepository(q *sqlc.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) CreateClient(ctx context.Context, clientID string, clientName *string, redirectURIs []string) (*Client, error) {
	row, err := r.q.CreateOAuthClient(ctx, sqlc.CreateOAuthClientParams{
		ClientID:     clientID,
		ClientName:   pgtypeconv.TextFromPtr(clientName),
		RedirectUris: redirectURIs,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		ClientID:     row.ClientID,
		ClientName:   pgtypeconv.PtrFromText(row.ClientName),
		RedirectURIs: row.RedirectUris,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *PostgresRepository) GetClient(ctx context.Context, clientID string) (*Client, error) {
	row, err := r.q.GetOAuthClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Client{
		ClientID:     row.ClientID,
		ClientName:   pgtypeconv.PtrFromText(row.ClientName),
		RedirectURIs: row.RedirectUris,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *PostgresRepository) CreateAuthorizationCode(ctx context.Context, codeHash, clientID string, userID int64, redirectURI, codeChallenge, codeChallengeMethod string, expiresAt time.Time) (*AuthorizationCode, error) {
	row, err := r.q.CreateAuthorizationCode(ctx, sqlc.CreateAuthorizationCodeParams{
		CodeHash:            codeHash,
		ClientID:            clientID,
		UserID:              userID,
		RedirectUri:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           pgtypeconv.TimestamptzFromTime(expiresAt),
	})
	if err != nil {
		return nil, err
	}
	return toCodeEntity(row), nil
}

func (r *PostgresRepository) ConsumeAuthorizationCode(ctx context.Context, codeHash string) (*AuthorizationCode, error) {
	row, err := r.q.ConsumeAuthorizationCode(ctx, codeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toCodeEntity(row), nil
}

func (r *PostgresRepository) CreateRefreshToken(ctx context.Context, tokenHash string, apiKeyID, userID int64, clientID string, expiresAt time.Time) (*RefreshToken, error) {
	row, err := r.q.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		TokenHash: tokenHash,
		ApiKeyID:  apiKeyID,
		UserID:    userID,
		ClientID:  clientID,
		ExpiresAt: pgtypeconv.TimestamptzFromTime(expiresAt),
	})
	if err != nil {
		return nil, err
	}
	return toRefreshEntity(row), nil
}

func (r *PostgresRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row, err := r.q.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toRefreshEntity(row), nil
}
func (r *PostgresRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.q.DeleteRefreshToken(ctx, tokenHash)
	return err
}

func toCodeEntity(row sqlc.OauthAuthorizationCode) *AuthorizationCode {
	return &AuthorizationCode{
		CodeHash:            row.CodeHash,
		ClientID:            row.ClientID,
		UserID:              row.UserID,
		RedirectURI:         row.RedirectUri,
		CodeChallenge:       row.CodeChallenge,
		CodeChallengeMethod: row.CodeChallengeMethod,
		ExpiresAt:           row.ExpiresAt.Time,
	}
}

func toRefreshEntity(row sqlc.OauthRefreshToken) *RefreshToken {
	return &RefreshToken{
		TokenHash: row.TokenHash,
		APIKeyID:  row.ApiKeyID,
		UserID:    row.UserID,
		ClientID:  row.ClientID,
		ExpiresAt: row.ExpiresAt.Time,
	}
}
