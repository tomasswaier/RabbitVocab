package user

import "time"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	APIKeyHash   string
	CreatedAt    time.Time
}
