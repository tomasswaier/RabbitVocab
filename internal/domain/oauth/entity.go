package oauth

import "time"

type Client struct {
	ClientID     string
	ClientName   *string
	RedirectURIs []string
	CreatedAt    time.Time
}

type AuthorizationCode struct {
	CodeHash            string
	ClientID            string
	UserID              int64
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
}

type RefreshToken struct {
	TokenHash string
	APIKeyID  int64
	UserID    int64
	ClientID  string
	ExpiresAt time.Time
}
