package session

import (
	"context"
	"time"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
)

const sessionDuration = 30 * 24 * time.Hour // 30 days

type Service struct {
	sessions Repository
}

func NewService(sessions Repository) *Service {
	return &Service{sessions: sessions}
}

// Create generates a new raw session token, stores its hash, and returns
// the raw token for the caller to set as a cookie.
func (s *Service) Create(ctx context.Context, userID int64) (rawToken string, expiresAt time.Time, err error) {
	rawToken, err = apikey.Generate()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt = time.Now().Add(sessionDuration)
	if _, err := s.sessions.Create(ctx, userID, apikey.Hash(rawToken), expiresAt); err != nil {
		return "", time.Time{}, err
	}

	return rawToken, expiresAt, nil
}

// Resolve validates a raw session token and returns the associated userID.
func (s *Service) Resolve(ctx context.Context, rawToken string) (int64, error) {
	sess, err := s.sessions.GetValidByTokenHash(ctx, apikey.Hash(rawToken))
	if err != nil {
		return 0, err
	}
	return sess.UserID, nil
}

func (s *Service) Delete(ctx context.Context, rawToken string) error {
	return s.sessions.DeleteByTokenHash(ctx, apikey.Hash(rawToken))
}
