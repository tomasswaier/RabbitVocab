package apikey

import "time"

type APIKey struct {
	ID        int64
	UserID    int64
	KeyHash   string
	Label     *string
	ClientID  *string
	ExpiresAt *time.Time
	CreatedAt time.Time
}
