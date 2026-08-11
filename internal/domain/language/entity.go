package language

import "time"

type Language struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
}
